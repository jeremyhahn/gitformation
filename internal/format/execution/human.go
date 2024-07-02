package execution

import (
	"github.com/jeremyhahn/gitformation/internal/executor"
	"github.com/op/go-logging"
)

type HumanFormat struct {
	logger *logging.Logger
	result *executor.ExecutionResult
	Formatter
}

func NewHumanFormat(
	logger *logging.Logger,
	executionResult *executor.ExecutionResult) Formatter {
	return &HumanFormat{
		logger: logger,
		result: executionResult}
}

func (formatter *HumanFormat) PrintResult() {
	formatter.logger.Info("")
	formatter.logger.Infof("--- RESULT ----")
	if formatter.result.HasErrors {
		if len(formatter.result.CreateResults.Errors) > 0 {
			formatter.printErrors("create", formatter.result.CreateResults.Errors)
		}
		if len(formatter.result.UpdateResults.Errors) > 0 {
			formatter.printErrors("update", formatter.result.UpdateResults.Errors)
		}
		if len(formatter.result.DeleteResults.Errors) > 0 {
			formatter.printErrors("delete", formatter.result.DeleteResults.Errors)
		}
	}
}

func (formatter *HumanFormat) printErrors(actionType string, errors map[string]error) {
	for k, v := range errors {
		formatter.logger.Info("")
		formatter.logger.Infof("service: %s, action: %s, file: %s, error: %s",
			formatter.result.ServiceName, actionType, k, v)
	}
}
