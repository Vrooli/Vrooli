package handlers

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/runreport"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetCrossScenarioLedger returns bounded receipt evidence and availability
// states; the projection remains opaque and is only included on request.
func (h *Handler) GetCrossScenarioLedger(ctx context.Context, req *connect.Request[domainpb.GetCrossScenarioLedgerRequest]) (*connect.Response[domainpb.GetCrossScenarioLedgerResponse], error) {
	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	report, err := h.svc.BuildRunReport(ctx, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	calls := make([]*domainpb.CrossScenarioCall, 0, len(report.CrossScenarioCalls))
	for _, call := range report.CrossScenarioCalls {
		entry := &domainpb.CrossScenarioCall{OccurredAt: call.OccurredAt.UTC().Format(time.RFC3339Nano), TargetScenario: call.TargetScenario, Operation: call.Operation, Outcome: call.Outcome, StatusCode: call.StatusCode, DurationMs: call.DurationMS, ReceiptEventId: call.ReceiptEventID, Verified: call.Verified, ProjectionDropCount: int64(call.ProjectionDropCount), PolicyVersion: call.PolicyVersion}
		if req.Msg.GetWithProjections() && len(call.Projection) > 0 {
			projection, projectionErr := structpb.NewStruct(call.Projection)
			if projectionErr != nil {
				return nil, connect.NewError(connect.CodeInternal, projectionErr)
			}
			entry.Projection = projection
		}
		calls = append(calls, entry)
	}
	rollups := make([]*domainpb.LedgerTargetRollup, 0, len(report.LedgerTargetRollups))
	for _, rollup := range report.LedgerTargetRollups {
		rollups = append(rollups, &domainpb.LedgerTargetRollup{TargetScenario: rollup.TargetScenario, Calls: int64(rollup.Calls), Failures: int64(rollup.Failures), TotalDurationMs: rollup.TotalDurationMS, MedianDurationMs: rollup.MedianDurationMS})
	}
	return connect.NewResponse(&domainpb.GetCrossScenarioLedgerResponse{LedgerAvailability: availabilityProto(report.LedgerAvailability), ProjectionAvailability: availabilityProto(report.ProjectionAvailability), TargetRollups: rollups, Calls: calls}), nil
}

// ImportTranscript is the typed mutation counterpart to the read-only
// investigation surfaces. The orchestrator owns all parsing and persistence.
func (h *Handler) ImportTranscript(ctx context.Context, req *connect.Request[domainpb.ImportTranscriptRequest]) (*connect.Response[domainpb.ImportTranscriptResponse], error) {
	labelSource := domain.RunLabelSource("")
	if req.Msg.GetLabel() != "" {
		labelSource = domain.RunLabelSourceManual
	}
	run, err := h.svc.ImportTranscript(ctx, orchestration.ImportTranscriptRequest{Path: req.Msg.GetPath(), RunnerType: domain.RunnerType(req.Msg.GetRunnerType()), Label: req.Msg.GetLabel(), LabelSource: labelSource, SourceHarness: req.Msg.GetSourceHarness(), SourceSessionID: req.Msg.GetSourceSessionId()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&domainpb.ImportTranscriptResponse{RunId: run.ID.String(), Status: string(run.Status), ExecutionMode: string(run.ExecutionMode)}), nil
}

func availabilityProto(value runreport.Availability) *domainpb.Availability {
	return &domainpb.Availability{State: string(value.State), Reason: value.Reason}
}
