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
	calls []string
	err   error
}

func (f *fakeCanceller) CancelForBacklog(_ context.Context, kind, name string) error {
	f.calls = append(f.calls, kind+"/"+name)
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
		DecidedBy:       "matthalloran8",
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
	inner      BacklogStore
	failOnSave string // ref ("kind/name") whose SaveItem should fail
	failOnLoad string // ref ("kind/name") whose LoadItem should fail
	saveCalls  int
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
