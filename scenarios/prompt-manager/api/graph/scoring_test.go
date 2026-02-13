package graph

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeScenarioProvider struct {
	scoreByScenario map[string]float64
	err             error
}

func (f *fakeScenarioProvider) ScenarioScore(_ context.Context, scenario string) (float64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.scoreByScenario[scenario], nil
}

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

// ---------------------------------------------------------------------------
// codeUsageScore 3-level tests
// ---------------------------------------------------------------------------

func TestCodeUsageScore_VrooliOnly(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "skill-a", Type: NodeSkill}},
		Edges: []Edge{
			{From: "skill-a", To: "cli:vrooli", Kind: EdgeCodeUsage, Category: CodeScenarioCLI},
		},
	}
	score := codeUsageScore("skill-a", g)
	if score != 1.0 {
		t.Errorf("expected 1.0 for Vrooli-only, got %f", score)
	}
}

func TestCodeUsageScore_HasExternal(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "skill-a", Type: NodeSkill}},
		Edges: []Edge{
			{From: "skill-a", To: "cli:grep", Kind: EdgeCodeUsage, Category: CodeExternalTool},
		},
	}
	score := codeUsageScore("skill-a", g)
	if score != 0.1 {
		t.Errorf("expected 0.1 for external tool, got %f", score)
	}
}

func TestCodeUsageScore_MixedVrooliAndExternal(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "skill-a", Type: NodeSkill}},
		Edges: []Edge{
			{From: "skill-a", To: "cli:vrooli", Kind: EdgeCodeUsage, Category: CodeScenarioCLI},
			{From: "skill-a", To: "cli:grep", Kind: EdgeCodeUsage, Category: CodeExternalTool},
		},
	}
	score := codeUsageScore("skill-a", g)
	if score != 0.1 {
		t.Errorf("expected 0.1 (external dominates), got %f", score)
	}
}

func TestCodeUsageScore_HasScript(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "skill-a", Type: NodeSkill}},
		Edges: []Edge{
			{From: "skill-a", To: "cli:deploy.sh", Kind: EdgeCodeUsage, Category: CodeScript},
		},
	}
	score := codeUsageScore("skill-a", g)
	if score != 0.1 {
		t.Errorf("expected 0.1 for script usage, got %f", score)
	}
}

func TestCodeUsageScore_NoUsage(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "skill-a", Type: NodeSkill}},
	}
	score := codeUsageScore("skill-a", g)
	if score != 0.5 {
		t.Errorf("expected 0.5 for no usage, got %f", score)
	}
}

func TestApplyCLIHealthPolicy_VrooliNeutral(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "cli:vrooli", Type: NodeCLI},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:vrooli", Kind: EdgeCodeUsage, Category: CodeScenarioCLI, Command: "vrooli"},
		},
	}
	base := []HealthScore{{NodeID: "cli:vrooli", Score: 0.8, Factors: map[string]float64{"x": 0.8}}}
	got := ApplyCLIHealthPolicy(context.Background(), g, base, &fakeScenarioProvider{scoreByScenario: map[string]float64{}})
	if len(got) != 0 {
		t.Fatalf("expected no health score for vrooli CLI, got %+v", got)
	}
}

func TestApplyCLIHealthPolicy_ExternalToolZero(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "cli:grep", Type: NodeCLI},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:grep", Kind: EdgeCodeUsage, Category: CodeExternalTool, Command: "grep"},
		},
	}
	got := ApplyCLIHealthPolicy(context.Background(), g, nil, &fakeScenarioProvider{scoreByScenario: map[string]float64{}})
	if len(got) != 1 {
		t.Fatalf("expected one score, got %d", len(got))
	}
	if got[0].Score != 0.0 {
		t.Fatalf("expected score 0 for external tool, got %f", got[0].Score)
	}
}

func TestApplyCLIHealthPolicy_ScenarioCLIUsesScenarioScore(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "cli:prompt-manager", Type: NodeCLI},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:prompt-manager", Kind: EdgeCodeUsage, Category: CodeScenarioCLI, Command: "prompt-manager"},
		},
	}
	got := ApplyCLIHealthPolicy(context.Background(), g, nil, &fakeScenarioProvider{
		scoreByScenario: map[string]float64{"prompt-manager": 73},
	})
	if len(got) != 1 {
		t.Fatalf("expected one score, got %d", len(got))
	}
	if got[0].Score != 0.73 {
		t.Fatalf("expected normalized scenario score 0.73, got %f", got[0].Score)
	}
}

func TestApplyCLIHealthPolicy_ScenarioCLIProviderErrorFallsBackZero(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "cli:prompt-manager", Type: NodeCLI},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:prompt-manager", Kind: EdgeCodeUsage, Category: CodeScenarioCLI, Command: "prompt-manager"},
		},
	}
	got := ApplyCLIHealthPolicy(context.Background(), g, nil, &fakeScenarioProvider{
		err: errors.New("boom"),
	})
	if len(got) != 1 {
		t.Fatalf("expected one score, got %d", len(got))
	}
	if got[0].Score != 0 {
		t.Fatalf("expected fallback score 0, got %f", got[0].Score)
	}
}
