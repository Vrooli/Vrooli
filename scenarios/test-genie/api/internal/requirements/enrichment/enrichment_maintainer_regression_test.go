package enrichment

import (
	"context"
	"testing"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// Auto-sync opt-outs must cover the hierarchy pass too. Otherwise Enrich can
// still write a synthetic not-run status into requirements whose module opted
// out of automatic enrichment.
func TestEnricherDoesNotMutateAutoSyncDisabledModule(t *testing.T) {
	disabled := false
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "requirements/manual.json",
		Metadata: types.ModuleMetadata{
			AutoSyncEnabled: &disabled,
		},
		Requirements: []types.Requirement{{
			ID:     "REQ-MANUAL-ONLY",
			Status: types.StatusInProgress,
		}},
	}
	if err := index.AddModule(module); err != nil {
		t.Fatal(err)
	}

	if err := New().Enrich(context.Background(), index, types.NewEvidenceBundle()); err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}

	requirement := index.GetRequirement("REQ-MANUAL-ONLY")
	if requirement.LiveStatus != "" {
		t.Errorf("disabled module was mutated by hierarchy rollup: LiveStatus = %q, want empty", requirement.LiveStatus)
	}
	if requirement.AggregatedStatus.LiveRollup != "" || requirement.AggregatedStatus.DeclaredRollup != "" {
		t.Errorf("disabled module aggregate was mutated by hierarchy rollup: %+v", requirement.AggregatedStatus)
	}
}

// A directory-qualified validation reference must not match a different test
// file just because both files share a basename.
func TestMatcherDoesNotMatchDifferentDirectoryByBasename(t *testing.T) {
	evidence := types.NewEvidenceBundle()
	evidence.PhaseResults.Add(types.EvidenceRecord{
		RequirementID: "REQ-AUTH",
		Status:        types.LivePassed,
		Phase:         "unit",
		SourcePath:    "internal/admin/auth_test.go",
	})

	validation := &types.Validation{
		Type: types.ValTypeTest,
		Ref:  "internal/public/auth_test.go",
	}
	if match := NewMatcher().FindBestMatch(validation, "REQ-AUTH", evidence); match != nil {
		t.Fatalf("validation matched evidence from a different directory: %+v", match)
	}
}

// A disabled child must not contribute synthetic unknown/pending values to an
// enabled parent's rollup.
func TestEnricherExcludesDisabledChildFromEnabledParentRollup(t *testing.T) {
	disabled := false
	index := parsing.NewModuleIndex()
	parentModule := &types.RequirementModule{
		FilePath: "requirements/parent.json",
		Requirements: []types.Requirement{{
			ID:       "REQ-PARENT",
			Status:   types.StatusInProgress,
			Children: []string{"REQ-DISABLED-CHILD"},
		}},
	}
	childModule := &types.RequirementModule{
		FilePath: "requirements/child.json",
		Metadata: types.ModuleMetadata{AutoSyncEnabled: &disabled},
		Requirements: []types.Requirement{{
			ID:         "REQ-DISABLED-CHILD",
			Status:     types.StatusComplete,
			LiveStatus: types.LivePassed,
		}},
	}
	if err := index.AddModule(parentModule); err != nil {
		t.Fatal(err)
	}
	if err := index.AddModule(childModule); err != nil {
		t.Fatal(err)
	}
	index.BuildHierarchy()

	if err := New().Enrich(context.Background(), index, types.NewEvidenceBundle()); err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}

	parent := index.GetRequirement("REQ-PARENT")
	if parent.LiveStatus != types.LiveNotRun {
		t.Errorf("disabled child affected enabled parent live rollup: got %q, want %q", parent.LiveStatus, types.LiveNotRun)
	}
	if parent.AggregatedStatus.DeclaredRollup != types.StatusInProgress {
		t.Errorf("disabled child affected enabled parent declared rollup: got %q, want %q", parent.AggregatedStatus.DeclaredRollup, types.StatusInProgress)
	}
	child := index.GetRequirement("REQ-DISABLED-CHILD")
	if child.LiveStatus != types.LivePassed || child.AggregatedStatus.LiveRollup != "" {
		t.Errorf("disabled child was changed or used as rollup evidence: %+v", child)
	}
}

func TestEnricherHonorsIndexAutoSyncOptOut(t *testing.T) {
	disabled := false
	index := parsing.NewModuleIndex()
	root := &types.RequirementModule{
		FilePath:     "requirements/index.json",
		RelativePath: "index.json",
		IsIndex:      true,
		Metadata:     types.ModuleMetadata{AutoSyncEnabled: &disabled},
	}
	module := &types.RequirementModule{
		FilePath: "requirements/module.json",
		Requirements: []types.Requirement{{
			ID:     "REQ-INDEX-OPT-OUT",
			Status: types.StatusInProgress,
		}},
	}
	if err := index.AddModule(root); err != nil {
		t.Fatal(err)
	}
	if err := index.AddModule(module); err != nil {
		t.Fatal(err)
	}

	if err := New().Enrich(context.Background(), index, types.NewEvidenceBundle()); err != nil {
		t.Fatalf("Enrich returned error: %v", err)
	}

	requirement := index.GetRequirement("REQ-INDEX-OPT-OUT")
	if requirement.LiveStatus != "" || requirement.AggregatedStatus != (types.AggregatedStatus{}) {
		t.Errorf("index opt-out did not preserve module state: %+v", requirement)
	}
}
