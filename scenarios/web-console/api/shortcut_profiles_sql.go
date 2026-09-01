// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"web-console/internal/dbx"
)

// [REQ:P1-002a] Shortcut Profile Storage — SQLite implementation

// SQLShortcutStore persists shortcut profiles in SQLite.
// It implements ShortcutStore.
type SQLShortcutStore struct {
	db dbx.Handle
}

// NewSQLShortcutStore creates a SQLite-backed shortcut store.
func NewSQLShortcutStore(db dbx.Handle) *SQLShortcutStore {
	return &SQLShortcutStore{db: db}
}

// List returns all shortcut profiles from the database.
func (s *SQLShortcutStore) List(ctx context.Context) []*ShortcutProfile {
	rows, err := s.db.QueryContext(ctx, `SELECT id, scope, name, shortcuts, created_at, updated_at FROM shortcut_profiles ORDER BY id`)
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
func (s *SQLShortcutStore) Get(ctx context.Context, id string) (*ShortcutProfile, bool) {
	row := s.db.QueryRowContext(ctx,
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
func (s *SQLShortcutStore) Upsert(ctx context.Context, id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile {
	shortcuts = normalizeShortcutEntries(shortcuts)
	shortcutsJSON, err := json.Marshal(shortcuts)
	if err != nil {
		log.Printf("SQLShortcutStore.Upsert: marshal shortcuts: %v", err)
		return nil
	}

	now := formatTime(time.Now())
	// ON CONFLICT: only update if content changed (replay safety)
	row := s.db.QueryRowContext(ctx, `
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
func (s *SQLShortcutStore) Delete(ctx context.Context, id string) bool {
	result, err := s.db.ExecContext(ctx, `DELETE FROM shortcut_profiles WHERE id = ?`, id)
	if err != nil {
		log.Printf("SQLShortcutStore.Delete: exec failed: %v", err)
		return false
	}
	n, _ := result.RowsAffected()
	return n > 0
}

// Effective returns the resolved shortcut list by selecting the
// highest-priority scope's profile, together with that profile's identity so
// a client can write an edited order back to the row it actually came from.
func (s *SQLShortcutStore) Effective(ctx context.Context) EffectiveShortcuts {
	fallback := EffectiveShortcuts{Shortcuts: normalizeShortcutEntries(defaultShortcuts)}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, scope, name, shortcuts FROM shortcut_profiles
		ORDER BY CASE scope
			WHEN 'parent' THEN 3
			WHEN 'workspace' THEN 2
			WHEN 'service' THEN 1
			ELSE 0
		END DESC
		LIMIT 1`)

	var id, scope, name, shortcutsJSON string
	if err := row.Scan(&id, &scope, &name, &shortcutsJSON); err != nil {
		if err == sql.ErrNoRows {
			return fallback
		}
		log.Printf("SQLShortcutStore.Effective: scan failed: %v", err)
		return fallback
	}

	var shortcuts []ShortcutEntry
	if err := json.Unmarshal([]byte(shortcutsJSON), &shortcuts); err != nil {
		log.Printf("SQLShortcutStore.Effective: unmarshal failed: %v", err)
		return fallback
	}
	return EffectiveShortcuts{ProfileID: id, Scope: scope, Name: name, Shortcuts: normalizeShortcutEntries(shortcuts)}
}

// reconcileDefaultShortcutProfile refreshes a stale persisted "default" service
// profile to the current built-in defaults — but only when the stored content is
// provably the unmodified seed.
//
// Background: older builds wrote a "default" service profile row into SQLite at
// boot. That seeding was since removed (fresh DBs now have no row and fall back
// to defaultShortcuts in Effective()), but DBs created under the old code still
// carry the persisted row, which shadows the Go-side default and never picks up
// newly added shortcuts (e.g. OpenCode/Grok).
//
// This is a value-preserving data migration, not a destructive one:
//   - No "default" row → no-op (fresh DBs use the Go fallback).
//   - Row present but its shortcuts differ from every known legacy default →
//     left untouched (a user customization, or already current/idempotent).
//   - Row present and equal to a known legacy default → updated in place to the
//     current defaults (the row stays; only its shortcuts/updated_at change).
//
// It never deletes a row and never touches a customized profile, so no user data
// is lost. Idempotent: after the update the content no longer matches a legacy
// default, so a re-run is a no-op.
func reconcileDefaultShortcutProfile(ctx context.Context, db dbx.Handle) error {
	var storedJSON string
	err := db.QueryRowContext(ctx,
		`SELECT shortcuts FROM shortcut_profiles WHERE id = 'default' AND scope = 'service'`,
	).Scan(&storedJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // fresh DB: Effective() uses the Go default
		}
		return err
	}

	var stored []ShortcutEntry
	if err := json.Unmarshal([]byte(storedJSON), &stored); err != nil {
		// Unparseable content is not a recognizable seed; leave it alone.
		log.Printf("reconcileDefaultShortcutProfile: unparseable stored shortcuts, leaving untouched: %v", err)
		return nil
	}

	if !shortcutsMatchAnyLegacyDefault(stored) {
		return nil // customized or already current — preserve as-is
	}

	current, err := json.Marshal(defaultShortcuts)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE shortcut_profiles SET shortcuts = ?, updated_at = ? WHERE id = 'default' AND scope = 'service'`,
		string(current), formatTime(time.Now()),
	); err != nil {
		return err
	}
	log.Printf("migration: refreshed unmodified default shortcut profile to current built-ins (%d entries)", len(defaultShortcuts))
	return nil
}

// shortcutsMatchAnyLegacyDefault reports whether the given shortcuts exactly
// equal one of the known prior built-in default lists (order-sensitive,
// field-for-field) — the test for "this is the untouched seed".
func shortcutsMatchAnyLegacyDefault(got []ShortcutEntry) bool {
	for _, legacy := range legacyDefaultShortcutSets {
		if shortcutsEqual(got, legacy) {
			return true
		}
	}
	return false
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
