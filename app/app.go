package app

import (
	logging "github.com/op/go-logging"
)

var Name string

type App struct {
	DebugFlag bool            `yaml:"debug" json:"debug" mapstructure:"debug"`
	DataDir   string          `yaml:"datadir" json:"datadir" mapstructure:"datadir"`
	HomeDir   string          `yaml:"homedir" json:"homedir" mapstructure:"homedir"`
	LogDir    string          `yaml:"logdir" json:"logdir" mapstructure:"logdir"`
	LogFile   string          `yaml:"logfile" json:"logfile" mapstructure:"logfile"`
	Logger    *logging.Logger `yaml:"-" json:"-" mapstructure:"-"`
}

func NewApp() *App {
	return new(App)
}
