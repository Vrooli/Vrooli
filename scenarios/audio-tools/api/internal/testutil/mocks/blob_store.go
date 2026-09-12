package mocks

import (
	"context"
	"errors"
	"sync"

	"audio-tools/internal/blobbytes"
)

// FakeBlobStore is the shared in-memory byte store for domain tests. Keeping
// this fake in testutil prevents each persistence suite from carrying a
// subtly different Put/Get/Delete implementation.
type FakeBlobStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func NewFakeBlobStore() *FakeBlobStore {
	return &FakeBlobStore{data: make(map[string][]byte)}
}

func (s *FakeBlobStore) Put(_ context.Context, key string, data []byte, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string][]byte)
	}
	s.data[key] = append([]byte(nil), data...)
	return nil
}

func (s *FakeBlobStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	if !ok {
		return nil, errors.New("blob not found")
	}
	return append([]byte(nil), data...), nil
}

func (s *FakeBlobStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *FakeBlobStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}

var _ blobbytes.Store = (*FakeBlobStore)(nil)
