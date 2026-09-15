package userconfig

import "testing"

// The stale-binary check reports a warning, so a "critical" trigger would leave
// it permanently detected and never acted on — the failure mode this check was
// added to remove.
func TestStaleServiceBinaryHealsOnWarning(t *testing.T) {
	got := GetCheckDefaults("system-stale-service-binary")
	if !got.Enabled {
		t.Error("check must be enabled by default")
	}
	if !got.AutoHeal {
		t.Error("stale supervised binaries must self-heal; that is the point of owning this")
	}
	if got.AutoHealOn != "warning+critical" {
		t.Fatalf("AutoHealOn = %q; a warning-level finding never fires on \"critical\"", got.AutoHealOn)
	}
}
