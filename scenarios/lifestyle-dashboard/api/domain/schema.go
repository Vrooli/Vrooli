// DOC: docs/internal/STORAGE_AUDIT.md#Schema-Status
// DOC: README.md#Event-Schema
// DOC: PRD.md#OT-P0-001
//
// Package domain contains the core business entities for the lifestyle dashboard.
package domain

import "database/sql"

// Schema contains the SQLite schema DDL for the lifestyle dashboard.
// This is idempotent and safe to run multiple times.
const Schema = `
-- Events table (P0-001): Common envelope with JSON payloads
CREATE TABLE IF NOT EXISTS events (
	id TEXT PRIMARY KEY,
	timestamp TEXT NOT NULL,
	domain TEXT NOT NULL,
	event_type TEXT NOT NULL,
	payload TEXT DEFAULT '{}',
	is_intervention INTEGER DEFAULT 0,
	hypothesis_id TEXT,
	created_at TEXT NOT NULL
);

-- Indexes for efficient cross-domain queries (P0-003)
CREATE INDEX IF NOT EXISTS idx_events_domain_timestamp ON events(domain, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_hypothesis ON events(hypothesis_id) WHERE hypothesis_id IS NOT NULL;

-- Domains table (P0-002): Domain registration and discovery
CREATE TABLE IF NOT EXISTS domains (
	name TEXT PRIMARY KEY,
	display_name TEXT NOT NULL,
	description TEXT DEFAULT '',
	capabilities TEXT DEFAULT '[]',
	status TEXT DEFAULT 'active',
	health_url TEXT DEFAULT '',
	last_health_at TEXT,
	registered_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_domains_status ON domains(status);

-- Domain weights table (P1-003): Configurable weights for lifestyle score
-- [REQ:LD-SCORE-CALC] Stores user-configurable domain weights.
CREATE TABLE IF NOT EXISTS domain_weights (
	domain TEXT PRIMARY KEY,
	weight TEXT NOT NULL DEFAULT 'medium',
	updated_at TEXT NOT NULL,
	FOREIGN KEY (domain) REFERENCES domains(name) ON DELETE CASCADE
);
`

// InitSchema creates all required tables and indexes in the database.
// It is idempotent and safe to call on every startup.
func InitSchema(db *sql.DB) error {
	_, err := db.Exec(Schema)
	return err
}
