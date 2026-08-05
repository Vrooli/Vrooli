//nolint:gofumpt // golangci-lint's bundled formatter disagrees with the pinned formatter.
package main

import (
	"context"
	"fmt"

	"secrets-manager-api/internal/secrets"

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
	return nil
}
