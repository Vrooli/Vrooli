package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVectorStore_Defaults(t *testing.T) {
	vs := NewVectorStore("http://localhost:6333", "", "", 0)
	if vs.Collection != "swarm-manager" {
		t.Errorf("expected default collection swarm-manager, got %s", vs.Collection)
	}
	if vs.VectorSize != 768 {
		t.Errorf("expected default vector size 768, got %d", vs.VectorSize)
	}
}

func TestVectorStore_EnsureCollection_AlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET for existing collection, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	if err := vs.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVectorStore_EnsureCollection_Create(t *testing.T) {
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodPut {
			created = true
			var req createCollectionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Vectors.Distance != "Cosine" {
				t.Errorf("expected Cosine distance, got %s", req.Vectors.Distance)
			}
			if req.Vectors.Size != 768 {
				t.Errorf("expected size 768, got %d", req.Vectors.Size)
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "new", 768)
	if err := vs.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected collection to be created")
	}
}

func TestVectorStore_Upsert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/points") {
			t.Errorf("expected /points path, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("wait") != "true" {
			t.Error("expected wait=true query")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	err := vs.Upsert(context.Background(), "id-1", []float64{0.1, 0.2}, map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVectorStore_Upsert_EmptyID(t *testing.T) {
	vs := NewVectorStore("http://localhost:6333", "", "test", 768)
	if err := vs.Upsert(context.Background(), "", []float64{0.1}, nil); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestVectorStore_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/points/delete") {
			t.Errorf("expected /points/delete path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	if err := vs.Delete(context.Background(), "id-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVectorStore_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/points/search") {
			t.Errorf("expected /points/search path, got %s", r.URL.Path)
		}
		resp := searchResponse{
			Result: []struct {
				ID      interface{}            `json:"id"`
				Score   float64                `json:"score"`
				Payload map[string]interface{} `json:"payload"`
			}{
				{ID: "a", Score: 0.9, Payload: map[string]interface{}{"name": "first"}},
				{ID: "b", Score: 0.7, Payload: map[string]interface{}{"name": "second"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	got, err := vs.Search(context.Background(), []float64{0.1}, 5, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].ID != "a" || got[0].Score != 0.9 {
		t.Errorf("unexpected first result: %+v", got[0])
	}
}

func TestVectorStore_CountPoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/points/count") {
			t.Errorf("expected /points/count, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
			Count int `json:"count"`
		}{Count: 42}})
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	n, err := vs.CountPoints(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

func TestVectorStore_CountPoints_404TreatedAsEmpty(t *testing.T) {
	// On first boot the collection does not yet exist. Qdrant returns 404
	// for /points/count; we must treat that as count=0 rather than an error
	// so startup backfill can detect drift and kick off the initial reindex.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "new", 768)
	n, err := vs.CountPoints(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestVectorStore_Available_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	if !vs.Available(context.Background()) {
		t.Error("expected Available true")
	}
}

func TestVectorStore_Available_EmptyURL(t *testing.T) {
	vs := NewVectorStore("", "", "test", 768)
	if vs.Available(context.Background()) {
		t.Error("expected Available false for empty URL")
	}
}

func TestVectorStore_APIKeyHeader(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("api-key")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "my-secret", "test", 768)
	vs.Available(context.Background())
	if got != "my-secret" {
		t.Errorf("expected api-key 'my-secret', got %q", got)
	}
}
