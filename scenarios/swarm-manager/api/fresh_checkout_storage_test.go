package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestFreshCheckout_ServesDomainDataFromStorage (T-E1): a fresh checkout carries
// no domain-data folders under the scenario source root. The running scenario
// creates and serves them from the api-core/storage data base instead, proving
// the cutover is complete (no scenarioRoot-rooted runtime data).
func TestFreshCheckout_ServesDomainDataFromStorage(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	// Fresh-checkout invariant: the scenario source root has no domain folders.
	for _, d := range []string{"ideas", "execute", "fix", "chore", "research", "initiatives", "agent-sessions", "captures"} {
		if _, err := os.Stat(filepath.Join(srv.scenarioRoot, d)); err == nil {
			t.Fatalf("scenario source root unexpectedly contains domain folder %q", d)
		}
	}

	// Create a work item through the API.
	mustPost(t, h, "/api/v1/backlog", map[string]any{
		"kind":     "execute",
		"name":     "e2e-store",
		"title":    "E2E store",
		"priority": 5,
	})

	// It is served back through the API.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/backlog/execute/e2e-store", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET created item: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// It physically lives under the storage data base, NOT the scenario root.
	dataRoot := testDataRoot(t)
	if _, err := os.Stat(filepath.Join(dataRoot, "execute", "e2e-store", "spec.json")); err != nil {
		t.Fatalf("expected spec.json under storage data base %q: %v", dataRoot, err)
	}
	if _, err := os.Stat(filepath.Join(srv.scenarioRoot, "execute", "e2e-store")); err == nil {
		t.Fatalf("item must not be written under the scenario source root")
	}
}
