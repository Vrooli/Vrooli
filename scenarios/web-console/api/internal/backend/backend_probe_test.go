package backend

import (
	"errors"
	"strings"
	"testing"
)

func TestTmuxProbe_ReportsMissingCommand(t *testing.T) {
	previous := tmuxProbeCommand
	t.Cleanup(func() { tmuxProbeCommand = previous })
	tmuxProbeCommand = func(_ string, args ...string) error {
		if len(args) > 0 && args[0] == "paste-buffer" {
			return errors.New("unsupported paste flag")
		}
		return nil
	}

	reason := runTmuxProbeCommand("tmux", "paste-buffer", "-p")
	if !strings.Contains(reason, "paste-buffer -p") {
		t.Fatalf("reason = %q, want missing command name", reason)
	}
}
