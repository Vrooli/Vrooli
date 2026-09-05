// Package mocks holds an in-memory metadata.Repository for testing callers.
package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"knowledge-observatory/internal/metadata"
)

// Repository is an in-memory metadata.Repository. Entries are keyed by
// vector_id and mappings by (namespace, external_id, kind), matching the real
// unique constraints.
type Repository struct {
	mu       sync.Mutex
	Entries  map[string]metadata.Entry
	Mappings map[string]metadata.ExternalIDMapping

	// Err, when set, is returned by every method.
	Err error
}

var _ metadata.Repository = (*Repository)(nil)

// New returns an empty repository.
func New() *Repository {
	return &Repository{
		Entries:  map[string]metadata.Entry{},
		Mappings: map[string]metadata.ExternalIDMapping{},
	}
}

func mappingKey(namespace, externalID, kind string) string {
	return fmt.Sprintf("%s\x00%s\x00%s", namespace, externalID, kind)
}

func (r *Repository) UpsertEntry(_ context.Context, e metadata.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	if e.VectorID == "" {
		return fmt.Errorf("vector_id is required")
	}
	if e.CollectionName == "" {
		return fmt.Errorf("collection_name is required")
	}
	if e.SourceType == "" {
		e.SourceType = "unknown"
	}
	if existing, ok := r.Entries[e.VectorID]; ok {
		e.ID = existing.ID
		e.CreatedAt = existing.CreatedAt
		if e.QualityScore == nil {
			e.QualityScore = existing.QualityScore // matches the COALESCE arm
		}
	} else {
		if e.ID == "" {
			e.ID = fmt.Sprintf("entry-%d", len(r.Entries)+1)
		}
		e.CreatedAt = time.Now().UTC()
	}
	e.UpdatedAt = time.Now().UTC()
	r.Entries[e.VectorID] = e
	return nil
}

func (r *Repository) GetEntry(_ context.Context, vectorID string) (metadata.Entry, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return metadata.Entry{}, false, r.Err
	}
	e, ok := r.Entries[vectorID]
	return e, ok, nil
}

func (r *Repository) LookupCollectionForVectorID(_ context.Context, vectorID string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return "", false, r.Err
	}
	e, ok := r.Entries[vectorID]
	if !ok {
		return "", false, nil
	}
	return e.CollectionName, true, nil
}

func (r *Repository) CountByCollection(_ context.Context, collection string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	var n int
	for _, e := range r.Entries {
		if e.CollectionName == collection {
			n++
		}
	}
	return n, nil
}

func (r *Repository) DeleteByCollection(_ context.Context, collection string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	var deleted int64
	for id, e := range r.Entries {
		if e.CollectionName == collection {
			delete(r.Entries, id)
			deleted++
		}
	}
	return deleted, nil
}

func (r *Repository) UpsertExternalIDMapping(_ context.Context, m metadata.ExternalIDMapping) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	if m.Namespace == "" || m.ExternalID == "" {
		return fmt.Errorf("namespace and external_id are required")
	}
	if m.Kind != "record" && m.Kind != "document" {
		return fmt.Errorf("kind must be one of: record, document")
	}
	if m.Kind == "record" && m.RecordID == "" {
		return fmt.Errorf("record_id is required for kind=record")
	}
	if m.Kind == "document" && m.DocumentID == "" {
		return fmt.Errorf("document_id is required for kind=document")
	}

	key := mappingKey(m.Namespace, m.ExternalID, m.Kind)
	if existing, ok := r.Mappings[key]; ok {
		m.ID = existing.ID
		m.CreatedAt = existing.CreatedAt
		if m.RecordID == "" {
			m.RecordID = existing.RecordID
		}
		if m.DocumentID == "" {
			m.DocumentID = existing.DocumentID
		}
		if m.ContentHash == "" {
			m.ContentHash = existing.ContentHash
		}
	} else {
		if m.ID == "" {
			m.ID = fmt.Sprintf("mapping-%d", len(r.Mappings)+1)
		}
		m.CreatedAt = time.Now().UTC()
	}
	m.UpdatedAt = time.Now().UTC()
	r.Mappings[key] = m
	return nil
}

func (r *Repository) LookupExternalIDMapping(_ context.Context, namespace, externalID, kind string) (metadata.ExternalIDMapping, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return metadata.ExternalIDMapping{}, false, r.Err
	}
	m, ok := r.Mappings[mappingKey(namespace, externalID, kind)]
	return m, ok, nil
}
