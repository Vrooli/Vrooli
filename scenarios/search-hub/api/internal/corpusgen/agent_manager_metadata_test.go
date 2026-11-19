package corpusgen_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/proto/searcheval"
	"search-hub/internal/corpusgen"
)

type metadataReader struct {
	pages map[string][]corpusgen.AgentManagerMetadataPage
	seen  map[string][]int64
}

func (r *metadataReader) GetRunEventMetadata(_ context.Context, runID string, after int64, _ int) (corpusgen.AgentManagerMetadataPage, error) {
	if r.seen == nil {
		r.seen = map[string][]int64{}
	}
	r.seen[runID] = append(r.seen[runID], after)
	pages := r.pages[runID]
	index := len(r.seen[runID]) - 1
	return pages[index], nil
}

func TestHarvestAgentManagerCallerSourcesUsesPayloadFreePagedMetadata(t *testing.T) {
	at := time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)
	reader := &metadataReader{pages: map[string][]corpusgen.AgentManagerMetadataPage{
		"run-a": {
			{RunID: "run-a", MetadataOnly: true, HasMore: true, Events: []corpusgen.AgentManagerEventMetadata{
				{RunID: "run-a", EventID: "event-a", Sequence: 0, EventType: "message", Timestamp: at, Role: "user", Source: "operator"},
				{RunID: "run-a", EventID: "event-b", Sequence: 1, EventType: "message", Timestamp: at.Add(time.Second), Role: "assistant", Source: "agent"},
			}},
			{RunID: "run-a", MetadataOnly: true, Events: []corpusgen.AgentManagerEventMetadata{
				{RunID: "run-a", EventID: "event-c", Sequence: 3, EventType: "message", Timestamp: at.Add(3 * time.Second), Role: "user", Source: "operator", RequestID: "request-3", Consumer: "agent-manager", PolicyRole: "chat.default", Model: "model-a", CapacityEvidence: "capacity://claim-3"},
			}},
		},
	}}
	got, err := corpusgen.HarvestAgentManagerCallerSources(context.Background(), reader, []string{"run-a"}, 2)
	if err != nil {
		t.Fatalf("HarvestAgentManagerCallerSources: %v", err)
	}
	if len(got) != 2 || got[0].EventID != "event-a" || got[1].EventID != "event-c" {
		t.Fatalf("caller source refs = %+v", got)
	}
	if got[1].RequestID != "request-3" || got[1].PolicyRole != "chat.default" || got[1].Consumer != "agent-manager" {
		t.Fatalf("route metadata was not retained: %+v", got[1])
	}
	if want := []int64{-1, 1}; len(reader.seen["run-a"]) != len(want) || reader.seen["run-a"][0] != want[0] || reader.seen["run-a"][1] != want[1] {
		t.Fatalf("page cursors = %v, want %v", reader.seen["run-a"], want)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "content") || strings.Contains(string(encoded), "prompt") {
		t.Fatalf("source refs exposed message fields: %s", encoded)
	}
}

