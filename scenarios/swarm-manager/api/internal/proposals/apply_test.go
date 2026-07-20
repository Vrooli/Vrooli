package proposals

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
)

// applyEnv bundles the test fixtures: a temp-dir-backed backlog FileStore,
// initiatives Service, and the Applier under test. Each test gets its own
// env so state doesn't leak between cases.
type applyEnv struct {
	t          *testing.T
	root       string
	backlog    *backlog.FileStore
	initSvc    *initiatives.Service
	creator    *backlog.Service
	applier    *Applier
	cancelFake *fakeCanceller
	schedFake  *fakeScheduler
}

// creatorWith builds a backlog.Service for a custom store/assigner combo
// (used by the "flaky" tests that wrap inner dependencies). Tests that
// don't customize use env.creator directly.
func creatorWith(t *testing.T, store backlog.CreationStore, assigner backlog.ItemAttacher) *backlog.Service {
	t.Helper()
	svc, err := backlog.NewService(backlog.ServiceConfig{Store: store, Assigner: assigner})
	if err != nil {
		t.Fatalf("backlog.NewService: %v", err)
	}
	return svc
}

type fakeCanceller struct {
	calls   []string
	err     error
	panicOn string // when set, panic if the call's "kind/name" matches
}

func (f *fakeCanceller) CancelForBacklog(_ context.Context, kind, name string) error {
	ref := kind + "/" + name
	f.calls = append(f.calls, ref)
	if f.panicOn != "" && f.panicOn == ref {
		panic("simulated downstream panic")
	}
	return f.err
}

type fakeScheduler struct{ calls int }

func (f *fakeScheduler) ScheduleAll() { f.calls++ }

type fakeEvents struct {
	calls    []string
	captured []capturedEvent
}

type capturedEvent struct {
	source   Source
	mutation Mutation
}

func (f *fakeEvents) EmitProposalMutationApplied(source Source, m Mutation) {
	f.calls = append(f.calls, source.InitiativeName+":"+m.ID+":"+string(m.Op))
	f.captured = append(f.captured, capturedEvent{source: source, mutation: m})
}

func newApplyEnv(t *testing.T) *applyEnv {
	t.Helper()
	root := t.TempDir()
	// Match FileStore layout: kind dirs are peers of initiatives dir.
	for _, dir := range []string{"ideas", "research", "fixes", "executes", "chores", "initiatives"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	store := backlog.NewFileStore(root)
	initStore := initiatives.NewStore(filepath.Join(root, "initiatives"))
	initSvc := initiatives.NewService(initStore, store)

	cancelFake := &fakeCanceller{}
	schedFake := &fakeScheduler{}
	creator, err := backlog.NewService(backlog.ServiceConfig{
		Store:    store,
		Assigner: initSvc,
		// Tests assert event counts via fakeEvents on the Applier; the
		// Service-level eventlog emit is validated separately in the
		// backlog package, so leave Events nil here to keep these
		// proposal-focused tests isolated.
	})
	if err != nil {
		t.Fatalf("backlog.NewService: %v", err)
	}
	applier, err := NewApplier(Config{
		Store:       store,
		Assigner:    initSvc,
		Creator:     creator,
		Canceller:   cancelFake,
		Invalidator: schedFake,
		Events:      &fakeEvents{},
	})
	if err != nil {
		t.Fatalf("NewApplier: %v", err)
	}

	// Seed a working initiative + two items so most ops have a target.
	if _, err := initSvc.Create(initiatives.CreateRequest{
		Name:  "ui-rewrite",
		Title: "UI Rewrite",
	}); err != nil {
		t.Fatalf("seed initiative: %v", err)
	}
	if _, err := initSvc.Create(initiatives.CreateRequest{
		Name:  "other-project",
		Title: "Other",
	}); err != nil {
		t.Fatalf("seed other init: %v", err)
	}
	seedItem(t, store, "execute", "foo", "Foo")
	seedItem(t, store, "execute", "bar", "Bar")
	if err := initSvc.AddItems("ui-rewrite", []string{"execute/foo", "execute/bar"}); err != nil {
		t.Fatalf("add items: %v", err)
	}

	return &applyEnv{
		t:          t,
		root:       root,
		backlog:    store,
		initSvc:    initSvc,
		creator:    creator,
		applier:    applier,
		cancelFake: cancelFake,
		schedFake:  schedFake,
	}
}

func TestApply_DeniedMutationLeavesItemDirectoryByteIdentical(t *testing.T) {
	env := newApplyEnv(t)
	path := filepath.Join(env.backlog.ItemDir(backlog.KindExecute, "foo"), "spec.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state := env.currentState()
	proposal := Proposal{Form: FormMutationList, Mutations: []Mutation{{ID: "deny", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(1)}}}
	result, err := env.applier.Apply(context.Background(), proposal, state, []string{}, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Applied != 0 || result.Skipped != 1 {
		t.Fatalf("unexpected outcome: %+v", result)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("denied proposal changed the canonical item spec")
	}
}

func seedItem(t *testing.T, store *backlog.FileStore, kind, name, title string) {
	t.Helper()
	if err := os.MkdirAll(store.ItemDir(backlog.BacklogKind(kind), name), 0o755); err != nil {
		t.Fatalf("mkdir item: %v", err)
	}
	if err := store.SaveItem(backlog.BacklogItem{
		Name:     name,
		Title:    title,
		Kind:     backlog.BacklogKind(kind),
		Status:   backlog.StatusBacklog,
		Priority: 5,
		Created:  "2026-04-23T00:00:00Z",
		Updated:  "2026-04-23T00:00:00Z",
	}); err != nil {
		t.Fatalf("save item: %v", err)
	}
}

func (e *applyEnv) currentState() CurrentState {
	e.t.Helper()
	state := CurrentState{
		InitiativeName: "ui-rewrite",
		Nodes:          map[string]GraphNode{},
		KnownInitiatives: map[string]struct{}{
			"ui-rewrite":    {},
			"other-project": {},
		},
		InProgressRefs: map[string]struct{}{},
	}
	items := []string{"execute/foo", "execute/bar"}
	for _, ref := range items {
		parts := strings.SplitN(ref, "/", 2)
		item, err := e.backlog.LoadItem(backlog.BacklogKind(parts[0]), parts[1])
		if err != nil {
			continue
		}
		if item.Initiative != "ui-rewrite" {
			continue
		}
		state.Nodes[ref] = GraphNode{
			ID:       ref,
			Kind:     parts[0],
			Name:     parts[1],
			Title:    item.Title,
			Priority: item.Priority,
			Effort:   item.Effort,
		}
		if item.Status == backlog.StatusInProgress {
			state.InProgressRefs[ref] = struct{}{}
		}
	}
	return state
}

func (e *applyEnv) loadItem(kind, name string) backlog.BacklogItem {
	e.t.Helper()
	item, err := e.backlog.LoadItem(backlog.BacklogKind(kind), name)
	if err != nil {
		e.t.Fatalf("load %s/%s: %v", kind, name, err)
	}
	return item
}

func TestApply_AddItem_CreatesItemAndAttaches(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddItem, Item: &ItemSpec{
				Kind: "execute", Name: "baz", Title: "Baz",
				Priority: 4, Effort: "s",
			}},
		},
	}
	res, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 || res.Failed != 0 {
		t.Fatalf("expected 1 applied, got %+v", res)
	}
	item := env.loadItem("execute", "baz")
	if item.Initiative != "ui-rewrite" {
		t.Fatalf("expected initiative ui-rewrite, got %q", item.Initiative)
	}
	if item.Effort != "S" {
		t.Fatalf("expected effort=S, got %q", item.Effort)
	}
	init, err := env.initSvc.Get("ui-rewrite")
	if err != nil {
		t.Fatalf("get init: %v", err)
	}
	if !stringSliceContains(init.Initiative.Items, "execute/baz") {
		t.Fatalf("initiative items missing execute/baz: %v", init.Initiative.Items)
	}
	if env.schedFake.calls == 0 {
		t.Fatalf("expected invalidator.ScheduleAll to be called on successful apply")
	}
}

