package optimization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"network-manager/internal/adapters"
	"network-manager/internal/snapshot"
)

func TestCreateRunRequiresBaseline(t *testing.T) {
	// [REQ:NM-P0-005] Optimization cannot start until baseline snapshot evidence exists.
	svc := newTestService(nil, []adapters.Capability{capability("read_network_status", true, true)}, fakeApplier{})

	_, err := svc.CreateRun(context.Background(), "", false)
	require.ErrorIs(t, err, ErrBaselineRequired)
}

func TestCandidateCaptureAndScoring(t *testing.T) {
	// [REQ:NM-P0-005] Candidates preserve before/candidate evidence and score reliability-first metrics.
	store := newSnapshotStore(baselineSnapshot())
	svc := newTestService(store, []adapters.Capability{capability("resolver_status", true, true)}, fakeApplier{})

	run, err := svc.CreateRun(context.Background(), "reliability", false)
	require.NoError(t, err)
	require.Equal(t, "draft", run.Status)
	require.Len(t, run.Candidates, 1)

	run, err = svc.RunCandidate(context.Background(), run.ID, run.Candidates[0].ID)
	require.NoError(t, err)
	require.Equal(t, "candidates_running", run.Status)
	require.NotEmpty(t, run.Candidates[0].CandidateSnapshotID)

	run, err = svc.Score(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, "scored", run.Status)
	require.Greater(t, run.Candidates[0].Score, 50.0)
	require.Contains(t, run.Candidates[0].Evidence[len(run.Candidates[0].Evidence)-1], "Reliability-first score")
}

func TestApprovalIsRequiredBeforePersistentApply(t *testing.T) {
	// [REQ:NM-P0-005] Persistent candidate apply is approval-gated.
	store := newSnapshotStore(baselineSnapshot())
	svc := newTestService(store, []adapters.Capability{capability("manage_dns_filtering", true, true)}, fakeApplier{})
	run, err := svc.CreateRun(context.Background(), "", false)
	require.NoError(t, err)

	run, err = svc.Approve(context.Background(), run.ID, run.Candidates[0].ID, false)
	require.NoError(t, err)
	require.Equal(t, "awaiting_approval", run.Status)
	require.Equal(t, "approval_required", run.Candidates[0].Status)
}

