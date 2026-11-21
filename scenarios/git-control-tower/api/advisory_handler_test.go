package main

import (
	"testing"

	"git-control-tower/internal/advisory"
)

func TestBuildAdvisoryDraftBindsEvidenceToSubject(t *testing.T) {
	subject := advisory.ChangeSubject{RepositoryID: "repo", Kind: advisory.SubjectCurrent, Scope: advisory.Scope{Paths: []string{"api/a.go"}, SelectionDigest: "sha256:selection"}, SnapshotDigest: "sha256:snapshot"}
	digest, err := subject.Digest()
	if err != nil { t.Fatal(err) }
	result, err := buildAdvisoryDraft(advisoryDraftRequest{Kind: "pr-draft", Subject: subject, Evidence: advisory.EvidenceBundle{OperationID: "op-1", SubjectDigest: digest, Status: "partial", Claims: []advisory.Claim{{Text: "Adds bounded retries", EvidenceRefs: []string{"hunk:1"}}}, Coverage: advisory.Coverage{IncludedFiles: 1, OmittedFiles: 1, Omissions: []advisory.Omission{{Path: "asset.bin", Reason: "binary"}}}, Unknowns: []string{"rollout date is unknown"}}})
	if err != nil { t.Fatal(err) }
	if result.Status != "partial" || result.SubjectDigest != digest || result.Kind != "pr-draft" || len(result.EvidenceRefs) != 1 { t.Fatalf("unexpected draft: %+v", result) }
}

func TestBuildAdvisoryDraftRejectsDigestMismatch(t *testing.T) {
	subject := advisory.ChangeSubject{RepositoryID: "repo", Kind: advisory.SubjectCurrent, Scope: advisory.Scope{Paths: []string{"a.go"}, SelectionDigest: "sha256:selection"}, SnapshotDigest: "sha256:snapshot"}
	_, err := buildAdvisoryDraft(advisoryDraftRequest{Kind: "summary", Subject: subject, Evidence: advisory.EvidenceBundle{OperationID: "op", SubjectDigest: "sha256:wrong", Status: "complete", Coverage: advisory.Coverage{}}})
	if err == nil { t.Fatal("mismatched subject accepted") }
}
