package cliapptest

import (
	"bytes"

	"github.com/vrooli/cli-core/cliapp"
)

// NewCapturedRunContext builds a RunContext whose Stdout is a fresh
// bytes.Buffer, and returns both. The Schema, Core, and Stdout fields of
// opts are overwritten by the helper; pass other fields (Flags, BoolFlags,
// Positionals, Repeated, JSON, Stderr) to drive the handler under test.
//
// This collapses the per-domain helper that test files would otherwise
// hand-roll:
//
//	func runCtx(core *cliapp.ScenarioApp, schema cliapp.ArgSchema, opts cliapptest.TestRunContextOptions) (cliapp.RunContext, *bytes.Buffer) {
//	    var buf bytes.Buffer
//	    opts.Schema, opts.Core, opts.Stdout = schema, core, &buf
//	    return cliapptest.NewTestRunContext(opts), &buf
//	}
//
// becomes the substrate's job. Domain handler tests call
// cliapptest.NewCapturedRunContext directly.
func NewCapturedRunContext(core *cliapp.ScenarioApp, schema cliapp.ArgSchema, opts TestRunContextOptions) (cliapp.RunContext, *bytes.Buffer) {
	var buf bytes.Buffer
	opts.Schema = schema
	opts.Core = core
	opts.Stdout = &buf
	return NewTestRunContext(opts), &buf
}
