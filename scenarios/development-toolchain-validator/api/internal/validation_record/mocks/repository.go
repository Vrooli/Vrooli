// Package mocks provides in-memory fakes for the validation_record
// seams. Production code never imports this package — the
// no_prod_import_test drift gate enforces that.
package mocks

import (
	"context"
	"sort"
	"sync"

	vr "development-toolchain-validator/internal/validation_record"
)

// FakeRepository is an in-memory validation_record.Repository for
// service-level tests.
type FakeRepository struct {
	mu      sync.Mutex
	records map[string]vr.Record
}

var _ vr.Repository = (*FakeRepository)(nil)

// NewFakeRepository constructs an empty FakeRepository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{records: map[string]vr.Record{}}
}

func (f *FakeRepository) Append(_ context.Context, r vr.Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records[r.ID] = r
	return nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (vr.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.records[id]
	if !ok {
		return vr.Record{}, vr.ErrRecordNotFound{ID: id}
	}
	return r, nil
}

func (f *FakeRepository) List(_ context.Context, fl vr.ListFilter, pageSize int, _ string) (vr.ListResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if pageSize <= 0 {
		pageSize = 50
	}
	var out []vr.Record
	for _, r := range f.records {
		if fl.GoldenSlug != "" && r.GoldenSlug != fl.GoldenSlug {
			continue
		}
		if fl.SubjectID != "" && r.SubjectID != fl.SubjectID {
			continue
		}
		if fl.TupleKind != vr.TupleKindUnspecified && r.TupleKind != fl.TupleKind {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].EndedAt.Equal(out[j].EndedAt) {
			return out[i].EndedAt.After(out[j].EndedAt)
		}
		return out[i].ID > out[j].ID
	})
	if len(out) > pageSize {
		out = out[:pageSize]
	}
	return vr.ListResult{Records: out}, nil
}
