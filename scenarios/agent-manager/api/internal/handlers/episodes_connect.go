package handlers

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/runreport"
	"agent-manager/internal/runsignal"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetEpisodes implements the proto-first EpisodesService read surface.
func (h *Handler) GetEpisodes(ctx context.Context, req *connect.Request[domainpb.GetEpisodesRequest]) (*connect.Response[domainpb.GetEpisodesResponse], error) {
	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := h.svc.GetRun(ctx, runID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	episodes, err := h.svc.Episodes(ctx, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*domainpb.FrictionEpisode, 0, len(episodes))
	for _, episode := range episodes {
		out = append(out, episodeProto(episode))
	}
	return connect.NewResponse(&domainpb.GetEpisodesResponse{
		ClassifierVersion: runsignal.EpisodeClassifierVersion,
		Episodes:          out,
	}), nil
}

// GetSelfReportSpans exposes the deterministic, versioned assistant-message
// projection through the same proto-first investigation service as episodes.
func (h *Handler) GetSelfReportSpans(ctx context.Context, req *connect.Request[domainpb.GetSelfReportSpansRequest]) (*connect.Response[domainpb.GetSelfReportSpansResponse], error) {
	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	spans, err := h.svc.SelfReportSpans(ctx, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*domainpb.SelfReportSpan, 0, len(spans))
	for _, span := range spans {
		out = append(out, &domainpb.SelfReportSpan{ClassifierVersion: span.ClassifierVersion, EventId: span.EventID, RuleId: span.RuleID, CauseScope: span.CauseScope, StartOffset: int64(span.StartOffset), EndOffset: int64(span.EndOffset), Text: span.Text})
	}
	return connect.NewResponse(&domainpb.GetSelfReportSpansResponse{ClassifierVersion: runsignal.SelfReportClassifierVersion, Spans: out}), nil
}

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
	run, err := h.svc.ImportTranscript(ctx, orchestration.ImportTranscriptRequest{Path: req.Msg.GetPath(), RunnerType: domain.RunnerType(req.Msg.GetRunnerType()), Label: req.Msg.GetLabel(), LabelSource: labelSource})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&domainpb.ImportTranscriptResponse{RunId: run.ID.String(), Status: string(run.Status), ExecutionMode: string(run.ExecutionMode)}), nil
}

func availabilityProto(value runreport.Availability) *domainpb.Availability {
	return &domainpb.Availability{State: string(value.State), Reason: value.Reason}
}

func episodeProto(episode runsignal.FrictionEpisode) *domainpb.FrictionEpisode {
	return &domainpb.FrictionEpisode{
		EpisodeId:              episode.EpisodeID,
		RunId:                  episode.RunID,
		ClassifierVersion:      episode.ClassifierVersion,
		Pattern:                episode.Pattern,
		CauseScope:             episode.CauseScope,
		Severity:               episode.Severity,
		HonestyFlags:           append([]string(nil), episode.HonestyFlags...),
		StartEventId:           episode.StartEventID,
		EndEventId:             episode.EndEventID,
		EvidenceEventIds:       append([]string(nil), episode.EvidenceEventIDs...),
		Turns:                  int64(episode.Turns),
		Tokens:                 int64(episode.Tokens),
		WallClockMs:            episode.WallClockMS,
		SuspectedOwnerScenario: episode.SuspectedOwnerScenario,
		SuspectedOwnerCommand:  episode.SuspectedOwnerCommand,
		OwnerConfidence:        episode.OwnerConfidence,
		Fingerprint:            episode.Fingerprint,
		CycleCount:             int64(episode.CycleCount),
		RepeatedElement:        episode.RepeatedElement,
	}
}
