package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

// tmuxSessionPrefix mirrors the package-main constant; duplicated so session
// recovery tests can address tmux sessions without importing package main.
const tmuxSessionPrefix = "wc-"

// tmuxCmd runs a tmux command against the per-test socket (configured by
// useIsolatedTmuxSocket via WC_TMUX_SOCKET). Mirrors the package-main helper.
func tmuxCmd(args ...string) *exec.Cmd {
	socket := os.Getenv("WC_TMUX_SOCKET")
	if socket == "" {
		socket = "default"
	}
	fullArgs := append([]string{"-L", socket}, args...)
	return exec.Command("tmux", fullArgs...)
}

// tmuxCmdForSocket runs a tmux command against an explicitly-named socket.
// Used by tests that simulate cross-socket isolation.
func tmuxCmdForSocket(socket string, args ...string) *exec.Cmd {
	fullArgs := append([]string{"-L", socket}, args...)
	return exec.Command("tmux", fullArgs...)
}

// useIsolatedSessionState points WC_SESSION_STATE_ROOT at a per-test temp
// directory so recovery/tailer tests can't read or delete the live app's
// session state. Mirrors the helper in package main; duplicated here so the
// session package tests don't need to import package main.
func useIsolatedSessionState(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "session-state")
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	return root
}

// setupRealTmuxHooks wires production-equivalent tmux hooks on sm for
// recovery tests that create real tmux sessions to verify Recover()'s end-
// to-end behavior. The discover function shells out to tmux via the per-test
// socket; the attach function returns a fake PTY because the recovery tests
// only assert on metadata/status — they don't inspect the PTY itself.
func setupRealTmuxHooks(t *testing.T, sm *Manager) {
	t.Helper()
	discover := func() ([]string, error) {
		out, err := tmuxCmd("list-sessions", "-F", "#{session_name}").Output()
		if err != nil {
			return nil, nil
		}
		var sessions []string
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if strings.HasPrefix(line, tmuxSessionPrefix) {
				sessions = append(sessions, strings.TrimPrefix(line, tmuxSessionPrefix))
			}
		}
		return sessions, nil
	}
	attach := func(string) (pty.PTY, error) {
		return ptyfake.NewFakePTYWithOutput(), nil
	}
	killSession := func(name string) {
		_ = tmuxCmd("kill-session", "-t", name).Run()
	}
	sm.SetTmuxHooks(
		attach,
		discover,
		func(string) {},
		killSession,
		tmuxSessionPrefix,
	)
}

// useIsolatedTmuxSocket points WC_TMUX_SOCKET at a per-test value so tmux
// integration tests can't collide with the live app's tmux server.
func useIsolatedTmuxSocket(t *testing.T) string {
	t.Helper()
	socket := "wc-test-" + sanitizeTestIdentifier(t.Name()+"-"+filepath.Base(t.TempDir()))
	t.Setenv("WC_TMUX_SOCKET", socket)
	t.Setenv("WC_TMUX_SCOPE_NAME", socket+"-scope")
	return socket
}

func sanitizeTestIdentifier(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "test"
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}
