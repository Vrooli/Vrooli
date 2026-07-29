package handlers

import (
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
	writeRunReportJSON(w, http.StatusOK, runReportToProto(report))
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
		Turns: int32(report.Turns), Tokens: int32(report.Tokens), CostUsd: report.CostUSD,
		Result:      &apipb.RunReportResult{SelectionStatus: string(report.Result.SelectionStatus), SelectionRule: report.Result.SelectionRule, CandidateCount: int32(report.Result.CandidateCount), StructuredStatus: string(report.Result.StructuredStatus), StructuredMethod: report.Result.StructuredMethod, DiagnosticCodes: report.Result.DiagnosticCodes},
		EventCounts: map[string]int32{}, ProjectOwnedToolCalls: int32(report.ProjectOwnedToolCalls), ExternalToolCalls: int32(report.ExternalToolCalls),
		RequestedModel: report.RequestedModel, ActualModel: report.ActualModel, FallbackCount: int32(report.FallbackCount),
		Diff:               &apipb.RunReportDiff{Files: int32(report.Diff.Files), Bytes: report.Diff.Bytes, Available: availabilityToProto(report.Diff.Available)},
		EventsAvailability: availabilityToProto(report.EventsAvailability), ReceiptsAvailability: availabilityToProto(report.ReceiptsAvailability), ReceiptCount: int32(report.ReceiptCount),
		RepeatedToolCalls: int32(report.RepeatedToolCalls), LongestEventGapMs: report.LongestEventGap.Milliseconds(),
		FilesReadMoreThanOnce: int32(report.FilesReadMoreThanOnce),
	}
	if report.ExitCode != nil {
		code := int32(*report.ExitCode)
		out.ExitCode = &code
	}
	for eventType, count := range report.Events {
		out.EventCounts[eventType] = int32(count)
	}
	for _, tool := range report.Tools {
		out.Tools = append(out.Tools, &apipb.RunReportTool{Name: tool.Name, Calls: int32(tool.Calls), Successes: int32(tool.Successes), Failures: int32(tool.Failures), Unresolved: int32(tool.Unresolved)})
	}
	return out
}

func availabilityToProto(value runreport.Availability) *apipb.RunReportAvailability {
	return &apipb.RunReportAvailability{State: value.State, Detail: value.Detail}
}