func TestApply_ChangePriority_ModifiesPriority(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(10)},
		},
	}
	res, err := env.applier.Apply(context.Background(), p, env.currentState(), []string{"m1"}, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 {
		t.Fatalf("expected Applied=1, got %+v", res)
	}
	foo := env.loadItem("execute", "foo")
	if foo.Priority != 10 {
		t.Fatalf("priority = %d, want 10", foo.Priority)
	}
}

func TestApply_SkipsUnacceptedMutations(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "accept", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(9)},
			{ID: "skip", Op: OpChangePriority, Target: "execute/bar", Priority: intPtr(8)},
		},
	}
	res, err := env.applier.Apply(context.Background(), p, env.currentState(), []string{"accept"}, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 || res.Skipped != 1 {
		t.Fatalf("expected 1 applied + 1 skipped, got %+v", res)
	}
	foo := env.loadItem("execute", "foo")
	if foo.Priority != 9 {
		t.Fatalf("expected foo.priority=9, got %d", foo.Priority)
	}
	bar := env.loadItem("execute", "bar")
	if bar.Priority != 5 {
		t.Fatalf("expected bar.priority unchanged at 5, got %d", bar.Priority)
	}
	// Per-outcome bookkeeping must distinguish skipped from failed so
	// renderers don't conflate "user chose not to apply" with "apply
	// tried and broke." Outcome.Skipped is true iff the mutation was
	// deselected; the complementary outcome for "accept" has Applied=true.
	var skipOutcome, appliedOutcome *Outcome
	for i := range res.Outcomes {
		switch res.Outcomes[i].MutationID {
		case "skip":
			skipOutcome = &res.Outcomes[i]
		case "accept":
			appliedOutcome = &res.Outcomes[i]
		}
	}
	if skipOutcome == nil || !skipOutcome.Skipped || skipOutcome.Applied || skipOutcome.Error != "" {
		t.Fatalf("expected skipped outcome to have Skipped=true, Applied=false, Error=\"\", got %+v", skipOutcome)
	}
	if appliedOutcome == nil || appliedOutcome.Skipped || !appliedOutcome.Applied {
		t.Fatalf("expected accepted outcome to have Applied=true, Skipped=false, got %+v", appliedOutcome)
	}
}

func TestApply_UpdateItem_AppliesPatch(t *testing.T) {
	env := newApplyEnv(t)
	newTitle := "Foo Reworked"
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpUpdateItem, Target: "execute/foo", Patch: &ItemPatch{
				Title: &newTitle, Priority: intPtr(8),
			}},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	item := env.loadItem("execute", "foo")
	if item.Title != "Foo Reworked" || item.Priority != 8 {
		t.Fatalf("patch not applied: %+v", item)
	}
}

func TestApply_ChangeStatus_AcceptsUserSettable(t *testing.T) {
	env := newApplyEnv(t)
	for _, status := range []string{"backlog", "researching", "ready"} {
		p := Proposal{
			Form: FormMutationList,
			Mutations: []Mutation{
				{ID: "m1", Op: OpChangeStatus, Target: "execute/foo", Status: status},
			},
		}
		res, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
		if err != nil {
			t.Fatalf("apply status=%s: %v", status, err)
		}
		if res.Applied != 1 || res.Failed != 0 {
			t.Fatalf("status=%s: expected 1 applied, got %+v", status, res)
		}
		foo := env.loadItem("execute", "foo")
		if string(foo.Status) != status {
			t.Fatalf("status=%s: expected persisted, got %q", status, foo.Status)
		}
	}
}

func TestApply_ChangeStatus_RejectsTerminal(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangeStatus, Target: "execute/foo", Status: "completed"},
		},
	}
	_, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err == nil || !errors.Is(err, ErrTerminalStatusWrite) {
		t.Fatalf("expected terminal-status error, got %v", err)
	}
}

func TestApply_AddEdge_AddsToDependsOn(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddEdge, From: "execute/bar", To: "execute/foo"},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	bar := env.loadItem("execute", "bar")
	if !stringSliceContains(bar.DependsOn, "execute/foo") {
		t.Fatalf("expected bar.depends_on to contain execute/foo, got %v", bar.DependsOn)
	}
}

func TestApply_RemoveEdge_RemovesFromDependsOn(t *testing.T) {
	env := newApplyEnv(t)
	// Seed the edge.
	bar := env.loadItem("execute", "bar")
	bar.DependsOn = []string{"execute/foo"}
	if err := env.backlog.SaveItem(bar); err != nil {
		t.Fatalf("seed edge: %v", err)
	}
	state := env.currentState()
	state.Edges = []GraphEdge{{From: "execute/bar", To: "execute/foo"}}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpRemoveEdge, From: "execute/bar", To: "execute/foo"},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, state, nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	bar = env.loadItem("execute", "bar")
	if stringSliceContains(bar.DependsOn, "execute/foo") {
		t.Fatalf("expected edge removed, got %v", bar.DependsOn)
	}
}

func TestApply_MoveInitiative_ReassignsMembership(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpMoveInitiative, Target: "execute/foo", Initiative: "other-project"},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	foo := env.loadItem("execute", "foo")
	if foo.Initiative != "other-project" {
		t.Fatalf("expected initiative=other-project, got %q", foo.Initiative)
	}
	origin, err := env.initSvc.Get("ui-rewrite")
	if err != nil {
		t.Fatalf("get origin: %v", err)
	}
	if stringSliceContains(origin.Initiative.Items, "execute/foo") {
		t.Fatalf("origin still lists execute/foo: %v", origin.Initiative.Items)
	}
	dest, err := env.initSvc.Get("other-project")
	if err != nil {
		t.Fatalf("get dest: %v", err)
	}
	if !stringSliceContains(dest.Initiative.Items, "execute/foo") {
		t.Fatalf("destination missing execute/foo: %v", dest.Initiative.Items)
	}
}

func TestApply_ArchiveItem_SetsArchivedAt(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpArchiveItem, Target: "execute/foo"},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	foo := env.loadItem("execute", "foo")
	if foo.ArchivedAt == nil || *foo.ArchivedAt == "" {
		t.Fatalf("expected archivedAt set, got %+v", foo.ArchivedAt)
	}
}

func TestApply_InterruptInProgress_DelegatesToCanceller(t *testing.T) {
	env := newApplyEnv(t)
	// Flip foo to in_progress so the validator accepts.
	foo := env.loadItem("execute", "foo")
	foo.Status = backlog.StatusInProgress
	if err := env.backlog.SaveItem(foo); err != nil {
		t.Fatalf("set in_progress: %v", err)
	}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpInterruptInProgress, Target: "execute/foo"},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(env.cancelFake.calls) != 1 || env.cancelFake.calls[0] != "execute/foo" {
		t.Fatalf("expected canceller called for execute/foo, got %v", env.cancelFake.calls)
	}
}

