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
	"fmt"
	"strings"
)

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

// InitSchema initializes the database schema for the agent-inbox scenario.
// This creates all required tables and indexes, and runs any necessary migrations.
func (r *Repository) InitSchema(ctx context.Context) error {
	schemaName := "agent_inbox"

	// Create scenario-specific schema
	if _, err := r.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName)); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Set search_path at database level so all connections use it
	// This is more robust than per-connection SET because connection pools may use different connections
	var dbname string
	if err := r.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbname); err == nil {
		if _, err := r.db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s SET search_path TO %s, public", dbname, schemaName)); err != nil {
			// Ignore error - may not have ALTER DATABASE permission, will use session-level instead
		}
	}

	// Set for current session to ensure this connection works
	if _, err := r.db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName)); err != nil {
		return fmt.Errorf("failed to set session search_path: %w", err)
	}

	// Create base tables
	baseSchema := `
	CREATE TABLE IF NOT EXISTS chats (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL DEFAULT 'New Chat',
		preview TEXT NOT NULL DEFAULT '',
		model TEXT NOT NULL DEFAULT 'anthropic/claude-3.5-sonnet',
		view_mode TEXT NOT NULL DEFAULT 'bubble' CHECK (view_mode IN ('bubble', 'terminal')),
		is_read BOOLEAN NOT NULL DEFAULT false,
		is_archived BOOLEAN NOT NULL DEFAULT false,
		is_starred BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		model TEXT,
		token_count INTEGER DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

	CREATE TABLE IF NOT EXISTS labels (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT '#6366f1',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS chat_labels (
		chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
		label_id UUID NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
		PRIMARY KEY (chat_id, label_id)
	);

	CREATE INDEX IF NOT EXISTS idx_chat_labels_chat_id ON chat_labels(chat_id);
	CREATE INDEX IF NOT EXISTS idx_chat_labels_label_id ON chat_labels(label_id);
	`

	if _, err := r.db.ExecContext(ctx, baseSchema); err != nil {
		return fmt.Errorf("failed to create base schema: %w", err)
	}

	// Run migrations
	migrations := []struct {
		name string
		sql  string
	}{
		{"add chats.system_prompt", `ALTER TABLE chats ADD COLUMN IF NOT EXISTS system_prompt TEXT DEFAULT ''`},
		{"add chats.tools_enabled", `ALTER TABLE chats ADD COLUMN IF NOT EXISTS tools_enabled BOOLEAN DEFAULT true`},
		{"add messages.tool_call_id", `ALTER TABLE messages ADD COLUMN IF NOT EXISTS tool_call_id TEXT`},
		{"add messages.tool_calls", `ALTER TABLE messages ADD COLUMN IF NOT EXISTS tool_calls JSONB`},
		{"add messages.response_id", `ALTER TABLE messages ADD COLUMN IF NOT EXISTS response_id TEXT`},
		{"add messages.finish_reason", `ALTER TABLE messages ADD COLUMN IF NOT EXISTS finish_reason TEXT`},
		{"create idx_messages_tool_call_id", `CREATE INDEX IF NOT EXISTS idx_messages_tool_call_id ON messages(tool_call_id) WHERE tool_call_id IS NOT NULL`},
		{"add messages.search_vector", `ALTER TABLE messages ADD COLUMN IF NOT EXISTS search_vector tsvector`},
		{"create idx_messages_search", `CREATE INDEX IF NOT EXISTS idx_messages_search ON messages USING gin(search_vector)`},
		{"add chats.search_vector", `ALTER TABLE chats ADD COLUMN IF NOT EXISTS search_vector tsvector`},
		{"create idx_chats_search", `CREATE INDEX IF NOT EXISTS idx_chats_search ON chats USING gin(search_vector)`},
		{"create search update trigger function", `
			CREATE OR REPLACE FUNCTION update_message_search_vector() RETURNS trigger AS $$
			BEGIN
				NEW.search_vector := to_tsvector('english', COALESCE(NEW.content, ''));
				RETURN NEW;
			END
			$$ LANGUAGE plpgsql`},
		{"create message search trigger", `
			DO $$
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'messages_search_update') THEN
					CREATE TRIGGER messages_search_update
					BEFORE INSERT OR UPDATE ON messages
					FOR EACH ROW EXECUTE FUNCTION update_message_search_vector();
				END IF;
			END
			$$`},
		{"create chat search update trigger function", `
			CREATE OR REPLACE FUNCTION update_chat_search_vector() RETURNS trigger AS $$
			BEGIN
				NEW.search_vector := to_tsvector('english', COALESCE(NEW.name, ''));
				RETURN NEW;
			END
			$$ LANGUAGE plpgsql`},
		{"create chat search trigger", `
			DO $$
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'chats_search_update') THEN
					CREATE TRIGGER chats_search_update
					BEFORE INSERT OR UPDATE ON chats
					FOR EACH ROW EXECUTE FUNCTION update_chat_search_vector();
				END IF;
			END
			$$`},
		{"backfill message search vectors", `UPDATE messages SET search_vector = to_tsvector('english', COALESCE(content, '')) WHERE search_vector IS NULL`},
		{"backfill chat search vectors", `UPDATE chats SET search_vector = to_tsvector('english', COALESCE(name, '')) WHERE search_vector IS NULL`},
		{"create tool_calls table", `
			CREATE TABLE IF NOT EXISTS tool_calls (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
				chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
				tool_name TEXT NOT NULL,
				arguments JSONB NOT NULL DEFAULT '{}',
				result JSONB,
				status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
				scenario_name TEXT,
				external_run_id TEXT,
				started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				completed_at TIMESTAMPTZ,
				error_message TEXT
			)`},
		{"create idx_tool_calls_message_id", `CREATE INDEX IF NOT EXISTS idx_tool_calls_message_id ON tool_calls(message_id)`},
		{"create idx_tool_calls_chat_id", `CREATE INDEX IF NOT EXISTS idx_tool_calls_chat_id ON tool_calls(chat_id)`},
		{"create idx_tool_calls_status", `CREATE INDEX IF NOT EXISTS idx_tool_calls_status ON tool_calls(status) WHERE status IN ('pending', 'running')`},
		{"create usage_records table", `
			CREATE TABLE IF NOT EXISTS usage_records (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
				message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
				model TEXT NOT NULL,
				prompt_tokens INTEGER NOT NULL DEFAULT 0,
				completion_tokens INTEGER NOT NULL DEFAULT 0,
				total_tokens INTEGER NOT NULL DEFAULT 0,
				prompt_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
				completion_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
				total_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`},
		{"create idx_usage_records_chat_id", `CREATE INDEX IF NOT EXISTS idx_usage_records_chat_id ON usage_records(chat_id)`},
		{"create idx_usage_records_created_at", `CREATE INDEX IF NOT EXISTS idx_usage_records_created_at ON usage_records(created_at)`},
		{"create idx_usage_records_model", `CREATE INDEX IF NOT EXISTS idx_usage_records_model ON usage_records(model)`},
	}

	for _, m := range migrations {
		if _, err := r.db.ExecContext(ctx, m.sql); err != nil {
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "duplicate") {
				return fmt.Errorf("migration %q failed: %w", m.name, err)
			}
		}
	}

	return nil
}

// Helper functions

func parsePostgresArray(arr string) []string {
	arr = strings.Trim(arr, "{}")
	if arr == "" {
		return []string{}
	}
	return strings.Split(arr, ",")
}
