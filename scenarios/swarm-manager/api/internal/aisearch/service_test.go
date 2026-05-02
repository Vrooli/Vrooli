package aisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/testutil/assertx"
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

type fakeInitReader struct {
	items  []initiatives.Initiative
	loaded int32
}

func (f *fakeInitReader) List() ([]initiatives.Initiative, error) {
	atomic.AddInt32(&f.loaded, 1)
	return f.items, nil
}

func (f *fakeInitReader) Get(name string) (*initiatives.Initiative, error) {
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

// fakeOllamaServer responds to POST /api/embeddings with a fixed vector.
func fakeOllamaServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
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
	ollama := fakeOllamaServer(t)
	defer ollama.Close()

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

	embedder := NewEmbedder(ollama.URL, "nomic-embed-text")
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
	// Broken ollama → embed fails → fallback path.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()

	embedder := NewEmbedder(bad.URL, "nomic-embed-text")
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
				ollama := fakeOllamaServer(t)
				t.Cleanup(ollama.Close)
				// Empty qdrant result set → vector path returns zero matches.
				qStub := &qdrantStub{}
				qServer := httptest.NewServer(qStub.handler(t))
				t.Cleanup(qServer.Close)
				return NewService(
					NewEmbedder(ollama.URL, "nomic-embed-text"),
					NewVectorStore(qServer.URL, "", "b", 3),
					NewVectorStore(qServer.URL, "", "i", 3),
					nil, nil, 0.5,
				)
			},
		},
		{
			name: "fallback-text-search-returns-nil",
			svc: func() *Service {
				bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				t.Cleanup(bad.Close)
				svc := NewService(NewEmbedder(bad.URL, "nomic-embed-text"), nil, nil, nil, nil, 0)
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
		{Entity: EntityInitiative, Score: 0.7, Payload: map[string]interface{}{
			"name": "audio-platform",
		}},
	}
	got := applyFilters(results, SearchFilters{TargetScenario: "web-console"})
	if len(got) != 1 {
		t.Errorf("initiative entities must pass target_scenario filter; got %+v", got)
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

func TestApplyFilters_InitiativeOnlyAppliesToBacklog(t *testing.T) {
	results := []AISearchResult{
		{Entity: EntityBacklog, Score: 0.9, Payload: map[string]interface{}{"initiative": "obs"}},
		{Entity: EntityBacklog, Score: 0.8, Payload: map[string]interface{}{"initiative": "other"}},
		{Entity: EntityInitiative, Score: 0.7, Payload: map[string]interface{}{"name": "obs"}},
	}
	got := applyFilters(results, SearchFilters{Initiative: "obs"})
	if len(got) != 2 {
		t.Fatalf("expected backlog item in 'obs' and all initiatives, got %d: %+v", len(got), got)
	}
}

func TestService_GetStatus_Available(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{count: 7}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(
		NewEmbedder(ollama.URL, "nomic-embed-text"),
		NewVectorStore(qServer.URL, "", "b", 3),
		NewVectorStore(qServer.URL, "", "i", 3),
		&fakeBacklogReader{items: make([]backlog.BacklogItem, 4)},
		&fakeInitReader{items: make([]initiatives.Initiative, 3)},
		0,
	)
	st := svc.GetStatus(context.Background())
	if !st.Available {
		t.Errorf("expected available, got %+v", st)
	}
	if !st.Ollama || !st.Qdrant {
		t.Errorf("expected both systems ok, got %+v", st)
	}
	if st.IndexedBacklog != 7 || st.IndexedInitiatives != 7 {
		t.Errorf("expected counts 7 each (stub returns same), got %+v", st)
	}
	if st.OnDiskBacklog != 4 || st.OnDiskInitiatives != 3 {
		t.Errorf("on-disk counts wrong: %+v", st)
	}
}

func TestService_GetStatus_Unavailable(t *testing.T) {
	svc := NewService(NewEmbedder("", ""), NewVectorStore("", "", "b", 3), NewVectorStore("", "", "i", 3), nil, nil, 0)
	st := svc.GetStatus(context.Background())
	if st.Available {
		t.Error("expected unavailable")
	}
	if !strings.Contains(st.Message, "Ollama") || !strings.Contains(st.Message, "Qdrant") {
		t.Errorf("expected message to mention both; got %q", st.Message)
	}
}

func TestService_NeedsReindex_MatchesCount(t *testing.T) {
	qStub := &qdrantStub{count: 2}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(nil,
		NewVectorStore(qServer.URL, "", "b", 3),
		NewVectorStore(qServer.URL, "", "i", 3),
		&fakeBacklogReader{items: make([]backlog.BacklogItem, 2)},
		&fakeInitReader{items: make([]initiatives.Initiative, 2)},
		0,
	)
	needs, indexed, disk, err := svc.NeedsReindex(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stub returns same count for both collections: 2+2=4 indexed; disk 2+2=4
	if needs {
		t.Errorf("expected no reindex needed; indexed=%d disk=%d", indexed, disk)
	}
}

func TestService_NeedsReindex_Divergence(t *testing.T) {
	qStub := &qdrantStub{count: 0}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(nil,
		NewVectorStore(qServer.URL, "", "b", 3),
		NewVectorStore(qServer.URL, "", "i", 3),
		&fakeBacklogReader{items: make([]backlog.BacklogItem, 5)},
		&fakeInitReader{items: make([]initiatives.Initiative, 0)},
		0,
	)
	needs, _, _, err := svc.NeedsReindex(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !needs {
		t.Error("expected reindex needed when indexed=0 and disk=5")
	}
}

func TestService_IndexBacklogItem_Upserts(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(
		NewEmbedder(ollama.URL, "nomic-embed-text"),
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

func TestService_ReindexAll_BothCollections(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(
		NewEmbedder(ollama.URL, "nomic-embed-text"),
		NewVectorStore(qServer.URL, "", "b", 3),
		NewVectorStore(qServer.URL, "", "i", 3),
		&fakeBacklogReader{items: []backlog.BacklogItem{
			{Name: "a", Title: "A", Kind: backlog.KindIdea},
			{Name: "b", Title: "B", Kind: backlog.KindExecute},
		}},
		&fakeInitReader{items: []initiatives.Initiative{
			{Name: "obs", Title: "Obs"},
		}},
		0,
	)
	resp, err := svc.ReindexAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Indexed != 3 {
		t.Errorf("expected 3 indexed, got %d", resp.Indexed)
	}
	if atomic.LoadInt32(&qStub.upsertCalls) != 3 {
		t.Errorf("expected 3 upserts, got %d", qStub.upsertCalls)
	}
}

func TestService_StartReindex_RunsAndCompletes(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(
		NewEmbedder(ollama.URL, "nomic-embed-text"),
		NewVectorStore(qServer.URL, "", "b", 3),
		nil,
		&fakeBacklogReader{items: []backlog.BacklogItem{{Name: "a", Title: "A", Kind: backlog.KindIdea}}},
		nil, 0,
	)
	_, started := svc.StartReindex()
	if !started {
		t.Fatal("expected reindex to start")
	}
	assertx.Eventually(t, 2*time.Second, "reindex completion", func() bool {
		st := svc.ReindexStatus()
		if !st.Running {
			if st.Indexed != 1 {
				t.Errorf("expected 1 indexed, got %d", st.Indexed)
			}
			return true
		}
		return false
	})
}

func TestService_StartReindex_SingletonSemantics(t *testing.T) {
	// Slow ollama so the first reindex is still running when we call again.
	// This fixed sleep lives in the fake upstream, not the assertion path.
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(embeddingResponse{Embedding: []float64{0.1}})
	}))
	defer slow.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc := NewService(
		NewEmbedder(slow.URL, "nomic-embed-text"),
		NewVectorStore(qServer.URL, "", "b", 1),
		nil,
		&fakeBacklogReader{items: []backlog.BacklogItem{{Name: "a", Kind: backlog.KindIdea}}},
		nil, 0,
	)
	_, started1 := svc.StartReindex()
	_, started2 := svc.StartReindex()
	if !started1 {
		t.Fatal("expected first start to succeed")
	}
	if started2 {
		t.Error("expected second start to be rejected while first is running")
	}
	// Let the first finish so the test doesn't leak goroutines.
	assertx.Eventually(t, 2*time.Second, "first singleton reindex cleanup", func() bool {
		return !svc.ReindexStatus().Running
	})
}
