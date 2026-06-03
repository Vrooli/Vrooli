package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLLMScores(t *testing.T) {
	cases := []struct {
		name    string
		resp    string
		want    int
		wantErr bool
	}{
		{"clean", `[{"index":0,"score":0.9},{"index":1,"score":0.1}]`, 2, false},
		{"with-think", "<think>let me judge</think>\nHere:\n[{\"index\":0,\"score\":0.8}]", 1, false},
		{"with-fence", "```json\n[{\"index\":2,\"score\":0.5}]\n```", 1, false},
		{"prose-around", `The scores are [{"index":0,"score":1.0}] done.`, 1, false},
		// qwen3 reasoning often contains stray brackets like [1] before the real array.
		{"think-stray-brackets", "<think>candidate [1] looks better than [0]</think>\n[{\"index\":1,\"score\":0.9},{\"index\":0,\"score\":0.2}]", 2, false},
		{"unterminated-think", "<think>reasoning [0] without close [{\"index\":0", 0, true},
		{"bracket-in-string", `[{"index":0,"score":0.9,"note":"see [docs]"}]`, 1, false},
		{"no-array", "I cannot answer", 0, true},
		{"empty-array", "[]", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLLMScores(tc.resp)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.want {
				t.Fatalf("got %d scores, want %d", len(got), tc.want)
			}
		})
	}
}

func TestLLMRerankerRerank(t *testing.T) {
	var gotArgs []string
	var gotStdin string
	run := func(_ context.Context, args []string, stdin []byte) ([]byte, error) {
		gotArgs = args
		gotStdin = string(stdin)
		// Simulate qwen3 reasoning preamble + the JSON array.
		body, _ := json.Marshal(generateResponse{Response: `<think>ok</think>[{"index":1,"score":0.95},{"index":0,"score":0.2}]`})
		return body, nil
	}
	r := NewLLMRerankerWithRunner("qwen3:4b", run)
	cands := []RerankCandidate{
		{ID: "a", Text: "restart a scenario from the CLI"},
		{ID: "b", Text: "how to restart a scenario lifecycle"},
	}
	scores, err := r.Rerank(context.Background(), "restart a scenario", cands)
	if err != nil {
		t.Fatalf("Rerank: %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("got %d scores, want 2", len(scores))
	}
	// index 1 -> id "b" with the high score.
	if scores[0].ID != "b" || scores[0].Score != 0.95 {
		t.Fatalf("first score = %+v, want b/0.95", scores[0])
	}
	if r.Name() != "llm:qwen3:4b" {
		t.Fatalf("Name = %q", r.Name())
	}
	// The prompt must reach generate via stdin and request JSON output.
	if gotStdin == "" || gotArgs[len(gotArgs)-1] != "--prompt-stdin" {
		t.Fatalf("unexpected invocation args=%v stdin=%q", gotArgs, gotStdin)
	}
}

func TestLLMRerankerRerankBadResponse(t *testing.T) {
	run := func(_ context.Context, _ []string, _ []byte) ([]byte, error) {
		body, _ := json.Marshal(generateResponse{Response: "no json here"})
		return body, nil
	}
	r := NewLLMRerankerWithRunner("", run)
	_, err := r.Rerank(context.Background(), "q", []RerankCandidate{{ID: "a", Text: "x"}})
	if err == nil {
		t.Fatal("expected error on unparseable response")
	}
}

func TestCrossEncoderRerankerRerank(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/rerank" {
			t.Errorf("path = %q, want /rerank", req.URL.Path)
		}
		var body teiRerankRequest
		_ = json.NewDecoder(req.Body).Decode(&body)
		if len(body.Texts) != 2 {
			t.Errorf("texts len = %d, want 2", len(body.Texts))
		}
		// TEI returns score-sorted: candidate index 1 is most relevant.
		_ = json.NewEncoder(w).Encode([]teiRankResult{{Index: 1, Score: 0.88}, {Index: 0, Score: 0.01}})
	}))
	defer srv.Close()

	r := NewCrossEncoderRerankerWithClient(srv.URL, srv.Client())
	cands := []RerankCandidate{{ID: "a", Text: "noise"}, {ID: "b", Text: "the answer"}}
	scores, err := r.Rerank(context.Background(), "the answer", cands)
	if err != nil {
		t.Fatalf("Rerank: %v", err)
	}
	if len(scores) != 2 || scores[0].ID != "b" || scores[0].Score != 0.88 {
		t.Fatalf("scores = %+v, want b first @0.88", scores)
	}
}

