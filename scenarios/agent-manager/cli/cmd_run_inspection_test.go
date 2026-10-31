package main

import (
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestRunReportDisplaysInt64ProtoJSONTokens(t *testing.T) {
	for _, test := range []struct {
		name      string
		tokens    string
		want      string
		wantError string
	}{
		{name: "int64 ProtoJSON", tokens: `"2147495992"`, want: "tokens: 2147495992"},
		{name: "legacy numeric", tokens: `2147495992`, want: "tokens: 2147495992"},
		{name: "overflow rejected", tokens: `"9223372036854775808"`, wantError: "decode run report tokens"},
		{name: "malformed rejected", tokens: `"not-a-number"`, wantError: "decode run report"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
				if request.Path == "/api/v1/runs/run-1/report" {
					return `{"run_id":"run-1","status":"complete","turns":1,"tokens":` + test.tokens + `,"cost_usd":0,"event_counts":{},"diff":{"bytes":"0","available":{"state":"available"}},"receipts_availability":{"state":"unobserved"}}`
				}
				return `{}`
			})
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
				return cliutil.APIBaseOptions{DefaultBase: server.URL()}
			}, nil)
			app := &App{services: NewServices(api)}
			if test.wantError != "" {
				err := app.runReport([]string{"run-1"})
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("expected report parse error containing %q, got %v", test.wantError, err)
				}
				return
			}
			var commandErr error
			output := captureStdout(t, func() error {
				commandErr = app.runReport([]string{"run-1"})
				return commandErr
			})
			if commandErr != nil || !strings.Contains(output, test.want) {
				t.Fatalf("run report output/error = %q / %v; want %q", output, commandErr, test.want)
			}
		})
	}
}

func TestRunInspectionCommandsEmitNextSteps(t *testing.T) {
	run := &domainpb.Run{Id: "run-1", Result: &domainpb.RunResult{
		TerminalReason: "completed",
		Selection:      &domainpb.FinalOutputSelection{Status: domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED, Rule: "terminal", SelectedCandidateId: "candidate-1"},
		Candidates:     []*domainpb.FinalOutputCandidate{{Id: "candidate-1", Sequence: 2, Terminal: true, CompletionReason: "turn_completed", EvidenceTier: 3}},
		Structured:     &domainpb.StructuredResult{Status: domainpb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_INVALID, Method: "deterministic", Diagnostics: []*domainpb.StructuredDiagnostic{{Code: "schema_invalid", Message: "expected object"}}},
	}}
	runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: run})
	if err != nil {
		t.Fatal(err)
	}
	eventsJSON, err := protojson.Marshal(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{
		{Sequence: 1, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "user", Content: "inspect"}}},
		{Sequence: 2, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "done"}}},
		{Sequence: 3, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT, Data: &domainpb.RunEvent_ToolResult{ToolResult: &domainpb.ToolResultEventData{ToolName: "bash", Success: false, Error: "denied"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	statsJSON, err := protojson.Marshal(&apipb.GetRunEventsResponse{Summary: &apipb.RunEventSummary{Counts: map[string]int64{"message": 2, "tool_result": 1}}})
	if err != nil {
		t.Fatal(err)
	}
	diffJSON, err := protojson.Marshal(&apipb.GetRunDiffResponse{Diff: &domainpb.RunDiff{RunId: "run-1", Content: "diff --git a/a b/a\n+change\n"}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		switch request.Path {
		case "/api/v1/runs/run-1/report":
			return `{"run_id":"run-1","status":"completed","turns":1,"tokens":2,"cost_usd":0,"project_owned_tool_calls":1,"external_tool_calls":0,"result":{"selection_status":"selected","selection_rule":"terminal","candidate_count":1},"tools":[{"name":"bash","calls":1,"successes":0,"failures":1}],"event_counts":{"tool_result":1,"error":1,"quota":32},"longest_event_gap_ms":"0","diff":{"bytes":"0","available":{"state":"available"}},"receipts_availability":{"state":"unobserved"}}`
		case "/api/v1/runs/run-1":
			return string(runJSON)
		case "/api/v1/runs/run-1/events":
			if strings.Contains(request.Query, "summary_only=true") {
				return string(statsJSON)
			}
			return string(eventsJSON)
		case "/api/v1/runs/run-1/observed-receipts":
			return `{"status":"unobserved","observations":[]}`
		case "/api/v1/runs/run-1/diff":
			return string(diffJSON)
		default:
			return `{}`
		}
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	app := &App{services: NewServices(api)}
	commands := []struct {
		name string
		run  func() error
	}{
		{"report", func() error { return app.runReport([]string{"run-1"}) }},
		{"result", func() error { return app.runResult([]string{"run-1"}) }},
		{"tools", func() error { return app.runTools([]string{"run-1", "--failed"}) }},
		{"messages", func() error { return app.runMessages([]string{"run-1"}) }},
		{"receipts", func() error { return app.runReceipts([]string{"run-1"}) }},
		{"events", func() error { return app.runEvents([]string{"run-1", "--stats"}) }},
		{"diff", func() error { return app.runDiff([]string{"run-1", "--stat"}) }},
	}
	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			output := captureStdout(t, command.run)
			if !strings.Contains(output, "Next: ") {
				t.Fatalf("%s did not emit progressive-disclosure footer:\n%s", command.name, output)
			}
			if command.name == "report" && !strings.Contains(output, "Diff: files=0 bytes=0 (available)") {
				t.Fatalf("report did not decode proto JSON int64 fields:\n%s", output)
			}
			if command.name == "report" && (!strings.Contains(output, "error=1") || !strings.Contains(output, "quota=32")) {
				t.Fatalf("report did not show error and quota event counts apart:\n%s", output)
			}
		})
	}
}

func TestDefaultMessagesAndToolsRejectMalformedEventResponses(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		if request.Path == "/api/v1/runs/"+runID+"/report" {
			return `{"tools":[]}`
		}
		if request.Path == "/api/v1/runs/"+runID+"/events" {
			return `{"events":[}`
		}
		return `{}`
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	app := &App{services: NewServices(api)}

	for name, call := range map[string]func() error{
		"messages": func() error { return app.runMessages([]string{runID}) },
		"tools":    func() error { return app.runTools([]string{runID}) },
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			if err == nil || !strings.Contains(err.Error(), "decode run events response") {
				t.Fatalf("malformed event response returned error %v, want decode failure", err)
			}
		})
	}
}

