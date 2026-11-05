package runs

import (
	"context"
	"log"
	"time"

	"data-backup-manager/internal/failures"
	internalruns "data-backup-manager/internal/runs"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs"
)

// VerifiedInfo is the proven-restorable rollup for one target: the latest
// successful verify and the snapshot it proved.
type VerifiedInfo struct {
	LastVerifiedAt time.Time
	SnapshotID     string
}

// VerifiedLookup supplies the latest successful verify per target, keyed by
// target id, so ListTargetStatus can carry "proven restorable" posture
// alongside "backed up" posture in a single call (no per-target fan-out).
//
// seam: satisfied by an adapter over restores.Service in main.go. Optional — a
// nil lookup yields no verified data, so every target renders as unverified.
type VerifiedLookup interface {
	LastVerifiedByTarget(ctx context.Context) (map[string]VerifiedInfo, error)
}

// Deps wires the seams the Connect runs handler needs.
type Deps struct {
	Service internalruns.Service
	// Verified is optional; when nil, ListTargetStatus omits verified posture.
	Verified VerifiedLookup
	Logger   *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the runs Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) TriggerRun(ctx context.Context, req *connect.Request[runsv1.TriggerRunRequest]) (*connect.Response[runsv1.TriggerRunResponse], error) {
	run, err := h.deps.Service.TriggerRun(ctx, req.Msg.PlanId, internalruns.TriggerManual)
	if err != nil {
		return nil, h.translate("TriggerRun", err)
	}
	return connect.NewResponse(&runsv1.TriggerRunResponse{Run: runToProto(run)}), nil
}

