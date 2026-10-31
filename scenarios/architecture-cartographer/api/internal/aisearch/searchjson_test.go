package aisearch

import (
	"path/filepath"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
	repocontract "github.com/vrooli/repo-contract-go"
)

// TestCommittedSearchJSON guards the scenario-owned descriptor SSOT: it must
// parse strictly (unknown keys rejected), validate, expose exactly the
// domain-map provider, and carry a well-formed tuning + tests block. This is the
// build-time twin of what the boot self-registration and the search-hub sweep
// load at runtime.
func TestCommittedSearchJSON(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repoRoot, "scenarios", "architecture-cartographer", ".vrooli", "search.json")

	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("LoadSearchFile: %v", err)
	}
	if err := file.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	const providerID = "architecture-cartographer.domain-map"
	provider, ok := file.Provider(providerID)
	if !ok {
		t.Fatalf("provider %q not found", providerID)
	}

	tuning := provider.ResolvedTuning()
	if err := tuning.Validate(); err != nil {
		t.Fatalf("tuning.Validate: %v", err)
	}
	if tuning.Engine != "dense" {
		t.Errorf("engine = %q, want dense (small corpus default)", tuning.Engine)
	}

	suite := provider.Tests
	if err := suite.Validate(); err != nil {
		t.Fatalf("suite.Validate: %v", err)
	}

	// The corpus must include the headline term-agnostic case and at least one
	// gibberish negative (the junk-rejection guard).
	var headline, negatives int
	for _, c := range suite.Cases {
		if c.ID == "authoring-in-plan-manager" {
			headline++
			if len(c.ExpectIDs) != 1 || c.ExpectIDs[0] != "plan-manager/authoring" {
				t.Errorf("headline case expect_ids = %v, want [plan-manager/authoring]", c.ExpectIDs)
			}
		}
		if c.ExpectNoStrongHit {
			negatives++
		}
	}
	if headline != 1 {
		t.Errorf("expected exactly one headline case, got %d", headline)
	}
	if negatives < 1 {
		t.Errorf("expected at least one gibberish negative, got %d", negatives)
	}
}