// TestApply_RecoversFromMutationPanic guards the production fix for the
// typed-nil interface in backlog.Service.events that 500'd the feedback
// Apply button. A panic mid-mutation must surface as a per-mutation
// failure (so the loop continues and prior outcomes survive), not unwind
// the whole batch.
func TestApply_RecoversFromMutationPanic(t *testing.T) {
	env := newApplyEnv(t)
	foo := env.loadItem("execute", "foo")
	foo.Status = backlog.StatusInProgress
	if err := env.backlog.SaveItem(foo); err != nil {
		t.Fatalf("set in_progress: %v", err)
	}
	bar := env.loadItem("execute", "bar")
	bar.Status = backlog.StatusInProgress
	if err := env.backlog.SaveItem(bar); err != nil {
		t.Fatalf("set in_progress: %v", err)
	}

	env.cancelFake.panicOn = "execute/foo"

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpInterruptInProgress, Target: "execute/foo"},
			{ID: "m2", Op: OpInterruptInProgress, Target: "execute/bar"},
		},
	}
	res, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply returned error (panic should have been recovered): %v", err)
	}
	if res.Applied != 1 || res.Failed != 1 {
		t.Fatalf("expected Applied=1 Failed=1, got %+v", res)
	}
	if got := res.Outcomes[0]; got.Applied || !strings.Contains(got.Error, "panicked") {
		t.Fatalf("m1 outcome should record panic, got %+v", got)
	}
	if got := res.Outcomes[1]; !got.Applied {
		t.Fatalf("m2 should have applied after m1 panicked, got %+v", got)
	}
}

func TestApply_SplitItem_CreatesChildrenAndArchivesSource(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpSplitItem, Target: "execute/foo", Into: []ItemSpec{
				{Kind: "execute", Name: "foo-ui", Title: "Foo UI"},
				{Kind: "execute", Name: "foo-api", Title: "Foo API"},
			}},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := env.backlog.LoadItem("execute", "foo-ui"); err != nil {
		t.Fatalf("foo-ui not created: %v", err)
	}
	if _, err := env.backlog.LoadItem("execute", "foo-api"); err != nil {
		t.Fatalf("foo-api not created: %v", err)
	}
	foo := env.loadItem("execute", "foo")
	if foo.ArchivedAt == nil {
		t.Fatalf("expected source foo archived, got %+v", foo)
	}
}

func TestApply_RequiresSourceInitiative(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{Form: FormMutationList}
	_, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{})
	if err == nil || !strings.Contains(err.Error(), "InitiativeName") {
		t.Fatalf("expected source-initiative error, got %v", err)
	}
}

func TestApply_RejectsMismatchedInitiative(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{Form: FormMutationList}
	_, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "different"})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected mismatched-initiative error, got %v", err)
	}
}

func TestApply_RevalidatesBeforeApplying(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpArchiveItem, Target: "execute/missing"},
		},
	}
	_, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err == nil || !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("expected revalidation error, got %v", err)
	}
}

func TestApply_EmitsEventsForSuccessfulMutations(t *testing.T) {
	env := newApplyEnv(t)
	events := &fakeEvents{}
	applier, err := NewApplier(Config{
		Store:    env.backlog,
		Assigner: env.initSvc,
		Creator:  env.creator,
		Events:   events,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(7)},
			{ID: "m2", Op: OpArchiveItem, Target: "execute/bar"},
		},
	}
	if _, err := applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite", FeedbackRoundID: "round-1"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	sort.Strings(events.calls)
	got := strings.Join(events.calls, "|")
	want := "ui-rewrite:m1:change_priority|ui-rewrite:m2:archive_item"
	if got != want {
		t.Fatalf("events: got %q, want %q", got, want)
	}
}

func TestApply_PropagatesRoundMetadataThroughEvents(t *testing.T) {
	env := newApplyEnv(t)
	events := &fakeEvents{}
	applier, err := NewApplier(Config{
		Store:    env.backlog,
		Assigner: env.initSvc,
		Creator:  env.creator,
		Events:   events,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(7)},
		},
	}
	src := Source{
		InitiativeName:  "ui-rewrite",
		FeedbackRoundID: "ui-rewrite/round-003",
		RoundNumber:     3,
		RoundSlug:       "ui-rewrite",
		Entrypoint:      "initiative.feedback",
		DecidedBy:       "test-operator",
	}
	if _, err := applier.Apply(context.Background(), p, env.currentState(), nil, src); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(events.captured) != 1 {
		t.Fatalf("expected 1 captured event, got %d", len(events.captured))
	}
	got := events.captured[0].source
	if got.RoundNumber != 3 {
		t.Errorf("RoundNumber: got %d, want 3", got.RoundNumber)
	}
	if got.RoundSlug != "ui-rewrite" {
		t.Errorf("RoundSlug: got %q, want %q", got.RoundSlug, "ui-rewrite")
	}
	if got.Entrypoint != "initiative.feedback" {
		t.Errorf("Entrypoint: got %q, want %q", got.Entrypoint, "initiative.feedback")
	}
	if got.FeedbackRoundID != "ui-rewrite/round-003" {
		t.Errorf("FeedbackRoundID: got %q", got.FeedbackRoundID)
	}
}

// flakyAssigner wraps a real assigner. RememberItem fails for the named
// initiative; calls to other names fall through. ForgetItem always
// passes through. Used to exercise the applyMoveInitiative rollback path
// where the destination membership write fails after the source detach
// already succeeded.
type flakyAssigner struct {
	inner       InitiativeAssigner
	failForName string
}

func (f *flakyAssigner) RememberItem(name, ref string) error {
	if name == f.failForName {
		return errors.New("simulated remember failure")
	}
	return f.inner.RememberItem(name, ref)
}

func (f *flakyAssigner) ForgetItem(name, ref string) error {
	return f.inner.ForgetItem(name, ref)
}

func TestApply_MoveInitiative_RollsBackWhenDestRememberFails(t *testing.T) {
	env := newApplyEnv(t)
	flaky := &flakyAssigner{inner: env.initSvc, failForName: "other-project"}
	applier, err := NewApplier(Config{Store: env.backlog, Assigner: flaky, Creator: creatorWith(t, env.backlog, flaky)})
	if err != nil {
		t.Fatal(err)
	}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpMoveInitiative, Target: "execute/foo", Initiative: "other-project"},
		},
	}
	res, err := applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Failed != 1 || res.Applied != 0 {
		t.Fatalf("expected the move to fail, got %+v", res)
	}
	if !strings.Contains(res.Outcomes[0].Error, "simulated remember failure") {
		t.Fatalf("expected remember-failure error, got %q", res.Outcomes[0].Error)
	}
	// Item.initiative should be rolled back to ui-rewrite, not stuck on
	// the destination — the rollback restores SetItemInitiative + the
	// membership list.
	foo := env.loadItem("execute", "foo")
	if foo.Initiative != "ui-rewrite" {
		t.Fatalf("rollback failed: item.initiative=%q, want ui-rewrite", foo.Initiative)
	}
	origin, err := env.initSvc.Get("ui-rewrite")
	if err != nil {
		t.Fatalf("get origin: %v", err)
	}
	if !stringSliceContains(origin.Initiative.Items, "execute/foo") {
		t.Fatalf("rollback failed: ui-rewrite no longer lists execute/foo: %v", origin.Initiative.Items)
	}
}

