package rootcli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/metrics"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const (
	ErrorCategoryUsage       = clipolicy.ErrorCategoryUsage
	ErrorCategoryEnvironment = clipolicy.ErrorCategoryEnvironment
	ErrorCategoryRuntime     = clipolicy.ErrorCategoryRuntime
)

type GlobalOptions struct {
	JSON         bool
	Verbose      bool
	Quiet        bool
	NoColor      bool
	NoStaleCheck bool
	NoMetrics    bool
}

func (g GlobalOptions) LogFormat() logx.Format {
	if g.JSON {
		return logx.FormatJSON
	}
	return ""
}

// Verbosity controls how much command output reaches the console.
// It is separate from the slog level: it also gates tool stdout
// (vite/pnpm) and the [INFO] step-header printer.
type Verbosity int

const (
	VerbosityQuiet Verbosity = iota
	VerbosityNormal
	VerbosityVerbose
)

// OutputEnvVar names the environment variable that selects a verbosity when
// no flag is provided. Accepted values are "quiet", "normal", and "verbose".
const OutputEnvVar = "VROOLI_OUTPUT"

// Output resolves the effective verbosity using this precedence:
//
//  1. --verbose flag wins over everything (widest surface).
//  2. --quiet flag.
//  3. --json (implies quiet on stdout).
//  4. VROOLI_OUTPUT env var ("quiet"|"normal"|"verbose").
//  5. Default: normal.
//
// An invalid env value falls back to normal; callers that want to warn the
// user should use OutputWarning.
func (g GlobalOptions) Output() Verbosity {
	if g.Verbose {
		return VerbosityVerbose
	}
	if g.Quiet {
		return VerbosityQuiet
	}
	if g.JSON {
		return VerbosityQuiet
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(OutputEnvVar))) {
	case "quiet":
		return VerbosityQuiet
	case "verbose":
		return VerbosityVerbose
	}
	return VerbosityNormal
}

// OutputWarning returns a user-visible warning when flag combinations are
// ambiguous or when VROOLI_OUTPUT holds an unrecognized value. An empty
// string means no warning.
func (g GlobalOptions) OutputWarning() string {
	if g.Verbose && g.Quiet {
		return "both --verbose and --quiet set; --verbose wins"
	}
	raw := strings.TrimSpace(os.Getenv(OutputEnvVar))
	if raw == "" {
		return ""
	}
	switch strings.ToLower(raw) {
	case "quiet", "normal", "verbose":
		return ""
	}
	return fmt.Sprintf("unrecognized %s=%q; falling back to normal", OutputEnvVar, raw)
}

type ParsedArgs struct {
	Command string
	Args    []string
	Globals GlobalOptions
}

func CommandHelpOnly(text string) error {
	return clipolicy.CommandHelpOnly(text)
}

func ParseArgs(args []string) (ParsedArgs, error) {
	parsed := ParsedArgs{}
	for len(args) > 0 {
		switch args[0] {
		case "--json":
			parsed.Globals.JSON = true
			args = args[1:]
		case "--verbose":
			parsed.Globals.Verbose = true
			args = args[1:]
		case "--quiet", "-q":
			parsed.Globals.Quiet = true
			args = args[1:]
		case "--no-color":
			parsed.Globals.NoColor = true
			args = args[1:]
		case "--no-stale-check":
			parsed.Globals.NoStaleCheck = true
			args = args[1:]
		case "--no-metrics":
			parsed.Globals.NoMetrics = true
			args = args[1:]
		case "--help", "-h":
			return ParsedArgs{Command: "help", Globals: parsed.Globals}, nil
		case "--version", "-v":
			return ParsedArgs{Command: "version", Globals: parsed.Globals}, nil
		default:
			parsed.Command = args[0]
			parsed.Args = append([]string(nil), args[1:]...)
			return parsed, nil
		}
	}
	return ParsedArgs{Command: "help", Globals: parsed.Globals}, nil
}

func ConsumeInlineGlobalFlags(globals GlobalOptions, args []string) (GlobalOptions, []string) {
	filtered := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--":
			filtered = append(filtered, args[index:]...)
			return globals, filtered
		case "--json":
			globals.JSON = true
		case "--verbose":
			globals.Verbose = true
		case "--quiet", "-q":
			globals.Quiet = true
		case "--no-color":
			globals.NoColor = true
		case "--no-stale-check":
			globals.NoStaleCheck = true
		case "--no-metrics":
			globals.NoMetrics = true
		default:
			filtered = append(filtered, args[index])
		}
	}
	return globals, filtered
}

func ContainsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

