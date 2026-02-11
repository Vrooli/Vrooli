package xrefs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IndexStore manages the persisted cross-reference index.
type IndexStore struct {
	storeDir string
	scanner  *Scanner
	mu       sync.Mutex
}

// NewIndexStore creates a new index store.
func NewIndexStore(storeDir string, scanner *Scanner) *IndexStore {
	return &IndexStore{
		storeDir: storeDir,
		scanner:  scanner,
	}
}

// indexPath returns the path to the xrefs index file.
func (s *IndexStore) indexPath() string {
	return filepath.Join(s.storeDir, "indexes", "xrefs.index.json")
}

// Get loads the index from file, regenerating if missing.
func (s *IndexStore) Get(ctx context.Context) (*XRefsIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try reading existing index
	data, err := os.ReadFile(s.indexPath())
	if err == nil {
		var idx XRefsIndex
		if err := json.Unmarshal(data, &idx); err == nil {
			return &idx, nil
		}
	}

	// Regenerate if missing or corrupt
	return s.regenerateLocked(ctx)
}

// GetForSkill returns references for a specific skill ID.
func (s *IndexStore) GetForSkill(ctx context.Context, skillID string) ([]Reference, error) {
	idx, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []Reference
	for _, ref := range idx.References {
		if ref.SkillID == skillID {
			filtered = append(filtered, ref)
		}
	}
	return filtered, nil
}

// Invalidate deletes the index file so it will be regenerated on next read.
func (s *IndexStore) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = os.Remove(s.indexPath())
}

// Regenerate rebuilds the index from scratch.
func (s *IndexStore) Regenerate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.regenerateLocked(ctx)
	return err
}

// regenerateLocked performs the actual regeneration. Caller must hold s.mu.
func (s *IndexStore) regenerateLocked(ctx context.Context) (*XRefsIndex, error) {
	refs, err := s.scanner.ScanAll(ctx)
	if err != nil {
		return nil, err
	}

	idx := &XRefsIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		References:  refs,
	}

	// Ensure directory exists
	dir := filepath.Dir(s.indexPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(s.indexPath(), data, 0o644); err != nil {
		return nil, err
	}

	return idx, nil
}
