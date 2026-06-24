package systemevents

import (
	"context"
	"sync"
	"time"
)

// CursorState is the persisted incremental-ingest position for a journal-backed
// source. Cursor is the journald __CURSOR of the last successfully-ingested
// entry; BootID pins it to the boot it was captured on so a reboot forces a
// fresh read rather than replaying a stale cursor against a rotated journal.
type CursorState struct {
	Cursor    string
	BootID    string
	UpdatedAt time.Time
}

// CursorStore persists per-source journal cursors and per-boot "already
// scanned" markers so kernel-signal ingestion can read incrementally instead
// of re-grepping every boot on every tick.
//
// Implementations must be safe for the single-goroutine ingest path; the
// production SQLite store is wrapped by a pool of one, and the in-memory test
// fake guards with a mutex.
type CursorStore interface {
	// GetJournalCursor returns the persisted cursor state for sourceKey. A
	// missing key returns an empty CursorState (not an error) so cold start is
	// indistinguishable from "nothing ingested yet".
	GetJournalCursor(ctx context.Context, sourceKey string) (CursorState, error)
	// SetJournalCursor persists the cursor state for sourceKey. Callers advance
	// the cursor ONLY after a successful ingest so a failed read never skips
	// events.
	SetJournalCursor(ctx context.Context, sourceKey string, state CursorState) error
	// IsBootScanned reports whether the (sourceKey, bootID) pair has already
	// been fully scanned. Immutable historical boots are scanned at most once.
	IsBootScanned(ctx context.Context, sourceKey, bootID string) (bool, error)
	// MarkBootScanned records that (sourceKey, bootID) has been scanned.
	MarkBootScanned(ctx context.Context, sourceKey, bootID string) error
}

// MemoryCursorStore is an in-memory CursorStore for tests. It is safe for
// concurrent use.
type MemoryCursorStore struct {
	mu      sync.Mutex
	cursors map[string]CursorState
	scanned map[string]map[string]bool
}

// NewMemoryCursorStore returns an empty in-memory cursor store.
func NewMemoryCursorStore() *MemoryCursorStore {
	return &MemoryCursorStore{
		cursors: make(map[string]CursorState),
		scanned: make(map[string]map[string]bool),
	}
}

func (m *MemoryCursorStore) GetJournalCursor(_ context.Context, sourceKey string) (CursorState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cursors[sourceKey], nil
}

func (m *MemoryCursorStore) SetJournalCursor(_ context.Context, sourceKey string, state CursorState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cursors[sourceKey] = state
	return nil
}

func (m *MemoryCursorStore) IsBootScanned(_ context.Context, sourceKey, bootID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scanned[sourceKey][bootID], nil
}

func (m *MemoryCursorStore) MarkBootScanned(_ context.Context, sourceKey, bootID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scanned[sourceKey] == nil {
		m.scanned[sourceKey] = make(map[string]bool)
	}
	m.scanned[sourceKey][bootID] = true
	return nil
}
