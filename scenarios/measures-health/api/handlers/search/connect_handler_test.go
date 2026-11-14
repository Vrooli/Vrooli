package search

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	measures "github.com/vrooli/measures-go"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/search"
)

type fakeSearcher struct {
	hits    []*measures.MeasureHit
	matcher string
	err     error
}

func (f *fakeSearcher) Query(_ context.Context, _ string, _ int) ([]*measures.MeasureHit, string, error) {
	return f.hits, f.matcher, f.err
}

func (f *fakeSearcher) Status(context.Context) (bool, bool, bool, int, string) {
	return len(f.hits) > 0, false, false, len(f.hits), "lexical"
}

func (f *fakeSearcher) IndexTimestamp() time.Time {
	return time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
}

func executedHit() *measures.MeasureHit {
	return &measures.MeasureHit{
		MeasureID:     "backlog.completed",
		Scenario:      "swarm-manager",
		Params:        map[string]string{"window": "this_week"},
		Answer:        "42 backlog items completed (this_week)",
		Effect:        "read",
		ExecutedQuery: "SELECT count(*) FROM backlog WHERE done",
		Confidence:    1.0,
		Score:         0.91,
	}
}

func TestSearch_MapsHitToProto(t *testing.T) {
	h := NewConnectHandler(Deps{Searcher: &fakeSearcher{hits: []*measures.MeasureHit{executedHit()}, matcher: "lexical"}})
	resp, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "how many backlog items this week", Limit: 1}))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Msg.GetMatcher() != "lexical" {
		t.Fatalf("expected matcher lexical, got %q", resp.Msg.GetMatcher())
	}
	if len(resp.Msg.GetResults()) != 1 {
		t.Fatalf("expected one result, got %d", len(resp.Msg.GetResults()))
	}
	r := resp.Msg.GetResults()[0]
	if r.GetScore() != 0.91 {
		t.Fatalf("expected outer score 0.91, got %v", r.GetScore())
	}
	m := r.GetMeasure()
	if m.GetMeasureId() != "backlog.completed" || m.GetExecutedQuery() == "" || m.GetAnswer() == "" {
		t.Fatalf("measure carrier not mapped: %+v", m)
	}
}

// TestSearch_WireShapeMatchesSearchHubAdapter is the integration guarantee for
// the federated provider: the protojson the Connect server emits (the exact
// codec connect-go uses) must carry the snake_case measure-carrier keys that
// search-hub's generic result adapter reads verbatim (results_path "results",
// score_field "score", measure_field "measure", then measure.{measure_id,
// scenario, params, answer, needs, effect, executed_query, confidence}). The
// json_name annotations on the proto are what make this hold; this test fails
// loudly if a future proto edit drops them and the carrier silently camelCases.
func TestSearch_WireShapeMatchesSearchHubAdapter(t *testing.T) {
	resp := &searchv1.SearchResponse{
		Matcher: "lexical",
		Results: []*searchv1.MeasureResult{{Score: 0.91, Measure: measureHitToProto(executedHit())}},
	}
	wire, err := protojson.Marshal(resp)
	if err != nil {
		t.Fatalf("protojson marshal: %v", err)
	}

	var root map[string]any
	if err := json.Unmarshal(wire, &root); err != nil {
		t.Fatalf("decode wire: %v", err)
	}
	results, ok := root["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("wire must carry a `results` array; got %s", wire)
	}
	item := results[0].(map[string]any)
	if _, ok := item["score"]; !ok {
		t.Fatalf("result item missing `score`: %s", wire)
	}
	measure, ok := item["measure"].(map[string]any)
	if !ok {
		t.Fatalf("result item missing `measure` object: %s", wire)
	}
	// These are the EXACT snake_case keys search-hub's decodeMeasureHit reads.
	for _, key := range []string{"measure_id", "scenario", "params", "answer", "effect", "executed_query", "confidence"} {
		if _, ok := measure[key]; !ok {
			t.Fatalf("measure carrier missing snake_case key %q (camelCase regression?): %s", key, wire)
		}
	}
	if measure["measure_id"] != "backlog.completed" {
		t.Fatalf("measure_id wrong on the wire: %v", measure["measure_id"])
	}
	if measure["executed_query"] == "" {
		t.Fatalf("executed_query must be populated on the wire: %s", wire)
	}
}

func TestSearch_NilSearcherUnimplemented(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "x"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("expected Unimplemented, got %v", err)
	}
	_, err = h.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("expected Unimplemented for Status, got %v", err)
	}
}

func TestSearch_QueryErrorDegradesToEmpty(t *testing.T) {
	h := NewConnectHandler(Deps{Searcher: &fakeSearcher{matcher: "lexical", err: errors.New("executor unreachable")}})
	resp, err := h.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "how many backlog items"}))
	if err != nil {
		t.Fatalf("a provider error must degrade to an empty group, not fail the query: %v", err)
	}
	if len(resp.Msg.GetResults()) != 0 {
		t.Fatalf("expected empty results on degrade, got %d", len(resp.Msg.GetResults()))
	}
	if resp.Msg.GetMatcher() != "lexical" {
		t.Fatalf("expected matcher label carried through degrade, got %q", resp.Msg.GetMatcher())
	}
}

func TestSearch_StatusMapsFields(t *testing.T) {
	h := NewConnectHandler(Deps{Searcher: &fakeSearcher{hits: []*measures.MeasureHit{executedHit()}}})
	resp, err := h.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !resp.Msg.GetAvailable() || resp.Msg.GetIndexedCount() != 1 || resp.Msg.GetMatcher() != "lexical" {
		t.Fatalf("status mapping wrong: %+v", resp.Msg)
	}
}
