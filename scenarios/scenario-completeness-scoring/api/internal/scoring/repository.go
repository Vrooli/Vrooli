package scoring

import (
	"context"
	"time"
)

// Snapshot is one digest-deduplicated persisted score observation.
type Snapshot struct {
	ID             int64
	Scenario       string
	Category       string
	Digest         string
	Composite      int
	Classification string
	WorkingRung    string
	BreakdownJSON  string
	Importance     *float64
	Source         string
	CreatedAt      time.Time

	// LastRunAt / LastStatus are scenario-level test recency carried alongside
	// the score: the newest test run in the scenario's run index and its
	// overall status. Zero/empty when no run is recorded or the row predates
	// recency capture. Unlike the score (digest-deduplicated), recency advances
	// monotonically even when the score digest is unchanged (see UpsertSnapshot).
	LastRunAt  time.Time
	LastStatus string
}

// TrendQuery selects a scenario's score history, newest first.
type TrendQuery struct {
	Scenario string
	Limit    int
	Since    time.Time
}

// SortBy selects the ListPage ordering.
type SortBy string

const (
	SortByComposite  SortBy = "composite"
	SortByRung       SortBy = "rung"
	SortByLastScored SortBy = "last_scored"
	SortByScenario   SortBy = "scenario"
	SortByPriority   SortBy = "priority"
)

// SortOrder selects ascending or descending ordering.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ListQuery describes the fleet bulk-view read. Offset is the decoded
// page-token cursor; callers own the token envelope.
type ListQuery struct {
	SortBy   SortBy
	Order    SortOrder
	Limit    int
	Offset   int
	MinScore *int
	MaxScore *int
	Rung     string
	Category string
}

// ListResult is one page of latest-per-scenario score rows.
type ListResult struct {
	Snapshots  []Snapshot
	NextOffset int
	HasNext    bool
}

// MeasureWindow scopes analytical aggregates over persisted snapshots.
type MeasureWindow struct {
	From time.Time
	To   time.Time
}

// ScoreSeriesPoint is one fleet-average score point for a time bucket.
type ScoreSeriesPoint struct {
	Bucket time.Time
	Score  float64
	Count  int
}

// seam: SnapshotRepository persists score_snapshots. Production wires
// SQLiteSnapshotRepository from sqlite_repository.go; tests use the same
// repository against a temp SQLite database.
type SnapshotRepository interface {
	LatestFor(ctx context.Context, scenario string) (Snapshot, bool, error)
	LatestDifferingDigest(ctx context.Context, scenario, digest string) (Snapshot, bool, error)
	SeriesFor(ctx context.Context, q TrendQuery) ([]Snapshot, error)
	UpsertSnapshot(ctx context.Context, snap Snapshot) (bool, error)
	ListPage(ctx context.Context, q ListQuery) (ListResult, error)
}
