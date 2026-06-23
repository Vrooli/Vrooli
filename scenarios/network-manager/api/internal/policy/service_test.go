package policy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeAdapter struct {
	previewPlan AdapterPlan
	applyResult AdapterApplyResult
	applyErr    error
	rollback    AdapterRollbackResult
	rollbackErr error
}

func (f fakeAdapter) Preview(context.Context, Change) (AdapterPlan, error) {
	if len(f.previewPlan.Effects) == 0 {
		return AdapterPlan{Effects: []string{"preview ok"}, RollbackSupported: true}, nil
	}
	return f.previewPlan, nil
}

func (f fakeAdapter) Apply(context.Context, Change) (AdapterApplyResult, error) {
	if f.applyErr != nil {
		return AdapterApplyResult{}, f.applyErr
	}
	if len(f.applyResult.Effects) == 0 {
		return AdapterApplyResult{Effects: []string{"applied"}, RollbackSupported: true, RollbackHandle: "rollback://policy/1"}, nil
	}
	return f.applyResult, nil
}

func (f fakeAdapter) Rollback(context.Context, Change) (AdapterRollbackResult, error) {
	if f.rollbackErr != nil {
		return AdapterRollbackResult{}, f.rollbackErr
	}
	if len(f.rollback.Effects) == 0 {
		return AdapterRollbackResult{Effects: []string{"rolled back"}}, nil
	}
	return f.rollback, nil
}

func TestPreviewPersistsPolicyChange(t *testing.T) {
	// [REQ:NM-P0-003] Policy changes are previewed and persisted before any apply.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{}, Now: fixedNow})

	change, err := svc.Preview(context.Background(), "device:phone", "blocklist", []string{"ads.example", "track.example"})
	require.NoError(t, err)
	require.Equal(t, "previewed", change.Status)
	require.True(t, change.RollbackSupported)
	require.Equal(t, []string{"ads.example", "track.example"}, change.Values)

	stored, err := repo.GetChange(context.Background(), change.ID)
	require.NoError(t, err)
	require.Equal(t, "blocklist", stored.Action)
}

func TestApplyRequiresApproval(t *testing.T) {
	// [REQ:NM-P0-003] Persistent DNS policy writes require explicit approval.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{}, Now: fixedNow})
	preview, err := svc.Preview(context.Background(), "network", "denylist", []string{"bad.example"})
	require.NoError(t, err)

	change, err := svc.Apply(context.Background(), preview.ID, false)
	require.NoError(t, err)
	require.Equal(t, "approval_required", change.Status)
	require.Contains(t, change.Effects, "Persistent policy changes require --approved acknowledgement.")
}

func TestApplyUnsupportedAdapterFailsClosedWithoutFakeSuccess(t *testing.T) {
	// [REQ:NM-P0-003] Unsupported resolver adapters return unsupported instead of claiming live policy was changed.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{applyErr: ErrUnsupported}, Now: fixedNow})
	preview, err := svc.Preview(context.Background(), "network", "allowlist", []string{"school.example"})
	require.NoError(t, err)

	change, err := svc.Apply(context.Background(), preview.ID, true)
	require.NoError(t, err)
	require.Equal(t, "unsupported", change.Status)
	require.False(t, change.RollbackSupported)
}

func TestApplyAndRollbackWithCapableAdapter(t *testing.T) {
	// [REQ:NM-P0-003] Capable adapters preserve rollback handles and can roll back applied policy changes.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{}, Now: fixedNow})
	preview, err := svc.Preview(context.Background(), "device:laptop", "pause_filtering", []string{"duration=15m"})
	require.NoError(t, err)

	applied, err := svc.Apply(context.Background(), preview.ID, true)
	require.NoError(t, err)
	require.Equal(t, "applied", applied.Status)
	require.True(t, applied.RollbackSupported)
	require.NotEmpty(t, applied.RollbackHandle)

	rolledBack, err := svc.Rollback(context.Background(), preview.ID)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", rolledBack.Status)
	require.Contains(t, rolledBack.Effects, "rolled back")
}

func TestRollbackFailureIsRecordedAsTerminalState(t *testing.T) {
	// [REQ:NM-P0-003] Rollback failures do not disappear; the change moves to rollback_failed.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{rollbackErr: errors.New("adapter rollback failed")}, Now: fixedNow})
	preview, err := svc.Preview(context.Background(), "network", "blocklist", []string{"bad.example"})
	require.NoError(t, err)
	_, err = svc.Apply(context.Background(), preview.ID, true)
	require.NoError(t, err)

	change, err := svc.Rollback(context.Background(), preview.ID)
	require.NoError(t, err)
	require.Equal(t, "rollback_failed", change.Status)
	require.Contains(t, change.Effects, "adapter rollback failed")
}

func TestPauseAndResumeCreatePreviews(t *testing.T) {
	// [REQ:NM-P0-003] Pause/resume controls are preview-first and require a later approved apply.
	svc := NewService(Config{Repo: newFakeRepo(), Adapter: fakeAdapter{}, Now: fixedNow})

	pause, err := svc.Pause(context.Background(), "device:tablet", "30m")
	require.NoError(t, err)
	require.Equal(t, "previewed", pause.Status)
	require.Equal(t, "pause_filtering", pause.Action)
	require.Contains(t, pause.Values, "duration=30m")

	resume, err := svc.Resume(context.Background(), "device:tablet")
	require.NoError(t, err)
	require.Equal(t, "previewed", resume.Status)
	require.Equal(t, "resume_filtering", resume.Action)
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 23, 17, 30, 0, 0, time.UTC)
}

type fakeRepo struct {
	changes map[string]Change
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{changes: map[string]Change{}}
}

func (r *fakeRepo) SaveChange(_ context.Context, change Change) (Change, error) {
	r.changes[change.ID] = cloneChange(change)
	return change, nil
}

func (r *fakeRepo) GetChange(_ context.Context, id string) (Change, error) {
	change, ok := r.changes[id]
	if !ok {
		return Change{}, ErrNotFound
	}
	return cloneChange(change), nil
}

func (r *fakeRepo) UpdateChange(_ context.Context, change Change) (Change, error) {
	if _, ok := r.changes[change.ID]; !ok {
		return Change{}, ErrNotFound
	}
	r.changes[change.ID] = cloneChange(change)
	return change, nil
}

func (r *fakeRepo) SaveApproval(_ context.Context, approval ApprovalRecord) (ApprovalRecord, error) {
	return approval, nil
}

func (r *fakeRepo) SaveRollback(_ context.Context, rollback RollbackRecord) (RollbackRecord, error) {
	return rollback, nil
}

func cloneChange(change Change) Change {
	change.Values = append([]string(nil), change.Values...)
	change.Effects = append([]string(nil), change.Effects...)
	return change
}
