package graph

import (
	"strings"
	"testing"
)

func sampleOperatingMap() operatingMap {
	return operatingMap{
		Teams: []operatingMapTeam{
			{ID: "team-a", Label: "team-a", GoalLinkage: "primary: The Forge", Valid: true},
			{ID: "team-b", Label: "team-b", GoalLinkage: "supporting: Mission Control", Valid: false},
		},
		Topics: []operatingMapTopic{{ID: "friction-inbox/<scope>/<slug>", Label: "friction-inbox/<scope>/<slug>"}},
		Edges:  []operatingMapEdge{{From: "team-a", To: "friction-inbox/<scope>/<slug>"}},
	}
}

func TestRenderOperatingMap_IsStableAndIncludesRequiredArtifactSections(t *testing.T) {
	m := sampleOperatingMap()
	first, second := renderOperatingMap(m, nil), renderOperatingMap(m, nil)
	if first != second {
		t.Fatal("map rendering is not stable")
	}
	for _, want := range []string{"```mermaid", "Goal linkage", "primary: The Forge", "Contract validation"} {
		if !strings.Contains(first, want) {
			t.Errorf("rendered map missing %q:\n%s", want, first)
		}
	}
}

func TestRenderOperatingMapUsesReadableNodeIDs(t *testing.T) {
	rendered := renderOperatingMap(sampleOperatingMap(), nil)

	// MAP.md is read as text more often than it is rendered, so node ids must
	// be legible. The previous renderer hex-encoded topic ids.
	if !strings.Contains(rendered, "team_team_a") {
		t.Errorf("team node id not readable:\n%s", rendered)
	}
	if !strings.Contains(rendered, "topic_friction_inbox_scope_slug") {
		t.Errorf("topic node id not readable:\n%s", rendered)
	}
	if strings.Contains(rendered, "X_6") {
		t.Errorf("hex-encoded node id still present:\n%s", rendered)
	}
}

func TestRenderOperatingMapMarksInvalidTeamInDiagramAndTable(t *testing.T) {
	rendered := renderOperatingMap(sampleOperatingMap(), nil)

	// A failing contract must be visible in the picture, not only the table.
	if !strings.Contains(rendered, "style team_team_b stroke:#c0392b") {
		t.Errorf("invalid team not styled in diagram:\n%s", rendered)
	}
	if !strings.Contains(rendered, "**invalid**") {
		t.Errorf("invalid team not marked in table:\n%s", rendered)
	}
}

func TestRenderOperatingMapOverlaysMembersAtMemberDepth(t *testing.T) {
	members := []mapMember{
		{Team: "team-a", Agent: "scout"},
		{Team: "team-a", Agent: "curator"},
	}
	rendered := renderOperatingMap(sampleOperatingMap(), members)

	for _, want := range []string{"member_scout", "member_curator", "## Members", "| Team | Members |"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("member overlay missing %q:\n%s", want, rendered)
		}
	}
	if !strings.Contains(rendered, "team_team_a --- member_scout") {
		t.Errorf("membership edge missing:\n%s", rendered)
	}

	// Team depth must stay clean of member detail.
	if strings.Contains(renderOperatingMap(sampleOperatingMap(), nil), "## Members") {
		t.Error("team-depth render leaked the member section")
	}
}

func TestSlugifyMapIDCollapsesSeparatorsAndTrims(t *testing.T) {
	cases := map[string]string{
		"friction-inbox/<scope>/<slug>": "friction_inbox_scope_slug",
		"a//b":                          "a_b",
		"***":                           "node",
	}
	for in, want := range cases {
		if got := slugifyMapID(in); got != want {
			t.Errorf("slugifyMapID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapIDAllocatorAvoidsCollisions(t *testing.T) {
	alloc := newMapIDAllocator()
	// Two distinct topics that slugify identically must not share a node id,
	// or their edges would merge in the diagram.
	first := alloc.id("topic", "a/b")
	second := alloc.id("topic", "a-b")
	if first == second {
		t.Fatalf("colliding ids: %q", first)
	}
	if again := alloc.id("topic", "a/b"); again != first {
		t.Fatalf("id not stable across calls: %q vs %q", again, first)
	}
}
