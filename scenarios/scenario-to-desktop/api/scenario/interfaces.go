// Package scenario provides scenario discovery and desktop status endpoints.
package scenario

import (
	"context"
	"time"
)

// Service discovers scenarios and their desktop deployment status.
type Service interface {
	// ListScenarios returns all scenarios with their desktop status.
	ListScenarios(ctx context.Context) ([]ScenarioDesktopStatus, error)

	// GetStats returns aggregate statistics about scenarios.
	GetStats(ctx context.Context) (*ScenarioStats, error)
}

// RecordStore provides access to desktop app records.
type RecordStore interface {
	// List returns all records.
	List() []*DesktopAppRecord
}

// DesktopAppRecord is a simplified record type for scenario listing.
type DesktopAppRecord struct {
	ID              string
	ScenarioName    string
	OutputPath      string
	StagingPath     string
	CustomPath      string
	DestinationPath string
	LocationMode    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
