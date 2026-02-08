package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCmdCollectionPruneStale_DefaultsToDryRun(t *testing.T) {
	var got collectionMaintenanceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/knowledge/collections/knowledge/maintenance/prune-stale-chunks" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	t.Setenv("KNOWLEDGE_OBSERVATORY_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := app.cmdCollectionPruneStale([]string{"knowledge"}); err != nil {
		t.Fatalf("cmdCollectionPruneStale failed: %v", err)
	}
	if !got.DryRun {
		t.Fatalf("expected dry_run=true by default")
	}
	if got.MaxDeletes != 0 {
		t.Fatalf("expected max_deletes=0, got %d", got.MaxDeletes)
	}
}

func TestCmdCollectionDedupe_ApplyOverridesDryRun(t *testing.T) {
	var got collectionMaintenanceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/knowledge/collections/knowledge/maintenance/dedupe-content" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	t.Setenv("KNOWLEDGE_OBSERVATORY_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := app.cmdCollectionDedupe([]string{"--collection", "knowledge", "--apply", "--max-deletes", "7"}); err != nil {
		t.Fatalf("cmdCollectionDedupe failed: %v", err)
	}
	if got.DryRun {
		t.Fatalf("expected dry_run=false when --apply is set")
	}
	if got.MaxDeletes != 7 {
		t.Fatalf("expected max_deletes=7, got %d", got.MaxDeletes)
	}
}

func TestCmdCollectionDiagnostics_QueryAndPath(t *testing.T) {
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/knowledge/collections/knowledge/diagnostics" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("mode"); got != "full" {
			t.Fatalf("expected mode=full, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "123" {
			t.Fatalf("expected limit=123, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection":"knowledge"}`))
	}))
	defer server.Close()

	t.Setenv("KNOWLEDGE_OBSERVATORY_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := app.cmdCollectionDiagnostics([]string{"knowledge", "--mode", "full", "--limit", "123"}); err != nil {
		t.Fatalf("cmdCollectionDiagnostics failed: %v", err)
	}
	if !hit {
		t.Fatal("expected diagnostics endpoint to be called")
	}
}

func TestCmdDocumentDelete_ValidateAndRequest(t *testing.T) {
	t.Setenv("KNOWLEDGE_OBSERVATORY_API_BASE", "http://127.0.0.1:1")
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	err = app.cmdDocumentDelete([]string{"--namespace", "docs"})
	if err == nil || !strings.Contains(err.Error(), "usage: document-delete") {
		t.Fatalf("expected usage error for missing identifiers, got %v", err)
	}

	var got documentDeleteRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/knowledge/documents/delete" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"deleted_count":1}`))
	}))
	defer server.Close()

	t.Setenv("KNOWLEDGE_OBSERVATORY_API_BASE", server.URL)
	app, err = NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := app.cmdDocumentDelete([]string{"--namespace", "docs", "--external-id", "ext-1", "--apply"}); err != nil {
		t.Fatalf("cmdDocumentDelete failed: %v", err)
	}
	if got.Namespace != "docs" || got.ExternalID != "ext-1" {
		t.Fatalf("unexpected request payload: %+v", got)
	}
	if got.DryRun {
		t.Fatalf("expected dry_run=false when --apply is set")
	}
}

func TestCmdIngestHealth(t *testing.T) {
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/ingest/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	t.Setenv("KNOWLEDGE_OBSERVATORY_API_BASE", server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := app.cmdIngestHealth(nil); err != nil {
		t.Fatalf("cmdIngestHealth failed: %v", err)
	}
	if !hit {
		t.Fatal("expected ingest health endpoint to be called")
	}
}
