package authoring

import "testing"

type writer struct {
	calls int
	draft Draft
}

func (w *writer) WriteAgent(d Draft) error { w.calls++; w.draft = d; return nil }

// [REQ:SWBD-P1-008]
func TestDraftDoesNotWriteUntilConfirmed(t *testing.T) {
	w := &writer{}
	s := New(w)
	d, err := s.Draft("Household planning assistant")
	if err != nil {
		t.Fatal(err)
	}
	if w.calls != 0 {
		t.Fatalf("draft wrote %d times", w.calls)
	}
	if len(d.Scopes) == 0 || len(d.OwnerOnlyScopes) != 0 {
		t.Fatalf("unsafe default draft = %+v", d)
	}
	if err := s.Confirm(d); err != nil {
		t.Fatal(err)
	}
	if w.calls != 1 {
		t.Fatalf("confirm wrote %d times", w.calls)
	}
}

// [REQ:SWBD-P1-008]
func TestManualPathFailsClearlyWhenWriterUnavailable(t *testing.T) {
	d, err := New(nil).Draft("Reader")
	if err != nil {
		t.Fatal(err)
	}
	if err := New(nil).Confirm(d); err == nil {
		t.Fatal("expected unavailable writer error")
	}
}
