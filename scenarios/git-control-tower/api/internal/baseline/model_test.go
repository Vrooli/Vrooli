package baseline

import (
	"strings"
	"testing"
	"time"
)

func TestBaselineManifestV2SingleRunInvariant(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	m := BaselineManifest{
		Name: "before", Scenario: "foo", Branch: "agi", CreatedAt: time.Now().UTC(),
		Run: RunAnchor{
			RunID: "run-1", CaptureProfile: CaptureProfile, TreeDigest: "td:abc", PhaseSetDigest: "ps:abc",
			DescriptorSnapshotRef: "test-genie-run:run-1#descriptor-snapshot", DescriptorSnapshotDigest: "ds:abc", DescriptorSnapshotSchemaVersion: 1,
		}, SchemaVersion: SchemaVersion,
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	if got := m.RunID(); got != "run-1" {
		t.Fatalf("RunID = %q, want run-1", got)
	}

	missingRun := m
	missingRun.Run.RunID = ""
	if err := missingRun.Validate(); err == nil || !strings.Contains(err.Error(), "run id") {
		t.Fatalf("missing run error = %v", err)
	}
	wrongProfile := m
	wrongProfile.Run.CaptureProfile = "quick"
	if err := wrongProfile.Validate(); err == nil || !strings.Contains(err.Error(), CaptureProfile) {
		t.Fatalf("wrong profile error = %v", err)
	}
}

func TestWorseVerdictPreservesComparisonSeverity(t *testing.T) {
	if got := WorseVerdict(VerdictNewFailure, VerdictRegression); got != VerdictRegression {
		t.Fatalf("got %q", got)
	}
	if got := WorseVerdict(VerdictChanged, VerdictClean); got != VerdictChanged {
		t.Fatalf("advisory changed should outrank clean, got %q", got)
	}
}
