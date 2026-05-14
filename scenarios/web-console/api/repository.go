package main

// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SEAMS.md#axis-5-storage-abstraction

// ShortcutStore abstracts shortcut profile storage. Implementations may be
// in-memory (for tests) or SQLite-backed (for production persistence).
type ShortcutStore interface {
	List() []*ShortcutProfile
	Get(id string) (*ShortcutProfile, bool)
	Upsert(id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile
	Delete(id string) bool
	Effective() []ShortcutEntry
}
