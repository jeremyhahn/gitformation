package git

import (
	"regexp"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	logging "github.com/op/go-logging"
)

type GitParser struct {
	logger *logging.Logger
	filter *regexp.Regexp
	repo   *git.Repository
}

// Parses a new local .git repository
func NewLocalRepoParser(logger *logging.Logger, filter string) *GitParser {
	r, err := git.PlainOpen("./.git")
	if err != nil {
		logger.Fatal(err)
	}
	// Create an optimized, compiled pattern matcher for --filter option
	var rFilter *regexp.Regexp
	if filter != "" {
		rFilter, err = regexp.Compile(filter)
		if err != nil {
			logger.Fatal(err)
		}
	}
	return &GitParser{
		logger: logger,
		filter: rFilter,
		repo:   r}
}

// Parses a remote repository with a disk clone
func NewRemotRepoParser(logger *logging.Logger, url string, filter string) *GitParser {
	r, err := git.PlainClone("/tmp", false, &git.CloneOptions{URL: url})
	if err != nil {
		logger.Fatal(err)
	}
	var rFilter *regexp.Regexp
	if filter != "" {
		rFilter, err = regexp.Compile(filter)
		if err != nil {
			logger.Fatal(err)
		}
	}
	return &GitParser{
		logger: logger,
		filter: rFilter,
		repo:   r}
}

// Parses a remote repository with an in-memory clone
func NewRemoteMemoryRepoParser(logger *logging.Logger, url string, filter string) *GitParser {
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{URL: url})
	if err != nil {
		logger.Fatal(err)
	}
	var rFilter *regexp.Regexp
	if filter != "" {
		rFilter, err = regexp.Compile(filter)
		if err != nil {
			logger.Fatal(err)
		}
	}
	return &GitParser{
		logger: logger,
		filter: rFilter,
		repo:   r}
}

// Diffs the last commit to determine which files have been
// created, modified, and/or deleted, and returns a ChangeSet
// containing the relative file paths.
func (parser *GitParser) Diff(targetHash string) *ChangeSet {

	// The target hash plumbing.Hash
	var pTargetHash plumbing.Hash

	// Get HEAD branch
	headRef, err := parser.repo.Head()
	if err != nil {
		parser.logger.Fatal(err)
	}

	// Retrieve the commit instance
	headCommit, err := parser.repo.CommitObject(headRef.Hash())
	if err != nil {
		parser.logger.Fatal(err)
	}

	// If HEAD doesn't have any parent hashes, this is a new repo
	if len(headCommit.ParentHashes) == 0 {
		return parser.parseInitialCommit(headCommit)
	}

	// git ls-tree -r HEAD
	headTree, err := headCommit.Tree()
	if err != nil {
		parser.logger.Fatal(err)
	}

	// Use the specified --commit if passed,
	// otherwise default to HEAD's parent hash.
	if targetHash == "" {
		pTargetHash = headCommit.ParentHashes[0]
	} else {
		pTargetHash = plumbing.NewHash(targetHash)
	}

	// Show the git log commit message, author, etc for the HEAD
	// commit (the commit to compare the HEAD commit with)
	parser.logger.Debugf("Deploying HEAD commit %s...", headRef.Hash())
	logIter, err := parser.repo.Log(&git.LogOptions{From: headRef.Hash()})
	if err != nil {
		parser.logger.Fatal(err)
	}
	// Loop over each commit
	err = logIter.ForEach(func(c *object.Commit) error {
		if c.Hash == headRef.Hash() {
			// Show the commit log (author, message, etc)
			parser.logger.Debugf("%+v+", c)
			return nil
		}
		return nil
	})
	if err != nil {
		parser.logger.Fatal(err)
	}

	// Show the git log commit message, author, etc for the target
	// commit (the commit to compare the HEAD commit with)
	parser.logger.Debugf("Diffing against target commit %s...", pTargetHash)
	logIter, err = parser.repo.Log(&git.LogOptions{From: pTargetHash})
	if err != nil {
		parser.logger.Fatal(err)
	}
	// Loop over each commit
	err = logIter.ForEach(func(c *object.Commit) error {
		if c.Hash == pTargetHash {
			// Show the commit log (author, message, etc)
			parser.logger.Debugf("%+v+", c)
			return nil
		}
		return nil
	})
	if err != nil {
		parser.logger.Fatal(err)
	}

	// Retrieve the target commit instance
	targetCommit, err := parser.repo.CommitObject(pTargetHash)
	if err != nil {
		parser.logger.Fatal(err)
	}

	// git ls-tree -r targetHash
	targetTree, err := targetCommit.Tree()
	if err != nil {
		parser.logger.Fatal(err)
	}

	// Diff HEAD against the requested --commit-hash, or the parent hash if empty
	return parser.diff(headTree, targetTree)
}

// Diff a tree against a commit hash and returns a ChangeSet that contains
// all of the files that were created, modified, and/or deleted.
func (parser *GitParser) diff(sourceTree *object.Tree, targetTree *object.Tree) *ChangeSet {

	creates := make([]string, 0)
	updates := make([]string, 0)
	deletes := make([]string, 0)

	// Diff the two trees to get a change set
	changes, err := sourceTree.Diff(targetTree)
	if err != nil {
		parser.logger.Fatal(err)
	}

	// Parse each of the file changes
	for _, change := range changes {

		action, err := change.Action()
		if err != nil {
			parser.logger.Fatal(err)
		}

		// Get the file name (even if its been renamed)
		changeName := parser.changeName(change)

		// If a filter is defined, only process this change
		// if it matches the specified regexp pattern.
		if parser.filter != nil {
			if !parser.filter.MatchString(changeName) {
				continue
			}
		}

		// Collect and organize changes based on the action type
		switch action {
		case merkletrie.Insert:
			creates = append(creates, changeName)
		case merkletrie.Modify:
			updates = append(updates, changeName)
		case merkletrie.Delete:
			deletes = append(deletes, changeName)
		default:
			parser.logger.Fatalf("unexpected git change action: %+v", action)
		}
	}

	if err != nil {
		parser.logger.Fatal(err)
	}

	// Return a ChangeSet that contains all of the files
	// that have been created, modified, and/or deleted
	// since the requested --commit (plumbing.Hash).
	return NewChangeSet(creates, updates, deletes)
}

// Parses the an initial commit with no prior history
func (parser *GitParser) parseInitialCommit(commit *object.Commit) *ChangeSet {

	parser.logger.Debug("No previous commits found...")

	empty := make([]string, 0)
	creates := make([]string, 0)

	tree, err := commit.Tree()
	if err != nil {
		parser.logger.Fatal(err)
	}

	tree.Files().ForEach(func(f *object.File) error {
		creates = append(creates, f.Name)
		return nil
	})

	return NewChangeSet(creates, empty, empty)
}

// Parses the file name from a change (the file may have been renamed)
func (parser *GitParser) changeName(change *object.Change) string {
	var empty = object.ChangeEntry{}
	if change.From != empty {
		return change.From.Name
	}
	return change.To.Name
}
