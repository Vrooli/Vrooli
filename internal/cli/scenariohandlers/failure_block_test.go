package scenariohandlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

type failureCtx struct {
	stderr  *bytes.Buffer
	home    string
	globals rootcli.GlobalOptions
}

func newFailureDeps(home string) rootcli.HandlerDeps[*failureCtx] {
	return rootcli.HandlerDeps[*failureCtx]{
		Stderr:  func(c *failureCtx) io.Writer { return c.stderr },
		Globals: func(c *failureCtx) rootcli.GlobalOptions { return c.globals },
		HomeDir: func(c *failureCtx) (string, error) {
			if home == "" {
				return c.home, nil
			}
			return home, nil
		},
	}
}

func TestEmitLifecycleFailureWritesBlock(t *testing.T) {
	stderr := &bytes.Buffer{}
	home := t.TempDir()
	ctx := &failureCtx{stderr: stderr, home: home}

	emitLifecycleFailure(newFailureDeps(home), ctx, "restart", []string{"alpha"}, errors.New("boom"))

	got := stderr.String()
	if !strings.HasPrefix(got, "\n") {
		t.Errorf("failure block missing leading blank line: %q", got)
	}
	for _, want := range []string{
		"✗ Failed to restart 'alpha'",
		"Error: boom",
		"Full log: " + filepath.Join(home, ".vrooli", "logs", "alpha.log"),
		"Tail:     vrooli scenario logs alpha --tail 100",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// Sanity: Full log path matches the canonical helper so a rename
	// over in internal/process would trip the test.
	logPath, err := process.ScenarioLifecycleLogPath(home, "alpha")
	if err != nil {
		t.Fatalf("ScenarioLifecycleLogPath: %v", err)
	}
	if !strings.Contains(got, logPath) {
		t.Errorf("log path did not match process.ScenarioLifecycleLogPath: %q", got)
	}
}

func TestEmitLifecycleFailureWritesPhaseStepExitDetails(t *testing.T) {
	stderr := &bytes.Buffer{}
	home := t.TempDir()
	ctx := &failureCtx{stderr: stderr, home: home}
	err := &lifecycle.PhaseStepError{
		Scenario: "alpha",
		Phase:    "setup",
		Step:     "build-ui",
		Exit:     143,
		Err:      errors.New("signal: terminated"),
	}

	emitLifecycleFailure(newFailureDeps(home), ctx, "restart", []string{"alpha"}, err)

	got := stderr.String()
	for _, want := range []string{
		"Phase: setup",
		"Step: build-ui",
		"Exit code: 143",
		"Error: scenario \"alpha\" phase \"setup\" step \"build-ui\" failed with exit code 143",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestEmitLifecycleFailureSuppressedInJSONMode(t *testing.T) {
	stderr := &bytes.Buffer{}
	ctx := &failureCtx{stderr: stderr, globals: rootcli.GlobalOptions{JSON: true}}

	emitLifecycleFailure(newFailureDeps(t.TempDir()), ctx, "start", []string{"alpha"}, errors.New("boom"))

	if stderr.Len() != 0 {
		t.Fatalf("JSON mode must not emit human failure block, got %q", stderr.String())
	}
}

func TestEmitLifecycleFailureNoopOnNilError(t *testing.T) {
	stderr := &bytes.Buffer{}
	ctx := &failureCtx{stderr: stderr}

	emitLifecycleFailure(newFailureDeps(t.TempDir()), ctx, "stop", []string{"alpha"}, nil)

	if stderr.Len() != 0 {
		t.Fatalf("nil error must not emit block, got %q", stderr.String())
	}
}

func TestShortErrorMessageTrimsMultilineAndCaps(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "nil-like empty", in: "", want: ""},
		{name: "single line trimmed", in: "  boom  ", want: "boom"},
		{name: "first line only", in: "first\nsecond\nthird", want: "first"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.in != "" {
				err = errors.New(tc.in)
			}
			if got := shortErrorMessage(err); got != tc.want {
				t.Fatalf("shortErrorMessage(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	// Capped: build a >240 char single-line error and confirm it
	// ends with an ellipsis so the failure block stays readable.
	long := strings.Repeat("a", 500)
	got := shortErrorMessage(errors.New(long))
	if len(got) == 500 {
		t.Fatalf("long error not capped: len=%d", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("long error missing ellipsis: %q", got)
	}
}

func TestCleanFailureNames(t *testing.T) {
	got := cleanFailureNames([]string{" alpha ", "", "alpha", "beta", "  "})
	want := []string{"alpha", "beta"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSilentLifecycleErrorContract(t *testing.T) {
	inner := &vroolierr.Error{Err: errors.New("boom"), Category: "runtime", Exit: 42}
	wrapped := silentLifecycleError{inner: inner}

	if !wrapped.Silent() {
		t.Fatalf("wrapper must report Silent()=true")
	}
	if got := wrapped.ExitCode(); got != 42 {
		t.Fatalf("exit code = %d, want 42 (preserved from inner)", got)
	}
	if got := wrapped.ErrorCategory(); got != "runtime" {
		t.Fatalf("category = %q, want runtime", got)
	}
	if !errors.Is(wrapped, inner) {
		t.Fatalf("errors.Is must unwrap through wrapper")
	}
}

func TestWithLifecycleFailureBlockWrapsErrorAsSilent(t *testing.T) {
	stderr := &bytes.Buffer{}
	home := t.TempDir()
	ctx := &failureCtx{stderr: stderr, home: home}

	call := withLifecycleFailureBlock(
		newFailureDeps(home),
		"start",
		func(req string) []string { return []string{req} },
		func(ctx *failureCtx, _ cliout.Format, req string) ([]LifecycleItemOutput, error) {
			return nil, errors.New("boom")
		},
	)

	_, err := call(ctx, cliout.FormatHuman, "alpha")
	if err == nil {
		t.Fatal("expected error propagation")
	}
	if silent, ok := err.(interface{ Silent() bool }); !ok || !silent.Silent() {
		t.Fatalf("wrapped error must be silent: %+v", err)
	}
	if !strings.Contains(stderr.String(), "✗ Failed to start 'alpha'") {
		t.Fatalf("failure block missing from stderr: %q", stderr.String())
	}
	if got := strings.Count(stderr.String(), "✗ Failed to start 'alpha'"); got != 1 {
		t.Fatalf("failure block count = %d, want 1: %q", got, stderr.String())
	}
}

func TestLifecycleFailureReachesCLIJSONBoundaryButHumanBlockStaysSingle(t *testing.T) {
	wantErr := vroolierr.New("scenario_cli_refresh_failed", rootcli.ErrorCategoryRuntime, "scenario CLI refresh failed")
	for _, asJSON := range []bool{false, true} {
		mode := "human"
		args := []string{"scenario"}
		if asJSON {
			mode = "json"
			args = []string{"--json", "scenario"}
		}
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			deps := newFailureDeps(home)
			deps.EnsureCLI = func(*failureCtx, string) error { return wantErr }
			registry := rootcli.NewRegistry(
				map[topcli.CommandID]rootcli.Handler[*failureCtx]{
					topcli.CommandScenario: func(ctx *failureCtx, _ []string) error {
						format := cliout.FormatHuman
						if ctx.globals.JSON {
							format = cliout.FormatJSON
						}
						call := withLifecycleFailureBlock(
							deps,
							"start",
							func(req string) []string { return []string{req} },
							func(runCtx *failureCtx, _ cliout.Format, _ string) ([]LifecycleItemOutput, error) {
								if err := ensureScenarioCLIs(deps, runCtx, "alpha"); err != nil {
									return nil, err
								}
								return nil, errors.New("scenario restart unexpectedly passed CLI preflight")
							},
						)
						_, err := call(ctx, format, "alpha")
						return err
					},
				},
				nil,
			)
			runner := rootcli.NewRunner(rootcli.RunnerConfig[*failureCtx]{
				Registry: registry,
				NewLogger: func(rootcli.GlobalOptions, io.Writer) (*slog.Logger, func()) {
					return slog.New(slog.NewTextHandler(io.Discard, nil)), func() {}
				},
				NewContext: func(globals rootcli.GlobalOptions, _ io.Writer, stderr io.Writer, _ *slog.Logger) *failureCtx {
					return &failureCtx{stderr: stderr.(*bytes.Buffer), home: home, globals: globals}
				},
				ResolveRoot:  func() (string, error) { return "/resolved", nil },
				ShowMainHelp: func(*failureCtx) {},
				ShowVersion:  func(*failureCtx) error { return nil },
			})

			var stdout, stderr bytes.Buffer
			if code := runner.Run(args, &stdout, &stderr); code != 1 {
				t.Fatalf("exit code = %d, want 1; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("failure unexpectedly wrote stdout: %q", stdout.String())
			}
			if asJSON {
				var payload struct {
					Success  bool   `json:"success"`
					Error    string `json:"error"`
					Code     string `json:"code"`
					Category string `json:"category"`
				}
				if err := json.Unmarshal(stderr.Bytes(), &payload); err != nil {
					t.Fatalf("JSON boundary error: %v; stderr=%q", err, stderr.String())
				}
				if payload.Success || payload.Code != "scenario_cli_refresh_failed" || payload.Category != rootcli.ErrorCategoryRuntime || payload.Error != wantErr.Error() {
					t.Fatalf("JSON error payload = %+v", payload)
				}
				return
			}
			if got := strings.Count(stderr.String(), "✗ Failed to start 'alpha'"); got != 1 {
				t.Fatalf("human failure block count = %d, want 1: %q", got, stderr.String())
			}
			if strings.Contains(stderr.String(), "scenario_cli_refresh_failed") {
				t.Fatalf("human mode duplicated boundary error after failure block: %q", stderr.String())
			}
		})
	}
}

func TestWithLifecycleFailureBlockPassesThroughSuccess(t *testing.T) {
	ctx := &failureCtx{stderr: &bytes.Buffer{}}
	wantItems := []LifecycleItemOutput{{Name: "alpha", Status: "started"}}

	call := withLifecycleFailureBlock(
		newFailureDeps(t.TempDir()),
		"start",
		func(req string) []string { return []string{req} },
		func(ctx *failureCtx, _ cliout.Format, req string) ([]LifecycleItemOutput, error) {
			return wantItems, nil
		},
	)

	items, err := call(ctx, cliout.FormatHuman, "alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "alpha" {
		t.Fatalf("items = %+v", items)
	}
	if ctx.stderr.Len() != 0 {
		t.Fatalf("success path must not write to stderr, got %q", ctx.stderr.String())
	}
}
