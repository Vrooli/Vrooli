package search

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"architecture-cartographer/internal/aisearch"
	pkg "github.com/vrooli/ai-go/search"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/search"
)

type fakeSearcher struct {
	resp   *aisearch.SearchResponse
	status aisearch.StatusReport
	gotOps int
}

func (f *fakeSearcher) Search(_ context.Context, _ string, _ int, _ aisearch.SearchMode, opts ...pkg.SearchOption) (*aisearch.SearchResponse, error) {
	f.gotOps = len(opts)
	return f.resp, nil
}

func (f *fakeSearcher) Status(context.Context) aisearch.StatusReport { return f.status }

func TestSearchProjectsHits(t *testing.T) {
	h := NewConnectHandler(Deps{Searcher: &fakeSearcher{resp: &aisearch.SearchResponse{
		Method:   "ai",
		Reranker: "none",
		Results: []aisearch.DomainHit{{
			ID:             "plan-manager/authoring",
			Scenario:       "plan-manager",
			Name:           "authoring",
			Responsibility: "Guided composer wizard.",
			Archetype:      "service",
			Paths:          []string{"api/internal/authoring"},
			Score:          0.82,
			Weak:           false,
		}},
	}}})

	resp, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "how does authoring work in plan-manager", Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.Msg.Results); got != 1 {
		t.Fatalf("got %d results, want 1", got)
	}
	r := resp.Msg.Results[0]
	if r.Id != "plan-manager/authoring" || r.Scenario != "plan-manager" || r.Name != "authoring" {
		t.Fatalf("projection wrong: %+v", r)
	}
	if r.Score != 0.82 {
		t.Fatalf("score = %v, want 0.82", r.Score)
	}
	if resp.Msg.ModeUsed != searchv1.Mode_MODE_AI {
		t.Fatalf("mode_used = %v, want MODE_AI", resp.Msg.ModeUsed)
	}
}

func TestSearchNilSearcherUnimplemented(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "x"}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("want Unimplemented, got %v", err)
	}
}

func TestStatusMapsReport(t *testing.T) {
	h := NewConnectHandler(Deps{Searcher: &fakeSearcher{status: aisearch.StatusReport{
		Available: true, Ollama: true, Qdrant: true, IndexedCount: 42, Reranker: "none",
	}}})
	resp, err := h.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.Available || !resp.Msg.Ollama || !resp.Msg.Qdrant || resp.Msg.IndexedCount != 42 {
		t.Fatalf("status mapping wrong: %+v", resp.Msg)
	}
}

// TestOverrideIgnoredWithoutToken proves the public path: an override header with
// no matching control token is silently dropped (no opts threaded), degrading to
// the ordinary search rather than erroring.
func TestOverrideIgnoredWithoutToken(t *testing.T) {
	fake := &fakeSearcher{resp: &aisearch.SearchResponse{Method: "ai"}}
	h := NewConnectHandler(Deps{Searcher: fake, Overrides: &OverrideGate{Token: func() string { return "secret" }}})
	req := connect.NewRequest(&searchv1.SearchRequest{Query: "x", Limit: 5})
	req.Header().Set(pkg.OverridesHeader, `{"rerank_enabled":true}`)
	// No control-token header → override must be ignored.
	if _, err := h.Search(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if fake.gotOps != 0 {
		t.Fatalf("expected override to be dropped (0 opts), got %d", fake.gotOps)
	}
}
