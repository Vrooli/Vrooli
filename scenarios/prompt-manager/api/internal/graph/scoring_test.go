package graph

import (
	"context"
	"errors"
	"fmt"
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

func TestScoreAllWithConfig_UsesEntityWeights(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "team-1", Type: NodeTeam},
			{ID: "team-2", Type: NodeTeam},
		},
		Edges: []Edge{
			{From: "team-1", To: "x", Kind: EdgeMembership},
		},
	}
	cfg := DefaultHealthConfig()
	cfg.Team = HealthWeights{
		OutgoingEdges:  2.0,
		IncomingEdges:  0,
		CodeUsage:      0,
		RecentActivity: 0,
	}

	scores := ScoreAllWithConfig(g, cfg)
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}

	var score1, score2 float64
	for _, hs := range scores {
		if hs.NodeID == "team-1" {
			score1 = hs.Score
		}
		if hs.NodeID == "team-2" {
			score2 = hs.Score
		}
	}
	if score1 <= score2 {
		t.Fatalf("expected team-1 score > team-2 score, got %f <= %f", score1, score2)
	}
}

func TestScoreAllWithConfig_ScoresActionHealth(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "action:scenario.status.show", Type: NodeAction},
			{ID: "action:missing.examples", Type: NodeAction},
		},
		Edges: []Edge{
			{From: "action:scenario.status.show", To: "cli:vrooli", Kind: EdgeActionCommand, Category: CodeScenarioCLI, Command: "vrooli"},
			{From: "action:missing.examples", To: "cli:vrooli", Kind: EdgeActionCommand, Category: CodeScenarioCLI, Command: "vrooli"},
		},
		NodeMetrics: map[string]NodeMetricSet{
			"action:scenario.status.show": {
				metricActionContractValid:   1,
				metricActionCommandDeclared: 1,
				metricActionExamples:        1,
				metricActionOwnerDeclared:   1,
			},
			"action:missing.examples": {
				metricActionContractValid:   1,
				metricActionCommandDeclared: 1,
				metricActionOwnerDeclared:   1,
			},
		},
	}
	cfg := DefaultHealthConfig()
	cfg.Action = HealthWeights{
		ActionContract: 1,
		ActionCommand:  1,
		ActionExamples: 1,
		ActionOwner:    1,
	}

	scores := ScoreAllWithConfig(g, cfg)
	byID := map[string]HealthScore{}
	for _, hs := range scores {
		byID[hs.NodeID] = hs
	}

	if byID["action:scenario.status.show"].Score != 1 {
		t.Fatalf("expected complete Action score 1, got %+v", byID["action:scenario.status.show"])
	}
	missing := byID["action:missing.examples"]
	if missing.Score >= 1 {
		t.Fatalf("expected missing examples to reduce score, got %+v", missing)
	}
	foundExamplesMessage := false
	for _, msg := range missing.Messages {
		if msg.Key == "action.examples.missing" {
			foundExamplesMessage = true
			break
		}
	}
	if !foundExamplesMessage {
		t.Fatalf("expected missing examples diagnostic, got %+v", missing.Messages)
	}
}

func TestApplyCLIHealthPolicy_SeesActionCommandAsScenarioCLI(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "cli:prompt-manager", Type: NodeCLI},
		},
		Edges: []Edge{
			{From: "action:team.swarm.work.list", To: "cli:prompt-manager", Kind: EdgeActionCommand, Category: CodeScenarioCLI, Command: "prompt-manager"},
		},
	}
	got := ApplyCLIHealthPolicy(context.Background(), g, nil, &fakeScenarioProvider{
		scoreByScenario: map[string]float64{"prompt-manager": 0.8},
	})
	if len(got) != 1 {
		t.Fatalf("expected one score, got %d", len(got))
	}
	if got[0].Score != 0.8 {
		t.Fatalf("expected scenario score from action-command edge, got %+v", got[0])
	}
}

func TestApplyCLIHealthPolicyWithConfig_UsesConfiguredExternalScore(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "cli:grep", Type: NodeCLI},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:grep", Kind: EdgeCodeUsage, Category: CodeExternalTool, Command: "grep"},
		},
	}
	cfg := DefaultHealthConfig().CLI
	cfg.ExternalToolScore = 0.25

	got := ApplyCLIHealthPolicyWithConfig(context.Background(), g, nil, nil, cfg)
	if len(got) != 1 {
		t.Fatalf("expected one score, got %d", len(got))
	}
	if got[0].Score != 0.25 {
		t.Fatalf("expected configured score 0.25, got %f", got[0].Score)
	}
}

func TestScoreWithWeights_SkipsMessagesForZeroWeightFactors(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "team-1", Type: NodeTeam},
			{ID: "skill-1", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "team-1", To: "skill-1", Kind: EdgeMembership},
		},
	}
	weights := HealthWeights{
		OutgoingEdges: 1.0,
		IncomingEdges: 0.0,
	}

	_, _, messages := scoreWithWeights("team-1", g, weights)
	for _, msg := range messages {
		if msg.Factor == "incoming-edges" {
			t.Fatalf("did not expect incoming-edges recommendation when weight is zero")
		}
	}
}

