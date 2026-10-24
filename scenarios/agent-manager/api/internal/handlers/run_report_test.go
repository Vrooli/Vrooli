package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/protoconv"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func TestRunReportProtoProjectionUsesCanonicalProtoFields(t *testing.T) {
	id := uuid.New()
	exitCode := 3
	message := runReportToProto(&runreport.RunReport{
		RunID: id, Status: "failed", ExitCode: &exitCode, Duration: 1500 * time.Millisecond,
		Events: map[string]int{"heartbeat.miss": 2, "error": 3}, ReceiptCount: 1,
		EventsAvailability:   runreport.Availability{State: runreport.AvailabilityAvailable},
		ExecutionErrorEvents: 1, RateLimitObservations: 1, UnclassifiedErrorEvents: 1,
		Tools:                []runreport.ToolSummary{{Name: "exec", Calls: 2, Returned: 1, UnknownOutcome: 1, Unresolved: 1}},
		ReceiptsAvailability: runreport.Availability{State: "available"},
	})
	body, err := protoconv.MarshalJSON(message)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode report JSON: %v", err)
	}
	tools := decoded["tools"].([]any)
	tool := tools[0].(map[string]any)
	if tool["returned"] != float64(1) || tool["unknown_outcome"] != float64(1) || tool["unresolved"] != float64(1) {
		t.Fatalf("tool wire=%+v", tool)
	}
	if decoded["run_id"] != id.String() || decoded["duration_ms"] != "1500" || decoded["receipt_count"] != float64(1) {
		t.Fatalf("canonical summary fields = %#v", decoded)
	}
	breakdown, ok := decoded["error_breakdown"].(map[string]any)
	if decoded["event_counts"].(map[string]any)["error"] != float64(3) || !ok || breakdown["execution_errors"] != float64(1) || breakdown["rate_limit_observations"] != float64(1) || breakdown["unclassified"] != float64(1) {
		t.Fatalf("raw and typed error counts = %#v", decoded)
	}
}

func TestRunReportPreservesTokensBeyondInt32InProtoJSON(t *testing.T) {
	const tokens int64 = 2_147_495_992
	message := runReportToProto(&runreport.RunReport{RunID: uuid.New(), Status: "complete", Tokens: tokens})
	if message.GetTokens() != tokens {
		t.Fatalf("RunReport tokens = %d, want %d", message.GetTokens(), tokens)
	}
	body, err := protoconv.MarshalJSON(message)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"tokens":"2147495992"`) {
		t.Fatalf("RunReport ProtoJSON did not preserve int64 tokens: %s", body)
	}
	var decoded apipb.RunReport
	if err := protoconv.UnmarshalJSON(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetTokens() != tokens {
		t.Fatalf("ProtoJSON round-trip tokens = %d, want %d", decoded.GetTokens(), tokens)
	}
}

func TestRunReportProtoProjectionOmitsErrorBreakdownWhenEventsAreUnavailable(t *testing.T) {
	message := runReportToProto(&runreport.RunReport{
		RunID: uuid.New(), Status: "failed", EventsAvailability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: "event store unavailable"},
	})
	if message.ErrorBreakdown != nil {
		t.Fatalf("unavailable event stream was projected as measured zero: %+v", message.ErrorBreakdown)
	}
}

func TestRunReportProtoProjectionPreservesMeasuredEmptyEventStream(t *testing.T) {
	message := runReportToProto(&runreport.RunReport{
		RunID: uuid.New(), Status: "complete", Events: map[string]int{},
		EventsAvailability: runreport.Availability{State: runreport.AvailabilityAvailable},
	})
	if message.ErrorBreakdown == nil || message.ErrorBreakdown.ExecutionErrors != 0 || message.ErrorBreakdown.RateLimitObservations != 0 || message.ErrorBreakdown.Unclassified != 0 {
		t.Fatalf("measured empty event stream breakdown = %+v", message.ErrorBreakdown)
	}
}

func TestWriteRunReportJSONEmitsMeasuredZeroMetrics(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeRunReportJSON(recorder, http.StatusOK, &apipb.RunReport{RunId: "run-1", Status: "complete", ErrorBreakdown: &apipb.RunReportErrorBreakdown{}})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	for _, field := range []string{
		`"external_tool_calls":0`, `"project_owned_tool_calls":0`,
		`"receipt_count":0`, `"repeated_tool_calls":0`, `"files_read_more_than_once":0`,
	} {
		if !strings.Contains(recorder.Body.String(), field) {
			t.Fatalf("report JSON missing zero metric %s: %s", field, recorder.Body.String())
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode report JSON: %v", err)
	}
	breakdown, ok := decoded["error_breakdown"].(map[string]any)
	if !ok || breakdown["execution_errors"] != float64(0) || breakdown["rate_limit_observations"] != float64(0) || breakdown["unclassified"] != float64(0) {
		t.Fatalf("measured zero error breakdown missing: %#v", decoded["error_breakdown"])
	}
}
