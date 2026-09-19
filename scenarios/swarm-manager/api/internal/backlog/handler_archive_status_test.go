package backlog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// archiveItem drives the Archive handler for one item and returns the reloaded
// item so the persisted status can be asserted.
func archiveItem(t *testing.T, h *Handler, rootDir string, kind BacklogKind, name string) BacklogItem {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/"+string(kind)+"/"+name+"/archive", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": string(kind), "name": name})
	rec := httptest.NewRecorder()
	h.Archive(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("archive %s/%s: expected 2xx, got %d: %s", kind, name, rec.Code, rec.Body.String())
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		t.Fatalf("reload %s/%s: %v", kind, name, err)
	}
	return item
}

// Archiving unfinished work must settle its status, not leave it reading
// `backlog`. An archived item stuck at `backlog` is not resolved, so it never
// satisfies a dependency gate and holds every dependent in `blocked` forever.
func TestArchive_SettlesUnfinishedItemToDropped(t *testing.T) {
	for _, priorStatus := range []BacklogStatus{
		StatusBacklog,
		StatusReady,
		StatusResearching,
		StatusReviewPending,
	} {
		t.Run(string(priorStatus), func(t *testing.T) {
			h, rootDir := setupTestHandler(t)
			name := "archive-" + string(priorStatus)
			createTestItem(t, rootDir, KindExecute, BacklogItem{
				Name: name, Kind: KindExecute, Status: priorStatus, Title: "Some work",
			})

			item := archiveItem(t, h, rootDir, KindExecute, name)

			if item.Status != StatusDropped {
				t.Errorf("status = %q, want %q — archiving unfinished work is a decision not to do it",
					item.Status, StatusDropped)
			}
			if !IsResolvedStatus(item.Status) {
				t.Errorf("status %q is not resolved; dependents would stay blocked forever", item.Status)
			}
			if item.ArchivedAt == nil {
				t.Error("archived_at was not stamped")
			}
		})
	}
}

// Archiving finished work must not rewrite its verdict: a completed item that
// gets tidied away still shipped, and a failed one still failed.
func TestArchive_PreservesTerminalStatus(t *testing.T) {
	for _, priorStatus := range []BacklogStatus{
		StatusCompleted,
		StatusFailed,
		StatusNeedsFollowup,
	} {
		t.Run(string(priorStatus), func(t *testing.T) {
			h, rootDir := setupTestHandler(t)
			name := "archive-terminal-" + string(priorStatus)
			createTestItem(t, rootDir, KindExecute, BacklogItem{
				Name: name, Kind: KindExecute, Status: priorStatus, Title: "Finished work",
			})

			item := archiveItem(t, h, rootDir, KindExecute, name)

			if item.Status != priorStatus {
				t.Errorf("status = %q, want %q preserved — archiving is not a re-verdict",
					item.Status, priorStatus)
			}
			if item.ArchivedAt == nil {
				t.Error("archived_at was not stamped")
			}
		})
	}
}
