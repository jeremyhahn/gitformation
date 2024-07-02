package changeset

import (
	"strings"

	"github.com/jeremyhahn/gitformation/internal/git"
	"github.com/op/go-logging"
)

type HumanFormat struct {
	logger    *logging.Logger
	changeSet *git.ChangeSet
	Formatter
}

func NewHumanFormat(logger *logging.Logger, changeSet *git.ChangeSet) Formatter {
	return &HumanFormat{
		logger:    logger,
		changeSet: changeSet}
}

func (formatter *HumanFormat) PrintChangeSet() {
	formatter.logger.Info("")

	formatter.logger.Info("--- Created ---")
	formatter.logger.Info(strings.Join(formatter.changeSet.Created[:], "\n"))
	formatter.logger.Info("")

	formatter.logger.Info("--- Updated ---")
	formatter.logger.Info(strings.Join(formatter.changeSet.Updated[:], "\n"))
	formatter.logger.Info("")

	formatter.logger.Info("--- Deleted ---")
	formatter.logger.Info(strings.Join(formatter.changeSet.Deleted[:], "\n"))
	formatter.logger.Info("")
}
