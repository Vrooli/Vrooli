package proposals

import (
	"strings"
	"testing"
)

// goalStateFixture builds a goal with one existing milestone that owns one
// item, plus a second closure item that is not yet assigned.
func goalStateFixture() GoalState {
	return GoalState{
		Name:    "web-console-production-ready",
		Version: "2026-07-25T00:00:00Z",
		Milestones: map[string]GoalMilestoneState{
			"suite-green": {Items: map[string]struct{}{"execute/fix-unit-phase": {}}},
		},
		Closure: map[string]struct{}{
			"execute/fix-unit-phase":      {},
			"execute/fix-standards-phase": {},
		},
		Targets: map[string]struct{}{"execute/fix-unit-phase": {}},
	}
}

func goalProposal(state GoalState, mutations ...Mutation) Proposal {
	return Proposal{Form: FormMutationList, BaseVersion: state.Version, Mutations: mutations}
}

func TestValidateGoalRejectsMilestoneWithoutAcceptanceCriteria(t *testing.T) {
	state := goalStateFixture()
	for _, tc := range []struct {
		name     string
		criteria []string
	}{
		{name: "omitted", criteria: nil},
		{name: "empty list", criteria: []string{}},
		{name: "blank entries", criteria: []string{"  ", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := goalProposal(state, Mutation{
				ID: "m1", Op: OpCreateMilestone,
				GoalMilestone: &GoalMilestone{Name: "desktop-verified", Title: "Desktop verified", AcceptanceCriteria: tc.criteria},
			})
			err := ValidateGoal(p, state)
			if err == nil {
				t.Fatal("expected create_milestone without acceptance criteria to be rejected")
			}
			if !strings.Contains(err.Error(), "acceptance_criteria") {
				t.Fatalf("error should name the missing field, got: %v", err)
			}
		})
	}
}

func TestValidateGoalAcceptsMilestoneWithAcceptanceCriteria(t *testing.T) {
	state := goalStateFixture()
	p := goalProposal(state, Mutation{
		ID: "m1", Op: OpCreateMilestone,
		GoalMilestone: &GoalMilestone{
			Name: "desktop-verified", Title: "Desktop verified",
			AcceptanceCriteria: []string{"Given a signed build, when it installs on a real device, then the console launches."},
		},
	})
	if err := ValidateGoal(p, state); err != nil {
		t.Fatalf("expected a milestone with criteria to validate, got: %v", err)
	}
}

// An update replaces the milestone definition wholesale, so omitting criteria
// would erase the definition of done rather than leave it untouched.
func TestValidateGoalRejectsUpdateThatErasesAcceptanceCriteria(t *testing.T) {
	state := goalStateFixture()
	p := goalProposal(state, Mutation{
		ID: "m1", Op: OpUpdateMilestone,
		GoalMilestone: &GoalMilestone{Name: "suite-green", Title: "Renamed only"},
	})
	err := ValidateGoal(p, state)
	if err == nil {
		t.Fatal("expected an update that drops acceptance criteria to be rejected")
	}
	if !strings.Contains(err.Error(), "acceptance_criteria") {
		t.Fatalf("error should name the missing field, got: %v", err)
	}
}

// A goal's structure and its membership belong in one operator decision:
// splitting them leaves an approved milestone with no members. Validation
// therefore reads its own list, so create-then-populate is a single envelope.
func TestValidateGoalAllowsCreateThenPopulateInOneEnvelope(t *testing.T) {
	state := goalStateFixture()
	p := goalProposal(state,
		Mutation{
			ID: "m1", Op: OpCreateMilestone,
			GoalMilestone: &GoalMilestone{
				Name: "desktop-verified", Title: "Desktop verified",
				AcceptanceCriteria: []string{"Given a signed build, when it installs on a real device, then the console launches."},
			},
		},
		Mutation{
			ID: "m2", Op: OpAssignMilestoneItems,
			MilestoneName: "desktop-verified",
			Items:         []string{"execute/fix-standards-phase"},
		},
	)
	if err := ValidateGoal(p, state); err != nil {
		t.Fatalf("create-then-populate should validate in one envelope, got: %v", err)
	}
}

