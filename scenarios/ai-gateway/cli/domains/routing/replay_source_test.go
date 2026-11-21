package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
)

func TestAgentManagerSourceVerifierChecksExactUserEvent(t *testing.T) {
	const runID = "11111111-1111-4111-8111-111111111111"
	const eventID = "22222222-2222-4222-8222-222222222222"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runs/"+runID+"/events" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("after_sequence") != "2" || r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("event_types") != "message" {
			t.Errorf("query = %v", r.URL.Query())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{map[string]any{
			"id": eventID, "run_id": runID, "sequence": "3", "event_type": "RUN_EVENT_TYPE_MESSAGE",
			"message": map[string]any{"role": "user", "source": "operator", "content": "exact prompt", "conversation_id": "conv-1"},
		}}})
	}))
	defer server.Close()
	verifier, err := newAgentManagerSourceVerifier(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	caseData := ReplayCase{SourceRunID: runID, SourceEventID: eventID, SourceSequence: 3, ConversationID: "conv-1", InputText: "exact prompt"}
	if err := verifier.verify(context.Background(), caseData); err != nil {
		t.Fatalf("verify event: %v", err)
	}
	caseData.InputText = "altered prompt"
	if err := verifier.verify(context.Background(), caseData); err == nil || !strings.Contains(err.Error(), "exact Agent Manager user message") {
		t.Fatalf("modified prompt error = %v", err)
	}
}

func TestAgentManagerSourceVerifierAcceptsSequenceZeroProtoJSONOmission(t *testing.T) {
	const runID = "11111111-1111-4111-8111-111111111111"
	const eventID = "22222222-2222-4222-8222-222222222222"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after_sequence") != "-1" {
			t.Errorf("after_sequence = %q; want -1", r.URL.Query().Get("after_sequence"))
		}
		// The owner serializer omits sequence because it is a proto3 int64
		// whose value is zero and EmitUnpopulated is disabled.
		_, _ = w.Write([]byte(`{"events":[{"id":"` + eventID + `","run_id":"` + runID + `","event_type":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"user","source":"operator","content":"real zero-sequence prompt","conversation_id":"conversation-0"}}]}`))
	}))
	defer server.Close()
	verifier, err := newAgentManagerSourceVerifier(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	caseData := ReplayCase{SourceRunID: runID, SourceEventID: eventID, SourceSequence: 0, ConversationID: "conversation-0", InputText: "real zero-sequence prompt"}
	if err := verifier.verify(context.Background(), caseData); err != nil {
		t.Fatalf("verify zero-sequence Agent Manager event: %v", err)
	}
	caseData.SourceEventID = "33333333-3333-4333-8333-333333333333"
	if err := verifier.verify(context.Background(), caseData); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("wrong event ID result = %v", err)
	}
}

func TestLocalAgentManagerURLRejectsRemoteHosts(t *testing.T) {
	for _, raw := range []string{"https://agent-manager.example", "file:///tmp/agent-manager"} {
		if _, err := localAgentManagerURL(raw); err == nil {
			t.Errorf("localAgentManagerURL(%q) succeeded", raw)
		}
	}
	if _, err := newAgentManagerSourceVerifier(connect.HTTPClient(nil), "http://localhost:8090"); err == nil {
		t.Fatal("nil authenticated HTTP client should be rejected")
	}
}
