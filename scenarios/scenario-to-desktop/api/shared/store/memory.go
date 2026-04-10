package store

import (
	"context"
	"sync"
	"time"
)

// InMemoryStore is a thread-safe in-memory implementation of Store.
// It supports all basic Store operations plus filtering and cleanup.
type InMemoryStore[K comparable, T any] struct {
	data    map[K]T
	mu      sync.RWMutex
	options StoreOptions[K, T]

	// Cleanup configuration
	cleanupFunc func(key K, item T) bool
}

// InMemoryStoreOption is a functional option for InMemoryStore.
type InMemoryStoreOption[K comparable, T any] func(*InMemoryStore[K, T])

// NewInMemoryStore creates a new in-memory store with optional configuration.
func NewInMemoryStore[K comparable, T any](opts ...InMemoryStoreOption[K, T]) *InMemoryStore[K, T] {
	s := &InMemoryStore[K, T]{
		data: make(map[K]T),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithKeyExtractor sets the key extractor function.
func WithKeyExtractor[K comparable, T any](fn KeyExtractor[K, T]) InMemoryStoreOption[K, T] {
	return func(s *InMemoryStore[K, T]) {
		s.options.KeyExtractor = fn
	}
}

// WithValidator sets the validator function.
func WithValidator[K comparable, T any](fn Validator[T]) InMemoryStoreOption[K, T] {
	return func(s *InMemoryStore[K, T]) {
		s.options.Validator = fn
	}
}

// WithCleanupFunc sets the cleanup predicate function.
// Items for which the function returns true will be removed during cleanup.
func WithCleanupFunc[K comparable, T any](fn func(key K, item T) bool) InMemoryStoreOption[K, T] {
	return func(s *InMemoryStore[K, T]) {
		s.cleanupFunc = fn
	}
}

// Get retrieves an item by key.
func (s *InMemoryStore[K, T]) Get(ctx context.Context, key K) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data[key]
	if !ok {
		var zero T
		return zero, ErrNotFound
	}

	if s.options.AfterLoad != nil {
		item = s.options.AfterLoad(item)
	}

	return item, nil
}

// Save stores or updates an item.
func (s *InMemoryStore[K, T]) Save(ctx context.Context, key K, item T) error {
	if s.options.Validator != nil {
		if err := s.options.Validator(item); err != nil {
			return err
		}
	}

	if s.options.BeforeSave != nil {
		item = s.options.BeforeSave(item)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = item
	return nil
}

// SaveWithExtractedKey saves an item using the configured key extractor.
// Panics if no key extractor is configured.
func (s *InMemoryStore[K, T]) SaveWithExtractedKey(ctx context.Context, item T) error {
	if s.options.KeyExtractor == nil {
		panic("SaveWithExtractedKey called without KeyExtractor configured")
	}
	key := s.options.KeyExtractor(item)
	return s.Save(ctx, key, item)
}

// Delete removes an item by key.
func (s *InMemoryStore[K, T]) Delete(ctx context.Context, key K) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[key]; !ok {
		return ErrNotFound
	}

	delete(s.data, key)
	return nil
}

// Exists checks if an item exists.
func (s *InMemoryStore[K, T]) Exists(ctx context.Context, key K) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data[key]
	return ok, nil
}

// List returns all items.
func (s *InMemoryStore[K, T]) List(ctx context.Context) ([]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]T, 0, len(s.data))
	for _, item := range s.data {
		if s.options.AfterLoad != nil {
			item = s.options.AfterLoad(item)
		}
		items = append(items, item)
	}

	return items, nil
}

// ListKeys returns all keys.
func (s *InMemoryStore[K, T]) ListKeys(ctx context.Context) ([]K, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]K, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}

	return keys, nil
}

// Count returns the number of items.
func (s *InMemoryStore[K, T]) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data), nil
}

// Filter returns items matching the predicate.
func (s *InMemoryStore[K, T]) Filter(ctx context.Context, predicate FilterFunc[T]) ([]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []T
	for _, item := range s.data {
		if s.options.AfterLoad != nil {
			item = s.options.AfterLoad(item)
		}
		if predicate(item) {
			result = append(result, item)
		}
	}

	return result, nil
}

// FindFirst returns the first item matching the predicate.
func (s *InMemoryStore[K, T]) FindFirst(ctx context.Context, predicate FilterFunc[T]) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.data {
		if s.options.AfterLoad != nil {
			item = s.options.AfterLoad(item)
		}
		if predicate(item) {
			return item, nil
		}
	}

	var zero T
	return zero, ErrNotFound
}

// Cleanup removes items that match the cleanup predicate.
// Returns the number of items removed.
func (s *InMemoryStore[K, T]) Cleanup(ctx context.Context) error {
	if s.cleanupFunc == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, item := range s.data {
		if s.cleanupFunc(key, item) {
			delete(s.data, key)
		}
	}

	return nil
}

