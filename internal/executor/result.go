package executor

type OperationResult struct {
	Responses map[string]string `yaml:"responses" json:"responses" mapstructure:"responses"`
	Errors    map[string]error  `yaml:"errors" json:"errors" mapstructure:"errors"`
}

func NewOperationResult(responses map[string]string, errors map[string]error) *OperationResult {
	return &OperationResult{
		Responses: responses,
		Errors:    errors,
	}
}
