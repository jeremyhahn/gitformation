package cmd

import (
	"github.com/jeremyhahn/gitformation/internal/executor"
	gitformation "github.com/jeremyhahn/gitformation/internal/git"
	"github.com/jeremyhahn/gitformation/internal/service/cloudformation"
	"github.com/spf13/cobra"
)

var Region string
var DeploymentBucketName string
var DeploymentBucketKeyPrefix string
var DeploymentParameters map[string]string
var Capabilities []string
var DisableRollback bool
var ExitOnError bool
var Parallel bool
var Filter string
var DryRun bool
var OutputFormat string
var WaitForStackResult bool
var ParameterFiles string
var DeploymentEnv string
var ProfilePrefix string
var Profile string
var CommitHash string
var ParameterFileMappings string
var DependencyGraph string

func init() {

	manageStacksCmd.PersistentFlags().StringVarP(&Region, "region", "r", "us-east-1", "Target AWS region (ex: us-east-1)")
	manageStacksCmd.PersistentFlags().StringVarP(&DeploymentBucketName, "template-bucket", "b", "", "S3 bucket name to deploy stacks from using --template-url (ex: my-bucket-name)")
	manageStacksCmd.PersistentFlags().StringVarP(&DeploymentBucketKeyPrefix, "template-bucket-key", "k", "", "S3 bucket key prefix where templates are stored (ex: /my/sub/folder)")
	manageStacksCmd.PersistentFlags().StringToStringVarP(&DeploymentParameters, "parameters", "p", nil, "Map of parameters to include with each cloudformation stack operation (ex: Environment=nonprod Foo=bar)")
	manageStacksCmd.PersistentFlags().StringArrayVar(&Capabilities, "capabilities", []string{}, "List of cloudformation capabilities to use for the deployment (ex: CAPABILITY_NAMED_IAM)")
	manageStacksCmd.PersistentFlags().BoolVar(&DisableRollback, "disable-rollback", false, "Disable cloudformation rollbacks on failure")
	manageStacksCmd.PersistentFlags().BoolVarP(&ExitOnError, "exit-on-error", "e", true, "Stop processing and exit with a failure message if an error is encountered during a clodformation operation")
	manageStacksCmd.PersistentFlags().BoolVarP(&Parallel, "parallel", "a", true, "Process each file in a parallel goroutine (async)")
	manageStacksCmd.PersistentFlags().StringVarP(&Filter, "filter", "f", "[a-zA-Z0-9./]+", "Regular expressin to filter files from the repository. Default is process all files. (ex: --filter=templates/*)")
	manageStacksCmd.PersistentFlags().BoolVar(&DryRun, "dry-run", false, "Regular expressin used to filter processed files in the repository")
	manageStacksCmd.PersistentFlags().StringVar(&OutputFormat, "format", "human", "The output format to use (human | json | yaml)")
	manageStacksCmd.PersistentFlags().BoolVarP(&WaitForStackResult, "wait", "w", false, "Wait for results from cloudformation stack operations")
	manageStacksCmd.PersistentFlags().StringVar(&ParameterFiles, "parameter-files", "./cloudformation/parameters", "Path to directory with cloudformation parameter files")
	manageStacksCmd.PersistentFlags().StringVar(&DeploymentEnv, "env", "nonprod", "Target deployment environment")
	manageStacksCmd.PersistentFlags().StringVar(&ProfilePrefix, "profile-prefix", "jeremyhahn", "Profile prefix to append the environment name to (ex: myco results in profile: myco-nonprod)")
	manageStacksCmd.PersistentFlags().StringVar(&Profile, "profile", "nonprod", "Target deployment account")
	manageStacksCmd.PersistentFlags().StringVar(&CommitHash, "commit", "", "The commit hash to process")
	manageStacksCmd.PersistentFlags().StringVar(&ParameterFileMappings, "parameter-mappings", "./examples/cloudformation/mappings/nonprod/mappings.yaml", "Path to template parameter file mappings")
	manageStacksCmd.PersistentFlags().StringVar(&DependencyGraph, "dependency-graph", "./examples/cloudformation/dependencies/nonprod/graph.yaml", "Path to template dependency graph")

	rootCmd.AddCommand(manageStacksCmd)
}

var manageStacksCmd = &cobra.Command{
	Use:   "manage-stacks",
	Short: "Binds git commit changes with AWS CloudFormation stack operations",
	Long: `Parses the last git commit to determine which files have been created,
	modified and/or deleted. For each change, the corresponding AWS CloudFormation
	stack operation is invoked. For new files, create-stack, for updated files, 
	update-stack, and for deleted files, delete-stack.`,
	Run: func(cmd *cobra.Command, args []string) {

		gitParser := gitformation.NewLocalRepoParser(App.Logger, Filter)
		changeSet := gitParser.Diff(CommitHash)

		if changeSet == nil {
			App.Logger.Fatal("unexpected git parser error: *git.ChangeSet is nil")
		}

		if DebugFlag {
			outputChangeSet(OutputFormat, changeSet)
		}

		var deploymentBucket *cloudformation.DeploymentBucket
		if DeploymentBucketName != "" {
			if DeploymentBucketKeyPrefix == "" {
				argRequiredError("--template-bucket-key")
			}
		}

		options := &cloudformation.ServiceOptions{
			Region:                Region,
			Profile:               Profile,
			ProfilePrefix:         ProfilePrefix,
			Environment:           DeploymentEnv,
			Bucket:                deploymentBucket,
			Parameters:            DeploymentParameters,
			ParameterFiles:        ParameterFiles,
			ParameterFileMappings: ParameterFileMappings,
			Capabilities:          Capabilities,
			DisableRollback:       DisableRollback,
			ExitOnError:           ExitOnError,
			WaitForStackResult:    WaitForStackResult,
			DependencyGraph:       DependencyGraph,
			DryRun:                DryRun}

		cloudformationService := cloudformation.NewCloudFormationService(App.Logger, options)

		executor := executor.NewExecutor(
			App.Logger,
			&executor.ExecutorOptions{
				Parallel:    Parallel,
				ExitOnError: ExitOnError},
			changeSet,
			cloudformationService)

		result := executor.Run()

		outputResult(OutputFormat, result)
	},
}
