package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/jeremyhahn/gitformation/app"
	"github.com/jeremyhahn/gitformation/internal/executor"
	"github.com/jeremyhahn/gitformation/internal/format/changeset"
	"github.com/jeremyhahn/gitformation/internal/format/execution"
	"github.com/jeremyhahn/gitformation/internal/git"

	logging "github.com/op/go-logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortpkg}.%{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

var App *app.App
var DebugFlag bool
var ConfigDir string
var DataDir string
var LogDir string
var LogFile string
var HomeDir string

var rootCmd = &cobra.Command{
	Use:   app.Name,
	Short: "Gitformation",
	Long: `Binds git commit changes with AWS CloudFormation stack operations,
by parseing a git log to determine which files have been created, modified
and/or deleted since the last commit. For each change, the corresponding AWS
CloudFormation stack operation is executed. For new files, create-stack, 
updated files, update-stack, and deleted files, delete-stack.`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		//initApp()
	},
	Run: func(cmd *cobra.Command, args []string) {
	},
	TraverseChildren: true,
}

func init() {
	cobra.OnInitialize(initApp)

	wd, _ := os.Getwd()

	rootCmd.PersistentFlags().BoolVarP(&DebugFlag, "debug", "", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVarP(&HomeDir, "home", "", wd, "Program home directory")

	viper.BindPFlags(rootCmd.PersistentFlags())

	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	return nil
}

func initApp() {
	App.DebugFlag = viper.GetBool("debug")
	App.HomeDir = viper.GetString("home")
	initLogger()
	initConfig()
	if App.DebugFlag {
		logging.SetLevel(logging.DEBUG, "")
		App.Logger.Debug("Starting logger in debug mode...")
		for k, v := range viper.AllSettings() {
			App.Logger.Debugf("%s: %+v", k, v)
		}
	} else {
		logging.SetLevel(logging.INFO, "")
	}
}

func initLogger() {
	App.LogDir = LogDir
	App.LogFile = LogFile
	stdout := logging.NewLogBackend(os.Stdout, "", 0)
	logging.SetBackend(stdout)
	if App.DebugFlag {
		logging.SetLevel(logging.DEBUG, "")
	} else {
		logging.SetLevel(logging.ERROR, "")
	}
	App.Logger = logging.MustGetLogger(app.Name)
}

func initConfig() {

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(ConfigDir)
	viper.AddConfigPath(fmt.Sprintf("/etc/%s/", app.Name))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s/", app.Name))
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		App.Logger.Errorf("%s", err)
	}

	viper.Unmarshal(&App)

	App.DataDir = viper.GetString("data-dir")

	App.Logger.Debugf("%+v", App)
}

func argRequiredError(arg string) {
	App.Logger.Fatalf("%s argument required ", arg)
}

func outputChangeSet(output string, changeSet *git.ChangeSet) {
	switch output {
	case "human":
		changeset.NewHumanFormat(App.Logger, changeSet).PrintChangeSet()
	case "json":
		changeset.NewJsonFormat(App.Logger, changeSet).PrintChangeSet()
	default:
		App.Logger.Fatalf("unsupported --format option: %s", output)
	}
}

func outputResult(output string, result *executor.ExecutionResult) {
	switch output {
	case "human":
		execution.NewHumanFormat(App.Logger, result).PrintResult()
	case "json":
		execution.NewJsonFormat(App.Logger, result).PrintResult()
	default:
		App.Logger.Fatalf("unsupported --format option: %s", output)
	}
}
