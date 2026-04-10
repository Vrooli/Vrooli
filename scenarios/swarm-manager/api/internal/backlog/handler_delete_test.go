package backlog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/testutil"
)

func TestDelete_Idempotent(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:        "delete-test",
		Title:       "Delete Test",
		Description: "To delete",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/delete-test", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "delete-test"})
	w := httptest.NewRecorder()

	h.Delete(w, req)
	testutil.AssertStatus(t, w, http.StatusNoContent)

	req2 := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/delete-test", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"kind": "idea", "name": "delete-test"})
	w2 := httptest.NewRecorder()

	h.Delete(w2, req2)
	testutil.AssertStatus(t, w2, http.StatusNoContent)
}

func TestDelete_CleansDependencyReferences(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Create a dependency target and two items that depend on it.
	dep := BacklogItem{
		Name:     "dep-target",
		Title:    "Dependency Target",
		Status:   StatusBacklog,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-01-28T00:00:00Z",
		Updated:  "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindFix, dep)

	downstream1 := BacklogItem{
		Name:      "downstream-1",
		Title:     "Downstream 1",
		Status:    StatusBacklog,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"fix/dep-target", "idea/other-dep"},
		Created:   "2026-01-28T00:00:00Z",
		Updated:   "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindExecute, downstream1)

	downstream2 := BacklogItem{
		Name:      "downstream-2",
		Title:     "Downstream 2",
		Status:    StatusBacklog,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"fix/dep-target"},
		Created:   "2026-01-28T00:00:00Z",
		Updated:   "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, downstream2)

	// Also create an unrelated item to verify it's untouched.
	unrelated := BacklogItem{
		Name:      "unrelated",
		Title:     "Unrelated",
		Status:    StatusBacklog,
		Priority:  3,
		Tags:      []string{},
		DependsOn: []string{"idea/something-else"},
		Created:   "2026-01-28T00:00:00Z",
		Updated:   "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindChore, unrelated)

	// Delete the dependency target.
	req := httptest.NewRequest("DELETE", "/api/v1/backlog/fix/dep-target", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "fix", "name": "dep-target"})
	w := httptest.NewRecorder()

	h.Delete(w, req)
	testutil.AssertStatus(t, w, http.StatusNoContent)

	store := NewFileStore(rootDir)

	// downstream-1 should have the deleted ref removed but keep other deps.
	d1, err := store.LoadItem(KindExecute, "downstream-1")
	if err != nil {
		t.Fatalf("failed to load downstream-1: %v", err)
	}
	if len(d1.DependsOn) != 1 || d1.DependsOn[0] != "idea/other-dep" {
		t.Errorf("expected downstream-1 depends_on=[idea/other-dep], got %v", d1.DependsOn)
	}

	// downstream-2 should have no deps remaining.
	d2, err := store.LoadItem(KindIdea, "downstream-2")
	if err != nil {
		t.Fatalf("failed to load downstream-2: %v", err)
	}
	if len(d2.DependsOn) != 0 {
		t.Errorf("expected downstream-2 depends_on=[], got %v", d2.DependsOn)
	}

	// unrelated should be untouched.
	u, err := store.LoadItem(KindChore, "unrelated")
	if err != nil {
		t.Fatalf("failed to load unrelated: %v", err)
	}
	if len(u.DependsOn) != 1 || u.DependsOn[0] != "idea/something-else" {
		t.Errorf("expected unrelated depends_on=[idea/something-else], got %v", u.DependsOn)
	}
}
