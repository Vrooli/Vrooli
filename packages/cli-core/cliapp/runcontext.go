package cliapp

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrHelpRequested is returned by the parser when the user passes --help/-h.
// The dispatcher catches this sentinel, prints help, and returns nil so the
// CLI exits with status 0.
var ErrHelpRequested = errors.New("help requested")

// OperationContext is the narrow surface a renderer-separated primitive's
// operation callback sees (the `call`/`report` funcs passed to ProtoList,
// ProtoMutation, ProtoOperational, DurableRun, …). It exposes exactly the inputs
// an operation needs to build its request and map its response — parsed
// flags/positionals, flag bindings, and the API client via Core — and NOTHING
// that reveals the output format.
//
// It deliberately omits JSON(), the Render* helpers, and Stdout()/Stderr(): an
// operation callback that cannot observe --json physically cannot branch its
// behavior on the output mode. --json is an output contract owned by the
// primitive (which renders from the single operation result), not an operation
// selector. This is the structural guarantee behind the L4 rung of the
// command-architecture maturity ladder (see
// scenarios/cli-health/docs/reference/cli-architecture-maturity.md) — it is
// enforced at compile time by the callback signatures, not by convention.
//
// Accessors panic on undeclared names. The schema is the source of truth, so a
// typo in a handler ("titel" vs "title") fails fast at the first invocation
// rather than silently returning empty strings.
type OperationContext interface {
	// Schema returns the immutable argument declaration used to parse this
	// invocation. Generic manifest-driven handlers use it to resolve every
	// declared argument against a proto request descriptor.
	Schema() ArgSchema
	// Flag returns the value of a valued flag. If the flag was not provided,
	// returns the schema's Default.
	Flag(name string) string

	// FlagValues returns every supplied value for a valued flag, in order.
	// If the flag was not provided, returns the schema's Default as a
	// single value when set, otherwise nil.
	FlagValues(name string) []string

	// BoolFlag returns true if the flag was provided.
	BoolFlag(name string) bool

	// Positional returns the value of a non-repeated positional. Empty string
	// when the positional was optional and not provided.
	Positional(name string) string

	// Positionals returns all values of a repeated positional, in order.
	Positionals(name string) []string

	// Args returns the raw positional arguments as a slice. Useful for
	// handlers that need a passthrough fallback.
	Args() []string

	// FlagBindings returns the subset of declared flags with non-zero
	// Bind metadata. Generic dispatchers (protodispatch) use this to
	// project --foo-file / --request-payload style flags onto specific
	// proto request fields without per-domain glue code.
	FlagBindings() []FlagBindEntry

	// FlagDeclared reports whether the named flag exists in the schema.
	// Useful for handlers that probe for optional flags declared only by
	// some command variants.
	FlagDeclared(name string) bool

	// FlagProvided reports whether the caller explicitly supplied the flag.
	// Unlike Flag, it distinguishes omission from an intentional empty value.
	FlagProvided(name string) bool

	// Core returns the underlying ScenarioApp for handlers that need direct
	// API access beyond Call[Req,Resp].
	Core() *ScenarioApp
}

// RunContext is the surface a Command's RunCtx handler sees. It is the full
// dispatch surface: an OperationContext (the parsed inputs and API client) plus
// the output-mode selector and renderers that a top-level handler owns.
//
// Renderer-separated primitive operation callbacks receive only the embedded
// OperationContext, never this wider surface, so they cannot reach JSON() or the
// Render* helpers.
type RunContext interface {
	OperationContext

	// JSON reports whether --json was passed (the parser reserves --json
	// as a built-in pseudo-flag for any command using ArgSchema).
	JSON() bool

	// RenderList routes a ListReport to either JSON or human output based on
	// the --json flag.
	RenderList(report ListReport) error

	// RenderMutation routes a MutationReport to either JSON or human output.
	RenderMutation(report MutationReport) error

	// RenderOperational routes an OperationalReport to either JSON or human output.
	RenderOperational(report OperationalReport) error

	// Stdout / Stderr expose the writers; tests inject these to capture
	// output.
	Stdout() io.Writer
	Stderr() io.Writer
}