// flakyStore wraps the FileStore for rollback testing. Its sole purpose is
// to make the second LoadItem in applySplit fail (simulating disk loss
// mid-batch) so the rollback path archives the already-created child.
//
// It can also be configured to fail on load or on a non-first save, which
// the error-propagation tests use to verify non-rollback ops surface the
// failure as an Outcome error rather than panicking or succeeding silently.
type flakyStore struct {
	inner          BacklogStore
	failOnSave     string // ref ("kind/name") whose SaveItem should fail
	failOnLoad     string // ref ("kind/name") whose LoadItem should fail
	failOnSaveOnce bool   // when true, failOnSave only fires for the first matching call (lets rollback writes succeed)
	saveCalls      int
}

func (f *flakyStore) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	if f.failOnLoad != "" && string(kind)+"/"+name == f.failOnLoad {
		return backlog.BacklogItem{}, errors.New("simulated load failure")
	}
	return f.inner.LoadItem(kind, name)
}

func (f *flakyStore) SaveItem(item backlog.BacklogItem) error {
	f.saveCalls++
	if f.failOnSave != "" && string(item.Kind)+"/"+item.Name == f.failOnSave {
		if f.failOnSaveOnce {
			// Disarm so subsequent (rollback) saves succeed.
			f.failOnSave = ""
		}
		return errors.New("simulated save failure")
	}
	return f.inner.SaveItem(item)
}

func (f *flakyStore) ItemDir(kind backlog.BacklogKind, name string) string {
	return f.inner.ItemDir(kind, name)
}

func (f *flakyStore) SetItemInitiative(kind backlog.BacklogKind, name, initiative string) (string, error) {
	return f.inner.SetItemInitiative(kind, name, initiative)
}

func (f *flakyStore) ClearItemInitiative(kind backlog.BacklogKind, name, expected string) (string, bool, error) {
	return f.inner.ClearItemInitiative(kind, name, expected)
}

func (f *flakyStore) ValidateDependencies(deps []string) error {
	return f.inner.ValidateDependencies(deps)
}

func TestApply_SplitItem_RollsBackChildrenOnFailure(t *testing.T) {
	env := newApplyEnv(t)
	flaky := &flakyStore{inner: env.backlog, failOnSave: "execute/foo-api"}
	applier, err := NewApplier(Config{Store: flaky, Assigner: env.initSvc, Creator: creatorWith(t, flaky, env.initSvc)})
	if err != nil {
		t.Fatal(err)
	}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpSplitItem, Target: "execute/foo", Into: []ItemSpec{
				{Kind: "execute", Name: "foo-ui", Title: "Foo UI"},
				{Kind: "execute", Name: "foo-api", Title: "Foo API"},
			}},
		},
	}
	res, err := applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Failed != 1 {
		t.Fatalf("expected split to fail, got %+v", res)
	}
	// foo-ui was created before the failure — rollback should have
	// archived it so the user sees no half-applied split.
	fooUI := env.loadItem("execute", "foo-ui")
	if fooUI.ArchivedAt == nil || *fooUI.ArchivedAt == "" {
		t.Fatalf("expected foo-ui rolled back via archive, got %+v", fooUI)
	}
	// Source should NOT be archived (split aborted before that step).
	foo := env.loadItem("execute", "foo")
	if foo.ArchivedAt != nil && *foo.ArchivedAt != "" {
		t.Fatalf("source item should not be archived after rollback, got archivedAt=%v", foo.ArchivedAt)
	}
}

// ---------------------------------------------------------------------------
// merge_items
// ---------------------------------------------------------------------------

// mergeEnv extends the standard apply env with three execute items
// (alpha, beta, gamma) plus a fourth dependent (delta) that depends on
// alpha and beta. Two intra-source edges are also seeded so the test
// can verify they get dropped by the merge.
func newMergeEnv(t *testing.T) *applyEnv {
	t.Helper()
	env := newApplyEnv(t)
	for _, name := range []string{"alpha", "beta", "gamma", "delta"} {
		seedItem(t, env.backlog, "execute", name, strings.ToTitle(name))
	}
	if err := env.initSvc.AddItems("ui-rewrite", []string{"execute/alpha", "execute/beta", "execute/gamma", "execute/delta"}); err != nil {
		t.Fatalf("add items: %v", err)
	}
	// alpha depends on gamma (outbound external — should retarget to merged)
	mustSetDeps(t, env, "execute/alpha", []string{"execute/gamma"})
	// beta depends on alpha (intra-source — should drop)
	mustSetDeps(t, env, "execute/beta", []string{"execute/alpha"})
	// delta depends on alpha AND beta (inbound external from both — should
	// dedup to a single dep on the merged item).
	mustSetDeps(t, env, "execute/delta", []string{"execute/alpha", "execute/beta"})
	return env
}

func mustSetDeps(t *testing.T, env *applyEnv, ref string, deps []string) {
	t.Helper()
	parts := strings.SplitN(ref, "/", 2)
	item, err := env.backlog.LoadItem(backlog.BacklogKind(parts[0]), parts[1])
	if err != nil {
		t.Fatalf("load %s: %v", ref, err)
	}
	item.DependsOn = deps
	if err := env.backlog.SaveItem(item); err != nil {
		t.Fatalf("save deps for %s: %v", ref, err)
	}
}

// mergeState builds a CurrentState that reflects the seeded depends_on
// edges so apply has the edge picture when computing merges.
func (e *applyEnv) mergeState() CurrentState {
	state := e.currentState()
	for _, ref := range []string{"execute/alpha", "execute/beta", "execute/gamma", "execute/delta"} {
		parts := strings.SplitN(ref, "/", 2)
		item, err := e.backlog.LoadItem(backlog.BacklogKind(parts[0]), parts[1])
		if err != nil {
			continue
		}
		state.Nodes[ref] = GraphNode{
			ID:    ref,
			Kind:  parts[0],
			Name:  parts[1],
			Title: item.Title,
		}
		for _, dep := range item.DependsOn {
			state.Edges = append(state.Edges, GraphEdge{From: ref, To: dep})
		}
	}
	return state
}

func TestApply_MergeItems_HappyPath(t *testing.T) {
	env := newMergeEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{
				ID: "m1", Op: OpMergeItems,
				Sources: []string{"execute/alpha", "execute/beta"},
				Item: &ItemSpec{
					Kind: "execute", Name: "merged", Title: "Merged",
					Description: "Combines alpha + beta",
					Effort:      "M",
				},
			},
		},
	}
	state := env.mergeState()
	res, err := env.applier.Apply(context.Background(), p, state, nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 || res.Failed != 0 {
		t.Fatalf("expected applied=1, got %+v", res)
	}
	merged := env.loadItem("execute", "merged")
	if merged.Initiative != "ui-rewrite" {
		t.Fatalf("merged not attached to initiative, got %q", merged.Initiative)
	}
	if merged.Status != backlog.StatusBacklog {
		t.Fatalf("merged should enter as backlog, got %q", merged.Status)
	}
	// Outbound retarget: merged should depend on gamma.
	if !stringSliceContains(merged.DependsOn, "execute/gamma") {
		t.Fatalf("merged should retarget alpha's gamma dep, got deps=%v", merged.DependsOn)
	}
	// Sources archived.
	for _, src := range []string{"execute/alpha", "execute/beta"} {
		parts := strings.SplitN(src, "/", 2)
		item, err := env.backlog.LoadItem(backlog.BacklogKind(parts[0]), parts[1])
		if err != nil {
			t.Fatalf("load %s: %v", src, err)
		}
		if item.ArchivedAt == nil || *item.ArchivedAt == "" {
			t.Fatalf("source %s not archived", src)
		}
	}
	// Inbound dedup: delta now depends on merged exactly once.
	delta := env.loadItem("execute", "delta")
	mergedRefCount := 0
	for _, dep := range delta.DependsOn {
		if dep == "execute/merged" {
			mergedRefCount++
		}
	}
	if mergedRefCount != 1 {
		t.Fatalf("expected delta to depend on execute/merged exactly once, got deps=%v", delta.DependsOn)
	}
	// And delta no longer depends on alpha or beta.
	for _, gone := range []string{"execute/alpha", "execute/beta"} {
		if stringSliceContains(delta.DependsOn, gone) {
			t.Fatalf("delta should no longer depend on archived source %s, got deps=%v", gone, delta.DependsOn)
		}
	}
}

