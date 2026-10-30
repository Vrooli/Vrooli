package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// AgentPreferences belongs to one effective shortcut profile; empty ProfileID
// identifies the built-in fallback. IDs are stable capability IDs, not labels
// or commands. Missing IDs default to visible and follow the ordered IDs.
type AgentPreferences struct {
	ProfileID       string
	ProfileInstance string
	OrderedAgentIDs []string
	HiddenAgentIDs  []string
	Revision        uint64
}

var ErrAgentPreferencesConflict = errors.New("shortcut agent preferences changed; refresh and retry")
var ErrAgentPreferencesInvalid = errors.New("invalid shortcut agent preferences")

func preferenceWriteError(err error) error {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && (sqliteErr.Code()&255 == sqlite3.SQLITE_BUSY || sqliteErr.Code()&255 == sqlite3.SQLITE_LOCKED) {
		return ErrAgentPreferencesConflict
	}
	return err
}

func validateAgentPreferences(p AgentPreferences) error {
	for _, ids := range [][]string{p.OrderedAgentIDs, p.HiddenAgentIDs} {
		if len(ids) > 256 {
			return fmt.Errorf("%w: too many agent IDs", ErrAgentPreferencesInvalid)
		}
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			if id == "" || strings.TrimSpace(id) != id || len(id) > 128 || seen[id] {
				return fmt.Errorf("%w: agent IDs must be nonempty, unique and at most 128 characters", ErrAgentPreferencesInvalid)
			}
			seen[id] = true
		}
	}
	return nil
}

func cloneAgentPreferences(p AgentPreferences) AgentPreferences {
	p.OrderedAgentIDs = append([]string{}, p.OrderedAgentIDs...)
	p.HiddenAgentIDs = append([]string{}, p.HiddenAgentIDs...)
	return p
}

func (s *ShortcutProfileStore) AgentPreferences(_ context.Context) (AgentPreferences, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id := s.effectiveLocked().ProfileID
	p := s.preferences[id]
	p.ProfileID = id
	if id != "" {
		p.ProfileInstance = s.profiles[id].CreatedAt
	}
	return cloneAgentPreferences(p), nil
}

func (s *ShortcutProfileStore) UpdateAgentPreferences(_ context.Context, p AgentPreferences, expected uint64) (AgentPreferences, error) {
	if expected >= math.MaxInt64 {
		return AgentPreferences{}, ErrAgentPreferencesInvalid
	}
	if err := validateAgentPreferences(p); err != nil {
		return AgentPreferences{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.effectiveLocked().ProfileID
	instance := ""
	if id != "" {
		instance = s.profiles[id].CreatedAt
	}
	if id != p.ProfileID || instance != p.ProfileInstance || s.preferences[p.ProfileID].Revision != expected {
		return AgentPreferences{}, ErrAgentPreferencesConflict
	}
	p.Revision = expected + 1
	p = cloneAgentPreferences(p)
	s.preferences[p.ProfileID] = p
	return cloneAgentPreferences(p), nil
}

// effectiveProfileID uses the same priority and deterministic tie-break as
// Effective, inside the same transaction as the preference read or write.
func effectiveProfileID(ctx context.Context, tx *sql.Tx) (string, string, error) {
	var id, instance string
	err := tx.QueryRowContext(ctx, `SELECT id, created_at FROM shortcut_profiles ORDER BY CASE scope WHEN 'parent' THEN 3 WHEN 'workspace' THEN 2 WHEN 'service' THEN 1 ELSE 0 END DESC, id ASC LIMIT 1`).Scan(&id, &instance)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	return id, instance, err
}

func readAgentPreferences(ctx context.Context, tx *sql.Tx, id string) (AgentPreferences, error) {
	p := AgentPreferences{ProfileID: id, OrderedAgentIDs: []string{}, HiddenAgentIDs: []string{}}
	var ordered, hidden string
	err := tx.QueryRowContext(ctx, `SELECT ordered_agent_ids, hidden_agent_ids, revision FROM shortcut_agent_preferences WHERE profile_id = ?`, id).Scan(&ordered, &hidden, &p.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return AgentPreferences{}, err
	}
	if err = json.Unmarshal([]byte(ordered), &p.OrderedAgentIDs); err != nil {
		return AgentPreferences{}, err
	}
	if err = json.Unmarshal([]byte(hidden), &p.HiddenAgentIDs); err != nil {
		return AgentPreferences{}, err
	}
	return p, nil
}

func (s *SQLShortcutStore) AgentPreferences(ctx context.Context) (AgentPreferences, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return AgentPreferences{}, err
	}
	defer tx.Rollback()
	id, instance, err := effectiveProfileID(ctx, tx)
	if err != nil {
		return AgentPreferences{}, err
	}
	p, err := readAgentPreferences(ctx, tx, id)
	if err != nil {
		return AgentPreferences{}, err
	}
	if err = tx.Commit(); err != nil {
		return AgentPreferences{}, err
	}
	p.ProfileInstance = instance
	return p, nil
}

func (s *SQLShortcutStore) UpdateAgentPreferences(ctx context.Context, p AgentPreferences, expected uint64) (AgentPreferences, error) {
	if expected >= math.MaxInt64 {
		return AgentPreferences{}, ErrAgentPreferencesInvalid
	}
	if err := validateAgentPreferences(p); err != nil {
		return AgentPreferences{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AgentPreferences{}, err
	}
	defer tx.Rollback()
	id, instance, err := effectiveProfileID(ctx, tx)
	if err != nil {
		return AgentPreferences{}, err
	}
	if id != p.ProfileID || instance != p.ProfileInstance {
		return AgentPreferences{}, ErrAgentPreferencesConflict
	}
	current, err := readAgentPreferences(ctx, tx, id)
	if err != nil {
		return AgentPreferences{}, err
	}
	if current.Revision != expected {
		return AgentPreferences{}, ErrAgentPreferencesConflict
	}
	ordered, err := json.Marshal(append([]string{}, p.OrderedAgentIDs...))
	if err != nil {
		return AgentPreferences{}, err
	}
	hidden, err := json.Marshal(append([]string{}, p.HiddenAgentIDs...))
	if err != nil {
		return AgentPreferences{}, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO shortcut_agent_preferences (profile_id, ordered_agent_ids, hidden_agent_ids, revision) VALUES (?, ?, ?, 1) ON CONFLICT(profile_id) DO UPDATE SET ordered_agent_ids = excluded.ordered_agent_ids, hidden_agent_ids = excluded.hidden_agent_ids, revision = shortcut_agent_preferences.revision + 1 WHERE shortcut_agent_preferences.revision = ?`, id, string(ordered), string(hidden), expected)
	if err != nil {
		return AgentPreferences{}, preferenceWriteError(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return AgentPreferences{}, err
	}
	if n != 1 {
		return AgentPreferences{}, ErrAgentPreferencesConflict
	}
	p.Revision = expected + 1
	if err := tx.Commit(); err != nil {
		return AgentPreferences{}, preferenceWriteError(err)
	}
	return cloneAgentPreferences(p), nil
}
