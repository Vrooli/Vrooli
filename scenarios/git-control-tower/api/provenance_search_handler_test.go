package main

import "testing"

func TestRankProvenanceSearchIsBoundedAndStable(t *testing.T) {
	groups := []workspaceSandboxProvenanceRunGroup{
		{RunID: "run-1", SandboxID: "sb-1", SandboxOwner: "alice", Files: []workspaceSandboxProvenanceFile{{RelativePath: "api/review.go", FilePath: "api/review.go", ChangeType: "modified"}}},
		{RunID: "run-2", SandboxID: "sb-2", SandboxOwner: "bob", Files: []workspaceSandboxProvenanceFile{{RelativePath: "docs/review.md", FilePath: "docs/review.md", ChangeType: "added"}}},
	}
	got := rankProvenanceSearch(groups, "run-1 review.go", 1)
	if len(got) != 1 || got[0].RunID != "run-1" || got[0].RelativePath != "api/review.go" {
		t.Fatalf("unexpected exact provenance result: %+v", got)
	}
	if got[0].ID == "" || got[0].EvidenceStanding != "applied" {
		t.Fatalf("result lacks stable evidence identity: %+v", got[0])
	}
}

func TestRankProvenanceSearchReturnsNoMatchesWithoutClaimingUnavailable(t *testing.T) {
	got := rankProvenanceSearch(nil, "missing", 20)
	if got == nil {
		t.Fatal("expected a present empty result set")
	}
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %+v", got)
	}
}

func TestRankProvenanceSearchDoesNotLeakPrivateFiles(t *testing.T) {
	groups := []workspaceSandboxProvenanceRunGroup{
		{RunID: "private-run", Files: []workspaceSandboxProvenanceFile{{RelativePath: "secret.txt", FilePath: "secret.txt", Visibility: "private"}}},
		{RunID: "public-run", Files: []workspaceSandboxProvenanceFile{{RelativePath: "README.md", FilePath: "README.md", Visibility: "public"}}},
	}
	got := rankProvenanceSearch(groups, "secret README", 20)
	if len(got) != 1 || got[0].RunID != "public-run" {
		t.Fatalf("private provenance leaked or public result missing: %+v", got)
	}
}
