package blobstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

// MemoryBlobStore is an in-memory BlobStore for tests.
type MemoryBlobStore struct {
	mu    sync.RWMutex
	items map[string]memoryBlob
}

type memoryBlob struct {
	data []byte
	mime string
}

func NewMemoryBlobStore() *MemoryBlobStore {
	return &MemoryBlobStore{items: make(map[string]memoryBlob)}
}

func (s *MemoryBlobStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("blobstore: key is required")
	}
	if r == nil {
		return errors.New("blobstore: reader is nil")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("blobstore: read blob: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = make(map[string]memoryBlob)
	}
	s.items[key] = memoryBlob{data: data, mime: strings.TrimSpace(mime)}
	return nil
}

func (s *MemoryBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	key = strings.TrimSpace(key)
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("blobstore: %q not found", key)
	}
	return io.NopCloser(bytes.NewReader(item.data)), item.mime, nil
}

func (s *MemoryBlobStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.items, strings.TrimSpace(key))
	s.mu.Unlock()
	return nil
}
