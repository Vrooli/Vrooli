package deliveryramp

import "testing"

func TestSourceProfileIsIndependentOfRuntimeProfiles(t *testing.T) {
	if DeliveryFormatSourceRepository == "linux" || DeliveryFormatSourceRepository == "github" {
		t.Fatal("source format must not be an OS or destination alias")
	}
	if len(SourceEvidenceProfile()) != 8 {
		t.Fatalf("source gate count = %d", len(SourceEvidenceProfile()))
	}
}

func TestSourceEvidenceFailsClosedForMissingAndMismatchedCandidate(t *testing.T) {
	v := EvaluateSourceEvidence(SourceEvidence{CandidateDigest: "sha256:candidate", ArchiveDigest: "sha256:archive"})
	if v.Disposition == GatePassed || len(v.Failures) < 5 {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestSourceEvidencePassesOnlyExactCompleteTuple(t *testing.T) {
	e := SourceEvidence{CandidateDigest: "sha256:archive", SourceDigest: "sha256:source", ClosureDigest: "sha256:closure", PolicyDigest: "sha256:policy", ArchiveDigest: "sha256:archive", Reproducible: true, CleanBuild: GatePassed, Documentation: GatePassed, Governance: GatePassed, Gates: map[SourceGate]GateDisposition{SourceGatePinned: GatePassed, SourceGateClosure: GatePassed, SourceGatePolicy: GatePassed, SourceGateIntegrity: GatePassed}}
	v := EvaluateSourceEvidence(e)
	if v.Disposition != GatePassed || len(v.Passed) != 8 {
		t.Fatalf("verdict = %+v", v)
	}
}
