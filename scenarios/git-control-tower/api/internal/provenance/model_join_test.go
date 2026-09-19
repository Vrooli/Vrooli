package provenance

import "testing"

func TestJoinEvidenceRequiresExactDigestAndCommit(t *testing.T) {
	input := JoinInput{NativeCommit: "c1", NativeDigest: "sha256:x", Complete: true, Evidence: []Evidence{{CommitID: "c1", ContentDigest: "sha256:x", RunID: "r1", Visibility: "public"}}}
	if got := JoinEvidence(input); got.Standing != ExactContent {
		t.Fatalf("standing=%s reasons=%v", got.Standing, got.Reasons)
	}
	input.Evidence[0].ContentDigest = "sha256:other"
	if got := JoinEvidence(input); got.Standing != CommitFile {
		t.Fatalf("path/commit evidence should downgrade: %s", got.Standing)
	}
	input.Evidence[0].CommitID = "other"
	if got := JoinEvidence(input); got.Standing != RunFile {
		t.Fatalf("run overlap should remain run-file: %s", got.Standing)
	}
}

func TestJoinEvidenceRedactsPrivateStanding(t *testing.T) {
	got := JoinEvidence(JoinInput{NativeCommit: "c1", NativeDigest: "sha256:x", Complete: true, Evidence: []Evidence{{CommitID: "c1", ContentDigest: "sha256:x", Visibility: "private"}}})
	if got.Standing != Private || len(got.Evidence) != 0 {
		t.Fatalf("private evidence leaked: %+v", got)
	}
}
