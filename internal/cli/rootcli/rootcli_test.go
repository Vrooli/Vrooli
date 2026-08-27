package rootcli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

func TestBindService(t *testing.T) {
	t.Run("constructor error", func(t *testing.T) {
		var parsed bool
		wantErr := errors.New("construct failed")
		handler := BindService(
			func(struct{}) io.Writer { return io.Discard },
			func(struct{}) (cliout.Format, struct{}, error) { return "", struct{}{}, wantErr },
			func(struct{}, []string) (struct{}, error) {
				parsed = true
				return struct{}{}, nil
			},
			func(struct{}, struct{}) (string, error) { return "", nil },
			func(io.Writer, cliout.Format, string) error { return nil },
		)
		if err := handler(struct{}{}, nil); !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
		if parsed {
			t.Fatal("parser ran after service construction failed")
		}
	})

	t.Run("service constructs before parse error", func(t *testing.T) {
		order := make([]string, 0, 2)
		wantErr := errors.New("parse failed")
		handler := BindService(
			func(struct{}) io.Writer { return io.Discard },
			func(struct{}) (cliout.Format, string, error) {
				order = append(order, "construct")
				return cliout.FormatJSON, "service", nil
			},
			func(struct{}, []string) (int, error) {
				order = append(order, "parse")
				return 0, wantErr
			},
			func(string, int) (string, error) { return "", nil },
			func(io.Writer, cliout.Format, string) error { return nil },
		)
		if err := handler(struct{}{}, nil); !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
		if got, want := strings.Join(order, ","), "construct,parse"; got != want {
			t.Fatalf("order = %q, want %q", got, want)
		}
	})

	t.Run("call error", func(t *testing.T) {
		wantErr := errors.New("call failed")
		handler := BindService(
			func(struct{}) io.Writer { return io.Discard },
			func(struct{}) (cliout.Format, string, error) { return cliout.FormatJSON, "service", nil },
			func(struct{}, []string) (int, error) { return 7, nil },
			func(string, int) (string, error) { return "", wantErr },
			func(io.Writer, cliout.Format, string) error { return nil },
		)
		if err := handler(struct{}{}, nil); !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})

	t.Run("success renders constructor format", func(t *testing.T) {
		var gotFormat cliout.Format
		var gotResponse string
		handler := BindService(
			func(struct{}) io.Writer { return io.Discard },
			func(struct{}) (cliout.Format, int, error) { return cliout.FormatJSON, 3, nil },
			func(struct{}, []string) (string, error) { return "request", nil },
			func(service int, request string) (string, error) {
				return request + "-" + fmt.Sprint(service), nil
			},
			func(_ io.Writer, format cliout.Format, response string) error {
				gotFormat = format
				gotResponse = response
				return nil
			},
		)
		if err := handler(struct{}{}, nil); err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if gotFormat != cliout.FormatJSON || gotResponse != "request-3" {
			t.Fatalf("rendered format/response = %q/%q", gotFormat, gotResponse)
		}
	})
}

