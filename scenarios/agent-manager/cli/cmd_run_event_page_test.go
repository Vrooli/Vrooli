package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	clitest "agent-manager/cli/internal/testutil"
	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

type runEventPageHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	mu            sync.Mutex
	requests      []*apipb.GetRunEventsRequest
	ignoreSummary bool
}

func (h *runEventPageHandler) GetRunEvents(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests = append(h.requests, req.Msg)
	if req.Msg.GetSummaryOnly() && !h.ignoreSummary {
		return connect.NewResponse(&apipb.GetRunEventsResponse{
			Summary: &apipb.RunEventSummary{Counts: map[string]int64{"message": 2}, LatestSequence: ptr(int64(9))},
			HasMore: true,
		}), nil
	}
	return connect.NewResponse(&apipb.GetRunEventsResponse{
		Events:  []*domainpb.RunEvent{{Sequence: 8}, {Sequence: 9}},
		HasMore: true,
	}), nil
}

func TestRunEventPageSummaryOnlyIsPayloadFreeAndFailsClosed(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	for _, ignore := range []bool{false, true} {
		owner := &runEventPageHandler{ignoreSummary: ignore}
		path, handler := apiconnect.NewAgentManagerServiceHandler(owner)
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		server := httptest.NewServer(mux)
		app, err := NewApp()
		if err != nil {
			t.Fatal(err)
		}
		app.core.APIOverride = server.URL
		var out strings.Builder
		err = app.core.CLI.RunWithWriters([]string{"run", "event-page", runID, "--limit=2", "--after-sequence=7", "--summary-only", "--json"}, &out, &strings.Builder{})
		server.Close()
		if len(owner.requests) != 1 || !owner.requests[0].GetSummaryOnly() || owner.requests[0].GetAfterSequence() != 7 {
			t.Fatalf("summary request not forwarded: %+v", owner.requests)
		}
		if ignore {
			if err == nil || strings.Contains(out.String(), "events") {
				t.Fatalf("legacy response was not refused: output=%s error=%v", out.String(), err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		var response apipb.GetRunEventsResponse
		if err := protojson.Unmarshal([]byte(out.String()), &response); err != nil {
			t.Fatal(err)
		}
		if len(response.GetEvents()) != 0 || response.GetSummary().GetLatestSequence() != 9 || !response.GetHasMore() {
			t.Fatalf("incorrect bounded summary: %s", out.String())
		}
	}
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

func TestRunEventsAcceptsValidEmptyPageAndPreservesQuery(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	server := clitest.NewRecordingServer(t, `{}`)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	body, events, err := NewServices(api).Runs.GetEvents(runID, 7, ptr(int64(0)))
	if err != nil || len(events) != 0 || string(body) != `{}` {
		t.Fatalf("valid empty event page: body=%q events=%v err=%v", body, events, err)
	}
	requests := server.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests=%d, want one", len(requests))
	}
	query, err := url.ParseQuery(requests[0].Query)
	if err != nil || query.Get("limit") != "7" || query.Get("after_sequence") != "0" || query.Get("summary_only") != "" {
		t.Fatalf("ordinary event query changed: %+v (parse error=%v)", requests[0], err)
	}
}

func TestRunEventsStatsUsesSummaryForHumanAndJSON(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	server := clitest.NewRecordingServer(t, `{"summary":{"counts":{"future.kind":"1","message":"2"},"latest_sequence":"0"},"has_more":true,"unexpected_payload":"must-not-be-rendered"}`)
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = server.URL()
	jsonOutput := captureStdout(t, func() error {
		return app.core.CLI.RunWithWriters([]string{"run", "events", runID, "--stats", "--json", "--limit=2", "--after-sequence=0"}, &strings.Builder{}, &strings.Builder{})
	})
	var decoded apipb.GetRunEventsResponse
	if err := protojson.Unmarshal([]byte(jsonOutput), &decoded); err != nil {
		t.Fatalf("decode JSON stats response: %v; output=%s", err, jsonOutput)
	}
	if decoded.GetSummary().GetCounts()["message"] != 2 || decoded.GetSummary().GetCounts()["future.kind"] != 1 || !decoded.GetHasMore() || decoded.GetSummary().LatestSequence == nil || decoded.GetSummary().GetLatestSequence() != 0 {
		t.Fatalf("JSON stats mismatch: %s", jsonOutput)
	}
	if len(decoded.GetEvents()) != 0 {
		t.Fatalf("summary hydrated event payloads: %s", jsonOutput)
	}
	if strings.Contains(jsonOutput, "must-not-be-rendered") {
		t.Fatalf("JSON stats forwarded an unknown response field: %s", jsonOutput)
	}
	humanOutput := captureStdout(t, func() error {
		return app.core.CLI.RunWithWriters([]string{"run", "events", runID, "--stats", "--limit=2", "--after-sequence=0"}, &strings.Builder{}, &strings.Builder{})
	})
	if !strings.Contains(humanOutput, "future.kind=1 message=2") {
		t.Fatalf("human stats do not render raw summary types: %s", humanOutput)
	}
	requests := make([]clitest.Request, 0, 2)
	for _, request := range server.Requests() {
		if request.Path == "/api/v1/runs/"+runID+"/events" {
			requests = append(requests, request)
		}
	}
	if len(requests) != 2 {
		t.Fatalf("request count=%d, want human and JSON summary requests: %+v", len(requests), requests)
	}
	for _, request := range requests {
		query, err := url.ParseQuery(request.Query)
		if err != nil || request.Path != "/api/v1/runs/"+runID+"/events" || query.Get("summary_only") != "true" || query.Get("limit") != "2" || query.Get("after_sequence") != "0" {
			t.Fatalf("stats did not request the bounded summary projection: %+v err=%v", request, err)
		}
	}
}

func TestRunEventsStatsFailsClosedWhenAPIReturnsEventPage(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	legacyPage := `{"events":[{"sequence":"1","event_type":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"assistant","content":"payload-must-not-be-rendered"}}]}`
	for _, flags := range [][]string{{"--stats"}, {"--stats", "--json"}} {
		t.Run(strings.Join(flags, "-"), func(t *testing.T) {
			server := clitest.NewRecordingServer(t, legacyPage)
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			app.core.APIOverride = server.URL()
			args := append([]string{"run", "events", runID}, flags...)
			output, callErr := captureRunEventsOutput(t, func() error {
				return app.core.CLI.RunWithWriters(args, &strings.Builder{}, &strings.Builder{})
			})
			if callErr == nil || !strings.Contains(callErr.Error(), "did not return the requested payload-free event summary") {
				t.Fatalf("legacy event-page response error=%v", callErr)
			}
			if output != "" || strings.Contains(output, "payload-must-not-be-rendered") {
				t.Fatalf("stats rendered unverified response bytes: %s", output)
			}
			for _, request := range server.Requests() {
				if request.Path == "/api/v1/runs/"+runID+"/events" && !strings.Contains(request.Query, "summary_only=true") {
					t.Fatalf("stats omitted summary projection: %+v", request)
				}
			}
		})
	}
}

func captureRunEventsOutput(t *testing.T, call func() error) (string, error) {
	t.Helper()
	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	callErr := call()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = previous
	output, err := io.ReadAll(reader)
	if closeErr := reader.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(output), callErr
}
