package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestClassifyCollectionOwnership(t *testing.T) {
	t.Run("default collection is KO managed", func(t *testing.T) {
		key, label := classifyCollectionOwnership(defaultKnowledgeCollection, 0, 0)
		if key != "knowledge_observatory" {
			t.Fatalf("expected knowledge_observatory key, got %q", key)
		}
		if label == "" {
			t.Fatal("expected non-empty label")
		}
	})

	t.Run("provenance in both stores is KO managed", func(t *testing.T) {
		key, _ := classifyCollectionOwnership("custom", 5, 5)
		if key != "knowledge_observatory" {
			t.Fatalf("expected knowledge_observatory key, got %q", key)
		}
	})

	t.Run("partial provenance is mixed", func(t *testing.T) {
		key, _ := classifyCollectionOwnership("custom", 5, 0)
		if key != "mixed" {
			t.Fatalf("expected mixed key, got %q", key)
		}
	})

	t.Run("no provenance is unknown", func(t *testing.T) {
		key, _ := classifyCollectionOwnership("external", 0, 0)
		if key != "external_or_unknown" {
			t.Fatalf("expected external_or_unknown key, got %q", key)
		}
	})
}

func TestMapCollectionRecordPreview(t *testing.T) {
	point := qdrantPoint{
		ID: "point-1",
		Payload: map[string]interface{}{
			"namespace":    "docs",
			"document_id":  "doc-1",
			"chunk_index":  2.0,
			"content_hash": "abc",
			"content":      "hello world",
			"tags":         []interface{}{"a", "b"},
		},
	}

	record := mapCollectionRecordPreview(point)
	if record.ID != "point-1" {
		t.Fatalf("record id mismatch: %q", record.ID)
	}
	if record.Namespace != "docs" {
		t.Fatalf("namespace mismatch: %q", record.Namespace)
	}
	if record.DocumentID != "doc-1" {
		t.Fatalf("document mismatch: %q", record.DocumentID)
	}
	if record.ChunkIndex == nil || *record.ChunkIndex != 2 {
		t.Fatalf("chunk index mismatch: %+v", record.ChunkIndex)
	}
	if len(record.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(record.Tags))
	}
	if record.ContentPreview == "" {
		t.Fatal("expected content preview")
	}
}

func TestHandleDeleteCollection(t *testing.T) {
	t.Run("deletes qdrant collection and returns deleted response", func(t *testing.T) {
		qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method=%s", r.Method)
			}
			if r.URL.Path != "/collections/alpha" {
				t.Fatalf("path=%s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer qdrant.Close()

		srv := newTestServer()
		srv.config.QdrantURL = qdrant.URL
		srv.router = mux.NewRouter()
		srv.router.HandleFunc("/api/v1/knowledge/collections/{collection}", srv.handleDeleteCollection).Methods(http.MethodDelete)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/knowledge/collections/alpha", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var resp deleteCollectionResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !resp.Deleted {
			t.Fatal("expected deleted=true")
		}
		if resp.Collection != "alpha" {
			t.Fatalf("collection=%q", resp.Collection)
		}
	})

	t.Run("returns not found when qdrant collection is missing", func(t *testing.T) {
		qdrant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"status":"not_found"}`))
		}))
		defer qdrant.Close()

		srv := newTestServer()
		srv.config.QdrantURL = qdrant.URL
		srv.router = mux.NewRouter()
		srv.router.HandleFunc("/api/v1/knowledge/collections/{collection}", srv.handleDeleteCollection).Methods(http.MethodDelete)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/knowledge/collections/missing", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
