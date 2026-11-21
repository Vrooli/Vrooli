package vrooli

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// [REQ:BOOT-RECOVERY-001] The scenario observes boot protection; it never
// offers to install, enable, linger or uninstall the unit. Those are
// `vrooli setup`'s, and an action the API cannot perform is a lie in the UI.
func TestOsWatchdogOffersNoInstallActions(t *testing.T) {
	var check checks.Check = NewWatchdogCheck(&platform.Capabilities{Platform: "linux", SupportsSystemd: true})
	if healable, ok := check.(checks.HealableCheck); ok {
		for _, action := range healable.RecoveryActions(nil) {
			t.Errorf("os-watchdog offers recovery action %q; it must offer none", action.ID)
		}
		if len(healable.RecoveryActions(nil)) == 0 {
			t.Error("os-watchdog implements HealableCheck; observation-only checks must not")
		}
	}
	result := check.Run(context.Background())
	if result.Details["remediation"] != "vrooli setup" {
		t.Errorf("remediation = %v, want \"vrooli setup\"", result.Details["remediation"])
	}
}
