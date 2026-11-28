//go:build liverepo

package memberflow

import "testing"

// TestLiveObjectiveJoinIsIntact runs the real objective parse against the real
// objective table and the real roster. Retirement of the memberflow objective
// rule family moved current-state validation to the objective authority
// (internal/objectives); this canary keeps the parse that the one-way import
// depends on from silently breaking.
//
// It runs under -tags liverepo because it reads the checked-in store rather
// than a fixture, matching the other repository-conformance canaries here.
func TestLiveObjectiveJoinIsIntact(t *testing.T) {
	storeDir, repoRoot := realPromptManagerStore(t)

	registry, err := LoadObjectives(repoRoot)
	if err != nil {
		t.Fatalf("LoadObjectives: %v", err)
	}
	if len(registry.Objectives) == 0 {
		t.Fatalf("no objectives parsed from %s; the table shape changed", ObjectivesDocPath)
	}
	declared, _, err := LoadTeamObjectives(storeDir)
	if err != nil {
		t.Fatalf("LoadTeamObjectives: %v", err)
	}

	// Every team in the store must trace to at least one objective. This is the
	// upward half of the coverage rule and the direction that catches effort
	// nobody asked for.
	for teamID, decls := range declared {
		if len(decls) == 0 {
			t.Errorf("team %s declares no objectivesServed", teamID)
		}
	}
}
