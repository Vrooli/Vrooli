//nolint:gofumpt // golangci-lint's bundled formatter disagrees with the pinned formatter.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"secrets-manager-api/internal/envx"

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
	schemaPath, err := postgresSchemaPath()
	if err != nil {
		return err
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read PostgreSQL schema %s: %w", schemaPath, err)
	}
	return database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(func() string {
		return string(schema)
	}))
}

func postgresSchemaPath() (string, error) {
	scenarioDir, err := optionalScenarioDirectory(envx.OS{})
	if err != nil {
		return "", err
	}
	if scenarioDir != "" {
		return filepath.Join(scenarioDir, "initialization", "storage", "postgres", "schema.sql"), nil
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory for PostgreSQL schema: %w", err)
	}
	return filepath.Join(filepath.Dir(workingDir), "initialization", "storage", "postgres", "schema.sql"), nil
}
