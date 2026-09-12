package scenarios

import "testing"

func TestRemediationTargetFingerprintIsSourceNeutral(t *testing.T) {
	manual := RemediationTarget{Scenario: " Swarm-Manager ", ProviderPhase: "unit", CapabilityID: "test_quality"}
	automatic := RemediationTarget{Scenario: "swarm-manager", ProviderPhase: "UNIT", CapabilityID: "TEST_QUALITY"}

	manualFingerprint, err := manual.Fingerprint()
	if err != nil {
		t.Fatalf("manual fingerprint: %v", err)
	}
	automaticFingerprint, err := automatic.Fingerprint()
	if err != nil {
		t.Fatalf("automatic fingerprint: %v", err)
	}
	if manualFingerprint != automaticFingerprint {
		t.Fatalf("fingerprints differ: %q != %q", manualFingerprint, automaticFingerprint)
	}
}

func TestRemediationTargetRejectsIncompleteIdentity(t *testing.T) {
	for _, target := range []RemediationTarget{
		{},
		{Scenario: "swarm-manager"},
		{Scenario: "swarm-manager", ProviderPhase: "unit"},
	} {
		if _, err := target.Fingerprint(); err == nil {
			t.Fatalf("expected invalid target %+v to fail", target)
		}
	}
}

func TestHealthSnapshotActionabilityOnlyAllowsFreshEvidence(t *testing.T) {
	for _, state := range []HealthEvidenceState{HealthEvidenceStale, HealthEvidenceDegraded, HealthEvidenceUnavailable, HealthEvidenceNone} {
		if (ScenarioHealthSnapshot{EvidenceState: state}).IsActionable() {
			t.Fatalf("%q evidence must not be actionable", state)
		}
	}
	if !(ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh}).IsActionable() {
		t.Fatal("fresh evidence should be actionable")
	}
}
