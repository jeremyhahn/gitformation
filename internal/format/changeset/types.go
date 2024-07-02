package changeset

type Formatter interface {
	PrintChangeSet()
}

//type FormatterFunc func(changeSet *git.ChangeSet)
