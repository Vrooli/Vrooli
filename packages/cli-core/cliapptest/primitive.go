package cliapptest

import (
	"bytes"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// PrimitiveModes holds the two captured stdout renderings of a renderer-separated
// command handler: the human report and the --json output, produced by running
// the same handler (and therefore the same operation) under both output modes.
type PrimitiveModes struct {
	Human    string
	HumanErr error
	JSON     string
	JSONErr  error
}

// RunPrimitiveModes runs handler under the human and --json output modes against
// schema+args using core, returning both captured stdout renderings. It drives
// the production parser via cliapp.NewTestRunContextFromArgs, so the built-in
// --json pseudo-flag is honored exactly as in dispatch.
//
// A handler built from a cli-core primitive (cliapp.ProtoList/ProtoMutation/
// ProtoOperational, or the durable primitive) renders both modes from one
// operation execution, so a test can assert the two outputs are consistent —
// the JSON parses as the proto wire shape while the human output carries the
// report headings — to prove the command is renderer-separated rather than
// branching its operation on the output format.
//
// args must NOT include --json; RunPrimitiveModes appends it for the JSON run.
func RunPrimitiveModes(
	tb testing.TB,
	handler func(cliapp.RunContext) error,
	schema cliapp.ArgSchema,
	args []string,
	core *cliapp.ScenarioApp,
) PrimitiveModes {
	tb.Helper()

	var out PrimitiveModes

	humanArgs := append([]string(nil), args...)
	var humanBuf, humanErrBuf bytes.Buffer
	humanCtx, err := cliapp.NewTestRunContextFromArgs(schema, humanArgs, core, &humanBuf, &humanErrBuf)
	if err != nil {
		tb.Fatalf("RunPrimitiveModes: parse human args %v: %v", humanArgs, err)
		return out
	}
	out.HumanErr = handler(humanCtx)
	out.Human = humanBuf.String()

	jsonArgs := append(append([]string(nil), args...), "--json")
	var jsonBuf, jsonErrBuf bytes.Buffer
	jsonCtx, err := cliapp.NewTestRunContextFromArgs(schema, jsonArgs, core, &jsonBuf, &jsonErrBuf)
	if err != nil {
		tb.Fatalf("RunPrimitiveModes: parse json args %v: %v", jsonArgs, err)
		return out
	}
	out.JSONErr = handler(jsonCtx)
	out.JSON = jsonBuf.String()

	return out
}

// RunPrimitiveHandlerModes is the PrimitiveHandler-aware form of
// RunPrimitiveModes: it drives ph.Run under both output modes. Use it when a test
// starts from a builder result (cliapp.ProtoList/ProtoMutation/ProtoOperational)
// and wants to exercise rendering behavior alongside AssertPrimitiveEvidence.
func RunPrimitiveHandlerModes(
	tb testing.TB,
	ph cliapp.PrimitiveHandler,
	schema cliapp.ArgSchema,
	args []string,
	core *cliapp.ScenarioApp,
) PrimitiveModes {
	tb.Helper()
	if ph.Run == nil {
		tb.Fatalf("RunPrimitiveHandlerModes: PrimitiveHandler has a nil Run")
		return PrimitiveModes{}
	}
	return RunPrimitiveModes(tb, ph.Run, schema, args, core)
}

// AssertPrimitiveEvidence fails the test unless cmd carries exactly the expected
// observed primitive evidence. This is the machine-readable proof that the
// command's handler was built by a cli-core primitive rather than declared in
// manifest text alone — the property CLI Health reconciles against the manifest.
func AssertPrimitiveEvidence(tb testing.TB, cmd cliapp.Command, want cliapp.PrimitiveClass) {
	tb.Helper()
	if cmd.PrimitiveEvidence() != want {
		tb.Fatalf("command %q primitive evidence = %q, want %q", cmd.Name, cmd.PrimitiveEvidence(), want)
	}
}

// AssertDeclaredArchitecture fails the test unless cmd declares exactly the
// expected architecture metadata (the manifest-declared side, distinct from the
// observed PrimitiveEvidence asserted by AssertPrimitiveEvidence).
func AssertDeclaredArchitecture(tb testing.TB, cmd cliapp.Command, want cliapp.CommandArchitecture) {
	tb.Helper()
	if cmd.Architecture != want {
		tb.Fatalf("command %q declared architecture = %+v, want %+v", cmd.Name, cmd.Architecture, want)
	}
}