func TestRunMessagesRangeUsesBoundedEventSequencePageAndPreservesSelection(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	selectedRun := &domainpb.Run{Id: runID, Result: &domainpb.RunResult{
		Selection:  &domainpb.FinalOutputSelection{SelectedCandidateId: "candidate-final"},
		Candidates: []*domainpb.FinalOutputCandidate{{Id: "candidate-final", Sequence: 13}},
	}}
	runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: selectedRun})
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := protojson.Marshal(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{
		{Sequence: 10, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "user", Content: "ordinary prompt"}}},
		{Sequence: 11, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "needle evidence", EvidenceOnly: true}}},
		{Sequence: 12, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL, Data: &domainpb.RunEvent_ToolCall{ToolCall: &domainpb.ToolCallEventData{ToolName: "bash"}}},
		{Sequence: 13, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "needle selected"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		switch request.Path {
		case "/api/v1/runs/" + runID:
			return string(runJSON)
		case "/api/v1/runs/" + runID + "/events":
			return string(eventJSON)
		default:
			return `{}`
		}
	})
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = server.URL()

	allOutput := captureStdout(t, func() error {
		return app.core.CLI.RunWithWriters([]string{"run", "messages", runID, "--range=10:13", "--all", "--grep=needle"}, &strings.Builder{}, &strings.Builder{})
	})
	if !strings.Contains(allOutput, "Messages: shown=2 total=3") || !strings.Contains(allOutput, "11 assistant: needle evidence") || !strings.Contains(allOutput, "13 assistant [selected final output]: needle selected") {
		t.Fatalf("all/range/grep/selected output mismatch:\n%s", allOutput)
	}
	if strings.Contains(allOutput, "12 ") {
		t.Fatalf("range included an outside event or non-message event:\n%s", allOutput)
	}
	defaultOutput := captureStdout(t, func() error {
		return app.core.CLI.RunWithWriters([]string{"run", "messages", runID, "--range=10:13"}, &strings.Builder{}, &strings.Builder{})
	})
	if !strings.Contains(defaultOutput, "Messages: shown=2 total=3") || strings.Contains(defaultOutput, "needle evidence") || !strings.Contains(defaultOutput, "13 assistant [selected final output]: needle selected") {
		t.Fatalf("default evidence filtering or selection label changed:\n%s", defaultOutput)
	}
	requests := server.Requests()
	eventReads := 0
	for _, request := range requests {
		if request.Path != "/api/v1/runs/"+runID+"/events" {
			continue
		}
		eventReads++
		query, err := url.ParseQuery(request.Query)
		if err != nil || query.Get("after_sequence") != "9" || query.Get("limit") != "4" || query.Get("summary_only") != "" {
			t.Fatalf("range request was not a bounded payload page: %+v (parse error=%v)", request, err)
		}
	}
	if eventReads != 2 {
		t.Fatalf("event page requests=%d, want one bounded page for each command", eventReads)
	}
}

