package gates

import (
	"context"
	"errors"
	"testing"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// fakeStore is an in-memory ItemStore.
type fakeStore struct {
	items []backlog.BacklogItem
	root  string
	err   error
}

func (f *fakeStore) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func item(kind, name string, status backlog.BacklogStatus, deps ...string) backlog.BacklogItem {
	return backlog.BacklogItem{
		Kind:      backlog.BacklogKind(kind),
		Name:      name,
		Title:     name + " title",
		Status:    status,
		DependsOn: deps,
		Updated:   "2026-07-01T00:00:00Z",
	}
}

type fakePlanReader struct {
	plan *sharedv1.Plan
	err  error
}

func (f fakePlanReader) ListPlans(context.Context) ([]*sharedv1.Plan, error) { return nil, f.err }
func (f fakePlanReader) GetPlan(context.Context, string) (*sharedv1.Plan, error) {
	return f.plan, f.err
}

func acceptedItem(kind, name string, status backlog.BacklogStatus) backlog.BacklogItem {
	it := item(kind, name, status)
	it.PlanRef = &backlog.PlanRef{Provider: "plan-manager", PlanID: "plan-" + name, Slug: name, Role: backlog.PlanRefRoleExecutionSpec}
	it.PlanAcceptance = &backlog.PlanAcceptance{Actor: "operator", AcceptedAt: "2026-07-01T00:00:00Z", PlanContentHash: "sha256:current"}
	it.PlanAcceptance.SubjectVersion = backlog.PlanAcceptanceSubjectVersion(it)
	return it
}

func archived(it backlog.BacklogItem) backlog.BacklogItem {
	ts := "2026-07-01T00:00:00Z"
	it.ArchivedAt = &ts
	return it
}

// --- WorkshopSource ---------------------------------------------------------

func TestWorkshopSource_UnacceptedPlanNeedsAcceptance(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		acceptedItem("execute", "raw", backlog.StatusBacklog),
	}}
	store.items[0].PlanAcceptance = nil
	got, err := WorkshopSource{Store: store, Plans: fakePlanReader{plan: &sharedv1.Plan{ContentHash: "sha256:current", Status: sharedv1.PlanStatus_PLAN_STATUS_ACTIVE}}}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Suggested != "accept-plan" {
		t.Fatalf("expected acceptance gate, got %+v", got)
	}
}

func TestWorkshopSource_AcceptedCurrentPlanHasNoGate(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		acceptedItem("fix", "ready", backlog.StatusReady),
	}}
	got, err := WorkshopSource{Store: store, Plans: fakePlanReader{plan: &sharedv1.Plan{ContentHash: "sha256:current", Status: sharedv1.PlanStatus_PLAN_STATUS_ACTIVE}}}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("accepted current plan should have no gate, got %+v", got)
	}
}

func TestWorkshopSource_ChangedPlanNeedsFreshAcceptance(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		acceptedItem("fix", "synth", backlog.StatusReady),
	}}
	got, err := WorkshopSource{Store: store, Plans: fakePlanReader{plan: &sharedv1.Plan{ContentHash: "sha256:changed", Status: sharedv1.PlanStatus_PLAN_STATUS_ACTIVE}}}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Suggested != "accept-plan" {
		t.Fatalf("expected stale acceptance gate, got %+v", got)
	}
}

func TestWorkshopSource_AuthorsPlanWhenPlanRefMissing(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "questions", backlog.StatusBacklog),
	}}
	got, err := WorkshopSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Suggested != "author-plan" {
		t.Errorf("missing plan should produce author-plan gate, got %+v", got)
	}
}

func TestWorkshopSource_SkipsNonQueueable(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "reviewing", backlog.StatusReviewPending),
		item("fix", "running", backlog.StatusInProgress),
	}}
	got, err := WorkshopSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("non-queueable items should have no workshop gate, got %+v", got)
	}
}

// --- ReviewSource -----------------------------------------------------------

type fakeExecs struct {
	records []execution.Record
	err     error
}

func (f fakeExecs) List(_ context.Context, _ execution.ListFilters) ([]execution.Record, error) {
	return f.records, f.err
}

