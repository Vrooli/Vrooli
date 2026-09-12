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

func TestHouseholdPolicyProfilePersistsIntentWithoutApplyingLivePolicy(t *testing.T) {
	// [REQ:NM-P1-001] Household profiles group devices by intent while keeping live changes approval-gated.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{}, Now: fixedNow})

	profile, err := svc.UpsertProfile(context.Background(), Profile{
		Name:              "Kids",
		DeviceGroup:       "kids",
		FilteringStrength: "strict",
		Schedule:          "daily:20:00-07:00",
		OverrideBehavior:  "parent_override",
	})
	require.NoError(t, err)
	require.Equal(t, "kids", profile.DeviceGroup)
	require.Equal(t, "strict", profile.FilteringStrength)
	require.Equal(t, "enabled", profile.Status)
	require.Contains(t, profile.Effects, "Persistent resolver changes still require preview and approval.")

	profiles, err := svc.ListProfiles(context.Background(), "kids")
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	require.Equal(t, profile.ID, profiles[0].ID)
}

func TestPolicyScheduleEvaluationUsesCurrentWindowAndManualStatus(t *testing.T) {
	// [REQ:NM-P1-002] Scheduled access controls expose active windows without applying persistent DNS changes.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{}, Now: fixedNow})
	profile, err := svc.UpsertProfile(context.Background(), Profile{
		Name:              "Focus",
		DeviceGroup:       "work",
		FilteringStrength: "standard",
		Schedule:          "daily:09:00-17:00",
	})
	require.NoError(t, err)

	active, err := svc.EvaluateSchedule(context.Background(), profile.ID, "group:work", time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, active.Active)
	require.Equal(t, "active", active.Status)
	require.Equal(t, "group:work", active.Target)
	require.Equal(t, time.Date(2026, 6, 23, 17, 0, 0, 0, time.UTC), active.NextChangeAt)
	require.Contains(t, active.Effects, "Schedule evaluation is advisory until an approved policy apply is run.")

	inactive, err := svc.EvaluateSchedule(context.Background(), profile.ID, "group:work", time.Date(2026, 6, 23, 18, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.False(t, inactive.Active)
	require.Equal(t, "inactive", inactive.Status)
	require.Equal(t, time.Date(2026, 6, 24, 9, 0, 0, 0, time.UTC), inactive.NextChangeAt)
}

func TestPolicyScheduleEvaluationHandlesOvernightWindows(t *testing.T) {
	// [REQ:NM-P1-002] Bedtime-style windows can cross midnight without becoming inactive before morning.
	repo := newFakeRepo()
	svc := NewService(Config{Repo: repo, Adapter: fakeAdapter{}, Now: fixedNow})
	profile, err := svc.UpsertProfile(context.Background(), Profile{
		Name:              "Kids bedtime",
		DeviceGroup:       "kids",
		FilteringStrength: "strict",
		Schedule:          "daily:20:00-07:00",
	})
	require.NoError(t, err)

	evaluation, err := svc.EvaluateSchedule(context.Background(), profile.ID, "group:kids", time.Date(2026, 6, 24, 6, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, evaluation.Active)
	require.Equal(t, time.Date(2026, 6, 24, 7, 0, 0, 0, time.UTC), evaluation.NextChangeAt)
}

func TestEncryptedDNSBypassGuidanceIsManualUnlessAdapterBacked(t *testing.T) {
	// [REQ:NM-P1-004] IPv6 and encrypted-DNS bypass checks generate manual guidance without fake router enforcement.
	svc := NewService(Config{Repo: newFakeRepo(), Adapter: fakeAdapter{}, Now: fixedNow})

	report := svc.DiagnoseEncryptedDNSBypass(context.Background(), "network", false)
	require.Equal(t, "ipv6-encrypted-dns", report.Profile)
	require.Equal(t, "manual_required", report.Status)
	require.Len(t, report.Checks, 3)
	require.Contains(t, report.AdapterActions, "No adapter-backed bypass action is currently available for this target.")
	require.Contains(t, report.Guardrails, "Do not inspect or log user browsing contents to detect bypasses.")

	adapterBacked := svc.DiagnoseEncryptedDNSBypass(context.Background(), "group:kids", true)
	require.Equal(t, "group:kids", adapterBacked.Target)
	require.Contains(t, adapterBacked.AdapterActions[0], "Preview adapter rule")
}

func TestEndpointDoHGuidanceAvoidsInvasiveInspection(t *testing.T) {
	// [REQ:NM-P1-008] Browser and endpoint DoH guidance avoids TLS interception and hidden traffic inspection.
	svc := NewService(Config{Repo: newFakeRepo(), Adapter: fakeAdapter{}, Now: fixedNow})

	report := svc.EndpointDoHGuidance(context.Background(), "Windows", "Chrome", "group policy")
	require.Equal(t, "endpoint-doh", report.Profile)
	require.Equal(t, "windows/chrome", report.Target)
	require.Equal(t, "guidance_only", report.Status)
	require.Contains(t, report.Guardrails, "No TLS interception.")
	require.Contains(t, report.Guardrails, "No hidden endpoint monitoring.")
	require.Contains(t, report.Checks[0].Evidence, "group-policy")
	require.Contains(t, report.Checks[1].Recommendations[0], "DNS-over-HTTPS")
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 23, 17, 30, 0, 0, time.UTC)
}

type fakeRepo struct {
	changes  map[string]Change
	profiles map[string]Profile
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{changes: map[string]Change{}, profiles: map[string]Profile{}}
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

func (r *fakeRepo) ListProfiles(_ context.Context, deviceGroup string) ([]Profile, error) {
	profiles := []Profile{}
	for _, profile := range r.profiles {
		if deviceGroup == "" || profile.DeviceGroup == deviceGroup {
			profiles = append(profiles, cloneProfile(profile))
		}
	}
	return profiles, nil
}

func (r *fakeRepo) UpsertProfile(_ context.Context, profile Profile) (Profile, error) {
	r.profiles[profile.ID] = cloneProfile(profile)
	return profile, nil
}

func (r *fakeRepo) GetProfile(_ context.Context, id string) (Profile, error) {
	profile, ok := r.profiles[id]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return cloneProfile(profile), nil
}

func cloneChange(change Change) Change {
	change.Values = append([]string(nil), change.Values...)
	change.Effects = append([]string(nil), change.Effects...)
	return change
}

func cloneProfile(profile Profile) Profile {
	profile.Effects = append([]string(nil), profile.Effects...)
	return profile
}
