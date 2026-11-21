package graph

import (
	"context"
	"errors"
	"strings"
	"testing"

	"prompt-manager/internal/store"
)

// ---------------------------------------------------------------------------
// Build tests
// ---------------------------------------------------------------------------

func TestBuild_Empty(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{},
		nil,
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(g.Edges))
	}
}

func TestBuild_NodesFromTeams(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{teams: []store.Team{
			{ID: "team-1", DisplayName: "Alpha Team", Mission: "Build things"},
		}},
		&mockSkillNodeSource{},
		&mockGraphScanner{},
		nil,
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	n := g.Nodes[0]
	if n.ID != "team-1" || n.Type != NodeTeam || n.Label != "Alpha Team" || n.Description != "Build things" {
		t.Errorf("unexpected node: %+v", n)
	}
}

func TestBuild_NodesFromAgents(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{agents: []store.Agent{
			{ID: "agent-1", DisplayName: "Bot", Description: "A bot", Status: "active", Tags: []string{"dev"}},
		}},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{},
		nil,
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	n := g.Nodes[0]
	if n.ID != "agent-1" || n.Type != NodeAgent || n.Label != "Bot" || n.Description != "A bot" || n.Status != "active" {
		t.Errorf("unexpected node: %+v", n)
	}
	if len(n.Tags) != 1 || n.Tags[0] != "dev" {
		t.Errorf("unexpected tags: %v", n.Tags)
	}
}

func TestBuild_NodesFromSkills(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{skills: []store.Skill{
			{ID: "skill-1", Name: "Testing", Description: "Test skill", Status: "active", Tags: []string{"qa"}},
		}},
		&mockGraphScanner{},
		nil,
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	n := g.Nodes[0]
	if n.ID != "skill-1" || n.Type != NodeSkill || n.Label != "Testing" {
		t.Errorf("unexpected node: %+v", n)
	}
}

func TestBuild_NodesFromActionsUseNamespacedIDs(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{skills: []store.Skill{
			{ID: "scenario.status.show", Name: "Skill With Same ID"},
		}},
		&mockGraphScanner{},
		nil,
		&mockActionLister{actions: []store.Action{
			{
				ID:          "scenario.status.show",
				Name:        "Show Scenario Status",
				Description: "Show scenario status",
				Status:      "active",
				Tags:        []string{"scenario"},
				Owner:       store.ActionOwner{Type: "scenario", ID: "prompt-manager"},
				Command:     store.ActionCommand{Argv: []string{"vrooli", "scenario", "status", "{{scenario}}"}},
				Examples: []store.ActionExample{{
					Description: "Prompt manager",
					Input:       map[string]any{"scenario": "prompt-manager"},
				}},
			},
		}},
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodes := map[string]Node{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}
	if nodes["scenario.status.show"].Type != NodeSkill {
		t.Fatalf("expected raw ID to remain a skill node, got %+v", nodes["scenario.status.show"])
	}
	actionNode, ok := nodes["action:scenario.status.show"]
	if !ok {
		t.Fatalf("expected namespaced Action node, got nodes: %+v", nodes)
	}
	if actionNode.Type != NodeAction || actionNode.Label != "Show Scenario Status" {
		t.Fatalf("unexpected action node: %+v", actionNode)
	}

	var actionCommand *Edge
	for i := range g.Edges {
		if g.Edges[i].Kind == EdgeActionCommand {
			actionCommand = &g.Edges[i]
			break
		}
	}
	if actionCommand == nil {
		t.Fatalf("expected action-command edge, got %+v", g.Edges)
	}
	if actionCommand.From != "action:scenario.status.show" || actionCommand.To != "cli:vrooli" || actionCommand.Command != "vrooli" || actionCommand.Subcommand != "scenario" {
		t.Fatalf("unexpected action-command edge: %+v", actionCommand)
	}
	if nodes["cli:vrooli"].Type != NodeCLI {
		t.Fatalf("expected cli:vrooli node from action-command edge, got %+v", nodes["cli:vrooli"])
	}
}