func TestScoreWithWeights_OrdersMessagesByWeightedImpact(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "team-1", Type: NodeTeam},
		},
	}
	weights := HealthWeights{
		OutgoingEdges: 2.0,
		IncomingEdges: 0.2,
	}

	_, _, messages := scoreWithWeights("team-1", g, weights)
	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(messages))
	}
	if messages[0].Factor != "outgoing-edges" {
		t.Fatalf("expected highest-impact recommendation first, got %q", messages[0].Factor)
	}
}

// ---------------------------------------------------------------------------
// Regression tests: edge count pre-computation (O(N+E) vs O(N*E))
// ---------------------------------------------------------------------------

// TestBuildEdgeCounts verifies the pre-computed edge count index
// produces the same results as the linear-scan approach.
func TestBuildEdgeCounts(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "a", Type: NodeAgent},
			{ID: "b", Type: NodeSkill},
			{ID: "c", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "a", To: "b", Kind: EdgeCLIRead},
			{From: "a", To: "c", Kind: EdgeBoldListed},
			{From: "b", To: "c", Kind: EdgeDefaultScope},
		},
	}

	ec := buildEdgeCounts(g)

	// Outgoing: a=2, b=1, c=0
	if ec.outgoing["a"] != 2 {
		t.Errorf("expected outgoing[a]=2, got %d", ec.outgoing["a"])
	}
	if ec.outgoing["b"] != 1 {
		t.Errorf("expected outgoing[b]=1, got %d", ec.outgoing["b"])
	}
	if ec.outgoing["c"] != 0 {
		t.Errorf("expected outgoing[c]=0, got %d", ec.outgoing["c"])
	}

	// Incoming: a=0, b=1, c=2
	if ec.incoming["a"] != 0 {
		t.Errorf("expected incoming[a]=0, got %d", ec.incoming["a"])
	}
	if ec.incoming["b"] != 1 {
		t.Errorf("expected incoming[b]=1, got %d", ec.incoming["b"])
	}
	if ec.incoming["c"] != 2 {
		t.Errorf("expected incoming[c]=2, got %d", ec.incoming["c"])
	}
}

// TestScoreAllWithConfig_MatchesScoreAll_EdgeCounts verifies that the
// pre-computed edge count path produces the same edge factor scores as
// the linear-scan fallback.
func TestScoreAllWithConfig_MatchesScoreAll_EdgeCounts(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "a", Type: NodeAgent},
			{ID: "b", Type: NodeSkill},
			{ID: "c", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "a", To: "b", Kind: EdgeCLIRead},
			{From: "a", To: "c", Kind: EdgeBoldListed},
			{From: "b", To: "c", Kind: EdgeDefaultScope},
			{From: "c", To: "a", Kind: EdgePathRef},
		},
	}

	cfg := DefaultHealthConfig()
	configScores := ScoreAllWithConfig(g, cfg)
	legacyScores := ScoreAll(g, DefaultScoreFns())

	// Both should have scores for all nodes.
	if len(configScores) != len(g.Nodes) {
		t.Fatalf("ScoreAllWithConfig returned %d scores, expected %d", len(configScores), len(g.Nodes))
	}
	if len(legacyScores) != len(g.Nodes) {
		t.Fatalf("ScoreAll returned %d scores, expected %d", len(legacyScores), len(g.Nodes))
	}

	// Build maps for comparison.
	configByID := make(map[string]HealthScore)
	for _, hs := range configScores {
		configByID[hs.NodeID] = hs
	}
	legacyByID := make(map[string]HealthScore)
	for _, hs := range legacyScores {
		legacyByID[hs.NodeID] = hs
	}

	// Edge count factors should match between the two approaches.
	for _, n := range g.Nodes {
		chs := configByID[n.ID]
		lhs := legacyByID[n.ID]

		cOut := chs.Factors["outgoing-edges"]
		lOut := lhs.Factors["outgoing-edges"]
		if cOut != lOut {
			t.Errorf("node %s: outgoing-edges mismatch: config=%f legacy=%f", n.ID, cOut, lOut)
		}

		cIn := chs.Factors["incoming-edges"]
		lIn := lhs.Factors["incoming-edges"]
		if cIn != lIn {
			t.Errorf("node %s: incoming-edges mismatch: config=%f legacy=%f", n.ID, cIn, lIn)
		}
	}
}

// BenchmarkScoreAllWithConfig_LargeGraph measures scoring performance
// for a large graph (200 nodes, 2000 edges) to verify the O(N+E)
// optimization is effective.
func BenchmarkScoreAllWithConfig_LargeGraph(b *testing.B) {
	const numNodes = 200
	const numEdges = 2000

	nodes := make([]Node, numNodes)
	for i := 0; i < numNodes; i++ {
		nodeType := NodeSkill
		if i%3 == 0 {
			nodeType = NodeAgent
		} else if i%5 == 0 {
			nodeType = NodeTeam
		}
		nodes[i] = Node{
			ID:   fmt.Sprintf("node-%d", i),
			Type: nodeType,
		}
	}

	edges := make([]Edge, numEdges)
	for i := 0; i < numEdges; i++ {
		edges[i] = Edge{
			From: fmt.Sprintf("node-%d", i%numNodes),
			To:   fmt.Sprintf("node-%d", (i*7+3)%numNodes),
			Kind: EdgeCLIRead,
		}
	}

	g := Graph{Nodes: nodes, Edges: edges}
	cfg := DefaultHealthConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScoreAllWithConfig(g, cfg)
	}
}
