package gates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// fakeStore is an in-memory ItemStore backed by a temp dir for workshop files.
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

func (f *fakeStore) ItemDir(kind backlog.BacklogKind, name string) string {
	return filepath.Join(f.root, string(kind), name)
}

type roundSpec struct {
	Round            int            `json:"round"`
	GeneratedAt      string         `json:"generated_at"`
	Mode             string         `json:"mode,omitempty"`
	PendingSynthesis bool           `json:"pending_synthesis,omitempty"`
	Readiness        map[string]int `json:"readiness"`
	Items            []roundItem    `json:"items"`
}

type roundItem struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Selected *string `json:"selected,omitempty"`
}

func writeRound(t *testing.T, store *fakeStore, kind, name string, spec roundSpec) {
	t.Helper()
	dir := filepath.Join(store.ItemDir(backlog.BacklogKind(kind), name), "workshop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fmt.Sprintf("round-%d.json", spec.Round))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
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

func archived(it backlog.BacklogItem) backlog.BacklogItem {
	ts := "2026-07-01T00:00:00Z"
	it.ArchivedAt = &ts
	return it
}

func strPtr(s string) *string { return &s }

func allReadiness(score int) map[string]int {
	m := make(map[string]int, len(backlog.ReadinessDimensions))
	for _, dim := range backlog.ReadinessDimensions {
		m[dim] = score
	}
	return m
}

// --- DecideSource -----------------------------------------------------------

func TestDecideSource_EnumeratesPendingDecisions(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "alpha", backlog.StatusBacklog),
		item("fix", "beta", backlog.StatusReady, "fix/alpha"),
	}}
	writeRound(t, store, "fix", "alpha", roundSpec{
		Round:       1,
		GeneratedAt: "2026-07-01T12:00:00Z",
		Readiness:   allReadiness(1),
		Items: []roundItem{
			{ID: "d1", Type: "decision"},
			{ID: "d2", Type: "decision", Selected: strPtr("A")},
			{ID: "i1", Type: "info"},
		},
	})

	got, err := DecideSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 gate, got %d: %+v", len(got), got)
	}
	g := got[0]
	if g.Kind != KindDecide || g.OwnerName != "alpha" || g.Count != 1 {
		t.Errorf("unexpected gate: %+v", g)
	}
	if g.ID != "decide:backlog/fix/alpha" {
		t.Errorf("unexpected id: %s", g.ID)
	}
	if g.DecidableSince != "2026-07-01T12:00:00Z" {
		t.Errorf("unexpected decidable_since: %s", g.DecidableSince)
	}
	if len(g.Blocks) != 1 || g.Blocks[0] != "fix/beta" {
		t.Errorf("unexpected blocks: %v", g.Blocks)
	}
}

func TestDecideSource_SkipsLockedTerminalArchived(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "locked", backlog.StatusInProgress),
		item("fix", "done", backlog.StatusCompleted),
		archived(item("fix", "gone", backlog.StatusBacklog)),
	}}
	for _, name := range []string{"locked", "done", "gone"} {
		writeRound(t, store, "fix", name, roundSpec{
			Round:     1,
			Readiness: allReadiness(1),
			Items:     []roundItem{{ID: "d1", Type: "decision"}},
		})
	}

	got, err := DecideSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no gates, got %+v", got)
	}
}

func TestDecideSource_NoRoundsNoGate(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "fresh", backlog.StatusBacklog),
	}}
	got, err := DecideSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no gates, got %+v", got)
	}
}

// --- WorkshopSource ---------------------------------------------------------

func TestWorkshopSource_UnworkshoppedItemNeedsWorkshop(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("execute", "raw", backlog.StatusBacklog),
	}}
	got, err := WorkshopSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Suggested != "workshop" {
		t.Fatalf("expected workshop gate, got %+v", got)
	}
}

func TestWorkshopSource_ReadyItemHasNoGate(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "ready", backlog.StatusReady),
	}}
	writeRound(t, store, "fix", "ready", roundSpec{
		Round:       1,
		GeneratedAt: "2026-07-01T12:00:00Z",
		Mode:        "finalize",
		Readiness:   allReadiness(3),
	})
	got, err := WorkshopSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("ready item should have no workshop gate, got %+v", got)
	}
}

func TestWorkshopSource_PendingSynthesisReadySuggestsFinalize(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "synth", backlog.StatusReady),
	}}
	writeRound(t, store, "fix", "synth", roundSpec{
		Round:            1,
		GeneratedAt:      "2026-07-01T12:00:00Z",
		PendingSynthesis: true,
		Readiness:        allReadiness(3),
	})
	got, err := WorkshopSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Suggested != "finalize" {
		t.Fatalf("expected finalize gate, got %+v", got)
	}
}

func TestWorkshopSource_PendingDecisionsTakePrecedence(t *testing.T) {
	store := &fakeStore{root: t.TempDir(), items: []backlog.BacklogItem{
		item("fix", "questions", backlog.StatusBacklog),
	}}
	writeRound(t, store, "fix", "questions", roundSpec{
		Round:     1,
		Readiness: allReadiness(1),
		Items:     []roundItem{{ID: "d1", Type: "decision"}},
	})
	got, err := WorkshopSource{Store: store}.Enumerate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("decide gate should suppress workshop gate, got %+v", got)
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