func TestParseArgsSeparatesGlobalsAndCommand(t *testing.T) {
	parsed, err := ParseArgs([]string{"--json", "--verbose", "scenario", "list"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if parsed.Command != "scenario" {
		t.Fatalf("command = %q, want scenario", parsed.Command)
	}
	if !parsed.Globals.JSON || !parsed.Globals.Verbose {
		t.Fatalf("globals = %#v, want json and verbose", parsed.Globals)
	}
	if got := strings.Join(parsed.Args, "|"); got != "list" {
		t.Fatalf("args = %q, want list", got)
	}
}

func TestConsumeInlineGlobalFlagsPreservesPassthroughTail(t *testing.T) {
	globals, args := ConsumeInlineGlobalFlags(GlobalOptions{}, []string{"list", "--json", "--", "--json"})
	if !globals.JSON {
		t.Fatalf("globals = %#v, want JSON enabled", globals)
	}
	if got := strings.Join(args, "|"); got != "list|--|--json" {
		t.Fatalf("args = %q", got)
	}
}

func TestRegistryCanRunWithoutRoot(t *testing.T) {
	registry := NewRegistry(
		map[topcli.CommandID]Handler[struct{}]{
			topcli.CommandSetup:    func(struct{}, []string) error { return nil },
			topcli.CommandScenario: func(struct{}, []string) error { return nil },
		},
		map[scenariocli.CommandID]Handler[struct{}]{
			scenariocli.CommandList: func(struct{}, []string) error { return nil },
		},
	)
	if !registry.CanRunWithoutRoot(ParsedArgs{Command: "help"}) {
		t.Fatal("help should run without root")
	}
	if registry.CanRunWithoutRoot(ParsedArgs{Command: "setup", Args: []string{"--dry-run"}}) {
		t.Fatal("setup should require root resolution")
	}
	if !registry.CanRunWithoutRoot(ParsedArgs{Command: "scenario", Args: []string{"--help"}}) {
		t.Fatal("scenario help should run without root")
	}
}

func TestPrintErrorWithContextUnknownCommandIncludesSuggestions(t *testing.T) {
	err := NewUnknownCommandError("statsu", []string{"status"})
	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, err)

	output := stderr.String()
	if !strings.Contains(output, clipolicy.UnknownCommandLabel+": statsu") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "status") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, clipolicy.MainHelpHint) {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintErrorWithContextUnknownScenarioCommandIncludesSuggestions(t *testing.T) {
	err := NewUnknownScenarioCommandError("statsu", []string{"status"})
	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, err)

	output := stderr.String()
	if !strings.Contains(output, clipolicy.UnknownScenarioCommandLabel+": statsu") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "status") {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintErrorWithContextCategorizedRuntimeError(t *testing.T) {
	err := NewErrorWithCategory(
		errors.New("pipeline failed"),
		ErrorCategoryRuntime,
		"Check the command inputs and try again.",
		[]string{"setup", "status"},
	)

	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, err)

	output := stderr.String()
	if !strings.Contains(output, "Runtime error: pipeline failed") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Check the command inputs and try again.") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Did you mean one of these?") {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintErrorWithContextPreservesPlainErrors(t *testing.T) {
	var stderr bytes.Buffer
	PrintErrorWithContext(&stderr, errors.New("plain failure"))

	if got := strings.TrimSpace(stderr.String()); got != "plain failure" {
		t.Fatalf("output = %q", got)
	}
}

func TestExitCodePrefersTypedExitCodes(t *testing.T) {
	if code := ExitCode(nil); code != 0 {
		t.Fatalf("ExitCode(nil) = %d", code)
	}
	if code := ExitCode(ExitCodeError{Code: 23}); code != 23 {
		t.Fatalf("ExitCode(typed) = %d", code)
	}
	if code := ExitCode(errors.New("boom")); code != 1 {
		t.Fatalf("ExitCode(default) = %d", code)
	}
}

func TestExitCodeErrorFormatting(t *testing.T) {
	if got := (ExitCodeError{Code: 7, Message: "boom"}).Error(); got != "boom" {
		t.Fatalf("ExitCodeError message = %q", got)
	}
	if got := (ExitCodeError{Code: 7}).Error(); got != "exit code 7" {
		t.Fatalf("ExitCodeError default = %q", got)
	}
}

func TestNewErrorWithCategoryPreservesExitCode(t *testing.T) {
	err := NewErrorWithCategory(ExitCodeError{Code: 23, Message: "wrapped"}, ErrorCategoryRuntime, "", nil)
	if got := ExitCode(err); got != 23 {
		t.Fatalf("ExitCode(err) = %d, want 23", got)
	}
}

func TestPassthroughFlagsOmitsExistingFlags(t *testing.T) {
	flags := PassthroughFlags(GlobalOptions{JSON: true, Verbose: true, NoColor: true}, []string{"--json", "scenario"})
	if got, want := strings.Join(flags, ","), "--verbose,--no-color"; got != want {
		t.Fatalf("flags = %q, want %q", got, want)
	}
}

func TestPassthroughFlagsIncludesQuiet(t *testing.T) {
	flags := PassthroughFlags(GlobalOptions{Quiet: true}, []string{"scenario"})
	if got, want := strings.Join(flags, ","), "--quiet"; got != want {
		t.Fatalf("flags = %q, want %q", got, want)
	}
	flags = PassthroughFlags(GlobalOptions{Quiet: true}, []string{"scenario", "-q"})
	if len(flags) != 0 {
		t.Fatalf("flags = %v, want empty (already has -q)", flags)
	}
}

func TestParseArgsAcceptsQuietShortAndLong(t *testing.T) {
	for _, flag := range []string{"--quiet", "-q"} {
		parsed, err := ParseArgs([]string{flag, "scenario", "list"})
		if err != nil {
			t.Fatalf("%s: ParseArgs returned error: %v", flag, err)
		}
		if !parsed.Globals.Quiet {
			t.Fatalf("%s: expected Quiet=true, got %#v", flag, parsed.Globals)
		}
	}
}

func TestGlobalOptionsOutputPrecedence(t *testing.T) {
	// Clear env to isolate from host
	t.Setenv(OutputEnvVar, "")

	cases := []struct {
		name string
		in   GlobalOptions
		want Verbosity
	}{
		{"default", GlobalOptions{}, VerbosityNormal},
		{"quiet", GlobalOptions{Quiet: true}, VerbosityQuiet},
		{"verbose", GlobalOptions{Verbose: true}, VerbosityVerbose},
		{"verbose beats quiet", GlobalOptions{Verbose: true, Quiet: true}, VerbosityVerbose},
		{"json implies quiet", GlobalOptions{JSON: true}, VerbosityQuiet},
		{"verbose beats json", GlobalOptions{JSON: true, Verbose: true}, VerbosityVerbose},
		{"quiet beats json", GlobalOptions{JSON: true, Quiet: true}, VerbosityQuiet},
	}
	for _, tc := range cases {
		if got := tc.in.Output(); got != tc.want {
			t.Errorf("%s: Output() = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestGlobalOptionsOutputReadsEnv(t *testing.T) {
	t.Setenv(OutputEnvVar, "quiet")
	if got := (GlobalOptions{}).Output(); got != VerbosityQuiet {
		t.Errorf("env quiet: Output() = %d, want VerbosityQuiet", got)
	}
	t.Setenv(OutputEnvVar, "verbose")
	if got := (GlobalOptions{}).Output(); got != VerbosityVerbose {
		t.Errorf("env verbose: Output() = %d, want VerbosityVerbose", got)
	}
	// Flag beats env.
	t.Setenv(OutputEnvVar, "verbose")
	if got := (GlobalOptions{Quiet: true}).Output(); got != VerbosityQuiet {
		t.Errorf("flag quiet + env verbose: Output() = %d, want VerbosityQuiet", got)
	}
}

func TestGlobalOptionsOutputInvalidEnvFallsBack(t *testing.T) {
	t.Setenv(OutputEnvVar, "loud")
	if got := (GlobalOptions{}).Output(); got != VerbosityNormal {
		t.Errorf("Output() = %d, want VerbosityNormal", got)
	}
	warn := (GlobalOptions{}).OutputWarning()
	if !strings.Contains(warn, "unrecognized") || !strings.Contains(warn, "loud") {
		t.Errorf("OutputWarning() = %q, want mention of unrecognized value", warn)
	}
}

func TestGlobalOptionsOutputWarningOnConflict(t *testing.T) {
	t.Setenv(OutputEnvVar, "")
	warn := GlobalOptions{Verbose: true, Quiet: true}.OutputWarning()
	if !strings.Contains(warn, "verbose") || !strings.Contains(warn, "quiet") {
		t.Errorf("OutputWarning() = %q, want mention of flag conflict", warn)
	}
	if warn := (GlobalOptions{}).OutputWarning(); warn != "" {
		t.Errorf("OutputWarning() on clean globals = %q, want empty", warn)
	}
}

func TestContainsArgDoesNotMatchAbsentFlag(t *testing.T) {
	if ContainsArg([]string{"alpha", "beta"}, "--json") {
		t.Fatal("ContainsArg() should not match absent flag")
	}
}

func TestContainsFlagMatchesBothForms(t *testing.T) {
	cases := []struct {
		args   []string
		target string
		want   bool
	}{
		{[]string{"--format", "json"}, "--format", true},
		{[]string{"--format=yaml"}, "--format", true},
		{[]string{"--format=json"}, "--json", false},
		{[]string{"--json"}, "--json", true},
		{[]string{"--jsonish"}, "--json", false},
		{[]string{}, "--json", false},
	}
	for _, c := range cases {
		if got := ContainsFlag(c.args, c.target); got != c.want {
			t.Errorf("ContainsFlag(%v, %q) = %v, want %v", c.args, c.target, got, c.want)
		}
	}
}

type runnerCtx struct {
	root string
}

func TestRunResolvesRootEvenForHelpOnlyCommands(t *testing.T) {
	captured := runnerCtx{}
	resolveCalls := 0
	primeCalls := 0

	registry := NewRegistry(
		map[topcli.CommandID]Handler[*runnerCtx]{
			topcli.CommandScenario: func(ctx *runnerCtx, args []string) error { return nil },
		},
		map[scenariocli.CommandID]Handler[*runnerCtx]{
			scenariocli.CommandRequirements: func(ctx *runnerCtx, args []string) error {
				if ctx.root == "" {
					t.Errorf("handler received empty root; args=%v", args)
				}
				return nil
			},
		},
	)

	runner := NewRunner(RunnerConfig[*runnerCtx]{
		Registry: registry,
		NewLogger: func(GlobalOptions, io.Writer) (*slog.Logger, func()) {
			return slog.New(slog.NewTextHandler(io.Discard, nil)), func() {}
		},
		NewContext: func(GlobalOptions, io.Writer, io.Writer, *slog.Logger) *runnerCtx {
			return &captured
		},
		SetRoot: func(ctx *runnerCtx, root string) { ctx.root = root },
		ResolveRoot: func() (string, error) {
			resolveCalls++
			return "/resolved/root", nil
		},
		PrimeRootEnv: func(string) { primeCalls++ },
		ShowMainHelp: func(*runnerCtx) {},
		ShowVersion:  func(*runnerCtx) error { return nil },
	})

	var stdout, stderr bytes.Buffer
	code := runner.Run([]string{"scenario", "requirements", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if resolveCalls != 1 {
		t.Fatalf("ResolveRoot calls = %d, want 1", resolveCalls)
	}
	if primeCalls != 1 {
		t.Fatalf("PrimeRootEnv calls = %d, want 1", primeCalls)
	}
	if captured.root != "/resolved/root" {
		t.Fatalf("ctx root = %q, want /resolved/root", captured.root)
	}
}

func TestRunHelpOnlyTolerantOfResolveRootError(t *testing.T) {
	captured := runnerCtx{}

	registry := NewRegistry(
		map[topcli.CommandID]Handler[*runnerCtx]{
			topcli.CommandScenario: func(*runnerCtx, []string) error { return nil },
		},
		map[scenariocli.CommandID]Handler[*runnerCtx]{
			scenariocli.CommandRequirements: func(*runnerCtx, []string) error { return nil },
		},
	)

	runner := NewRunner(RunnerConfig[*runnerCtx]{
		Registry: registry,
		NewLogger: func(GlobalOptions, io.Writer) (*slog.Logger, func()) {
			return slog.New(slog.NewTextHandler(io.Discard, nil)), func() {}
		},
		NewContext: func(GlobalOptions, io.Writer, io.Writer, *slog.Logger) *runnerCtx {
			return &captured
		},
		SetRoot:      func(ctx *runnerCtx, root string) { ctx.root = root },
		ResolveRoot:  func() (string, error) { return "", errors.New("no root") },
		ShowMainHelp: func(*runnerCtx) {},
		ShowVersion:  func(*runnerCtx) error { return nil },
	})

	var stdout, stderr bytes.Buffer
	if code := runner.Run([]string{"scenario", "requirements", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if captured.root != "" {
		t.Fatalf("ctx root = %q, want empty (resolve failed)", captured.root)
	}
}

func newScenarioCLILookalikeRunner(lister func() []string) *Runner[*runnerCtx] {
	registry := NewRegistry(
		map[topcli.CommandID]Handler[*runnerCtx]{
			topcli.CommandScenario: func(*runnerCtx, []string) error { return nil },
		},
		map[scenariocli.CommandID]Handler[*runnerCtx]{},
	)
	captured := &runnerCtx{}
	return NewRunner(RunnerConfig[*runnerCtx]{
		Registry: registry,
		NewLogger: func(GlobalOptions, io.Writer) (*slog.Logger, func()) {
			return slog.New(slog.NewTextHandler(io.Discard, nil)), func() {}
		},
		NewContext: func(GlobalOptions, io.Writer, io.Writer, *slog.Logger) *runnerCtx {
			return captured
		},
		SetRoot:           func(ctx *runnerCtx, root string) { ctx.root = root },
		ResolveRoot:       func() (string, error) { return "/resolved", nil },
		PrimeRootEnv:      func(string) {},
		ShowMainHelp:      func(*runnerCtx) {},
		ShowVersion:       func(*runnerCtx) error { return nil },
		ScenarioCLILister: lister,
	})
}

func TestDispatchEmitsExternalCLIErrorForInstalledScenarioCLI(t *testing.T) {
	runner := newScenarioCLILookalikeRunner(func() []string {
		return []string{"agent-manager", "prompt-manager"}
	})
	var stdout, stderr bytes.Buffer
	code := runner.Run([]string{"prompt-manager", "skill", "read", "x"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	out := stderr.String()
	if !strings.Contains(out, "'prompt-manager' is a scenario CLI") {
		t.Errorf("expected external-CLI message; got:\n%s", out)
	}
	if !strings.Contains(out, "prompt-manager skill read x") {
		t.Errorf("expected corrected command; got:\n%s", out)
	}
}

func TestDispatchFallsBackToUnknownCommandWhenNotScenarioCLI(t *testing.T) {
	runner := newScenarioCLILookalikeRunner(func() []string {
		return []string{"prompt-manager"}
	})
	var stdout, stderr bytes.Buffer
	code := runner.Run([]string{"definitely-not-a-thing"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	out := stderr.String()
	if !strings.Contains(out, "Unknown command: definitely-not-a-thing") {
		t.Errorf("expected unknown-command path; got:\n%s", out)
	}
	if strings.Contains(out, "scenario CLI") {
		t.Errorf("did not expect scenario-CLI message; got:\n%s", out)
	}
}

func TestDispatchUnknownCommandWhenListerNil(t *testing.T) {
	runner := newScenarioCLILookalikeRunner(nil)
	var stdout, stderr bytes.Buffer
	code := runner.Run([]string{"prompt-manager"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	out := stderr.String()
	if !strings.Contains(out, "Unknown command: prompt-manager") {
		t.Errorf("expected unknown-command path; got:\n%s", out)
	}
}

func TestExternalCLIErrorCarriesCode(t *testing.T) {
	err := clipolicy.NewExternalCLIError("prompt-manager", []string{"skill"})
	if got := vroolierr.Code(err, ""); got != clipolicy.CodeExternalCLIInvocation {
		t.Fatalf("code = %q, want %q", got, clipolicy.CodeExternalCLIInvocation)
	}
}
