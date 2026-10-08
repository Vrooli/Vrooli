// Package persistence provides database operations for the Agent Inbox scenario.
// This package centralizes all database access, providing a clean seam for testing
// and potential database abstraction.
//
// File organization by aggregate:
//   - repository.go: Base repository and schema initialization
//   - chat.go: Chat and message operations
//   - label.go: Label operations
//   - tool_call.go: Tool call record operations
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	core "agent-inbox/internal/core"

	coredb "github.com/vrooli/api-core/database"
)

// =============================================================================
// SQLite Time Scanning Helpers
// =============================================================================

// scanTime returns a scanner that parses SQLite TEXT timestamps into *time.Time.
func scanTime(dest *time.Time) *sqliteTime {
	return (*sqliteTime)(dest)
}

type sqliteTime time.Time

func (t *sqliteTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		parsed, err := parseTimestamp(v)
		if err != nil {
			return err
		}
		*(*time.Time)(t) = parsed
		return nil
	case time.Time:
		*(*time.Time)(t) = v
		return nil
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan %T into time.Time", src)
	}
}

// sqliteNullTime handles nullable timestamps from SQLite TEXT columns.
// Drop-in replacement for sql.NullTime when using modernc.org/sqlite.
type sqliteNullTime struct {
	Time  time.Time
	Valid bool
}

func (t *sqliteNullTime) Scan(src interface{}) error {
	if src == nil {
		t.Valid = false
		return nil
	}
	t.Valid = true
	switch v := src.(type) {
	case string:
		parsed, err := parseTimestamp(v)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	case time.Time:
		t.Time = v
		return nil
	default:
		return fmt.Errorf("cannot scan %T into sqliteNullTime", src)
	}
}

// parseTimestamp parses common SQLite datetime text formats.
func parseTimestamp(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp: %q", s)
}

// Repository provides database operations for the inbox domain.
// All database access flows through this interface, enabling test doubles
// and potential database abstraction.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository with the given database connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying database connection for direct access when needed.
func (r *Repository) DB() *sql.DB {
	return r.db
}

// InitSchema initializes the database schema by executing schema.sql.
func (r *Repository) InitSchema(ctx context.Context) error {
	if err := coredb.EnsureSchemas(ctx, r.db, coredb.SchemaProviderFunc(core.Schema)); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}
	return nil
}

// newID generates a new UUID string for use as a primary key.
func newID() string {
	return uuid.New().String()
}

// parseArrayString parses a comma-separated string or JSON array into a string slice.
func parseArrayString(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return []string{}
	}
	// Handle JSON array format
	if strings.HasPrefix(s, "[") {
		var result []string
		if json.Unmarshal([]byte(s), &result) == nil {
			return result
		}
	}
	// Handle comma-separated format (from GROUP_CONCAT)
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