func TestBuild_EdgesFromScanner(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{edges: []Edge{
			{From: "agent-1", To: "skill-1", Kind: EdgeCLIRead},
		}},
		nil,
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].From != "agent-1" || g.Edges[0].To != "skill-1" {
		t.Errorf("unexpected edge: %+v", g.Edges[0])
	}
}

func TestBuild_CLINodesExtracted(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{agents: []store.Agent{{ID: "agent-1"}}},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{edges: []Edge{
			{From: "agent-1", To: "cli:vrooli", Kind: EdgeCodeUsage},
		}},
		nil,
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have agent-1 node + cli:vrooli node
	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.Nodes))
	}
	var cliNode *Node
	for i := range g.Nodes {
		if g.Nodes[i].Type == NodeCLI {
			cliNode = &g.Nodes[i]
		}
	}
	if cliNode == nil {
		t.Fatal("expected CLI node")
	}
	if cliNode.ID != "cli:vrooli" || cliNode.Label != "vrooli" {
		t.Errorf("unexpected CLI node: %+v", cliNode)
	}
}

func TestBuild_HealthScores(t *testing.T) {
	scoreFn := ScoreFn{
		Name:   "test-score",
		Weight: 1.0,
		Fn:     func(nodeID string, g Graph) float64 { return 0.75 },
	}
	b := NewBuilder(
		&mockAgentNodeSource{agents: []store.Agent{{ID: "agent-1"}}},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{},
		[]ScoreFn{scoreFn},
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.HealthScores) != 1 {
		t.Fatalf("expected 1 health score, got %d", len(g.HealthScores))
	}
	if g.HealthScores[0].Score != 0.75 {
		t.Errorf("expected score 0.75, got %f", g.HealthScores[0].Score)
	}
}

func TestBuild_HealthScores_CLIOverridesApplied(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{agents: []store.Agent{{ID: "agent-1"}}},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{edges: []Edge{
			{From: "agent-1", To: "cli:grep", Kind: EdgeCodeUsage, Category: CodeExternalTool, Command: "grep"},
		}},
		DefaultScoreFns(),
	)
	b.SetScenarioHealthProvider(&fakeScenarioProvider{scoreByScenario: map[string]float64{}})

	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var grepScore *HealthScore
	for i := range g.HealthScores {
		if g.HealthScores[i].NodeID == "cli:grep" {
			grepScore = &g.HealthScores[i]
			break
		}
	}
	if grepScore == nil {
		t.Fatalf("expected score for cli:grep, got %+v", g.HealthScores)
	}
	if grepScore.Score != 0 {
		t.Fatalf("expected cli:grep score 0, got %f", grepScore.Score)
	}
}

func TestBuild_SkillCommandReferencesValidatedThroughCLIHealth(t *testing.T) {
	validator := &mockCommandReferenceValidator{results: map[string]CommandReferenceResult{
		"vrooli scenario tost cli-health": {
			Verdict:     "invalid",
			Issues:      []CommandIssue{{Code: "command_not_found", Message: "unknown command path"}},
			Suggestions: []string{"vrooli scenario test"},
			Guidance:    []string{"fix this to a current command"},
		},
	}}
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{skills: []store.Skill{{ID: "skill-1", Name: "Skill"}}},
		&mockGraphScanner{edges: []Edge{{
			From:        "skill-1",
			To:          "cli:vrooli",
			Kind:        EdgeCodeUsage,
			Category:    CodeScenarioCLI,
			Command:     "vrooli",
			Subcommand:  "scenario",
			CommandText: "vrooli scenario tost cli-health",
		}}},
		DefaultScoreFns(),
	)
	b.SetCommandReferenceValidator(validator)

	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(validator.calls) != 1 || validator.calls[0].CommandText != "vrooli scenario tost cli-health" {
		t.Fatalf("unexpected validator calls: %+v", validator.calls)
	}
	var skillScore *HealthScore
	for i := range g.HealthScores {
		if g.HealthScores[i].NodeID == "skill-1" {
			skillScore = &g.HealthScores[i]
			break
		}
	}
	if skillScore == nil {
		t.Fatalf("expected health score for skill-1: %+v", g.HealthScores)
	}
	if skillScore.Score > 0.2 {
		t.Fatalf("invalid command should cap skill health, got %f", skillScore.Score)
	}
	found := false
	for _, msg := range skillScore.Messages {
		if msg.Key == "skill.command.invalid" && msg.Severity == severityCritical && strings.Contains(msg.Recommendation, "vrooli scenario test") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing invalid command diagnostic: %+v", skillScore.Messages)
	}
}