// CleanupWithCallback removes items and calls the callback for each removed item.
// This is useful for resource cleanup (e.g., closing connections, removing temp files).
func (s *InMemoryStore[K, T]) CleanupWithCallback(ctx context.Context, shouldRemove func(K, T) bool, onRemove func(K, T)) (int, error) {
	s.mu.Lock()

	var toRemove []struct {
		key  K
		item T
	}

	for key, item := range s.data {
		if shouldRemove(key, item) {
			toRemove = append(toRemove, struct {
				key  K
				item T
			}{key, item})
		}
	}

	for _, r := range toRemove {
		delete(s.data, r.key)
	}

	s.mu.Unlock()

	// Call callbacks outside the lock to avoid deadlocks
	for _, r := range toRemove {
		if onRemove != nil {
			onRemove(r.key, r.item)
		}
	}

	return len(toRemove), nil
}

// GetAndDelete retrieves an item and removes it atomically.
func (s *InMemoryStore[K, T]) GetAndDelete(ctx context.Context, key K) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.data[key]
	if !ok {
		var zero T
		return zero, ErrNotFound
	}

	delete(s.data, key)

	if s.options.AfterLoad != nil {
		item = s.options.AfterLoad(item)
	}

	return item, nil
}

// Update atomically updates an item using an update function.
// If the item doesn't exist, returns ErrNotFound.
func (s *InMemoryStore[K, T]) Update(ctx context.Context, key K, updateFn func(T) T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.data[key]
	if !ok {
		return ErrNotFound
	}

	updated := updateFn(item)

	if s.options.Validator != nil {
		if err := s.options.Validator(updated); err != nil {
			return err
		}
	}

	if s.options.BeforeSave != nil {
		updated = s.options.BeforeSave(updated)
	}

	s.data[key] = updated
	return nil
}

// GetOrCreate retrieves an item, or creates it if it doesn't exist.
func (s *InMemoryStore[K, T]) GetOrCreate(ctx context.Context, key K, create func() T) (T, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item, ok := s.data[key]; ok {
		if s.options.AfterLoad != nil {
			item = s.options.AfterLoad(item)
		}
		return item, false, nil
	}

	item := create()

	if s.options.Validator != nil {
		if err := s.options.Validator(item); err != nil {
			var zero T
			return zero, false, err
		}
	}

	if s.options.BeforeSave != nil {
		item = s.options.BeforeSave(item)
	}

	s.data[key] = item
	return item, true, nil
}

// Clear removes all items from the store.
func (s *InMemoryStore[K, T]) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[K]T)
	return nil
}

// Snapshot returns a copy of all data at a point in time.
func (s *InMemoryStore[K, T]) Snapshot(ctx context.Context) (map[K]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[K]T, len(s.data))
	for key, item := range s.data {
		snapshot[key] = item
	}

	return snapshot, nil
}

// TTLStore wraps an InMemoryStore with time-to-live support.
// Items are automatically expired based on their TTL.
type TTLStore[K comparable, T any] struct {
	*InMemoryStore[K, ttlWrapper[T]]
	defaultTTL time.Duration
}

type ttlWrapper[T any] struct {
	Item      T
	ExpiresAt time.Time
}

// NewTTLStore creates a new store with TTL support.
func NewTTLStore[K comparable, T any](defaultTTL time.Duration) *TTLStore[K, T] {
	store := NewInMemoryStore[K, ttlWrapper[T]](
		WithCleanupFunc[K, ttlWrapper[T]](func(_ K, w ttlWrapper[T]) bool {
			return time.Now().After(w.ExpiresAt)
		}),
	)
	return &TTLStore[K, T]{
		InMemoryStore: store,
		defaultTTL:    defaultTTL,
	}
}

// Get retrieves an item, checking expiry.
func (s *TTLStore[K, T]) Get(ctx context.Context, key K) (T, error) {
	wrapper, err := s.InMemoryStore.Get(ctx, key)
	if err != nil {
		var zero T
		return zero, err
	}

	if time.Now().After(wrapper.ExpiresAt) {
		_ = s.InMemoryStore.Delete(ctx, key)
		var zero T
		return zero, ErrNotFound
	}

	return wrapper.Item, nil
}

// Save stores an item with the default TTL.
func (s *TTLStore[K, T]) Save(ctx context.Context, key K, item T) error {
	return s.SaveWithTTL(ctx, key, item, s.defaultTTL)
}

// SaveWithTTL stores an item with a specific TTL.
func (s *TTLStore[K, T]) SaveWithTTL(ctx context.Context, key K, item T, ttl time.Duration) error {
	return s.InMemoryStore.Save(ctx, key, ttlWrapper[T]{
		Item:      item,
		ExpiresAt: time.Now().Add(ttl),
	})
}

// Refresh extends the TTL of an item.
func (s *TTLStore[K, T]) Refresh(ctx context.Context, key K, ttl time.Duration) error {
	return s.InMemoryStore.Update(ctx, key, func(w ttlWrapper[T]) ttlWrapper[T] {
		w.ExpiresAt = time.Now().Add(ttl)
		return w
	})
}

// List returns all non-expired items.
func (s *TTLStore[K, T]) List(ctx context.Context) ([]T, error) {
	wrappers, err := s.InMemoryStore.List(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var items []T
	for _, w := range wrappers {
		if now.Before(w.ExpiresAt) {
			items = append(items, w.Item)
		}
	}

	return items, nil
}
