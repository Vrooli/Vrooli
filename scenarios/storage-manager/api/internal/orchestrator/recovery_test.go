package orchestrator

import (
	"path/filepath"
	"testing"

	"storage-manager/internal/cleanup"
)

func TestRateRecoveryChildRequiresDirectGovernedGoWorkChild(t *testing.T) {
	home := t.TempDir()
	t.Setenv("VROOLI_HOME", home)
	base := filepath.Join(home, "tmp", "go-work")

	tests := []struct {
		name      string
		trigger   string
		partition string
		want      bool
	}{
		{"direct child", "PRESSURE_TRIGGER_RATE", filepath.Join(base, "chaos"), true},
		{"parent", "PRESSURE_TRIGGER_RATE", base, false},
		{"nested child", "PRESSURE_TRIGGER_RATE", filepath.Join(base, "chaos", "nested"), false},
		{"band trigger", "PRESSURE_TRIGGER_BAND", filepath.Join(base, "chaos"), false},
		{"outside", "PRESSURE_TRIGGER_RATE", filepath.Join(home, "tmp", "other"), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rateRecoveryChild(test.trigger, test.partition); got != test.want {
				t.Fatalf("rateRecoveryChild(%q, %q) = %t, want %t", test.trigger, test.partition, got, test.want)
			}
		})
	}
}

func TestRecoveryFilesRemovedUsesAppliedIDsOrBoundedBatch(t *testing.T) {
	batch := cleanup.Preview{Items: []cleanup.PreviewItem{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	if got := recoveryFilesRemoved(cleanup.ApplyResult{AppliedItems: []string{"a"}, ReclaimedBytes: 1}, batch); got != 1 {
		t.Fatalf("explicit applied count = %d, want 1", got)
	}
	if got := recoveryFilesRemoved(cleanup.ApplyResult{ReclaimedBytes: 1, SkippedItems: []string{"c"}}, batch); got != 2 {
		t.Fatalf("bounded inferred count = %d, want 2", got)
	}
	if got := recoveryFilesRemoved(cleanup.ApplyResult{ReclaimedBytes: 0}, batch); got != 0 {
		t.Fatalf("zero-progress count = %d, want 0", got)
	}
}

func TestRecoveryTargetForRunServicesRateRootEvenWhenDeviceTargetMet(t *testing.T) {
	if got := recoveryTargetForRun("PRESSURE_TRIGGER_RATE", "/tmp/hot", 73, 500, 0); got != 501 {
		t.Fatalf("rate root target = %d, want one byte above observed free space", got)
	}
	if got := recoveryTargetForRun("PRESSURE_TRIGGER_BAND", "/", 73, 500, 0); got <= 500 {
		t.Fatalf("ordinary pressure target = %d, want device target above current free space", got)
	}
}
