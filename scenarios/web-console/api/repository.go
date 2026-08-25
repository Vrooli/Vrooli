package main

import "context"

// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SEAMS.md#session-metadata-store-seam-api

// ShortcutStore abstracts shortcut profile storage. Implementations may be
// in-memory (for tests) or SQLite-backed (for production persistence).
type ShortcutStore interface {
	List(ctx context.Context) []*ShortcutProfile
	Get(ctx context.Context, id string) (*ShortcutProfile, bool)
	Upsert(ctx context.Context, id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile
	Delete(ctx context.Context, id string) bool
	Effective(ctx context.Context) []ShortcutEntry
}