// runContext is the concrete implementation. The parser builds it; tests
// can build it via NewTestRunContext.
type runContext struct {
	schema      ArgSchema
	flagValues  map[string]string
	flagLists   map[string][]string
	flagSet     map[string]bool
	positionals map[string]string
	repeated    map[string][]string
	rawArgs     []string
	jsonOutput  bool
	core        *ScenarioApp
	stdout      io.Writer
	stderr      io.Writer
}

func (r *runContext) requireFlag(name string) Flag {
	for _, f := range r.schema.Flags {
		if f.Name == name {
			return f
		}
	}
	panic(fmt.Sprintf("RunContext: flag %q not declared in ArgSchema", name))
}

func (r *runContext) requirePositional(name string) Positional {
	for _, p := range r.schema.Positionals {
		if p.Name == name {
			return p
		}
	}
	panic(fmt.Sprintf("RunContext: positional %q not declared in ArgSchema", name))
}

func (r *runContext) Flag(name string) string {
	f := r.requireFlag(name)
	if v, ok := r.flagValues[name]; ok {
		return v
	}
	return f.Default
}

func (r *runContext) FlagValues(name string) []string {
	f := r.requireFlag(name)
	if values := r.flagLists[name]; len(values) > 0 {
		out := make([]string, len(values))
		copy(out, values)
		return out
	}
	if f.Default != "" {
		return []string{f.Default}
	}
	return nil
}

func (r *runContext) BoolFlag(name string) bool {
	r.requireFlag(name)
	return r.flagSet[name]
}

func (r *runContext) Positional(name string) string {
	r.requirePositional(name)
	return r.positionals[name]
}

func (r *runContext) Positionals(name string) []string {
	p := r.requirePositional(name)
	if !p.Repeated {
		panic(fmt.Sprintf("RunContext: Positionals(%q) called on non-repeated positional", name))
	}
	out := make([]string, len(r.repeated[name]))
	copy(out, r.repeated[name])
	return out
}

func (r *runContext) Args() []string {
	out := make([]string, len(r.rawArgs))
	copy(out, r.rawArgs)
	return out
}

func (r *runContext) JSON() bool { return r.jsonOutput }

func (r *runContext) RenderList(report ListReport) error {
	if r.jsonOutput {
		return PrintReportJSON(r.stdout, report)
	}
	return RenderListReport(r.stdout, report)
}

func (r *runContext) RenderMutation(report MutationReport) error {
	if r.jsonOutput {
		return PrintReportJSON(r.stdout, report)
	}
	return RenderMutationReport(r.stdout, report)
}

func (r *runContext) RenderOperational(report OperationalReport) error {
	if r.jsonOutput {
		return PrintReportJSON(r.stdout, report)
	}
	return RenderOperationalReport(r.stdout, report)
}

func (r *runContext) Core() *ScenarioApp { return r.core }

func (r *runContext) Schema() ArgSchema { return r.schema }

// FlagBindEntry pairs a declared flag's name with its parsed bind metadata
// and whether the user supplied the flag (BoolFlag-style "provided" signal).
type FlagBindEntry struct {
	Name     string
	Bind     FlagBind
	Bool     bool
	Provided bool
	Value    string
}

func (r *runContext) FlagBindings() []FlagBindEntry {
	var out []FlagBindEntry
	for _, f := range r.schema.Flags {
		if f.Bind.IsZero() {
			continue
		}
		entry := FlagBindEntry{
			Name: f.Name,
			Bind: f.Bind,
			Bool: f.Bool,
		}
		if v, ok := r.flagValues[f.Name]; ok {
			entry.Value = v
			entry.Provided = true
		} else if r.flagSet[f.Name] {
			entry.Provided = true
		} else if f.Default != "" {
			entry.Value = f.Default
		}
		out = append(out, entry)
	}
	return out
}

