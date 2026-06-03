package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// capturingDoer records every request and replies with a scripted response.
type capturingDoer struct {
	requests []capturedReq
	respond  func(req capturedReq) (int, string)
}

type capturedReq struct {
	method string
	url    string
	body   map[string]any
}

func (c *capturingDoer) Do(req *http.Request) (*http.Response, error) {
	cap := capturedReq{method: req.Method, url: req.URL.String()}
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &cap.body)
		}
	}
	c.requests = append(c.requests, cap)
	status, body := http.StatusOK, "{}"
	if c.respond != nil {
		status, body = c.respond(cap)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     make(http.Header),
	}, nil
}

func (c *capturingDoer) last() capturedReq { return c.requests[len(c.requests)-1] }

func TestEnsureCollectionHybridShape(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if req.method == http.MethodGet {
			return http.StatusNotFound, "{}" // force creation
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	if err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: 768, Sparse: true, SparseModifier: "idf"}); err != nil {
		t.Fatal(err)
	}
	put := doer.last()
	if put.method != http.MethodPut {
		t.Fatalf("expected PUT create, got %s", put.method)
	}
	vectors, _ := put.body["vectors"].(map[string]any)
	if _, ok := vectors["dense"]; !ok {
		t.Fatalf("expected named dense vector, got %v", put.body["vectors"])
	}
	sparse, ok := put.body["sparse_vectors"].(map[string]any)
	if !ok {
		t.Fatalf("expected sparse_vectors block, got %v", put.body)
	}
	s, _ := sparse["sparse"].(map[string]any)
	if s["modifier"] != "idf" {
		t.Fatalf("expected idf modifier, got %v", sparse["sparse"])
	}
}

func TestEnsureCollectionDenseOnlyOmitsSparse(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if req.method == http.MethodGet {
			return http.StatusNotFound, "{}"
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: 768}); err != nil {
		t.Fatal(err)
	}
	put := doer.last()
	if _, ok := put.body["sparse_vectors"]; ok {
		t.Fatalf("dense-only collection must not declare sparse_vectors: %v", put.body)
	}
	vectors, _ := put.body["vectors"].(map[string]any)
	if _, ok := vectors["dense"]; !ok {
		t.Fatal("dense-only collection must still use a NAMED dense vector (D5)")
	}
}

func TestQueryHybridFusionShape(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		return http.StatusOK, `{"result":{"points":[{"id":"p1","score":0.83,"payload":{"body":"x"}}]}}`
	}}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	got, err := store.Query(context.Background(), HybridQuery{
		Dense:         []float64{0.1, 0.2},
		Sparse:        &SparseVector{Indices: []uint32{1, 2}, Values: []float32{1, 1}},
		Fusion:        "rrf",
		Limit:         5,
		PrefetchLimit: 50,
		Filter:        &QueryFilter{Must: []FieldMatch{{Key: "scope", Value: "project"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "p1" || got[0].Score != 0.83 {
		t.Fatalf("unexpected query result: %+v", got)
	}
	req := doer.last()
	if !strings.HasSuffix(req.url, "/collections/vrooli-docs/points/query") {
		t.Fatalf("expected Query API endpoint, got %s", req.url)
	}
	prefetch, ok := req.body["prefetch"].([]any)
	if !ok || len(prefetch) != 2 {
		t.Fatalf("expected 2 prefetch legs, got %v", req.body["prefetch"])
	}
	usings := map[string]bool{}
	for _, leg := range prefetch {
		m := leg.(map[string]any)
		usings[m["using"].(string)] = true
		if _, ok := m["filter"].(map[string]any); !ok {
			t.Fatalf("each prefetch leg must carry the scope filter, got %v", m)
		}
	}
	if !usings["dense"] || !usings["sparse"] {
		t.Fatalf("expected dense+sparse legs, got %v", usings)
	}
	q, _ := req.body["query"].(map[string]any)
	if q["fusion"] != "rrf" {
		t.Fatalf("expected rrf fusion, got %v", req.body["query"])
	}
}

func TestQueryDenseOnlyShape(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		return http.StatusOK, `{"result":{"points":[]}}`
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if _, err := store.Query(context.Background(), HybridQuery{Dense: []float64{0.1}, Limit: 10}); err != nil {
		t.Fatal(err)
	}
	req := doer.last()
	if _, ok := req.body["prefetch"]; ok {
		t.Fatalf("dense-only query must not use prefetch fusion: %v", req.body)
	}
	if req.body["using"] != "dense" {
		t.Fatalf("dense-only query must address the named dense vector, got %v", req.body["using"])
	}
}

func TestUpsertNamedVectorShape(t *testing.T) {
	doer := &capturingDoer{}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	err := store.Upsert(context.Background(), Point{
		ID:      "p1",
		Dense:   []float64{0.1, 0.2},
		Sparse:  &SparseVector{Indices: []uint32{5}, Values: []float32{2}},
		Payload: map[string]any{"body": "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := doer.last()
	points, _ := req.body["points"].([]any)
	if len(points) != 1 {
		t.Fatalf("expected one point, got %v", req.body)
	}
	p := points[0].(map[string]any)
	vec, ok := p["vector"].(map[string]any)
	if !ok {
		t.Fatalf("vector must be a named-vector map, got %T", p["vector"])
	}
	if _, ok := vec["dense"]; !ok {
		t.Fatal("missing named dense vector")
	}
	if _, ok := vec["sparse"]; !ok {
		t.Fatal("missing named sparse vector")
	}
}

func TestSetPayloadShape(t *testing.T) {
	doer := &capturingDoer{}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	if err := store.SetPayload(context.Background(), "p1", map[string]any{"source_hash": "sha256:abc"}); err != nil {
		t.Fatal(err)
	}
	req := doer.last()
	if !strings.Contains(req.url, "/points/payload") {
		t.Fatalf("expected set-payload endpoint, got %s", req.url)
	}
	pts, _ := req.body["points"].([]any)
	if len(pts) != 1 || pts[0] != "p1" {
		t.Fatalf("expected point id p1, got %v", req.body["points"])
	}
}
