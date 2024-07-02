package cloudformation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/op/go-logging"
	"gopkg.in/yaml.v3"

	"github.com/jeremyhahn/gitformation/internal/executor"
)

type CloudFormationService struct {
	name         string
	logger       *logging.Logger
	client       *cloudformation.Client
	options      *ServiceOptions
	Mappings     map[string]string // Template mappings
	Dependencies [][]string        // Template dependencies
	executor.ServiceExecutor
}

func NewCloudFormationService(logger *logging.Logger,
	options *ServiceOptions) executor.ServiceExecutor {

	logger.Debug("Creating new CloudFormation service")

	var cfg aws.Config
	var err error
	if options.ProfilePrefix != "" {
		profile := fmt.Sprintf("%s-%s", options.ProfilePrefix, options.Environment)
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithSharedConfigProfile(profile))
	} else if options.Profile != "" {
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithSharedConfigProfile(options.Profile))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(options.Region))
	}
	if err != nil {
		logger.Fatalf("unable to load AWS SDK config: %v", err)
	}

	cfn := &CloudFormationService{
		name:         "cloudformation",
		logger:       logger,
		client:       cloudformation.NewFromConfig(cfg),
		options:      options,
		Mappings:     make(map[string]string, 0),
		Dependencies: make([][]string, 0)}

	cfn.loadParameterMappings(options.ParameterFileMappings)
	cfn.loadDependencies(options.DependencyGraph)

	return cfn
}

// Returns the service name
func (cfn *CloudFormationService) Name() string {
	return cfn.name
}