// ContainsFlag reports whether `target` appears as a flag — either as its own
// token (`--format`) or in the `--format=value` long-form. Use this when you
// need to decide whether the user has already supplied a flag in either form.
func ContainsFlag(args []string, target string) bool {
	prefix := target + "="
	for _, arg := range args {
		if arg == target || strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

func PassthroughFlags(globals GlobalOptions, existing []string) []string {
	flags := make([]string, 0, 4)
	if globals.JSON && !ContainsArg(existing, "--json") {
		flags = append(flags, "--json")
	}
	if globals.Verbose && !ContainsArg(existing, "--verbose") {
		flags = append(flags, "--verbose")
	}
	if globals.Quiet && !ContainsArg(existing, "--quiet") && !ContainsArg(existing, "-q") {
		flags = append(flags, "--quiet")
	}
	if globals.NoColor && !ContainsArg(existing, "--no-color") {
		flags = append(flags, "--no-color")
	}
	return flags
}

type Handler[C any] func(ctx C, args []string) error

type ResourceHandler[C any] func(ctx C, controller *resources.Controller, args []string) error

type Registry[C any] struct {
	topLevelTable []commandtree.Spec[Handler[C]]
	scenarioTable []commandtree.Spec[Handler[C]]
	topHandlers   map[string]Handler[C]
	scenarioMap   map[string]Handler[C]
	topSpecs      map[string]commandtree.Spec[Handler[C]]
	scenarioSpecs map[string]commandtree.Spec[Handler[C]]
}

func NewRegistry[C any](
	topLevelHandlers map[topcli.CommandID]Handler[C],
	scenarioHandlers map[scenariocli.CommandID]Handler[C],
) *Registry[C] {
	registry := &Registry[C]{}
	registry.scenarioTable = buildScenarioCommandTable(scenarioHandlers)
	registry.topLevelTable = buildTopLevelCommandTable(topLevelHandlers, func(args []string) bool {
		return registry.ScenarioCanRunWithoutRoot(args)
	})
	registry.topHandlers = commandtree.BuildHandlerMap(registry.topLevelTable)
	registry.scenarioMap = commandtree.BuildHandlerMap(registry.scenarioTable)
	registry.topSpecs = commandtree.BuildSpecMap(registry.topLevelTable)
	registry.scenarioSpecs = commandtree.BuildSpecMap(registry.scenarioTable)
	return registry
}

func buildTopLevelCommandTable[C any](
	handlers map[topcli.CommandID]Handler[C],
	scenarioCanRunWithoutRoot func(args []string) bool,
) []commandtree.Spec[Handler[C]] {
	return commandtree.BindSpecsFunc(topcli.CommandSpecs(), func(spec commandtree.Spec[topcli.CommandID]) (commandtree.Spec[Handler[C]], bool) {
		handler, ok := handlers[spec.Handler]
		if !ok {
			return commandtree.Spec[Handler[C]]{}, false
		}
		bound := commandtree.BindSpec(spec, handler)
		if spec.Handler == topcli.CommandScenario {
			bound.RootPolicy.CanRunWithoutRoot = scenarioCanRunWithoutRoot
		}
		return bound, true
	})
}

func buildScenarioCommandTable[C any](handlers map[scenariocli.CommandID]Handler[C]) []commandtree.Spec[Handler[C]] {
	return commandtree.BindSpecs(scenariocli.CommandSpecs(), handlers)
}

func (r *Registry[C]) TopLevelHandler(name string) (Handler[C], bool) {
	handler, ok := r.topHandlers[commandtree.NormalizeName(name)]
	return handler, ok
}

func (r *Registry[C]) ScenarioHandler(name string) (Handler[C], bool) {
	handler, ok := r.scenarioMap[commandtree.NormalizeName(name)]
	return handler, ok
}

func (r *Registry[C]) TopLevelNames() []string {
	return commandtree.SuggestableNames(r.topLevelTable)
}

func (r *Registry[C]) ScenarioNames() []string {
	return commandtree.SuggestableNames(r.scenarioTable)
}

func (r *Registry[C]) SuggestTopLevel(command string) []string {
	return SuggestCommandNames(command, r.TopLevelNames())
}

func (r *Registry[C]) SuggestScenario(command string) []string {
	return SuggestCommandNames(command, r.ScenarioNames())
}

func (r *Registry[C]) CanRunWithoutRoot(parsed ParsedArgs) bool {
	switch parsed.Command {
	case "help", "version":
		return true
	}
	descriptor, ok := r.topSpecs[commandtree.NormalizeName(parsed.Command)]
	if !ok {
		return true
	}
	if !descriptor.RootPolicy.RequiresRoot {
		return true
	}
	if descriptor.RootPolicy.CanRunWithoutRoot == nil {
		return false
	}
	return descriptor.RootPolicy.CanRunWithoutRoot(parsed.Args)
}

func (r *Registry[C]) ScenarioCanRunWithoutRoot(args []string) bool {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		return true
	}
	descriptor, ok := r.scenarioSpecs[commandtree.NormalizeName(args[0])]
	if !ok {
		return true
	}
	if !descriptor.RootPolicy.RequiresRoot {
		return true
	}
	if descriptor.RootPolicy.CanRunWithoutRoot == nil {
		return false
	}
	return descriptor.RootPolicy.CanRunWithoutRoot(args[1:])
}

func SuggestCommandNames(command string, names []string) []string {
	candidates := make([]string, 0, len(names))
	for _, candidate := range names {
		if candidate == command {
			continue
		}
		if simpleDistance(command, candidate) <= 2 {
			candidates = append(candidates, candidate)
		}
	}
	sort.Strings(candidates)
	return candidates
}

func simpleDistance(left, right string) int {
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}
	minLen := len(left)
	if len(right) < minLen {
		minLen = len(right)
	}
	distance := maxLen - minLen
	for i := 0; i < minLen; i++ {
		if left[i] != right[i] {
			distance++
		}
	}
	return distance
}

