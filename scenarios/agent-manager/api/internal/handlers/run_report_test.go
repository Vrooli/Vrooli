package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/protoconv"
	"agent-manager/internal/researchbudget"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func TestRunReportProtoProjectionUsesCanonicalProtoFields(t *testing.T) {
	id := uuid.New()
	exitCode := 3
	message := runReportToProto(&runreport.RunReport{
		RunID: id, Status: "failed", ExitCode: &exitCode, Duration: 1500 * time.Millisecond,
		Events: map[string]int{"heartbeat.miss": 2, "error": 3, "quota": 4}, ReceiptCount: 1,
		EventsAvailability:   runreport.Availability{State: runreport.AvailabilityAvailable},
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
	counts := decoded["event_counts"].(map[string]any)
	if counts["error"] != float64(3) || counts["quota"] != float64(4) {
		t.Fatalf("error and quota counts = %#v", decoded)
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

func TestWriteRunReportWithResearchBudgetIncludesRunBoundSpendAndBenefit(t *testing.T) {
	id := uuid.New()
	report := &runreport.RunReport{RunID: id, Status: "complete", ResearchBudget: runreport.ResearchBudgetSummary{
		Availability:      runreport.Availability{State: runreport.AvailabilityAvailable},
		SpendAvailability: runreport.Availability{State: runreport.AvailabilityAvailable},
		Summary:           &researchbudget.Summary{RunID: id.String(), TreasuryBudgetID: "budget-1", MandateID: "mandate-1", Currency: "USDC", UnitCode: "USDC_ATOMS", CapMinor: 500, SpendMinor: 125, RemainingMinor: 375, SpendCount: 1, Spend: []researchbudget.Spend{{ID: "spend-1", RunID: id.String(), SettlementID: "settlement-1", TreasuryReceiptID: "receipt-1", IntegrationID: "x402:local", Category: "x402_call", AmountMinor: 125, Currency: "USDC", UnitCode: "USDC_ATOMS", ResultRef: "checkpoint-1", BenefitState: "measured", BenefitSummary: "3 fixture records accepted", SettledAt: time.Date(2026, 11, 13, 1, 0, 0, 0, time.UTC), FreeAlternative: &researchbudget.FreeAlternative{RunID: id.String(), SettlementID: "settlement-1", SourceID: "offer-desk:free-sample", Label: "no-signup sample", ResultRef: "free-result", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ResponseBytes: 81, Limitation: "format only", ObservedAt: time.Date(2026, 11, 13, 1, 0, 1, 0, time.UTC)}}}},
	}}
	recorder := httptest.NewRecorder()
	writeRunReportWithResearchBudgetJSON(recorder, report)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	budget, ok := decoded["research_budget"].(map[string]any)
	if !ok || budget["availability"].(map[string]any)["state"] != "available" || budget["spend_availability"].(map[string]any)["state"] != "available" {
		t.Fatalf("research_budget missing availability: %#v", decoded)
	}
	summary := budget["summary"].(map[string]any)
	spend := summary["spend"].([]any)[0].(map[string]any)
	free := spend["free_alternative"].(map[string]any)
	if summary["run_id"] != id.String() || summary["spend_minor"] != float64(125) || spend["treasury_receipt_id"] != "receipt-1" || spend["benefit_summary"] != "3 fixture records accepted" || free["source_id"] != "offer-desk:free-sample" || free["sha256"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || free["evidence_basis"] != "agent-observed; not independently verified" || !strings.Contains(budget["provenance"].(string), "Missing rows do not establish zero spend") {
		t.Fatalf("run-bound spend/benefit evidence lost: %#v", budget)
	}
}