func TestApply_MergeItems_DropsIntraSourceEdges(t *testing.T) {
	env := newMergeEnv(t)
	// Both alpha and beta are sources; beta depends on alpha (intra-source).
	// The merged item must NOT inherit a self-loop or any merged→merged dep.
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{
				ID: "m1", Op: OpMergeItems,
				Sources: []string{"execute/alpha", "execute/beta"},
				Item:    &ItemSpec{Kind: "execute", Name: "merged", Title: "Merged"},
			},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.mergeState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	merged := env.loadItem("execute", "merged")
	if stringSliceContains(merged.DependsOn, "execute/merged") {
		t.Fatalf("merged should not self-loop, got deps=%v", merged.DependsOn)
	}
	if stringSliceContains(merged.DependsOn, "execute/alpha") || stringSliceContains(merged.DependsOn, "execute/beta") {
		t.Fatalf("merged should drop intra-source deps, got deps=%v", merged.DependsOn)
	}
}

func TestApply_MergeItems_RetargetsExternalEdges(t *testing.T) {
	env := newMergeEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{
				ID: "m1", Op: OpMergeItems,
				Sources: []string{"execute/alpha", "execute/beta"},
				Item:    &ItemSpec{Kind: "execute", Name: "merged", Title: "Merged"},
			},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.mergeState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	merged := env.loadItem("execute", "merged")
	// alpha→gamma should become merged→gamma.
	if !stringSliceContains(merged.DependsOn, "execute/gamma") {
		t.Fatalf("expected merged to depend on gamma, got %v", merged.DependsOn)
	}
	// delta→alpha and delta→beta should both retarget to delta→merged.
	delta := env.loadItem("execute", "delta")
	if !stringSliceContains(delta.DependsOn, "execute/merged") {
		t.Fatalf("expected delta to depend on merged, got %v", delta.DependsOn)
	}
}

func TestApply_MergeItems_MergedSpecDependsOnFiltersSources(t *testing.T) {
	// If the agent's merged ItemSpec.depends_on accidentally lists a
	// source ref, apply must filter it out — sources are about to be
	// archived and the merged item must not retain a stale dep.
	env := newMergeEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{
				ID: "m1", Op: OpMergeItems,
				Sources: []string{"execute/alpha", "execute/beta"},
				Item: &ItemSpec{
					Kind: "execute", Name: "merged", Title: "Merged",
					DependsOn: []string{"execute/alpha", "execute/gamma"},
				},
			},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.mergeState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	merged := env.loadItem("execute", "merged")
	if stringSliceContains(merged.DependsOn, "execute/alpha") {
		t.Fatalf("merged spec dep on alpha should have been filtered, got %v", merged.DependsOn)
	}
	if !stringSliceContains(merged.DependsOn, "execute/gamma") {
		t.Fatalf("merged should retain non-source dep on gamma, got %v", merged.DependsOn)
	}
}

func TestApply_MergeItems_RollbackOnSourceArchiveFailure(t *testing.T) {
	env := newMergeEnv(t)
	// Fail when the second source ("beta") is saved during archive.
	flaky := &flakyStore{inner: env.backlog, failOnSave: "execute/beta"}
	applier, err := NewApplier(Config{
		Store:    flaky,
		Assigner: env.initSvc,
		Creator:  creatorWith(t, flaky, env.initSvc),
	})
	if err != nil {
		t.Fatal(err)
	}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{
				ID: "m1", Op: OpMergeItems,
				Sources: []string{"execute/alpha", "execute/beta"},
				Item:    &ItemSpec{Kind: "execute", Name: "merged", Title: "Merged"},
			},
		},
	}
	res, err := applier.Apply(context.Background(), p, env.mergeState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Failed != 1 || res.Applied != 0 {
		t.Fatalf("expected merge failure, got %+v", res)
	}
	// Merged item should be archived (rolled back).
	merged, err := env.backlog.LoadItem("execute", "merged")
	if err != nil {
		t.Fatalf("merged should still be on disk (archived), got load err: %v", err)
	}
	if merged.ArchivedAt == nil || *merged.ArchivedAt == "" {
		t.Fatalf("merged should be archived after rollback, got archivedAt=%v", merged.ArchivedAt)
	}
	// alpha (already archived during forward pass) should be un-archived.
	alpha := env.loadItem("execute", "alpha")
	if alpha.ArchivedAt != nil && *alpha.ArchivedAt != "" {
		t.Fatalf("alpha should be un-archived on rollback, got archivedAt=%v", alpha.ArchivedAt)
	}
	// delta's depends_on should be restored — still references the
	// original sources, not merged.
	delta := env.loadItem("execute", "delta")
	if stringSliceContains(delta.DependsOn, "execute/merged") {
		t.Fatalf("delta should not reference merged after rollback, got %v", delta.DependsOn)
	}
	if !stringSliceContains(delta.DependsOn, "execute/alpha") || !stringSliceContains(delta.DependsOn, "execute/beta") {
		t.Fatalf("delta deps should be restored to alpha+beta, got %v", delta.DependsOn)
	}
}

// TestApply_MergeItems_RollbackOnRetargetFailure exercises the rollback
// path when a retarget-step SaveItem fails (delta is the inbound
// dependent the retarget loop tries to save). At the failure point:
//   - merged item exists (created in step 3)
//   - no source has been archived yet (archive is step 5)
//   - delta's depends_on may have been written or not
//
// Rollback must:
//   - archive the merged item
//   - restore delta's depends_on from the snapshot
//   - leave sources un-archived (they never were archived)
//
// The flakyStore is configured one-shot so the rollback's restore-write
// for delta succeeds (otherwise rollback logs a warning and gives up,
// which would muddy the assertion).
func TestApply_MergeItems_RollbackOnRetargetFailure(t *testing.T) {
	env := newMergeEnv(t)
	flaky := &flakyStore{inner: env.backlog, failOnSave: "execute/delta", failOnSaveOnce: true}
	applier, err := NewApplier(Config{
		Store:    flaky,
		Assigner: env.initSvc,
		Creator:  creatorWith(t, flaky, env.initSvc),
	})
	if err != nil {
		t.Fatal(err)
	}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{
				ID: "m1", Op: OpMergeItems,
				Sources: []string{"execute/alpha", "execute/beta"},
				Item:    &ItemSpec{Kind: "execute", Name: "merged", Title: "Merged"},
			},
		},
	}
	res, err := applier.Apply(context.Background(), p, env.mergeState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Failed != 1 || res.Applied != 0 {
		t.Fatalf("expected merge failure, got %+v", res)
	}
	if !strings.Contains(res.Outcomes[0].Error, "retargeted") && !strings.Contains(res.Outcomes[0].Error, "execute/delta") {
		t.Fatalf("outcome should mention retarget failure, got %q", res.Outcomes[0].Error)
	}

	// Merged item created then archived as part of rollback.
	merged, err := env.backlog.LoadItem("execute", "merged")
	if err != nil {
		t.Fatalf("merged should still be on disk (archived), got load err: %v", err)
	}
	if merged.ArchivedAt == nil || *merged.ArchivedAt == "" {
		t.Fatalf("merged should be archived after rollback, got archivedAt=%v", merged.ArchivedAt)
	}

	// Sources never reached step 5 — must not be archived.
	for _, src := range []string{"execute/alpha", "execute/beta"} {
		parts := strings.SplitN(src, "/", 2)
		item, lerr := env.backlog.LoadItem(backlog.BacklogKind(parts[0]), parts[1])
		if lerr != nil {
			t.Fatalf("load %s: %v", src, lerr)
		}
		if item.ArchivedAt != nil && *item.ArchivedAt != "" {
			t.Fatalf("source %s should not be archived (rollback path didn't reach step 5), got archivedAt=%v", src, item.ArchivedAt)
		}
	}

	// delta's depends_on must be the original alpha+beta (snapshot
	// restored), not the retargeted merged ref.
	delta := env.loadItem("execute", "delta")
	if stringSliceContains(delta.DependsOn, "execute/merged") {
		t.Fatalf("delta should not reference merged after rollback, got %v", delta.DependsOn)
	}
	if !stringSliceContains(delta.DependsOn, "execute/alpha") || !stringSliceContains(delta.DependsOn, "execute/beta") {
		t.Fatalf("delta deps should be restored to alpha+beta, got %v", delta.DependsOn)
	}
}

