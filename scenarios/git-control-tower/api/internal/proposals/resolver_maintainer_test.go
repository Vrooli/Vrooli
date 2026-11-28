package proposals

import (
	"context"
	"testing"

	"git-control-tower/internal/trailers"
)

func TestLocalResolverResolvesExistingCommitByItsRepositoryObjectID(t *testing.T) {
	git := NewMemoryGit(map[string]string{"a.go": "committed"})
	commitID, err := git.Head(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	ownerID, status, detail, err := testService(t).Resolver(repoFor(git)).Resolve(
		context.Background(), "Vrooli-Continues", commitID,
	)
	if err != nil || ownerID != commitID || status != trailers.Resolved || detail != "commit exists in this repository" {
		t.Fatalf("resolution = (%q, %q, %q, %v)", ownerID, status, detail, err)
	}
}
