package journal

import (
	"context"
	"time"
)

// Repository deliberately exposes append and reads only. The journal has no
// update or delete operation because entries are permanent evidence.
type Repository interface {
	Append(context.Context, Entry, []string) (Entry, error)
	Get(context.Context, string) (Entry, error)
	List(context.Context, int) ([]Entry, error)
	ListRecent(context.Context, string, int) ([]Entry, error)
	ListAfter(context.Context, string, int) ([]Entry, error)
	ListByRun(context.Context, string, int) ([]Entry, error)
	CountInWindow(context.Context, time.Time, time.Time) (int64, error)
	FindByImportKey(context.Context, string) (Entry, bool, error)
	// ClassificationRetries returns queued classification work without exposing
	// any mutation surface for immutable journal entries.
	ClassificationRetries(context.Context, int) ([]RetryItem, error)
	AcknowledgeRetry(context.Context, string) error
	PruneResolvedClassificationRetries(context.Context) (int, error)
	// EnqueueUnclassified re-queues every entry still carrying the sentinel
	// facet, so a lost queue row cannot strand a memory outside the vocabulary.
	EnqueueUnclassified(context.Context) (int, error)
	EmbeddingRetries(context.Context, int) ([]RetryItem, error)
	StoreFacetEmbedding(context.Context, string, []float64) error
	AcknowledgeEmbeddingRetries(context.Context, string) error
	PruneResolvedEmbeddingRetries(context.Context) (int, error)
}
