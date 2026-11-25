package deployment

import (
	"context"
	"errors"
	"sync"
)

var ErrRecordNotFound = errors.New("deployment record not found")

// RecordRepository persists deployment lifecycle receipts independently of the
// storage and naming adapters that perform the external effects.
type RecordRepository interface {
	Get(context.Context, string) (Record, error)
	Put(context.Context, Record) error
}

type MemoryRecordRepository struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewMemoryRecordRepository() *MemoryRecordRepository {
	return &MemoryRecordRepository{records: map[string]Record{}}
}

func (r *MemoryRecordRepository) Get(_ context.Context, id string) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[id]
	if !ok {
		return Record{}, ErrRecordNotFound
	}
	return cloneRecord(record), nil
}

func (r *MemoryRecordRepository) Put(_ context.Context, record Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.records == nil {
		r.records = map[string]Record{}
	}
	r.records[record.ID] = cloneRecord(record)
	return nil
}

func cloneRecord(record Record) Record {
	record.Names = append([]NamedContent(nil), record.Names...)
	return record
}
