package planlog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"plan-manager/internal/planlog"
	planmodel "plan-manager/internal/planmodel"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/provenance"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

// --- fakes ---

type fakeResolver struct {
	scope planlog.Scope
	ok    bool
}

func (f fakeResolver) Resolve(context.Context, string) (planlog.Scope, bool, error) {
	if !f.ok {
		return planlog.Scope{}, false, nil
	}
	return f.scope, true, nil
}

// recordingSink is a configurable BugReporter + RecordWriter for sync tests.
type recordingSink struct {
	ref planmodel.DownstreamRef
	err error
}

func (s *recordingSink) FileBug(context.Context, planlog.Entry) (planlog.DownstreamRef, error) {
	return s.ref, s.err
}

func (s *recordingSink) WriteRecord(context.Context, planlog.Entry) (planlog.DownstreamRef, error) {
	return s.ref, s.err
}

// countingBugSink counts FileBug calls so a test can assert a side-effecting
// downstream forward fires at most once across repeat promotions.
type countingBugSink struct {
	ref  planmodel.DownstreamRef
	bugs int
}

func (s *countingBugSink) FileBug(context.Context, planlog.Entry) (planlog.DownstreamRef, error) {
	s.bugs++
	return s.ref, nil
}

// planEchoResolver maps a handle verbatim to a plan id with no execution scope,
// so a test can exercise the plan-handle (empty execution_id) dedup path across
// distinct plans.
type planEchoResolver struct{}

func (planEchoResolver) Resolve(_ context.Context, handle string) (planlog.Scope, bool, error) {
	return planlog.Scope{PlanID: handle}, true, nil
}

func newService(t *testing.T, d planlog.Deps) (planlog.Service, planlog.Repository) {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), sqlDB,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(planlog.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC))
	repo := planlog.NewSQLiteRepository(sqlDB, clk)
	d.Repo = repo
	d.Clock = clk
	if d.Resolver == nil {
		d.Resolver = fakeResolver{scope: defaultScope(), ok: true}
	}
	return planlog.NewService(d), repo
}

func defaultScope() planlog.Scope {
	return planlog.Scope{
		PlanID:      "plan-1",
		ExecutionID: "exec-1",
		Phases: []planlog.PhaseRef{
			{ID: "ph-1", Order: 1, Title: "First"},
			{ID: "ph-2", Order: 2, Title: "Second"},
		},
	}
}

// --- tests ---

func TestAddDecisionAndResolveScope(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	e, dedup, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", PhaseID: "ph-1", Title: "chose SQLite", Detail: "home store",
	})
	require.NoError(t, err)
	require.False(t, dedup)
	require.Equal(t, planmodel.LogEntryDecision, e.Type)
	require.Equal(t, "plan-1", e.PlanID, "resolver binds the entry to the canonical plan")
	require.Equal(t, "exec-1", e.ExecutionID)
	require.Equal(t, "ph-1", e.PhaseID)
	require.Equal(t, planmodel.LogSyncLocal, e.SyncStatus, "decisions are local-only")
}

func TestAddDecisionResolvesPhaseOrdinal(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	e, _, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", PhaseID: "2", Title: "use adapter seam",
	})
	require.NoError(t, err)
	require.Equal(t, "ph-2", e.PhaseID)
}

func TestAddDecisionInfersCurrentPhaseForExecutionHandle(t *testing.T) {
	scope := defaultScope()
	scope.CurrentPhaseID = "ph-2"
	svc, _ := newService(t, planlog.Deps{Resolver: fakeResolver{scope: scope, ok: true}})
	e, _, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", Title: "current phase decision",
	})
	require.NoError(t, err)
	require.Equal(t, "ph-2", e.PhaseID)
}

func TestAddDecisionRejectsUnknownPhase(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	_, _, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", PhaseID: "9", Title: "bad phase",
	})
	require.ErrorAs(t, err, &planlog.ErrInvalidEntry{})
}

func TestAddFindingFilesCandidate(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-x"})
	e, _, _, err := svc.AddFinding(ctx, planlog.AddInputs{
		PlanOrExecution: "exec-1", Title: "maybe a bug", RunID: "run-x",
	})
	require.NoError(t, err)
	require.Equal(t, planmodel.LogEntryFinding, e.Type)
	require.Equal(t, planmodel.TriageCandidate, e.Triage, "findings file as CANDIDATE, never auto-promoted")
	require.Equal(t, "run-x", e.AttributionRunID)
	require.Equal(t, provenance.VerificationVerified, e.VerificationStatus)
}

