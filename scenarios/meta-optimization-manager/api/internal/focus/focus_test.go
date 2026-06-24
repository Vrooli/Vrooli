package focus

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/spacedoc"
)

// fakeSource is an in-memory GapSource.
type fakeSource struct {
	gaps []Gap
	err  error
}

func (f *fakeSource) DerivedGaps(_ context.Context) ([]Gap, error) { return f.gaps, f.err }

// fakeRepo is an in-memory Repository.
type fakeRepo struct {
	rows map[string]Gap
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[string]Gap{}} }

func (r *fakeRepo) List(_ context.Context) ([]Gap, error) {
	out := make([]Gap, 0, len(r.rows))
	for _, g := range r.rows {
		out = append(out, g)
	}
	return out, nil
}

func (r *fakeRepo) Get(_ context.Context, id string) (Gap, bool, error) {
	g, ok := r.rows[id]
	return g, ok, nil
}

func (r *fakeRepo) Upsert(_ context.Context, g Gap) error {
	r.rows[g.ID] = g
	return nil
}

func derivedFixture() []Gap {
	return []Gap{
		{ID: "answer/1", Projection: ProjectionAnswer, Title: "explain domain map", Status: spacedoc.StatusMissing, SourceCellID: "1"},
		{ID: "validate/2", Projection: ProjectionValidate, Title: "verify perf", Status: spacedoc.StatusInReach, SourceCellID: "2"},
		{ID: "guide/3", Projection: ProjectionGuide, Title: "rename skill", Status: spacedoc.StatusInReach, SourceCellID: "3"},
	}
}

func newSvc(src GapSource, repo Repository) Service {
	return NewService(Deps{Source: src, Repo: repo})
}

func TestGetFocusRanksMissingAnswerHighest(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	items, err := svc.GetFocus(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("GetFocus: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}
	// answer/1 is MISSING (impact 1.0) × answer (importance 1.0) = 1.0 — top.
	if items[0].Gap.ID != "answer/1" {
		t.Fatalf("want answer/1 ranked first, got %s (priority=%.3f)", items[0].Gap.ID, items[0].Priority)
	}
	if items[0].Priority <= items[1].Priority {
		t.Fatalf("expected descending priority, got %.3f then %.3f", items[0].Priority, items[1].Priority)
	}
	if items[0].Rationale == "" {
		t.Fatalf("expected a rationale on the top item")
	}
}

func TestGetFocusLimitAndProjectionFilter(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	items, err := svc.GetFocus(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("GetFocus: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("limit=1 should cap to 1, got %d", len(items))
	}

	only, err := svc.GetFocus(context.Background(), 0, ProjectionGuide)
	if err != nil {
		t.Fatalf("GetFocus(guide): %v", err)
	}
	if len(only) != 1 || only[0].Gap.Projection != ProjectionGuide {
		t.Fatalf("projection filter failed: %+v", only)
	}
}

func TestGetFocusUnknownProjectionErrors(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	if _, err := svc.GetFocus(context.Background(), 0, Projection("bogus")); err == nil {
		t.Fatalf("expected error on unknown projection")
	}
}

func TestListGapsFilters(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	all, err := svc.ListGaps(context.Background(), GapFilter{})
	if err != nil || len(all) != 3 {
		t.Fatalf("ListGaps all: len=%d err=%v", len(all), err)
	}
	missing, err := svc.ListGaps(context.Background(), GapFilter{Status: spacedoc.StatusMissing})
	if err != nil || len(missing) != 1 || missing[0].ID != "answer/1" {
		t.Fatalf("status filter failed: %+v err=%v", missing, err)
	}
	byCell, err := svc.ListGaps(context.Background(), GapFilter{CellID: "2"})
	if err != nil || len(byCell) != 1 || byCell[0].ID != "validate/2" {
		t.Fatalf("cell filter failed: %+v err=%v", byCell, err)
	}
}

func TestAddGapNoteMaterializesAndAppends(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, repo)

	// First note materializes a registry row from the derived gap.
	g, err := svc.AddGapNote(context.Background(), "answer/1", "try a cartographer provider")
	if err != nil {
		t.Fatalf("AddGapNote: %v", err)
	}
	if len(g.Approaches) != 1 || g.Approaches[0] != "try a cartographer provider" {
		t.Fatalf("approach not appended: %+v", g.Approaches)
	}
	if _, ok := repo.rows["answer/1"]; !ok {
		t.Fatalf("expected registry row materialized for answer/1")
	}

	// Second note appends without clobbering; duplicates are de-duped.
	g, err = svc.AddGapNote(context.Background(), "answer/1", "or extend code-facts")
	if err != nil {
		t.Fatalf("AddGapNote 2: %v", err)
	}
	if len(g.Approaches) != 2 {
		t.Fatalf("want 2 approaches, got %+v", g.Approaches)
	}
	g, err = svc.AddGapNote(context.Background(), "answer/1", "or extend code-facts")
	if err != nil {
		t.Fatalf("AddGapNote dup: %v", err)
	}
	if len(g.Approaches) != 2 {
		t.Fatalf("duplicate approach should be de-duped, got %+v", g.Approaches)
	}
}

func TestAddGapNoteUnknownGapErrors(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, newFakeRepo())
	if _, err := svc.AddGapNote(context.Background(), "answer/999", "x"); err == nil {
		t.Fatalf("expected error adding note to unknown gap")
	}
}

func TestAddGapNoteNilRepoErrors(t *testing.T) {
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, nil)
	if _, err := svc.AddGapNote(context.Background(), "answer/1", "x"); err == nil {
		t.Fatalf("expected error when registry unavailable")
	}
}

func TestRegistryOnlyGlobalGapSurfaces(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["global-typed-contracts"] = Gap{
		ID:         "global-typed-contracts",
		Title:      "typed contracts everywhere",
		Global:     true,
		Status:     spacedoc.StatusMissing,
		Approaches: []string{"ratchet"},
	}
	svc := newSvc(&fakeSource{gaps: derivedFixture()}, repo)
	all, err := svc.ListGaps(context.Background(), GapFilter{})
	if err != nil {
		t.Fatalf("ListGaps: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("want 4 (3 derived + 1 global), got %d", len(all))
	}
	g, err := svc.GetGap(context.Background(), "global-typed-contracts")
	if err != nil || !g.Global {
		t.Fatalf("expected global gap, got %+v err=%v", g, err)
	}
}

func TestDegradesWhenSourceErrors(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["global-x"] = Gap{ID: "global-x", Title: "x", Global: true}
	svc := newSvc(&fakeSource{err: context.DeadlineExceeded}, repo)
	all, err := svc.ListGaps(context.Background(), GapFilter{})
	if err != nil {
		t.Fatalf("ListGaps should degrade, got err=%v", err)
	}
	if len(all) != 1 || all[0].ID != "global-x" {
		t.Fatalf("expected registry-only gap to survive a down source, got %+v", all)
	}
}
