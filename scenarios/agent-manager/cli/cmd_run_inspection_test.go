package main

import (
	"io"
	"os"
	"strings"
	"testing"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

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
	diffJSON, err := protojson.Marshal(&apipb.GetRunDiffResponse{Diff: &domainpb.RunDiff{RunId: "run-1", Content: "diff --git a/a b/a\n+change\n"}})
	if err != nil {
		t.Fatal(err)
	}
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		switch request.Path {
		case "/api/v1/runs/run-1/report":
			return `{"run_id":"run-1","status":"completed","turns":1,"tokens":2,"cost_usd":0,"project_owned_tool_calls":1,"external_tool_calls":0,"result":{"selection_status":"selected","selection_rule":"terminal","candidate_count":1},"tools":[{"name":"bash","calls":1,"successes":0,"failures":1}],"event_counts":{"tool_result":1},"longest_event_gap_ms":"0","diff":{"bytes":"0","available":{"state":"available"}},"receipts_availability":{"state":"unobserved"}}`
		case "/api/v1/runs/run-1":
			return string(runJSON)
		case "/api/v1/runs/run-1/events":
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
