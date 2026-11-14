package search

import (
	"context"
	"time"
)

// History is one recorded search.
type History struct {
	ID             string
	Query          string
	Collection     string
	ResultCount    int
	AvgScore       *float64
	ResponseTimeMS int64
	UserSession    string
	CreatedAt      time.Time
}

// Repository is the search domain's storage surface.
type Repository interface {
	InsertHistory(ctx context.Context, h History) (string, error)
	RecentHistory(ctx context.Context, limit int) ([]History, error)
	CountHistory(ctx context.Context) (int64, error)
}
