package handlers

import (
	"encoding/json"
	"net/http"

	"agent-manager/internal/runreport"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

// GetRunReport returns the shared bounded investigation projection. It is JSON
// by design so CLI and UI consumers receive exactly the same discriminators.
func (h *Handler) GetRunReport(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	report, err := h.svc.BuildRunReport(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeRunReportWithResearchBudgetJSON(w, report)
}

// writeRunReportWithResearchBudgetJSON preserves the generated RunReport
// contract while adding the run-bound research ledger as an additive HTTP
// JSON field. Connect remains governed by the shared protobuf schema.
func writeRunReportWithResearchBudgetJSON(w http.ResponseWriter, report *runreport.RunReport) {
	protoReport := runReportToProto(report)
	data, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(protoReport)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to serialize run report")
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to serialize run report")
		return
	}
	budget, err := json.Marshal(researchBudgetJSON(report.ResearchBudget))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to serialize research budget summary")
		return
	}
	fields["research_budget"] = budget
	data, err = json.Marshal(fields)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to serialize run report")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func researchBudgetJSON(value runreport.ResearchBudgetSummary) map[string]any {
	var summary any
	if s := value.Summary; s != nil {
		spend := make([]map[string]any, 0, len(s.Spend))
		for _, item := range s.Spend {
			row := map[string]any{
				"id": item.ID, "run_id": item.RunID, "settlement_id": item.SettlementID,
				"treasury_receipt_id": item.TreasuryReceiptID, "integration_id": item.IntegrationID,
				"category": item.Category, "amount_minor": item.AmountMinor, "currency": item.Currency,
				"unit_code": item.UnitCode, "result_ref": item.ResultRef, "benefit_state": item.BenefitState,
				"benefit_summary": item.BenefitSummary, "settled_at": item.SettledAt,
			}
			if item.FreeAlternative != nil {
				row["free_alternative"] = map[string]any{
					"source_id": item.FreeAlternative.SourceID, "label": item.FreeAlternative.Label,
					"result_ref": item.FreeAlternative.ResultRef, "sha256": item.FreeAlternative.SHA256,
					"response_bytes": item.FreeAlternative.ResponseBytes, "limitation": item.FreeAlternative.Limitation,
					"observed_at": item.FreeAlternative.ObservedAt, "evidence_basis": "agent-observed; not independently verified",
				}
			}
			spend = append(spend, row)
		}
		summary = map[string]any{
			"run_id": s.RunID, "treasury_budget_id": s.TreasuryBudgetID, "mandate_id": s.MandateID,
			"currency": s.Currency, "unit_code": s.UnitCode, "cap_minor": s.CapMinor,
			"spend_minor": s.SpendMinor, "remaining_minor": s.RemainingMinor,
			"spend_count": s.SpendCount, "spend": spend,
		}
	}
	return map[string]any{
		"availability":       map[string]string{"state": string(value.Availability.State), "reason": value.Availability.Reason},
		"spend_availability": map[string]string{"state": string(value.SpendAvailability.State), "reason": value.SpendAvailability.Reason},
		"provenance":         "Budget data comes from the run-bound research ledger; spend rows are recorded Treasury settlement evidence. Free-alternative rows are immutable agent-observed comparator evidence linked to one settlement, not independently verified provider claims. Missing rows do not establish zero spend or prove no free alternative was available.",
		"summary":            summary,
	}
}

// writeRunReportJSON deliberately emits zero-valued numeric metrics. A run
// report is a stable analytical row: consumers must distinguish a measured
// zero (for example, zero external tools) from an omitted/unknown metric.
func writeRunReportJSON(w http.ResponseWriter, status int, report *apipb.RunReport) {
	data, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(report)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to serialize run report")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func runReportToProto(report *runreport.RunReport) *apipb.RunReport {
	if report == nil {
		return &apipb.RunReport{}
	}
	out := &apipb.RunReport{
		RunId: report.RunID.String(), Status: string(report.Status), Error: report.Error,
		DurationMs: report.Duration.Milliseconds(), HeartbeatGapMs: report.HeartbeatGap.Milliseconds(),
		Turns: int32(report.Turns), Tokens: report.Tokens, CostUsd: report.CostUSD,
		Result:      &apipb.RunReportResult{SelectionStatus: string(report.Result.SelectionStatus), SelectionRule: report.Result.SelectionRule, CandidateCount: int32(report.Result.CandidateCount), StructuredStatus: string(report.Result.StructuredStatus), StructuredMethod: report.Result.StructuredMethod, DiagnosticCodes: report.Result.DiagnosticCodes},
		EventCounts: map[string]int32{}, ProjectOwnedToolCalls: int32(report.ProjectOwnedToolCalls), ExternalToolCalls: int32(report.ExternalToolCalls),
		RequestedModel: report.RequestedModel, ActualModel: report.ActualModel, FallbackCount: int32(report.FallbackCount),
		Diff:               &apipb.RunReportDiff{Files: int32(report.Diff.Files), Bytes: report.Diff.Bytes, Available: availabilityToProto(report.Diff.Available)},
		EventsAvailability: availabilityToProto(report.EventsAvailability), ReceiptsAvailability: availabilityToProto(report.ReceiptsAvailability), ReceiptCount: int32(report.ReceiptCount),
		RepeatedToolCalls: int32(report.RepeatedToolCalls), LongestEventGapMs: report.LongestEventGap.Milliseconds(),
		FilesReadMoreThanOnce: int32(report.FilesReadMoreThanOnce),
		WorkReferences:        report.WorkReferences,
		TimeAccounting:        &apipb.RunTimeAccounting{ModelGeneratingMs: report.TimeAccounting.ModelGeneratingMS, ToolExecutingMs: report.TimeAccounting.ToolExecutingMS, IdleWaitingMs: report.TimeAccounting.IdleWaitingMS, AwaitingHumanMs: report.TimeAccounting.AwaitingHumanMS, UnattributableMs: report.TimeAccounting.UnattributableMS, ModelTokens: report.TimeAccounting.ModelTokens, ToolTokens: report.TimeAccounting.ToolTokens, IdleTokens: report.TimeAccounting.IdleTokens, HumanTokens: report.TimeAccounting.HumanTokens, UnattributableTokens: report.TimeAccounting.UnattributableTokens},
	}
	if report.ExitCode != nil {
		code := int32(*report.ExitCode)
		out.ExitCode = &code
	}
	for eventType, count := range report.Events {
		out.EventCounts[eventType] = int32(count)
	}
	for _, tool := range report.Tools {
		out.Tools = append(out.Tools, &apipb.RunReportTool{Name: tool.Name, Calls: int32(tool.Calls), Successes: int32(tool.Successes), Failures: int32(tool.Failures), Unresolved: int32(tool.Unresolved), Returned: int32(tool.Returned), UnknownOutcome: int32(tool.UnknownOutcome)})
	}
	return out
}

func availabilityToProto(value runreport.Availability) *apipb.RunReportAvailability {
	return &apipb.RunReportAvailability{State: string(value.State), Reason: value.Reason}
}
