// Package config — retention store.
//
// Diff-archive retention has a tunable surface (age, total size,
// per-project cap) that operators want to adjust at runtime without
// editing systemd unit files or restarting the service. The store
// persists a single RetentionConfig blob to disk via api-core/storage's
// ClassConfig, mirroring the pattern of FileProfileStore.
//
// Lifecycle on startup:
//
//  1. Defaults() seeds RetentionConfig (90 days / 10 GiB / unlimited).
//  2. LoadFromEnv overrides those with WORKSPACE_SANDBOX_RETENTION_*
//     environment variables, if set.
//  3. The retention store loads retention.json (if present) and that
//     value becomes the runtime source of truth — env-driven defaults
//     are the seed only when the file is missing.
//  4. Admin PUT /config/retention persists a new value via the store.
//     The reconciler reads from the store on each tick, so updates
//     take effect on the next pass.
//
// Why a separate store rather than mutating Config in place:
//
//   - Validation can refuse a Set without partially-mutating handler
//     state.
//   - Reads are atomic and lock-coordinated; the reconciler can read
//     mid-tick without racing the handler.
//   - Persistence is decoupled from request handling — the handler
//     fails the PUT cleanly if the file write fails, and a successful
//     PUT survives process restart.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrooli/api-core/storage"
)

// RetentionStore is the abstraction the retention reconciler reads from
// and the /config/retention handler writes through. Two implementations
// ship: FileRetentionStore (production, JSON file under ClassConfig)
// and an in-memory store used by tests (constructed via
// NewMemoryRetentionStore).
type RetentionStore interface {
	// Get returns the current retention config. Never nil; the zero
	// value of RetentionConfig (all zeros = everything disabled) is
	// a valid return value.
	Get() RetentionConfig

	// Set validates and persists the new config. Returns the same
	// validation errors as Config.Validate's retention block. On
	// success the new value is immediately visible via Get.
	Set(cfg RetentionConfig) error
}

// FileRetentionStore persists RetentionConfig to a JSON file.
type FileRetentionStore struct {
	path string

	mu     sync.RWMutex
	cache  RetentionConfig
	loaded bool
}

// NewFileRetentionStore constructs a store backed by retention.json
// under api-core/storage's ClassConfig for the workspace-sandbox
// scenario. seed provides the initial value when no file exists yet
// (typically Config.Retention from LoadFromEnv).
func NewFileRetentionStore(seed RetentionConfig) (*FileRetentionStore, error) {
	path, err := resolveRetentionPath()
	if err != nil {
		return nil, err
	}
	s := &FileRetentionStore{path: path, cache: seed}
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	return s, nil
}

// NewFileRetentionStoreAtPath constructs a store pinned to an explicit
// file path. Primarily for tests; production should call
// NewFileRetentionStore.
func NewFileRetentionStoreAtPath(path string, seed RetentionConfig) *FileRetentionStore {
	return &FileRetentionStore{path: path, cache: seed}
}

// resolveRetentionPath returns the canonical retention.json location
// under ClassConfig.
func resolveRetentionPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	return resolver.Path(
		storage.Options{ScenarioID: "workspace-sandbox"},
		storage.ClassConfig,
		"retention.json",
	)
}

// Get returns the current config, lazily loading from disk on first use.
func (s *FileRetentionStore) Get() RetentionConfig {
	s.mu.RLock()
	if s.loaded {
		out := s.cache
		s.mu.RUnlock()
		return out
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// ensureLoaded is no-op if another goroutine loaded between locks.
	_ = s.ensureLoadedLocked()
	return s.cache
}

// Set validates and persists the new config.
func (s *FileRetentionStore) Set(cfg RetentionConfig) error {
	if err := validateRetentionConfig(cfg); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = cfg
	s.loaded = true
	return s.persistLocked()
}

// ensureLoaded loads the cache from disk if not yet loaded.
func (s *FileRetentionStore) ensureLoaded() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensureLoadedLocked()
}

// ensureLoadedLocked must be called with s.mu held.
func (s *FileRetentionStore) ensureLoadedLocked() error {
	if s.loaded {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		// First boot: keep the seed (set by NewFileRetentionStore) as
		// the in-memory value and mark loaded so we don't re-stat on
		// every Get. We do NOT eagerly persist on load; the seed
		// becomes durable on the first explicit Set or never if the
		// operator is fine with the default.
		s.loaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("retention_store: read %s: %w", s.path, err)
	}
	var cfg RetentionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("retention_store: parse %s: %w", s.path, err)
	}
	if vErr := validateRetentionConfig(cfg); vErr != nil {
		return fmt.Errorf("retention_store: invalid persisted config in %s: %w", s.path, vErr)
	}
	s.cache = cfg
	s.loaded = true
	return nil
}

// persistLocked writes the cache to disk atomically. Must be called with
// s.mu held.
func (s *FileRetentionStore) persistLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("retention_store: mkdir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(s.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("retention_store: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".retention-*.json")
	if err != nil {
		return fmt.Errorf("retention_store: tmpfile: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("retention_store: write tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("retention_store: close tmp: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("retention_store: rename %s: %w", s.path, err)
	}
	return nil
}

// validateRetentionConfig is the same rule set Config.Validate enforces
// on the boot path, scoped to the retention block. Kept as a free
// function so the store and the global Validate share one truth.
func validateRetentionConfig(c RetentionConfig) error {
	if c.MaxArchiveAgeDays < 0 {
		return errors.New("retention.maxArchiveAgeDays must be >= 0 (0 disables age-based eviction)")
	}
	if c.MaxArchiveSizeBytes < 0 {
		return errors.New("retention.maxArchiveSizeBytes must be >= 0 (0 disables size-based eviction)")
	}
	if c.MaxArchivesPerProject < 0 {
		return errors.New("retention.maxArchivesPerProject must be >= 0 (0 disables the per-project cap)")
	}
	return nil
}

// MemoryRetentionStore is an in-memory RetentionStore used by tests.
type MemoryRetentionStore struct {
	mu    sync.RWMutex
	cache RetentionConfig
}

// NewMemoryRetentionStore constructs an in-memory store seeded with cfg.
func NewMemoryRetentionStore(cfg RetentionConfig) *MemoryRetentionStore {
	return &MemoryRetentionStore{cache: cfg}
}

// Get returns the current config.
func (m *MemoryRetentionStore) Get() RetentionConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache
}

// Set validates and replaces the current config.
func (m *MemoryRetentionStore) Set(cfg RetentionConfig) error {
	if err := validateRetentionConfig(cfg); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = cfg
	return nil
}

var (
	_ RetentionStore = (*FileRetentionStore)(nil)
	_ RetentionStore = (*MemoryRetentionStore)(nil)
)
