//go:build testing
// +build testing

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// TestVectorSearchStructures exercises production wire structures instead of
// reimplementing cosine similarity or result selection in the test.
func TestVectorSearchStructures(t *testing.T) {
	point := VectorPoint{ID: "technique-1", Vector: []float32{1, 0, 0}, Payload: map[string]interface{}{"name": "Direct override", "category": "prompt-injection"}}
	encoded, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("marshal vector point: %v", err)
	}
	var decoded VectorPoint
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal vector point: %v", err)
	}
	if decoded.ID != point.ID || !reflect.DeepEqual(decoded.Vector, point.Vector) {
		t.Fatalf("vector point round trip changed identity/vector: %#v", decoded)
	}
	if decoded.Payload["name"] != point.Payload["name"] {
		t.Fatalf("vector point round trip changed payload: %#v", decoded.Payload)
	}
}

func TestVectorSearchQdrantContract(t *testing.T) {
	var mu sync.Mutex
	var collectionCreated, pointsUpserted bool
	var searchRequest struct {
		Vector         []float32 `json:"vector"`
		Limit          int       `json:"limit"`
		ScoreThreshold float32   `json:"score_threshold"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/injection_techniques":
			mu.Lock()
			collectionCreated = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/collections/injection_techniques/points":
			var body struct {
				Points []VectorPoint `json:"points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Points) != 1 {
				http.Error(w, "invalid points", http.StatusBadRequest)
				return
			}
			mu.Lock()
			pointsUpserted = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/collections/injection_techniques/points/search":
			if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
				http.Error(w, "invalid search", http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"result": []SearchResult{{ID: "high-confidence", Score: 0.91, Payload: map[string]interface{}{"name": "Direct override", "category": "prompt-injection", "description": "Override prior instructions", "example_prompt": "Ignore the system prompt"}}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalConfig := appConfig
	appConfig = &Config{QdrantURL: server.URL}
	t.Cleanup(func() { appConfig = originalConfig })
	originalRunner := ollamaGatewayRunner
	ollamaGatewayRunner = func(_ context.Context, args []string, stdin string) ([]byte, error) {
		if !reflect.DeepEqual(args, []string{"gateway", "embed", "--role", defaultEmbeddingRole, "--json", "--input-stdin"}) {
			t.Errorf("unexpected gateway args: %#v", args)
		}
		if stdin != "find direct overrides" {
			t.Errorf("unexpected embedding input %q", stdin)
		}
		return []byte(`{"embedding":[1,0,0]}`), nil
	}
	t.Cleanup(func() { ollamaGatewayRunner = originalRunner })

	client := NewQdrantClient()
	if err := client.CreateCollection("injection_techniques", 3); err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if err := client.UpsertPoints("injection_techniques", []VectorPoint{{ID: "technique-1", Vector: []float32{1, 0, 0}}}); err != nil {
		t.Fatalf("upsert points: %v", err)
	}
	results, err := FindSimilarInjectionsWithThreshold("find direct overrides", 3, 0.75)
	if err != nil {
		t.Fatalf("find similar injections: %v", err)
	}
	if len(results) != 1 || results[0]["id"] != "high-confidence" {
		t.Fatalf("unexpected search results: %#v", results)
	}
	if results[0]["name"] != "Direct override" || results[0]["score"] != float32(0.91) {
		t.Fatalf("search result payload/score not preserved: %#v", results[0])
	}
	mu.Lock()
	created, upserted := collectionCreated, pointsUpserted
	mu.Unlock()
	if !created || !upserted {
		t.Fatalf("expected collection and upsert requests, created=%v upserted=%v", created, upserted)
	}
	if !reflect.DeepEqual(searchRequest.Vector, []float32{1, 0, 0}) || searchRequest.Limit != 3 || searchRequest.ScoreThreshold != 0.75 {
		t.Fatalf("search contract mismatch: %#v", searchRequest)
	}
}

func TestVectorSearchExternalErrors(t *testing.T) {
	originalRunner := ollamaGatewayRunner
	t.Cleanup(func() { ollamaGatewayRunner = originalRunner })

	t.Run("malformed embedding response", func(t *testing.T) {
		ollamaGatewayRunner = func(context.Context, []string, string) ([]byte, error) { return []byte(`{"embedding":`), nil }
		_, err := GenerateEmbedding("query")
		if err == nil || !strings.Contains(err.Error(), "decode gateway embed response") {
			t.Fatalf("expected malformed embedding error, got %v", err)
		}
	})
	t.Run("gateway process error", func(t *testing.T) {
		ollamaGatewayRunner = func(context.Context, []string, string) ([]byte, error) { return nil, fmt.Errorf("fixture unavailable") }
		_, err := GenerateEmbedding("query")
		if err == nil || !strings.Contains(err.Error(), "fixture unavailable") {
			t.Fatalf("expected gateway error, got %v", err)
		}
	})
}

func TestVectorSearchHandlersValidateWireInput(t *testing.T) {
	router := setupTestRouter()
	for _, tc := range []struct {
		name string
		body map[string]interface{}
	}{
		{name: "missing query", body: map[string]interface{}{"limit": 5}},
		{name: "invalid threshold", body: map[string]interface{}{"query": "test", "threshold": 1.5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := makeHTTPRequest(router, HTTPTestRequest{Method: http.MethodPost, Path: "/api/v1/vector/search", Body: tc.body})
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
	badJSON := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/search", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(badJSON, req)
	if badJSON.Code != http.StatusBadRequest {
		t.Fatalf("expected malformed JSON to return 400, got %d", badJSON.Code)
	}
}