func TestCrossEncoderRerankerAvailable(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer down.Close()

	if !NewCrossEncoderRerankerWithClient(healthy.URL, healthy.Client()).Available(context.Background()) {
		t.Fatal("expected healthy reranker Available")
	}
	if NewCrossEncoderRerankerWithClient(down.URL, down.Client()).Available(context.Background()) {
		t.Fatal("expected unhealthy reranker not Available")
	}
}

func TestResolveRerankerURL(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"base-url", map[string]string{"RERANKER_BASE_URL": "http://x:9/"}, "http://x:9"},
		{"url", map[string]string{"RERANKER_URL": "http://y:80"}, "http://y:80"},
		{"host-port", map[string]string{"RERANKER_HOST": "h", "RERANKER_PORT": "1234"}, "http://h:1234"},
		{"default", map[string]string{}, "http://127.0.0.1:11453"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			get := func(k string) string { return tc.env[k] }
			if got := ResolveRerankerURL(get); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

// stubReranker is a controllable Reranker for chain tests.
type stubReranker struct {
	name      string
	available bool
	scores    []RerankScore
	err       error
	called    bool
}

func (s *stubReranker) Name() string                   { return s.name }
func (s *stubReranker) Available(context.Context) bool { return s.available }
func (s *stubReranker) Rerank(_ context.Context, _ string, _ []RerankCandidate) ([]RerankScore, error) {
	s.called = true
	return s.scores, s.err
}

func TestRerankerChainPrefersFirstAvailable(t *testing.T) {
	cross := &stubReranker{name: "cross", available: true, scores: []RerankScore{{ID: "a", Score: 1}}}
	llm := &stubReranker{name: "llm", available: true, scores: []RerankScore{{ID: "b", Score: 1}}}
	chain := NewRerankerChain(cross, llm)

	if got := chain.ActiveName(context.Background()); got != "cross" {
		t.Fatalf("active = %q, want cross", got)
	}
	_, err := chain.Rerank(context.Background(), "q", []RerankCandidate{{ID: "a"}})
	if err != nil {
		t.Fatalf("Rerank: %v", err)
	}
	if !cross.called || llm.called {
		t.Fatalf("expected only cross called; cross=%v llm=%v", cross.called, llm.called)
	}
}

func TestRerankerChainFallsBack(t *testing.T) {
	cross := &stubReranker{name: "cross", available: false}
	llm := &stubReranker{name: "llm", available: true, scores: []RerankScore{{ID: "b", Score: 1}}}
	chain := NewRerankerChain(cross, llm)

	if got := chain.ActiveName(context.Background()); got != "llm" {
		t.Fatalf("active = %q, want llm", got)
	}
	if _, err := chain.Rerank(context.Background(), "q", []RerankCandidate{{ID: "b"}}); err != nil {
		t.Fatalf("Rerank: %v", err)
	}
	if cross.called || !llm.called {
		t.Fatalf("expected only llm called; cross=%v llm=%v", cross.called, llm.called)
	}
}

func TestRerankerChainNoneAvailable(t *testing.T) {
	chain := NewRerankerChain(
		&stubReranker{name: "cross", available: false},
		&stubReranker{name: "llm", available: false},
	)
	if chain.Available(context.Background()) {
		t.Fatal("expected chain unavailable")
	}
	if got := chain.ActiveName(context.Background()); got != "none" {
		t.Fatalf("active = %q, want none", got)
	}
	scores, err := chain.Rerank(context.Background(), "q", []RerankCandidate{{ID: "a"}})
	if err != nil || scores != nil {
		t.Fatalf("expected (nil,nil) when none available, got %v / %v", scores, err)
	}
}

func TestApplyRerank(t *testing.T) {
	hits := []SearchHit{
		{ID: "a", Score: 0.5},
		{ID: "b", Score: 0.4},
		{ID: "c", Score: 0.3},
	}
	// Reranker promotes c, demotes a; b is unscored and keeps trailing order.
	scores := []RerankScore{{ID: "c", Score: 0.99}, {ID: "a", Score: 0.10}}
	got := ApplyRerank(hits, scores)
	if got[0].ID != "c" || got[1].ID != "a" {
		t.Fatalf("order = %s,%s,%s; want c,a,b", got[0].ID, got[1].ID, got[2].ID)
	}
	if got[2].ID != "b" {
		t.Fatalf("unscored hit b should trail; got %s", got[2].ID)
	}
	if got[0].Score != 0.99 {
		t.Fatalf("reranked score not applied: %v", got[0].Score)
	}
}

func TestApplyRerankNoScores(t *testing.T) {
	hits := []SearchHit{{ID: "a"}, {ID: "b"}}
	got := ApplyRerank(hits, nil)
	if len(got) != 2 || got[0].ID != "a" {
		t.Fatalf("expected unchanged order, got %+v", got)
	}
}
