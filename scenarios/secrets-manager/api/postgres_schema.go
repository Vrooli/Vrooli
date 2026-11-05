//nolint:gofumpt // golangci-lint's bundled formatter disagrees with the pinned formatter.
package main

import (
	"context"
	"fmt"

	"secrets-manager-api/internal/secrets"
	"secrets-manager-api/internal/vault"

	"github.com/vrooli/api-core/database"
)

// ensurePostgresSchema applies the scenario-owned PostgreSQL schema through
// the same api-core seam used by the routed test-pool contract. This boot-time
// application makes schema availability deterministic for normal starts and
// isolated test pools.
func ensurePostgresSchema(ctx context.Context, db *database.RoutedDB) error {
	if db == nil {
		return fmt.Errorf("database is required for PostgreSQL schema initialization")
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(secrets.Schema)); err != nil {
		return err
	}
	if _, err := db.Primary().ExecContext(ctx, secrets.ResourceSecretMetadataMigration()); err != nil {
		return fmt.Errorf("apply resource secret metadata migration: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, secrets.CredentialAuthorityStorageMigration()); err != nil {
		return fmt.Errorf("apply credential authority storage migration: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, vault.Schema()); err != nil {
		return fmt.Errorf("apply password-manager vault schema: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, `ALTER TABLE pm_broker_sessions ADD COLUMN IF NOT EXISTS resolved_ips TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("migrate broker session address pinning: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, `ALTER TABLE pm_broker_sessions ADD COLUMN IF NOT EXISTS one_use BOOLEAN NOT NULL DEFAULT FALSE`); err != nil {
		return fmt.Errorf("migrate broker one-use sessions: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, `ALTER TABLE pm_broker_sessions ADD COLUMN IF NOT EXISTS use_count INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("migrate broker session use count: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, `ALTER TABLE pm_broker_sessions ADD COLUMN IF NOT EXISTS run_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("migrate broker run binding: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, `ALTER TABLE pm_broker_sessions ADD COLUMN IF NOT EXISTS recovery_epoch BIGINT NOT NULL DEFAULT 1`); err != nil {
		return fmt.Errorf("migrate broker recovery epoch: %w", err)
	}
	if _, err := db.Primary().ExecContext(ctx, `ALTER TABLE pm_grants ADD COLUMN IF NOT EXISTS parent_grant_id TEXT`); err != nil {
		return fmt.Errorf("migrate grant parent binding: %w", err)
	}
	for _, statement := range []string{
		`ALTER TABLE pm_audit_events ADD COLUMN IF NOT EXISTS correlation_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pm_audit_events ADD COLUMN IF NOT EXISTS event_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pm_audit_events ADD COLUMN IF NOT EXISTS decision TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pm_audit_events ADD COLUMN IF NOT EXISTS destination TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pm_audit_events ADD COLUMN IF NOT EXISTS integrity_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pm_audit_events ADD COLUMN IF NOT EXISTS integrity_status TEXT NOT NULL DEFAULT 'unknown'`,
		`UPDATE pm_audit_events SET event_key=id WHERE event_key=''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS pm_audit_events_workspace_event_key_idx ON pm_audit_events(workspace_id,event_key)`,
	} {
		if _, err := db.Primary().ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate audit integrity metadata: %w", err)
		}
	}
	return nil
}
