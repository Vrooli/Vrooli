package userconfig

import "testing"

func TestHostIntegrityChecksAreObservableButNotAutoHealableByDefault(t *testing.T) {
	for _, id := range []string{
		"host-kernel-module-drift",
		"host-device-driver-binding",
		"host-runtime-integrity",
		"host-package-state",
		"host-kernel-error-signals",
		"host-capability-drift",
	} {
		defaults := GetCheckDefaults(id)
		if !defaults.Enabled || defaults.AutoHeal {
			t.Fatalf("%s defaults = %+v, want enabled and autoHeal=false", id, defaults)
		}
		if defaults.IntervalSeconds != 300 {
			t.Fatalf("%s interval = %d, want 300", id, defaults.IntervalSeconds)
		}
	}
}
