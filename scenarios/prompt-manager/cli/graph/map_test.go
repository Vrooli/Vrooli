package graph

import (
	"strings"
	"testing"
)

func TestRenderOperatingMap_IsStableAndIncludesRequiredArtifactSections(t *testing.T) {
	m := operatingMap{
		Teams:  []operatingMapTeam{{ID: "team-a", Label: "team-a", GoalLinkage: "primary: The Forge", Valid: true}},
		Topics: []operatingMapTopic{{ID: "a/output", Label: "a/output"}},
		Edges:  []operatingMapEdge{{From: "team-a", To: "a/output"}},
	}
	first, second := renderOperatingMap(m), renderOperatingMap(m)
	if first != second {
		t.Fatal("map rendering is not stable")
	}
	for _, want := range []string{"```mermaid", "T_team_a", "X_", "Goal linkage", "primary: The Forge", "Contract validation"} {
		if !strings.Contains(first, want) {
			t.Errorf("rendered map missing %q:\n%s", want, first)
		}
	}
}
