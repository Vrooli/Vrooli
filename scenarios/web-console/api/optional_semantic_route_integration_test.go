package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
	conversationH "web-console/handlers/conversation"
)

// This local Agent Manager stand-in returns a deterministic dense-ranking
// replay over generated documents. The test measures the full Web Console
// Connect route and its local-store preservation, not model quality.
type generatedSemanticRoute struct {
	expected map[string]map[string]struct{}
	queries  int
}

func (h *generatedSemanticRoute) Handler() http.Handler {
	return connect.NewUnaryHandler(agentManagerSearchConversationsProcedure, func(_ context.Context, req *connect.Request[structpb.Struct]) (*connect.Response[structpb.Struct], error) {
		request := req.Msg.AsMap()
		if request["mode"] != "CONVERSATION_SEARCH_MODE_SEMANTIC" || request["pageSize"] != float64(10) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unexpected semantic request shape: %v", request))
		}
		filters, _ := request["filters"].(map[string]any)
		harnesses, _ := filters["harnesses"].([]any)
		if len(harnesses) != 1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("harness filter = %v", harnesses))
		}
		harness, _ := harnesses[0].(string)
		query, _ := request["query"].(string)
		label := ""
		for candidate := range h.expected {
			candidateLabel, _, _ := strings.Cut(candidate, "/")
			if query == candidateLabel {
				label = candidateLabel
				break
			}
		}
		if label == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unlabeled generated query %q", query))
		}
		relevant := h.expected[label+"/"+harness]
		h.queries++
		hits := make([]any, 0, 10)
		seen := 0
		for id := range relevant {
			if seen == 4 { // Four of five labeled targets: recall@10 is exactly .8.
				break
			}
			hits = append(hits, generatedSemanticHitMap(id, harness, "generated dense hit"))
			seen++
		}
		for i := 0; i < 6; i++ {
			hits = append(hits, generatedSemanticHitMap(fmt.Sprintf("distractor-%s-%s-%d", label, harness, i), harness, "generated distractor"))
		}
		response, err := structpb.NewStruct(map[string]any{
			"hits":     hits,
			"coverage": map[string]any{"canonicalVisibleMessages": float64(100), "catalogDocuments": float64(100), "lexicalDocuments": float64(100), "semanticDocuments": float64(100), "semanticRatio": float64(1)},
			"tookMs":   float64(4),
		})
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(response), nil
	})
}

func generatedSemanticHitMap(id, harness, snippet string) map[string]any {
	return map[string]any{
		"stableHitId": id, "runId": "generated-run-" + harness, "eventId": id,
		"role": "assistant", "occurredAt": "2026-11-22T12:00:00Z", "snippet": snippet, "weak": false,
		"provenance": map[string]any{"harness": harness, "sourceSessionId": "generated-session-" + harness},
	}
}

