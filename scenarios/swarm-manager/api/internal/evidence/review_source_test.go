package evidence

import (
	"path/filepath"
	"testing"

	"swarm-manager/internal/attemptstore"
	"swarm-manager/internal/review"
)

func TestLoadReviewRoundSourcesLimitsTraversalToRegisteredBacklogKinds(t *testing.T) {
	root := t.TempDir()
	if err := attemptstore.SaveRound(filepath.Join(root, "execute", "kept"), "review", review.Round{RoundNum: 2}); err != nil {
		t.Fatal(err)
	}
	if err := attemptstore.SaveRound(filepath.Join(root, "goals", "ignored"), "review", review.Round{RoundNum: 1}); err != nil {
		t.Fatal(err)
	}
	sources, err := LoadReviewRoundSources(root)
	if err != nil {
		t.Fatalf("LoadReviewRoundSources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("sources = %#v, want only registered backlog source", sources)
	}
	if got := sources[0]; got.Kind != "execute" || got.Name != "kept" || got.Round.RoundNum != 2 {
		t.Fatalf("source = %#v", got)
	}
}
