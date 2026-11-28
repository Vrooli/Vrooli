package spec

import "testing"

func TestIndexPageRefsByIDPreservesValuesAndLastDuplicate(t *testing.T) {
	refs := []DocumentRef{
		{ID: "intro", Path: "first.json", Title: "First", Status: "draft"},
		{ID: "plans", Path: "plans.json", Title: "Plans", Status: "active"},
		{ID: "intro", Path: "updated.json", Title: "Updated", Status: "active"},
	}
	got := IndexPageRefsByID(refs)
	if len(got) != 2 {
		t.Fatalf("IndexPageRefsByID returned %d entries, want 2", len(got))
	}
	if want := (DocumentRef{ID: "intro", Path: "updated.json", Title: "Updated", Status: "active"}); got["intro"] != want {
		t.Fatalf("indexed intro = %#v, want %#v", got["intro"], want)
	}
	if got["missing"].ID != "" {
		t.Fatalf("missing ID unexpectedly indexed: %#v", got["missing"])
	}
}

func TestIndexPageStatusesPreservesLastDuplicateAndEmptyMap(t *testing.T) {
	got := IndexPageStatuses([]DocumentRef{
		{ID: "intro", Status: "draft"},
		{ID: "plans", Status: "active"},
		{ID: "intro", Status: "deprecated"},
	})
	if len(got) != 2 || got["intro"] != "deprecated" || got["plans"] != "active" {
		t.Fatalf("IndexPageStatuses = %#v, want intro=deprecated and plans=active", got)
	}
	if empty := IndexPageStatuses(nil); empty == nil || len(empty) != 0 {
		t.Fatalf("IndexPageStatuses(nil) = %#v, want a non-nil empty map", empty)
	}
}