func TestAgentManagerMetadataHTTPReaderRequiresMetadataOnlyResponse(t *testing.T) {
	const sentinel = "PRIVATE_MESSAGE_SENTINEL"
	var requestedPath string
	payload := `{"run_id":"run-a","metadata_only":true,"events":[{"run_id":"run-a","event_id":"event-a","sequence":0,"event_type":"message","timestamp":"2026-11-16T04:00:00Z","role":"user","source":"operator"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		if r.URL.Query().Get("after_sequence") != "-1" || r.URL.Query().Get("limit") != "10" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()
	reader := corpusgen.AgentManagerMetadataHTTPReader{BaseURL: server.URL, Client: server.Client()}
	page, err := reader.GetRunEventMetadata(context.Background(), "run-a", -1, 10)
	if err != nil {
		t.Fatalf("GetRunEventMetadata: %v", err)
	}
	if requestedPath != "/api/v1/runs/run-a/events/metadata" {
		t.Fatalf("request path = %q", requestedPath)
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), sentinel) || strings.Contains(string(encoded), "content") {
		t.Fatalf("metadata response retained message content: %s", encoded)
	}
	payload = `{"run_id":"run-a","metadata_only":true,"events":[{"run_id":"run-a","event_id":"event-a","sequence":0,"event_type":"message","timestamp":"2026-11-16T04:00:00Z","role":"user","source":"operator","content":"` + sentinel + `"}]}`
	page, err = reader.GetRunEventMetadata(context.Background(), "run-a", -1, 10)
	if err == nil || len(page.Events) != 0 {
		t.Fatalf("unexpected event-body field was accepted: page=%+v err=%v", page, err)
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("metadata error leaked body sentinel: %v", err)
	}
}

func TestMatchAgentManagerCallerSourceRequiresExactGatewayReceipt(t *testing.T) {
	at := time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)
	source := corpusgen.AgentManagerCallerSource{RunID: "run-a", EventID: "event-a", Sequence: 8, Timestamp: at, MessageRole: "user", MessageSource: "operator"}
	requestID, err := searcheval.AgentManagerReplayRequestID(source.RunID, source.EventID, source.Sequence, "code.default", "agent-manager")
	if err != nil {
		t.Fatal(err)
	}
	route := corpusgen.GatewayRouteProof{
		SourceRunID: source.RunID, SourceEventID: source.EventID, SourceSequence: uint64(source.Sequence),
		GatewayEventID: "gateway-event", GatewayRequestID: requestID, GatewayRoleEvidenceURI: "gateway-ledger://events/gateway-event", TaskRole: "code.default", Consumer: "agent-manager",
		SelectedResourceRole: "chat.default", Status: "succeeded", SelectedProvider: "ollama", SelectedModel: "qwen3.5:9b",
		Locality: "local", CapacityClaimID: "claim-1", CapacityRequiredBytes: 1, CapacityGrantedBytes: 1, CapacityVerdict: "fit", CapacityEstimateEvidenceURI: "capacity-footprint://host/2026-11-16T04:00:00Z",
	}
	if got, err := corpusgen.MatchAgentManagerCallerSource(source, route); err != nil || got != "matched" {
		t.Fatalf("match = %q, %v", got, err)
	}
	source.Model = "different-model"
	if got, err := corpusgen.MatchAgentManagerCallerSource(source, route); err != nil || got != "unmatched_metadata_identity" {
		t.Fatalf("mismatched model metadata = %q, %v", got, err)
	}
	source.Model = ""
	route.GatewayRequestID = "unrelated"
	if got, err := corpusgen.MatchAgentManagerCallerSource(source, route); err != nil || got != "unmatched_request_identity" {
		t.Fatalf("mismatched request = %q, %v", got, err)
	}
	route.GatewayRequestID = requestID
	route.SourceSequence++
	if got, err := corpusgen.MatchAgentManagerCallerSource(source, route); err != nil || got != "unmatched_source_identity" {
		t.Fatalf("mismatched source = %q, %v", got, err)
	}
}

func TestJoinAgentManagerCallerSourcesReportsBothUnmatchedSides(t *testing.T) {
	at := time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)
	sources := []corpusgen.AgentManagerCallerSource{{RunID: "run-a", EventID: "event-a", Sequence: 1, Timestamp: at, MessageRole: "user", MessageSource: "operator"}}
	routes := []corpusgen.GatewayRouteProof{{SourceRunID: "run-b", SourceEventID: "event-b", SourceSequence: 2, GatewayRequestID: "request-b"}}
	got, err := corpusgen.JoinAgentManagerCallerSources(sources, routes)
	if err != nil {
		t.Fatalf("JoinAgentManagerCallerSources: %v", err)
	}
	if len(got) != 2 || got[0].Status != "unmatched_gateway_receipt" || got[1].Status != "unmatched_agent_manager_source" {
		t.Fatalf("unmatched joins = %+v", got)
	}
}

func TestJoinAgentManagerCallerSourcesUsesRequestIdentityWhenSourceHasIt(t *testing.T) {
	at := time.Date(2026, 11, 16, 4, 0, 0, 0, time.UTC)
	requestID, err := searcheval.AgentManagerReplayRequestID("run-a", "event-a", 1, "code.default", "agent-manager")
	if err != nil {
		t.Fatal(err)
	}
	source := corpusgen.AgentManagerCallerSource{RunID: "run-a", EventID: "event-a", Sequence: 1, Timestamp: at, MessageRole: "user", MessageSource: "operator", RequestID: requestID}
	routes := []corpusgen.GatewayRouteProof{
		{SourceRunID: "run-a", SourceEventID: "event-a", SourceSequence: 1, GatewayRequestID: "request-a"},
		{SourceRunID: "run-a", SourceEventID: "event-a", SourceSequence: 1, GatewayRequestID: requestID, TaskRole: "code.default", Consumer: "agent-manager"},
	}
	got, err := corpusgen.JoinAgentManagerCallerSources([]corpusgen.AgentManagerCallerSource{source}, routes)
	if err != nil {
		t.Fatalf("JoinAgentManagerCallerSources: %v", err)
	}
	if len(got) != 2 || !containsJoinStatus(got, "unmatched_request_identity") || !containsJoinStatus(got, "unmatched_route_evidence") {
		t.Fatalf("request-scoped join = %+v", got)
	}
	source.RequestID = ""
	got, err = corpusgen.JoinAgentManagerCallerSources([]corpusgen.AgentManagerCallerSource{source}, routes)
	if err != nil {
		t.Fatalf("ambiguous JoinAgentManagerCallerSources: %v", err)
	}
	if len(got) != 3 || !containsJoinStatus(got, "ambiguous_gateway_receipts") {
		t.Fatalf("ambiguous source join = %+v", got)
	}
}

func containsJoinStatus(joins []corpusgen.AgentManagerCallerSourceJoin, status string) bool {
	for _, join := range joins {
		if join.Status == status {
			return true
		}
	}
	return false
}
