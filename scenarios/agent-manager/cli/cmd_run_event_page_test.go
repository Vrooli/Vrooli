package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

type runEventPageHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	mu       sync.Mutex
	requests []*apipb.GetRunEventsRequest
}

func (h *runEventPageHandler) GetRunEvents(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests = append(h.requests, req.Msg)
	return connect.NewResponse(&apipb.GetRunEventsResponse{
		Events:  []*domainpb.RunEvent{{Sequence: 8}, {Sequence: 9}},
		HasMore: true,
	}), nil
}

func TestRunEventPageUsesTypedConnectBindingAndRendersHasMore(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	owner := &runEventPageHandler{}
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(owner)
	mux := http.NewServeMux()
	mux.Handle(path, connectHandler)
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = server.URL
	var jsonOutput strings.Builder
	if err := app.core.CLI.RunWithWriters([]string{"run", "event-page", runID, "--limit=2", "--after-sequence=7", "--json"}, &jsonOutput, &strings.Builder{}); err != nil {
		t.Fatalf("typed JSON command: %v", err)
	}
	var decoded apipb.GetRunEventsResponse
	if err := protojson.Unmarshal([]byte(jsonOutput.String()), &decoded); err != nil {
		t.Fatalf("decode typed response: %v; output=%s", err, jsonOutput.String())
	}
	if len(decoded.GetEvents()) != 2 || !decoded.GetHasMore() {
		t.Fatalf("JSON response lost page fields: %s", jsonOutput.String())
	}

	var humanOutput strings.Builder
	if err := app.core.CLI.RunWithWriters([]string{"run", "event-page", runID, "--limit=2", "--after-sequence=7"}, &humanOutput, &strings.Builder{}); err != nil {
		t.Fatalf("typed human command: %v", err)
	}
	if !strings.Contains(humanOutput.String(), "has_more=true") {
		t.Fatalf("human report omitted has_more: %s", humanOutput.String())
	}
	if len(owner.requests) != 2 {
		t.Fatalf("GetRunEvents calls=%d, want 2", len(owner.requests))
	}
	for i, req := range owner.requests {
		if req.GetRunId() != runID || req.GetLimit() != 2 || req.GetAfterSequence() != 7 {
			t.Fatalf("request[%d]=%v, want run=%s limit=2 after_sequence=7", i, req, runID)
		}
	}
}
