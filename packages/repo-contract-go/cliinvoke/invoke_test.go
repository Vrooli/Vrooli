package cliinvoke

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fakeCLI(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-specific")
	}
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return script
}

// [REQ:CLI-INVOKE-001] Run returns on the command's own timescale even when a
// descendant keeps the inherited output pipe open (the 2026-08-01 outage).
func TestRunReturnsWhenDescendantHoldsThePipe(t *testing.T) {
	script := fakeCLI(t, "echo started\nsleep 300 &\nexit 0\n")
	done := make(chan Result, 1)
	go func() {
		done <- Run(context.Background(), Invocation{Binary: script, Args: []string{"scenario", "start", "x"}})
	}()
	select {
	case res := <-done:
		if res.Class != OK {
			t.Fatalf("class = %s, err = %v", res.Class, res.Err)
		}
		if !strings.Contains(string(res.Stdout), "started") {
			t.Fatalf("stdout = %q, want the command's own output", res.Stdout)
		}
	case <-time.After(DefaultWaitDelay + 20*time.Second):
		t.Fatal("Run blocked on a pipe held by a descendant")
	}
}

// [REQ:CLI-INVOKE-002] A usage error is its own class: retrying it identically
// can never succeed.
func TestRunClassifiesUsageError(t *testing.T) {
	script := fakeCLI(t, "echo 'Unknown command: --bogus-flag' >&2\necho \"Run 'vrooli --help' for usage information\" >&2\nexit 1\n")
	res := Run(context.Background(), Invocation{Binary: script, Args: []string{"--bogus-flag", "version"}})
	if res.Class != Usage {
		t.Fatalf("class = %s, want usage (stderr=%q)", res.Class, res.Stderr)
	}
	if res.Class.Retryable() {
		t.Fatal("usage must not be retryable")
	}
	if res.ExitCode != 1 {
		t.Fatalf("exit code = %d", res.ExitCode)
	}
}

func TestRunClassifiesLifecycleFailure(t *testing.T) {
	script := fakeCLI(t, "echo boom >&2\nexit 3\n")
	res := Run(context.Background(), Invocation{Binary: script, Args: []string{"scenario", "start", "x"}})
	if res.Class != Lifecycle || res.ExitCode != 3 {
		t.Fatalf("class = %s exit = %d", res.Class, res.ExitCode)
	}
	if !strings.Contains(string(res.Combined()), "boom") {
		t.Fatalf("combined = %q, want diagnostics", res.Combined())
	}
	if res.Error() == nil {
		t.Fatal("non-OK result must render an error")
	}
}

func TestRunClassifiesRefusal(t *testing.T) {
	script := fakeCLI(t, "echo 'credential store is locked; run vrooli credentials store unlock' >&2\nexit 1\n")
	res := Run(context.Background(), Invocation{Binary: script, Args: []string{"credentials", "get"}})
	if res.Class != Refusal {
		t.Fatalf("class = %s, want refusal", res.Class)
	}
}

func TestRunClassifiesTimeout(t *testing.T) {
	script := fakeCLI(t, "sleep 30\n")
	res := Run(context.Background(), Invocation{Binary: script, Timeout: 300 * time.Millisecond, WaitDelay: 200 * time.Millisecond})
	if res.Class != Timeout {
		t.Fatalf("class = %s, want timeout (err=%v)", res.Class, res.Err)
	}
	if !errors.Is(res.Err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", res.Err)
	}
}

func TestRunClassifiesBinaryMissing(t *testing.T) {
	res := Run(context.Background(), Invocation{Binary: filepath.Join(t.TempDir(), "absent"), Args: []string{"version"}})
	if res.Class != BinaryMissing {
		t.Fatalf("class = %s, want binary-missing (err=%v)", res.Class, res.Err)
	}
	if Run(context.Background(), Invocation{}).Class != BinaryMissing {
		t.Fatal("an empty binary path must classify as binary-missing")
	}
}

func TestRunTeesStderrToCallerWriter(t *testing.T) {
	script := fakeCLI(t, "echo out\necho err >&2\nexit 0\n")
	var stdout, stderr strings.Builder
	res := Run(context.Background(), Invocation{Binary: script, Stdout: &stdout, Stderr: &stderr})
	if stdout.String() != "out\n" || stderr.String() != "err\n" {
		t.Fatalf("writers got %q / %q", stdout.String(), stderr.String())
	}
	if len(res.Stdout) != 0 || string(res.Stderr) != "err\n" {
		t.Fatalf("result captured %q / %q; stdout must not be duplicated, stderr tail must be kept", res.Stdout, res.Stderr)
	}
}

func TestClassifyTable(t *testing.T) {
	cases := []struct {
		stderr string
		want   Class
	}{
		{"Unknown scenario command: foo", Usage},
		{"Usage error: missing value for --yes", Usage},
		{"flag provided but not defined: -wait", Usage},
		{"credential store uninitialized", Refusal},
		{"keyring is unresponsive", Refusal},
		{"Runtime error: scenario failed to start", Lifecycle},
	}
	for _, tc := range cases {
		if got := Classify(context.Background(), errors.New("exit status 1"), []byte(tc.stderr)); got != tc.want {
			t.Errorf("Classify(%q) = %s, want %s", tc.stderr, got, tc.want)
		}
	}
}

// [REQ:CLI-INVOKE-003] Resolution order is fixed and every miss is named.
func TestResolveOrderAndMissingCandidates(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit-vrooli")
	fromEnv := filepath.Join(dir, "env-vrooli")
	home := filepath.Join(dir, "home")
	homeBin := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(homeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	fromHome := filepath.Join(homeBin, binaryName())
	fromPath := filepath.Join(dir, "path-vrooli")
	for _, p := range []string{explicit, fromEnv, fromHome, fromPath} {
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := map[string]string{BinaryEnvVar: fromEnv}
	opts := ResolveOptions{
		Explicit:    explicit,
		RuntimeHome: home,
		Getenv:      func(k string) string { return env[k] },
		LookPath:    func(string) (string, error) { return fromPath, nil },
	}
	if got, _ := Resolve(opts); got != explicit {
		t.Fatalf("explicit should win, got %q", got)
	}
	opts.Explicit = filepath.Join(dir, "missing-explicit")
	if got, _ := Resolve(opts); got != fromEnv {
		t.Fatalf("VROOLI_BIN should be second, got %q", got)
	}
	env[BinaryEnvVar] = ""
	if got, _ := Resolve(opts); got != fromHome {
		t.Fatalf("runtime home bin should be third, got %q", got)
	}
	if err := os.Remove(fromHome); err != nil {
		t.Fatal(err)
	}
	if got, _ := Resolve(opts); got != fromPath {
		t.Fatalf("PATH should be last, got %q", got)
	}
	opts.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	_, err := Resolve(opts)
	var missing *BinaryMissingError
	if !errors.As(err, &missing) || !errors.Is(err, ErrBinaryMissing) {
		t.Fatalf("err = %v, want BinaryMissingError", err)
	}
	for _, want := range []string{opts.Explicit, fromHome, "PATH"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name candidate %q", err, want)
		}
	}
}
