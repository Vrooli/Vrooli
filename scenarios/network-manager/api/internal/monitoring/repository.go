package monitoring

import "context"

type Repository interface {
	ListSchedules(ctx context.Context, includeDisabled bool) ([]Schedule, error)
	GetSchedule(ctx context.Context, id string) (Schedule, error)
	UpsertSchedule(ctx context.Context, schedule Schedule) (Schedule, error)
	SaveRun(ctx context.Context, run Run) (Run, error)
	SaveAlert(ctx context.Context, alert Alert) (Alert, error)
	ListAlerts(ctx context.Context, scheduleID string, openOnly bool) ([]Alert, error)
}
