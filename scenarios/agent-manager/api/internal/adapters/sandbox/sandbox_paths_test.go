// Responsibility: retain sandbox launcher test declarations within their original package.
package sandbox

import (
	"agent-manager/internal/adapters/runner"
	"context"
	"errors"
	"github.com/google/uuid"
	"io"
	"strings"
	"testing"
	"time"
)

// TestTranslateHostPathToNamespace exercises the host→namespace path
// rewriter directly for BOTH layouts: the path-illusion layout (bwrap,
// workspacePath="/workspace") rewrites host-merged prefixes onto the
// reported path, and the identity layout (copy driver / macOS, workspacePath
// == hostMerged) leaves values effectively unchanged.
func TestTranslateHostPathToNamespace(t *testing.T) {
	const host = "/home/alice/.local/share/workspace-sandbox/abc/merged"

	t.Run("path-illusion layout", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			want string
		}{
			{"empty input passes through", "", ""},
			{"exact match → workspacePath", host, nsPath},
			{"subpath → workspacePath/<rest>", host + "/sub/file.txt", nsPath + "/sub/file.txt"},
			{"unrelated absolute path passes through", "/etc/hosts", "/etc/hosts"},
			{"already namespace path passes through", nsPath + "/x", nsPath + "/x"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				got := translateHostPathToNamespace(c.in, host, nsPath)
				if got != c.want {
					t.Errorf("translateHostPathToNamespace(%q, host, %q) = %q; want %q", c.in, nsPath, got, c.want)
				}
			})
		}
	})

	t.Run("identity layout (workspacePath == hostMerged)", func(t *testing.T) {
		// pathIllusion=false ⇒ workspacePath == hostMerged, so every value
		// maps back to itself (identity).
		cases := []string{host, host + "/sub/file.txt", "/etc/hosts", ""}
		for _, in := range cases {
			if got := translateHostPathToNamespace(in, host, host); got != in {
				t.Errorf("identity layout: translate(%q) = %q; want %q", in, got, in)
			}
		}
	})

	t.Run("empty hostMerged is identity", func(t *testing.T) {
		got := translateHostPathToNamespace("/some/path", "", nsPath)
		if got != "/some/path" {
			t.Errorf("got %q; want identity passthrough", got)
		}
	})
}

// TestResolveWorkingDir pins the workdir contract:
//   - empty → nsPath
//   - host merged dir / subpath → translated
//   - already in-namespace → unchanged
//   - other absolute host path → *LaunchBlocked{workdir_outside_sandbox}
func TestResolveWorkingDir(t *testing.T) {
	const host = "/home/x/.local/share/workspace-sandbox/sb1/merged"

	t.Run("empty defaults to namespace path", func(t *testing.T) {
		got, err := resolveWorkingDir("", host, nsPath)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != nsPath {
			t.Errorf("got %q; want %q", got, nsPath)
		}
	})

	t.Run("host merged path translates", func(t *testing.T) {
		got, err := resolveWorkingDir(host, host, nsPath)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != nsPath {
			t.Errorf("got %q; want %q", got, nsPath)
		}
	})

	t.Run("subpath of merged translates", func(t *testing.T) {
		got, err := resolveWorkingDir(host+"/foo", host, nsPath)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		want := nsPath + "/foo"
		if got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	})

	t.Run("already namespace path passes through", func(t *testing.T) {
		got, err := resolveWorkingDir(nsPath, host, nsPath)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != nsPath {
			t.Errorf("got %q; want %q", got, nsPath)
		}
	})

	t.Run("unrelated host path is blocked", func(t *testing.T) {
		_, err := resolveWorkingDir("/etc/hosts", host, nsPath)
		var blocked *LaunchBlocked
		if !errors.As(err, &blocked) {
			t.Fatalf("err = %T (%v); want *LaunchBlocked", err, err)
		}
		if blocked.Code != "workdir_outside_sandbox" {
			t.Errorf("blocked.Code = %q; want workdir_outside_sandbox", blocked.Code)
		}
	})

	// Identity layout: workspacePath == hostMerged (copy driver / macOS).
	// The working dir stays the host merged dir; a subpath stays under it;
	// an unrelated host path is still a contract violation.
	t.Run("identity layout keeps host merged dir", func(t *testing.T) {
		if got, err := resolveWorkingDir("", host, host); err != nil || got != host {
			t.Fatalf("empty → %q (err=%v); want %q", got, err, host)
		}
		if got, err := resolveWorkingDir(host, host, host); err != nil || got != host {
			t.Fatalf("host → %q (err=%v); want %q", got, err, host)
		}
		if got, err := resolveWorkingDir(host+"/foo", host, host); err != nil || got != host+"/foo" {
			t.Fatalf("subpath → %q (err=%v); want %q", got, err, host+"/foo")
		}
		if _, err := resolveWorkingDir("/etc/hosts", host, host); err == nil {
			t.Error("unrelated host path should still be blocked under identity layout")
		}
	})
}