func NewErrorWithCategory(err error, category, hint string, suggestions []string) error {
	return clipolicy.NewErrorWithCategory(err, category, hint, suggestions)
}

func UsageHint(helpTarget string) string {
	return clipolicy.UsageHint(helpTarget)
}

func NewUsageError(message, helpTarget string) error {
	return clipolicy.NewUsageError(message, helpTarget)
}

func UsageErrorf(helpTarget, format string, args ...any) error {
	return clipolicy.UsageErrorf(helpTarget, format, args...)
}

func RuntimeErrorf(hint, format string, args ...any) error {
	return NewErrorWithCategory(fmt.Errorf(format, args...), ErrorCategoryRuntime, hint, nil)
}

func UnknownOptionError(command, option string) error {
	return clipolicy.UnknownOptionError(command, option)
}

func NewUnknownCommandError(command string, suggestions []string) error {
	return clipolicy.NewUnknownCommandError(command, suggestions)
}

func NewUnknownScenarioCommandError(command string, suggestions []string) error {
	return clipolicy.NewUnknownScenarioCommandError(command, suggestions)
}

func PrintErrorWithContext(w io.Writer, err error) {
	clipolicy.PrintErrorWithContext(w, err)
}

func WriteHelp(w io.Writer, text string) {
	commandtree.WriteHelp(w, text)
}

func HandleHelp(w io.Writer, err error) bool {
	return commandtree.HandleHelp(w, err)
}

type ExitCodeError struct {
	Code    int
	Message string
	Silent_ bool
}

func (e ExitCodeError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e ExitCodeError) ExitCode() int {
	return e.Code
}

func (e ExitCodeError) Silent() bool {
	return e.Silent_
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if codeErr, ok := err.(interface{ ExitCode() int }); ok {
		return codeErr.ExitCode()
	}
	return vroolierr.ExitCode(err, 1)
}

type RunnerConfig[C any] struct {
	Registry         *Registry[C]
	NewLogger        func(GlobalOptions, io.Writer) (*slog.Logger, func())
	NewContext       func(GlobalOptions, io.Writer, io.Writer, *slog.Logger) C
	SetRoot          func(C, string)
	ResolveRoot      func() (string, error)
	PrimeRootEnv     func(string)
	ShouldRebuild    func() (bool, error)
	RebuildAndReexec func([]string) error
	ShowMainHelp     func(C)
	ShowVersion      func(C) error
	DebugLog         func(*slog.Logger, string, ...any)

	MetricsRecorder *metrics.Recorder
	CLIVersion      string
	PlatformVersion string

	// ScenarioCLILister returns the names of installed scenario/resource
	// CLIs. When set, dispatchInner consults it on the unknown-command path
	// to detect invocations like `vrooli prompt-manager ...` and emit a
	// corrective error pointing at the standalone CLI. May be nil; callers
	// that don't wire it preserve the previous unknown-command behavior.
	// The lister is invoked only on the error path, so callers should
	// memoize internally if the lookup is non-trivial.
	ScenarioCLILister func() []string
}

type Runner[C any] struct {
	config RunnerConfig[C]
}

func NewRunner[C any](config RunnerConfig[C]) *Runner[C] {
	return &Runner[C]{config: config}
}

