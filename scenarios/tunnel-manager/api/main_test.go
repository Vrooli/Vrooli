package main

import (
	"os"
	"testing"
)

// TestRecoverySchedulerEnabledFromEnv pins the opt-out lever: recovery is
// default-on and only the symmetric TUNNEL_MANAGER_RECOVERY_SCHEDULER_DISABLED
// (1/true/yes) turns it off — matching the probe and exposure schedulers. The
// old opt-in TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED name no longer exists.
func TestRecoverySchedulerEnabledFromEnv(t *testing.T) {
	const key = "TUNNEL_MANAGER_RECOVERY_SCHEDULER_DISABLED"
	cases := []struct {
		name string
		set  bool
		val  string
		want bool
	}{
		{name: "unset_defaults_on", set: false, want: true},
		{name: "empty_defaults_on", set: true, val: "", want: true},
		{name: "disabled_1", set: true, val: "1", want: false},
		{name: "disabled_true", set: true, val: "true", want: false},
		{name: "disabled_yes", set: true, val: "yes", want: false},
		{name: "disabled_TRUE_caseinsensitive", set: true, val: "TRUE", want: false},
		{name: "other_value_stays_on", set: true, val: "0", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(key)
			if tc.set {
				t.Setenv(key, tc.val)
			}
			if got := recoverySchedulerEnabledFromEnv(); got != tc.want {
				t.Fatalf("recoverySchedulerEnabledFromEnv() with %s=%q (set=%v) = %v, want %v", key, tc.val, tc.set, got, tc.want)
			}
		})
	}

	// The deleted opt-in name must have no effect — greenfield, no alias.
	os.Unsetenv(key)
	t.Setenv("TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED", "1")
	if !recoverySchedulerEnabledFromEnv() {
		t.Fatalf("legacy *_ENABLED var must not gate recovery; recovery should stay default-on")
	}
}
