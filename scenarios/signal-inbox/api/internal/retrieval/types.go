// Package retrieval owns rebuildable corpus projections. Its filters only
// select results; neither category nor disposition changes what is indexed.
package retrieval

import (
	"context"
	"time"

	"signal-inbox/internal/signals"
)

type Filter struct {
	Text                          string
	CategoryID                    string
	Disposition                   string
	SourceKind                    signals.SourceKind
	CapturedAfter, CapturedBefore *time.Time
	Tags                          []string
	PageAfter                     string
	PageAfterCapturedAt           *time.Time
	PageAfterSignalID             string
	Limit                         int
}

type Result struct {
	Signal      signals.Signal
	CategoryID  string
	Disposition string
	Score       float64
}

type Page struct {
	Results       []Result
	NextPageAfter string
}

// SemanticSearch is the narrow, scenario-owned boundary for the derived
// vector projection. Its corpus is always the complete journal; filters are
// applied only when presenting results.
type SemanticSearch interface {
	Search(context.Context, string, []Result, int) ([]SemanticHit, error)
}

type SemanticHit struct {
	SignalID string
	Score    float64
}

type Repository interface {
	Rebuild(context.Context) error
	Search(context.Context, Filter) ([]Result, error)
	Ambient(context.Context, string, int, time.Time) ([]Result, error)
	IndexedCount(context.Context) (int, error)
	JournalCount(context.Context) (int, error)
}
