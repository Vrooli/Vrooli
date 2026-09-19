// Package enrichment appends extraction evidence after capture without ever
// mutating the signal journal.
package enrichment

import (
	"context"
	"fmt"
	"time"

	"signal-inbox/internal/signals"
)

type ExtractionResult struct {
	Content      string
	ContentUnits int
}

type Extractor interface {
	Supports(signals.SourceKind) bool
	Extract(context.Context, signals.Signal) (ExtractionResult, error)
}

type Record struct {
	ID               string
	SignalID         string
	ExtractedContent string
	ContentUnits     int
	NeedsAttention   bool
	AttentionReason  string
	AttemptedAt      time.Time
}

type Repository interface {
	Append(context.Context, Record) error
	Latest(context.Context, string) (Record, bool, error)
}

type ErrInvalidRecord struct{ Reason string }

func (e ErrInvalidRecord) Error() string {
	return fmt.Sprintf("invalid enrichment record: %s", e.Reason)
}
