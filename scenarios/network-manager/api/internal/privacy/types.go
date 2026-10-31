package privacy

import (
	"context"
	"errors"
	"time"
)

const (
	TimeFormat              = time.RFC3339Nano
	SettingsID              = "default"
	ProfileHomeMinimal      = "home-minimal"
	ProfileHomeExtended     = "home-extended"
	ProfileSmallOfficeAudit = "small-office-audit"
)

var ErrInvalidSettings = errors.New("invalid privacy settings")

type RetentionSettings struct {
	QueryLogDays   int32
	SnapshotDays   int32
	ExperimentDays int32
	Profile        string
	UpdatedAt      time.Time
}

type VisibilitySettings struct {
	ShowQueryDomains  bool
	ShowDeviceHistory bool
	HouseholdMode     bool
	Notes             []string
	UpdatedAt         time.Time
}

type SweepResult struct {
	ID               string
	Profile          string
	SnapshotCutoff   time.Time
	SnapshotsDeleted int
	Notes            []string
	CreatedAt        time.Time
}

type Repository interface {
	GetRetention(ctx context.Context) (RetentionSettings, error)
	SaveRetention(ctx context.Context, settings RetentionSettings) (RetentionSettings, error)
	GetVisibility(ctx context.Context) (VisibilitySettings, error)
	SaveVisibility(ctx context.Context, settings VisibilitySettings) (VisibilitySettings, error)
	Sweep(ctx context.Context, settings RetentionSettings, now time.Time) (SweepResult, error)
}