func (h *connectHandler) PreflightRun(ctx context.Context, req *connect.Request[runsv1.PreflightRunRequest]) (*connect.Response[runsv1.PreflightRunResponse], error) {
	result, err := h.deps.Service.Preflight(ctx, req.Msg.PlanId)
	if err != nil {
		return nil, h.translate("PreflightRun", err)
	}
	resp := &runsv1.PreflightRunResponse{Ready: result.Ready}
	if !result.CheckedAt.IsZero() {
		resp.CheckedAt = timestamppb.New(result.CheckedAt)
	}
	for _, incident := range result.Incidents {
		resp.Incidents = append(resp.Incidents, failureCauseToProto(incident))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetRun(ctx context.Context, req *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	run, err := h.deps.Service.GetRun(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetRun", err)
	}
	return connect.NewResponse(&runsv1.GetRunResponse{Run: runToProto(run)}), nil
}

func (h *connectHandler) ListRuns(ctx context.Context, req *connect.Request[runsv1.ListRunsRequest]) (*connect.Response[runsv1.ListRunsResponse], error) {
	list, err := h.deps.Service.ListRuns(ctx, req.Msg.PlanId, int(req.Msg.PageSize))
	if err != nil {
		return nil, h.translate("ListRuns", err)
	}
	resp := &runsv1.ListRunsResponse{Runs: make([]*runsv1.Run, 0, len(list))}
	for _, r := range list {
		resp.Runs = append(resp.Runs, runToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListTargetStatus(ctx context.Context, req *connect.Request[runsv1.ListTargetStatusRequest]) (*connect.Response[runsv1.ListTargetStatusResponse], error) {
	// owner filtering is not yet wired here — runs keys on target id. The
	// health rollup applies the owner mapping (see handlers/health). v1 returns
	// all targets seen in run history.
	statuses, err := h.deps.Service.ListTargetStatus(ctx, nil)
	if err != nil {
		return nil, h.translate("ListTargetStatus", err)
	}

	// Enrich with proven-restorable posture from restore history. A target with
	// a recent backup but no (or a stale) verify is the case the UI flags — so
	// "verified" is composed here, not derived from run history.
	var verified map[string]VerifiedInfo
	if h.deps.Verified != nil {
		verified, err = h.deps.Verified.LastVerifiedByTarget(ctx)
		if err != nil {
			return nil, h.translate("ListTargetStatus", err)
		}
	}

	resp := &runsv1.ListTargetStatusResponse{Statuses: make([]*runsv1.TargetStatus, 0, len(statuses))}
	for _, s := range statuses {
		ps := targetStatusToProto(s)
		if v, ok := verified[s.TargetID]; ok && !v.LastVerifiedAt.IsZero() {
			ps.LastVerifiedAt = timestamppb.New(v.LastVerifiedAt)
			ps.LastVerifiedSnapshotId = v.SnapshotID
		}
		resp.Statuses = append(resp.Statuses, ps)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) BrowseSnapshot(ctx context.Context, req *connect.Request[runsv1.BrowseSnapshotRequest]) (*connect.Response[runsv1.BrowseSnapshotResponse], error) {
	entries, err := h.deps.Service.BrowseSnapshot(ctx, req.Msg.DestinationId, req.Msg.SnapshotId, req.Msg.Path)
	if err != nil {
		return nil, h.translate("BrowseSnapshot", err)
	}
	resp := &runsv1.BrowseSnapshotResponse{Entries: make([]*runsv1.SnapshotEntry, 0, len(entries))}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, &runsv1.SnapshotEntry{Path: e.Path, SizeBytes: e.SizeBytes, IsDir: e.IsDir})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetRunStats(ctx context.Context, req *connect.Request[runsv1.GetRunStatsRequest]) (*connect.Response[runsv1.GetRunStatsResponse], error) {
	stats, err := h.deps.Service.GetRunStats(ctx, req.Msg.PlanId)
	if err != nil {
		return nil, h.translate("GetRunStats", err)
	}
	return connect.NewResponse(&runsv1.GetRunStatsResponse{Stats: runStatsToProto(stats)}), nil
}

func runStatsToProto(s internalruns.RunStats) *runsv1.RunStats {
	return &runsv1.RunStats{
		TotalRuns:                s.TotalRuns,
		Completed:                s.Completed,
		PartialFailed:            s.PartialFailed,
		Failed:                   s.Failed,
		SuccessRate:              s.SuccessRate,
		P50DurationMs:            s.P50DurationMs,
		P95DurationMs:            s.P95DurationMs,
		TotalBytes:               s.TotalBytes,
		AvgBytesPerRun:           s.AvgBytesPerRun,
		AvgThroughputBytesPerSec: s.AvgThroughputBytesPerSec,
		Window:                   s.Window,
		TotalPhysicalBytes:       s.TotalPhysicalBytes,
		DedupRatio:               s.DedupRatio,
	}
}

func (h *connectHandler) translate(op string, err error) error {
	connectErr := internalruns.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("runs.%s: %v", op, err)
	}
	return connectErr
}

func runToProto(r internalruns.Run) *runsv1.Run {
	pr := &runsv1.Run{
		Id:              r.ID,
		PlanId:          r.PlanID,
		Trigger:         triggerToProto(r.Trigger),
		Status:          runStatusToProto(r.Status),
		Error:           r.Error,
		FailureCode:     string(r.FailureCode),
		FailureCategory: string(r.FailureCategory),
		NextAction:      r.NextAction,
		PhysicalBytes:   r.PhysicalBytes,
	}
	if !r.StartedAt.IsZero() {
		pr.StartedAt = timestamppb.New(r.StartedAt)
	}
	if !r.FinishedAt.IsZero() {
		pr.FinishedAt = timestamppb.New(r.FinishedAt)
	}
	if !r.UpdatedAt.IsZero() {
		pr.UpdatedAt = timestamppb.New(r.UpdatedAt)
	}
	for _, c := range r.Preflight {
		pr.PreflightIncidents = append(pr.PreflightIncidents, failureCauseToProto(c))
	}
	for _, o := range r.Outcomes {
		po := &runsv1.TargetOutcome{
			TargetId:        o.TargetID,
			DestinationId:   o.DestinationID,
			Status:          outcomeToProto(o.Status),
			SnapshotId:      o.SnapshotID,
			Bytes:           o.Bytes,
			Error:           o.Error,
			FailureCode:     string(o.FailureCode),
			FailureCategory: string(o.FailureCategory),
			Warning:         o.Warning,
		}
		if !o.StartedAt.IsZero() {
			po.StartedAt = timestamppb.New(o.StartedAt)
		}
		if !o.FinishedAt.IsZero() {
			po.FinishedAt = timestamppb.New(o.FinishedAt)
		}
		pr.Outcomes = append(pr.Outcomes, po)
	}
	return pr
}

func failureCauseToProto(c failures.Cause) *runsv1.FailureCause {
	out := &runsv1.FailureCause{
		Code: string(c.Code), Category: string(c.Category), Scope: string(c.Scope),
		Message: c.Message, NextAction: c.NextAction, DestinationId: c.DestinationID,
		TargetIds: append([]string(nil), c.TargetIDs...), Retryable: c.Retryable,
	}
	if c.RetryAfter > 0 {
		out.RetryAfterSeconds = int64(c.RetryAfter.Seconds())
	}
	if !c.FirstObserved.IsZero() {
		out.FirstObserved = timestamppb.New(c.FirstObserved)
	}
	if !c.LastObserved.IsZero() {
		out.LastObserved = timestamppb.New(c.LastObserved)
	}
	if !c.LastKnownGood.IsZero() {
		out.LastKnownGood = timestamppb.New(c.LastKnownGood)
	}
	return out
}

func targetStatusToProto(s internalruns.TargetStatus) *runsv1.TargetStatus {
	ps := &runsv1.TargetStatus{
		TargetId:              s.TargetID,
		LastRunStatus:         runStatusToProto(s.LastRunStatus),
		Overdue:               s.Overdue,
		LastSuccessAgeSeconds: s.LastSuccessAgeSeconds,
	}
	if !s.LastSuccessAt.IsZero() {
		ps.LastSuccessAt = timestamppb.New(s.LastSuccessAt)
	}
	if !s.LastRunAt.IsZero() {
		ps.LastRunAt = timestamppb.New(s.LastRunAt)
	}
	if !s.NextScheduledAt.IsZero() {
		ps.NextScheduledAt = timestamppb.New(s.NextScheduledAt)
	}
	return ps
}

func runStatusToProto(s internalruns.RunStatus) runsv1.RunStatus {
	switch s {
	case internalruns.RunPending:
		return runsv1.RunStatus_RUN_STATUS_PENDING
	case internalruns.RunCapturing:
		return runsv1.RunStatus_RUN_STATUS_CAPTURING
	case internalruns.RunSnapshotting:
		return runsv1.RunStatus_RUN_STATUS_SNAPSHOTTING
	case internalruns.RunCompleted:
		return runsv1.RunStatus_RUN_STATUS_COMPLETED
	case internalruns.RunPartialFailed:
		return runsv1.RunStatus_RUN_STATUS_PARTIAL_FAILED
	case internalruns.RunFailed:
		return runsv1.RunStatus_RUN_STATUS_FAILED
	default:
		return runsv1.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}

func triggerToProto(t internalruns.TriggerSource) runsv1.TriggerSource {
	switch t {
	case internalruns.TriggerScheduler:
		return runsv1.TriggerSource_TRIGGER_SOURCE_SCHEDULER
	case internalruns.TriggerManual:
		return runsv1.TriggerSource_TRIGGER_SOURCE_MANUAL
	default:
		return runsv1.TriggerSource_TRIGGER_SOURCE_UNSPECIFIED
	}
}

func outcomeToProto(s internalruns.OutcomeStatus) runsv1.TargetOutcomeStatus {
	switch s {
	case internalruns.OutcomeSucceeded:
		return runsv1.TargetOutcomeStatus_TARGET_OUTCOME_STATUS_SUCCEEDED
	case internalruns.OutcomeFailed:
		return runsv1.TargetOutcomeStatus_TARGET_OUTCOME_STATUS_FAILED
	case internalruns.OutcomeBlocked:
		return runsv1.TargetOutcomeStatus_TARGET_OUTCOME_STATUS_BLOCKED
	default:
		return runsv1.TargetOutcomeStatus_TARGET_OUTCOME_STATUS_UNSPECIFIED
	}
}
