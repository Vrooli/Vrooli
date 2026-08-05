package journal

import "context"

// Repository deliberately exposes append and reads only. The journal has no
// update or delete operation because entries are permanent evidence.
type Repository interface {
	Append(context.Context, Entry, []string) (Entry, error)
	Get(context.Context, string) (Entry, error)
	List(context.Context, int) ([]Entry, error)
	ListByRun(context.Context, string, int) ([]Entry, error)
	FindByImportKey(context.Context, string) (Entry, bool, error)
	// RepairImportProvenance fills only missing provenance on an already
	// content-addressed imported entry. It never changes memory content.
	RepairImportProvenance(context.Context, string, ImportProvenance) error
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
