package aisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedsearch "github.com/vrooli/ai-go/search"
)

func TestVectorStore_DefaultCollectionRequiresResolvedVectorSize(t *testing.T) {
	// Empty collection still means the variant-aware defaultCollection(), but
	// vector dimensions must be resolved from Ollama policy by startup wiring
	// before this store is constructed.
	var observedPath string
	var createBody createCollectionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedPath = r.URL.Path
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&createBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "", 3)
	if err := vs.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantCollection := defaultCollection()
	if !strings.HasSuffix(observedPath, "/collections/"+wantCollection) {
		t.Errorf("expected default collection %q in path, got %q", wantCollection, observedPath)
	}
	if createBody.Vectors.Size != 3 {
		t.Errorf("expected resolved vector size 3 on the wire, got %d", createBody.Vectors.Size)
	}
}

func TestVectorStoreForPolicyCreateUsesPolicyDimensions(t *testing.T) {
	var gotSize int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			var req createCollectionRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			gotSize = req.Vectors.Size
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	vs := NewVectorStoreForPolicy(server.URL, "", "", sharedsearch.EmbeddingPolicy{
		Role:       "embedding.default",
		Model:      "fixture-embed-model:latest",
		Dimensions: 1234,
	})
	if err := vs.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	if gotSize != 1234 {
		t.Errorf("expected policy dimensions 1234, got %d", gotSize)
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

// ---- ScrollIDs ----

// scrollPage is a per-page response builder used by the scroll tests below.
// Mirrors qdrant's /points/scroll response shape.
type scrollPagePoint struct {
	ID      interface{}            `json:"id"`
	Payload map[string]interface{} `json:"payload"`
}

type scrollPageBody struct {
	Result struct {
		Points         []scrollPagePoint `json:"points"`
		NextPageOffset interface{}       `json:"next_page_offset"`
	} `json:"result"`
}

func encodeScrollPage(t *testing.T, page scrollPageBody) []byte {
	t.Helper()
	b, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("encode scroll page: %v", err)
	}
	return b
}

func TestVectorStore_ScrollIDs_SinglePage(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/points/scroll") {
			t.Errorf("expected /points/scroll, got %s", r.URL.Path)
		}
		hits++
		var page scrollPageBody
		page.Result.Points = []scrollPagePoint{
			{ID: "id-a", Payload: map[string]interface{}{"payload_hash": "sha256:aaa", "archived": false}},
			{ID: "id-b", Payload: map[string]interface{}{"payload_hash": "sha256:bbb", "archived": true}},
			{ID: "id-c", Payload: map[string]interface{}{"payload_hash": "sha256:ccc"}},
		}
		page.Result.NextPageOffset = nil
		_, _ = w.Write(encodeScrollPage(t, page))
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 1 {
		t.Errorf("expected 1 POST, got %d", hits)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got["id-a"].PayloadHash != "sha256:aaa" || got["id-b"].PayloadHash != "sha256:bbb" {
		t.Errorf("payload_hash not preserved: %+v", got)
	}
	if got["id-b"].Archived != true || got["id-a"].Archived != false {
		t.Errorf("archived flag not preserved: %+v", got)
	}
}

func TestVectorStore_ScrollIDs_MultiPage(t *testing.T) {
	var hits int
	var receivedOffsets []interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req scrollRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedOffsets = append(receivedOffsets, req.Offset)
		hits++
		var page scrollPageBody
		switch hits {
		case 1:
			// First page: 256 points, advertise offset for next page.
			points := make([]scrollPagePoint, 256)
			for i := range points {
				points[i] = scrollPagePoint{ID: fmt.Sprintf("p1-%d", i), Payload: map[string]interface{}{"payload_hash": "sha256:p1"}}
			}
			page.Result.Points = points
			page.Result.NextPageOffset = "next-token"
		case 2:
			// Second page: 50 points, terminate.
			points := make([]scrollPagePoint, 50)
			for i := range points {
				points[i] = scrollPagePoint{ID: fmt.Sprintf("p2-%d", i), Payload: map[string]interface{}{"payload_hash": "sha256:p2"}}
			}
			page.Result.Points = points
			page.Result.NextPageOffset = nil
		default:
			t.Fatalf("unexpected third request")
		}
		_, _ = w.Write(encodeScrollPage(t, page))
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 2 {
		t.Errorf("expected 2 POSTs, got %d", hits)
	}
	if len(got) != 306 {
		t.Errorf("expected 306 entries, got %d", len(got))
	}
	// First request must omit offset; second must echo back the page-1 token.
	if receivedOffsets[0] != nil {
		t.Errorf("expected first request offset to be nil, got %v", receivedOffsets[0])
	}
	if receivedOffsets[1] != "next-token" {
		t.Errorf("expected second request offset to echo 'next-token', got %v", receivedOffsets[1])
	}
}

func TestVectorStore_ScrollIDs_MissingPayloadHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var page scrollPageBody
		page.Result.Points = []scrollPagePoint{
			{ID: "legacy", Payload: map[string]interface{}{"name": "x"}},
		}
		_, _ = w.Write(encodeScrollPage(t, page))
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["legacy"].PayloadHash != "" {
		t.Errorf("expected empty PayloadHash for legacy point, got %q", got["legacy"].PayloadHash)
	}
}

