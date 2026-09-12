//go:build liverepo

package memberflow

import "testing"

// TestLiveObjectiveJoinIsIntact runs the real coverage rule against the real
// objective table and the real roster. It is the check that makes recommendation
// 1 load-bearing: before this existed, editing OBJECTIVES.md fired nothing, and
// a team could be renamed, retired, or repointed without any surface noticing.
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
	declared, paths, err := LoadTeamObjectives(storeDir)
	if err != nil {
		t.Fatalf("LoadTeamObjectives: %v", err)
	}
	models, err := LoadOperatingModelDocuments(repoRoot)
	if err != nil {
		t.Fatalf("LoadOperatingModelDocuments: %v", err)
	}

	result := ValidateObjectives(ObjectiveValidationInput{
		Registry:        registry,
		Declared:        declared,
		TeamSourcePaths: paths,
		Models:          models,
	})
	for _, f := range result.Findings {
		if f.Severity == SeverityError {
			t.Errorf("objective join error: [%s] team=%s objective=%s: %s", f.Rule, f.Team, f.NodeID, f.Detail)
		}
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
