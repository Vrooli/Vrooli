package backlog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
)

// TestDelete_CascadesMilestoneMembership verifies that deleting an item
// removes the ref from both other items' depends_on AND the enclosing
// milestone's items[] list in a single atomic operation.
func TestDelete_CascadesMilestoneMembership(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockMilestoneAssigner()
	ia.snapshots["my-init"] = MilestoneSnapshot{
		Name:   "my-init",
		Title:  "My Milestone",
		Status: "active",
		Items:  []string{"idea/to-delete", "idea/to-keep"},
	}
	h.SetMilestoneAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "to-delete",
		Title:     "To Delete",
		Status:    StatusBacklog,
		Priority:  3,
		Tags:      []string{},
		Milestone: "my-init",
		Created:   "2026-04-21T00:00:00Z",
		Updated:   "2026-04-21T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:      "to-keep",
		Title:     "To Keep",
		Status:    StatusBacklog,
		Priority:  3,
		Tags:      []string{},
		Milestone: "my-init",
		Created:   "2026-04-21T00:00:00Z",
		Updated:   "2026-04-21T00:00:00Z",
	})

	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/to-delete", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "to-delete"})
	w := httptest.NewRecorder()
	h.Delete(w, req)
	testutil.AssertStatus(t, w, http.StatusNoContent)

	snap, ok := ia.snapshots["my-init"]
	if !ok {
		t.Fatalf("milestone missing after delete cascade")
	}
	if got := snap.Items; len(got) != 1 || got[0] != "idea/to-keep" {
		t.Errorf("milestone.items[] should contain only idea/to-keep after cascade, got %v", got)
	}
}

// TestDelete_NoMilestone_SkipsForgetItem verifies delete still works when
// the item has no milestone and the cascade harmlessly skips ForgetItem.
func TestDelete_NoMilestone_SkipsForgetItem(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockMilestoneAssigner()
	h.SetMilestoneAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "orphan", Title: "Orphan", Status: StatusBacklog, Priority: 3, Tags: []string{},
		Created: "2026-04-21T00:00:00Z", Updated: "2026-04-21T00:00:00Z",
	})

	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/orphan", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "orphan"})
	w := httptest.NewRecorder()
	h.Delete(w, req)
	testutil.AssertStatus(t, w, http.StatusNoContent)
}

// TestPatch_MoveMilestone_SyncsBothSides verifies moving an item between
// milestones updates both old and new milestones' items[] lists.
func TestPatch_MoveMilestone_SyncsBothSides(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockMilestoneAssigner()
	ia.snapshots["old-init"] = MilestoneSnapshot{
		Name: "old-init", Title: "Old", Status: "active",
		Items: []string{"idea/the-item"},
	}
	ia.snapshots["new-init"] = MilestoneSnapshot{
		Name: "new-init", Title: "New", Status: "active",
		Items: []string{},
	}
	h.SetMilestoneAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "the-item", Title: "The Item", Status: StatusBacklog, Priority: 3, Tags: []string{},
		Milestone: "old-init",
		Created:   "2026-04-21T00:00:00Z", Updated: "2026-04-21T00:00:00Z",
	})

	body := bytes.NewBufferString(`{"milestone":"new-init"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/the-item", body)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "the-item"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	testutil.AssertStatus(t, w, http.StatusOK)

	if got := ia.snapshots["old-init"].Items; len(got) != 0 {
		t.Errorf("old-init.items[] should be empty after move, got %v", got)
	}
	if got := ia.snapshots["new-init"].Items; len(got) != 1 || got[0] != "idea/the-item" {
		t.Errorf("new-init.items[] should contain idea/the-item after move, got %v", got)
	}
}

// TestPatch_MoveMilestone_UnknownTarget_Rejected verifies a move to a
// non-existent milestone is rejected with 400 and leaves state untouched.
func TestPatch_MoveMilestone_UnknownTarget_Rejected(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockMilestoneAssigner()
	ia.snapshots["old-init"] = MilestoneSnapshot{
		Name: "old-init", Title: "Old", Status: "active",
		Items: []string{"idea/the-item"},
	}
	h.SetMilestoneAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "the-item", Title: "The Item", Status: StatusBacklog, Priority: 3, Tags: []string{},
		Milestone: "old-init",
		Created:   "2026-04-21T00:00:00Z", Updated: "2026-04-21T00:00:00Z",
	})

	body := bytes.NewBufferString(`{"milestone":"does-not-exist"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/the-item", body)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "the-item"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	testutil.AssertStatus(t, w, http.StatusBadRequest)

	if got := ia.snapshots["old-init"].Items; len(got) != 1 || got[0] != "idea/the-item" {
		t.Errorf("old-init.items[] should be untouched on reject, got %v", got)
	}
}

// TestPatch_ClearMilestone_RemovesFromOldItems verifies clearing the
// milestone field removes the ref from the old milestone.
func TestPatch_ClearMilestone_RemovesFromOldItems(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockMilestoneAssigner()
	ia.snapshots["old-init"] = MilestoneSnapshot{
		Name: "old-init", Title: "Old", Status: "active",
		Items: []string{"idea/the-item"},
	}
	h.SetMilestoneAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "the-item", Title: "The Item", Status: StatusBacklog, Priority: 3, Tags: []string{},
		Milestone: "old-init",
		Created:   "2026-04-21T00:00:00Z", Updated: "2026-04-21T00:00:00Z",
	})

	body := bytes.NewBufferString(`{"milestone":""}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/the-item", body)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "the-item"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	testutil.AssertStatus(t, w, http.StatusOK)

	if got := ia.snapshots["old-init"].Items; len(got) != 0 {
		t.Errorf("old-init.items[] should be empty after clear, got %v", got)
	}
}

// TestCreate_AttachesToMilestoneMembership verifies a create with an
// milestone field also appends the ref to the milestone's items[].
func TestCreate_AttachesToMilestoneMembership(t *testing.T) {
	h, _ := setupTestHandler(t)
	ia := newMockMilestoneAssigner()
	ia.snapshots["my-init"] = MilestoneSnapshot{
		Name: "my-init", Title: "My Init", Status: "active", Items: []string{},
	}
	h.SetMilestoneAssigner(ia)

	body := bytes.NewBufferString(`{"name":"fresh-item","title":"Fresh Item","kind":"idea","milestone":"my-init"}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog", body)
	w := httptest.NewRecorder()
	h.Create(w, req)
	testutil.AssertStatus(t, w, http.StatusCreated)

	if got := ia.snapshots["my-init"].Items; len(got) != 1 || got[0] != "idea/fresh-item" {
		t.Errorf("my-init.items[] should contain idea/fresh-item after create, got %v", got)
	}
}

// TestCreate_UnknownMilestone_Rejected verifies a create with a non-existent
// milestone is rejected (validates via MilestoneAssigner.Get).
func TestCreate_UnknownMilestone_Rejected(t *testing.T) {
	h, _ := setupTestHandler(t)
	ia := newMockMilestoneAssigner()
	h.SetMilestoneAssigner(ia)

	body := bytes.NewBufferString(`{"name":"orphan","title":"Orphan","kind":"idea","milestone":"does-not-exist"}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog", body)
	w := httptest.NewRecorder()
	h.Create(w, req)
	testutil.AssertStatus(t, w, http.StatusBadRequest)

	if got := w.Body.String(); !strings.Contains(got, "does not exist") {
		t.Errorf("expected 'does not exist' in response, got %q", got)
	}
}