func (r *runContext) FlagDeclared(name string) bool {
	for _, f := range r.schema.Flags {
		if f.Name == name {
			return true
		}
		for _, a := range f.Aliases {
			if a == name {
				return true
			}
		}
	}
	return false
}

func (r *runContext) FlagProvided(name string) bool {
	r.requireFlag(name)
	if _, ok := r.flagValues[name]; ok {
		return true
	}
	if len(r.flagLists[name]) > 0 {
		return true
	}
	return r.flagSet[name]
}

func (r *runContext) Stdout() io.Writer {
	if r.stdout == nil {
		return os.Stdout
	}
	return r.stdout
}

func (r *runContext) Stderr() io.Writer {
	if r.stderr == nil {
		return os.Stderr
	}
	return r.stderr
}

// TestRunContextOptions configures NewTestRunContext.
type TestRunContextOptions struct {
	Schema      ArgSchema
	Flags       map[string]string
	FlagLists   map[string][]string
	BoolFlags   map[string]bool
	Positionals map[string]string
	Repeated    map[string][]string
	RawArgs     []string
	JSON        bool
	Core        *ScenarioApp
	Stdout      io.Writer
	Stderr      io.Writer
}

// NewTestRunContext constructs a RunContext for unit tests that need to
// drive a handler without going through the full parser. Useful when the
// test wants to populate flag/positional values directly without composing
// raw argv.
//
// For tests that need parity with the production dispatcher (e.g., to
// verify Required:true enforcement), use NewTestRunContextFromArgs
// instead — that path runs the actual parser.
//
// Production code does not use either helper; the dispatcher builds the
// RunContext via parseArgs.
func NewTestRunContext(opts TestRunContextOptions) RunContext {
	flagSet := make(map[string]bool, len(opts.Flags)+len(opts.BoolFlags))
	flagValues := make(map[string]string, len(opts.Flags))
	flagLists := make(map[string][]string, len(opts.Flags)+len(opts.FlagLists))
	for k, v := range opts.Flags {
		flagValues[k] = v
		flagLists[k] = []string{v}
		flagSet[k] = true
	}
	for k, v := range opts.FlagLists {
		flagLists[k] = append([]string(nil), v...)
		if len(v) > 0 {
			flagValues[k] = v[len(v)-1]
			flagSet[k] = true
		}
	}
	for k, v := range opts.BoolFlags {
		if v {
			flagSet[k] = true
		}
	}
	positionals := make(map[string]string, len(opts.Positionals))
	for k, v := range opts.Positionals {
		positionals[k] = v
	}
	repeated := make(map[string][]string, len(opts.Repeated))
	for k, v := range opts.Repeated {
		repeated[k] = append([]string(nil), v...)
	}
	rawArgs := append([]string(nil), opts.RawArgs...)
	return &runContext{
		schema:      opts.Schema,
		flagValues:  flagValues,
		flagLists:   flagLists,
		flagSet:     flagSet,
		positionals: positionals,
		repeated:    repeated,
		rawArgs:     rawArgs,
		jsonOutput:  opts.JSON,
		core:        opts.Core,
		stdout:      opts.Stdout,
		stderr:      opts.Stderr,
	}
}

// NewTestRunContextFromArgs runs the production parser against the given
// argv slice and returns the resulting RunContext. Tests use this when
// they want parity with the dispatcher path — for example, to verify that
// a schema's Required:true triggers a "missing required flag" error
// rather than reaching the handler.
//
// The returned error is the parser's error (or ErrHelpRequested if the
// argv contained --help/-h). On success, RunContext.Stdout()/Stderr()
// fall back to os.Stdout/os.Stderr if the caller passes nil writers; pass
// a *bytes.Buffer to capture rendered output.
func NewTestRunContextFromArgs(schema ArgSchema, args []string, core *ScenarioApp, stdout, stderr io.Writer) (RunContext, error) {
	return parseArgs(schema, args, core, stdout, stderr)
}
