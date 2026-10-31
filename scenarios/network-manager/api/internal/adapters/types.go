package adapters

import (
	"context"
	"errors"
	"time"
)

const TimeFormat = time.RFC3339Nano

var ErrNotFound = errors.New("adapter capability not found")

type Capability struct {
	Adapter           string
	Action            string
	Supported         bool
	RequiresAdmin     bool
	RollbackSupported bool
	Reason            string
	ObservedAt        time.Time
}

type PlatformSummary struct {
	OS         string
	Arch       string
	Profile    string
	Notes      []string
	ObservedAt time.Time
}

type Report struct {
	Capabilities []Capability
	Platform     PlatformSummary
	ObservedAt   time.Time
}

type Registry interface {
	Report(ctx context.Context) (Report, error)
}

type Repository interface {
	SaveReport(ctx context.Context, report Report) error
	LatestCapabilities(ctx context.Context) ([]Capability, error)
	LatestPlatformSummary(ctx context.Context) (PlatformSummary, error)
}