func TestRunMessagesRangePaginatesBySequenceAndStopsAtInclusiveEnd(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: &domainpb.Run{Id: runID}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		if request.Path == "/api/v1/runs/"+runID {
			return string(runJSON)
		}
		if request.Path != "/api/v1/runs/"+runID+"/events" {
			return `{}`
		}
		query, err := url.ParseQuery(request.Query)
		if err != nil {
			return `{}`
		}
		after, _ := strconv.ParseInt(query.Get("after_sequence"), 10, 64)
		limit, _ := strconv.Atoi(query.Get("limit"))
		if limit < 1 || limit > 1000 {
			return `{}`
		}
		page := make([]*domainpb.RunEvent, 0, limit)
		for sequence := after + 1; sequence <= 1006 && len(page) < limit; sequence++ {
			content := "ordinary"
			if sequence >= 1004 {
				content = "needle"
			}
			page = append(page, &domainpb.RunEvent{Sequence: sequence, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: content}}})
		}
		response, marshalErr := protojson.Marshal(&apipb.GetRunEventsResponse{Events: page, HasMore: after < 1006-int64(len(page))})
		if marshalErr != nil {
			return `{}`
		}
		return string(response)
	})
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = server.URL()
	output := captureStdout(t, func() error {
		return app.core.CLI.RunWithWriters([]string{"run", "messages", runID, "--range=5:1005", "--grep=needle"}, &strings.Builder{}, &strings.Builder{})
	})
	if !strings.Contains(output, "Messages: shown=2 total=1001") || !strings.Contains(output, "1004 assistant: needle") || !strings.Contains(output, "1005 assistant: needle") || strings.Contains(output, "1006 assistant") {
		t.Fatalf("paged range omitted rows or exceeded its inclusive end:\n%s", output)
	}
	var eventRequests []clitest.Request
	for _, request := range server.Requests() {
		if request.Path == "/api/v1/runs/"+runID+"/events" {
			eventRequests = append(eventRequests, request)
		}
	}
	if len(eventRequests) != 2 {
		t.Fatalf("event page requests=%d, want 2", len(eventRequests))
	}
	first, _ := url.ParseQuery(eventRequests[0].Query)
	second, _ := url.ParseQuery(eventRequests[1].Query)
	if first.Get("after_sequence") != "4" || first.Get("limit") != "1000" || second.Get("after_sequence") != "1004" || second.Get("limit") != "1" {
		t.Fatalf("unexpected sequence paging: first=%q second=%q", eventRequests[0].Query, eventRequests[1].Query)
	}
}

func TestRunMessagesExactSequenceRangeRequestsOneEvent(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: &domainpb.Run{Id: runID}})
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := protojson.Marshal(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: 207, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "selected body"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		if request.Path == "/api/v1/runs/"+runID {
			return string(runJSON)
		}
		if request.Path == "/api/v1/runs/"+runID+"/events" {
			return string(eventJSON)
		}
		return `{}`
	})
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = server.URL()
	output := captureStdout(t, func() error {
		return app.core.CLI.RunWithWriters([]string{"run", "messages", runID, "--range=207:207"}, &strings.Builder{}, &strings.Builder{})
	})
	if !strings.Contains(output, "207 assistant: selected body") {
		t.Fatalf("exact event was not rendered:\n%s", output)
	}
	for _, request := range server.Requests() {
		if request.Path != "/api/v1/runs/"+runID+"/events" {
			continue
		}
		query, err := url.ParseQuery(request.Query)
		if err != nil || query.Get("after_sequence") != "206" || query.Get("limit") != "1" {
			t.Fatalf("exact sequence request=%+v (parse error=%v); want after=206 limit=1", request, err)
		}
		return
	}
	t.Fatal("no event request was made")
}

