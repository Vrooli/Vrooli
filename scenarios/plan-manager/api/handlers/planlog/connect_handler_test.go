package planlog

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	internalplanlog "plan-manager/internal/planlog"
	planmodel "plan-manager/internal/planmodel"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"

	logv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// fakeLogService is a minimal stand-in for internalplanlog.Service that records
// the inputs it was called with and returns canned results.
type fakeLogService struct {
	entry  internalplanlog.Entry
	source internalplanlog.Entry
	dedup  bool
	err    error

	gotInputs  internalplanlog.AddInputs
	gotID      string
	gotUpdate  internalplanlog.UpdateInputs
	gotPromote planmodel.LogEntryType
	gotFilter  internalplanlog.Filter
}

func (f *fakeLogService) AddDecision(_ context.Context, in internalplanlog.AddInputs) (internalplanlog.Entry, bool, internalplanlog.GuidedStep, error) {
	f.gotInputs = in
	return f.entry, f.dedup, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) AddFinding(_ context.Context, in internalplanlog.AddInputs) (internalplanlog.Entry, bool, internalplanlog.GuidedStep, error) {
	f.gotInputs = in
	return f.entry, f.dedup, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) AddBug(_ context.Context, in internalplanlog.AddInputs) (internalplanlog.Entry, bool, internalplanlog.GuidedStep, error) {
	f.gotInputs = in
	return f.entry, f.dedup, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) AddRecord(_ context.Context, in internalplanlog.AddInputs) (internalplanlog.Entry, bool, internalplanlog.GuidedStep, error) {
	f.gotInputs = in
	return f.entry, f.dedup, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) AddNote(_ context.Context, in internalplanlog.AddInputs) (internalplanlog.Entry, bool, internalplanlog.GuidedStep, error) {
	f.gotInputs = in
	return f.entry, f.dedup, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) ListEntries(_ context.Context, fl internalplanlog.Filter) ([]internalplanlog.Entry, internalplanlog.Summary, internalplanlog.GuidedStep, error) {
	f.gotFilter = fl
	return []internalplanlog.Entry{f.entry}, internalplanlog.Summary{Total: 1}, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) GetEntry(_ context.Context, id string) (internalplanlog.Entry, internalplanlog.GuidedStep, error) {
	f.gotID = id
	return f.entry, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) UpdateEntry(_ context.Context, id string, in internalplanlog.UpdateInputs) (internalplanlog.Entry, internalplanlog.GuidedStep, error) {
	f.gotID, f.gotUpdate = id, in
	return f.entry, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) PromoteEntry(_ context.Context, id string, toType internalplanlog.EntryType, _, _ string, _ internalplanlog.Severity) (internalplanlog.Entry, internalplanlog.Entry, internalplanlog.GuidedStep, error) {
	f.gotID, f.gotPromote = id, toType
	return f.entry, f.source, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) SyncEntry(_ context.Context, id string) (internalplanlog.Entry, internalplanlog.GuidedStep, error) {
	f.gotID = id
	return f.entry, internalplanlog.GuidedStep{}, f.err
}

func (f *fakeLogService) Summarize(_ context.Context, fl internalplanlog.Filter) (internalplanlog.Summary, []internalplanlog.Entry, error) {
	f.gotFilter = fl
	return internalplanlog.Summary{Total: 1}, []internalplanlog.Entry{f.entry}, f.err
}

var _ internalplanlog.Service = (*fakeLogService)(nil)

func newLogHandler(svc internalplanlog.Service) *connectHandler {
	return NewConnectHandler(Deps{Service: svc, Logger: log.New(io.Discard, "", 0)})
}

func TestAddDecisionForwardsInputs(t *testing.T) {
	svc := &fakeLogService{entry: internalplanlog.Entry{ID: "le-1", Type: planmodel.LogEntryDecision, Title: "d"}}
	h := newLogHandler(svc)
	resp, err := h.AddDecision(context.Background(), connect.NewRequest(&logv1.AddDecisionRequest{
		PlanOrExecution: "exec-1", PhaseId: "ph-1", Title: "d", Detail: "why", Evidence: []string{"e1"},
		SourceCommand: "plan-manager log decision-add", IdempotencyKey: "k", RunId: "run-1",
	}))
	require.NoError(t, err)
	require.Equal(t, "le-1", resp.Msg.GetEntry().GetId())
	require.Equal(t, sharedv1.LogEntryType_LOG_ENTRY_TYPE_DECISION, resp.Msg.GetEntry().GetType())
	require.Equal(t, "exec-1", svc.gotInputs.PlanOrExecution)
	require.Equal(t, "ph-1", svc.gotInputs.PhaseID)
	require.Equal(t, []string{"e1"}, svc.gotInputs.Evidence)
	require.Equal(t, "k", svc.gotInputs.IdempotencyKey)
	require.Equal(t, "run-1", svc.gotInputs.RunID)
}

