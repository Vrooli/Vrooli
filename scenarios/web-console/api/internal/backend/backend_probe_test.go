package backend

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTmuxProbe_ReportsMissingCommand(t *testing.T) {
	previous := tmuxProbeCommand
	t.Cleanup(func() { tmuxProbeCommand = previous })
	tmuxProbeCommand = func(_ string, args ...string) error {
		if strings.Contains(strings.Join(args, " "), "paste-buffer") {
			return errors.New("unsupported paste flag")
		}
		return nil
	}

	reason := runTmuxProbeCommand("tmux", "paste-buffer", "-p")
	if !strings.Contains(reason, "paste-buffer -p") {
		t.Fatalf("reason = %q, want missing command name", reason)
	}
}

func TestDefaultTmuxProbeUsesDedicatedSocketAndCleansBuffer(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}

	socket := filepath.Join(t.TempDir(), "tmux", "wc.sock")
	t.Setenv("WC_TMUX_SOCKET", socket)
	tmuxProbeMu.Lock()
	tmuxProbeCache = make(map[string]struct {
		available bool
		reason    string
	})
	tmuxProbeMu.Unlock()

	available, reason := defaultCheckTmuxAvailable()
	if !available {
		t.Fatalf("dedicated tmux probe unavailable: %s", reason)
	}

	args := []string{"-S", socket, "list-buffers", "-F", "#{buffer_name}"}
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "no server running") {
		t.Fatalf("list dedicated buffers: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	if strings.Contains(string(out), "vrooli-probe-buffer") {
		t.Fatalf("probe buffer leaked on dedicated socket: %q", string(out))
	}

	// A probe must not create or mutate the operator's default tmux server.
	defaultOut, defaultErr := exec.Command("tmux", "list-buffers", "-F", "#{buffer_name}").CombinedOutput()
	if defaultErr == nil && strings.Contains(string(defaultOut), "vrooli-probe-buffer") {
		t.Fatalf("probe buffer leaked on default socket: %q", string(defaultOut))
	}
}

func TestTmuxProbeCommandsAllTargetDedicatedSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "tmux", "wc.sock")
	for _, args := range tmuxProbeCommands(socket, "probe-session") {
		if len(args) < 2 || args[0] != "-S" || args[1] != socket {
			t.Fatalf("probe command does not target dedicated socket: %#v", args)
		}
	}
}
