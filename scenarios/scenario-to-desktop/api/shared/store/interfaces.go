// Package store provides generic store interfaces and implementations
// for persisting domain entities.
//
// This package offers two primary store implementations:
//   - InMemoryStore: Fast, non-persistent storage suitable for development and testing
//   - JSONFileStore: Persistent file-based storage using JSON serialization
//
// Both implementations are safe for concurrent use.
package store

import (
	"context"
	"errors"
)

// Common store errors.
var (
	// ErrNotFound is returned when an item is not found in the store.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when trying to create an item that already exists.
	ErrAlreadyExists = errors.New("already exists")
	// ErrStoreClosed is returned when operating on a closed store.
	ErrStoreClosed = errors.New("store closed")
)

// Store is the generic interface for entity storage.
// It provides CRUD operations with context support for cancellation.
//
// Type parameter T represents the stored entity type.
// Type parameter K represents the key type (usually string).
type Store[K comparable, T any] interface {
	// Get retrieves an item by its key.
	// Returns ErrNotFound if the item doesn't exist.
	Get(ctx context.Context, key K) (T, error)

	// Save stores or updates an item.
	// If the item doesn't exist, it will be created.
	// If it exists, it will be updated.
	Save(ctx context.Context, key K, item T) error

	// Delete removes an item by its key.
	// Returns ErrNotFound if the item doesn't exist.
	Delete(ctx context.Context, key K) error

	// Exists checks if an item exists without retrieving it.
	Exists(ctx context.Context, key K) (bool, error)

	// List returns all items in the store.
	// For large stores, consider using ListKeys + Get for pagination.
	List(ctx context.Context) ([]T, error)

	// ListKeys returns all keys in the store.
	ListKeys(ctx context.Context) ([]K, error)

	// Count returns the number of items in the store.
	Count(ctx context.Context) (int, error)
}

// KeyedStore extends Store with the ability to get an item's key.
// This is useful when the key is embedded in the item itself.
type KeyedStore[K comparable, T any] interface {
	Store[K, T]

	// KeyOf extracts the key from an item.
	// This is used when the key is part of the item structure.
	KeyOf(item T) K
}

// FilterFunc is a predicate function for filtering items.
type FilterFunc[T any] func(item T) bool

// FilterableStore extends Store with filtering capabilities.
type FilterableStore[K comparable, T any] interface {
	Store[K, T]

	// Filter returns items matching the predicate.
	Filter(ctx context.Context, predicate FilterFunc[T]) ([]T, error)

	// FindFirst returns the first item matching the predicate.
	// Returns ErrNotFound if no item matches.
	FindFirst(ctx context.Context, predicate FilterFunc[T]) (T, error)
}

// CleanableStore extends Store with cleanup capabilities.
// This is useful for stores that need periodic garbage collection.
type CleanableStore[K comparable, T any] interface {
	Store[K, T]

	// Cleanup removes stale entries based on implementation-specific criteria.
	// For example, removing expired sessions or completed jobs.
	Cleanup(ctx context.Context) error
}

// PersistentStore extends Store with explicit persistence control.
type PersistentStore[K comparable, T any] interface {
	Store[K, T]

	// Flush ensures all pending writes are persisted.
	Flush(ctx context.Context) error

	// Load reloads data from the underlying storage.
	Load(ctx context.Context) error

	// Close closes the store and releases resources.
	Close() error
}

// KeyExtractor is a function that extracts a key from an item.
// Used by stores to automatically derive keys from items.
type KeyExtractor[K comparable, T any] func(item T) K

// Validator is a function that validates an item before storage.
// Returns an error if the item is invalid.
type Validator[T any] func(item T) error

// Transformer is a function that transforms an item before storage or retrieval.
type Transformer[T any] func(item T) T

// StoreOptions provides common configuration for store implementations.
type StoreOptions[K comparable, T any] struct {
	// KeyExtractor extracts keys from items (optional).
	// If not provided, keys must be explicitly passed to Save.
	KeyExtractor KeyExtractor[K, T]

	// Validator validates items before storage (optional).
	Validator Validator[T]

	// BeforeSave transforms items before saving (optional).
	BeforeSave Transformer[T]

	// AfterLoad transforms items after loading (optional).
	AfterLoad Transformer[T]
}

// WatchEvent represents a change to a stored item.
type WatchEvent[K comparable, T any] struct {
	// Type is the type of change: "create", "update", or "delete"
	Type string
	// Key is the affected item's key
	Key K
	// Item is the new value (nil for deletes)
	Item *T
	// Previous is the previous value (nil for creates)
	Previous *T
}

// WatchableStore extends Store with change notification capabilities.
type WatchableStore[K comparable, T any] interface {
	Store[K, T]

	// Watch returns a channel that receives events for all changes.
	// The returned stop function should be called to stop watching.
	Watch(ctx context.Context) (<-chan WatchEvent[K, T], func())
}

// TransactionalStore provides atomic operations across multiple changes.
type TransactionalStore[K comparable, T any] interface {
	Store[K, T]

	// Transaction executes a function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	Transaction(ctx context.Context, fn func(Store[K, T]) error) error
}

// CompositeKey creates a composite key from multiple string parts.
// Useful for stores that need hierarchical keys.
func CompositeKey(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "/" + parts[i]
	}
	return result
}

// WithPrefix returns a new key prefixed with the given prefix.
func WithPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "/" + key
}
