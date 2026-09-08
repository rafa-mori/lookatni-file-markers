package main

import (
	gl "github.com/kubex-ecosystem/logz"
	"github.com/kubex-ecosystem/lookatni-file-markers/internal/module"
)

var logger = gl.GetLogger("lookatni")

func init() {
	gl.GetLogger("lookatni")
	// l.SetLogConfig(logz.LoggerZ.GetConfig())

	// gl.SetLogLevel("info")
	// l.SetLogWriter(logz.LoggerZ.GetWriter())
}

// main initializes the logger and creates a new LookAtni instance.
func main() {

	if err := module.RegX().Command().Execute(); err != nil {
		gl.Log("fatal", err.Error())
	}
}