// Creates a new cloudformation stack
func (cfn *CloudFormationService) Create(serviceParams *executor.ServiceParams) {

	defer serviceParams.WaitGroup.Done()

	params := cfn.createStackParams(serviceParams.FilePath)
	cfn.logger.Debugf("Creating cloudformation stack: %s", *params.StackName)

	if cfn.options.DryRun {
		cfn.logger.Fatalf("create-stack params: %+v", params)
	}

	result, err := cfn.client.CreateStack(context.TODO(), params)
	if err != nil {
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		response := make(map[string]error, 1)
		response[serviceParams.FilePath] = err
		serviceParams.ErrorChan <- response
		//cfn.logger.Errorf("cloudformation service: %s", err)
		return
	}

	if cfn.options.WaitForStackResult {
		for {
			dsResult, err := cfn.client.DescribeStacks(context.TODO(), &cloudformation.DescribeStacksInput{StackName: params.StackName})
			if err != nil {
				if cfn.options.ExitOnError {
					cfn.logger.Fatal(err)
				}
			}
			if len(dsResult.Stacks) == 0 {
				if cfn.options.ExitOnError {
					cfn.logger.Fatal("unexpected response from describe-stacks: stacks.length = 0")
				}
			}
			if strings.Contains(string(dsResult.Stacks[0].StackStatus), "FAILED") {
				cfn.logger.Fatal("%s", dsResult.Stacks[0].StackStatus)
			}
			if strings.Contains(string(dsResult.Stacks[0].StackStatus), "COMPLETE") {
				cfn.logger.Fatal("%s", dsResult.Stacks[0].StackStatus)
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	cfn.logger.Debugf("%+v", result)
	stackInfo := make(map[string]string, 1)
	stackInfo[serviceParams.FilePath] = *result.StackId //fmt.Sprintf("%+v", result.ResultMetadata)
	serviceParams.ResponseChan <- stackInfo
}

// Updates an existing cloudformation stack
func (cfn *CloudFormationService) Update(serviceParams *executor.ServiceParams) {

	defer serviceParams.WaitGroup.Done()

	params := cfn.updateStackParams(serviceParams.FilePath)
	cfn.logger.Debugf("Updating cloudformation stack: %+v", *params.StackName)

	if cfn.options.DryRun {
		cfn.logger.Fatalf("update-stack params: %+v", params)
	}

	result, err := cfn.client.UpdateStack(context.TODO(), params)
	if err != nil {
		response := make(map[string]error, 1)
		response[serviceParams.FilePath] = err
		serviceParams.ErrorChan <- response
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		cfn.logger.Error(err)
		return
	}
	cfn.logger.Debugf("%+v", result)
	stackInfo := make(map[string]string, 1)
	stackInfo[*result.StackId] = fmt.Sprintf("%+v", result.ResultMetadata)
	serviceParams.ResponseChan <- stackInfo
}

// Deletes an existing cloudformation stack
func (cfn *CloudFormationService) Delete(serviceParams *executor.ServiceParams) {

	defer serviceParams.WaitGroup.Done()
	params := cfn.deleteStackParams(serviceParams.FilePath)
	cfn.logger.Debugf("Deleting cloudformation stack: %s", *params.StackName)

	if cfn.options.DryRun {
		cfn.logger.Fatalf("delete-stack params: %+v", params)
	}

	result, err := cfn.client.DeleteStack(context.TODO(), params)
	if err != nil {
		response := make(map[string]error, 1)
		response[serviceParams.FilePath] = err
		serviceParams.ErrorChan <- response
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		cfn.logger.Error(err)
		return
	}
	cfn.logger.Debugf("%+v", result)
	stackInfo := make(map[string]string, 1)
	stackInfo[serviceParams.FilePath] = "success" // fmt.Sprintf("%+v", result.ResultMetadata); json.Marshal has problems with this
	serviceParams.ResponseChan <- stackInfo
}

// Returns a feasible cloudformation stack name, given a file name
func (cfn *CloudFormationService) parseStackNameFromFile(file string) *string {

	var stackName string

	pathPieces := strings.Split(file, "/")
	fileName := pathPieces[len(pathPieces)-1]

	fileNamePieces := strings.Split(fileName, ".")
	if len(fileNamePieces) == 0 {
		stackName = cfn.cleanStackName(fileName)
		return &stackName
	}

	stackName = cfn.cleanStackName(fileNamePieces[0])
	return &stackName
}

// Attempt to correct common template naming and consistency problems
func (Cfn *CloudFormationService) cleanStackName(raw string) string {
	s := strings.ToLower(raw)
	s = strings.Replace(s, "_", "-", -1)
	return s
}

// create-stack params
func (cfn *CloudFormationService) createStackParams(filePath string) *cloudformation.CreateStackInput {

	stackInputParams := &cloudformation.CreateStackInput{
		StackName:       cfn.parseStackNameFromFile(filePath),
		DisableRollback: &cfn.options.DisableRollback}

	// Use --template-url if deployment bucket is defined
	if cfn.options.Bucket != nil {
		templateUrl := fmt.Sprintf("https://%s.s3.amazonaws.com/%s/%s",
			cfn.options.Bucket.BucketName,
			cfn.options.Bucket.KeyPrefix,
			filePath)
		stackInputParams.TemplateURL = &templateUrl
	} else {
		stackInputParams.TemplateBody = &filePath
	}

	// Pass --parameters if defined
	if len(cfn.options.Parameters) > 0 {
		params := make([]types.Parameter, len(cfn.options.Parameters))
		i := 0
		for k, v := range cfn.options.Parameters {
			params[i] = types.Parameter{
				ParameterKey:   &k,
				ParameterValue: &v}
			i++
		}
		stackInputParams.Parameters = params
	}

	// Load parameters from --parameter-files location if specified
	parametersFile := cfn.parametersFromFile(filePath)
	if parametersFile != nil {
		parameters := cfn.parseParametersFile(*parametersFile)
		if len(parameters) > 0 {
			cfn.logger.Infof("using parameters file: %s", *parametersFile)
			stackInputParams.Parameters = parameters
		}
	}

	// Pass --capabilities if defined
	if len(cfn.options.Capabilities) > 0 {
		capabilities := make([]types.Capability, len(cfn.options.Capabilities))
		i := 0
		for _, capabilitiy := range cfn.options.Capabilities {
			switch capabilitiy {
			case "CAPABILITY_IAM":
				capabilities[i] = types.CapabilityCapabilityIam
			case "CAPABILITY_NAMED_IAM":
				capabilities[i] = types.CapabilityCapabilityNamedIam
			case "CAPABILITY_AUTO_EXPAND":
				capabilities[i] = types.CapabilityCapabilityAutoExpand
			default:
				cfn.logger.Fatal("invalid capability: %s", capabilitiy)
			}
			i++
		}
		stackInputParams.Capabilities = capabilities
	}

	return stackInputParams
}

// update-stack params
func (cfn *CloudFormationService) updateStackParams(filePath string) *cloudformation.UpdateStackInput {

	stackUpdateParams := &cloudformation.UpdateStackInput{
		StackName:       cfn.parseStackNameFromFile(filePath),
		DisableRollback: &cfn.options.DisableRollback}

	// Use --template-url if deployment bucket is defined
	if cfn.options.Bucket != nil {
		templateUrl := fmt.Sprintf("https://%s.s3.amazonaws.com/%s/%s",
			cfn.options.Bucket.BucketName,
			cfn.options.Bucket.KeyPrefix,
			filePath)
		stackUpdateParams.TemplateURL = &templateUrl
	} else {
		stackUpdateParams.TemplateBody = &filePath
	}

	// Pass --parameters if defined
	if len(cfn.options.Parameters) > 0 {
		params := make([]types.Parameter, len(cfn.options.Parameters))
		i := 0
		for k, v := range cfn.options.Parameters {
			params[i] = types.Parameter{
				ParameterKey:   &k,
				ParameterValue: &v}
			i++
		}
		stackUpdateParams.Parameters = params
	}

	// Load parameters from --parameter-files location if specified
	parametersFile := cfn.parametersFromFile(filePath)
	if parametersFile != nil {
		parameters := cfn.parseParametersFile(*parametersFile)
		if len(parameters) > 0 {
			cfn.logger.Infof("using parameters file: %s", *parametersFile)
			stackUpdateParams.Parameters = parameters
		}
	}

	// Pass --capabilities if defined
	if len(cfn.options.Capabilities) > 0 {
		capabilities := make([]types.Capability, len(cfn.options.Capabilities))
		i := 0
		for _, capabilitiy := range cfn.options.Capabilities {
			switch capabilitiy {
			case "CAPABILITY_IAM":
				capabilities[i] = types.CapabilityCapabilityIam
			case "CAPABILITY_NAMED_IAM":
				capabilities[i] = types.CapabilityCapabilityNamedIam
			case "CAPABILITY_AUTO_EXPAND":
				capabilities[i] = types.CapabilityCapabilityAutoExpand
			default:
				cfn.logger.Fatal("invalid capability: %s", capabilitiy)
			}
			i++
		}
		stackUpdateParams.Capabilities = capabilities
	}

	return stackUpdateParams
}

// delete-stack params
func (cfn *CloudFormationService) deleteStackParams(filePath string) *cloudformation.DeleteStackInput {
	return &cloudformation.DeleteStackInput{
		StackName: cfn.parseStackNameFromFile(filePath)}
}

// Check to see if a parameter file exists at --parameter-files
func (cfn *CloudFormationService) parametersFromFile(filePath string) *string {
	file := filepath.Base(filePath)
	fileNameNoExt := strings.Split(file, ".")[0]
	parameterFile := fmt.Sprintf("%s/%s/%s.parameters", cfn.options.ParameterFiles,
		cfn.options.Environment, fileNameNoExt)
	if _, err := os.Stat(parameterFile); err == nil {
		return &parameterFile
	}
	return nil
}

// Parses a parameters file and returns all of the parameters in a format suitable
// for create-stack and update-stack operations.
func (cfn *CloudFormationService) parseParametersFile(file string) []types.Parameter {

	data, err := os.ReadFile(file)
	if err != nil {
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		cfn.logger.Error(err)
	}

	cfn.logger.Debugf("Loading parameters file: %s", file)

	var jsonParams []Parameter
	err = json.Unmarshal(data, &jsonParams)
	if err != nil {
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
	}

	params := make([]types.Parameter, len(jsonParams))
	for i, p := range jsonParams {
		params[i] = types.Parameter{
			ParameterKey:   &p.ParameterKey,
			ParameterValue: &p.ParameterValue}
		cfn.logger.Debugf("%s=%s", p.ParameterKey, p.ParameterValue)
	}

	return params
}

// Parses a --mappings file to locate the parameters file for a given stack.
// This allows mapping a standard template name such as vpc.template to a
// parameters file located in a different directory, file name, or extention,
// such as vpc-nonprod.parameters, parameters/nonprod/vpc.parameters, or
// /custom/modules/parameters/vpc.json.
func (cfn *CloudFormationService) loadParameterMappings(mappingsYaml string) {

	data, err := os.ReadFile(mappingsYaml)
	if err != nil {
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		cfn.logger.Error(err)
	}

	cfn.logger.Debugf("Loading parameter mappings: %s", mappingsYaml)

	var mappings MappingsYaml
	err = yaml.Unmarshal(data, &mappings)
	if err != nil {
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		cfn.logger.Error(err)
	}

	cfn.Mappings = mappings.Templates
}

// Parses a --dependency-graph dependency graph descriptor
func (cfn *CloudFormationService) loadDependencies(dependenciesYaml string) {

	data, err := os.ReadFile(dependenciesYaml)
	if err != nil {
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		cfn.logger.Error(err)
	}

	cfn.logger.Debugf("Loading dependency graph: %s", dependenciesYaml)

	var depsYaml []map[string]string
	err = yaml.Unmarshal(data, &depsYaml)
	if err != nil {
		if cfn.options.ExitOnError {
			cfn.logger.Fatal(err)
		}
		cfn.logger.Error(err)
	}

	// Copy the unmarshalled map into a new local map
	// so the dependency graph can work with the references
	newDeps := make([]map[string]string, len(depsYaml))
	for i, dep := range depsYaml {
		for k, v := range dep {
			cfn.logger.Debugf("%s = %s", k, v)
			newMap := make(map[string]string, 1)
			newMap[k] = v
			newDeps[i] = newMap
		}
	}

	// Build the dependency graph from the config
	g := NewDependencyGraph()
	for _, dep := range newDeps {
		for k, v := range dep {
			g.DependOn(k, v)
		}
	}

	cfn.Dependencies = g.TopoSortedLayers()

	for i, layer := range g.TopoSortedLayers() {
		cfn.logger.Debugf("execution plan, step %d: %s\n", i+1, strings.Join(layer, ", "))
	}
}