func TestRunMessagesRangePageWidthIsOverflowSafeAtSequenceBounds(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	for _, test := range []struct {
		name          string
		rangeArg      string
		sequence      int64
		wantAfter     string
		wantLimit     string
		wantAfterOmit bool
	}{
		{name: "zero through maximum", rangeArg: "0:9223372036854775807", sequence: 0, wantLimit: "1000", wantAfterOmit: true},
		{name: "maximum singleton", rangeArg: "9223372036854775807:9223372036854775807", sequence: 9223372036854775807, wantAfter: "9223372036854775806", wantLimit: "1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: &domainpb.Run{Id: runID}})
			if err != nil {
				t.Fatal(err)
			}
			eventJSON, err := protojson.Marshal(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: test.sequence, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "boundary"}}}}})
			if err != nil {
				t.Fatal(err)
			}
			server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
				if request.Path == "/api/v1/runs/"+runID {
					return string(runJSON)
				}
				if request.Path == "/api/v1/runs/"+runID+"/events" {
					return string(eventJSON)
				}
				return `{}`
			})
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			app.core.APIOverride = server.URL()
			if err := app.core.CLI.RunWithWriters([]string{"run", "messages", runID, "--range=" + test.rangeArg}, &strings.Builder{}, &strings.Builder{}); err != nil {
				t.Fatal(err)
			}
			for _, request := range server.Requests() {
				if request.Path != "/api/v1/runs/"+runID+"/events" {
					continue
				}
				query, err := url.ParseQuery(request.Query)
				if err != nil || query.Get("limit") != test.wantLimit || query.Get("after_sequence") != test.wantAfter {
					t.Fatalf("boundary request=%+v; want limit=%s after=%q (parse error=%v)", request, test.wantLimit, test.wantAfter, err)
				}
				if test.wantAfterOmit && query.Has("after_sequence") {
					t.Fatalf("zero-start range unexpectedly set after_sequence: %q", request.Query)
				}
				return
			}
			t.Fatal("no event page request was made")
		})
	}
}

func TestRunMessagesRangeRejectsDuplicateOrOutOfOrderPage(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: &domainpb.Run{Id: runID}})
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := protojson.Marshal(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{
		{Sequence: 10, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "first"}}},
		{Sequence: 10, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "duplicate"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		if request.Path == "/api/v1/runs/"+runID {
			return string(runJSON)
		}
		if request.Path == "/api/v1/runs/"+runID+"/events" {
			return string(eventJSON)
		}
		return `{}`
	})
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = server.URL()
	err = app.core.CLI.RunWithWriters([]string{"run", "messages", runID, "--range=10:11"}, &strings.Builder{}, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "not strictly ordered") {
		t.Fatalf("duplicate sequence error=%v, want strict-order refusal", err)
	}
}

func TestRunMessagesWithoutRangeKeepsExistingUnboundedRequest(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: &domainpb.Run{Id: runID}})
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := protojson.Marshal(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: 1, EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "complete default history"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		if request.Path == "/api/v1/runs/"+runID {
			return string(runJSON)
		}
		if request.Path == "/api/v1/runs/"+runID+"/events" {
			return string(eventJSON)
		}
		return `{}`
	})
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.APIOverride = server.URL()
	output := captureStdout(t, func() error {
		return app.core.CLI.RunWithWriters([]string{"run", "messages", runID}, &strings.Builder{}, &strings.Builder{})
	})
	if !strings.Contains(output, "complete default history") || !strings.Contains(output, "Messages: shown=1 total=1") {
		t.Fatalf("no-range output changed:\n%s", output)
	}
	for _, request := range server.Requests() {
		if request.Path == "/api/v1/runs/"+runID+"/events" && request.Query != "" {
			t.Fatalf("no-range request unexpectedly gained pagination: %q", request.Query)
		}
	}
}

