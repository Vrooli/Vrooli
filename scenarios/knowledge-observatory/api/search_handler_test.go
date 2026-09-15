package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleSearch exercises the re-pointed /api/v1/knowledge/search handler,
// which now delegates to the hybrid documentation engine (Phase 6 cutover).
func TestHandleSearch(t *testing.T) {
	srv := &Server{docSearch: fakeDocSearch{}}

	t.Run("accepts valid request", func(t *testing.T) {
		raw, _ := json.Marshal(map[string]interface{}{"query": "test query", "limit": 10})
		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewReader(raw))
		rec := httptest.NewRecorder()

		srv.handleSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}

		var decoded SearchResponse
		if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if decoded.Query != "test query" {
			t.Fatalf("query=%q", decoded.Query)
		}
		if len(decoded.Results) != 1 {
			t.Fatalf("results=%d", len(decoded.Results))
		}
		if decoded.Results[0].Content != "demo" {
			t.Fatalf("content=%q", decoded.Results[0].Content)
		}
	})

	t.Run("ignores unknown fields gracefully", func(t *testing.T) {
		// Legacy records-era fields (ingested_after, ingested_before, collection,
		// namespaces, visibility, tags) are no longer in SearchRequest and are
		// silently discarded by the JSON decoder.
		raw, _ := json.Marshal(map[string]interface{}{"query": "test query", "ingested_after": "nope"})
		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewReader(raw))
		rec := httptest.NewRecorder()

		srv.handleSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s (expected 200; unknown fields should be silently discarded)", rec.Code, rec.Body.String())
		}
	})

	t.Run("503 when engine unavailable", func(t *testing.T) {
		empty := &Server{}
		raw, _ := json.Marshal(map[string]interface{}{"query": "test query"})
		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewReader(raw))
		rec := httptest.NewRecorder()

		empty.handleSearch(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
