package runs

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

func TestDescriptorSnapshotRoundTripAndDigestValidation(t *testing.T) { // [REQ:TESTGENIE-DESCRIPTOR-SNAPSHOT-P0]
	scenarioDir := t.TempDir()
	snapshot, err := NewDescriptorSnapshot([]PhaseDescriptorSnapshot{{
		Phase: "unit", DisplayName: "Unit Health", Provider: "unit-health", OrderHint: 20,
		EvidenceKinds: []string{"coverage.report"},
		Applicability: ApplicabilityDecisionSnapshot{Status: "applies", Planned: true},
	}})
	if err != nil {
		t.Fatalf("NewDescriptorSnapshot: %v", err)
	}
	if err := WriteDescriptorSnapshot(scenarioDir, "run-1", snapshot); err != nil {
		t.Fatalf("WriteDescriptorSnapshot: %v", err)
	}
	loaded, err := ReadDescriptorSnapshot(scenarioDir, "run-1")
	if err != nil {
		t.Fatalf("ReadDescriptorSnapshot: %v", err)
	}
	if loaded.Digest != snapshot.Digest || loaded.Phases[0].DisplayName != "Unit Health" {
		t.Fatalf("round trip = %+v, want digest %q and captured display name", loaded, snapshot.Digest)
	}

	path := sharedartifacts.RunDescriptorSnapshotPath(scenarioDir, "run-1")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var tampered map[string]any
	if err := json.Unmarshal(raw, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["phases"].([]any)[0].(map[string]any)["display_name"] = "Rewritten Live Label"
	raw, _ = json.Marshal(tampered)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDescriptorSnapshot(scenarioDir, "run-1"); !errors.Is(err, ErrInvalidDescriptorSnapshot) {
		t.Fatalf("tampered snapshot error = %v, want ErrInvalidDescriptorSnapshot", err)
	}
}

func TestDescriptorSnapshotRejectsFutureSchema(t *testing.T) {
	snapshot, err := NewDescriptorSnapshot([]PhaseDescriptorSnapshot{{
		Phase: "unit", Applicability: ApplicabilityDecisionSnapshot{Status: "applies", Planned: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	snapshot.SchemaVersion++
	if err := WriteDescriptorSnapshot(t.TempDir(), "run-1", snapshot); !errors.Is(err, ErrUnsupportedDescriptorSnapshotVersion) {
		t.Fatalf("future schema error = %v", err)
	}
}

func TestPhaseComparisonFingerprintIncludesValidationContract(t *testing.T) {
	base := PhaseDescriptorSnapshot{Phase: "unit", ValidationContract: "scenario-validation/v1", ValidationDeliveryMode: "inline"}
	changed := base
	changed.ValidationDeliveryMode = "durable-run"
	a, err := PhaseComparisonFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}
	b, err := PhaseComparisonFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("validation delivery change must invalidate the semantic comparison fingerprint")
	}
}
