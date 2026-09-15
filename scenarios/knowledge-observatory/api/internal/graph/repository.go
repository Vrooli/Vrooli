package graph

import (
	"context"
	"time"
)

// Edge is one relationship between two knowledge entries.
type Edge struct {
	ID               string
	SourceID         string
	TargetID         string
	RelationshipType string
	Weight           float64
	DiscoveredAt     time.Time
}

// Repository is the graph domain's storage surface.
type Repository interface {
	UpsertEdges(ctx context.Context, edges []Edge) error
	ListEdges(ctx context.Context, limit int) ([]Edge, error)
	CountEdges(ctx context.Context) (int64, error)
}
