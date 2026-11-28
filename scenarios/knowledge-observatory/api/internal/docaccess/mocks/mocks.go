// Package mocks holds an in-memory docaccess.Repository for testing callers.
package mocks

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"knowledge-observatory/internal/docaccess"
)

// Repository is an in-memory docaccess.Repository.
type Repository struct {
	mu      sync.Mutex
	Entries []docaccess.Access

	// Err, when set, is returned by every method.
	Err error
}

var _ docaccess.Repository = (*Repository)(nil)

// New returns an empty repository.
func New() *Repository { return &Repository{} }

func (r *Repository) LogAccess(_ context.Context, a docaccess.Access) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	if a.ID == "" {
		a.ID = fmt.Sprintf("access-%d", len(r.Entries)+1)
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	r.Entries = append(r.Entries, a)
	return nil
}

func (r *Repository) QueryStats(_ context.Context, filter docaccess.Filter) ([]docaccess.Stat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return nil, r.Err
	}

	type key struct{ scenario, docType string }
	tally := map[key]*docaccess.Stat{}
	for _, e := range r.Entries {
		if filter.ScenarioName != "" && e.ScenarioName != filter.ScenarioName {
			continue
		}
		if filter.DocType != "" && e.DocType != filter.DocType {
			continue
		}
		k := key{e.ScenarioName, e.DocType}
		if tally[k] == nil {
			tally[k] = &docaccess.Stat{ScenarioName: e.ScenarioName, DocType: e.DocType}
		}
		switch e.Operation {
		case "read":
			tally[k].ReadCount++
		case "write":
			tally[k].WriteCount++
		case "reset":
			tally[k].ResetCount++
		}
	}

	out := make([]docaccess.Stat, 0, len(tally))
	for _, stat := range tally {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScenarioName != out[j].ScenarioName {
			return out[i].ScenarioName < out[j].ScenarioName
		}
		return out[i].DocType < out[j].DocType
	})
	return out, nil
}

func (r *Repository) Recent(_ context.Context, limit int) ([]docaccess.Access, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return nil, r.Err
	}
	if limit <= 0 {
		limit = 100
	}
	out := make([]docaccess.Access, 0, limit)
	for i := len(r.Entries) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.Entries[i])
	}
	return out, nil
}
