package mocks

import (
	"context"
	"database/sql"

	"vrooli-memory/internal/journal"
)

// Repository is a controllable append-only repository double for journal tests.
type Repository struct {
	AppendOut    journal.Entry
	AppendErr    error
	GetOut       journal.Entry
	GetErr       error
	ListOut      []journal.Entry
	ListErr      error
	Appends      []journal.Entry
	Retries      [][]string
	RetryItems   []journal.RetryItem
	Acknowledged []string
	Pruned       int
	Reconciled   int
}

func (r *Repository) Append(_ context.Context, entry journal.Entry, retries []string) (journal.Entry, error) {
	r.Appends = append(r.Appends, entry)
	r.Retries = append(r.Retries, append([]string(nil), retries...))
	if r.AppendErr != nil {
		return journal.Entry{}, r.AppendErr
	}
	if r.AppendOut.ID != "" {
		return r.AppendOut, nil
	}
	return entry, nil
}

func (r *Repository) Get(context.Context, string) (journal.Entry, error) {
	if r.GetErr != nil {
		return journal.Entry{}, r.GetErr
	}
	if r.GetOut.ID == "" {
		return journal.Entry{}, sql.ErrNoRows
	}
	return r.GetOut, nil
}

func (r *Repository) List(context.Context, int) ([]journal.Entry, error) {
	if r.ListErr != nil {
		return nil, r.ListErr
	}
	return append([]journal.Entry(nil), r.ListOut...), nil
}

func (r *Repository) ListByRun(_ context.Context, runID string, _ int) ([]journal.Entry, error) {
	var out []journal.Entry
	for _, entry := range r.ListOut {
		if entry.Correlation.RunID == runID {
			out = append(out, entry)
		}
	}
	return out, r.ListErr
}

func (r *Repository) FindByImportKey(_ context.Context, key string) (journal.Entry, bool, error) {
	for _, entry := range r.Appends {
		if entry.ImportKey == key {
			entry.Existing = true
			return entry, true, nil
		}
	}
	return journal.Entry{}, false, nil
}

func (r *Repository) ClassificationRetries(context.Context, int) ([]journal.RetryItem, error) {
	return append([]journal.RetryItem(nil), r.RetryItems...), nil
}

func (r *Repository) AcknowledgeRetry(_ context.Context, id string) error {
	r.Acknowledged = append(r.Acknowledged, id)
	return nil
}

func (r *Repository) PruneResolvedClassificationRetries(context.Context) (int, error) {
	return r.Pruned, nil
}

func (r *Repository) EnqueueUnclassified(context.Context) (int, error) {
	r.Reconciled++
	return r.Reconciled, nil
}

func (r *Repository) EmbeddingRetries(context.Context, int) ([]journal.RetryItem, error) {
	return append([]journal.RetryItem(nil), r.RetryItems...), nil
}

func (r *Repository) StoreFacetEmbedding(context.Context, string, []float64) error { return nil }

func (r *Repository) AcknowledgeEmbeddingRetries(_ context.Context, id string) error {
	r.Acknowledged = append(r.Acknowledged, id)
	return nil
}

func (r *Repository) PruneResolvedEmbeddingRetries(context.Context) (int, error) {
	return r.Pruned, nil
}

var _ journal.Repository = (*Repository)(nil)
