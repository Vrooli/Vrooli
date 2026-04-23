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
	applier    *Applier
	cancelFake *fakeCanceller
	schedFake  *fakeScheduler
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
	calls []string
}

func (f *fakeEvents) EmitProposalMutationApplied(source Source, m Mutation) {
	f.calls = append(f.calls, source.InitiativeName+":"+m.ID+":"+string(m.Op))
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
	applier, err := NewApplier(Config{
		Store:       store,
		Assigner:    initSvc,
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

func stringSliceContains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}
