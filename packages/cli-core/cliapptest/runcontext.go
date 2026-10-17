package cliapptest

import (
	"io"

	"github.com/vrooli/cli-core/cliapp"
)

// TestRunContextOptions mirrors cliapp.TestRunContextOptions.
type TestRunContextOptions = cliapp.TestRunContextOptions

// NewTestRunContext mirrors cliapp.NewTestRunContext.
func NewTestRunContext(opts TestRunContextOptions) cliapp.RunContext {
	return cliapp.NewTestRunContext(opts)
}

// NewTestRunContextFromArgs mirrors cliapp.NewTestRunContextFromArgs.
func NewTestRunContextFromArgs(schema cliapp.ArgSchema, args []string, core *cliapp.ScenarioApp, stdout, stderr io.Writer) (cliapp.RunContext, error) {
	return cliapp.NewTestRunContextFromArgs(schema, args, core, stdout, stderr)
}
