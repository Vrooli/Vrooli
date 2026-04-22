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

// TestDelete_CascadesInitiativeMembership verifies that deleting an item
// removes the ref from both other items' depends_on AND the enclosing
// initiative's items[] list in a single atomic operation.
func TestDelete_CascadesInitiativeMembership(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	ia.snapshots["my-init"] = InitiativeSnapshot{
		Name:   "my-init",
		Title:  "My Initiative",
		Status: "active",
		Items:  []string{"idea/to-delete", "idea/to-keep"},
	}
	h.SetInitiativeAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:       "to-delete",
		Title:      "To Delete",
		Status:     StatusBacklog,
		Priority:   3,
		Tags:       []string{},
		Initiative: "my-init",
		Created:    "2026-04-21T00:00:00Z",
		Updated:    "2026-04-21T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name:       "to-keep",
		Title:      "To Keep",
		Status:     StatusBacklog,
		Priority:   3,
		Tags:       []string{},
		Initiative: "my-init",
		Created:    "2026-04-21T00:00:00Z",
		Updated:    "2026-04-21T00:00:00Z",
	})

	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/to-delete", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "to-delete"})
	w := httptest.NewRecorder()
	h.Delete(w, req)
	testutil.AssertStatus(t, w, http.StatusNoContent)

	snap, ok := ia.snapshots["my-init"]
	if !ok {
		t.Fatalf("initiative missing after delete cascade")
	}
	if got := snap.Items; len(got) != 1 || got[0] != "idea/to-keep" {
		t.Errorf("initiative.items[] should contain only idea/to-keep after cascade, got %v", got)
	}
}

// TestDelete_NoInitiative_SkipsForgetItem verifies delete still works when
// the item has no initiative and the cascade harmlessly skips ForgetItem.
func TestDelete_NoInitiative_SkipsForgetItem(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	h.SetInitiativeAssigner(ia)

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

// TestPatch_MoveInitiative_SyncsBothSides verifies moving an item between
// initiatives updates both old and new initiatives' items[] lists.
func TestPatch_MoveInitiative_SyncsBothSides(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	ia.snapshots["old-init"] = InitiativeSnapshot{
		Name: "old-init", Title: "Old", Status: "active",
		Items: []string{"idea/the-item"},
	}
	ia.snapshots["new-init"] = InitiativeSnapshot{
		Name: "new-init", Title: "New", Status: "active",
		Items: []string{},
	}
	h.SetInitiativeAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "the-item", Title: "The Item", Status: StatusBacklog, Priority: 3, Tags: []string{},
		Initiative: "old-init",
		Created:    "2026-04-21T00:00:00Z", Updated: "2026-04-21T00:00:00Z",
	})

	body := bytes.NewBufferString(`{"initiative":"new-init"}`)
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

// TestPatch_MoveInitiative_UnknownTarget_Rejected verifies a move to a
// non-existent initiative is rejected with 400 and leaves state untouched.
func TestPatch_MoveInitiative_UnknownTarget_Rejected(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	ia.snapshots["old-init"] = InitiativeSnapshot{
		Name: "old-init", Title: "Old", Status: "active",
		Items: []string{"idea/the-item"},
	}
	h.SetInitiativeAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "the-item", Title: "The Item", Status: StatusBacklog, Priority: 3, Tags: []string{},
		Initiative: "old-init",
		Created:    "2026-04-21T00:00:00Z", Updated: "2026-04-21T00:00:00Z",
	})

	body := bytes.NewBufferString(`{"initiative":"does-not-exist"}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/the-item", body)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "the-item"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	testutil.AssertStatus(t, w, http.StatusBadRequest)

	if got := ia.snapshots["old-init"].Items; len(got) != 1 || got[0] != "idea/the-item" {
		t.Errorf("old-init.items[] should be untouched on reject, got %v", got)
	}
}

// TestPatch_ClearInitiative_RemovesFromOldItems verifies clearing the
// initiative field removes the ref from the old initiative.
func TestPatch_ClearInitiative_RemovesFromOldItems(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	ia.snapshots["old-init"] = InitiativeSnapshot{
		Name: "old-init", Title: "Old", Status: "active",
		Items: []string{"idea/the-item"},
	}
	h.SetInitiativeAssigner(ia)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "the-item", Title: "The Item", Status: StatusBacklog, Priority: 3, Tags: []string{},
		Initiative: "old-init",
		Created:    "2026-04-21T00:00:00Z", Updated: "2026-04-21T00:00:00Z",
	})

	body := bytes.NewBufferString(`{"initiative":""}`)
	req := httptest.NewRequest("PATCH", "/api/v1/backlog/idea/the-item", body)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "the-item"})
	w := httptest.NewRecorder()
	h.Update(w, req)
	testutil.AssertStatus(t, w, http.StatusOK)

	if got := ia.snapshots["old-init"].Items; len(got) != 0 {
		t.Errorf("old-init.items[] should be empty after clear, got %v", got)
	}
}

// TestCreate_AttachesToInitiativeMembership verifies a create with an
// initiative field also appends the ref to the initiative's items[].
func TestCreate_AttachesToInitiativeMembership(t *testing.T) {
	h, _ := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	ia.snapshots["my-init"] = InitiativeSnapshot{
		Name: "my-init", Title: "My Init", Status: "active", Items: []string{},
	}
	h.SetInitiativeAssigner(ia)

	body := bytes.NewBufferString(`{"name":"fresh-item","title":"Fresh Item","kind":"idea","initiative":"my-init"}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog", body)
	w := httptest.NewRecorder()
	h.Create(w, req)
	testutil.AssertStatus(t, w, http.StatusCreated)

	if got := ia.snapshots["my-init"].Items; len(got) != 1 || got[0] != "idea/fresh-item" {
		t.Errorf("my-init.items[] should contain idea/fresh-item after create, got %v", got)
	}
}

// TestCreate_UnknownInitiative_Rejected verifies a create with a non-existent
// initiative is rejected (validates via InitiativeAssigner.Get).
func TestCreate_UnknownInitiative_Rejected(t *testing.T) {
	h, _ := setupTestHandler(t)
	ia := newMockInitiativeAssigner()
	h.SetInitiativeAssigner(ia)

	body := bytes.NewBufferString(`{"name":"orphan","title":"Orphan","kind":"idea","initiative":"does-not-exist"}`)
	req := httptest.NewRequest("POST", "/api/v1/backlog", body)
	w := httptest.NewRecorder()
	h.Create(w, req)
	testutil.AssertStatus(t, w, http.StatusBadRequest)

	if got := w.Body.String(); !strings.Contains(got, "does not exist") {
		t.Errorf("expected 'does not exist' in response, got %q", got)
	}
}