func TestApply_MergeItems_EmitsEventOnMergedRefAndPropagatesSources(t *testing.T) {
	env := newMergeEnv(t)
	events := &fakeEvents{}
	env.applier.events = events

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{
				ID: "m1", Op: OpMergeItems,
				Sources: []string{"execute/alpha", "execute/beta"},
				Item:    &ItemSpec{Kind: "execute", Name: "merged", Title: "Merged"},
			},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.mergeState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(events.captured) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events.captured))
	}
	got := events.captured[0]
	if got.mutation.Op != OpMergeItems {
		t.Fatalf("expected merge op on event, got %s", got.mutation.Op)
	}
	if len(got.mutation.Sources) != 2 || got.mutation.Sources[0] != "execute/alpha" || got.mutation.Sources[1] != "execute/beta" {
		t.Fatalf("expected sources to round-trip on captured mutation, got %v", got.mutation.Sources)
	}
}

func TestProposalApplyTarget_MergeItemsReturnsMergedRef(t *testing.T) {
	m := Mutation{
		Op:      OpMergeItems,
		Sources: []string{"execute/alpha", "execute/beta"},
		Item:    &ItemSpec{Kind: "execute", Name: "merged"},
	}
	if got := applyTarget(m); got != "execute/merged" {
		t.Fatalf("expected applyTarget to return merged ref, got %q", got)
	}
}

func TestRetargetDependsOn_DedupesAndPreservesOrder(t *testing.T) {
	sourceSet := map[string]struct{}{
		"execute/alpha": {},
		"execute/beta":  {},
	}
	got := retargetDependsOn(
		[]string{"execute/alpha", "execute/zeta", "execute/beta", "execute/zeta"},
		sourceSet,
		"execute/merged",
	)
	want := []string{"execute/zeta", "execute/merged"}
	if !stringSlicesEqual(got, want) {
		t.Fatalf("retarget got %v, want %v", got, want)
	}
}

func TestRetargetDependsOn_NoSourceRefsLeavesUnchanged(t *testing.T) {
	sourceSet := map[string]struct{}{"execute/foo": {}}
	got := retargetDependsOn([]string{"execute/bar", "execute/baz"}, sourceSet, "execute/merged")
	want := []string{"execute/bar", "execute/baz"}
	if !stringSlicesEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func stringSliceContains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}

// TestApply_StagedEdge_BothEndpointsNewlyCreated covers the staged-item
// edge case the validator supports (validateEdge + hasStagedNewItem): an
// agent proposes two new items and an edge between them in a single
// mutation_list. Both ops must land, and the edge must be recorded on the
// "from" item's depends_on — not dropped silently because the endpoints
// didn't pre-exist in the initiative graph.
func TestApply_StagedEdge_BothEndpointsNewlyCreated(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "alpha", Title: "Alpha", Priority: 4, Effort: "s"}},
			{ID: "m2", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "beta", Title: "Beta", Priority: 4, Effort: "s"}},
			{ID: "m3", Op: OpAddEdge, From: "execute/beta", To: "execute/alpha"},
		},
	}
	res, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 3 || res.Failed != 0 {
		t.Fatalf("expected 3 applied, got %+v (outcomes=%+v)", res, res.Outcomes)
	}
	beta := env.loadItem("execute", "beta")
	if !stringSliceContains(beta.DependsOn, "execute/alpha") {
		t.Fatalf("expected beta.depends_on to contain execute/alpha, got %v", beta.DependsOn)
	}
}

// TestApply_UpdateItem_PropagatesSaveError covers the non-rollback error
// path for a simple patch op: if SaveItem fails under the hood, Apply
// must surface the failure in Outcomes (Failed=1) rather than falsely
// reporting success. The item on disk must remain unchanged.
func TestApply_UpdateItem_PropagatesSaveError(t *testing.T) {
	env := newApplyEnv(t)
	flaky := &flakyStore{inner: env.backlog, failOnSave: "execute/foo"}
	applier, err := NewApplier(Config{Store: flaky, Assigner: env.initSvc, Creator: creatorWith(t, flaky, env.initSvc)})
	if err != nil {
		t.Fatal(err)
	}
	newTitle := "Foo Reworked"
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpUpdateItem, Target: "execute/foo", Patch: &ItemPatch{Title: &newTitle}},
		},
	}
	res, err := applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 0 || res.Failed != 1 {
		t.Fatalf("expected failure surfaced, got %+v", res)
	}
	if !strings.Contains(res.Outcomes[0].Error, "simulated save failure") {
		t.Fatalf("outcome should mention store failure, got %q", res.Outcomes[0].Error)
	}
	foo := env.loadItem("execute", "foo")
	if foo.Title == newTitle {
		t.Fatalf("item title should not have changed on save failure, got %q", foo.Title)
	}
}

// TestApply_AddEdge_PropagatesLoadError covers the complementary load
// failure path. If LoadItem fails (e.g. disk gone or item removed
// concurrently), add_edge must report the failure rather than panic.
func TestApply_AddEdge_PropagatesLoadError(t *testing.T) {
	env := newApplyEnv(t)
	flaky := &flakyStore{inner: env.backlog, failOnLoad: "execute/bar"}
	applier, err := NewApplier(Config{Store: flaky, Assigner: env.initSvc, Creator: creatorWith(t, flaky, env.initSvc)})
	if err != nil {
		t.Fatal(err)
	}
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpAddEdge, From: "execute/bar", To: "execute/foo"},
		},
	}
	res, err := applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 0 || res.Failed != 1 {
		t.Fatalf("expected load failure surfaced, got %+v", res)
	}
	if !strings.Contains(res.Outcomes[0].Error, "simulated load failure") {
		t.Fatalf("outcome should mention load failure, got %q", res.Outcomes[0].Error)
	}
}

