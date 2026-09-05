package projectcli

import "testing"

// [REQ:BOOT-RECOVERY-001] `setup status --phase readiness` selects the
// inspection-only readiness phase without disturbing the lifecycle options.
func TestParseSetupOptionsStatusPhase(t *testing.T) {
	opts, err := ParseSetupOptions([]string{"status", "--phase", "readiness", "--environment", "development"})
	if err != nil {
		t.Fatalf("ParseSetupOptions: %v", err)
	}
	if opts.Subcommand != "status" || opts.Phase != "readiness" || opts.Environment != "development" {
		t.Fatalf("opts = %+v", opts)
	}
	opts, err = ParseSetupOptions([]string{"status"})
	if err != nil {
		t.Fatalf("ParseSetupOptions: %v", err)
	}
	if opts.Phase != "" {
		t.Fatalf("default phase should be empty (setup), got %q", opts.Phase)
	}
	if _, err := ParseSetupOptions([]string{"--phase", "readiness"}); err == nil {
		t.Fatal("--phase is a status-only option and must be rejected on the apply flow")
	}
}
