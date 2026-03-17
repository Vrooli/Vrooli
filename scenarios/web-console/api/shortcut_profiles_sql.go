// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

// [REQ:P1-002a] Shortcut Profile Storage — SQLite implementation

// SQLShortcutStore persists shortcut profiles in SQLite.
// It implements ShortcutStore.
type SQLShortcutStore struct {
	db *sql.DB
}

// NewSQLShortcutStore creates a SQLite-backed shortcut store.
func NewSQLShortcutStore(db *sql.DB) *SQLShortcutStore {
	return &SQLShortcutStore{db: db}
}

// List returns all shortcut profiles from the database.
func (s *SQLShortcutStore) List() []*ShortcutProfile {
	rows, err := s.db.Query(`SELECT id, scope, name, shortcuts, created_at, updated_at FROM shortcut_profiles ORDER BY id`)
	if err != nil {
		log.Printf("SQLShortcutStore.List: query failed: %v", err)
		return nil
	}
	defer rows.Close()

	var profiles []*ShortcutProfile
	for rows.Next() {
		p, err := scanShortcutProfile(rows)
		if err != nil {
			log.Printf("SQLShortcutStore.List: scan failed: %v", err)
			continue
		}
		profiles = append(profiles, p)
	}
	if profiles == nil {
		profiles = make([]*ShortcutProfile, 0)
	}
	return profiles
}

// Get returns a shortcut profile by ID.
func (s *SQLShortcutStore) Get(id string) (*ShortcutProfile, bool) {
	row := s.db.QueryRow(
		`SELECT id, scope, name, shortcuts, created_at, updated_at FROM shortcut_profiles WHERE id = ?`, id)

	p, err := scanShortcutProfileRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		log.Printf("SQLShortcutStore.Get: scan failed: %v", err)
		return nil, false
	}
	return p, true
}

// Upsert creates or updates a shortcut profile. Replay-safe: if the content
// is identical to the existing profile, UpdatedAt is preserved.
func (s *SQLShortcutStore) Upsert(id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile {
	shortcutsJSON, err := json.Marshal(shortcuts)
	if err != nil {
		log.Printf("SQLShortcutStore.Upsert: marshal shortcuts: %v", err)
		return nil
	}

	now := formatTime(time.Now())
	// ON CONFLICT: only update if content changed (replay safety)
	row := s.db.QueryRow(`
		INSERT INTO shortcut_profiles (id, scope, name, shortcuts, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			scope = excluded.scope,
			name = excluded.name,
			shortcuts = excluded.shortcuts,
			updated_at = CASE
				WHEN shortcut_profiles.scope = excluded.scope
				  AND shortcut_profiles.name = excluded.name
				  AND shortcut_profiles.shortcuts = excluded.shortcuts
				THEN shortcut_profiles.updated_at
				ELSE excluded.updated_at
			END
		RETURNING id, scope, name, shortcuts, created_at, updated_at`,
		id, scope, name, string(shortcutsJSON), now, now)

	p, err := scanShortcutProfileRow(row)
	if err != nil {
		log.Printf("SQLShortcutStore.Upsert: scan failed: %v", err)
		return nil
	}
	return p
}

// Delete removes a shortcut profile by ID. Returns false if not found.
func (s *SQLShortcutStore) Delete(id string) bool {
	result, err := s.db.Exec(`DELETE FROM shortcut_profiles WHERE id = ?`, id)
	if err != nil {
		log.Printf("SQLShortcutStore.Delete: exec failed: %v", err)
		return false
	}
	n, _ := result.RowsAffected()
	return n > 0
}

// Effective returns the resolved shortcut list by selecting the
// highest-priority scope's profile.
func (s *SQLShortcutStore) Effective() []ShortcutEntry {
	row := s.db.QueryRow(`
		SELECT shortcuts FROM shortcut_profiles
		ORDER BY CASE scope
			WHEN 'parent' THEN 3
			WHEN 'workspace' THEN 2
			WHEN 'service' THEN 1
			ELSE 0
		END DESC
		LIMIT 1`)

	var shortcutsJSON string
	if err := row.Scan(&shortcutsJSON); err != nil {
		if err == sql.ErrNoRows {
			return defaultShortcuts
		}
		log.Printf("SQLShortcutStore.Effective: scan failed: %v", err)
		return defaultShortcuts
	}

	var shortcuts []ShortcutEntry
	if err := json.Unmarshal([]byte(shortcutsJSON), &shortcuts); err != nil {
		log.Printf("SQLShortcutStore.Effective: unmarshal failed: %v", err)
		return defaultShortcuts
	}
	return shortcuts
}

// scanShortcutProfile reads a ShortcutProfile from a *sql.Rows cursor.
func scanShortcutProfile(rows *sql.Rows) (*ShortcutProfile, error) {
	var p ShortcutProfile
	var shortcutsJSON string
	if err := rows.Scan(&p.ID, &p.Scope, &p.Name, &shortcutsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(shortcutsJSON), &p.Shortcuts); err != nil {
		return nil, err
	}
	return &p, nil
}

// scanShortcutProfileRow reads a ShortcutProfile from a *sql.Row.
func scanShortcutProfileRow(row *sql.Row) (*ShortcutProfile, error) {
	var p ShortcutProfile
	var shortcutsJSON string
	if err := row.Scan(&p.ID, &p.Scope, &p.Name, &shortcutsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(shortcutsJSON), &p.Shortcuts); err != nil {
		return nil, err
	}
	return &p, nil
}