func TestRunResultSuggestsExactSelectedMessageRangeWhenAvailable(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	for _, test := range []struct {
		name string
		run  *domainpb.Run
		want string
	}{
		{
			name: "selected candidate",
			run: &domainpb.Run{Id: runID, Result: &domainpb.RunResult{
				Selection:  &domainpb.FinalOutputSelection{Status: domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED, SelectedCandidateId: "final"},
				Candidates: []*domainpb.FinalOutputCandidate{{Id: "final", Sequence: 207}},
			}},
			want: "agent-manager run messages " + runID + " --range 207:207",
		},
		{
			name: "selected first event sequence",
			run: &domainpb.Run{Id: runID, Result: &domainpb.RunResult{
				Selection:  &domainpb.FinalOutputSelection{Status: domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED, SelectedCandidateId: "first"},
				Candidates: []*domainpb.FinalOutputCandidate{{Id: "first", Sequence: 0}},
			}},
			want: "agent-manager run messages " + runID + " --range 0:0",
		},
		{
			name: "ambiguous selection",
			run: &domainpb.Run{Id: runID, Result: &domainpb.RunResult{
				Selection:  &domainpb.FinalOutputSelection{Status: domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_AMBIGUOUS},
				Candidates: []*domainpb.FinalOutputCandidate{{Id: "candidate-a", Sequence: 207}, {Id: "candidate-b", Sequence: 211}},
			}},
			want: "agent-manager run messages " + runID,
		},
		{
			name: "candidate id without selected status",
			run: &domainpb.Run{Id: runID, Result: &domainpb.RunResult{
				Selection:  &domainpb.FinalOutputSelection{SelectedCandidateId: "final"},
				Candidates: []*domainpb.FinalOutputCandidate{{Id: "final", Sequence: 207}},
			}},
			want: "agent-manager run messages " + runID,
		},
		{
			name: "selected candidate without usable sequence",
			run: &domainpb.Run{Id: runID, Result: &domainpb.RunResult{
				Selection:  &domainpb.FinalOutputSelection{Status: domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED, SelectedCandidateId: "final"},
				Candidates: []*domainpb.FinalOutputCandidate{{Id: "final", Sequence: -1}},
			}},
			want: "agent-manager run messages " + runID,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, err := protojson.Marshal(&apipb.GetRunResponse{Run: test.run})
			if err != nil {
				t.Fatal(err)
			}
			server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
				if request.Path == "/api/v1/runs/"+runID {
					return string(response)
				}
				return `{}`
			})
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			app.core.APIOverride = server.URL()
			output := captureStdout(t, func() error {
				return app.core.CLI.RunWithWriters([]string{"run", "result", runID}, &strings.Builder{}, &strings.Builder{})
			})
			if !strings.Contains(output, "Next: "+test.want) || !strings.Contains(output, "agent-manager run report "+runID) {
				t.Fatalf("result footer = %q; want message route %q and report route", output, test.want)
			}
		})
	}
}

func TestRunResultLabelsStoredResultWhenCurrentExecutionHasNoTerminalResult(t *testing.T) {
	const runID = "33333333-3333-4333-8333-333333333333"
	ended := timestamppb.New(time.Date(2026, 10, 20, 14, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		name           string
		status         domainpb.RunStatus
		statusText     string
		endedAt        *timestamppb.Timestamp
		wantHistorical bool
	}{
		{name: "active run with prior failure", status: domainpb.RunStatus_RUN_STATUS_RUNNING, statusText: "running", wantHistorical: true},
		{name: "parked run with prior failure", status: domainpb.RunStatus_RUN_STATUS_PARKED, statusText: "parked", wantHistorical: true},
		{name: "terminal failure missing ended time", status: domainpb.RunStatus_RUN_STATUS_FAILED},
		{name: "terminal complete result", status: domainpb.RunStatus_RUN_STATUS_COMPLETE, endedAt: ended},
		{name: "needs review with completed execution", status: domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW, endedAt: ended},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runJSON, err := protojson.Marshal(&apipb.GetRunResponse{Run: &domainpb.Run{
				Id: runID, Status: tc.status, EndedAt: tc.endedAt,
				Result: &domainpb.RunResult{
					TerminalReason: "failed",
					Selection:      &domainpb.FinalOutputSelection{Status: domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNAVAILABLE, Rule: "no_viable_assistant_candidate"},
				},
			}})
			if err != nil {
				t.Fatal(err)
			}
			server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
				if request.Path == "/api/v1/runs/"+runID {
					return string(runJSON)
				}
				return `{}`
			})
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			app.core.APIOverride = server.URL()
			output := captureStdout(t, func() error {
				return app.core.CLI.RunWithWriters([]string{"run", "result", runID}, &strings.Builder{}, &strings.Builder{})
			})
			if !strings.Contains(output, "Terminal reason: failed") {
				t.Fatalf("retained result was hidden: %q", output)
			}
			gotHistorical := strings.Contains(output, "current execution has no terminal result yet") && strings.Contains(output, "retained from an earlier invocation")
			if tc.wantHistorical && !strings.Contains(output, "Current run status: "+tc.statusText) {
				t.Fatalf("current execution status not shown: %q", output)
			}
			if gotHistorical != tc.wantHistorical {
				t.Fatalf("historical qualifier=%t, want %t; output=%q", gotHistorical, tc.wantHistorical, output)
			}
		})
	}
}

