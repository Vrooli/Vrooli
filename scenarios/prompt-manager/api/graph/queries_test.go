package graph

import (
	"testing"
)

func makeTestGraph() Graph {
	return Graph{
		Nodes: []Node{
			{ID: "team-alpha", Type: NodeTeam, Label: "Alpha Team"},
			{ID: "team-beta", Type: NodeTeam, Label: "Beta Team"},
			{ID: "agent-a", Type: NodeAgent, Label: "Agent A"},
			{ID: "agent-b", Type: NodeAgent, Label: "Agent B"},
			{ID: "agent-c", Type: NodeAgent, Label: "Agent C"},
			{ID: "skill-x", Type: NodeSkill, Label: "Skill X"},
			{ID: "skill-y", Type: NodeSkill, Label: "Skill Y"},
			{ID: "skill-z", Type: NodeSkill, Label: "Skill Z"},
		},
		Edges: []Edge{
			// team-alpha has agent-a as member
			{From: "team-alpha", To: "agent-a", Kind: EdgeMembership},
			// agent-a references skill-x
			{From: "agent-a", To: "skill-x", Kind: EdgeCLIRead},
			// skill-x references skill-y (cross-reference)
			{From: "skill-x", To: "skill-y", Kind: EdgeDefaultScope},
		},
	}
}

func TestOrphanedSkills(t *testing.T) {
	g := makeTestGraph()
	orphans := OrphanedSkills(g)

	// skill-z is orphaned (no incoming edges)
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphaned skill, got %d", len(orphans))
	}
	if orphans[0].ID != "skill-z" {
		t.Errorf("expected skill-z, got %s", orphans[0].ID)
	}
}

func TestSkilllessAgents(t *testing.T) {
	g := makeTestGraph()
	agents := SkilllessAgents(g)

	// agent-b and agent-c have no skill edges
	if len(agents) != 2 {
		t.Fatalf("expected 2 skillless agents, got %d", len(agents))
	}
	ids := map[string]bool{}
	for _, a := range agents {
		ids[a.ID] = true
	}
	if !ids["agent-b"] || !ids["agent-c"] {
		t.Errorf("expected agent-b and agent-c, got %v", ids)
	}
}

func TestEmptyTeams(t *testing.T) {
	g := makeTestGraph()
	teams := EmptyTeams(g)

	// team-beta has no members
	if len(teams) != 1 {
		t.Fatalf("expected 1 empty team, got %d", len(teams))
	}
	if teams[0].ID != "team-beta" {
		t.Errorf("expected team-beta, got %s", teams[0].ID)
	}
}

func TestUnaffiliatedAgents(t *testing.T) {
	g := makeTestGraph()
	agents := UnaffiliatedAgents(g)

	// agent-b and agent-c are not members of any team
	if len(agents) != 2 {
		t.Fatalf("expected 2 unaffiliated agents, got %d", len(agents))
	}
}

func TestCLIlessSkills(t *testing.T) {
	g := makeTestGraph()
	skills := CLIlessSkills(g)

	// All 3 skills have no code-usage edges
	if len(skills) != 3 {
		t.Fatalf("expected 3 CLIless skills, got %d", len(skills))
	}
}

func TestPopular(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "a", Type: NodeSkill},
			{ID: "b", Type: NodeSkill},
			{ID: "c", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "x", To: "a", Kind: EdgeCLIRead},
			{From: "y", To: "a", Kind: EdgeBoldListed},
			{From: "z", To: "a", Kind: EdgePathRef},
			{From: "x", To: "b", Kind: EdgeCLIRead},
		},
	}

	popular := Popular(g, 2)
	if len(popular) != 2 {
		t.Fatalf("expected 2 popular nodes, got %d", len(popular))
	}
	if popular[0].ID != "a" {
		t.Errorf("expected most popular to be 'a', got %s", popular[0].ID)
	}
}

func TestDetectCircularRefs_NoCycles(t *testing.T) {
	g := makeTestGraph()
	cycles := DetectCircularRefs(g)

	if len(cycles) != 0 {
		t.Fatalf("expected 0 cycles, got %d", len(cycles))
	}
}

func TestDetectCircularRefs_WithCycle(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "skill-a", Type: NodeSkill},
			{ID: "skill-b", Type: NodeSkill},
			{ID: "skill-c", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "skill-a", To: "skill-b", Kind: EdgeDefaultScope},
			{From: "skill-b", To: "skill-c", Kind: EdgeCLIRead},
			{From: "skill-c", To: "skill-a", Kind: EdgeBoldListed},
		},
	}

	cycles := DetectCircularRefs(g)
	if len(cycles) == 0 {
		t.Fatal("expected at least 1 cycle, got 0")
	}
}

// ---------------------------------------------------------------------------
// CLIlessSkills with Category awareness
// ---------------------------------------------------------------------------

func TestCLIlessSkills_IgnoresExternalEdges(t *testing.T) {
	// A skill with only external tool edges should still be "CLI-less"
	// because it doesn't use Vrooli CLIs.
	g := Graph{
		Nodes: []Node{
			{ID: "skill-a", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:grep", Kind: EdgeCodeUsage, Category: CodeExternalTool},
		},
	}
	skills := CLIlessSkills(g)
	if len(skills) != 1 {
		t.Fatalf("expected 1 CLIless skill, got %d", len(skills))
	}
	if skills[0].ID != "skill-a" {
		t.Errorf("expected skill-a, got %s", skills[0].ID)
	}
}

func TestCLIlessSkills_ExcludesVrooliSkills(t *testing.T) {
	// A skill with a Vrooli CLI edge should NOT be "CLI-less".
	g := Graph{
		Nodes: []Node{
			{ID: "skill-a", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:vrooli", Kind: EdgeCodeUsage, Category: CodeScenarioCLI},
		},
	}
	skills := CLIlessSkills(g)
	if len(skills) != 0 {
		t.Fatalf("expected 0 CLIless skills, got %d", len(skills))
	}
}

// ---------------------------------------------------------------------------
// ExternalToolSkills
// ---------------------------------------------------------------------------

func TestExternalToolSkills(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "skill-a", Type: NodeSkill},
			{ID: "skill-b", Type: NodeSkill},
			{ID: "skill-c", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:grep", Kind: EdgeCodeUsage, Category: CodeExternalTool},
			{From: "skill-b", To: "cli:vrooli", Kind: EdgeCodeUsage, Category: CodeScenarioCLI},
			{From: "skill-c", To: "cli:deploy.sh", Kind: EdgeCodeUsage, Category: CodeScript},
		},
	}
	skills := ExternalToolSkills(g)
	if len(skills) != 2 {
		t.Fatalf("expected 2 external tool skills, got %d", len(skills))
	}
	ids := map[string]bool{}
	for _, s := range skills {
		ids[s.ID] = true
	}
	if !ids["skill-a"] {
		t.Error("expected skill-a (external tool)")
	}
	if !ids["skill-c"] {
		t.Error("expected skill-c (script)")
	}
	if ids["skill-b"] {
		t.Error("skill-b should not appear (Vrooli CLI only)")
	}
}

func TestExternalToolSkills_Empty(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "skill-a", Type: NodeSkill},
		},
		Edges: []Edge{
			{From: "skill-a", To: "cli:vrooli", Kind: EdgeCodeUsage, Category: CodeScenarioCLI},
		},
	}
	skills := ExternalToolSkills(g)
	if len(skills) != 0 {
		t.Fatalf("expected 0 external tool skills, got %d", len(skills))
	}
}