func TestOptionalSemanticRouteGeneratedCorpusCoverageRecallAndDegradation(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	harnesses := []string{"claude", "codex", "grok", "opencode"}
	labels := []string{"portfolio outcome", "retention cadence", "risk mitigation", "partner pipeline", "target receipt"}
	want := make(map[string]map[string]struct{}, len(labels)*len(harnesses))
	for _, label := range labels {
		for _, harness := range harnesses {
			key := label + "/" + harness
			want[key] = make(map[string]struct{}, 5)
			for n := 0; n < 5; n++ {
				session := "generated-" + harness
				id := fmt.Sprintf("generated:%s:%s:%d", label, harness, n)
				_, appended := store.AppendNativeEvent(ctx, session, "synthetic-seed", "assistant",
					fmt.Sprintf("%s synthetic labeled document %d", label, n),
					&NativeProvenance{Provider: harness, SessionID: session, TurnID: fmt.Sprintf("%s-%d", label, n), MessageID: id})
				if !appended.Appended {
					t.Fatalf("append generated event %s: %+v", id, appended)
				}
				want[key][id] = struct{}{}
			}
		}
	}
	if got := store.CountSessionEvents(ctx, "generated-claude") + store.CountSessionEvents(ctx, "generated-codex") + store.CountSessionEvents(ctx, "generated-grok") + store.CountSessionEvents(ctx, "generated-opencode"); got != 100 {
		t.Fatalf("generated archive corpus events=%d, want 100", got)
	}

	manager := &generatedSemanticRoute{expected: want}
	managerServer := httptest.NewServer(manager.Handler())
	defer managerServer.Close()
	server := &Server{conversations: store}
	adapter := newConversationAdapter(server)
	adapter.semanticClientFactory = func(context.Context) (semanticSearchClient, error) {
		return &connectJSONSemanticSearchClient{http: managerServer.Client(), baseURL: managerServer.URL}, nil
	}
	_, wcRPC := conversationconnect.NewConversationServiceHandler(conversationH.NewConnectHandler(conversationH.Deps{
		Service: adapter, OptionalSearch: adapter, Logger: log.New(ioDiscard{}, "", 0),
	}))
	wcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { wcRPC.ServeHTTP(w, r) }))
	defer wcServer.Close()
	wc := conversationconnect.NewConversationServiceClient(wcServer.Client(), wcServer.URL)
	queryCount, hits, relevant := 0, 0, 0
	routeLatencies := make([]time.Duration, 0, 20)
	for _, label := range labels {
		for _, harness := range harnesses {
			query := label
			started := time.Now()
			response, err := wc.SearchSemantic(ctx, connect.NewRequest(&conversationv1.SearchSemanticRequest{Query: query, Harnesses: []string{harness}, Limit: 10}))
			routeLatencies = append(routeLatencies, time.Since(started))
			if err != nil {
				t.Fatalf("Web Console semantic route for %q: %v", query, err)
			}
			if !response.Msg.GetSemanticAvailable() || response.Msg.GetCoverage().GetSemanticRatio() != 1 || response.Msg.GetCoverage().GetSemanticDocuments() != 100 {
				t.Fatalf("semantic coverage for %q = %+v, error %q", query, response.Msg.GetCoverage(), response.Msg.GetSemanticError())
			}
			if response.Msg.GetSemanticTookMs() > 1200 {
				t.Fatalf("dense leg for %q took %dms, exceeds 1200ms budget", query, response.Msg.GetSemanticTookMs())
			}
			if len(response.Msg.GetLocal().GetMatches()) == 0 {
				t.Fatalf("Web Console own-store hits missing for %q", query)
			}
			if len(response.Msg.GetSemanticHits()) != 10 {
				t.Fatalf("semantic result count for %q = %d, want 10", query, len(response.Msg.GetSemanticHits()))
			}
			gotRelevant := 0
			for _, hit := range response.Msg.GetSemanticHits() {
				if hit.GetHarness() != harness || hit.GetSourceSessionId() != "generated-session-"+harness || hit.GetOccurredAt() != "2026-11-22T12:00:00Z" {
					t.Fatalf("native source identity mapping for %q: %+v", query, hit)
				}
				if _, ok := want[label+"/"+harness][hit.GetStableHitId()]; ok {
					gotRelevant++
				}
			}
			if gotRelevant < 4 {
				t.Fatalf("recall@10 for %q = %d/5; expected generated dense replay to reach >= .8", query, gotRelevant)
			}
			queryCount++
			hits += len(response.Msg.GetSemanticHits())
			relevant += gotRelevant
		}
	}
	sort.Slice(routeLatencies, func(i, j int) bool { return routeLatencies[i] < routeLatencies[j] })
	p95 := routeLatencies[(len(routeLatencies)*95+99)/100-1]
	if manager.queries != 20 || queryCount != 20 {
		t.Fatalf("semantic queries manager=%d route=%d; want 20", manager.queries, queryCount)
	}
	if recall := float64(relevant) / float64(queryCount*5); recall < .8 {
		t.Fatalf("measured generated recall@10 = %.3f, want >= .8", recall)
	}
	if p95 > time.Second {
		t.Fatalf("local Web Console semantic route p95 %s exceeds 1s", p95)
	}

	// Stop the optional integration and prove the same production Connect route
	// still returns own-store matches with an explicit degradation reason.
	managerServer.Close()
	degraded, err := wc.SearchSemantic(ctx, connect.NewRequest(&conversationv1.SearchSemanticRequest{Query: labels[0], Harnesses: []string{"claude"}, Limit: 10}))
	if err != nil {
		t.Fatalf("local search after Agent Manager stop: %v", err)
	}
	if degraded.Msg.GetSemanticAvailable() || degraded.Msg.GetSemanticError() == "" || len(degraded.Msg.GetLocal().GetMatches()) == 0 {
		t.Fatalf("stopped Agent Manager did not preserve local results/degrade: %+v", degraded.Msg)
	}
	t.Logf("generated semantic route: events=100 queries=%d returned_hits=%d recall@10=%.3f reported_coverage=100/100 route_p95=%s dense_leg_budget_ms=1200; Agent Manager is a deterministic local Connect fixture, not live model-quality evidence", queryCount, hits, float64(relevant)/float64(queryCount*5), p95)
}