// TestSandboxLauncher_LaunchTranslatesHostMergedPath verifies that when
// the run executor passes the *host* merged path as WorkingDir, the
// launcher rewrites it to /workspace before POSTing — preventing the
// "bwrap: Can't chdir to /home/.../merged: No such file or directory"
// regression.
func TestSandboxLauncher_LaunchTranslatesHostMergedPath(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-test/merged"

	mock := newSandboxTestServer(707)
	mock.hostMergedDir = host
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		WorkingDir: host,
		Env:        []string{"VROOLI_SANDBOX_MERGED=" + host, "VROOLI_SANDBOX_MERGED_HOST=" + host, "PATH=/usr/bin", "HOME=.home"},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()
	if got, _ := body["workingDir"].(string); got != nsPath {
		t.Errorf("workingDir = %q; want %q (host path must be translated)", got, nsPath)
	}
	envAny, _ := body["env"].(map[string]any)
	if got, _ := envAny["VROOLI_SANDBOX_MERGED"].(string); got != nsPath {
		t.Errorf("env VROOLI_SANDBOX_MERGED = %q; want %q", got, nsPath)
	}
	if got, _ := envAny["VROOLI_SANDBOX_MERGED_HOST"].(string); got != host {
		t.Errorf("env VROOLI_SANDBOX_MERGED_HOST = %q; want host path %q", got, host)
	}
	if got, _ := envAny["VROOLI_HOST_LIFECYCLE_BASE"].(string); got != server.URL {
		t.Errorf("env VROOLI_HOST_LIFECYCLE_BASE = %q; want %q", got, server.URL)
	}
	if _, ok := envAny["PATH"]; ok {
		t.Errorf("env PATH should be omitted; workspace-sandbox owns vrooli-aware PATH construction")
	}
	// A relative inherited HOME (e.g. HOME=.home from a sandboxed parent)
	// must not cross the boundary: with --chdir <workspace> it would
	// materialize `<workspace>/.home` from $HOME-relative writes. The
	// vrooli-aware profile owns HOME, so it is dropped here.
	if _, ok := envAny["HOME"]; ok {
		t.Errorf("env HOME should be omitted; workspace-sandbox owns vrooli-aware HOME construction")
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	_ = proc.Wait()
}

// TestSandboxLauncher_LaunchIdentityLayoutKeepsHostPath is the identity-
// layout counterpart: when the server reports pathIllusion=false
// (workspacePath == host merged dir — copy driver / macOS), the launcher
// must NOT rewrite the workingDir to "/workspace" (which would not exist);
// it keeps the host merged path so the agent runs against the real dir.
func TestSandboxLauncher_LaunchIdentityLayoutKeepsHostPath(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-identity/merged"

	mock := newSandboxTestServer(717)
	mock.hostMergedDir = host
	mock.identityLayout = true
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		WorkingDir: host,
		Env:        []string{"VROOLI_SANDBOX_MERGED=" + host},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()
	if got, _ := body["workingDir"].(string); got != host {
		t.Errorf("workingDir = %q; want %q (identity layout keeps host path)", got, host)
	}
	envAny, _ := body["env"].(map[string]any)
	if got, _ := envAny["VROOLI_SANDBOX_MERGED"].(string); got != host {
		t.Errorf("env VROOLI_SANDBOX_MERGED = %q; want %q (identity layout)", got, host)
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	_ = proc.Wait()
}

// TestTranslateCommandToNamespace pins the binary-path rewrite contract.
// The mapping must stay in lockstep with the vrooli-aware bind layout in
// scenarios/workspace-sandbox/api/internal/driver/bwrap.go.
func TestTranslateCommandToNamespace(t *testing.T) {
	const home = "/home/alice"
	cases := []struct {
		name        string
		command     string
		hostHome    string
		state       HomeOverlayState
		want        string
		wantRewrite bool
		wantErr     bool
	}{
		{
			name:        "empty passes through",
			command:     "",
			hostHome:    home,
			want:        "",
			wantRewrite: false,
		},
		{
			name:        "bare basename → unchanged (PATH lookup in sandbox)",
			command:     "claude",
			hostHome:    home,
			want:        "claude",
			wantRewrite: false,
		},
		{
			name:        "relative path → unchanged",
			command:     "./bin/claude",
			hostHome:    home,
			want:        "./bin/claude",
			wantRewrite: false,
		},
		{
			name:        "$HOME/.local/bin/claude → unchanged (profile binds at host path)",
			command:     home + "/.local/bin/claude",
			hostHome:    home,
			want:        home + "/.local/bin/claude",
			wantRewrite: false,
		},
		{
			name:        "$HOME with trailing slash still matches",
			command:     home + "/.local/bin/codex",
			hostHome:    home + "/",
			want:        home + "/.local/bin/codex",
			wantRewrite: false,
		},
		{
			name:        "$HOME/.local/share/X/Y → unchanged (companion bind for symlink targets)",
			command:     home + "/.local/share/claude/versions/2.1.121",
			hostHome:    home,
			want:        home + "/.local/share/claude/versions/2.1.121",
			wantRewrite: false,
		},
		{
			name:        "/usr/bin/X → unchanged (sandbox mounts /usr at /usr)",
			command:     "/usr/bin/git",
			hostHome:    home,
			want:        "/usr/bin/git",
			wantRewrite: false,
		},
		{
			name:        "/bin/X → unchanged (sandbox mounts /bin at /bin)",
			command:     "/bin/sh",
			hostHome:    home,
			want:        "/bin/sh",
			wantRewrite: false,
		},
		{
			name:        "/usr/local/bin/X → unchanged (already namespace path)",
			command:     "/usr/local/bin/foo",
			hostHome:    home,
			want:        "/usr/local/bin/foo",
			wantRewrite: false,
		},
		{
			name:        "unknown host-absolute → basename fallback",
			command:     "/opt/homebrew/bin/claude",
			hostHome:    home,
			want:        "claude",
			wantRewrite: true,
		},
		{
			name:        "empty hostHome disables ~/.local/bin rule but basename fallback still applies",
			command:     "/some/random/path/tool",
			hostHome:    "",
			want:        "tool",
			wantRewrite: true,
		},
		{
			name:        "different user's home directory does not match",
			command:     "/home/other/.local/bin/claude",
			hostHome:    home,
			want:        "claude",
			wantRewrite: true,
		},
	}
	// Default state for cases that don't set it explicitly: Present.
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := c.state
			if state == "" {
				state = HomeOverlayPresent
			}
			// These cases model the bwrap path-illusion layout, where
			// unmapped host-absolute binaries must fall back to their
			// basename + sandbox PATH.
			layout := NamespaceLayout{HostHome: c.hostHome, HomeOverlayState: state, PathIllusion: true}
			got, err := translateCommandToNamespace(c.command, layout)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got command=%q", got)
				}
				var typed *ErrCommandHomeOverlayUnavailable
				if !errors.As(err, &typed) {
					t.Errorf("expected ErrCommandHomeOverlayUnavailable, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got command %q; want %q", got, c.want)
			}
		})
	}
}

