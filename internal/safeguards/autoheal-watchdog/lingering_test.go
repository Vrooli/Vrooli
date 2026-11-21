package autohealwatchdog

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// [REQ:WATCH-LINUX-002] Lingering is enabled only when it is off; an
// already-lingering host must not need privilege to converge the safeguard.
func TestEnsureLingeringSkipsCommandWhenAlreadyEnabled(t *testing.T) {
	originalLinger, originalRun := lingeringEnabledFn, hostreqkit.RunCommandFn
	t.Cleanup(func() { lingeringEnabledFn, hostreqkit.RunCommandFn = originalLinger, originalRun })
	var calls []string
	hostreqkit.RunCommandFn = func(command string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, command+" "+strings.Join(args, " "))
		return nil
	}
	config := map[string]any{"boot_policy": "dedicated"}
	host := hostreqkit.Host{OS: "linux", SupportsSystemd: true}

	lingeringEnabledFn = func(string) bool { return true }
	if err := ensureLingering(host, config, hostreqkit.EnsureOptions{SudoMode: "skip"}); err != nil {
		t.Fatalf("ensureLingering with lingering on returned %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("lingering already enabled but a command ran: %v", calls)
	}

	lingeringEnabledFn = func(string) bool { return false }
	_ = ensureLingering(host, config, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if len(calls) != 1 || !strings.Contains(calls[0], "enable-linger") {
		t.Fatalf("lingering off should run enable-linger once, got %v", calls)
	}
}
