package executor

import (
	"sync"

	"github.com/jeremyhahn/gitformation/internal/git"
)

type OperationExecFunc func(serviceParams *ServiceParams)

type ServiceExecutor interface {
	Name() string
	Create(serviceParams *ServiceParams)
	Update(serviceParams *ServiceParams)
	Delete(serviceParams *ServiceParams)
}

type ChangeSetExecutor interface {
	Run() *ExecutionResult
	Execute(actionType git.ActionType, files []string, execFunc OperationExecFunc) *OperationResult
	Create(creates []string) *OperationResult
	Update(updates []string) *OperationResult
	Delete(deletes []string) *OperationResult
}

type ExecutionResult struct {
	ServiceName   string           `yaml:"service" json:"service"`
	HasErrors     bool             `yaml:"errors" json:"errors"`
	CreateResults *OperationResult `yaml:"create" json:"create"`
	UpdateResults *OperationResult `yaml:"update" json:"update"`
	DeleteResults *OperationResult `yaml:"delete" json:"delete"`
}

type ExecutorOptions struct {
	Parallel    bool
	ExitOnError bool
}

type ServiceParams struct {
	FilePath     string
	ResponseChan chan map[string]string
	ErrorChan    chan map[string]error
	WaitGroup    *sync.WaitGroup
}