// TestTranslateCommandToNamespace_HostMergedWinsOverHome is the regression
// pin for the empty-workspace defect (sandbox cc371116, 2026-07-20): the
// host merged dir lives under $HOME, so the $HOME-unchanged rule used to
// swallow it — codex received `-C <hostMerged>` untranslated and chdir'd
// into the home-overlay-shadowed empty view of the merged dir. The
// hostMerged→workspacePath rewrite must win over the $HOME passthrough.
func TestTranslateCommandToNamespace_HostMergedWinsOverHome(t *testing.T) {
	const home = "/home/alice"
	hostMerged := home + "/.local/share/workspace-sandbox/abc/merged"
	layout := NamespaceLayout{
		HostHome:         home,
		HomeOverlayState: HomeOverlayPresent,
		HostMerged:       hostMerged,
		WorkspacePath:    "/workspace",
		PathIllusion:     true,
	}
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"exact merged dir", hostMerged, "/workspace"},
		{"subpath of merged dir", hostMerged + "/scenarios/react-component-library", "/workspace/scenarios/react-component-library"},
		{"sibling under $HOME still passes through", home + "/.local/bin/codex", home + "/.local/bin/codex"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := translateCommandToNamespace(c.in, layout)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q; want %q", got, c.want)
			}
		})
	}
	// Identity layout (workspacePath == hostMerged): rewrite is a no-op.
	identity := layout
	identity.PathIllusion = false
	identity.WorkspacePath = hostMerged
	got, err := translateCommandToNamespace(hostMerged+"/ui", identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := hostMerged + "/ui"; got != want {
		t.Errorf("identity layout: got %q; want %q", got, want)
	}
}

