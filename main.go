package main

import (
	"github.com/jeremyhahn/gitformation/app"
	"github.com/jeremyhahn/gitformation/cmd"
)

func main() {
	cmd.App = app.NewApp()
	cmd.Execute()
}
