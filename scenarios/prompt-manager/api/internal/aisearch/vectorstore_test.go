package aisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// helper: spin up a fake qdrant server with custom handler logic, return
// VectorStore pointed at it.
func newTestVS(t *testing.T, handler http.HandlerFunc) (VectorStore, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewVectorStore(srv.URL, "", "test", 3), srv
}

func TestVectorStore_ScrollIDs_SinglePage(t *testing.T) {
	var calls int
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if !strings.HasSuffix(r.URL.Path, "/points/scroll") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var req scrollRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.WithPayload) != 1 || req.WithPayload[0] != payloadHashKey {
			t.Errorf("expected with_payload=[payload_hash], got %v", req.WithPayload)
		}
		if req.WithVectors {
			t.Errorf("expected with_vectors=false")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"points": []map[string]any{
					{"id": "id-1", "payload": map[string]any{payloadHashKey: "sha256:aa"}},
					{"id": "id-2", "payload": map[string]any{payloadHashKey: "sha256:bb"}},
					{"id": "id-3", "payload": map[string]any{payloadHashKey: "sha256:cc"}},
				},
				"next_page_offset": nil,
			},
		})
	})

	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("ScrollIDs: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 POST, got %d", calls)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 entries, got %d", len(got))
	}
	if got["id-1"].PayloadHash != "sha256:aa" {
		t.Errorf("id-1 hash = %q", got["id-1"].PayloadHash)
	}
}

func TestVectorStore_ScrollIDs_MultiPage(t *testing.T) {
	var calls int
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req scrollRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			if req.Offset != nil {
				t.Errorf("page 1 offset should be nil, got %v", req.Offset)
			}
			points := make([]map[string]any, 256)
			for i := 0; i < 256; i++ {
				points[i] = map[string]any{
					"id":      fmt.Sprintf("id-%d", i),
					"payload": map[string]any{payloadHashKey: "sha256:aa"},
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"points":           points,
					"next_page_offset": "id-255",
				},
			})
		case 2:
			if req.Offset != "id-255" {
				t.Errorf("page 2 offset should be 'id-255', got %v", req.Offset)
			}
			points := make([]map[string]any, 50)
			for i := 0; i < 50; i++ {
				points[i] = map[string]any{
					"id":      fmt.Sprintf("id-%d", 256+i),
					"payload": map[string]any{payloadHashKey: "sha256:bb"},
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"points":           points,
					"next_page_offset": nil,
				},
			})
		default:
			t.Errorf("unexpected 3rd call")
		}
	})

	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("ScrollIDs: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 POSTs, got %d", calls)
	}
	if len(got) != 306 {
		t.Errorf("expected 306 entries, got %d", len(got))
	}
}

func TestVectorStore_ScrollIDs_MissingPayloadHash(t *testing.T) {
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"points": []map[string]any{
					{"id": "id-1", "payload": map[string]any{}},
				},
				"next_page_offset": nil,
			},
		})
	})
	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("ScrollIDs: %v", err)
	}
	if got["id-1"].PayloadHash != "" {
		t.Errorf("expected empty hash, got %q", got["id-1"].PayloadHash)
	}
}

func TestVectorStore_ScrollIDs_HTTPError_Propagates(t *testing.T) {
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := vs.ScrollIDs(context.Background())
	if err == nil {
		t.Fatalf("expected error on 500")
	}
}

func TestVectorStore_ScrollIDs_404IsEmpty(t *testing.T) {
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	got, err := vs.ScrollIDs(context.Background())
	if err != nil {
		t.Fatalf("expected nil error on 404, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

type capturedDelete struct {
	mu    sync.Mutex
	calls [][]string
}

func (c *capturedDelete) record(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dup := make([]string, len(ids))
	copy(dup, ids)
	c.calls = append(c.calls, dup)
}

func (c *capturedDelete) snapshot() [][]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]string, len(c.calls))
	copy(out, c.calls)
	return out
}

func TestVectorStore_BatchDelete_SingleRequest(t *testing.T) {
	captured := &capturedDelete{}
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Points []string `json:"points"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		captured.record(body.Points)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{}}`))
	})

	if err := vs.BatchDelete(context.Background(), []string{"a", "b", "c", "d", "e"}); err != nil {
		t.Fatalf("BatchDelete: %v", err)
	}
	calls := captured.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 POST, got %d", len(calls))
	}
	if len(calls[0]) != 5 {
		t.Errorf("expected 5 ids in single request, got %d", len(calls[0]))
	}
}

func TestVectorStore_BatchDelete_ChunksAtBoundary(t *testing.T) {
	captured := &capturedDelete{}
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Points []string `json:"points"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		captured.record(body.Points)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{}}`))
	})

	ids := make([]string, 600)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
	}
	if err := vs.BatchDelete(context.Background(), ids); err != nil {
		t.Fatalf("BatchDelete: %v", err)
	}
	calls := captured.snapshot()
	if len(calls) != 3 {
		t.Fatalf("expected 3 POSTs (256+256+88), got %d", len(calls))
	}
	if len(calls[0]) != 256 || len(calls[1]) != 256 || len(calls[2]) != 88 {
		t.Errorf("expected chunk sizes 256/256/88, got %d/%d/%d",
			len(calls[0]), len(calls[1]), len(calls[2]))
	}
}

func TestVectorStore_BatchDelete_EmptyIDsNoop(t *testing.T) {
	var calls int
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	})
	if err := vs.BatchDelete(context.Background(), nil); err != nil {
		t.Fatalf("nil slice: %v", err)
	}
	if err := vs.BatchDelete(context.Background(), []string{}); err != nil {
		t.Fatalf("empty slice: %v", err)
	}
	if calls != 0 {
		t.Errorf("expected zero HTTP calls, got %d", calls)
	}
}

func TestVectorStore_BatchDelete_PartialFailure(t *testing.T) {
	var calls int
	var mu sync.Mutex
	vs, _ := newTestVS(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		c := calls
		mu.Unlock()
		if c == 2 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	ids := make([]string, 600)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
	}
	err := vs.BatchDelete(context.Background(), ids)
	if err == nil {
		t.Fatal("expected error on chunk 2 failure")
	}
	if !strings.Contains(err.Error(), "256-512") {
		t.Errorf("expected error to name failed chunk range, got %q", err.Error())
	}
}