// TestTranslateCommandToNamespace_IdentityLayoutKeepsHostPaths pins that
// under an identity layout (pathIllusion=false — copy driver / macOS) a
// host-absolute binary path with no system-bind mapping is left intact
// rather than stripped to its basename: the process runs against real host
// paths, so a binary outside /usr,/bin must still resolve.
func TestTranslateCommandToNamespace_IdentityLayoutKeepsHostPaths(t *testing.T) {
	layout := NamespaceLayout{HostHome: "/home/alice", HomeOverlayState: HomeOverlayPresent, PathIllusion: false}
	for _, cmd := range []string{"/opt/homebrew/bin/claude", "/some/random/path/tool"} {
		got, err := translateCommandToNamespace(cmd, layout)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", cmd, err)
		}
		if got != cmd {
			t.Errorf("identity layout: translate(%q) = %q; want unchanged", cmd, got)
		}
	}
}

func TestNamespaceLayoutPathEntriesMatchesVrooliAwareProfile(t *testing.T) {
	layout := NamespaceLayout{
		HostHome:         "/home/alice/",
		HomeOverlayState: HomeOverlayPresent,
	}
	want := []string{
		"/home/alice/.vrooli/bin",
		"/home/alice/go/bin",
		"/home/alice/.local/bin",
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
	}
	got := layout.PathEntries()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("PathEntries() = %v, want %v", got, want)
	}
}

func TestNamespaceLayoutPathEntriesWithoutHomeOverlay(t *testing.T) {
	layout := NamespaceLayout{
		HostHome:         "/home/alice",
		HomeOverlayState: HomeOverlayAbsent,
	}
	want := []string{"/usr/local/bin", "/usr/bin", "/bin"}
	got := layout.PathEntries()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("PathEntries() = %v, want %v", got, want)
	}
}

// TestTranslateCommandToNamespace_RefusesHomeWhenStateAbsent — the
// load-bearing seam introduced in Phase F. A command pointing under
// $HOME with state != Present must surface as ErrCommandHomeOverlayUnavailable
// before the launcher POSTs to workspace-sandbox.
func TestTranslateCommandToNamespace_RefusesHomeWhenStateAbsent(t *testing.T) {
	const home = "/home/alice"
	for _, state := range []HomeOverlayState{HomeOverlayAbsent, HomeOverlayUnsupported, HomeOverlayNotRequested} {
		t.Run(string(state), func(t *testing.T) {
			layout := NamespaceLayout{HostHome: home, HomeOverlayState: state}
			_, err := translateCommandToNamespace(home+"/.local/bin/claude", layout)
			if err == nil {
				t.Fatalf("expected error for state=%s, got nil", state)
			}
			var typed *ErrCommandHomeOverlayUnavailable
			if !errors.As(err, &typed) {
				t.Fatalf("expected ErrCommandHomeOverlayUnavailable, got %T", err)
			}
			if typed.Code() != "SANDBOX_HOME_OVERLAY_UNAVAILABLE" {
				t.Errorf("Code()=%q; want SANDBOX_HOME_OVERLAY_UNAVAILABLE", typed.Code())
			}
		})
	}
}

