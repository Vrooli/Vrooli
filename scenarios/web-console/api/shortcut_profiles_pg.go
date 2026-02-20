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

// [REQ:P1-002a] Shortcut Profile Storage — PostgreSQL implementation

// PGShortcutStore persists shortcut profiles in PostgreSQL.
// It implements ShortcutStore.
type PGShortcutStore struct {
	db *sql.DB
}

// NewPGShortcutStore creates a PostgreSQL-backed shortcut store.
func NewPGShortcutStore(db *sql.DB) *PGShortcutStore {
	return &PGShortcutStore{db: db}
}

// List returns all shortcut profiles from the database.
func (s *PGShortcutStore) List() []*ShortcutProfile {
	rows, err := s.db.Query(`SELECT id, scope, name, shortcuts, created_at, updated_at FROM shortcut_profiles ORDER BY id`)
	if err != nil {
		log.Printf("PGShortcutStore.List: query failed: %v", err)
		return nil
	}
	defer rows.Close()

	var profiles []*ShortcutProfile
	for rows.Next() {
		p, err := scanShortcutProfile(rows)
		if err != nil {
			log.Printf("PGShortcutStore.List: scan failed: %v", err)
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
func (s *PGShortcutStore) Get(id string) (*ShortcutProfile, bool) {
	row := s.db.QueryRow(
		`SELECT id, scope, name, shortcuts, created_at, updated_at FROM shortcut_profiles WHERE id = $1`, id)

	p, err := scanShortcutProfileRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		log.Printf("PGShortcutStore.Get: scan failed: %v", err)
		return nil, false
	}
	return p, true
}

// Upsert creates or updates a shortcut profile. Replay-safe: if the content
// is identical to the existing profile, UpdatedAt is preserved.
func (s *PGShortcutStore) Upsert(id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile {
	shortcutsJSON, err := json.Marshal(shortcuts)
	if err != nil {
		log.Printf("PGShortcutStore.Upsert: marshal shortcuts: %v", err)
		return nil
	}

	now := time.Now().UTC()
	// ON CONFLICT: only update if content changed (replay safety)
	row := s.db.QueryRow(`
		INSERT INTO shortcut_profiles (id, scope, name, shortcuts, created_at, updated_at)
		VALUES ($1, $2::shortcut_scope, $3, $4::jsonb, $5, $5)
		ON CONFLICT (id) DO UPDATE SET
			scope = EXCLUDED.scope,
			name = EXCLUDED.name,
			shortcuts = EXCLUDED.shortcuts,
			updated_at = CASE
				WHEN shortcut_profiles.scope = EXCLUDED.scope
				  AND shortcut_profiles.name = EXCLUDED.name
				  AND shortcut_profiles.shortcuts = EXCLUDED.shortcuts
				THEN shortcut_profiles.updated_at
				ELSE EXCLUDED.updated_at
			END
		RETURNING id, scope, name, shortcuts, created_at, updated_at`,
		id, scope, name, shortcutsJSON, now)

	p, err := scanShortcutProfileRow(row)
	if err != nil {
		log.Printf("PGShortcutStore.Upsert: scan failed: %v", err)
		return nil
	}
	return p
}

// Delete removes a shortcut profile by ID. Returns false if not found.
func (s *PGShortcutStore) Delete(id string) bool {
	result, err := s.db.Exec(`DELETE FROM shortcut_profiles WHERE id = $1`, id)
	if err != nil {
		log.Printf("PGShortcutStore.Delete: exec failed: %v", err)
		return false
	}
	n, _ := result.RowsAffected()
	return n > 0
}

// Effective returns the resolved shortcut list by selecting the
// highest-priority scope's profile.
func (s *PGShortcutStore) Effective() []ShortcutEntry {
	row := s.db.QueryRow(`
		SELECT shortcuts FROM shortcut_profiles
		ORDER BY CASE scope
			WHEN 'parent' THEN 3
			WHEN 'workspace' THEN 2
			WHEN 'service' THEN 1
			ELSE 0
		END DESC
		LIMIT 1`)

	var shortcutsJSON []byte
	if err := row.Scan(&shortcutsJSON); err != nil {
		if err == sql.ErrNoRows {
			return defaultShortcuts
		}
		log.Printf("PGShortcutStore.Effective: scan failed: %v", err)
		return defaultShortcuts
	}

	var shortcuts []ShortcutEntry
	if err := json.Unmarshal(shortcutsJSON, &shortcuts); err != nil {
		log.Printf("PGShortcutStore.Effective: unmarshal failed: %v", err)
		return defaultShortcuts
	}
	return shortcuts
}

// scanShortcutProfile reads a ShortcutProfile from a *sql.Rows cursor.
func scanShortcutProfile(rows *sql.Rows) (*ShortcutProfile, error) {
	var p ShortcutProfile
	var shortcutsJSON []byte
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&p.ID, &p.Scope, &p.Name, &shortcutsJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(shortcutsJSON, &p.Shortcuts); err != nil {
		return nil, err
	}
	p.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	p.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	return &p, nil
}

// scanShortcutProfileRow reads a ShortcutProfile from a *sql.Row.
func scanShortcutProfileRow(row *sql.Row) (*ShortcutProfile, error) {
	var p ShortcutProfile
	var shortcutsJSON []byte
	var createdAt, updatedAt time.Time
	if err := row.Scan(&p.ID, &p.Scope, &p.Name, &shortcutsJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(shortcutsJSON, &p.Shortcuts); err != nil {
		return nil, err
	}
	p.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	p.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	return &p, nil
}