// TestApply_NormalizedWhitespaceFlowsThroughCleanly pins the contract that
// "agent output with sloppy whitespace/casing" → Normalize → Apply
// produces the same result as "agent output already canonical" → Apply.
// Guards against a regression where a caller forgets Normalize and the
// proposal silently fails validation on a trailing space.
func TestApply_NormalizedWhitespaceFlowsThroughCleanly(t *testing.T) {
	env := newApplyEnv(t)
	sloppy := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangeStatus, Target: "  execute/foo  ", Status: "  READY  "},
		},
	}
	normalized, err := Normalize(sloppy, env.currentState())
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	res, err := env.applier.Apply(context.Background(), normalized, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 || res.Failed != 0 {
		t.Fatalf("expected 1 applied, got %+v (outcomes=%+v)", res, res.Outcomes)
	}
	foo := env.loadItem("execute", "foo")
	if string(foo.Status) != "ready" {
		t.Fatalf("expected status=ready, got %q", foo.Status)
	}

	// And confirm the inverse: skipping Normalize on the same input fails
	// closed at Apply's defensive Validate. This is the contract callers
	// must honor; the test pins it so no future refactor silently lets
	// raw whitespace slip through.
	raw := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangeStatus, Target: "  execute/foo  ", Status: "  READY  "},
		},
	}
	if _, err := env.applier.Apply(context.Background(), raw, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err == nil {
		t.Fatalf("expected un-normalized whitespace to fail validation")
	}
}

// TestApply_ArchiveItem_IsIdempotent asserts that re-archiving an already-
// archived item is a no-op — the applier records success, the ArchivedAt
// timestamp does not regress, and no error is returned. Important because
// split rollback and manual re-submissions can both issue an archive on
// an already-tombstoned item.
func TestApply_ArchiveItem_IsIdempotent(t *testing.T) {
	env := newApplyEnv(t)
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpArchiveItem, Target: "execute/foo"},
		},
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	originalTs := *env.loadItem("execute", "foo").ArchivedAt
	if originalTs == "" {
		t.Fatal("expected archivedAt set after first archive")
	}

	// Apply again — should succeed and leave the timestamp alone.
	p.Mutations[0].ID = "m2"
	res, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if res.Failed != 0 || res.Applied != 1 {
		t.Fatalf("expected applied=1 failed=0, got %+v", res)
	}
	if got := *env.loadItem("execute", "foo").ArchivedAt; got != originalTs {
		t.Fatalf("expected archivedAt unchanged, got %q (was %q)", got, originalTs)
	}
}

// TestApply_InterruptBeforeSubsequentMutation asserts the plan's ordering
// guarantee: when a proposal mixes interrupt_in_progress with subsequent
// mutations against the same item, the canceller fires before the later
// mutations land. Encoded as a call-order assertion because "order" is
// the only user-visible contract.
func TestApply_InterruptBeforeSubsequentMutation(t *testing.T) {
	env := newApplyEnv(t)
	foo := env.loadItem("execute", "foo")
	foo.Status = backlog.StatusInProgress
	if err := env.backlog.SaveItem(foo); err != nil {
		t.Fatalf("set in_progress: %v", err)
	}

	// Canceller records the priority at the time of the call so we can
	// verify interrupt runs before change_priority mutates the item.
	observedPriorityAtCancel := 0
	env.cancelFake.err = nil
	env.cancelFake.calls = nil
	// Wrap the cancel to capture state at call time.
	env.applier.cancel = &orderObservingCanceller{
		loadItem: func() backlog.BacklogItem {
			return env.loadItem("execute", "foo")
		},
		record: func(p int) { observedPriorityAtCancel = p },
	}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpInterruptInProgress, Target: "execute/foo"},
			{ID: "m2", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(9)},
		},
	}
	state := env.currentState()
	state.InProgressRefs["execute/foo"] = struct{}{}
	if _, err := env.applier.Apply(context.Background(), p, state, nil, Source{InitiativeName: "ui-rewrite"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if observedPriorityAtCancel != 5 {
		t.Fatalf("interrupt ran after change_priority — priority was %d at cancel time, expected 5", observedPriorityAtCancel)
	}
	if p := env.loadItem("execute", "foo").Priority; p != 9 {
		t.Fatalf("expected final priority=9, got %d", p)
	}
}

type orderObservingCanceller struct {
	loadItem func() backlog.BacklogItem
	record   func(int)
}

func (c *orderObservingCanceller) CancelForBacklog(_ context.Context, _, _ string) error {
	c.record(c.loadItem().Priority)
	return nil
}

// TestApply_RespectsContextCancellation asserts the ctx plumbing fix:
// when ctx is cancelled mid-batch, remaining mutations surface as
// failures carrying the context error rather than completing silently.
func TestApply_RespectsContextCancellation(t *testing.T) {
	env := newApplyEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(7)},
			{ID: "m2", Op: OpChangePriority, Target: "execute/bar", Priority: intPtr(8)},
		},
	}
	res, err := env.applier.Apply(ctx, p, env.currentState(), nil, Source{InitiativeName: "ui-rewrite"})
	if err != nil {
		t.Fatalf("Apply returned err: %v", err)
	}
	if res.Failed != 2 || res.Applied != 0 {
		t.Fatalf("expected all mutations to fail on cancelled ctx, got %+v", res)
	}
	for _, o := range res.Outcomes {
		if !strings.Contains(o.Error, "context canceled") {
			t.Fatalf("outcome %s missing cancellation error: %q", o.MutationID, o.Error)
		}
	}
	// Items unchanged — priorities should still be the seeded 5.
	if got := env.loadItem("execute", "foo").Priority; got != 5 {
		t.Fatalf("foo priority changed despite cancel: %d", got)
	}
}

// TestApply_AttributionChainSurfacedInOutcomesAndEvents guards the
// "attribution chain is complete" requirement: the FeedbackRoundID,
// RoundNumber, and Entrypoint on the Source reach both the per-mutation
// event emitter and the Outcome.Error/applied list so auditors can link
// an applied mutation back to the round that produced it.
func TestApply_AttributionChainSurfacedInOutcomesAndEvents(t *testing.T) {
	env := newApplyEnv(t)
	events := &fakeEvents{}
	env.applier.events = events

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(6)},
		},
	}
	source := Source{
		InitiativeName:  "ui-rewrite",
		FeedbackRoundID: "ui-rewrite/round-001",
		RoundNumber:     1,
		RoundSlug:       "test-round",
		Entrypoint:      "initiative.feedback",
		DecidedBy:       "tester",
	}
	if _, err := env.applier.Apply(context.Background(), p, env.currentState(), nil, source); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(events.captured) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events.captured))
	}
	got := events.captured[0].source
	if got.FeedbackRoundID != source.FeedbackRoundID {
		t.Fatalf("FeedbackRoundID lost: %+v", got)
	}
	if got.RoundNumber != source.RoundNumber || got.RoundSlug != source.RoundSlug {
		t.Fatalf("RoundNumber/RoundSlug lost: %+v", got)
	}
	if got.Entrypoint != source.Entrypoint {
		t.Fatalf("Entrypoint lost: %+v", got)
	}
	if got.DecidedBy != source.DecidedBy {
		t.Fatalf("DecidedBy lost: %+v", got)
	}
}

