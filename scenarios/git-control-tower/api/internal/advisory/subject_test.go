package advisory

import "testing"

func validSubject() ChangeSubject {
	return ChangeSubject{RepositoryID: "repo-1", Kind: SubjectCurrent, Scope: Scope{Paths: []string{"api/service.go"}, SelectionDigest: "sha256:selection"}, SnapshotDigest: "sha256:snapshot"}
}

func TestSubjectDigestIsStableAcrossPathOrder(t *testing.T) {
	a := validSubject()
	a.Scope.Paths = []string{"z.go", "a.go"}
	b := validSubject()
	b.Scope.Paths = []string{"a.go", "z.go"}
	b.Scope.SelectionDigest = a.Scope.SelectionDigest
	da, err := a.Digest()
	if err != nil {
		t.Fatal(err)
	}
	db, err := b.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if da != db {
		t.Fatalf("digests differ: %s != %s", da, db)
	}
}

func TestSubjectRejectsAmbiguousPullRequestIdentity(t *testing.T) {
	s := validSubject()
	s.Kind = SubjectPullRequest
	s.Host = &HostSubject{Provider: "github", InstanceID: "github.com", ChangeNumber: "42"}
	if err := s.Validate(); err == nil {
		t.Fatal("incomplete host identity accepted")
	}
}

func TestSubjectRequiresImmutableEndpointsForRangesAndPullRequests(t *testing.T) {
	for _, kind := range []SubjectKind{SubjectRange, SubjectPullRequest} {
		s := validSubject()
		s.Kind = kind
		if kind == SubjectPullRequest {
			s.Host = &HostSubject{Provider: "github", InstanceID: "github.com", RepositoryID: "owner/repo", ChangeNumber: "42"}
		}
		if err := s.Validate(); err == nil {
			t.Fatalf("%s without immutable endpoints was accepted", kind)
		}
		s.BaseRevision = "base-1"
		s.HeadRevision = "head-1"
		if err := s.Validate(); err != nil {
			t.Fatalf("%s with immutable endpoints rejected: %v", kind, err)
		}
	}
}

func TestEvidenceBundleRejectsCompleteCoverageWithOmittedFiles(t *testing.T) {
	b := EvidenceBundle{OperationID: "op-1", SubjectDigest: "sha256:x", Status: "complete", Coverage: Coverage{IncludedFiles: 1, OmittedFiles: 1, Omissions: []Omission{{Path: "asset.bin", Reason: "binary"}}}, Claims: []Claim{{Text: "one", EvidenceRefs: []string{"artifact:1"}}}}
	if err := b.Validate(); err == nil {
		t.Fatal("complete evidence with an omission was accepted")
	}
	b.Status = "partial"
	if err := b.Validate(); err != nil {
		t.Fatalf("partial evidence with an explicit omission rejected: %v", err)
	}
}

func TestEvidenceBundleRejectsUnattributedClaimsAndMalformedReceipts(t *testing.T) {
	base := EvidenceBundle{OperationID: "op-1", SubjectDigest: "sha256:x", Status: "complete", Coverage: Coverage{IncludedFiles: 1}, Claims: []Claim{{Text: "one", EvidenceRefs: []string{"artifact:1"}}}}
	base.Claims[0].Text = " "
	if err := base.Validate(); err == nil {
		t.Fatal("blank claim text was accepted")
	}
	base.Claims[0].Text = "one"
	base.Validation = []ValidationRef{{ExecutionID: "", Availability: "available", Verdict: "passed"}}
	if err := base.Validate(); err == nil {
		t.Fatal("validation receipt without an execution identity was accepted")
	}
}