func TestOptionalSemanticRouteBoundsSlowAgentManagerAndKeepsLocalHits(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	if err := ensureConversationFTS(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStoreWithRepository(NewSQLConversationRepository(db))
	_, appended := store.AppendNativeEvent(ctx, "generated-slow-agent", "synthetic-seed", "assistant", "bounded generated semantic local evidence", &NativeProvenance{
		Provider: "claude", SessionID: "generated-slow-agent", TurnID: "bounded-turn", MessageID: "generated:bounded-semantic",
	})
	if !appended.Appended {
		t.Fatalf("append local evidence: %+v", appended)
	}

	managerRPC := connect.NewUnaryHandler(agentManagerSearchConversationsProcedure, func(ctx context.Context, _ *connect.Request[structpb.Struct]) (*connect.Response[structpb.Struct], error) {
		<-ctx.Done()
		return nil, connect.NewError(connect.CodeUnavailable, ctx.Err())
	})
	managerServer := httptest.NewServer(managerRPC)
	defer managerServer.Close()
	adapter := newConversationAdapter(&Server{conversations: store})
	adapter.semanticClientFactory = func(context.Context) (semanticSearchClient, error) {
		return &connectJSONSemanticSearchClient{http: managerServer.Client(), baseURL: managerServer.URL}, nil
	}
	_, wcRPC := conversationconnect.NewConversationServiceHandler(conversationH.NewConnectHandler(conversationH.Deps{
		Service: adapter, OptionalSearch: adapter, Logger: log.New(ioDiscard{}, "", 0),
	}))
	wcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { wcRPC.ServeHTTP(w, r) }))
	defer wcServer.Close()
	started := time.Now()
	response, err := conversationconnect.NewConversationServiceClient(wcServer.Client(), wcServer.URL).SearchSemantic(ctx, connect.NewRequest(&conversationv1.SearchSemanticRequest{
		Query: "bounded generated semantic", Harnesses: []string{"claude"}, Limit: 10,
	}))
	if err != nil {
		t.Fatalf("slow optional leg should preserve successful local route: %v", err)
	}
	if elapsed := time.Since(started); elapsed < optionalSemanticSearchTimeout || elapsed > 2*optionalSemanticSearchTimeout {
		t.Fatalf("slow Agent Manager route elapsed %s; want bounded near %s", elapsed, optionalSemanticSearchTimeout)
	}
	if response.Msg.GetSemanticAvailable() || response.Msg.GetSemanticError() == "" || len(response.Msg.GetLocal().GetMatches()) != 1 {
		t.Fatalf("slow Agent Manager did not degrade to local evidence: %+v", response.Msg)
	}
}

// ioDiscard avoids exposing test logs while keeping the handler's production
// logger seam non-nil and deterministic.
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