// TestSandboxLauncher_LaunchPreservesEnvShimArgs verifies the realistic
// runner shape: claude_code/codex/opencode go through
// BuildEnvWrappedLaunchRequest, which sets Command="env" and stuffs the
// host-absolute binary path into Args[1] (after a TAG=value env-var
// assignment in Args[0]). The runtime profile binds $HOME/.local/bin at
// the *host path* inside the namespace (the profile's dst mapping is
// not honored by buildBwrapArgs), so the host-absolute binary path is
// already valid inside the sandbox — the launcher must NOT rewrite it.
// This guards against a regression where a too-aggressive rewrite mapped
// $HOME/.local/bin/X to /usr/local/bin/X, which doesn't exist inside the
// sandbox under the profile-based isolation path.
func TestSandboxLauncher_LaunchPreservesEnvShimArgs(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-envshim/merged"

	mock := newSandboxTestServer(910)
	mock.hostMergedDir = host
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	t.Setenv("HOME", "/home/testuser")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command: "env",
		Args: []string{
			"CLAUDE_CODE_AGENT_TAG=run-12345",
			"/home/testuser/.local/bin/claude",
			"--print",
			"--output-format=stream-json",
		},
		WorkingDir: host,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()

	if got, _ := body["command"].(string); got != "env" {
		t.Errorf("command = %q; want %q (basename passes through)", got, "env")
	}

	argsAny, _ := body["args"].([]any)
	want := []string{
		"CLAUDE_CODE_AGENT_TAG=run-12345",  // tag arg untouched
		"/home/testuser/.local/bin/claude", // binary path PRESERVED — profile binds at host path
		"--print",                          // flag untouched
		"--output-format=stream-json",      // flag untouched
	}
	if len(argsAny) != len(want) {
		t.Fatalf("args length = %d; want %d (args=%v)", len(argsAny), len(want), argsAny)
	}
	for i, w := range want {
		if got, _ := argsAny[i].(string); got != w {
			t.Errorf("args[%d] = %q; want %q", i, got, w)
		}
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	_ = proc.Wait()
}

// TestSandboxLauncher_LaunchBasenameFallback verifies the basename
// fallback path: a host-absolute binary path that doesn't match any
// known sandbox mount (e.g. /opt/homebrew/bin/X on macOS) gets stripped
// to its basename so the sandbox PATH lookup can find it.
func TestSandboxLauncher_LaunchBasenameFallback(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-cmd/merged"

	mock := newSandboxTestServer(909)
	mock.hostMergedDir = host
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	// $HOME doesn't matter here — the test path is /opt/* which has no
	// known sandbox mapping, so it falls through to the basename rule.
	t.Setenv("HOME", "/home/testuser")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "/opt/homebrew/bin/claude",
		Args:       []string{"--version"},
		WorkingDir: host,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()
	if got, _ := body["command"].(string); got != "claude" {
		t.Errorf("command = %q; want %q (unknown host-absolute path must basename-fallback)", got, "claude")
	}
	// Args must be left untouched — they're typically data, not paths.
	if argsAny, _ := body["args"].([]any); len(argsAny) != 1 || argsAny[0] != "--version" {
		t.Errorf("args = %v; want [--version] (must not be rewritten)", argsAny)
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	_ = proc.Wait()
}

// TestSandboxLauncher_LaunchRejectsUntranslatableHostPath verifies that a
// WorkingDir that is neither under the sandbox nor under
// nsPath is rejected as a contract violation, not silently
// passed through.
func TestSandboxLauncher_LaunchRejectsUntranslatableHostPath(t *testing.T) {
	mock := newSandboxTestServer(808)
	mock.hostMergedDir = "/var/lib/workspace-sandbox/sb-test/merged"
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		WorkingDir: "/etc",
	})
	if err == nil {
		t.Fatal("Launch returned nil; want *LaunchBlocked")
	}
	var blocked *LaunchBlocked
	if !errors.As(err, &blocked) {
		t.Fatalf("err = %T (%v); want *LaunchBlocked", err, err)
	}
	if blocked.Code != "workdir_outside_sandbox" {
		t.Errorf("blocked.Code = %q; want workdir_outside_sandbox", blocked.Code)
	}
	if mock.startProcessSeen.Load() {
		t.Error("startProcess should NOT have been called for an untranslatable workdir")
	}
}

// TestEnvSliceToMap verifies the os.Environ() → map[string]string conversion.
func TestEnvSliceToMap(t *testing.T) {
	in := []string{"A=1", "B=hello world", "C=", "noEqualsHere", "D=multi=equals=ok"}
	got := envSliceToMap(in)
	want := map[string]string{
		"A": "1",
		"B": "hello world",
		"C": "",
		"D": "multi=equals=ok",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d; want %d (got %v)", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("got[%q] = %q; want %q", k, got[k], v)
		}
	}
	if _, exists := got["noEqualsHere"]; exists {
		t.Error("env entry without '=' should be skipped")
	}
}