func TestAddEntryIgnoresLegacyRunIDEnvironmentClaim(t *testing.T) {
	t.Setenv("VROOLI_AGENT_MANAGER_RUN"+"_ID", "spoofed-run")

	svc, _ := newService(t, planlog.Deps{})
	e, _, _, err := svc.AddFinding(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", Title: "environment claim must not attribute",
	})
	require.NoError(t, err)
	require.Empty(t, e.AttributionRunID)
	require.Equal(t, provenance.VerificationAbsent, e.VerificationStatus)
}

func TestAddEntryPersistsHarnessObservationWithoutRunAttribution(t *testing.T) {
	svc, repo := newService(t, planlog.Deps{})
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Invocation: provenance.Invocation{HarnessSessionID: "claude-session-1", HarnessKind: "claude-code"}})
	e, _, _, err := svc.AddNote(ctx, planlog.AddInputs{PlanOrExecution: "exec-1", Title: "channel observation"})
	require.NoError(t, err)
	require.Empty(t, e.AttributionRunID)
	require.Equal(t, provenance.VerificationAbsent, e.VerificationStatus)
	stored, ok, err := repo.GetEntry(context.Background(), e.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "claude-session-1", stored.HarnessSessionID)
	require.Equal(t, "claude-code", stored.HarnessKind)
	require.Empty(t, stored.AttributionRunID)
}

func TestIdempotencyKeyDedup(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	first, dup1, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", Title: "d", IdempotencyKey: "key-1",
	})
	require.NoError(t, err)
	require.False(t, dup1)
	second, dup2, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", Title: "different title", IdempotencyKey: "key-1",
	})
	require.NoError(t, err)
	require.True(t, dup2, "same idempotency key returns the existing entry")
	require.Equal(t, first.ID, second.ID)
}

func TestAttributionDedup(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"})
	first, _, _, err := svc.AddFinding(ctx, planlog.AddInputs{
		PlanOrExecution: "exec-1", Title: "Config Drift", RunID: "run-1",
	})
	require.NoError(t, err)
	second, dup, _, err := svc.AddFinding(ctx, planlog.AddInputs{
		PlanOrExecution: "exec-1", Title: "  config drift ", RunID: "run-1",
	})
	require.NoError(t, err)
	require.True(t, dup, "same run id + type + normalized title is not double-filed")
	require.Equal(t, first.ID, second.ID)
}

func TestAddBugPendingWhenDownstreamUnavailable(t *testing.T) {
	// Default stub BugReporter reports unavailable.
	svc, _ := newService(t, planlog.Deps{})
	e, _, _, err := svc.AddBug(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "real bug"})
	require.NoError(t, err, "downstream unavailability is never fatal")
	require.Equal(t, planmodel.LogEntryBugReport, e.Type)
	require.Equal(t, planmodel.LogSyncPending, e.SyncStatus, "unavailable downstream leaves the entry pending+retryable")
	require.NotEmpty(t, e.Downstream.Detail)
}

func TestAddRecordSyncsViaSink(t *testing.T) {
	sink := &recordingSink{ref: planmodel.DownstreamRef{System: "swarm-manager", Kind: "record", Reference: "rec-123"}}
	svc, _ := newService(t, planlog.Deps{Records: sink})
	e, _, _, err := svc.AddRecord(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "reusable win"})
	require.NoError(t, err)
	require.Equal(t, planmodel.LogSyncSynced, e.SyncStatus)
	require.Equal(t, "rec-123", e.Downstream.Reference)
	require.NotEmpty(t, e.Downstream.SyncedAt)
}

func TestAddBugFailedSyncStaysDurable(t *testing.T) {
	sink := &recordingSink{err: errors.New("tracker rejected")}
	svc, repo := newService(t, planlog.Deps{Bugs: sink})
	e, _, _, err := svc.AddBug(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "bug"})
	require.NoError(t, err, "a downstream error is non-fatal")
	require.Equal(t, planmodel.LogSyncFailed, e.SyncStatus)
	// The entry is durable even though the downstream write failed.
	stored, ok, err := repo.GetEntry(context.Background(), e.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, planmodel.LogSyncFailed, stored.SyncStatus)
}

