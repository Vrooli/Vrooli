package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// JSONFileStore is a file-based implementation of Store using JSON serialization.
// Each item can be stored in a single JSON file or all items in one file.
//
// Thread-safe for concurrent access within a single process.
// For multi-process safety, use an external locking mechanism.
type JSONFileStore[K comparable, T any] struct {
	// path is the base path for storage
	path string
	// mode determines storage layout
	mode FileStoreMode
	// cache holds in-memory data for performance
	cache map[K]T
	// mu protects cache
	mu sync.RWMutex
	// dirty tracks if cache has unsaved changes
	dirty bool
	// options holds configuration
	options StoreOptions[K, T]
	// keyToFileName converts keys to file names (for PerItem mode)
	keyToFileName func(K) string
	// fileNameToKey converts file names to keys (for PerItem mode)
	fileNameToKey func(string) (K, bool)
}

// FileStoreMode determines how items are stored on disk.
type FileStoreMode int

const (
	// SingleFile stores all items in one JSON file.
	// Good for small datasets that need atomic updates.
	SingleFile FileStoreMode = iota
	// PerItem stores each item in a separate JSON file.
	// Good for large datasets or when items are accessed individually.
	PerItem
)

// JSONFileStoreOption is a functional option for JSONFileStore.
type JSONFileStoreOption[K comparable, T any] func(*JSONFileStore[K, T])

// NewJSONFileStore creates a new JSON file store.
// For SingleFile mode, path should be a .json file path.
// For PerItem mode, path should be a directory path.
func NewJSONFileStore[K comparable, T any](path string, mode FileStoreMode, opts ...JSONFileStoreOption[K, T]) (*JSONFileStore[K, T], error) {
	s := &JSONFileStore[K, T]{
		path:  path,
		mode:  mode,
		cache: make(map[K]T),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Set defaults for PerItem mode
	if mode == PerItem {
		if s.keyToFileName == nil {
			// Default: use key as filename with .json extension
			s.keyToFileName = func(k K) string {
				return fmt.Sprintf("%v.json", k)
			}
		}
		if s.fileNameToKey == nil {
			// This will be overridden for string keys
			s.fileNameToKey = func(filename string) (K, bool) {
				// Can't implement generically, must be set via option
				var zero K
				return zero, false
			}
		}
	}

	// Ensure directory exists
	if mode == PerItem {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, fmt.Errorf("create store directory: %w", err)
		}
	} else {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create store directory: %w", err)
		}
	}

	// Load existing data
	if err := s.Load(context.Background()); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load existing data: %w", err)
	}

	return s, nil
}

// WithFileKeyConverter sets the key-to-filename and filename-to-key converters.
// Required for PerItem mode with non-string keys.
func WithFileKeyConverter[K comparable, T any](toFileName func(K) string, fromFileName func(string) (K, bool)) JSONFileStoreOption[K, T] {
	return func(s *JSONFileStore[K, T]) {
		s.keyToFileName = toFileName
		s.fileNameToKey = fromFileName
	}
}

// WithFileStoreOptions sets the common store options.
func WithFileStoreOptions[K comparable, T any](opts StoreOptions[K, T]) JSONFileStoreOption[K, T] {
	return func(s *JSONFileStore[K, T]) {
		s.options = opts
	}
}

// Get retrieves an item by key.
func (s *JSONFileStore[K, T]) Get(ctx context.Context, key K) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.cache[key]
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
func (s *JSONFileStore[K, T]) Save(ctx context.Context, key K, item T) error {
	if s.options.Validator != nil {
		if err := s.options.Validator(item); err != nil {
			return err
		}
	}

	if s.options.BeforeSave != nil {
		item = s.options.BeforeSave(item)
	}

	s.mu.Lock()
	s.cache[key] = item
	s.dirty = true
	s.mu.Unlock()

	// For PerItem mode, write immediately
	if s.mode == PerItem {
		return s.writeItem(key, item)
	}

	// For SingleFile mode, write all data
	return s.Flush(ctx)
}

// Delete removes an item by key.
func (s *JSONFileStore[K, T]) Delete(ctx context.Context, key K) error {
	s.mu.Lock()
	if _, ok := s.cache[key]; !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	delete(s.cache, key)
	s.dirty = true
	s.mu.Unlock()

	// For PerItem mode, remove the file
	if s.mode == PerItem {
		filename := s.keyToFileName(key)
		path := filepath.Join(s.path, filename)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove file: %w", err)
		}
		return nil
	}

	// For SingleFile mode, write all data
	return s.Flush(ctx)
}

// Exists checks if an item exists.
func (s *JSONFileStore[K, T]) Exists(ctx context.Context, key K) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.cache[key]
	return ok, nil
}

// List returns all items.
func (s *JSONFileStore[K, T]) List(ctx context.Context) ([]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]T, 0, len(s.cache))
	for _, item := range s.cache {
		if s.options.AfterLoad != nil {
			item = s.options.AfterLoad(item)
		}
		items = append(items, item)
	}

	return items, nil
}

// ListKeys returns all keys.
func (s *JSONFileStore[K, T]) ListKeys(ctx context.Context) ([]K, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]K, 0, len(s.cache))
	for key := range s.cache {
		keys = append(keys, key)
	}

	return keys, nil
}

// Count returns the number of items.
func (s *JSONFileStore[K, T]) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache), nil
}

