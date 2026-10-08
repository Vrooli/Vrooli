package catalog

import (
	"context"
	"time"
)

type Repository interface {
	ListTemplates(ctx context.Context, kind TemplateKind) ([]TemplateRecord, error)
	GetTemplate(ctx context.Context, id string) (TemplateRecord, error)
	SyncScenarioTemplates(ctx context.Context, templates []ScenarioTemplate) error
	SaveValidationRun(ctx context.Context, run ValidationRun) error
	ListValidationRuns(ctx context.Context, templateID string) ([]ValidationRun, error)
	GetValidationRun(ctx context.Context, id string) (ValidationRun, error)
	SaveDriftSnapshot(ctx context.Context, snapshot DriftSnapshot) error
	SupersedePendingDriftSnapshots(ctx context.Context, supersededAt time.Time) error
	ListDriftSnapshots(ctx context.Context, templateID string) ([]DriftSnapshot, error)
	UpsertDebt(ctx context.Context, entry DebtEntry) error
	ResolveSourceDebt(ctx context.Context, templateID string, resolvedAt time.Time) error
	ResolveSupersededDeepValidationDebt(ctx context.Context, templateID string, resolvedAt time.Time) error
	ListDebt(ctx context.Context, templateID, status string) ([]DebtEntry, error)
	GetDebt(ctx context.Context, key string) (DebtEntry, error)
}

type MeasuresRepository interface {
	CountOpenDebt(ctx context.Context, window MeasureWindow) (int64, error)
	DeepValidateGreenStreak(ctx context.Context, templateID string) (int64, error)
	FleetStandingDistribution(ctx context.Context) ([]StandingBucket, error)
	MaxVersionLag(ctx context.Context) (int64, error)
}
