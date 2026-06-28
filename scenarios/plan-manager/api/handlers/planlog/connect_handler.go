package planlog

import (
	"context"
	"log"

	internalplanlog "plan-manager/internal/planlog"
	"plan-manager/internal/planproto"

	"connectrpc.com/connect"

	logv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log"
)

// Deps wires the seams the Connect log handler needs.
type Deps struct {
	Service internalplanlog.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the LogService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) AddDecision(ctx context.Context, req *connect.Request[logv1.AddDecisionRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	m := req.Msg
	entry, dedup, step, err := h.deps.Service.AddDecision(ctx, internalplanlog.AddInputs{
		PlanOrExecution: m.GetPlanOrExecution(),
		PhaseID:         m.GetPhaseId(),
		Title:           m.GetTitle(),
		Detail:          m.GetDetail(),
		Evidence:        m.GetEvidence(),
		SourceCommand:   m.GetSourceCommand(),
		IdempotencyKey:  m.GetIdempotencyKey(),
		RunID:           m.GetRunId(),
	})
	return addResponse(entry, dedup, step, err)
}

func (h *connectHandler) AddFinding(ctx context.Context, req *connect.Request[logv1.AddFindingRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	m := req.Msg
	entry, dedup, step, err := h.deps.Service.AddFinding(ctx, internalplanlog.AddInputs{
		PlanOrExecution: m.GetPlanOrExecution(),
		PhaseID:         m.GetPhaseId(),
		Title:           m.GetTitle(),
		Detail:          m.GetDetail(),
		Severity:        planproto.LogSeverityFromProto(m.GetSeverity()),
		Evidence:        m.GetEvidence(),
		SourceCommand:   m.GetSourceCommand(),
		IdempotencyKey:  m.GetIdempotencyKey(),
		RunID:           m.GetRunId(),
	})
	return addResponse(entry, dedup, step, err)
}

func (h *connectHandler) AddBug(ctx context.Context, req *connect.Request[logv1.AddBugRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	m := req.Msg
	entry, dedup, step, err := h.deps.Service.AddBug(ctx, internalplanlog.AddInputs{
		PlanOrExecution: m.GetPlanOrExecution(),
		PhaseID:         m.GetPhaseId(),
		Title:           m.GetTitle(),
		Detail:          m.GetDetail(),
		Severity:        planproto.LogSeverityFromProto(m.GetSeverity()),
		Evidence:        m.GetEvidence(),
		SourceCommand:   m.GetSourceCommand(),
		IdempotencyKey:  m.GetIdempotencyKey(),
		RunID:           m.GetRunId(),
	})
	return addResponse(entry, dedup, step, err)
}

func (h *connectHandler) AddRecord(ctx context.Context, req *connect.Request[logv1.AddRecordRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	m := req.Msg
	entry, dedup, step, err := h.deps.Service.AddRecord(ctx, internalplanlog.AddInputs{
		PlanOrExecution: m.GetPlanOrExecution(),
		PhaseID:         m.GetPhaseId(),
		Title:           m.GetTitle(),
		Detail:          m.GetDetail(),
		Evidence:        m.GetEvidence(),
		SourceCommand:   m.GetSourceCommand(),
		IdempotencyKey:  m.GetIdempotencyKey(),
		RunID:           m.GetRunId(),
	})
	return addResponse(entry, dedup, step, err)
}

func (h *connectHandler) AddNote(ctx context.Context, req *connect.Request[logv1.AddNoteRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	m := req.Msg
	entry, dedup, step, err := h.deps.Service.AddNote(ctx, internalplanlog.AddInputs{
		PlanOrExecution: m.GetPlanOrExecution(),
		PhaseID:         m.GetPhaseId(),
		Title:           m.GetTitle(),
		Detail:          m.GetDetail(),
		Evidence:        m.GetEvidence(),
		SourceCommand:   m.GetSourceCommand(),
		IdempotencyKey:  m.GetIdempotencyKey(),
		RunID:           m.GetRunId(),
	})
	return addResponse(entry, dedup, step, err)
}

func addResponse(entry internalplanlog.Entry, dedup bool, step internalplanlog.GuidedStep, err error) (*connect.Response[logv1.AddEntryResponse], error) {
	if err != nil {
		return nil, internalplanlog.ToConnectError(err)
	}
	return connect.NewResponse(&logv1.AddEntryResponse{
		Entry:        entryToProto(entry),
		Step:         guidedStepToProto(step),
		Deduplicated: dedup,
	}), nil
}

func (h *connectHandler) ListEntries(ctx context.Context, req *connect.Request[logv1.ListEntriesRequest]) (*connect.Response[logv1.ListEntriesResponse], error) {
	m := req.Msg
	entries, summary, step, err := h.deps.Service.ListEntries(ctx, internalplanlog.Filter{
		PlanID:     m.GetPlanOrExecution(),
		PhaseID:    m.GetPhaseId(),
		Type:       planproto.LogEntryTypeFromProto(m.GetType()),
		Triage:     planproto.TriageFromProto(m.GetTriage()),
		SyncStatus: planproto.LogSyncStatusFromProto(m.GetSyncStatus()),
	})
	if err != nil {
		return nil, internalplanlog.ToConnectError(err)
	}
	return connect.NewResponse(&logv1.ListEntriesResponse{
		Entries: entriesToProto(entries),
		Summary: summaryToProto(summary),
		Step:    guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) GetEntry(ctx context.Context, req *connect.Request[logv1.GetEntryRequest]) (*connect.Response[logv1.GetEntryResponse], error) {
	entry, step, err := h.deps.Service.GetEntry(ctx, req.Msg.GetId())
	if err != nil {
		return nil, internalplanlog.ToConnectError(err)
	}
	return connect.NewResponse(&logv1.GetEntryResponse{Entry: entryToProto(entry), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) UpdateEntry(ctx context.Context, req *connect.Request[logv1.UpdateEntryRequest]) (*connect.Response[logv1.GetEntryResponse], error) {
	m := req.Msg
	entry, step, err := h.deps.Service.UpdateEntry(ctx, m.GetId(), internalplanlog.UpdateInputs{
		Title:       m.GetTitle(),
		Detail:      m.GetDetail(),
		Severity:    planproto.LogSeverityFromProto(m.GetSeverity()),
		Triage:      planproto.TriageFromProto(m.GetTriage()),
		AddEvidence: m.GetAddEvidence(),
	})
	if err != nil {
		return nil, internalplanlog.ToConnectError(err)
	}
	return connect.NewResponse(&logv1.GetEntryResponse{Entry: entryToProto(entry), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) PromoteEntry(ctx context.Context, req *connect.Request[logv1.PromoteEntryRequest]) (*connect.Response[logv1.PromoteEntryResponse], error) {
	m := req.Msg
	entry, source, step, err := h.deps.Service.PromoteEntry(ctx, m.GetId(),
		planproto.LogEntryTypeFromProto(m.GetToType()), m.GetTitle(), m.GetDetail(),
		planproto.LogSeverityFromProto(m.GetSeverity()))
	if err != nil {
		return nil, internalplanlog.ToConnectError(err)
	}
	return connect.NewResponse(&logv1.PromoteEntryResponse{
		Entry:  entryToProto(entry),
		Source: entryToProto(source),
		Step:   guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) SyncEntry(ctx context.Context, req *connect.Request[logv1.SyncEntryRequest]) (*connect.Response[logv1.GetEntryResponse], error) {
	entry, step, err := h.deps.Service.SyncEntry(ctx, req.Msg.GetId())
	if err != nil {
		return nil, internalplanlog.ToConnectError(err)
	}
	return connect.NewResponse(&logv1.GetEntryResponse{Entry: entryToProto(entry), Step: guidedStepToProto(step)}), nil
}
