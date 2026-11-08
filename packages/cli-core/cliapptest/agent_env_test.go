package cliapptest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// pollutedAgentEnvironment simulates the environment of an agent session
// that runs `go test`: every detector cliutil consults reports an agent.
func pollutedAgentEnvironment(t *testing.T) {
	t.Helper()
	values := map[string]string{
		"CLAUDECODE":            "1",
		"GROK_AGENT":            "1",
		"ANTIGRAVITY_AGENT":     "1",
		"VROOLI_CALLER":         "agent",
		"CODEX_CI":              "1",
		"OPENCODE_PID":          "1",
		cliutil.AgentSessionEnv: "vrooli-agent-test-session",
	}
	for _, key := range cliutil.AmbientAgentEnvironment() {
		value, ok := values[key]
		if !ok {
			value = "ambient-" + strings.ToLower(key)
		}
		t.Setenv(key, value)
	}
}

func TestMain(m *testing.M) { os.Exit(RunWithoutAgentSession(m)) }

func TestIsolateAgentEnvironment_DetachesEveryDetector(t *testing.T) {
	pollutedAgentEnvironment(t)
	if !cliutil.IsAgentControlledContext() || cliutil.DetectCallerKind() == cliutil.CallerKindUnknown {
		t.Fatal("polluted fixture does not look like an agent session; the test proves nothing")
	}
	liveTemp := os.TempDir()

	IsolateAgentEnvironment(t)
	requireDetached(t, liveTemp)
}

// TestRunWithoutAgentSession_DetachesPackage re-executes this test binary in a
// polluted environment; its TestMain must detach before any test runs.
func TestRunWithoutAgentSession_DetachesPackage(t *testing.T) {
	if os.Getenv("CLIAPPTEST_SESSION_PROBE") == "1" {
		requireDetached(t, os.Getenv("CLIAPPTEST_LIVE_TEMP"))
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pollutedAgentEnvironment(t)
	cmd := exec.Command(executable, "-test.run=^TestRunWithoutAgentSession_DetachesPackage$", "-test.count=1")
	cmd.Env = append(os.Environ(), "CLIAPPTEST_SESSION_PROBE=1", "CLIAPPTEST_LIVE_TEMP="+os.TempDir())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("probe in a polluted environment failed: %v\n%s", err, output)
	}
}

func requireDetached(t *testing.T, liveTemp string) {
	t.Helper()
	if cliutil.IsAgentControlledContext() {
		t.Fatal("IsAgentControlledContext() = true after isolation")
	}
	if kind := cliutil.DetectCallerKind(); kind != cliutil.CallerKindUnknown {
		t.Fatalf("DetectCallerKind() = %v after isolation, want unknown", kind)
	}
	if got := cliutil.ResolveCallerActor(""); !strings.Contains(got, "no agent signal") {
		t.Fatalf("ResolveCallerActor() = %q after isolation, want no agent signal", got)
	}
	headers := cliutil.InvocationHeaders("scenario-test", "probe")()
	for _, header := range []string{cliutil.HeaderAgentIdentityToken, cliutil.HeaderHarnessSessionID, cliutil.HeaderHarnessKind} {
		if value, ok := headers[header]; ok {
			t.Fatalf("invocation header %s = %q after isolation", header, value)
		}
	}
	if socket := cliutil.AgentCredentialSocket(); strings.HasPrefix(socket, liveTemp+string(os.PathSeparator)+"vrooli-agent-manager-") {
		t.Fatalf("AgentCredentialSocket() = %q still points at the live broker", socket)
	}
	if _, err := os.Stat(cliutil.AgentCredentialSocket()); !os.IsNotExist(err) {
		t.Fatalf("isolated credential socket should not exist, stat err = %v", err)
	}
}

func TestIsolateAgentEnvironment_RestoresOnCleanup(t *testing.T) {
	pollutedAgentEnvironment(t)
	before := os.Getenv(cliutil.EnvIdentityToken)
	beforeTemp := os.Getenv("TMPDIR")
	var isolatedTemp string

	t.Run("inner", func(tt *testing.T) {
		IsolateAgentEnvironment(tt)
		isolatedTemp = os.Getenv("TMPDIR")
	})

	if got := os.Getenv(cliutil.EnvIdentityToken); got != before {
		t.Fatalf("%s after cleanup = %q, want %q", cliutil.EnvIdentityToken, got, before)
	}
	if got := os.Getenv("TMPDIR"); got != beforeTemp {
		t.Fatalf("TMPDIR after cleanup = %q, want %q", got, beforeTemp)
	}
	if _, err := os.Stat(isolatedTemp); !os.IsNotExist(err) {
		t.Fatalf("isolated temp dir %q survived cleanup (stat err = %v)", isolatedTemp, err)
	}
}

// TestNewAPIServer_ReachesTestServerInsideAgentSession is the regression for
// CLI tests that, run from an agent session, sent their requests to the live
// scenario: cli-core ignores API_BASE_URL in agent-controlled contexts.
func TestNewAPIServer_ReachesTestServerInsideAgentSession(t *testing.T) {
	pollutedAgentEnvironment(t)
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("request reached the live scenario stand-in")
	}))
	defer live.Close()
	server := NewAPIServer(t, okHandler())

	base := cliutil.DetermineAPIBase(cliutil.APIBaseOptions{
		EnvVars:      []string{"API_BASE_URL"},
		PortDetector: func() string { return strings.TrimPrefix(live.URL, "http://127.0.0.1:") },
	})
	if base != server.URL {
		t.Fatalf("resolved API base = %q, want test server %q", base, server.URL)
	}
}
