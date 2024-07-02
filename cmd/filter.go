package cmd

import (
	gitformation "github.com/jeremyhahn/gitformation/internal/git"
	"github.com/spf13/cobra"
)

func init() {

	filterCmd.PersistentFlags().StringVarP(&Filter, "filter", "f", "[a-zA-Z0-9./]+", "Regular expressin to filter files from the repository. Default is process all files. (ex: --filter=templates/*)")
	filterCmd.PersistentFlags().BoolVar(&DryRun, "dry-run", false, "Regular expressin used to filter processed files in the repository")

	rootCmd.AddCommand(filterCmd)
}

var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Test git filter",
	Long: `Parses the git log using the --filter option and returns
		   all of the matching files that will be processed when run
		   using live services.`,
	Run: func(cmd *cobra.Command, args []string) {

		gitParser := gitformation.NewLocalRepoParser(App.Logger, CommitHash)
		changeSet := gitParser.Diff(Filter)

		outputChangeSet(OutputFormat, changeSet)
	},
}
