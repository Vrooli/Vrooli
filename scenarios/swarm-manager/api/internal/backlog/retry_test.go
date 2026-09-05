package backlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/execution"

	"github.com/gorilla/mux"
)

func TestBacklogRetry_TerminalItemReopens(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "retry-item",
		Title:  "Retry Item",
		Status: StatusFailed,
		Kind:   KindExecute,
	})

	eq := &mockExecutionQueuer{
		retryLatestHasPrior: true,
		retryLatestRecord: execution.Record{
			ExecutionID:       "new-exec-1",
			ParentExecutionID: "parent-exec-1",
			Status:            execution.StatusStarting,
			BacklogKind:       "execute",
			BacklogName:       "retry-item",
		},
	}
	h.SetExecutionQueuer(eq)

	body, _ := json.Marshal(RetryRequest{Note: "fixed environmental issue"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/retry-item/retry", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "retry-item"})
	rec := httptest.NewRecorder()
	h.Retry(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(eq.retryLatestCalls) != 1 {
		t.Fatalf("expected 1 RetryLatestForBacklog call, got %d", len(eq.retryLatestCalls))
	}
	if eq.retryLatestCalls[0].Note != "fixed environmental issue" {
		t.Errorf("note not propagated, got %q", eq.retryLatestCalls[0].Note)
	}

	// Item should be reopened to in_progress.
	updated, err := h.store.LoadItem(KindExecute, "retry-item")
	if err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if updated.Status != StatusInProgress {
		t.Errorf("status = %q, want %q", updated.Status, StatusInProgress)
	}

	// Reopen audit record should exist.
	decisionsDir := filepath.Join(rootDir, backlogKindDirs[KindExecute], "retry-item", "review", "decisions")
	entries, err := os.ReadDir(decisionsDir)
	if err != nil {
		t.Fatalf("read decisions dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-reopen.json") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a -reopen.json decision file, got %v", entries)
	}

	// Response should include both execution ids.
	var resp RetryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.NewExecutionID != "new-exec-1" {
		t.Errorf("new_execution_id = %q, want new-exec-1", resp.NewExecutionID)
	}
	if resp.ParentExecutionID != "parent-exec-1" {
		t.Errorf("parent_execution_id = %q, want parent-exec-1", resp.ParentExecutionID)
	}
}

func TestBacklogRetry_NonTerminalItem_NoReopen(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "in-progress-item",
		Title:  "In Progress",
		Status: StatusInProgress,
		Kind:   KindExecute,
	})

	eq := &mockExecutionQueuer{
		retryLatestHasPrior: true,
		retryLatestRecord: execution.Record{
			ExecutionID:       "new-exec-2",
			ParentExecutionID: "parent-2",
			Status:            execution.StatusStarting,
		},
	}
	h.SetExecutionQueuer(eq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/in-progress-item/retry", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "in-progress-item"})
	rec := httptest.NewRecorder()
	h.Retry(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	// Item status should be unchanged (no reopen needed).
	updated, err := h.store.LoadItem(KindExecute, "in-progress-item")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if updated.Status != StatusInProgress {
		t.Errorf("status changed unexpectedly: %q", updated.Status)
	}

	// No reopen audit record should be written for non-terminal items.
	decisionsDir := filepath.Join(rootDir, backlogKindDirs[KindExecute], "in-progress-item", "review", "decisions")
	if _, err := os.Stat(decisionsDir); err == nil {
		entries, _ := os.ReadDir(decisionsDir)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), "-reopen.json") {
				t.Errorf("unexpected reopen audit record for non-terminal item: %s", e.Name())
			}
		}
	}
}

func TestBacklogRetry_NoPriorExecution_400(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "never-run",
		Title:  "Never Run",
		Status: StatusReady,
		Kind:   KindExecute,
	})

	eq := &mockExecutionQueuer{
		retryLatestHasPrior: false, // signals "no prior execution"
	}
	h.SetExecutionQueuer(eq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/never-run/retry", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "never-run"})
	rec := httptest.NewRecorder()
	h.Retry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "no prior execution") {
		t.Errorf("error message should mention missing prior execution, got %q", rec.Body.String())
	}
}

func TestBacklogRetry_ItemNotFound_404(t *testing.T) {
	h, _ := setupTestHandler(t)
	eq := &mockExecutionQueuer{retryLatestHasPrior: true}
	h.SetExecutionQueuer(eq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/missing/retry", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "missing"})
	rec := httptest.NewRecorder()
	h.Retry(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
