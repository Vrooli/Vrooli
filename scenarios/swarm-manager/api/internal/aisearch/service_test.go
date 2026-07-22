package aisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

// --- Test doubles ---

type fakeBacklogReader struct {
	items  []backlog.BacklogItem
	loaded int32
}

func (f *fakeBacklogReader) LoadAll() ([]backlog.BacklogItem, error) {
	atomic.AddInt32(&f.loaded, 1)
	return f.items, nil
}

func (f *fakeBacklogReader) LoadItem(_ backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	for _, it := range f.items {
		if it.Name == name {
			return it, nil
		}
	}
	return backlog.BacklogItem{}, nil
}

type fakeGoalReader struct {
	items  []goals.Goal
	loaded int32
}

func (f *fakeGoalReader) List() ([]goals.Goal, error) {
	atomic.AddInt32(&f.loaded, 1)
	return f.items, nil
}

func (f *fakeGoalReader) Get(name string) (*goals.Goal, error) {
	for i, it := range f.items {
		if it.Name == name {
			return &f.items[i], nil
		}
	}
	return nil, nil
}

type fakeTextSearcher struct {
	results []AISearchResult
	calls   int32
}

func (f *fakeTextSearcher) Search(_ context.Context, _ string, _ EntityType, _ int, _ SearchFilters) ([]AISearchResult, error) {
	atomic.AddInt32(&f.calls, 1)
	return f.results, nil
}

// fakeEmbedderOK returns an Embedder that always yields a fixed 3-dim vector.
// Replaces the old httptest fake-ollama; the production embedder shells out to
// resource-ollama, so tests inject a runner stub instead.
func fakeEmbedderOK() Embedder {
	return newEmbedderWithRunner("embedding.default", func(_ context.Context, _ []string, _ []byte) ([]byte, error) {
		return []byte(`{"embedding":[0.1,0.2,0.3]}`), nil
	})
}

// fakeEmbedderErr returns an Embedder whose every Embed call fails. Replaces
// the old "broken httptest server" pattern.
func fakeEmbedderErr() Embedder {
	return newEmbedderWithRunner("embedding.default", func(_ context.Context, _ []string, _ []byte) ([]byte, error) {
		return nil, fmt.Errorf("HTTP 500: ollama down")
	})
}

// fakeQdrantServer is a minimal stub that responds to collection existence,
// count, search, and upsert/delete operations with canned data.
type qdrantStub struct {
	count        int
	searchResult []struct {
		ID      interface{}
		Score   float64
		Payload map[string]interface{}
	}
	upsertCalls int32
	deleteCalls int32
}

func (q *qdrantStub) handler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/collections" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/points/count"):
			_ = json.NewEncoder(w).Encode(countResponse{Result: struct {
				Count int `json:"count"`
			}{Count: q.count}})
		case strings.Contains(r.URL.Path, "/points/search"):
			resp := searchResponse{}
			for _, sr := range q.searchResult {
				resp.Result = append(resp.Result, struct {
					ID      interface{}            `json:"id"`
					Score   float64                `json:"score"`
					Payload map[string]interface{} `json:"payload"`
				}{ID: sr.ID, Score: sr.Score, Payload: sr.Payload})
			}
			_ = json.NewEncoder(w).Encode(resp)
		case strings.Contains(r.URL.Path, "/points/delete"):
			atomic.AddInt32(&q.deleteCalls, 1)
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/points") && r.Method == http.MethodPut:
			atomic.AddInt32(&q.upsertCalls, 1)
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/collections/") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/collections/") && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("unhandled qdrant request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}
	})
}

// --- Service tests ---

func TestService_Search_RequiresQuery(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil, 0)
	_, err := s.Search(context.Background(), AISearchRequest{Query: ""})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestService_Search_InvalidEntity(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil, 0)
	_, err := s.Search(context.Background(), AISearchRequest{Query: "x", Entity: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid entity")
	}
}

