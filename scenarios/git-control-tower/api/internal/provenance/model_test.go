package provenance

import "testing"

func TestCommittedAttributionRequiresContentBackedEvidence(t *testing.T) {
	record := Record{RepositoryID: "repo", SubjectDigest: "sha256:subject", Files: []File{{Path: "api/a.go", State: Committed, Evidence: []Evidence{{CommitID: "commit"}}}}}
	if err := record.Validate(); err == nil {
		t.Fatal("path/commit metadata without content evidence was accepted")
	}
	record.Files[0].ContentDigest = "sha256:file"
	record.Files[0].Evidence[0].ContentDigest = "sha256:file"
	record.Files[0].Evidence[0].Visibility = "public"
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMixedAndUnknownAttributionRemainsValid(t *testing.T) {
	record := Record{RepositoryID: "repo", SubjectDigest: "sha256:subject", Files: []File{{Path: "a.go", State: Applied, Evidence: []Evidence{{RunID: "run-1", SandboxID: "sandbox-1", Visibility: "private"}}}, {Path: "b.go", State: Unknown}}}
	if _, err := record.Digest(); err != nil {
		t.Fatal(err)
	}
}

func TestDigestBindsContributorsAndEvidence(t *testing.T) {
	record := Record{RepositoryID: "repo", SubjectDigest: "sha256:subject", Files: []File{{Path: "a.go", State: Applied, Contributors: []string{"run-1"}, Evidence: []Evidence{{RunID: "run-1", SandboxID: "sandbox-1", ApplicationReceipt: "receipt-1", Visibility: "public"}}}}}
	one, err := record.Digest()
	if err != nil {
		t.Fatal(err)
	}
	record.Files[0].Evidence[0].ApplicationReceipt = "receipt-2"
	two, err := record.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if one == two {
		t.Fatal("provenance receipt change did not change digest")
	}
	record.Files[0].Evidence[0].ApplicationReceipt = "receipt-1"
	record.Files[0].Contributors = append(record.Files[0].Contributors, "run-2")
	three, err := record.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if one == three {
		t.Fatal("provenance contributor change did not change digest")
	}
}

func TestRejectsUnknownVisibility(t *testing.T) {
	record := Record{RepositoryID: "repo", SubjectDigest: "sha256:subject", Files: []File{{Path: "a.go", State: Applied, Evidence: []Evidence{{RunID: "run-1", Visibility: "internal"}}}}}
	if err := record.Validate(); err == nil {
		t.Fatal("unknown evidence visibility was accepted")
	}
}
