package records

import (
	"testing"
	"time"
)

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	return NewFileStore(t.TempDir())
}

func mkRecord(id, scenario string, kind RecordKind) Record {
	return Record{
		ID:        id,
		Kind:      kind,
		Scenario:  scenario,
		Trigger:   "t",
		Approach:  "a",
		Outcome:   OutcomeShipped,
		CreatedAt: time.Now().UTC(),
	}
}

func TestFileStoreCreateGet(t *testing.T) {
	s := newTestStore(t)
	r := mkRecord("rec-aaa", "audio-tools", KindFix)
	if err := s.Create(r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get("rec-aaa")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != r.ID || got.Scenario != r.Scenario || got.Kind != r.Kind {
		t.Errorf("Get returned wrong record: %+v", got)
	}
}

func TestFileStoreCreateRejectsDuplicate(t *testing.T) {
	s := newTestStore(t)
	r := mkRecord("rec-dup", "x", KindFix)
	if err := s.Create(r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Create(r); err == nil {
		t.Errorf("expected duplicate Create to fail")
	}
}

func TestFileStoreCreateRejectsBadID(t *testing.T) {
	s := newTestStore(t)
	for _, bad := range []string{"", "../escape", "a/b", "has space"} {
		r := mkRecord(bad, "x", KindFix)
		if err := s.Create(r); err == nil {
			t.Errorf("Create with bad id %q should fail", bad)
		}
	}
}

func TestFileStoreListFilters(t *testing.T) {
	s := newTestStore(t)
	must := func(r Record) {
		if err := s.Create(r); err != nil {
			t.Fatalf("Create %s: %v", r.ID, err)
		}
	}
	must(mkRecord("rec-a", "alpha", KindFix))
	must(mkRecord("rec-b", "alpha", KindResearch))
	must(mkRecord("rec-c", "beta", KindFix))

	all, err := s.List(ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 records, got %d", len(all))
	}

	alphaOnly, _ := s.List(ListFilter{Scenario: "alpha"})
	if len(alphaOnly) != 2 {
		t.Errorf("Scenario filter: expected 2, got %d", len(alphaOnly))
	}

	fixOnly, _ := s.List(ListFilter{Kind: KindFix})
	if len(fixOnly) != 2 {
		t.Errorf("Kind filter: expected 2, got %d", len(fixOnly))
	}
}

func TestFileStoreListHidesStubsByDefault(t *testing.T) {
	s := newTestStore(t)
	r := mkRecord("rec-stub", "x", KindFix)
	r.Stub = true
	r.Trigger = ""
	r.Approach = ""
	if err := s.Create(r); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, _ := s.List(ListFilter{})
	if len(got) != 0 {
		t.Errorf("default list should hide stubs, got %d", len(got))
	}
	withStubs, _ := s.List(ListFilter{IncludeStubs: true})
	if len(withStubs) != 1 {
		t.Errorf("IncludeStubs list: expected 1, got %d", len(withStubs))
	}
}

func TestFileStoreUpdateNarrativeFlipsStub(t *testing.T) {
	s := newTestStore(t)
	r := mkRecord("rec-fill", "x", KindFix)
	r.Stub = true
	r.Trigger = ""
	r.Approach = ""
	if err := s.Create(r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	now := time.Now().UTC()
	filled, err := s.UpdateNarrative("rec-fill", Narrative{
		Trigger:  "real trigger",
		Approach: "real approach",
		Outcome:  OutcomeShipped,
	}, now)
	if err != nil {
		t.Fatalf("UpdateNarrative: %v", err)
	}
	if filled.Stub {
		t.Errorf("Stub should flip to false after fill")
	}
	if filled.NarrativeAt.IsZero() {
		t.Errorf("NarrativeAt should be set")
	}

	// Second update must be rejected.
	if _, err := s.UpdateNarrative("rec-fill", Narrative{Trigger: "x"}, now); err != ErrStubLocked {
		t.Errorf("expected ErrStubLocked on second fill, got %v", err)
	}
}

func TestFileStoreUpdateNarrativeRequiresContent(t *testing.T) {
	s := newTestStore(t)
	r := mkRecord("rec-empty", "x", KindFix)
	r.Stub = true
	r.Trigger = ""
	r.Approach = ""
	_ = s.Create(r)
	if _, err := s.UpdateNarrative("rec-empty", Narrative{}, time.Now()); err == nil {
		t.Errorf("expected empty narrative to be rejected")
	}
}

func TestFileStoreSetSupersededBy(t *testing.T) {
	s := newTestStore(t)
	if err := s.Create(mkRecord("rec-orig", "x", KindFix)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	r, err := s.SetSupersededBy("rec-orig", "rec-new")
	if err != nil {
		t.Fatalf("SetSupersededBy: %v", err)
	}
	if r.SupersededBy != "rec-new" {
		t.Errorf("SupersededBy = %q, want rec-new", r.SupersededBy)
	}
	// Re-setting with same successor is idempotent.
	if _, err := s.SetSupersededBy("rec-orig", "rec-new"); err != nil {
		t.Errorf("idempotent re-set should succeed, got %v", err)
	}
	// Re-setting with different successor is rejected.
	if _, err := s.SetSupersededBy("rec-orig", "rec-other"); err != ErrAlreadySuperseded {
		t.Errorf("expected ErrAlreadySuperseded, got %v", err)
	}
}

func TestFileStoreGetMissing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get("rec-nope"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