func (r *Runner[C]) Run(args []string, stdout, stderr io.Writer) int {
	parsed, err := ParseArgs(args)
	if err != nil {
		PrintErrorWithContext(stderr, NewErrorWithCategory(err, ErrorCategoryUsage, clipolicy.GeneralHelpHint, nil))
		return 1
	}
	parsed.Globals, parsed.Args = ConsumeInlineGlobalFlags(parsed.Globals, parsed.Args)
	if warning := parsed.Globals.OutputWarning(); warning != "" {
		fmt.Fprintln(stderr, "warning:", warning)
	}
	// Mirror the resolved verbosity into VROOLI_OUTPUT so render functions
	// and spawned subprocesses share a single source of truth. Done here
	// once — before child processes inherit env — so downstream code can
	// just read the env var instead of threading globals through every
	// call site. Mirrors how NO_COLOR is set a few lines below.
	switch parsed.Globals.Output() {
	case VerbosityQuiet:
		_ = os.Setenv(OutputEnvVar, "quiet")
	case VerbosityVerbose:
		_ = os.Setenv(OutputEnvVar, "verbose")
	default:
		_ = os.Setenv(OutputEnvVar, "normal")
	}
	logger, restoreLogger := r.config.NewLogger(parsed.Globals, stderr)
	defer restoreLogger()
	if r.config.DebugLog != nil {
		r.config.DebugLog(logger, "Parsed command",
			"command", parsed.Command,
			"args", parsed.Args,
			"json", parsed.Globals.JSON,
			"verbose", parsed.Globals.Verbose,
		)
	}

	ctx := r.config.NewContext(parsed.Globals, stdout, stderr, logger)

	if r.config.Registry.CanRunWithoutRoot(parsed) {
		if r.config.ResolveRoot != nil {
			if root, err := r.config.ResolveRoot(); err == nil {
				if r.config.SetRoot != nil {
					r.config.SetRoot(ctx, root)
				}
				if r.config.DebugLog != nil {
					r.config.DebugLog(logger, "Resolved root", "path", root)
				}
				if r.config.PrimeRootEnv != nil {
					r.config.PrimeRootEnv(root)
				}
			}
		}
		if err := r.dispatch(ctx, parsed); err != nil {
			PrintErrorWithContext(stderr, err)
			return ExitCode(err)
		}
		return 0
	}

	root, err := r.config.ResolveRoot()
	if err != nil {
		PrintErrorWithContext(stderr, NewErrorWithCategory(err, ErrorCategoryEnvironment, "Run from a Vrooli repository root or set VROOLI_SOURCE_ROOT", nil))
		return 1
	}
	if r.config.SetRoot != nil {
		r.config.SetRoot(ctx, root)
	}
	if r.config.DebugLog != nil {
		r.config.DebugLog(logger, "Resolved root", "path", root)
	}
	if r.config.PrimeRootEnv != nil {
		r.config.PrimeRootEnv(root)
	}

	if parsed.Globals.NoColor {
		_ = os.Setenv("NO_COLOR", "1")
		if r.config.DebugLog != nil {
			r.config.DebugLog(logger, "NO_COLOR requested by user flags")
		}
	}

	if !parsed.Globals.NoStaleCheck && r.config.ShouldRebuild != nil {
		stale, err := r.config.ShouldRebuild()
		if err != nil {
			PrintErrorWithContext(stderr, NewErrorWithCategory(
				fmt.Errorf("stale binary check failed: %w", err),
				ErrorCategoryRuntime,
				"Use --no-stale-check for local experiments",
				nil,
			))
			return 1
		}
		if stale {
			if r.config.DebugLog != nil {
				r.config.DebugLog(logger, "Stale check triggered")
			}
			if err := r.config.RebuildAndReexec(args); err != nil {
				PrintErrorWithContext(stderr, NewErrorWithCategory(
					fmt.Errorf("stale binary check failed: %w", err),
					ErrorCategoryRuntime,
					"Use --no-stale-check for local experiments",
					nil,
				))
				return 1
			}
			if r.config.DebugLog != nil {
				r.config.DebugLog(logger, "Rebuilt command binary and re-executed")
			}
			return 0
		}
	}

	if err := r.dispatch(ctx, parsed); err != nil {
		PrintErrorWithContext(stderr, err)
		return ExitCode(err)
	}
	return 0
}

func (r *Runner[C]) dispatch(ctx C, parsed ParsedArgs) error {
	start := time.Now()
	err := r.dispatchInner(ctx, parsed)
	r.recordMetrics(parsed, start, err)
	return err
}