func TestSyncEntryRetrySucceeds(t *testing.T) {
	sink := &recordingSink{err: errors.New("temporarily down")}
	svc, _ := newService(t, planlog.Deps{Records: sink})
	e, _, _, err := svc.AddRecord(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "win"})
	require.NoError(t, err)
	require.Equal(t, planmodel.LogSyncFailed, e.SyncStatus)

	// Downstream recovers; retry via SyncEntry succeeds.
	sink.err = nil
	sink.ref = planmodel.DownstreamRef{System: "swarm-manager", Kind: "record", Reference: "rec-999"}
	synced, _, err := svc.SyncEntry(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, planmodel.LogSyncSynced, synced.SyncStatus)
	require.Equal(t, "rec-999", synced.Downstream.Reference)
}

func TestPromoteFindingToBugPreservesOrigin(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	finding, _, _, err := svc.AddFinding(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "leak"})
	require.NoError(t, err)

	bug, source, _, err := svc.PromoteEntry(context.Background(), finding.ID, planmodel.LogEntryBugReport, "", "", planmodel.LogSeverityHigh)
	require.NoError(t, err)
	require.Equal(t, planmodel.LogEntryBugReport, bug.Type)
	require.Equal(t, finding.ID, bug.PromotedFromID, "the promoted entry links back to the finding")
	require.Equal(t, planmodel.LogSeverityHigh, bug.Severity)
	require.Equal(t, planmodel.TriagePromoted, source.Triage, "the original finding is preserved, marked promoted")
}

func TestPromoteIsIdempotent(t *testing.T) {
	sink := &countingBugSink{ref: planmodel.DownstreamRef{System: "scenario-qa", Kind: "bug", Reference: "bug-1"}}
	svc, _ := newService(t, planlog.Deps{Bugs: sink})
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"})
	finding, _, _, err := svc.AddFinding(ctx, planlog.AddInputs{PlanOrExecution: "exec-1", Title: "leak"})
	require.NoError(t, err)

	first, _, _, err := svc.PromoteEntry(ctx, finding.ID, planmodel.LogEntryBugReport, "", "", planmodel.LogSeverityHigh)
	require.NoError(t, err)
	// Promoting the same finding again returns the existing promotion, does not
	// create a second entry, and does not re-fire the downstream forward.
	second, source, _, err := svc.PromoteEntry(ctx, finding.ID, planmodel.LogEntryBugReport, "", "", planmodel.LogSeverityHigh)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "re-promote returns the existing promotion")
	require.Equal(t, planmodel.TriagePromoted, source.Triage)
	require.Equal(t, 1, sink.bugs, "downstream bug is filed exactly once across repeat promotions")

	bugs, _, _, err := svc.ListEntries(ctx, planlog.Filter{ExecutionID: "exec-1", Type: planmodel.LogEntryBugReport})
	require.NoError(t, err)
	require.Len(t, bugs, 1, "only one bug_report exists after double promote")
}

func TestAttributionDedupIsScopedPerPlan(t *testing.T) {
	// planEchoResolver gives each handle its own plan id with empty execution
	// scope — the path where a missing plan_id would have falsely deduped.
	svc, _ := newService(t, planlog.Deps{Resolver: planEchoResolver{}})
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"})
	a, dupA, _, err := svc.AddFinding(ctx, planlog.AddInputs{PlanOrExecution: "plan-A", Title: "Config Drift", RunID: "run-1"})
	require.NoError(t, err)
	require.False(t, dupA)
	b, dupB, _, err := svc.AddFinding(ctx, planlog.AddInputs{PlanOrExecution: "plan-B", Title: "Config Drift", RunID: "run-1"})
	require.NoError(t, err)
	require.False(t, dupB, "same run id + title in a DIFFERENT plan is a distinct entry, not a dup")
	require.NotEqual(t, a.ID, b.ID)
	require.Equal(t, "plan-A", a.PlanID)
	require.Equal(t, "plan-B", b.PlanID)

	// Same plan + run id + title still dedups.
	again, dupAgain, _, err := svc.AddFinding(ctx, planlog.AddInputs{PlanOrExecution: "plan-A", Title: "config drift", RunID: "run-1"})
	require.NoError(t, err)
	require.True(t, dupAgain, "same plan + run id + normalized title is not double-filed")
	require.Equal(t, a.ID, again.ID)
}