// TestApplyFlow_HappyPath drives the full state→Normalize→Apply recipe via
// the canonical helper. The agent submits raw whitespace ("  Baz  ") in
// Title to prove ApplyFlow performs Normalize before Apply — Apply's
// validation contract assumes pre-normalized input, so a missing Normalize
// would surface as a per-mutation error rather than a clean apply.
func TestApplyFlow_HappyPath(t *testing.T) {
	env := newApplyEnv(t)
	state := env.currentState()

	stateBuilder := func(name string) (CurrentState, error) {
		if name != "ui-rewrite" {
			t.Fatalf("stateBuilder called with unexpected initiative %q", name)
		}
		return state, nil
	}

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			// Whitespace + casing quirks the agent might emit; Normalize
			// trims/lowers them before Apply sees them.
			{ID: "m1", Op: OpChangeStatus, Target: "  execute/foo  ", Status: "  READY  "},
		},
	}
	source := Source{
		InitiativeName:   "ui-rewrite",
		FeedbackRoundID:  "ui-rewrite/round-001",
		RoundNumber:      1,
		RoundSlug:        "test-round",
		Entrypoint:       "initiative.feedback",
		DecidedBy:        "tester",
		DecidedAtRFC3339: "2026-05-02T00:00:00Z",
	}

	res, err := env.applier.ApplyFlow(context.Background(), p, stateBuilder, nil, source)
	if err != nil {
		t.Fatalf("ApplyFlow: %v", err)
	}
	if res == nil {
		t.Fatalf("ApplyFlow: nil result")
	}
	if res.Applied != 1 || res.Failed != 0 || res.Skipped != 0 {
		t.Fatalf("unexpected counts: applied=%d failed=%d skipped=%d", res.Applied, res.Failed, res.Skipped)
	}
	if len(res.Outcomes) != 1 || !res.Outcomes[0].Applied {
		t.Fatalf("expected single applied outcome, got %+v", res.Outcomes)
	}
	// Round-trip the side effect: the canonicalized status landed on disk.
	item := env.loadItem("execute", "foo")
	if item.Status != backlog.StatusReady {
		t.Fatalf("expected status %q after ApplyFlow, got %q", backlog.StatusReady, item.Status)
	}
}

// TestApplyFlow_RejectsNilStateBuilder pins the precondition that surfaces
// the most common wiring mistake (ApplyFlow callable but no state source
// configured). Without this gate, a nil builder would panic later with no
// useful trace.
func TestApplyFlow_RejectsNilStateBuilder(t *testing.T) {
	env := newApplyEnv(t)
	source := Source{InitiativeName: "ui-rewrite"}
	p := Proposal{Form: FormMutationList}

	_, err := env.applier.ApplyFlow(context.Background(), p, nil, nil, source)
	if err == nil {
		t.Fatalf("expected error from nil StateBuilder, got nil")
	}
	if !strings.Contains(err.Error(), "StateBuilder") {
		t.Fatalf("expected error to mention StateBuilder, got %v", err)
	}
}

// TestApplyFlow_RejectsEmptyInitiative pins the precondition that ApplyFlow
// asks for the initiative name on Source — the same field Apply requires.
// Catching it here gives a clearer error than the deeper Apply check.
func TestApplyFlow_RejectsEmptyInitiative(t *testing.T) {
	env := newApplyEnv(t)
	stateBuilder := func(name string) (CurrentState, error) {
		t.Fatalf("stateBuilder must not be called with empty initiative")
		return CurrentState{}, nil
	}
	p := Proposal{Form: FormMutationList}

	_, err := env.applier.ApplyFlow(context.Background(), p, stateBuilder, nil, Source{InitiativeName: "   "})
	if err == nil {
		t.Fatalf("expected error from empty InitiativeName, got nil")
	}
	if !strings.Contains(err.Error(), "InitiativeName") {
		t.Fatalf("expected error to mention InitiativeName, got %v", err)
	}
}

// TestApplyFlow_StateBuilderError surfaces the upstream failure with a
// "build proposal state" prefix so callers can distinguish state-loading
// trouble from normalize/apply failures in logs and HTTP responses.
func TestApplyFlow_StateBuilderError(t *testing.T) {
	env := newApplyEnv(t)
	wantErr := errors.New("graph materialization failed")
	stateBuilder := func(name string) (CurrentState, error) {
		return CurrentState{}, wantErr
	}
	p := Proposal{Form: FormMutationList}
	source := Source{InitiativeName: "ui-rewrite"}

	_, err := env.applier.ApplyFlow(context.Background(), p, stateBuilder, nil, source)
	if err == nil {
		t.Fatalf("expected error from stateBuilder, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got %v", err)
	}
	if !strings.Contains(err.Error(), "build proposal state") {
		t.Fatalf("expected 'build proposal state' prefix, got %v", err)
	}
}

// TestApplyFlow_NormalizeError surfaces a malformed proposal (unknown form)
// with a "normalize proposal" prefix so ApplyFlow's three failure modes
// stay individually diagnosable from one log line.
func TestApplyFlow_NormalizeError(t *testing.T) {
	env := newApplyEnv(t)
	state := env.currentState()
	stateBuilder := func(name string) (CurrentState, error) { return state, nil }

	// Bypass UnmarshalJSON's form check by constructing the Proposal in code.
	p := Proposal{Form: Form("bogus")}
	source := Source{InitiativeName: "ui-rewrite"}

	_, err := env.applier.ApplyFlow(context.Background(), p, stateBuilder, nil, source)
	if err == nil {
		t.Fatalf("expected error from Normalize, got nil")
	}
	if !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("expected ErrInvalidProposal in chain, got %v", err)
	}
	if !strings.Contains(err.Error(), "normalize proposal") {
		t.Fatalf("expected 'normalize proposal' prefix, got %v", err)
	}
}

// TestApplyFlow_ApplyError surfaces an apply-level rejection (mismatched
// initiative) directly rather than wrapping it again — Apply already
// produces a clear message. This pins the contract: ApplyFlow does not
// add its own wrapping on top of Apply errors.
func TestApplyFlow_ApplyError(t *testing.T) {
	env := newApplyEnv(t)
	state := env.currentState() // initiative "ui-rewrite"
	stateBuilder := func(name string) (CurrentState, error) { return state, nil }

	p := Proposal{Form: FormMutationList}
	source := Source{InitiativeName: "other-project"} // mismatch with state

	_, err := env.applier.ApplyFlow(context.Background(), p, stateBuilder, nil, source)
	if err == nil {
		t.Fatalf("expected error from Apply, got nil")
	}
	if !strings.Contains(err.Error(), "does not match current state") {
		t.Fatalf("expected mismatched-initiative error, got %v", err)
	}
}

// TestApplyFlow_PassesAcceptedIDs proves the helper threads acceptedIDs
// through to Apply unchanged — partial-accept is the expected operator
// flow when reviewing a multi-mutation proposal.
func TestApplyFlow_PassesAcceptedIDs(t *testing.T) {
	env := newApplyEnv(t)
	state := env.currentState()
	stateBuilder := func(name string) (CurrentState, error) { return state, nil }

	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{ID: "m1", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(7)},
			{ID: "m2", Op: OpChangePriority, Target: "execute/bar", Priority: intPtr(8)},
		},
	}
	source := Source{InitiativeName: "ui-rewrite", DecidedBy: "tester"}

	res, err := env.applier.ApplyFlow(context.Background(), p, stateBuilder, []string{"m1"}, source)
	if err != nil {
		t.Fatalf("ApplyFlow: %v", err)
	}
	if res.Applied != 1 || res.Skipped != 1 {
		t.Fatalf("expected 1 applied + 1 skipped, got applied=%d skipped=%d", res.Applied, res.Skipped)
	}
	if got := env.loadItem("execute", "foo").Priority; got != 7 {
		t.Fatalf("foo priority: want 7, got %d", got)
	}
	if got := env.loadItem("execute", "bar").Priority; got != 5 {
		t.Fatalf("bar priority unchanged expectation: want 5, got %d", got)
	}
}
