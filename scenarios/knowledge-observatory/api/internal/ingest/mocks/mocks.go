// Package mocks holds an in-memory ingest.Repository for testing callers.
package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"knowledge-observatory/internal/ingest"
)

// Repository is an in-memory ingest.Repository.
type Repository struct {
	mu      sync.Mutex
	History []ingest.HistoryEntry
	Jobs    map[string]ingest.Job

	// Err, when set, is returned by every method.
	Err error
}

var _ ingest.Repository = (*Repository)(nil)

// New returns an empty repository.
func New() *Repository { return &Repository{Jobs: map[string]ingest.Job{}} }

func (r *Repository) InsertHistory(_ context.Context, h ingest.HistoryEntry) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return "", r.Err
	}
	if h.ID == "" {
		h.ID = fmt.Sprintf("ingest-%d", len(r.History)+1)
	}
	for _, existing := range r.History {
		if existing.ID == h.ID {
			return h.ID, nil // matches ON CONFLICT DO NOTHING
		}
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now().UTC()
	}
	r.History = append(r.History, h)
	return h.ID, nil
}

func (r *Repository) GetHistory(_ context.Context, id string) (ingest.HistoryEntry, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return ingest.HistoryEntry{}, false, r.Err
	}
	for _, h := range r.History {
		if h.ID == id {
			return h, true, nil
		}
	}
	return ingest.HistoryEntry{}, false, nil
}

func (r *Repository) ProvenanceForCollection(_ context.Context, collection string) (ingest.Provenance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return ingest.Provenance{}, r.Err
	}

	var out ingest.Provenance
	namespaces := map[string]bool{}
	for _, h := range r.History {
		if h.CollectionName != collection {
			continue
		}
		out.IngestAttempts++
		namespaces[h.Namespace] = true
		if out.LastIngestAt == nil || h.CreatedAt.After(*out.LastIngestAt) {
			at := h.CreatedAt
			out.LastIngestAt = &at
		}
	}
	out.DistinctNamespaces = len(namespaces)
	return out, nil
}

func (r *Repository) HealthForCollection(_ context.Context, collection string) (ingest.Health, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return ingest.Health{}, r.Err
	}

	var out ingest.Health
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	for _, h := range r.History {
		if h.CollectionName != collection {
			continue
		}
		out.TotalAttempts++
		switch h.Status {
		case "success":
			out.SuccessCount++
		case "failure":
			out.FailureCount++
			if h.CreatedAt.After(cutoff) {
				out.FailureCountLast24H++
			}
			if out.LastFailureAt == nil || h.CreatedAt.After(*out.LastFailureAt) {
				at := h.CreatedAt
				out.LastFailureAt = &at
			}
		}
	}
	return out, nil
}

func (r *Repository) DeleteHistoryByCollection(_ context.Context, collection string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	kept := r.History[:0]
	var deleted int64
	for _, h := range r.History {
		if h.CollectionName == collection {
			deleted++
			continue
		}
		kept = append(kept, h)
	}
	r.History = kept
	return deleted, nil
}

func (r *Repository) UpsertJob(_ context.Context, j ingest.Job) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return "", r.Err
	}
	if j.ID == "" {
		j.ID = fmt.Sprintf("job-%d", len(r.Jobs)+1)
	}
	if existing, ok := r.Jobs[j.ID]; ok {
		j.CreatedAt = existing.CreatedAt
		if j.StartedAt == nil {
			j.StartedAt = existing.StartedAt
		}
		if j.FinishedAt == nil {
			j.FinishedAt = existing.FinishedAt
		}
	} else {
		j.CreatedAt = time.Now().UTC()
	}
	if j.Status == "" {
		j.Status = "pending"
	}
	if j.RequestJSON == "" {
		j.RequestJSON = "{}"
	}
	r.Jobs[j.ID] = j
	return j.ID, nil
}

func (r *Repository) GetJob(_ context.Context, id string) (ingest.Job, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return ingest.Job{}, false, r.Err
	}
	j, ok := r.Jobs[id]
	return j, ok, nil
}
