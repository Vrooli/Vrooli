package graph

import (
	"testing"
	"time"
)

func TestScoreAll_Basic(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "skill-a", Type: NodeSkill},
			{ID: "skill-b", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "x", To: "skill-a", Kind: EdgeCLIRead},
			{From: "y", To: "skill-a", Kind: EdgeBoldListed},
		},
	}

	fns := DefaultScoreFns()
	scores := ScoreAll(g, fns)

	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}

	// skill-a should have a higher score than skill-b (more incoming edges)
	var scoreA, scoreB float64
	for _, s := range scores {
		if s.NodeID == "skill-a" {
			scoreA = s.Score
		}
		if s.NodeID == "skill-b" {
			scoreB = s.Score
		}
	}
	if scoreA <= scoreB {
		t.Errorf("expected skill-a score (%f) > skill-b score (%f)", scoreA, scoreB)
	}
}

func TestRecentActivityScoreFromTimestamp(t *testing.T) {
	// Recent (within 7 days)
	recent := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	score := RecentActivityScoreFromTimestamp(recent)
	if score != 1.0 {
		t.Errorf("expected 1.0 for recent timestamp, got %f", score)
	}

	// Old (beyond 90 days)
	old := time.Now().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	score = RecentActivityScoreFromTimestamp(old)
	if score != 0.0 {
		t.Errorf("expected 0.0 for old timestamp, got %f", score)
	}

	// Invalid
	score = RecentActivityScoreFromTimestamp("not-a-date")
	if score != 0.0 {
		t.Errorf("expected 0.0 for invalid timestamp, got %f", score)
	}
}

func TestScoreAll_NoFunctions(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "a", Type: NodeSkill}},
	}

	scores := ScoreAll(g, nil)
	if len(scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(scores))
	}
	if scores[0].Score != 0 {
		t.Errorf("expected 0 score with no functions, got %f", scores[0].Score)
	}
}
