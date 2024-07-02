package changeset

import (
	"encoding/json"

	"github.com/jeremyhahn/gitformation/internal/git"
	"github.com/op/go-logging"
)

type JsonFormat struct {
	logger    *logging.Logger
	changeSet *git.ChangeSet
	Formatter
}

func NewJsonFormat(logger *logging.Logger, changeSet *git.ChangeSet) Formatter {
	return &JsonFormat{
		logger:    logger,
		changeSet: changeSet}
}

func (formatter *JsonFormat) PrintChangeSet() {
	data, err := json.Marshal(formatter.changeSet)
	if err != nil {
		formatter.logger.Fatal(err)
	}
	formatter.logger.Info(string(data))
}