func TestVectorStore_ScrollIDs_PreservesArchivedFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var page scrollPageBody
		page.Result.Points = []scrollPagePoint{
			{ID: "a", Payload: map[string]interface{}{"archived": true, "payload_hash": "sha256:x"}},
		}
		_, _ = w.Write(encodeScrollPage(t, page))
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got["a"].Archived {
		t.Error("expected archived=true to be preserved in ScrollItem")
	}
}

func TestVectorStore_ScrollIDs_HTTPError_Propagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":{"error":"internal"}}`))
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	_, err := vs.ScrollIDs(context.Background())
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got %v", err)
	}
}

func TestVectorStore_ScrollIDs_404IsEmpty(t *testing.T) {
	// Same gentle-degradation contract as CountPoints: a missing collection is
	// the legitimate first-boot state and must not block reconcile from running.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("expected nil error on 404, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map on 404, got %d entries", len(got))
	}
}

// ---- BatchDelete ----

type batchDeleteCapture struct {
	Points []string `json:"points"`
}

func TestVectorStore_BatchDelete_SingleRequest(t *testing.T) {
	var hits int
	var captured batchDeleteCapture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/points/delete") {
			t.Errorf("expected /points/delete, got %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&captured)
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	if err := vs.BatchDelete(context.Background(), []string{"a", "b", "c", "d", "e"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 1 {
		t.Errorf("expected 1 POST, got %d", hits)
	}
	if len(captured.Points) != 5 {
		t.Errorf("expected 5 points in body, got %d", len(captured.Points))
	}
}

func TestVectorStore_BatchDelete_ChunksAtBoundary(t *testing.T) {
	var hits int
	var sizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body batchDeleteCapture
		_ = json.NewDecoder(r.Body).Decode(&body)
		sizes = append(sizes, len(body.Points))
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ids := make([]string, 600)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
	}
	vs := NewVectorStore(server.URL, "", "test", 768)
	if err := vs.BatchDelete(context.Background(), ids); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 3 {
		t.Errorf("expected 3 chunks, got %d", hits)
	}
	wantSizes := []int{MaxDeleteBatch, MaxDeleteBatch, 600 - 2*MaxDeleteBatch}
	for i, w := range wantSizes {
		if sizes[i] != w {
			t.Errorf("chunk %d: expected %d points, got %d", i, w, sizes[i])
		}
	}
}

func TestVectorStore_BatchDelete_EmptyIDsNoop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("BatchDelete with empty ids should make no HTTP call")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	vs := NewVectorStore(server.URL, "", "test", 768)
	if err := vs.BatchDelete(context.Background(), nil); err != nil {
		t.Errorf("expected nil error for empty ids, got %v", err)
	}
}

func TestVectorStore_BatchDelete_PartialFailure(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		// Fail the second of three chunks; verify the first and third still ran
		// and that the error names which chunk failed.
		if hits == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("kaboom"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ids := make([]string, 600)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
	}
	vs := NewVectorStore(server.URL, "", "test", 768)
	err := vs.BatchDelete(context.Background(), ids)
	if err == nil {
		t.Fatal("expected error from failing chunk")
	}
	if hits != 3 {
		t.Errorf("expected all 3 chunks attempted, got %d", hits)
	}
	if !strings.Contains(err.Error(), "[256:512]") {
		t.Errorf("expected error to identify failing chunk range, got %v", err)
	}
}
