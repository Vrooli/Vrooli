package scenariohandlers

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive
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

func newFailureDeps(home string) HandlerDeps[*failureCtx] {
	return HandlerDeps[*failureCtx]{
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
		func(ctx *failureCtx, req string) (cliout.Format, []LifecycleItemOutput, error) {
			return cliout.FormatHuman, nil, errors.New("boom")
		},
	)

	_, _, err := call(ctx, "alpha")
	if err == nil {
		t.Fatal("expected error propagation")
	}
	if silent, ok := err.(interface{ Silent() bool }); !ok || !silent.Silent() {
		t.Fatalf("wrapped error must be silent: %+v", err)
	}
	if !strings.Contains(stderr.String(), "✗ Failed to start 'alpha'") {
		t.Fatalf("failure block missing from stderr: %q", stderr.String())
	}
}

func TestWithLifecycleFailureBlockPassesThroughSuccess(t *testing.T) {
	ctx := &failureCtx{stderr: &bytes.Buffer{}}
	wantItems := []LifecycleItemOutput{{Name: "alpha", Status: "started"}}

	call := withLifecycleFailureBlock(
		newFailureDeps(t.TempDir()),
		"start",
		func(req string) []string { return []string{req} },
		func(ctx *failureCtx, req string) (cliout.Format, []LifecycleItemOutput, error) {
			return cliout.FormatHuman, wantItems, nil
		},
	)

	format, items, err := call(ctx, "alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != cliout.FormatHuman {
		t.Fatalf("format = %q", format)
	}
	if len(items) != 1 || items[0].Name != "alpha" {
		t.Fatalf("items = %+v", items)
	}
	if ctx.stderr.Len() != 0 {
		t.Fatalf("success path must not write to stderr, got %q", ctx.stderr.String())
	}
}