func TestReviewSource_ReviewPendingItem(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("execute", "waiting", backlog.StatusReviewPending),
		item("execute", "child", backlog.StatusBacklog, "execute/waiting"),
	}}
	got, err := ReviewSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 gate, got %+v", got)
	}
	if got[0].Kind != KindReview || got[0].OwnerType != "backlog" {
		t.Errorf("unexpected gate: %+v", got[0])
	}
	if len(got[0].Blocks) != 1 || got[0].Blocks[0] != "execute/child" {
		t.Errorf("unexpected blocks: %v", got[0].Blocks)
	}
}

func TestReviewSource_ExecutionNeedsReview(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "flagged", backlog.StatusInProgress),
	}}
	execs := fakeExecs{records: []execution.Record{
		{ExecutionID: "e1", BacklogKind: "fix", BacklogName: "flagged", Status: execution.StatusNeedsReview, FinishedAt: "2026-07-01T13:00:00Z"},
		{ExecutionID: "e2", BacklogKind: "fix", BacklogName: "flagged", Status: execution.StatusRunning},
	}}
	got, err := ReviewSource{Store: store, Executions: execs}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 gate, got %+v", got)
	}
	g := got[0]
	if g.OwnerType != "execution" || g.ID != "review:execution/e1" || g.DecidableSince != "2026-07-01T13:00:00Z" {
		t.Errorf("unexpected gate: %+v", g)
	}
	if g.OwnerTitle != "flagged title" {
		t.Errorf("expected owner title from backlog item, got %q", g.OwnerTitle)
	}
}

func TestReviewSource_SkipsArchivedOwnerExecution(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		archived(item("fix", "gone", backlog.StatusCompleted)),
	}}
	execs := fakeExecs{records: []execution.Record{
		{ExecutionID: "e1", BacklogKind: "fix", BacklogName: "gone", Status: execution.StatusNeedsFixup},
	}}
	got, err := ReviewSource{Store: store, Executions: execs}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("archived owner should suppress execution gate, got %+v", got)
	}
}

// --- ClassifySource ---------------------------------------------------------

type fakeCaptures struct {
	entries []CaptureEntry
	err     error
}

func (f fakeCaptures) ListCaptures() ([]CaptureEntry, error) { return f.entries, f.err }

func TestClassifySource_ClassifiedWithItems(t *testing.T) {
	src := ClassifySource{Captures: fakeCaptures{entries: []CaptureEntry{
		{ID: "c1", Text: "do the thing", Status: "classified", ClassifiedItems: 2, CreatedAt: "2026-07-01T10:00:00Z"},
		{ID: "c2", Text: "still working", Status: "classifying", ClassifiedItems: 0},
		{ID: "c3", Text: "empty", Status: "classified", ClassifiedItems: 0},
	}}}
	got, err := src.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 gate, got %+v", got)
	}
	g := got[0]
	if g.Kind != KindClassify || g.OwnerName != "c1" || g.Count != 2 {
		t.Errorf("unexpected gate: %+v", g)
	}
}

// --- Service ----------------------------------------------------------------

type stubSource struct {
	name  string
	gates []Gate
	err   error
}

func (s stubSource) Name() string                              { return s.name }
func (s stubSource) Enumerate(context.Context) ([]Gate, error) { return s.gates, s.err }

func TestService_ConcatsAndSorts(t *testing.T) {
	svc := NewService(
		stubSource{name: "b", gates: []Gate{{ID: "review:backlog/fix/z", Kind: KindReview, OwnerType: "backlog", OwnerName: "z"}}},
		stubSource{name: "a", gates: []Gate{{ID: "decide:backlog/fix/a", Kind: KindDecide, OwnerType: "backlog", OwnerName: "a"}}},
	)
	got := svc.Enumerate(context.Background())
	if len(got) != 2 {
		t.Fatalf("expected 2 gates, got %+v", got)
	}
	if got[0].Kind != KindDecide {
		t.Errorf("expected decide first, got %+v", got[0])
	}
}

func TestService_DegradesOnSourceError(t *testing.T) {
	svc := NewService(
		stubSource{name: "broken", err: errors.New("boom")},
		stubSource{name: "ok", gates: []Gate{{ID: "decide:backlog/fix/a", Kind: KindDecide}}},
	)
	got := svc.Enumerate(context.Background())
	if len(got) != 1 {
		t.Errorf("expected surviving source's gate, got %+v", got)
	}
}
