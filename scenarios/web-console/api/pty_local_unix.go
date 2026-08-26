//go:build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	platform "github.com/vrooli/platform-go"
	"web-console/backends/claude"
	"web-console/backends/codex"
	"web-console/backends/grok"
	"web-console/internal/config"
	"web-console/internal/pty"

	creackpty "github.com/creack/pty/v2"
)

// localPTYAvailable is a compile-time platform seam. The implementation file
// itself is selected by the build, so this is only a truthful descriptor hook,
// not a runtime capability switch.
func localPTYAvailable() bool { return true }

// realPTY wraps a creack/pty process.
type realPTY struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

func (p *realPTY) Read(buf []byte) (int, error) { return p.ptmx.Read(buf) }

// WriteInput writes bytes directly to the PTY master. For the standard
// (non-tmux) backend, keystroke and paste are indistinguishable at the
// kernel level — they end up in the same pipe either way.
func (p *realPTY) WriteInput(data []byte, _ pty.InputKind) error {
	_, err := p.ptmx.Write(data)
	return err
}
func (p *realPTY) Close() error { return p.ptmx.Close() }

func (p *realPTY) SetSize(cols, rows uint16) error {
	return creackpty.Setsize(p.ptmx, &creackpty.Winsize{Rows: rows, Cols: cols})
}

func (p *realPTY) Kill() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (p *realPTY) ExitCode() int {
	return exitCodeOnce(p.cmd)
}

// ProbeReady is a no-op for the standard PTY: pty.StartWithSize has already
// opened the master/slave pair synchronously, so the next Write will reach
// the shell without any additional handshake.
func (p *realPTY) ProbeReady(_ context.Context) error { return nil }

func (p *realPTY) CurrentDir(_ context.Context) (string, error) {
	if p.cmd == nil || p.cmd.Process == nil {
		cwd, err := filepath.Abs(config.ResolveWorkingDir())
		if err != nil {
			return config.ResolveWorkingDir(), nil
		}
		return cwd, nil
	}
	return platform.ProcessWorkingDir(p.cmd.Process.Pid)
}

// defaultPTYFactory starts a real shell process with a PTY.
func defaultPTYFactory(spec pty.LaunchSpec) (pty.PTY, error) {
	cmd := exec.Command(spec.Shell)
	// Filter Claude Code env vars first, then ensure TERM is set.
	// This prevents nested session detection when users run `claude` in
	// web-console terminals, even if the server was started from Claude Code.
	cmd.Env = applySessionEnv(ensureTermEnv(filterServiceEnv(claude.FilterEnv(os.Environ()))), spec.Env)
	cmd.Dir = resolveLaunchDir(spec)
	ptmx, err := creackpty.StartWithSize(cmd, &creackpty.Winsize{Rows: spec.Rows, Cols: spec.Cols})
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	return &realPTY{ptmx: ptmx, cmd: cmd}, nil
}

// resolveLaunchDir gives both PTY backends the same working-directory rule.
func resolveLaunchDir(spec pty.LaunchSpec) string {
	if strings.TrimSpace(spec.WorkingDir) != "" {
		return spec.WorkingDir
	}
	return config.ResolveWorkingDir()
}

// exitCodeOnce returns an exec.Cmd's exit code without calling Wait twice.
// ProcessState is populated by Wait, and a second Wait can panic on some Go
// platforms.
func exitCodeOnce(cmd *exec.Cmd) int {
	if cmd == nil {
		return -1
	}
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return -1
	}
	return 0
}

// ensureTermEnv guarantees TERM=xterm-256color in the environment slice.
// The server's own TERM (which may be "dumb" or unset when launched via
// systemd/lifecycle) is irrelevant — the browser-side xterm.js emulator
// supports xterm-256color, and shell programs need this to emit colors.
func ensureTermEnv(env []string) []string {
	const want = "TERM=xterm-256color"
	for i, v := range env {
		if strings.HasPrefix(v, "TERM=") {
			env[i] = want
			return env
		}
	}
	return append(env, want)
}

