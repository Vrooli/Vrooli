package cliapptest

import (
	"fmt"
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// IsolateAgentEnvironment detaches one test from any coding-agent session that
// runs it, so the verdict is the same in an operator shell, CI and an agent
// session. Inside a session cli-core ignores the generic API_BASE_URL (a CLI
// pointed at a test server would reach the live scenario instead), forwards
// the session's identity, and asks the live Agent Manager credential broker
// for a grant on every local request.
//
// It clears cliutil.AmbientAgentEnvironment for the test and points TMPDIR at
// a fresh directory, which moves cliutil.AgentCredentialSocket away from the
// live broker. Every value is restored on cleanup (t.Setenv), so like
// t.Setenv it cannot be used from a parallel test. Tests that exercise agent
// behaviour set the variables they need after calling it.
func IsolateAgentEnvironment(tb testing.TB) {
	tb.Helper()
	for _, key := range cliutil.AmbientAgentEnvironment() {
		tb.Setenv(key, "")
	}
	dir, err := isolatedTempDir()
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = os.RemoveAll(dir) })
	tb.Setenv("TMPDIR", dir)
}

// RunWithoutAgentSession runs a package's tests detached from any enclosing
// agent session, as IsolateAgentEnvironment does for one test. Use it from
// TestMain when the package's tests talk to local servers through cli-core:
//
//	func TestMain(m *testing.M) { os.Exit(cliapptest.RunWithoutAgentSession(m)) }
func RunWithoutAgentSession(m *testing.M) int {
	for _, key := range cliutil.AmbientAgentEnvironment() {
		_ = os.Unsetenv(key)
	}
	dir, err := isolatedTempDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)
	_ = os.Setenv("TMPDIR", dir)
	return m.Run()
}

// isolatedTempDir creates a short sibling of the current temp dir rather than
// nesting under tb.TempDir(): later t.TempDir() calls are created under
// TMPDIR, and Unix socket paths a test creates there must stay under the
// kernel's 108-byte limit.
func isolatedTempDir() (string, error) {
	dir, err := os.MkdirTemp("", "cliapptest-")
	if err != nil {
		return "", fmt.Errorf("create isolated temp dir: %w", err)
	}
	return dir, nil
}