// Flush writes all pending changes to disk.
func (s *JSONFileStore[K, T]) Flush(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dirty && s.mode == SingleFile {
		return nil
	}

	if s.mode == SingleFile {
		return s.writeAllLocked()
	}

	// PerItem mode: nothing to flush, items are written immediately
	s.dirty = false
	return nil
}

// Load reloads data from disk.
func (s *JSONFileStore[K, T]) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mode == SingleFile {
		return s.loadSingleFile()
	}
	return s.loadPerItem()
}

// Close flushes and closes the store.
func (s *JSONFileStore[K, T]) Close() error {
	return s.Flush(context.Background())
}

// writeAllLocked writes all cache data to the single file.
// Must be called with lock held.
func (s *JSONFileStore[K, T]) writeAllLocked() error {
	data, err := json.MarshalIndent(s.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	// Write to temp file first for atomicity
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	s.dirty = false
	return nil
}

// writeItem writes a single item to its file.
func (s *JSONFileStore[K, T]) writeItem(key K, item T) error {
	filename := s.keyToFileName(key)
	path := filepath.Join(s.path, filename)

	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal item: %w", err)
	}

	// Write to temp file first for atomicity
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// loadSingleFile loads all data from a single JSON file.
func (s *JSONFileStore[K, T]) loadSingleFile() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.cache = make(map[K]T)
			return nil
		}
		return fmt.Errorf("read file: %w", err)
	}

	if len(data) == 0 {
		s.cache = make(map[K]T)
		return nil
	}

	if err := json.Unmarshal(data, &s.cache); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	s.dirty = false
	return nil
}

// loadPerItem loads all items from individual files in the directory.
func (s *JSONFileStore[K, T]) loadPerItem() error {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.cache = make(map[K]T)
			return nil
		}
		return fmt.Errorf("read directory: %w", err)
	}

	s.cache = make(map[K]T)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Skip temp files
		name := entry.Name()
		if len(name) > 4 && name[len(name)-8:] == ".tmp.json" {
			continue
		}

		key, ok := s.fileNameToKey(name)
		if !ok {
			continue
		}

		path := filepath.Join(s.path, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", name, err)
		}

		var item T
		if err := json.Unmarshal(data, &item); err != nil {
			return fmt.Errorf("unmarshal file %s: %w", name, err)
		}

		s.cache[key] = item
	}

	s.dirty = false
	return nil
}

// Filter returns items matching the predicate.
func (s *JSONFileStore[K, T]) Filter(ctx context.Context, predicate FilterFunc[T]) ([]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []T
	for _, item := range s.cache {
		if s.options.AfterLoad != nil {
			item = s.options.AfterLoad(item)
		}
		if predicate(item) {
			result = append(result, item)
		}
	}

	return result, nil
}

// NewJSONFileStoreString is a convenience constructor for string-keyed stores.
// For PerItem mode, it automatically handles key-to-filename conversion.
func NewJSONFileStoreString[T any](path string, mode FileStoreMode, opts ...JSONFileStoreOption[string, T]) (*JSONFileStore[string, T], error) {
	// Add string key converter for PerItem mode
	if mode == PerItem {
		opts = append([]JSONFileStoreOption[string, T]{
			WithFileKeyConverter[string, T](
				func(k string) string {
					// Sanitize key for safe filename
					return sanitizeFilename(k) + ".json"
				},
				func(filename string) (string, bool) {
					// Remove .json extension
					if len(filename) < 6 || filename[len(filename)-5:] != ".json" {
						return "", false
					}
					return filename[:len(filename)-5], true
				},
			),
		}, opts...)
	}

	return NewJSONFileStore[string, T](path, mode, opts...)
}

// sanitizeFilename makes a string safe for use as a filename.
func sanitizeFilename(s string) string {
	// Replace unsafe characters
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			result = append(result, c)
		} else {
			// Replace unsafe chars with underscore
			result = append(result, '_')
		}
	}
	return string(result)
}

// HybridStore combines in-memory caching with file persistence.
// Changes are written to disk on Save but kept in memory for fast reads.
// This is essentially what JSONFileStore already does, but this wrapper
// makes the intent explicit and adds explicit flush control.
type HybridStore[K comparable, T any] struct {
	*JSONFileStore[K, T]
	autoFlush bool
}

// NewHybridStore creates a store with explicit flush control.
func NewHybridStore[K comparable, T any](path string, mode FileStoreMode, autoFlush bool, opts ...JSONFileStoreOption[K, T]) (*HybridStore[K, T], error) {
	jsonStore, err := NewJSONFileStore[K, T](path, mode, opts...)
	if err != nil {
		return nil, err
	}
	return &HybridStore[K, T]{
		JSONFileStore: jsonStore,
		autoFlush:     autoFlush,
	}, nil
}

// Save stores an item with optional auto-flush.
func (s *HybridStore[K, T]) Save(ctx context.Context, key K, item T) error {
	if s.options.Validator != nil {
		if err := s.options.Validator(item); err != nil {
			return err
		}
	}

	if s.options.BeforeSave != nil {
		item = s.options.BeforeSave(item)
	}

	s.mu.Lock()
	s.cache[key] = item
	s.dirty = true
	s.mu.Unlock()

	if s.autoFlush {
		return s.Flush(ctx)
	}
	return nil
}

// Delete removes an item with optional auto-flush.
func (s *HybridStore[K, T]) Delete(ctx context.Context, key K) error {
	s.mu.Lock()
	if _, ok := s.cache[key]; !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	delete(s.cache, key)
	s.dirty = true
	s.mu.Unlock()

	if s.autoFlush {
		return s.Flush(ctx)
	}
	return nil
}
