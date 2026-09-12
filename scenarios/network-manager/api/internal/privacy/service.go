package privacy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type Service struct {
	repo  Repository
	clock schedule.Clock
}

type Config struct {
	Repo  Repository
	Clock schedule.Clock
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, clock: cfg.Clock}
	if s.clock == nil {
		s.clock = schedule.System()
	}
	return s
}

func (s *Service) GetRetention(ctx context.Context) (RetentionSettings, error) {
	settings, err := s.repo.GetRetention(ctx)
	if err != nil {
		return RetentionSettings{}, err
	}
	return withRetentionDefaults(settings, s.clock.Now().UTC()), nil
}

func (s *Service) UpdateRetention(ctx context.Context, settings RetentionSettings) (RetentionSettings, error) {
	settings = withRetentionDefaults(settings, s.clock.Now().UTC())
	if err := validateRetention(settings); err != nil {
		return RetentionSettings{}, err
	}
	return s.repo.SaveRetention(ctx, settings)
}

func (s *Service) GetVisibility(ctx context.Context) (VisibilitySettings, error) {
	settings, err := s.repo.GetVisibility(ctx)
	if err != nil {
		return VisibilitySettings{}, err
	}
	return withVisibilityDefaults(settings, s.clock.Now().UTC()), nil
}

func (s *Service) Sweep(ctx context.Context) (SweepResult, error) {
	settings, err := s.GetRetention(ctx)
	if err != nil {
		return SweepResult{}, err
	}
	return s.repo.Sweep(ctx, settings, s.clock.Now().UTC())
}

func DefaultRetention(now time.Time) RetentionSettings {
	return RetentionSettings{
		QueryLogDays:   0,
		SnapshotDays:   30,
		ExperimentDays: 30,
		Profile:        ProfileHomeMinimal,
		UpdatedAt:      now.UTC(),
	}
}

func DefaultVisibility(now time.Time) VisibilitySettings {
	return VisibilitySettings{
		ShowQueryDomains:  false,
		ShowDeviceHistory: false,
		HouseholdMode:     true,
		Notes: []string{
			"DNS query-level visibility is disabled by default.",
			"Device history visibility is hidden until the operator explicitly changes privacy settings.",
		},
		UpdatedAt: now.UTC(),
	}
}

func withRetentionDefaults(settings RetentionSettings, now time.Time) RetentionSettings {
	defaults := DefaultRetention(now)
	if settings.Profile == "" {
		settings.Profile = defaults.Profile
	}
	if settings.SnapshotDays == 0 {
		settings.SnapshotDays = defaults.SnapshotDays
	}
	if settings.ExperimentDays == 0 {
		settings.ExperimentDays = defaults.ExperimentDays
	}
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = now.UTC()
	}
	return settings
}

func withVisibilityDefaults(settings VisibilitySettings, now time.Time) VisibilitySettings {
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = now.UTC()
	}
	if !settings.ShowQueryDomains && !settings.ShowDeviceHistory && !settings.HouseholdMode {
		settings.HouseholdMode = true
	}
	if len(settings.Notes) == 0 {
		settings.Notes = DefaultVisibility(now).Notes
	}
	return settings
}

func validateRetention(settings RetentionSettings) error {
	profile := strings.TrimSpace(settings.Profile)
	switch profile {
	case ProfileHomeMinimal, ProfileHomeExtended:
		if settings.QueryLogDays > 7 {
			return fmt.Errorf("%w: household query log retention must be 7 days or less", ErrInvalidSettings)
		}
	case ProfileSmallOfficeAudit:
		if settings.QueryLogDays > 90 {
			return fmt.Errorf("%w: small-office audit query log retention must be 90 days or less", ErrInvalidSettings)
		}
	default:
		return fmt.Errorf("%w: unsupported profile %q", ErrInvalidSettings, settings.Profile)
	}
	if settings.QueryLogDays < 0 || settings.SnapshotDays < 1 || settings.ExperimentDays < 1 {
		return fmt.Errorf("%w: retention days must be non-negative and snapshot/experiment retention must be at least one day", ErrInvalidSettings)
	}
	if settings.SnapshotDays > 3650 || settings.ExperimentDays > 3650 {
		return fmt.Errorf("%w: snapshot and experiment retention must be 3650 days or less", ErrInvalidSettings)
	}
	return nil
}