func TestAddBugForwardsSeverity(t *testing.T) {
	svc := &fakeLogService{entry: internalplanlog.Entry{ID: "le-2", Type: planmodel.LogEntryBugReport, SyncStatus: planmodel.LogSyncPending}}
	h := newLogHandler(svc)
	resp, err := h.AddBug(context.Background(), connect.NewRequest(&logv1.AddBugRequest{
		PlanOrExecution: "exec-1", Title: "bug", Severity: sharedv1.LogSeverity_LOG_SEVERITY_HIGH,
	}))
	require.NoError(t, err)
	require.Equal(t, sharedv1.LogSyncStatus_LOG_SYNC_STATUS_PENDING, resp.Msg.GetEntry().GetSyncStatus())
	require.Equal(t, planmodel.LogSeverityHigh, svc.gotInputs.Severity)
}

func TestAddDecisionReportsDedup(t *testing.T) {
	svc := &fakeLogService{entry: internalplanlog.Entry{ID: "le-1"}, dedup: true}
	h := newLogHandler(svc)
	resp, err := h.AddDecision(context.Background(), connect.NewRequest(&logv1.AddDecisionRequest{PlanOrExecution: "exec-1", Title: "d"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDeduplicated())
}

func TestListEntriesForwardsFilter(t *testing.T) {
	svc := &fakeLogService{entry: internalplanlog.Entry{ID: "le-1", Type: planmodel.LogEntryFinding}}
	h := newLogHandler(svc)
	resp, err := h.ListEntries(context.Background(), connect.NewRequest(&logv1.ListEntriesRequest{
		PlanOrExecution: "exec-1", Type: sharedv1.LogEntryType_LOG_ENTRY_TYPE_FINDING,
		Triage: sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE, SyncStatus: sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetEntries(), 1)
	require.Equal(t, int32(1), resp.Msg.GetSummary().GetTotal())
	require.Equal(t, planmodel.LogEntryFinding, svc.gotFilter.Type)
	require.Equal(t, planmodel.TriageCandidate, svc.gotFilter.Triage)
	require.Equal(t, planmodel.LogSyncLocal, svc.gotFilter.SyncStatus)
}

func TestUpdateEntryForwardsTriage(t *testing.T) {
	svc := &fakeLogService{entry: internalplanlog.Entry{ID: "le-1"}}
	h := newLogHandler(svc)
	_, err := h.UpdateEntry(context.Background(), connect.NewRequest(&logv1.UpdateEntryRequest{
		Id: "le-1", Triage: sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED, AddEvidence: []string{"x"},
	}))
	require.NoError(t, err)
	require.Equal(t, planmodel.TriageDismissed, svc.gotUpdate.Triage)
	require.Equal(t, []string{"x"}, svc.gotUpdate.AddEvidence)
}

func TestPromoteEntryForwardsType(t *testing.T) {
	svc := &fakeLogService{
		entry:  internalplanlog.Entry{ID: "bug-1", Type: planmodel.LogEntryBugReport, PromotedFromID: "f-1"},
		source: internalplanlog.Entry{ID: "f-1", Type: planmodel.LogEntryFinding, Triage: planmodel.TriagePromoted},
	}
	h := newLogHandler(svc)
	resp, err := h.PromoteEntry(context.Background(), connect.NewRequest(&logv1.PromoteEntryRequest{
		Id: "f-1", ToType: sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT,
	}))
	require.NoError(t, err)
	require.Equal(t, "f-1", svc.gotID)
	require.Equal(t, planmodel.LogEntryBugReport, svc.gotPromote)
	require.Equal(t, "f-1", resp.Msg.GetSource().GetId())
	require.Equal(t, "bug-1", resp.Msg.GetEntry().GetId())
}

func TestLogErrorMapping(t *testing.T) {
	t.Run("invalid_is_invalid_argument", func(t *testing.T) {
		h := newLogHandler(&fakeLogService{err: internalplanlog.ErrInvalidEntry{Reason: "title required"}})
		_, err := h.AddDecision(context.Background(), connect.NewRequest(&logv1.AddDecisionRequest{}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("not_found_is_not_found", func(t *testing.T) {
		h := newLogHandler(&fakeLogService{err: internalplanlog.ErrEntryNotFound{ID: "x"}})
		_, err := h.GetEntry(context.Background(), connect.NewRequest(&logv1.GetEntryRequest{Id: "x"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("not_promotable_is_failed_precondition", func(t *testing.T) {
		h := newLogHandler(&fakeLogService{err: internalplanlog.ErrNotPromotable{Reason: "not a finding"}})
		_, err := h.PromoteEntry(context.Background(), connect.NewRequest(&logv1.PromoteEntryRequest{Id: "x"}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})
	t.Run("unknown_is_internal", func(t *testing.T) {
		h := newLogHandler(&fakeLogService{err: errors.New("boom")})
		_, err := h.SyncEntry(context.Background(), connect.NewRequest(&logv1.SyncEntryRequest{Id: "x"}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