func captureStdout(t *testing.T, run func() error) string {
	t.Helper()
	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = previous }()
	done := make(chan []byte, 1)
	go func() { data, _ := io.ReadAll(reader); done <- data }()
	if err := run(); err != nil {
		_ = writer.Close()
		<-done
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return string(<-done)
}

func TestNativeToolConsumersKeepUnknownOutOfFailureFilters(t *testing.T) {
	unknown := false
	zero := int32(0)
	failure := int32(9)
	rows := []*domainpb.RunEvent{}
	for index, d := range []*domainpb.ToolResultEventData{
		{ToolName: "unknown-entry", OutcomeKnown: &unknown},
		{ToolName: "zero-entry", ExitCode: &zero},
		{ToolName: "failure-entry", ExitCode: &failure, Success: true},
		{ToolName: "legacy-success-entry", Success: true},
		{ToolName: "legacy-failure-entry", Success: false, Error: "failed"},
	} {
		rows = append(rows, &domainpb.RunEvent{Sequence: int64(index + 1), EventType: domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT, Data: &domainpb.RunEvent_ToolResult{ToolResult: d}})
	}
	events, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&apipb.GetRunEventsResponse{Events: rows})
	if err != nil {
		t.Fatal(err)
	}
	report, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&apipb.RunReport{RunId: "run-1", Tools: []*apipb.RunReportTool{{Name: "exec", Calls: 5, Returned: 5, Successes: 2, Failures: 2, UnknownOutcome: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(req clitest.Request) string {
		if strings.HasSuffix(req.Path, "/report") {
			return string(report)
		}
		return string(events)
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL()} }, nil)
	a := &App{services: NewServices(api)}
	for _, command := range []struct {
		name string
		run  func() error
	}{
		{"report", func() error { return a.runReport([]string{"run-1"}) }}, {"tools", func() error { return a.runTools([]string{"run-1"}) }},
	} {
		t.Run(command.name, func(t *testing.T) {
			output := captureStdout(t, command.run)
			if !strings.Contains(output, "calls=5 returned=5 success=2 failed=2 unknown=1 unresolved=0") {
				t.Fatalf("summary=%s", output)
			}
			if command.name == "tools" && !strings.Contains(output, "unknown-entry outcome=unknown") {
				t.Fatalf("unknown display=%s", output)
			}
		})
	}
	for _, command := range []struct {
		name string
		run  func() error
	}{
		{"tools failed", func() error { return a.runTools([]string{"run-1", "--failed"}) }}, {"events failed", func() error { return a.runEvents([]string{"run-1", "--failed"}) }},
	} {
		t.Run(command.name, func(t *testing.T) {
			output := captureStdout(t, command.run)
			for _, absent := range []string{"unknown-entry", "zero-entry", "legacy-success-entry"} {
				if strings.Contains(output, absent) {
					t.Fatalf("nonfailure %q in failed output: %s", absent, output)
				}
			}
			for _, present := range []string{"failure-entry", "legacy-failure-entry"} {
				if !strings.Contains(output, present) {
					t.Fatalf("failure %q absent: %s", present, output)
				}
			}
		})
	}
}
