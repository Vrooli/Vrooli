package drills

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, d Drill) (Drill, error)
	MarkRunning(ctx context.Context, id, restoreID string, startedAt time.Time) error
	Finish(ctx context.Context, id string, status Status, errMsg, nextAction string, finishedAt time.Time) error
	Get(ctx context.Context, id string) (Drill, error)
	List(ctx context.Context, planID, targetID string, limit int) ([]Drill, error)
	FindByIdempotency(ctx context.Context, key string) (Drill, bool, error)
	LatestForUnit(ctx context.Context, planID, targetID, destinationID string) (Drill, bool, error)
}

type PlanLookup interface {
	PlanForDrill(ctx context.Context, planID string) (Plan, error)
	SchedulableDrillPlans(ctx context.Context) ([]Plan, error)
}

type SnapshotLookup interface {
	LatestSuccessfulSnapshot(ctx context.Context, planID, targetID, destinationID string) (Snapshot, bool, error)
}

type RestoreRunner interface {
	VerifyTarget(ctx context.Context, targetID, destinationID, snapshotID string) (Restore, error)
	GetRestore(ctx context.Context, restoreID string) (Restore, error)
}

type Clock interface{ Now() time.Time }
