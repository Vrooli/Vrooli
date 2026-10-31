package services

import (
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestFindRepoRootUsesRepoContract(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("ResolveRepoRoot() error = %v", err)
	}

	got, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("findRepoRoot() = %q, want %q", got, root)
	}
}
