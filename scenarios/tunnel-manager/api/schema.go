package main

import (
	"database/sql"
	"log"
)

// ensureSchema creates the database tables if they don't exist.
// All statements use IF NOT EXISTS for idempotent runs.
func ensureSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS routes (
			id            SERIAL PRIMARY KEY,
			subdomain     TEXT NOT NULL UNIQUE,
			scenario_name TEXT NOT NULL,
			local_port    INT  NOT NULL,
			health_path   TEXT NOT NULL DEFAULT '/health',
			public_url    TEXT NOT NULL DEFAULT '',
			enabled       BOOLEAN NOT NULL DEFAULT TRUE,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS probe_results (
			id           SERIAL PRIMARY KEY,
			route_id     INT  NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
			probe_type   TEXT NOT NULL CHECK (probe_type IN ('internal', 'external')),
			status       TEXT NOT NULL CHECK (status IN ('up', 'down', 'timeout', 'error')),
			latency_ms   INT,
			status_code  INT,
			error_msg    TEXT,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS recovery_events (
			id           SERIAL PRIMARY KEY,
			trigger_type TEXT NOT NULL,
			action       TEXT NOT NULL,
			outcome      TEXT NOT NULL CHECK (outcome IN ('success', 'failure', 'skipped')),
			details      TEXT,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_probe_results_route_id ON probe_results(route_id)`,
		`CREATE INDEX IF NOT EXISTS idx_probe_results_created_at ON probe_results(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_recovery_events_created_at ON recovery_events(created_at)`,
		`CREATE TABLE IF NOT EXISTS metrics_history (
			id              SERIAL PRIMARY KEY,
			ha_connections  INT     NOT NULL DEFAULT 0,
			request_errors  FLOAT8  NOT NULL DEFAULT 0,
			active_streams  INT     NOT NULL DEFAULT 0,
			smoothed_rtt_ms FLOAT8  NOT NULL DEFAULT 0,
			scraped_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_history_scraped_at ON metrics_history(scraped_at)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("schema migration failed: %v\nSQL: %s", err, stmt)
		}
	}
	log.Println("schema: all tables ensured")
}
