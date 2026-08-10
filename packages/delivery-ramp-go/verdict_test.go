package deliveryramp

import "testing"

func TestTargetVerdictContainsReferencesOnly(t *testing.T) {
	verdict, err := NewTargetVerdict(TargetVerdictInput{
		Producer: "reference-ramp", RunID: "run-1", Disposition: DispositionPass,
		Target:     Target{ID: "local-linux-amd64", Ramp: "reference-ramp", Platform: "desktop", OS: "linux", DeviceKind: "desktop", Transport: Transport{Kind: TransportLocal}, Available: true},
		References: []EvidenceReference{{ID: "recording-1", Kind: "recording", URI: "capture://recording-1", Checksum: "sha256:abc", Redacted: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if verdict.GetDisposition().String() != "DISPOSITION_PASSED" || len(verdict.GetRefs()) != 1 || verdict.GetRefs()[0].GetArtifactId() != "recording-1" {
		t.Fatalf("verdict = %+v", verdict)
	}
}

func TestTargetVerdictRejectsPassingWithoutEvidence(t *testing.T) {
	_, err := NewTargetVerdict(TargetVerdictInput{
		Producer: "ramp", RunID: "run-1", Disposition: DispositionPass,
		Target: Target{ID: "local", Platform: "desktop", Transport: Transport{Kind: TransportLocal}, Available: true},
	})
	if err == nil {
		t.Fatal("expected pass without evidence to be rejected")
	}
}
