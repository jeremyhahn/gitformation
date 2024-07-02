package cloudformation

// import (
// 	"context"
// 	"fmt"
// 	"strings"
// 	"sync"

// 	"github.com/aws/aws-sdk-go-v2/config"
// 	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
// 	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
// 	"github.com/op/go-logging"
// 	"golang.org/x/exp/maps"

// 	"github.com/jeremyhahn/gitformation/internal/executor"
// )

// type DeploymentBucket struct {
// 	BucketName string
// 	KeyPrefix  string
// }

// type CloudFormationService struct {
// 	logger           *logging.Logger
// 	client           *cloudformation.Client
// 	deploymentBucket *DeploymentBucket
// 	parameters       map[string]string
// 	capabilities     []string
// 	disableRollback  bool
// 	exitOnError      bool
// 	executor.ServiceExecutor
// }

// type CloudFormationServiceOptions struct {
// 	DeploymentBucket *DeploymentBucket
// 	Parameters       map[string]string
// 	Capabilities     []string
// 	DisableRollback  bool
// 	ExitOnError      bool
// }

// func NewCloudFormationService(logger *logging.Logger, options *CloudFormationServiceOptions) executor.ServiceExecutor {

// 	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2")) // WithSharedConfigProfile
// 	if err != nil {
// 		logger.Fatalf("unable to load AWS SDK config, %v", err)
// 	}

// 	return &CloudFormationService{
// 		logger:           logger,
// 		client:           cloudformation.NewFromConfig(cfg),
// 		deploymentBucket: options.DeploymentBucket,
// 		parameters:       options.Parameters,
// 		capabilities:     options.Capabilities,
// 		disableRollback:  options.DisableRollback}
// }

// // Creates a new cloudformation stack
// func (cfn *CloudFormationService) CreateStacks(creates []string) executor.OperationResult {

// 	var wg sync.WaitGroup
// 	stackChan := make(chan map[string]string, len(creates))
// 	errorChan := make(chan map[string]error, len(creates))

// 	stacks := make(map[string]string, len(creates))
// 	errors := make(map[string]error, len(creates))

// 	for _, filePath := range creates {

// 		stackInputParams := &cloudformation.CreateStackInput{
// 			StackName:       cfn.parseStackNameFromFile(filePath),
// 			DisableRollback: &cfn.disableRollback}

// 		// Use --template-url if deployment bucket defined
// 		if cfn.deploymentBucket != nil {
// 			templateUrl := fmt.Sprintf("https://%s.s3.amazonaws.com/%s/%s",
// 				cfn.deploymentBucket.BucketName,
// 				cfn.deploymentBucket.KeyPrefix,
// 				filePath)
// 			stackInputParams.TemplateURL = &templateUrl
// 		} else {
// 			stackInputParams.TemplateBody = &filePath
// 		}

// 		// Pass --parameters if defined
// 		if len(cfn.parameters) > 0 {
// 			params := make([]types.Parameter, len(cfn.parameters))
// 			i := 0
// 			for k, v := range cfn.parameters {
// 				params[i] = types.Parameter{
// 					ParameterKey:   &k,
// 					ParameterValue: &v}
// 				i++
// 			}
// 			stackInputParams.Parameters = params
// 		}

// 		// Pass --capabilities if defined
// 		if len(cfn.capabilities) > 0 {
// 			capabilities := make([]types.Capability, len(cfn.capabilities))
// 			i := 0
// 			for _, capabilitiy := range cfn.capabilities {
// 				switch capabilitiy {
// 				case "CAPABILITY_IAM":
// 					capabilities[i] = types.CapabilityCapabilityIam
// 				case "CAPABILITY_NAMED_IAM":
// 					capabilities[i] = types.CapabilityCapabilityNamedIam
// 				case "CAPABILITY_AUTO_EXPAND":
// 					capabilities[i] = types.CapabilityCapabilityAutoExpand
// 				default:
// 					cfn.logger.Fatal("invalid capability: %s", capabilitiy)
// 				}
// 				i++
// 			}
// 			stackInputParams.Capabilities = capabilities
// 		}

// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			cfn.logger.Debugf("Creating stack: %s", *stackInputParams.StackName)
// 			result, err := cfn.client.CreateStack(context.TODO(), stackInputParams)
// 			if err != nil {
// 				response := make(map[string]error, 1)
// 				response[*stackInputParams.StackName] = err
// 				errorChan <- response
// 				if cfn.exitOnError {
// 					cfn.logger.Fatal(err)
// 				}
// 				cfn.logger.Error(err)
// 				return
// 			}
// 			cfn.logger.Debugf("%+v", result)
// 			stackInfo := make(map[string]string, 1)
// 			stackInfo[*result.StackId] = fmt.Sprintf("%+v", result.ResultMetadata)
// 			stackChan <- stackInfo
// 		}()
// 	}

// 	select {
// 	case stack := <-stackChan:
// 		stacks[maps.Keys(stack)[0]] = maps.Values(stack)[0]
// 	case err := <-errorChan:
// 		errors[maps.Keys(err)[0]] = maps.Values(err)[0]
// 		if cfn.exitOnError {
// 			cfn.logger.Fatal(err)
// 		}
// 	default:
// 		cfn.logger.Info("Waiting for responses on cloudformation create-stack channel...")
// 	}

// 	wg.Wait()
// 	close(stackChan)
// 	close(errorChan)

// 	return executor.NewOperationResult(stacks, errors)
// }

// func (cfn *CloudFormationService) UpdateStacks(creates []string) {
// }

// func (cfn *CloudFormationService) DeleteStacks(creates []string) {
// }

// // Returns a feasible cloudformation stack name, given a file name
// func (cfn *CloudFormationService) parseStackNameFromFile(file string) *string {

// 	var stackName string

// 	pathPieces := strings.Split(file, "/")
// 	fileName := pathPieces[len(pathPieces)-1]

// 	fileNamePieces := strings.Split(fileName, ".")
// 	if len(fileNamePieces) == 0 {
// 		stackName = cfn.cleanStackName(fileName)
// 		return &stackName
// 	}

// 	stackName = cfn.cleanStackName(fileNamePieces[0])
// 	return &stackName
// }

// // Attempt to correct common template naming and consistency problems
// func (Cfn *CloudFormationService) cleanStackName(raw string) string {
// 	s := strings.ToLower(raw)
// 	s = strings.Replace(s, "_", "-", -1)
// 	return s
// }
