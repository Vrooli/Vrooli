package proposals

import (
	"errors"
	"strings"
	"testing"
)

func baseState(t *testing.T) CurrentState {
	t.Helper()
	return CurrentState{
		InitiativeName: "ui-rewrite",
		Nodes: map[string]GraphNode{
			"execute/foo": {ID: "execute/foo", Kind: "execute", Name: "foo", Title: "Foo", Priority: 5},
			"execute/bar": {ID: "execute/bar", Kind: "execute", Name: "bar", Title: "Bar", Priority: 5},
		},
		Edges: []GraphEdge{
			{From: "execute/foo", To: "execute/bar"},
		},
		KnownInitiatives: map[string]struct{}{
			"ui-rewrite":    {},
			"other-project": {},
		},
		InProgressRefs: map[string]struct{}{},
	}
}

func intPtr(i int) *int { return &i }

func strPtr(s string) *string { return &s }

func TestValidate_RequiresMutationListForm(t *testing.T) {
	p := Proposal{Form: FormFullGraph}
	err := Validate(p, baseState(t))
	if err == nil || !strings.Contains(err.Error(), "Normalize") {
		t.Fatalf("expected guidance to Normalize first, got %v", err)
	}
}

func TestValidate_DetectsDuplicateIDs(t *testing.T) {
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpArchiveItem, Target: "execute/foo"},
			{ID: "m1", Op: OpArchiveItem, Target: "execute/bar"},
		},
	}
	err := Validate(p, baseState(t))
	if err == nil || !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestValidate_RejectsUnknownOp(t *testing.T) {
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: "rename_the_universe", Target: "execute/foo"},
		},
	}
	err := Validate(p, baseState(t))
	if err == nil || !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("expected ErrInvalidProposal, got %v", err)
	}
	if !strings.Contains(err.Error(), "unknown mutation op") {
		t.Fatalf("expected unknown-op message, got %v", err)
	}
}

func TestValidate_AddItem_RejectsDuplicate(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "foo", Title: "Clash"}},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestValidate_AddItem_AllowsStagedDependent(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "baz", Title: "Baz"}},
			{ID: "m2", Op: OpUpdateItem, Target: "execute/baz", Patch: &ItemPatch{Priority: intPtr(3)}},
		},
	}
	if err := Validate(p, state); err != nil {
		t.Fatalf("expected staged-item update to validate, got %v", err)
	}
}

func TestValidate_UpdateItem_RejectsEmptyPatch(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpUpdateItem, Target: "execute/foo", Patch: &ItemPatch{}},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "at least one field") {
		t.Fatalf("expected empty-patch error, got %v", err)
	}
}

func TestValidate_UpdateItem_RejectsInvalidPriority(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpUpdateItem, Target: "execute/foo", Patch: &ItemPatch{Priority: intPtr(0)}},
			{ID: "m2", Op: OpUpdateItem, Target: "execute/foo", Patch: &ItemPatch{Priority: intPtr(11)}},
		},
	}
	err := Validate(p, state)
	if err == nil {
		t.Fatalf("expected invalid priority error")
	}
}

func TestValidate_ChangeStatus_RejectsTerminal(t *testing.T) {
	state := baseState(t)
	for _, status := range []string{"completed", "failed", "needs_followup"} {
		p := Proposal{
			Form: FormMutationList,
			Mutations: []Mutation{
				{ID: "m1", Op: OpChangeStatus, Target: "execute/foo", Status: status},
			},
		}
		err := Validate(p, state)
		if err == nil || !errors.Is(err, ErrTerminalStatusWrite) {
			t.Fatalf("status=%s: expected terminal-status error, got %v", status, err)
		}
	}
}

func TestValidate_ChangeStatus_RejectsLifecycleOwned(t *testing.T) {
	state := baseState(t)
	for _, status := range []string{"queued", "in_progress", "in_review", "review_pending"} {
		p := Proposal{
			Form: FormMutationList,
			Mutations: []Mutation{
				{ID: "m1", Op: OpChangeStatus, Target: "execute/foo", Status: status},
			},
		}
		err := Validate(p, state)
		if err == nil || !strings.Contains(err.Error(), "controlled by") {
			t.Fatalf("status=%s: expected lifecycle-controlled error, got %v", status, err)
		}
	}
}