func (r *Runner[C]) dispatchInner(ctx C, parsed ParsedArgs) error {
	switch parsed.Command {
	case "help":
		r.config.ShowMainHelp(ctx)
		return nil
	case "version":
		return r.config.ShowVersion(ctx)
	}
	handler, ok := r.config.Registry.TopLevelHandler(parsed.Command)
	if !ok {
		if r.config.ScenarioCLILister != nil {
			for _, name := range r.config.ScenarioCLILister() {
				if name == parsed.Command {
					return clipolicy.NewExternalCLIError(parsed.Command, parsed.Args)
				}
			}
		}
		return NewUnknownCommandError(parsed.Command, r.config.Registry.SuggestTopLevel(parsed.Command))
	}
	return handler(ctx, parsed.Args)
}

func (r *Runner[C]) recordMetrics(parsed ParsedArgs, start time.Time, err error) {
	if r.config.MetricsRecorder == nil || r.config.MetricsRecorder.Disabled() {
		return
	}
	if parsed.Globals.NoMetrics {
		return
	}
	r.config.MetricsRecorder.Record(metrics.Event{
		StartedAt:       start.UTC(),
		Command:         parsed.Command,
		Args:            metrics.RedactArgs(parsed.Args),
		Argc:            len(parsed.Args),
		DurationMs:      time.Since(start).Milliseconds(),
		ExitCode:        ExitCode(err),
		ErrorClass:      metrics.ClassifyError(err),
		CLIVersion:      r.config.CLIVersion,
		PlatformVersion: r.config.PlatformVersion,
		Hostname:        metrics.Hostname(),
		PID:             os.Getpid(),
	})
}

func BindGlobalCommand[C any, Req any, Resp any](
	stdout func(C) io.Writer,
	parse func(C, []string) (Req, error),
	run func(C, Req) (cliout.Format, Resp, error),
	render func(w io.Writer, format cliout.Format, resp Resp) error,
) Handler[C] {
	type output struct {
		format cliout.Format
		resp   Resp
	}
	return func(ctx C, args []string) error {
		return commandtree.ExecuteAction(stdout(ctx), args, commandtree.Action[Req, output]{
			Parse: func(args []string) (Req, error) {
				return parse(ctx, args)
			},
			Execute: func(req Req) (output, error) {
				format, resp, err := run(ctx, req)
				return output{format: format, resp: resp}, err
			},
			Render: func(w io.Writer, item output) error {
				return render(w, item.format, item.resp)
			},
		})
	}
}

func BindContextCommand[C any, Req any, Resp any](
	stdout func(C) io.Writer,
	parse func(C, []string) (Req, error),
	run func(C, Req) (cliout.Format, Resp, error),
	render func(w io.Writer, format cliout.Format, resp Resp) error,
) Handler[C] {
	return BindGlobalCommand(stdout, parse, run, render)
}

func BindResourceCommand[C any, Req any, Resp any](
	stdout func(C) io.Writer,
	parse func(C, []string) (Req, error),
	run func(controller *resources.Controller, ctx C, req Req) (cliout.Format, Resp, error),
	render func(w io.Writer, format cliout.Format, resp Resp) error,
) ResourceHandler[C] {
	type output struct {
		format cliout.Format
		resp   Resp
	}
	return func(ctx C, controller *resources.Controller, args []string) error {
		return commandtree.ExecuteAction(stdout(ctx), args, commandtree.Action[Req, output]{
			Parse: func(args []string) (Req, error) {
				return parse(ctx, args)
			},
			Execute: func(req Req) (output, error) {
				format, resp, err := run(controller, ctx, req)
				return output{format: format, resp: resp}, err
			},
			Render: func(w io.Writer, item output) error {
				return render(w, item.format, item.resp)
			},
		})
	}
}

func RunSubcommandSet[C any](
	ctx C,
	args []string,
	usage func(io.Writer),
	command string,
	handlers map[string]Handler[C],
	stdout func(C) io.Writer,
) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		usage(stdout(ctx))
		return nil
	}
	handler, ok := handlers[commandtree.NormalizeName(args[0])]
	if !ok {
		return UsageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(ctx, args[1:])
}

func RunResourceSubcommandSet[C any](
	ctx C,
	controller *resources.Controller,
	args []string,
	usage func(io.Writer),
	command string,
	handlers map[string]ResourceHandler[C],
	stdout func(C) io.Writer,
) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		usage(stdout(ctx))
		return nil
	}
	handler, ok := handlers[commandtree.NormalizeName(args[0])]
	if !ok {
		return UsageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(ctx, controller, args[1:])
}
