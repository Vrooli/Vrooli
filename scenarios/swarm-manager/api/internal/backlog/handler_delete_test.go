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