func TestService_Search_Both_Success(t *testing.T) {
	qStub := &qdrantStub{
		searchResult: []struct {
			ID      interface{}
			Score   float64
			Payload map[string]interface{}
		}{
			{ID: "id-a", Score: 0.9, Payload: map[string]interface{}{"name": "alpha", "status": "ready"}},
		},
	}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	embedder := fakeEmbedderOK()
	backlogVS := NewVectorStore(qServer.URL, "", "sm-backlog", 3)
	initVS := NewVectorStore(qServer.URL, "", "sm-init", 3)
	svc := NewService(embedder, backlogVS, initVS, nil, nil, 0.5)

	resp, err := svc.Search(context.Background(), AISearchRequest{Query: "retry", Entity: EntityBoth, Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Fallback != FallbackNone {
		t.Errorf("expected Fallback=none, got %s", resp.Fallback)
	}
	// Both entities queried → 2 results total from the stub (same payload twice)
	if resp.Total < 1 {
		t.Errorf("expected at least 1 result, got %d", resp.Total)
	}
	if resp.Results[0].ID != "alpha" {
		t.Errorf("expected first result ID to be payload name 'alpha', got %s", resp.Results[0].ID)
	}
}

func TestService_Search_FallbackToTextSearcher(t *testing.T) {
	// Broken embedder → embed fails → fallback path.
	embedder := fakeEmbedderErr()
	text := &fakeTextSearcher{results: []AISearchResult{{Entity: EntityBacklog, ID: "alpha", Score: 0.5}}}
	svc := NewService(embedder, nil, nil, nil, nil, 0)
	svc.SetTextSearcher(text)

	resp, err := svc.Search(context.Background(), AISearchRequest{Query: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Fallback != FallbackTextSearch {
		t.Errorf("expected Fallback=text-search, got %s", resp.Fallback)
	}
	if len(resp.Results) != 1 || resp.Results[0].ID != "alpha" {
		t.Errorf("unexpected results: %+v", resp.Results)
	}
	if atomic.LoadInt32(&text.calls) != 1 {
		t.Errorf("expected text searcher called once, got %d", text.calls)
	}
}

func TestService_Search_FallbackUnavailable(t *testing.T) {
	// No embedder, no text searcher → empty unavailable response.
	svc := NewService(nil, nil, nil, nil, nil, 0)
	resp, err := svc.Search(context.Background(), AISearchRequest{Query: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Fallback != FallbackUnavailable {
		t.Errorf("expected Fallback=unavailable, got %s", resp.Fallback)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected no results, got %d", len(resp.Results))
	}
}

// The wire contract is "results is always an array." A nil Go slice marshals
// to JSON null, which crashes clients that call `resp.results.length`. These
// tests pin the contract at the JSON level for every response path.
func TestService_Search_ResultsAlwaysMarshalsAsArray(t *testing.T) {
	cases := []struct {
		name string
		svc  func() *Service
	}{
		{
			name: "vector-path-with-zero-matches",
			svc: func() *Service {
				// Empty qdrant result set → vector path returns zero matches.
				qStub := &qdrantStub{}
				qServer := httptest.NewServer(qStub.handler(t))
				t.Cleanup(qServer.Close)
				return NewService(
					fakeEmbedderOK(),
					NewVectorStore(qServer.URL, "", "b", 3),
					NewVectorStore(qServer.URL, "", "i", 3),
					nil, nil, 0.5,
				)
			},
		},
		{
			name: "fallback-text-search-returns-nil",
			svc: func() *Service {
				svc := NewService(fakeEmbedderErr(), nil, nil, nil, nil, 0)
				// Text searcher that returns a typed nil slice — the exact shape
				// that previously slipped through to JSON as `null`.
				svc.SetTextSearcher(&fakeTextSearcher{results: nil})
				return svc
			},
		},
		{
			name: "fallback-unavailable",
			svc: func() *Service {
				return NewService(nil, nil, nil, nil, nil, 0)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := tc.svc().Search(context.Background(), AISearchRequest{Query: "x"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Results == nil {
				t.Fatalf("Results is a nil slice — will serialize to JSON null")
			}
			raw, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			s := string(raw)
			if strings.Contains(s, `"results":null`) {
				t.Errorf("response serialized results as null — breaks clients doing results.length: %s", s)
			}
			if !strings.Contains(s, `"results":[]`) {
				t.Errorf("expected empty results to serialize as `[]`; got: %s", s)
			}
		})
	}
}

func TestApplyFilters_ExcludesArchivedByDefault(t *testing.T) {
	results := []AISearchResult{
		{Entity: EntityBacklog, Score: 0.9, Payload: map[string]interface{}{"archived": true}},
		{Entity: EntityBacklog, Score: 0.8, Payload: map[string]interface{}{"archived": false}},
	}
	got := applyFilters(results, SearchFilters{})
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
}

func TestApplyFilters_IncludeArchived(t *testing.T) {
	results := []AISearchResult{
		{Entity: EntityBacklog, Score: 0.9, Payload: map[string]interface{}{"archived": true}},
	}
	got := applyFilters(results, SearchFilters{IncludeArchived: true})
	if len(got) != 1 {
		t.Fatalf("expected archived result retained, got %d", len(got))
	}
}

func TestApplyFilters_StatusAndKind(t *testing.T) {
	results := []AISearchResult{
		{Entity: EntityBacklog, Score: 0.9, Payload: map[string]interface{}{"status": "ready", "kind": "execute"}},
		{Entity: EntityBacklog, Score: 0.8, Payload: map[string]interface{}{"status": "backlog", "kind": "idea"}},
		{Entity: EntityBacklog, Score: 0.7, Payload: map[string]interface{}{"status": "ready", "kind": "idea"}},
	}
	got := applyFilters(results, SearchFilters{Status: []string{"ready"}, Kind: []string{"execute"}})
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(got), got)
	}
	if got[0].Payload["status"] != "ready" || got[0].Payload["kind"] != "execute" {
		t.Errorf("unexpected result: %+v", got[0])
	}
}

func TestApplyFilters_TargetScenario_StringSlicePayload(t *testing.T) {
	results := []AISearchResult{
		{Entity: EntityBacklog, Score: 0.9, Payload: map[string]interface{}{
			"target_scenarios": []string{"web-console", "command-center"},
		}},
		{Entity: EntityBacklog, Score: 0.8, Payload: map[string]interface{}{
			"target_scenarios": []string{"command-center"},
		}},
	}
	got := applyFilters(results, SearchFilters{TargetScenario: "web-console"})
	if len(got) != 1 || got[0].Score != 0.9 {
		t.Errorf("expected only the web-console-targeting result, got %+v", got)
	}
}

func TestApplyFilters_TargetScenario_InterfaceSlicePayload(t *testing.T) {
	// Round-tripping through Qdrant turns []string into []interface{}; the
	// filter must handle both shapes.
	results := []AISearchResult{
		{Entity: EntityBacklog, Score: 0.9, Payload: map[string]interface{}{
			"target_scenarios": []interface{}{"web-console"},
		}},
		{Entity: EntityBacklog, Score: 0.7, Payload: map[string]interface{}{
			"target_scenarios": []interface{}{"other"},
		}},
	}
	got := applyFilters(results, SearchFilters{TargetScenario: "web-console"})
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(got), got)
	}
}

func TestApplyFilters_TargetScenario_OnlyAppliesToBacklog(t *testing.T) {
	results := []AISearchResult{
		{Entity: EntityGoal, Score: 0.7, Payload: map[string]interface{}{
			"name": "audio-platform",
		}},
	}
	got := applyFilters(results, SearchFilters{TargetScenario: "web-console"})
	if len(got) != 1 {
		t.Errorf("goal entities must pass target_scenario filter; got %+v", got)
	}
}

func TestNormalizeFilters_FixKindDefaultsToIncludeArchived(t *testing.T) {
	got := normalizeFilters(SearchFilters{Kind: []string{"fix"}})
	if !got.IncludeArchived {
		t.Errorf("expected IncludeArchived=true for kind=[fix], got %+v", got)
	}
}

func TestNormalizeFilters_NonFixKindNotPromoted(t *testing.T) {
	got := normalizeFilters(SearchFilters{Kind: []string{"execute"}})
	if got.IncludeArchived {
		t.Errorf("IncludeArchived must remain false for kind=[execute], got %+v", got)
	}
}

func TestNormalizeFilters_MultiKindNotPromoted(t *testing.T) {
	got := normalizeFilters(SearchFilters{Kind: []string{"fix", "execute"}})
	if got.IncludeArchived {
		t.Errorf("multi-kind must not auto-promote IncludeArchived, got %+v", got)
	}
}

func TestNormalizeFilters_ExplicitFalseNotOverwrittenForFix(t *testing.T) {
	// Edge case: an explicit IncludeArchived:false on fix is currently
	// indistinguishable from the zero value (Go's bool default). The product
	// rule chooses to favor "fix history wants archived" — that's the chosen
	// semantics. This test pins the behavior so a future change is intentional.
	got := normalizeFilters(SearchFilters{Kind: []string{"fix"}, IncludeArchived: false})
	if !got.IncludeArchived {
		t.Errorf("kind=[fix] always promotes IncludeArchived; got %+v", got)
	}
}

func TestApplyFilters_GoalMatchesGoalPayload(t *testing.T) {
	results := []AISearchResult{
		{Entity: EntityBacklog, Score: 0.9, Payload: map[string]interface{}{"milestone": "obs"}},
		{Entity: EntityBacklog, Score: 0.8, Payload: map[string]interface{}{"milestone": "other"}},
		{Entity: EntityGoal, Score: 0.7, Payload: map[string]interface{}{"name": "obs"}},
	}
	got := applyFilters(results, SearchFilters{Goal: "obs"})
	if len(got) != 3 {
		t.Fatalf("expected all backlogs plus matching goal, got %d: %+v", len(got), got)
	}
}

func TestService_GetStatus_Available(t *testing.T) {
	qStub := &qdrantStub{count: 7}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(
		fakeEmbedderOK(),
		NewVectorStore(qServer.URL, "", "b", 3),
		NewVectorStore(qServer.URL, "", "i", 3),
		&fakeBacklogReader{items: make([]backlog.BacklogItem, 4)},
		&fakeGoalReader{items: make([]goals.Goal, 3)},
		0,
	)
	st := svc.GetStatus(context.Background())
	if !st.Available {
		t.Errorf("expected available, got %+v", st)
	}
	if !st.Ollama || !st.Qdrant {
		t.Errorf("expected both systems ok, got %+v", st)
	}
	if st.IndexedBacklog != 7 || st.IndexedGoals != 7 {
		t.Errorf("expected counts 7 each (stub returns same), got %+v", st)
	}
	if st.OnDiskBacklog != 4 || st.OnDiskGoals != 3 {
		t.Errorf("on-disk counts wrong: %+v", st)
	}
}

func TestService_GetStatus_Unavailable(t *testing.T) {
	svc := NewService(fakeEmbedderErr(), NewVectorStore("", "", "b", 3), NewVectorStore("", "", "i", 3), nil, nil, 0)
	st := svc.GetStatus(context.Background())
	if st.Available {
		t.Error("expected unavailable")
	}
	if !strings.Contains(st.Message, "Ollama") || !strings.Contains(st.Message, "Qdrant") {
		t.Errorf("expected message to mention both; got %q", st.Message)
	}
}

// NeedsReindex behavior is now lifted into Reconciler.Plan / DriftReport;
// see reconciler_test.go for the convergence + drift coverage.

func TestService_IndexBacklogItem_Upserts(t *testing.T) {
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(
		fakeEmbedderOK(),
		NewVectorStore(qServer.URL, "", "b", 3),
		nil, nil, nil, 0,
	)
	err := svc.IndexBacklogItem(context.Background(), backlog.BacklogItem{
		Name: "alpha", Title: "Alpha", Kind: backlog.KindExecute,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&qStub.upsertCalls) != 1 {
		t.Errorf("expected 1 upsert, got %d", qStub.upsertCalls)
	}
}

func TestService_DeleteBacklogItem_Calls(t *testing.T) {
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(nil, NewVectorStore(qServer.URL, "", "b", 3), nil, nil, nil, 0)
	err := svc.DeleteBacklogItem(context.Background(), backlog.KindExecute, "alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&qStub.deleteCalls) != 1 {
		t.Errorf("expected 1 delete, got %d", qStub.deleteCalls)
	}
}

// Reindex/StartReindex/StartReindex-singleton coverage moved to
// reconciler_test.go (TestReconciler_Plan_*, TestReconciler_RunOnce_*,
// TestReconciler_RunOnce_SingletonWhileRunning).