func TestValidate_ChangeStatus_AcceptsUserSettable(t *testing.T) {
	state := baseState(t)
	for _, status := range []string{"backlog", "researching", "ready"} {
		p := Proposal{
			Form: FormMutationList,
			Mutations: []Mutation{
				{ID: "m1", Op: OpChangeStatus, Target: "execute/foo", Status: status},
			},
		}
		if err := Validate(p, state); err != nil {
			t.Fatalf("status=%s: expected accepted, got %v", status, err)
		}
	}
}

func TestValidate_Edge_RejectsSelfLoop(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddEdge, From: "execute/foo", To: "execute/foo"},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "distinct endpoints") {
		t.Fatalf("expected self-loop error, got %v", err)
	}
}

func TestValidate_Edge_RejectsMissingEndpoint(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddEdge, From: "execute/foo", To: "execute/missing"},
		},
	}
	err := Validate(p, state)
	if err == nil || !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected target-not-found error, got %v", err)
	}
}

func TestValidate_Edge_RejectsDuplicateAdd(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddEdge, From: "execute/foo", To: "execute/bar"},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "edge already exists") {
		t.Fatalf("expected already-exists error, got %v", err)
	}
}

func TestValidate_Edge_RejectsRemovingMissingEdge(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpRemoveEdge, From: "execute/bar", To: "execute/foo"},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "edge does not exist") {
		t.Fatalf("expected does-not-exist error, got %v", err)
	}
}

func TestValidate_MoveInitiative_RejectsUnknownDest(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpMoveInitiative, Target: "execute/foo", Initiative: "ghost-initiative"},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "not a known initiative") {
		t.Fatalf("expected unknown-dest error, got %v", err)
	}
}

func TestValidate_MoveInitiative_RejectsSelfMove(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpMoveInitiative, Target: "execute/foo", Initiative: "ui-rewrite"},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "is the current initiative") {
		t.Fatalf("expected self-move error, got %v", err)
	}
}

func TestValidate_MoveInitiative_AllowsDetach(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpMoveInitiative, Target: "execute/foo", Initiative: ""},
		},
	}
	if err := Validate(p, state); err != nil {
		t.Fatalf("expected detach to validate, got %v", err)
	}
}

func TestValidate_Interrupt_RequiresInProgressState(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpInterruptInProgress, Target: "execute/foo"},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "requires execute/foo to be in_progress") {
		t.Fatalf("expected in_progress-required error, got %v", err)
	}

	state.InProgressRefs = map[string]struct{}{"execute/foo": {}}
	if err := Validate(p, state); err != nil {
		t.Fatalf("expected interrupt to validate when item is in_progress, got %v", err)
	}
}

func TestValidate_Split_RequiresTwoChildren(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpSplitItem, Target: "execute/foo", Into: []ItemSpec{
				{Kind: "execute", Name: "foo-a", Title: "A"},
			}},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "at least 2 new items") {
		t.Fatalf("expected split-requires-2 error, got %v", err)
	}
}

func TestValidate_Split_RejectsDuplicateChildWithExisting(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpSplitItem, Target: "execute/foo", Into: []ItemSpec{
				{Kind: "execute", Name: "bar", Title: "Collides"},
				{Kind: "execute", Name: "new", Title: "OK"},
			}},
		},
	}
	err := Validate(p, state)
	if err == nil || !errors.Is(err, ErrDuplicateItem) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestValidate_ItemSpec_ValidatesEffort(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "baz", Title: "Baz", Effort: "HUGE"}},
		},
	}
	err := Validate(p, state)
	if err == nil || !strings.Contains(err.Error(), "effort must be") {
		t.Fatalf("expected effort error, got %v", err)
	}
}

func TestValidate_AccumulatesAllProblems(t *testing.T) {
	state := baseState(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "", Op: OpArchiveItem, Target: "execute/foo"},
			{ID: "m1", Op: OpArchiveItem, Target: "execute/ghost"},
			{ID: "m1", Op: OpArchiveItem, Target: "execute/foo"},
		},
	}
	err := Validate(p, state)
	if err == nil {
		t.Fatalf("expected accumulated errors")
	}
	msg := err.Error()
	if !strings.Contains(msg, "id is required") {
		t.Fatalf("missing id error: %v", err)
	}
	if !strings.Contains(msg, "target not found") {
		t.Fatalf("missing target-not-found error: %v", err)
	}
	if !strings.Contains(msg, "duplicate id") {
		t.Fatalf("missing duplicate-id error: %v", err)
	}
}

func TestValidate_StrPtrHelper(_ *testing.T) {
	_ = strPtr("unused") // silence unused-helper warning until a later test needs it
}
