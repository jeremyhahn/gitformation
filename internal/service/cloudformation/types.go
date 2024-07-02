package cloudformation

type Parameter struct {
	ParameterKey   string `json:"ParameterKey"`
	ParameterValue string `json:"ParameterValue"`
}

type DeploymentBucket struct {
	BucketName string
	KeyPrefix  string
}

type ServiceOptions struct {
	Region                string
	Profile               string
	ProfilePrefix         string
	Environment           string
	Bucket                *DeploymentBucket
	Parameters            map[string]string
	ParameterFiles        string
	Capabilities          []string
	DisableRollback       bool
	ExitOnError           bool
	WaitForStackResult    bool
	ParameterFileMappings string
	DependencyGraph       string
	DryRun                bool
}

type MappingsYaml struct {
	Templates map[string]string
}

func (mappings *MappingsYaml) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&mappings.Templates)
}