func TestApprovedApplyAndRollback(t *testing.T) {
	// [REQ:NM-P0-005] Approved optimization candidates can apply, verify, and roll back through a capable adapter seam.
	store := newSnapshotStore(baselineSnapshot())
	svc := newTestService(store, []adapters.Capability{capability("manage_dns_filtering", true, true)}, fakeApplier{})
	run, err := svc.CreateRun(context.Background(), "", false)
	require.NoError(t, err)
	run, err = svc.RunCandidate(context.Background(), run.ID, run.Candidates[0].ID)
	require.NoError(t, err)
	run, err = svc.Score(context.Background(), run.ID)
	require.NoError(t, err)

	run, err = svc.Approve(context.Background(), run.ID, run.Candidates[0].ID, true)
	require.NoError(t, err)
	require.Equal(t, "verified", run.Status)
	require.Equal(t, "applied", run.Candidates[0].Status)
	require.NotEmpty(t, run.Candidates[0].RollbackHandle)
	require.NotEmpty(t, run.Candidates[0].AfterSnapshotID)

	run, err = svc.Rollback(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", run.Status)
	require.Equal(t, "rolled_back", run.Candidates[0].Status)
}

func TestApplyFailureIsTerminal(t *testing.T) {
	// [REQ:NM-P0-005] Apply failures are recorded instead of being reported as successful optimization.
	store := newSnapshotStore(baselineSnapshot())
	svc := newTestService(store, []adapters.Capability{capability("manage_dns_filtering", true, true)}, fakeApplier{applyErr: errors.New("adapter apply failed")})
	run, err := svc.CreateRun(context.Background(), "", false)
	require.NoError(t, err)

	run, err = svc.Approve(context.Background(), run.ID, run.Candidates[0].ID, true)
	require.NoError(t, err)
	require.Equal(t, "failed", run.Status)
	require.Equal(t, "apply_failed", run.Candidates[0].Status)
	require.Contains(t, run.Candidates[0].Evidence, "adapter apply failed")
}

func TestRollbackFailureIsRecorded(t *testing.T) {
	// [REQ:NM-P0-005] Rollback failures remain visible for operator recovery.
	store := newSnapshotStore(baselineSnapshot())
	svc := newTestService(store, []adapters.Capability{capability("manage_dns_filtering", true, true)}, fakeApplier{rollbackErr: errors.New("adapter rollback failed")})
	run, err := svc.CreateRun(context.Background(), "", false)
	require.NoError(t, err)
	run, err = svc.Approve(context.Background(), run.ID, run.Candidates[0].ID, true)
	require.NoError(t, err)

	run, err = svc.Rollback(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", run.Status)
	require.Equal(t, "rollback_failed", run.Candidates[0].Status)
	require.Contains(t, run.Candidates[0].Evidence, "adapter rollback failed")
}

func TestManualRequiredCandidateIsNotApplied(t *testing.T) {
	// [REQ:NM-P0-005] Candidates without automatic rollback become manual_required instead of being silently applied.
	store := newSnapshotStore(baselineSnapshot())
	cap := capability("manage_dns_filtering", true, false)
	cap.RequiresAdmin = true
	svc := newTestService(store, []adapters.Capability{cap}, fakeApplier{})
	run, err := svc.CreateRun(context.Background(), "", false)
	require.NoError(t, err)
	require.Equal(t, "manual_required", run.Candidates[0].Status)

	run, err = svc.Approve(context.Background(), run.ID, run.Candidates[0].ID, true)
	require.NoError(t, err)
	require.Equal(t, "manual_required", run.Status)
	require.Equal(t, "manual_required", run.Candidates[0].Status)
}

func newTestService(store *snapshotStore, caps []adapters.Capability, applier fakeApplier) *Service {
	if store == nil {
		store = newSnapshotStore()
	}
	return NewService(Config{
		Repo:         newFakeRepo(),
		Capabilities: fakeCapabilities{caps: caps},
		Snapshots:    store,
		Runner:       store,
		Applier:      applier,
		Now:          fixedNow,
	})
}

func capability(action string, supported, rollback bool) adapters.Capability {
	return adapters.Capability{Adapter: "fake-adapter", Action: action, Supported: supported, RollbackSupported: rollback, Reason: "test capability", ObservedAt: fixedNow()}
}

func baselineSnapshot() snapshot.Snapshot {
	return snapshot.Snapshot{
		ID:        "baseline-1",
		Status:    "baseline",
		Profile:   "home",
		Summary:   "2 healthy, 0 degraded, 0 unavailable, 0 failed probe results.",
		Metrics:   []snapshot.Metric{{Name: "dns_lookup_latency", Value: "12", Unit: "ms", Status: "healthy"}, {Name: "wan_reachability", Value: "ok", Unit: "state", Status: "healthy"}},
		Findings:  []string{"baseline ready"},
		CreatedAt: fixedNow(),
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 23, 18, 0, 0, 0, time.UTC)
}

type fakeCapabilities struct {
	caps []adapters.Capability
}

func (f fakeCapabilities) ListCapabilities(context.Context) ([]adapters.Capability, error) {
	return append([]adapters.Capability(nil), f.caps...), nil
}

type snapshotStore struct {
	items []snapshot.Snapshot
	next  int
}

func newSnapshotStore(items ...snapshot.Snapshot) *snapshotStore {
	return &snapshotStore{items: append([]snapshot.Snapshot(nil), items...)}
}

func (s *snapshotStore) List(context.Context) ([]snapshot.Snapshot, error) {
	out := append([]snapshot.Snapshot(nil), s.items...)
	return out, nil
}

func (s *snapshotStore) Run(context.Context, string, bool) (snapshot.Snapshot, error) {
	s.next++
	snap := snapshot.Snapshot{
		ID:        "snap-test-" + string(rune('0'+s.next)),
		Status:    "complete",
		Profile:   "optimization",
		Summary:   "2 healthy, 0 degraded, 0 unavailable, 0 failed probe results.",
		Metrics:   baselineSnapshot().Metrics,
		Findings:  []string{"candidate evidence captured"},
		CreatedAt: fixedNow().Add(time.Duration(s.next) * time.Minute),
	}
	s.items = append([]snapshot.Snapshot{snap}, s.items...)
	return snap, nil
}

type fakeApplier struct {
	applyErr    error
	rollbackErr error
}

func (f fakeApplier) Apply(context.Context, Run, Candidate) (ApplyResult, error) {
	if f.applyErr != nil {
		return ApplyResult{}, f.applyErr
	}
	return ApplyResult{Evidence: []string{"applied by fake adapter"}, RollbackHandle: "rollback://optimization/test"}, nil
}

func (f fakeApplier) Rollback(context.Context, Run, Candidate) (RollbackResult, error) {
	if f.rollbackErr != nil {
		return RollbackResult{}, f.rollbackErr
	}
	return RollbackResult{Evidence: []string{"rolled back by fake adapter"}}, nil
}

type fakeRepo struct {
	runs map[string]Run
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{runs: map[string]Run{}}
}

func (r *fakeRepo) SaveRun(_ context.Context, run Run) (Run, error) {
	run.Candidates = nil
	r.runs[run.ID] = cloneRun(run)
	return run, nil
}

func (r *fakeRepo) GetRun(_ context.Context, id string) (Run, error) {
	run, ok := r.runs[id]
	if !ok {
		return Run{}, ErrNotFound
	}
	return cloneRun(run), nil
}

func (r *fakeRepo) UpdateRun(_ context.Context, run Run) (Run, error) {
	current, ok := r.runs[run.ID]
	if !ok {
		return Run{}, ErrNotFound
	}
	run.Candidates = current.Candidates
	r.runs[run.ID] = cloneRun(run)
	return run, nil
}

func (r *fakeRepo) SaveCandidate(_ context.Context, c Candidate) (Candidate, error) {
	run, ok := r.runs[c.RunID]
	if !ok {
		return Candidate{}, ErrNotFound
	}
	run.Candidates = append(run.Candidates, c)
	r.runs[c.RunID] = cloneRun(run)
	return c, nil
}

func (r *fakeRepo) UpdateCandidate(_ context.Context, c Candidate) (Candidate, error) {
	run, ok := r.runs[c.RunID]
	if !ok {
		return Candidate{}, ErrNotFound
	}
	for i := range run.Candidates {
		if run.Candidates[i].ID == c.ID {
			run.Candidates[i] = c
			r.runs[c.RunID] = cloneRun(run)
			return c, nil
		}
	}
	return Candidate{}, ErrCandidateNotFound
}

func (r *fakeRepo) SaveApproval(_ context.Context, approval ApprovalRecord) (ApprovalRecord, error) {
	return approval, nil
}

func (r *fakeRepo) SaveRollback(_ context.Context, rollback RollbackRecord) (RollbackRecord, error) {
	return rollback, nil
}

func cloneRun(run Run) Run {
	run.Candidates = append([]Candidate(nil), run.Candidates...)
	return run
}

var _ Repository = (*fakeRepo)(nil)