func TestPromoteRejectsNonFinding(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	dec, _, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "d"})
	require.NoError(t, err)
	_, _, _, err = svc.PromoteEntry(context.Background(), dec.ID, planmodel.LogEntryBugReport, "", "", "")
	require.ErrorAs(t, err, &planlog.ErrNotPromotable{})
}

func TestUpdateEntryTriageAndEvidence(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	f, _, _, err := svc.AddFinding(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "x", Evidence: []string{"a"}})
	require.NoError(t, err)
	updated, _, err := svc.UpdateEntry(context.Background(), f.ID, planlog.UpdateInputs{
		Triage:      planmodel.TriageDismissed,
		AddEvidence: []string{"b", "c"},
	})
	require.NoError(t, err)
	require.Equal(t, planmodel.TriageDismissed, updated.Triage)
	require.Equal(t, []string{"a", "b", "c"}, updated.Evidence)
}

func TestReassignEntryResolvesOrdinal(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	e, _, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{
		PlanOrExecution: "exec-1", PhaseID: "ph-1", Title: "needs move",
	})
	require.NoError(t, err)
	updated, _, err := svc.ReassignEntry(context.Background(), e.ID, "2")
	require.NoError(t, err)
	require.Equal(t, "ph-2", updated.PhaseID)
}

func TestListEntriesAndSummary(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	ctx := context.Background()
	_, _, _, err := svc.AddDecision(ctx, planlog.AddInputs{PlanOrExecution: "exec-1", Title: "d1"})
	require.NoError(t, err)
	_, _, _, err = svc.AddFinding(ctx, planlog.AddInputs{PlanOrExecution: "exec-1", Title: "f1"})
	require.NoError(t, err)
	_, _, _, err = svc.AddBug(ctx, planlog.AddInputs{PlanOrExecution: "exec-1", Title: "b1"})
	require.NoError(t, err)

	entries, summary, _, err := svc.ListEntries(ctx, planlog.Filter{ExecutionID: "exec-1"})
	require.NoError(t, err)
	require.Len(t, entries, 3)
	require.Equal(t, 3, summary.Total)
	require.Equal(t, 1, summary.Decisions)
	require.Equal(t, 1, summary.Findings)
	require.Equal(t, 1, summary.BugReports)
	require.Equal(t, 1, summary.CandidateFindings)
	require.Equal(t, 1, summary.PendingSync, "the unsynced bug is pending")

	// Type filter.
	findings, _, _, err := svc.ListEntries(ctx, planlog.Filter{ExecutionID: "exec-1", Type: planmodel.LogEntryFinding})
	require.NoError(t, err)
	require.Len(t, findings, 1)
}

func TestSummarizeForExecutionSeam(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	ctx := context.Background()
	_, _, _, err := svc.AddDecision(ctx, planlog.AddInputs{PlanOrExecution: "exec-1", Title: "d"})
	require.NoError(t, err)
	summary, entries, err := svc.Summarize(ctx, planlog.Filter{ExecutionID: "exec-1"})
	require.NoError(t, err)
	require.Equal(t, 1, summary.Total)
	require.Len(t, entries, 1)
}

func TestAddRequiresTitle(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	_, _, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{PlanOrExecution: "exec-1", Title: "  "})
	require.ErrorAs(t, err, &planlog.ErrInvalidEntry{})
}

func TestGetEntryNotFound(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{})
	_, _, err := svc.GetEntry(context.Background(), "missing")
	require.ErrorAs(t, err, &planlog.ErrEntryNotFound{})
}

func TestUnresolvedHandleIsInvalid(t *testing.T) {
	svc, _ := newService(t, planlog.Deps{Resolver: fakeResolver{ok: false}})
	_, _, _, err := svc.AddDecision(context.Background(), planlog.AddInputs{PlanOrExecution: "nope", Title: "d"})
	require.ErrorAs(t, err, &planlog.ErrInvalidEntry{})
}
