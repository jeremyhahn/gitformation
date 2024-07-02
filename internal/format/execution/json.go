package execution

import (
	"encoding/json"

	"github.com/jeremyhahn/gitformation/internal/executor"
	"github.com/op/go-logging"
)

type JsonExecutionResult struct {
	ServiceName   string               `yaml:"service" json:"service"`
	HasErrors     bool                 `yaml:"errors" json:"errors"`
	CreateResults *JsonOperationResult `yaml:"create" json:"create"`
	UpdateResults *JsonOperationResult `yaml:"update" json:"update"`
	DeleteResults *JsonOperationResult `yaml:"delete" json:"delete"`
}

type JsonOperationResult struct {
	Responses map[string]string `yaml:"responses" json:"responses" mapstructure:"responses"`
	Errors    map[string]string `yaml:"errors" json:"errors" mapstructure:"errors"`
}

type JsonFormat struct {
	logger *logging.Logger
	result *executor.ExecutionResult
	Formatter
}

func NewJsonFormat(
	logger *logging.Logger,
	result *executor.ExecutionResult) Formatter {
	return &JsonFormat{
		logger: logger,
		result: result}
}

func (formatter *JsonFormat) PrintResult() {

	// formatter.result.CreateResults = nil
	///formatter.result.UpdateResults = nil
	//formatter.result.DeleteResults = nil

	createJsonResults := &JsonOperationResult{
		Responses: formatter.result.CreateResults.Responses,
		Errors:    make(map[string]string, len(formatter.result.CreateResults.Errors)),
	}
	for k, err := range formatter.result.CreateResults.Errors {
		createJsonResults.Errors[k] = err.Error()
	}

	updateJsonResults := &JsonOperationResult{
		Responses: formatter.result.UpdateResults.Responses,
		Errors:    make(map[string]string, len(formatter.result.UpdateResults.Errors)),
	}
	for k, err := range formatter.result.UpdateResults.Errors {
		updateJsonResults.Errors[k] = err.Error()
	}

	deleteJsonResults := &JsonOperationResult{
		Responses: formatter.result.DeleteResults.Responses,
		Errors:    make(map[string]string, len(formatter.result.DeleteResults.Errors)),
	}
	for k, err := range formatter.result.DeleteResults.Errors {
		deleteJsonResults.Errors[k] = err.Error()
	}

	jsonExecutionResult := &JsonExecutionResult{
		ServiceName:   formatter.result.ServiceName,
		HasErrors:     formatter.result.HasErrors,
		CreateResults: createJsonResults,
		UpdateResults: updateJsonResults,
		DeleteResults: deleteJsonResults}

	data, err := json.Marshal(jsonExecutionResult)
	if err != nil {
		formatter.logger.Fatal(err)
	}
	formatter.logger.Info(string(data))
}
