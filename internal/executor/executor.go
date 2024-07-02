package executor

import (
	"sync"

	"github.com/jeremyhahn/gitformation/internal/git"
	"github.com/op/go-logging"
	"golang.org/x/exp/maps"
)

type Executor struct {
	logger    *logging.Logger
	options   *ExecutorOptions
	changeSet *git.ChangeSet
	service   ServiceExecutor
	hasErrors bool
	ChangeSetExecutor
}

// Creates a new executor instance. The executor is responsble for
// running the create, update, and delete operations for a service
// using the CLI defined parameters for parallel and exit-on-error.
func NewExecutor(
	logger *logging.Logger,
	options *ExecutorOptions,
	changeSet *git.ChangeSet,
	service ServiceExecutor) ChangeSetExecutor {

	return &Executor{
		logger:    logger,
		options:   options,
		changeSet: changeSet,
		service:   service}
}

// Executes create, update, and delete operations for
// each file in the changeset.
func (e *Executor) Run() *ExecutionResult {
	return &ExecutionResult{
		ServiceName:   e.service.Name(),
		HasErrors:     e.hasErrors,
		CreateResults: e.Create(e.changeSet.Created),
		UpdateResults: e.Update(e.changeSet.Updated),
		DeleteResults: e.Delete(e.changeSet.Deleted)}
}

// Perform a create operation for each file in the changeset
func (executor *Executor) Create(creates []string) *OperationResult {
	return executor.Execute(git.Insert, creates, executor.service.Create)
}

// Perform an update operation for each deleted file in the changeset
func (executor *Executor) Update(updates []string) *OperationResult {
	return executor.Execute(git.Update, updates, executor.service.Update)
}

// Perform a delete operation for each deleted file in the changeset
func (executor *Executor) Delete(deletes []string) *OperationResult {
	return executor.Execute(git.Delete, deletes, executor.service.Delete)
}

// Execute the desired action (create, update, delete) using the passed
// options for parallelism and exit behavior.
func (executor *Executor) Execute(actionType git.ActionType, files []string,
	execFunc OperationExecFunc) *OperationResult {

	var wg sync.WaitGroup

	fileLen := len(files)
	responses := make(map[string]string, fileLen)
	errors := make(map[string]error, fileLen)

	responseChan := make(chan map[string]string, fileLen)
	errorChan := make(chan map[string]error, fileLen)
	doneChan := make(chan bool, 1)

	go executor.listen(actionType, responseChan, errorChan, responses, errors, doneChan)

	for _, filePath := range files {
		// Should have aborted by now if ExitOnError is true,
		// but putting this here for a safeguard to stop
		// executing jobs as soon as an error is seen.
		if executor.options.ExitOnError && len(errors) > 0 {
			break
		}
		wg.Add(1)
		if executor.options.Parallel {
			executor.logger.Debugf("executing asyncronous %s %s operation on %s",
				executor.service.Name(), actionType.String(), filePath)
			go execFunc(&ServiceParams{
				FilePath:     filePath,
				ResponseChan: responseChan,
				ErrorChan:    errorChan,
				WaitGroup:    &wg})
		} else {
			executor.logger.Debugf("executing synchronous %s %s operation on %s",
				executor.service.Name(), actionType.String(), filePath)
			execFunc(&ServiceParams{
				FilePath:     filePath,
				ResponseChan: responseChan,
				ErrorChan:    errorChan,
				WaitGroup:    &wg})
		}
	}

	wg.Wait()
	doneChan <- true

	executor.logger.Debugf("%s %s operations complete", executor.service.Name(), actionType.String())

	return NewOperationResult(responses, errors)
}

// Listen for responses and errors from the service
func (executor *Executor) listen(
	actionType git.ActionType,
	responseChan chan map[string]string,
	errorChan chan map[string]error,
	responses map[string]string,
	errors map[string]error,
	doneChan chan bool) {

	processing := true
	for processing {
		select {
		case response := <-responseChan:
			responses[maps.Keys(response)[0]] = maps.Values(response)[0]
		case err := <-errorChan:
			errors[maps.Keys(err)[0]] = maps.Values(err)[0]
			executor.hasErrors = true
			if executor.options.ExitOnError {
				executor.logger.Fatalf("%s %s encountered an error: %s", executor.service.Name(), actionType.String(), err)
			}
			executor.logger.Errorf("%s %s encountered an error: %s", executor.service.Name(), actionType.String(), err)
		case <-doneChan:
			executor.logger.Debugf("Done with %s %s operations complete. Closing response and error channels.",
				executor.service.Name(), actionType.String())
			processing = false
			close(responseChan)
			close(errorChan)
			close(doneChan)
		}
	}

	executor.logger.Debugf("%s %s channel listener exiting", executor.service.Name(), actionType.String())
}
