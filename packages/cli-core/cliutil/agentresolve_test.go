package cliutil

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// writeExecutable creates a runnable stub at dir/name and returns its path.
func writeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestResolveAgentBinaryExcludingSkipsSelf(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "executable-bit stubs are not meaningful on windows")
	}
	shimDir := t.TempDir()
	realDir := t.TempDir()
	shim := writeExecutable(t, shimDir, "codex")
	real := writeExecutable(t, realDir, "codex")

	// The shim sits ahead of the real agent, exactly as installed.
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+realDir)

	resolved, err := ResolveAgentBinaryExcluding("codex", shim)
	if err != nil {
		t.Fatalf("ResolveAgentBinaryExcluding() error = %v", err)
	}
	if resolved != real {
		t.Fatalf("resolved %q, want the real agent at %q — a shim that resolves to itself fork-bombs the host", resolved, real)
	}
}

func TestResolveAgentBinaryExcludingSkipsSelfReachedByAnotherName(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "symlink stubs are not meaningful on windows")
	}
	shimDir := t.TempDir()
	linkDir := t.TempDir()
	realDir := t.TempDir()
	shim := writeExecutable(t, shimDir, "vrooli-agent-launcher")
	// A second PATH entry reaches the very same file through a link, which is
	// how the installer actually publishes aliases.
	if err := os.Symlink(shim, filepath.Join(linkDir, "codex")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	real := writeExecutable(t, realDir, "codex")

	t.Setenv("PATH", linkDir+string(os.PathListSeparator)+realDir)

	resolved, err := ResolveAgentBinaryExcluding("codex", shim)
	if err != nil {
		t.Fatalf("ResolveAgentBinaryExcluding() error = %v", err)
	}
	if resolved != real {
		t.Fatalf("resolved %q, want %q — self must be recognised through a link, not just by directory", resolved, real)
	}
}

func TestResolveAgentBinaryExcludingExplainsShimOnlyPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "executable-bit stubs are not meaningful on windows")
	}
	shimDir := t.TempDir()
	shim := writeExecutable(t, shimDir, "codex")
	t.Setenv("PATH", shimDir)

	_, err := ResolveAgentBinaryExcluding("codex", shim)
	if err == nil {
		t.Fatal("expected an error when only the shim is on PATH")
	}
	// The operator needs to know the agent is missing, not that a lookup failed.
	if got := err.Error(); !strings.Contains(got, "not installed") {
		t.Fatalf("error = %q, want it to say the real agent is not installed", got)
	}
}

func TestResolveAgentBinaryExcludingIgnoresDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "executable-bit stubs are not meaningful on windows")
	}
	trapDir := t.TempDir()
	realDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(trapDir, "codex"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	real := writeExecutable(t, realDir, "codex")
	t.Setenv("PATH", trapDir+string(os.PathListSeparator)+realDir)

	resolved, err := ResolveAgentBinaryExcluding("codex", "")
	if err != nil {
		t.Fatalf("ResolveAgentBinaryExcluding() error = %v", err)
	}
	if resolved != real {
		t.Fatalf("resolved %q, want %q — a directory must not shadow the agent", resolved, real)
	}
}

func TestShimAliasFromArgv0(t *testing.T) {
	cases := []struct {
		argv0 string
		want  string
		ok    bool
	}{
		{"codex", "codex", true},
		{"/usr/local/bin/codex", "codex", true},
		{"claude", "claude-code", true},
		{"agy", "antigravity", true},
		{"opencode", "opencode", true},
		{"grok", "grok", true},
		{"vrooli-agent-launcher", "", false},
		{"/home/x/.vrooli/bin/vrooli-agent-launcher", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := ShimAliasFromArgv0(tc.argv0)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ShimAliasFromArgv0(%q) = (%q, %v), want (%q, %v)", tc.argv0, got, ok, tc.want, tc.ok)
		}
	}
}

func TestEveryAliasResolvesToASupportedRunner(t *testing.T) {
	// The alias table and the runner table are separate maps; this keeps them
	// from drifting apart when a new agent is added to only one of them.
	for _, alias := range CodingAgentAliases() {
		runner, ok := CodingAgentForAlias(alias)
		if !ok {
			t.Fatalf("alias %q is listed but does not map to a runner", alias)
		}
		binary, err := CodingAgentBinary(runner)
		if err != nil {
			t.Errorf("alias %q maps to runner %q, which LaunchCodingAgent rejects: %v", alias, runner, err)
			continue
		}
		if binary != alias {
			t.Errorf("alias %q maps to runner %q whose binary is %q; a shim must be installed under the binary's own name", alias, runner, binary)
		}
	}
}

func TestLaunchCodingAgentAdoptsInheritedIdentity(t *testing.T) {
	// An inherited token means a run already exists for this work. Opening a
	// second one would double-count the same agent, so the launcher must adopt
	// and never call agent-manager at all.
	t.Setenv(AgentManagerIdentityTokenEnv, "inherited-token")
	t.Setenv(AgentManagerLauncherBaseEnv, "http://127.0.0.1:9")

	var seen []string
	err := LaunchCodingAgent(t.Context(), AgentLaunchRequest{
		Agent:    "codex",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		RunChild: func(_ context.Context, _ string, _, environment []string, _ io.Reader, _, _ io.Writer) error {
			seen = environment
			return nil
		},
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgent() error = %v", err)
	}
	if got := environmentValue(seen, AgentManagerIdentityTokenEnv); got != "inherited-token" {
		t.Fatalf("identity token = %q, want the inherited token to be carried through unchanged", got)
	}
}

func TestStdioIsInheritedOnlyForThisProcessStreams(t *testing.T) {
	cases := []struct {
		name    string
		request AgentLaunchRequest
		want    bool
	}{
		{"unset streams", AgentLaunchRequest{}, true},
		{"process streams", AgentLaunchRequest{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}, true},
		{"buffered stdout", AgentLaunchRequest{Stdin: os.Stdin, Stdout: &bytes.Buffer{}, Stderr: os.Stderr}, false},
		{"buffered stdin", AgentLaunchRequest{Stdin: &bytes.Buffer{}}, false},
		{"discarded stderr", AgentLaunchRequest{Stderr: io.Discard}, false},
	}
	for _, tc := range cases {
		// exec inherits descriptors and cannot copy bytes through an arbitrary
		// reader or writer, so anything but this process's own streams must keep
		// the spawn-and-wait path or the caller silently loses its output.
		if got := stdioIsInherited(tc.request); got != tc.want {
			t.Errorf("%s: stdioIsInherited() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
