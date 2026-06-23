package privacy

import (
	"context"
	"errors"
	"testing"
	"time"

	"network-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func TestServiceDefaultsAreMinimal(t *testing.T) {
	// [REQ:NM-P0-008] Sensitive network metadata starts with minimal
	// query visibility and short household retention.
	now := time.Date(2026, 6, 23, 20, 0, 0, 0, time.UTC)
	repo := &fakeRepository{}
	service := NewService(Config{Repo: repo, Clock: mocks.NewFakeClock(now)})

	retention, err := service.GetRetention(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(0), retention.QueryLogDays)
	require.Equal(t, int32(30), retention.SnapshotDays)
	require.Equal(t, ProfileHomeMinimal, retention.Profile)

	visibility, err := service.GetVisibility(context.Background())
	require.NoError(t, err)
	require.False(t, visibility.ShowQueryDomains)
	require.False(t, visibility.ShowDeviceHistory)
	require.True(t, visibility.HouseholdMode)
	require.Contains(t, visibility.Notes, "DNS query-level visibility is disabled by default.")
}

func TestServiceUpdateValidation(t *testing.T) {
	// [REQ:NM-P0-008] Household query-log retention must stay deliberately
	// short unless audit mode is explicitly selected.
	service := NewService(Config{Repo: &fakeRepository{}, Clock: mocks.NewFakeClock(time.Date(2026, 6, 23, 20, 0, 0, 0, time.UTC))})

	_, err := service.UpdateRetention(context.Background(), RetentionSettings{
		QueryLogDays:   14,
		SnapshotDays:   30,
		ExperimentDays: 30,
		Profile:        ProfileHomeMinimal,
	})
	require.ErrorIs(t, err, ErrInvalidSettings)

	updated, err := service.UpdateRetention(context.Background(), RetentionSettings{
		QueryLogDays:   30,
		SnapshotDays:   365,
		ExperimentDays: 365,
		Profile:        ProfileSmallOfficeAudit,
	})
	require.NoError(t, err)
	require.Equal(t, ProfileSmallOfficeAudit, updated.Profile)
	require.Equal(t, int32(30), updated.QueryLogDays)
}

func TestServiceSweepUsesRetentionSettings(t *testing.T) {
	// [REQ:NM-P0-008] Retention sweeps use the configured policy and clock,
	// making pruning deterministic in tests.
	now := time.Date(2026, 6, 23, 20, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		retention: RetentionSettings{
			QueryLogDays:   0,
			SnapshotDays:   7,
			ExperimentDays: 30,
			Profile:        ProfileHomeMinimal,
			UpdatedAt:      now,
		},
	}
	service := NewService(Config{Repo: repo, Clock: mocks.NewFakeClock(now)})

	result, err := service.Sweep(context.Background())
	require.NoError(t, err)
	require.Equal(t, now.AddDate(0, 0, -7), result.SnapshotCutoff)
	require.Equal(t, int32(7), repo.sweepSettings.SnapshotDays)
}

type fakeRepository struct {
	retention       RetentionSettings
	visibility      VisibilitySettings
	sweepSettings   RetentionSettings
	retentionSaved  bool
	visibilitySaved bool
}

func (r *fakeRepository) GetRetention(context.Context) (RetentionSettings, error) {
	return r.retention, nil
}

func (r *fakeRepository) SaveRetention(_ context.Context, settings RetentionSettings) (RetentionSettings, error) {
	if settings.Profile == "force-error" {
		return RetentionSettings{}, errors.New("forced save error")
	}
	r.retentionSaved = true
	r.retention = settings
	return settings, nil
}

func (r *fakeRepository) GetVisibility(context.Context) (VisibilitySettings, error) {
	return r.visibility, nil
}

func (r *fakeRepository) SaveVisibility(_ context.Context, settings VisibilitySettings) (VisibilitySettings, error) {
	r.visibilitySaved = true
	r.visibility = settings
	return settings, nil
}

func (r *fakeRepository) Sweep(_ context.Context, settings RetentionSettings, now time.Time) (SweepResult, error) {
	r.sweepSettings = settings
	return SweepResult{
		ID:             "sweep-fake",
		Profile:        settings.Profile,
		SnapshotCutoff: now.AddDate(0, 0, -int(settings.SnapshotDays)),
		CreatedAt:      now,
	}, nil
}
