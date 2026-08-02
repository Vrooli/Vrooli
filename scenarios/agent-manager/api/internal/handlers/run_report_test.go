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
		Events: map[string]int{"heartbeat.miss": 2}, ReceiptCount: 1,
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
	if decoded["run_id"] != id.String() || decoded["duration_ms"] != "1500" || decoded["receipt_count"] != float64(1) {
		t.Fatalf("canonical summary fields = %#v", decoded)
	}
}

func TestWriteRunReportJSONEmitsMeasuredZeroMetrics(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeRunReportJSON(recorder, http.StatusOK, &apipb.RunReport{RunId: "run-1", Status: "complete"})
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
}
