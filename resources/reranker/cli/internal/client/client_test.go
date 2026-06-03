package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// capturedRerankFixture is a real TEI /rerank response shape: an array of
// {index, score} sorted by score descending. (Captured so CI needs no live
// container.) Here doc 2 is most relevant, doc 0 least.
const capturedRerankFixture = `[
  {"index": 2, "score": 0.98765},
  {"index": 0, "score": 0.4213},
  {"index": 1, "score": 0.01124}
]`

func TestRerankDecodesCapturedFixture(t *testing.T) {
	var gotBody rerankRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rerank" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("request body not valid json: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, capturedRerankFixture)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	results, err := c.Rerank(context.Background(), "how do I restart a scenario",
		[]string{"unrelated passage", "another unrelated", "restart a scenario via the CLI"}, false)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if results[0].Index != 2 {
		t.Errorf("top result index = %d, want 2", results[0].Index)
	}
	if results[0].Score < results[1].Score || results[1].Score < results[2].Score {
		t.Errorf("results not sorted descending: %+v", results)
	}
	if gotBody.Query == "" || len(gotBody.Texts) != 3 {
		t.Errorf("request body did not carry query+texts: %+v", gotBody)
	}
}

func TestRerankRejectsEmptyInputs(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1:1", HTTP: http.DefaultClient}
	if _, err := c.Rerank(context.Background(), "", []string{"a"}, false); err == nil {
		t.Error("expected error for empty query")
	}
	if _, err := c.Rerank(context.Background(), "q", nil, false); err == nil {
		t.Error("expected error for empty documents")
	}
}

func TestHealthStatuses(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer bad.Close()

	if err := (&Client{BaseURL: ok.URL, HTTP: ok.Client()}).Health(context.Background()); err != nil {
		t.Errorf("Health() on 200 = %v, want nil", err)
	}
	if err := (&Client{BaseURL: bad.URL, HTTP: bad.Client()}).Health(context.Background()); err == nil {
		t.Error("Health() on 503 = nil, want error")
	}
}

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"base url wins", map[string]string{"RERANKER_BASE_URL": "http://h:9/", "RERANKER_URL": "http://x:1"}, "http://h:9"},
		{"url next", map[string]string{"RERANKER_URL": "http://h:9"}, "http://h:9"},
		{"host+port", map[string]string{"RERANKER_HOST": "box", "RERANKER_PORT": "8080"}, "http://box:8080"},
		{"host with port", map[string]string{"RERANKER_HOST": "box:7070"}, "http://box:7070"},
		{"default", map[string]string{}, "http://127.0.0.1:11453"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveBaseURL(func(k string) string { return tt.env[k] })
			if got != tt.want {
				t.Errorf("ResolveBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
