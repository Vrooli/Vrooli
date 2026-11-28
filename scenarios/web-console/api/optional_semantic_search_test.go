package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeSemanticSearchClient struct {
	request semanticArchiveSearchRequest
	result  *semanticArchiveSearchResult
	err     error
}

func (f *fakeSemanticSearchClient) SearchConversations(_ context.Context, req semanticArchiveSearchRequest) (*semanticArchiveSearchResult, error) {
	f.request = req
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestOptionalSemanticSearchUsesSemanticModeAndNativeCoverage(t *testing.T) {
	want := &semanticArchiveSearchResult{
		Hits:     []*semanticArchiveSearchHit{{StableHitID: "synthetic:1", Snippet: "generated archive evidence"}},
		Coverage: &semanticArchiveSearchCoverage{CanonicalVisibleMessages: 4, SemanticDocuments: 4, SemanticRatio: 1},
		TookMs:   3,
	}
	client := &fakeSemanticSearchClient{result: want}
	got, err := runOptionalSemanticSearch(context.Background(), client, semanticArchiveSearchRequest{Query: "find the synthetic design discussion", Limit: 7})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("response = %#v, want same coverage response %#v", got, want)
	}
	if client.request.Query != "find the synthetic design discussion" || client.request.Limit != 7 {
		t.Fatalf("semantic search request = %+v", client.request)
	}
	if wantHarnesses := []string{"claude", "codex", "grok", "opencode", "gemini-cli", "qwen-code", "antigravity", "cursor"}; !reflect.DeepEqual(client.request.Harnesses, wantHarnesses) {
		t.Fatalf("harness coverage filter = %v, want %v", client.request.Harnesses, wantHarnesses)
	}
}

func TestNormalizeSemanticHarnessesCoversAllAdmittedNativeArchives(t *testing.T) {
	want := []string{"claude", "codex", "grok", "opencode", "gemini-cli", "qwen-code", "antigravity", "cursor"}
	got, err := normalizeSemanticHarnesses(nil)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("default semantic harnesses = %v, err=%v; want %v", got, err, want)
	}
	got, err = normalizeSemanticHarnesses([]string{" CURSOR ", "gemini-cli", "claude", "cursor"})
	if err != nil || !reflect.DeepEqual(got, []string{"cursor", "gemini-cli", "claude"}) {
		t.Fatalf("normalized semantic harnesses = %v, err=%v", got, err)
	}
	if _, err := normalizeSemanticHarnesses([]string{"unknown-format"}); err == nil {
		t.Fatal("unknown format is not an admitted native archive")
	}
}