func TestBuild_NoScoreFns(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{agents: []store.Agent{{ID: "agent-1"}}},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{},
		nil,
	)
	g, err := b.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.HealthScores != nil {
		t.Errorf("expected nil HealthScores, got %v", g.HealthScores)
	}
}

func TestBuild_TeamListError(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{err: errors.New("team fail")},
		&mockSkillNodeSource{},
		&mockGraphScanner{},
		nil,
	)
	_, err := b.Build(context.Background())
	if err == nil || err.Error() != "team fail" {
		t.Fatalf("expected team error, got: %v", err)
	}
}

func TestBuild_AgentListError(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{err: errors.New("agent fail")},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{},
		nil,
	)
	_, err := b.Build(context.Background())
	if err == nil || err.Error() != "agent fail" {
		t.Fatalf("expected agent error, got: %v", err)
	}
}

func TestBuild_SkillListError(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{err: errors.New("skill fail")},
		&mockGraphScanner{},
		nil,
	)
	_, err := b.Build(context.Background())
	if err == nil || err.Error() != "skill fail" {
		t.Fatalf("expected skill error, got: %v", err)
	}
}

func TestBuild_ScannerError(t *testing.T) {
	b := NewBuilder(
		&mockAgentNodeSource{},
		&mockTeamNodeSource{},
		&mockSkillNodeSource{},
		&mockGraphScanner{err: errors.New("scan fail")},
		nil,
	)
	_, err := b.Build(context.Background())
	if err == nil || err.Error() != "scan fail" {
		t.Fatalf("expected scanner error, got: %v", err)
	}
}

func TestExtractCLINodes(t *testing.T) {
	edges := []Edge{
		{From: "agent-a", To: "cli:vrooli", Kind: EdgeCodeUsage},
		{From: "agent-b", To: "cli:prompt-manager", Kind: EdgeCodeUsage},
		{From: "agent-a", To: "cli:vrooli", Kind: EdgeCodeUsage}, // duplicate
		{From: "agent-a", To: "skill-x", Kind: EdgeCLIRead},      // not code-usage
	}
	existing := []Node{
		{ID: "agent-a", Type: NodeAgent},
		{ID: "agent-b", Type: NodeAgent},
		{ID: "skill-x", Type: NodeSkill},
	}

	nodes := extractCLINodes(edges, existing)

	if len(nodes) != 2 {
		t.Fatalf("expected 2 CLI nodes, got %d", len(nodes))
	}

	ids := map[string]bool{}
	for _, n := range nodes {
		ids[n.ID] = true
		if n.Type != NodeCLI {
			t.Errorf("expected NodeCLI type, got %s", n.Type)
		}
	}
	if !ids["cli:vrooli"] || !ids["cli:prompt-manager"] {
		t.Errorf("expected cli:vrooli and cli:prompt-manager, got %v", ids)
	}
}

func TestExtractCLINodes_NoCodeUsage(t *testing.T) {
	edges := []Edge{
		{From: "agent-a", To: "skill-x", Kind: EdgeCLIRead},
	}
	nodes := extractCLINodes(edges, nil)
	if len(nodes) != 0 {
		t.Fatalf("expected 0 CLI nodes, got %d", len(nodes))
	}
}

func TestExtractCLINodes_ExistingNodeSkipped(t *testing.T) {
	edges := []Edge{
		{From: "agent-a", To: "cli:vrooli", Kind: EdgeCodeUsage},
	}
	existing := []Node{
		{ID: "cli:vrooli", Type: NodeCLI}, // already exists
	}

	nodes := extractCLINodes(edges, existing)
	if len(nodes) != 0 {
		t.Fatalf("expected 0 new CLI nodes (already exists), got %d", len(nodes))
	}
}
