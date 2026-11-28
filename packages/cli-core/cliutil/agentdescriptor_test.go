package cliutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCodingAgentNamesComeFromDescriptors(t *testing.T) {
	for _, tc := range []struct{ name, harnessKind, binary string }{
		{"claude", "claude-code", "claude"},
		{"claude-code", "claude-code", "claude"},
		{"codex", "codex", "codex"},
		{"grok", "grok", "grok"},
		{"opencode", "opencode", "opencode"},
		{"agy", "antigravity", "agy"},
		{"antigravity", "antigravity", "agy"},
	} {
		harnessKind, binary, err := codingAgentSpec(tc.name)
		if err != nil || harnessKind != tc.harnessKind || binary != tc.binary {
			t.Errorf("codingAgentSpec(%q) = %q, %q, %v", tc.name, harnessKind, binary, err)
		}
	}
	if _, _, err := codingAgentSpec("sh -c whoami"); err == nil || !strings.Contains(err.Error(), "antigravity") {
		t.Fatalf("an unknown runner must be refused and name the supported runners: %v", err)
	}
	if got := CodingAgentAliases(); !slices.Equal(got, []string{"agy", "claude", "codex", "grok", "opencode"}) {
		t.Fatalf("shim aliases must be exactly the shipped binaries: %v", got)
	}
	// A resource name is not something a shim is installed under.
	if _, ok := CodingAgentForAlias("claude-code"); ok {
		t.Fatal("claude-code is not a shim alias")
	}
}

func TestHarnessSessionEnvironmentsPutNativeNamesFirst(t *testing.T) {
	for harnessKind, want := range map[string][]string{
		"claude-code": {EnvClaudeCodeSessionID, "VROOLI_HARNESS_SESSION_ID"},
		"codex":       {EnvCodexThreadID, "VROOLI_HARNESS_SESSION_ID"},
		"grok":        {"VROOLI_HARNESS_SESSION_ID"},
		"unknown":     {"VROOLI_HARNESS_SESSION_ID"},
	} {
		if got := harnessSessionEnvironments(harnessKind); !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", harnessKind, got, want)
		}
	}
}

// A new harness needs only its own resource directory: it launches, gets a
// shim alias, reports its session variable and isolates its home with no edit
// to any table here.
func TestSixthHarnessNeedsOnlyItsResourceDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "resources", "sixth")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"sixth","harness":{"id":"six","aliases":["sixth-agent"],"binary":"sixth-cli","display_name":"Sixth","icon":"six",
		"session_env":["SIXTH_SESSION"],
		"home_isolation":{"kind":"env","env":"SIXTH_HOME","shared_dir":".sixth","state_dir":"sixth","layout":"mirror"},
		"features":{"provider_selection":false,"resume":false,"goal_mode":false,"mcp":false,"hooks":false}}}`
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
	t.Setenv("HOME", t.TempDir())
	t.Setenv(AgentManagerIdentityTokenEnv, "")

	if runner, ok := CodingAgentForAlias("sixth-cli"); !ok || runner != "sixth" {
		t.Fatalf("CodingAgentForAlias = %q, %t", runner, ok)
	}
	if !slices.Contains(CodingAgentAliases(), "sixth-cli") {
		t.Fatalf("no shim alias: %v", CodingAgentAliases())
	}
	if got := harnessSessionEnvironments("sixth"); !slices.Equal(got, []string{"SIXTH_SESSION", "VROOLI_HARNESS_SESSION_ID"}) {
		t.Fatalf("session environments: %v", got)
	}

	stateRoot := t.TempDir()
	var resolved string
	var childEnvironment []string
	err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
		Agent:       "sixth-agent",
		APIBase:     "http://127.0.0.1:1",
		Environment: []string{"WC_WEB_CONSOLE_SESSION_ID=session-1", "WC_SESSION_STATE_ROOT=" + stateRoot},
		LookPath: func(binary string) (string, error) {
			resolved = binary
			return "/safe/" + binary, nil
		},
		RunChild: func(_ context.Context, _ string, _ []string, environment []string, _ io.Reader, _, _ io.Writer) error {
			childEnvironment = environment
			return nil
		},
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgent() error = %v", err)
	}
	if resolved != "sixth-cli" {
		t.Fatalf("resolved binary %q", resolved)
	}
	if got, want := environmentValue(childEnvironment, "SIXTH_HOME"), filepath.Join(stateRoot, "sixth", "session-1"); got != want {
		t.Fatalf("SIXTH_HOME = %q, want %q", got, want)
	}
}

func TestPrepareWebConsoleAgentHomeLeavesUnisolatedHarnessesAlone(t *testing.T) {
	base := []string{"WC_WEB_CONSOLE_SESSION_ID=s", "WC_SESSION_STATE_ROOT=" + t.TempDir()}
	for _, agent := range []string{"claude", "claude-code", "opencode", "agy", "unknown"} {
		if got := PrepareWebConsoleAgentHome(agent, base); !slices.Equal(got, base) {
			t.Errorf("%s changed the environment: %v", agent, got)
		}
	}
}

func TestPrepareWebConsoleAgentHomeIsolatesGrokAndKeepsAnExplicitHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stateRoot := t.TempDir()
	base := []string{"WC_WEB_CONSOLE_SESSION_ID=s", "WC_SESSION_STATE_ROOT=" + stateRoot}
	if got, want := environmentValue(PrepareWebConsoleAgentHome("grok", base), "GROK_HOME"), filepath.Join(stateRoot, "grok", "s"); got != want {
		t.Fatalf("GROK_HOME = %q, want %q", got, want)
	}
	explicit := append(slices.Clone(base), "GROK_HOME=/chosen")
	if got := PrepareWebConsoleAgentHome("grok", explicit); !slices.Equal(got, explicit) {
		t.Fatalf("an explicit home must be kept: %v", got)
	}
}
