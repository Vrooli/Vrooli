package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty/v2"
)

// DOC: docs/internal/SEAMS.md#pty-factory-seam-api
// PTY represents a pseudo-terminal process with read/write, resize, and
// lifecycle control. The default implementation wraps creack/pty; tests can
// substitute a pipe-based fake via PTYFactory.
type PTY interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	SetSize(cols, rows uint16) error
	Close() error
	Kill() error
	// ExitCode waits for the process to finish and returns its exit code.
	// Call only after Read returns an error (process exited). Returns -1 if
	// the exit code cannot be determined.
	ExitCode() int
	// HasChildProcess reports whether the shell process has any child
	// processes running (e.g. a script, interactive program, etc.).
	HasChildProcess() bool
	// ProbeReady blocks until the PTY pipeline is confirmed to be accepting
	// writes that will reach the underlying process. For synchronous
	// backends (realPTY) this is a no-op; for async backends (tmuxPTY)
	// this waits for the attach-session handshake to complete.
	ProbeReady(ctx context.Context) error
}

// SessionLaunchSpec contains the environment and execution parameters for a
// newly created terminal session.
type SessionLaunchSpec struct {
	SessionID string
	Shell     string
	Cols      uint16
	Rows      uint16
	Env       map[string]string
}

// PTYFactory creates a PTY-backed process for the given launch spec.
// Inject a custom factory into SessionManager for testing without real processes.
type PTYFactory func(spec SessionLaunchSpec) (PTY, error)

// realPTY wraps a creack/pty process.
type realPTY struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

func (p *realPTY) Read(buf []byte) (int, error)  { return p.ptmx.Read(buf) }
func (p *realPTY) Write(buf []byte) (int, error) { return p.ptmx.Write(buf) }
func (p *realPTY) Close() error                  { return p.ptmx.Close() }

func (p *realPTY) SetSize(cols, rows uint16) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (p *realPTY) Kill() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (p *realPTY) ExitCode() int {
	if err := p.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return -1
	}
	return 0
}

// ProbeReady is a no-op for the standard PTY: pty.StartWithSize has already
// opened the master/slave pair synchronously, so the next Write will reach
// the shell without any additional handshake.
func (p *realPTY) ProbeReady(_ context.Context) error { return nil }

func (p *realPTY) HasChildProcess() bool {
	if p.cmd.Process == nil {
		return false
	}
	childrenPath := fmt.Sprintf("/proc/%d/task/%d/children", p.cmd.Process.Pid, p.cmd.Process.Pid)
	data, err := os.ReadFile(childrenPath)
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(data)) > 0
}

// defaultPTYFactory starts a real shell process with a PTY.
func defaultPTYFactory(spec SessionLaunchSpec) (PTY, error) {
	cmd := exec.Command(spec.Shell)
	// Filter Claude Code env vars first, then ensure TERM is set.
	// This prevents nested session detection when users run `claude` in
	// web-console terminals, even if the server was started from Claude Code.
	cmd.Env = applySessionEnv(ensureTermEnv(filterServiceEnv(filterClaudeEnv(os.Environ()))), spec.Env)
	cmd.Dir = resolveWorkingDir()
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: spec.Rows, Cols: spec.Cols})
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	return &realPTY{ptmx: ptmx, cmd: cmd}, nil
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

// filterClaudeEnv removes Claude Code-specific environment variables from
// the environment slice. This prevents nested session detection when users
// run `claude` in a web-console terminal, even if the web-console server
// was started from within a Claude Code session.
//
// Filtered patterns:
//   - CLAUDECODE (nested session detection marker)
//   - CLAUDE_* (session IDs, config paths, internal state)
//   - BASH_FUNC_claude_code::* (exported shell functions)
func filterClaudeEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, v := range env {
		// Filter CLAUDECODE exactly
		if strings.HasPrefix(v, "CLAUDECODE=") {
			continue
		}
		// Filter all CLAUDE_* variables
		if strings.HasPrefix(v, "CLAUDE_") {
			continue
		}
		// Filter exported bash functions for claude_code::
		if strings.HasPrefix(v, "BASH_FUNC_claude_code::") {
			continue
		}
		result = append(result, v)
	}
	return result
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

func ensureDir(path string) string {
	if path == "" {
		return path
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return path
	}
	return path
}

func resolveUserConfigDir(name string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, name)
}

func sharedCodexHome() string {
	return ensureDir(resolveUserConfigDir(".codex"))
}

func ensureSymlink(dst, src string) {
	info, err := os.Lstat(dst)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(dst)
			if readErr == nil && target == src {
				return
			}
		}
		return
	}
	if !os.IsNotExist(err) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		log.Printf("session-state: failed creating parent for %s: %v", dst, err)
		return
	}
	if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
		log.Printf("session-state: failed linking %s -> %s: %v", dst, src, err)
	}
}

func prepareCodexSessionHome(sessionHome, sharedHome string) string {
	sessionHome = ensureDir(sessionHome)
	if sessionHome == "" {
		return sessionHome
	}

	for _, dir := range []string{"sessions", "log", "logs", "outputs", "tmp"} {
		ensureDir(filepath.Join(sessionHome, dir))
	}

	if sharedHome == "" {
		return sessionHome
	}

	for _, entry := range []string{
		"auth.json",
		"config.toml",
		"settings.json",
		"skills",
		"rules",
		"version.json",
		".personality_migration",
	} {
		src := filepath.Join(sharedHome, entry)
		if _, err := os.Lstat(src); err != nil {
			continue
		}
		ensureSymlink(filepath.Join(sessionHome, entry), src)
	}
	return sessionHome
}

func sessionCodexHome(sessionID string) string {
	return prepareCodexSessionHome(
		filepath.Join(resolveSessionStateRoot(), "codex", sessionID),
		sharedCodexHome(),
	)
}

func sessionCodexSessionsDir(sessionID string) string {
	return ensureDir(filepath.Join(sessionCodexHome(sessionID), "sessions"))
}