func TestOptionalSemanticSearchConnectJSONContractAndCoverage(t *testing.T) {
	var request struct {
		Query   string `json:"query"`
		Mode    string `json:"mode"`
		Filters struct {
			Harnesses []string `json:"harnesses"`
		} `json:"filters"`
		PageSize int32 `json:"pageSize"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != agentManagerSearchConversationsProcedure || r.Method != http.MethodPost {
			t.Errorf("Connect endpoint = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Connect-Protocol-Version") != "1" {
			t.Errorf("Connect headers = %+v", r.Header)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":[{"stableHitId":"synthetic:design-1","snippet":"generated design discussion","provenance":{"harness":"claude","sourceSessionId":"generated-session"}}],"modeUsed":"CONVERSATION_SEARCH_MODE_SEMANTIC","coverage":{"canonicalVisibleMessages":"12","semanticDocuments":"12","semanticRatio":1},"tookMs":"2"}`))
	}))
	defer server.Close()
	client := &connectJSONSemanticSearchClient{http: server.Client(), baseURL: server.URL}
	got, err := runOptionalSemanticSearch(context.Background(), client, semanticArchiveSearchRequest{Query: "find a design choice", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if request.Query != "find a design choice" || request.Mode != "CONVERSATION_SEARCH_MODE_SEMANTIC" || request.PageSize != 10 || len(request.Filters.Harnesses) != 6 {
		t.Fatalf("wire request = %+v", request)
	}
	if len(got.Hits) != 1 || got.Coverage.SemanticRatio != 1 || uint64(got.Coverage.SemanticDocuments) != 12 || uint64(got.TookMs) != 2 {
		t.Fatalf("semantic JSON response = %+v", got)
	}
}

func TestOptionalSemanticSearchDegradesWithoutAgentManager(t *testing.T) {
	if _, err := runOptionalSemanticSearch(context.Background(), nil, semanticArchiveSearchRequest{Query: "generated local query"}); err == nil {
		t.Fatal("missing Agent Manager unexpectedly returned semantic results")
	}
	failed := &fakeSemanticSearchClient{err: errors.New("local scenario stopped")}
	if _, err := runOptionalSemanticSearch(context.Background(), failed, semanticArchiveSearchRequest{Query: "generated local query"}); err == nil {
		t.Fatal("stopped Agent Manager unexpectedly returned semantic results")
	}
}

func TestOptionalSemanticSearchRejectsUnregisteredHarness(t *testing.T) {
	client := &fakeSemanticSearchClient{result: &semanticArchiveSearchResult{}}
	if _, err := runOptionalSemanticSearch(context.Background(), client, semanticArchiveSearchRequest{Query: "generated", Harnesses: []string{"other"}}); err == nil {
		t.Fatal("unregistered harness was sent to Agent Manager")
	}
	if client.request.Query != "" {
		t.Fatal("invalid harness request reached semantic client")
	}
}

func TestSemanticSearchDropsHitsOutsideRequestedHarnessFilter(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	if _, appended := store.AppendNativeEvent(ctx, "generated-filter-session", "synthetic-seed", "assistant", "generated local archive match", &NativeProvenance{
		Provider: "claude", SessionID: "generated-filter-session", TurnID: "generated-filter-turn", MessageID: "generated:filter-local",
	}); !appended.Appended {
		t.Fatalf("append generated local evidence: %+v", appended)
	}
	client := &fakeSemanticSearchClient{result: &semanticArchiveSearchResult{Coverage: &semanticArchiveSearchCoverage{SemanticDocuments: 1, SemanticRatio: 1}, Hits: []*semanticArchiveSearchHit{
		{StableHitID: "generated:conflicting-harness", Harness: "codex", SourceSession: "generated-claude-session", Provenance: &semanticArchiveSourceIdentity{Harness: "claude", SourceSessionID: "generated-claude-session"}},
		{StableHitID: "generated:conflicting-session", Harness: "claude", SourceSession: "generated-session-a", Provenance: &semanticArchiveSourceIdentity{Harness: "claude", SourceSessionID: "generated-session-b"}},
		{StableHitID: "generated:codex-leak", Harness: "codex", SourceSession: "generated-codex-session"},
		{StableHitID: "generated:provenance-leak", Harness: "claude", Provenance: &semanticArchiveSourceIdentity{Harness: "opencode", SourceSessionID: "generated-opencode-session"}},
		{Harness: "claude", SourceSession: "generated-claude-session"},
		{StableHitID: "generated:claude-hit", Harness: "claude", SourceSession: "generated-claude-session"},
		{StableHitID: "generated:claude-hit", Harness: "claude", SourceSession: "generated-claude-session"},
		{StableHitID: "generated:claude-hit-2", Harness: "claude", SourceSession: "generated-claude-session"},
		{StableHitID: "generated:claude-hit-3", Harness: "claude", SourceSession: "generated-claude-session"},
	}}}
	adapter := newConversationAdapter(&Server{conversations: store})
	adapter.semanticClientFactory = func(context.Context) (semanticSearchClient, error) { return client, nil }
	response, err := adapter.SearchSemantic(ctx, &conversationv1.SearchSemanticRequest{Query: "generated local", Harnesses: []string{"claude"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetLocal().GetMatches()) == 0 {
		t.Fatal("semantic filtering discarded local Web Console results")
	}
	if len(response.GetSemanticHits()) != 2 || response.GetSemanticHits()[0].GetStableHitId() != "generated:claude-hit" || response.GetSemanticHits()[1].GetStableHitId() != "generated:claude-hit-2" {
		t.Fatalf("semantic results escaped source consistency, filter, deduplication, or page-size contract: %+v", response.GetSemanticHits())
	}
}

func TestSemanticSearchTreatsMissingCoverageAsDegraded(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	if _, appended := store.AppendNativeEvent(ctx, "generated-coverage-session", "synthetic-seed", "assistant", "generated local archive match", &NativeProvenance{
		Provider: "claude", SessionID: "generated-coverage-session", TurnID: "generated-coverage-turn", MessageID: "generated:coverage-local",
	}); !appended.Appended {
		t.Fatalf("append generated local evidence: %+v", appended)
	}
	adapter := newConversationAdapter(&Server{conversations: store})
	adapter.semanticClientFactory = func(context.Context) (semanticSearchClient, error) {
		return &fakeSemanticSearchClient{result: &semanticArchiveSearchResult{Hits: []*semanticArchiveSearchHit{{StableHitID: "generated:unverified"}}}}, nil
	}
	response, err := adapter.SearchSemantic(ctx, &conversationv1.SearchSemanticRequest{Query: "generated local", Harnesses: []string{"claude"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetSemanticAvailable() || !strings.Contains(response.GetSemanticError(), "no coverage") || len(response.GetSemanticHits()) != 0 {
		t.Fatalf("missing coverage was reported as healthy semantic data: %+v", response)
	}
	if len(response.GetLocal().GetMatches()) == 0 {
		t.Fatal("missing semantic coverage discarded local Web Console results")
	}
}

func TestSemanticRPCFixtureUsesGenericConnectProtobufBoundary(t *testing.T) {
	// Keep the optional RPC boundary decoupled from generated Agent Manager Go
	// types so the Web Console module can enforce its inbox ownership rule.
	handler := connect.NewUnaryHandler(agentManagerSearchConversationsProcedure, func(_ context.Context, req *connect.Request[structpb.Struct]) (*connect.Response[structpb.Struct], error) {
		if req.Msg.AsMap()["mode"] != "CONVERSATION_SEARCH_MODE_SEMANTIC" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("wrong mode"))
		}
		response, err := structpb.NewStruct(map[string]any{"tookMs": float64(2), "coverage": map[string]any{"semanticRatio": 1.0}})
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(response), nil
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &connectJSONSemanticSearchClient{http: server.Client(), baseURL: server.URL}
	got, err := client.SearchConversations(context.Background(), semanticArchiveSearchRequest{Query: "generated", Harnesses: []string{"claude"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got.Coverage == nil || got.Coverage.SemanticRatio != 1 || uint64(got.TookMs) != 2 {
		t.Fatalf("generic Connect fixture response = %+v", got)
	}
}