// serviceEnvVars lists environment variables that belong to the parent
// service process (web-console) and must not leak into terminal sessions.
// Without this filter, scenario CLIs (e.g. tunnel-manager) inherit
// web-console's API_PORT and connect to the wrong API.
//
// Vrooli lifecycle vars (VROOLI_LIFECYCLE_MANAGED, VROOLI_SCENARIO, etc.)
// are also stripped because the tmux server inherits them and the autoheal
// orphan checker then detects the tmux server as a Vrooli process. Since
// tmux is not tracked by the lifecycle system, it gets classified as an
// "orphan" and killed — destroying all persistent sessions.
//
// Host-terminal vars (TMUX, TMUX_PANE, TERM_PROGRAM, TERM_PROGRAM_VERSION)
// are stripped because web-console-api typically runs inside the user's own
// terminal (often tmux itself) and these would otherwise leak into every
// child shell. For the standard backend this is critical: leaving TMUX set
// makes programs like Claude Code think they're in tmux and emit tmux DCS
// passthrough escapes that nothing consumes, producing a silent hang before
// any UI is rendered. For the persistent backend this is a no-op because
// tmux re-sets TMUX/TMUX_PANE for each pane it spawns.
var serviceEnvVars = map[string]struct{}{
	"API_PORT":                 {},
	"API_BASE_URL":             {},
	"API_BASE":                 {},
	"UI_PORT":                  {},
	"WS_PORT":                  {},
	"VITE_API_BASE_URL":        {},
	"VROOLI_LIFECYCLE_MANAGED": {},
	"VROOLI_SCENARIO":          {},
	"VROOLI_STEP":              {},
	"VROOLI_PHASE":             {},
	"TMUX":                     {},
	"TMUX_PANE":                {},
	"TERM_PROGRAM":             {},
	"TERM_PROGRAM_VERSION":     {},
	// NO_COLOR belongs to the web-console host environment, not to the
	// interactive child. Keeping it would make color-aware CLI programs
	// suppress ANSI output even though the child is attached to a real PTY.
	"NO_COLOR": {},
}

// filterServiceEnv removes service-specific environment variables that
// belong to the parent process (web-console). These variables would cause
// other scenario CLIs running in terminal sessions to misdetect their API
// endpoint, connecting to web-console instead of their own service.
func filterServiceEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, v := range env {
		name, _, ok := strings.Cut(v, "=")
		if ok {
			if _, blocked := serviceEnvVars[name]; blocked {
				continue
			}
		}
		result = append(result, v)
	}
	return result
}

func applySessionEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}

	applied := make([]string, 0, len(base)+len(extra))
	seen := make(map[string]struct{}, len(extra))
	for _, entry := range base {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			applied = append(applied, entry)
			continue
		}
		if value, found := extra[name]; found {
			applied = append(applied, name+"="+value)
			seen[name] = struct{}{}
			continue
		}
		applied = append(applied, entry)
	}
	for name, value := range extra {
		if _, ok := seen[name]; ok {
			continue
		}
		applied = append(applied, name+"="+value)
	}
	return applied
}

// sessionCodexHome returns the per-session CODEX_HOME, delegating layout
// to backends/codex. Kept in package main as a thin wrapper so callers
// don't need to plumb the state root.
func sessionCodexHome(sessionID string) string {
	return codex.SessionHomePath(resolveSessionStateRoot(), sessionID)
}

// sessionCodexSessionsDir returns the per-session rollout JSONL dir.
func sessionCodexSessionsDir(sessionID string) string {
	return codex.SessionsDirPath(resolveSessionStateRoot(), sessionID)
}

// sessionGrokHome returns the per-session GROK_HOME, delegating layout to
// backends/grok. Each pane gets its own home so the grok transcripts beneath it
// belong unambiguously to that pane.
func sessionGrokHome(sessionID string) string {
	return grok.SessionHomePath(resolveSessionStateRoot(), sessionID)
}

// sessionGrokSessionsDir returns the per-session grok transcript root. grok
// writes <dir>/<url-encoded-cwd>/<session-id>/updates.jsonl beneath it.
func sessionGrokSessionsDir(sessionID string) string {
	return grok.SessionsDirPath(resolveSessionStateRoot(), sessionID)
}