// A target added earlier in the same list is in scope: the closure is derived
// from the target set, so assignment must see the list's own additions.
func TestValidateGoalAllowsAssigningATargetAddedInTheSameEnvelope(t *testing.T) {
	state := goalStateFixture()
	p := goalProposal(state,
		Mutation{ID: "m1", Op: OpAddGoalTarget, Targets: []string{"execute/new-work"}},
		Mutation{ID: "m2", Op: OpAssignMilestoneItems, MilestoneName: "suite-green", Items: []string{"execute/new-work"}},
	)
	if err := ValidateGoal(p, state); err != nil {
		t.Fatalf("assigning a target added in the same envelope should validate, got: %v", err)
	}
}

func TestValidateGoalStillRejectsUnknownMilestoneAndOutOfScopeItem(t *testing.T) {
	state := goalStateFixture()
	unknown := goalProposal(state, Mutation{
		ID: "m1", Op: OpAssignMilestoneItems, MilestoneName: "never-created", Items: []string{"execute/fix-unit-phase"},
	})
	if err := ValidateGoal(unknown, state); err == nil {
		t.Fatal("expected assignment to an uncreated milestone to be rejected")
	}
	outOfScope := goalProposal(state, Mutation{
		ID: "m1", Op: OpAssignMilestoneItems, MilestoneName: "suite-green", Items: []string{"execute/unrelated"},
	})
	if err := ValidateGoal(outOfScope, state); err == nil {
		t.Fatal("expected assignment of an out-of-closure item to be rejected")
	}
}

// A milestone archived earlier in the list no longer exists for later ops.
func TestValidateGoalRejectsAssignmentToAMilestoneArchivedEarlierInTheList(t *testing.T) {
	state := goalStateFixture()
	p := goalProposal(state,
		Mutation{ID: "m1", Op: OpArchiveMilestone, MilestoneName: "suite-green"},
		Mutation{ID: "m2", Op: OpAssignMilestoneItems, MilestoneName: "suite-green", Items: []string{"execute/fix-unit-phase"}},
	)
	if err := ValidateGoal(p, state); err == nil {
		t.Fatal("expected assignment to a milestone archived earlier in the same list to be rejected")
	}
}

// Goal planning exists to find work that has no covering item yet, so a goal
// proposal may create items and place them in one operator decision.
func TestValidateGoalAllowsCreatingAndPlacingWorkInOneEnvelope(t *testing.T) {
	state := goalStateFixture()
	p := goalProposal(state,
		Mutation{
			ID: "m1", Op: OpCreateMilestone,
			GoalMilestone: &GoalMilestone{
				Name: "desktop-verified", Title: "Desktop verified",
				AcceptanceCriteria: []string{"Given a signed build, when it installs on a real device, then the console launches."},
			},
		},
		Mutation{ID: "m2", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "sign-the-build", Title: "Sign the build"}},
		Mutation{ID: "m3", Op: OpAddGoalTarget, Targets: []string{"execute/sign-the-build"}},
		Mutation{ID: "m4", Op: OpAssignMilestoneItems, MilestoneName: "desktop-verified", Items: []string{"execute/sign-the-build"}},
	)
	if err := ValidateGoal(p, state); err != nil {
		t.Fatalf("create milestone, create item, target it, assign it should validate as one envelope, got: %v", err)
	}
}

func TestValidateGoalRejectsDuplicateAndMalformedAddedItems(t *testing.T) {
	state := goalStateFixture()
	duplicate := goalProposal(state,
		Mutation{ID: "m1", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "fix-unit-phase", Title: "Already in closure"}},
	)
	if err := ValidateGoal(duplicate, state); err == nil {
		t.Fatal("expected an item already in the goal closure to be rejected")
	}
	missingSpec := goalProposal(state, Mutation{ID: "m1", Op: OpAddItem})
	if err := ValidateGoal(missingSpec, state); err == nil {
		t.Fatal("expected add_item without an item spec to be rejected")
	}
}

// Item ops other than creation stay with the item path, where the item's own
// context is hydrated. A goal proposal must not silently retarget, archive, or
// re-status work it is not showing the operator.
func TestValidateGoalStillRejectsOtherItemOperations(t *testing.T) {
	state := goalStateFixture()
	for _, m := range []Mutation{
		{ID: "m1", Op: OpArchiveItem, Target: "execute/fix-unit-phase"},
		{ID: "m1", Op: OpChangeStatus, Target: "execute/fix-unit-phase", Status: "ready"},
		{ID: "m1", Op: OpUpdateItem, Target: "execute/fix-unit-phase", Patch: &ItemPatch{}},
	} {
		if err := ValidateGoal(goalProposal(state, m), state); err == nil {
			t.Fatalf("op %q should not be valid for a goal proposal", m.Op)
		}
	}
}
