package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/health"
	"agent-manager/internal/testutil"
)

func TestHealthAuditHandlerNilStoreAndInvalidQueries(t *testing.T) {
	h := NewHealthAuditHandler(nil)
	for _, path := range []string{"/models", "/runners", "/audit", "/audit?scope=bad", "/audit?since=bad", "/audit?until=bad"} {
		rw := httptest.NewRecorder()
		handler := h.GetAudit
		if path == "/models" {
			handler = h.GetModels
		} else if path == "/runners" {
			handler = h.GetRunners
		}
		handler(rw, httptest.NewRequest(http.MethodGet, path, nil))
		if path == "/audit?scope=bad" || path == "/audit?since=bad" || path == "/audit?until=bad" {
			if rw.Code != http.StatusBadRequest {
				t.Fatalf("%s status=%d", path, rw.Code)
			}
		} else if rw.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, rw.Code)
		}
	}
}

func TestHealthAuditHandlerServesPersistedModelAndRunnerRows(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	store := health.NewStore(db)
	store.RegisterRunners([]string{"codex"})
	if err := store.RecordModel(t.Context(), "codex", "gpt-test", health.StatusFailed, "rate_limit", "429", "run-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordRunner(t.Context(), "codex", health.StatusOK, "", "", "run-1"); err != nil {
		t.Fatal(err)
	}
	h := NewHealthAuditHandler(store)
	for _, handler := range []func(http.ResponseWriter, *http.Request){h.GetModels, h.GetRunners, h.GetAudit} {
		rw := httptest.NewRecorder()
		handler(rw, httptest.NewRequest(http.MethodGet, "/audit?scope=model&limit=1", nil))
		if rw.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rw.Code, rw.Body.String())
		}
	}
}
