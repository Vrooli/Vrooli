package alerts

import (
	"context"
	"time"
)

// Alert is one quality threshold breach.
type Alert struct {
	ID             string
	Level          string // "info" | "warning" | "critical"
	CollectionName string
	MetricName     string
	ThresholdValue *float64
	ActualValue    *float64
	Message        string
	Acknowledged   bool
	AcknowledgedAt *time.Time
	CreatedAt      time.Time
}

// Repository is the alerts domain's storage surface.
//
// No handler calls it yet — the tables predate any feature that writes them
// (see docs/internal/STORAGE_AUDIT.md §2, "Dead tables"). It exists so the
// domain is complete and its DDL is exercised; deleting the feature means
// deleting this folder.
type Repository interface {
	Insert(ctx context.Context, a Alert) (string, error)
	Get(ctx context.Context, id string) (Alert, bool, error)
	ListUnacknowledged(ctx context.Context, limit int) ([]Alert, error)
	Acknowledge(ctx context.Context, id string, at time.Time) error
}
