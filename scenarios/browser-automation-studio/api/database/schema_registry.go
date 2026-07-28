package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	coredb "github.com/vrooli/api-core/database"
)

type schemaExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// SchemaDefinition names one product-owned SQLite schema source. Order is
// deliberate: later domains may reference tables created by earlier domains,
// while triggers must be installed only after every owning table exists.
type SchemaDefinition struct {
	Domain string
	File   string
}

// SchemaRegistry is the canonical database-domain ownership map. Schema files
// contain only one product domain each; shared initialization infrastructure
// owns their ordering and execution, not their table definitions.
var SchemaRegistry = []SchemaDefinition{
	{Domain: "core", File: "core.sql"},
	{Domain: "recording", File: "recording.sql"},
	{Domain: "billing", File: "billing.sql"},
	{Domain: "uxmetrics", File: "uxmetrics.sql"},
	{Domain: "lifecycle", File: "triggers.sql"},
}

func loadSchemaStatements(scenarioRoot string) ([]SchemaDefinition, [][]byte, error) {
	schemaRoot := filepath.Join(scenarioRoot, "initialization", "storage", "sqlite", "schemas")
	statements := make([][]byte, 0, len(SchemaRegistry))
	for _, schema := range SchemaRegistry {
		contents, err := os.ReadFile(filepath.Join(schemaRoot, schema.File))
		if err != nil {
			return nil, nil, fmt.Errorf("read %s schema %q: %w", schema.Domain, schema.File, err)
		}
		statements = append(statements, contents)
	}
	return SchemaRegistry, statements, nil
}

// SchemaProviders exposes the ordered BAS schema registry through api-core's
// canonical bootstrap contract. The same domain-owned SQL files remain the
// source of truth for primary and leased routed pools.
func SchemaProviders(scenarioRoot string) ([]coredb.SchemaProvider, error) {
	_, statements, err := loadSchemaStatements(scenarioRoot)
	if err != nil {
		return nil, err
	}
	providers := make([]coredb.SchemaProvider, 0, len(statements))
	for _, statement := range statements {
		schema := string(statement)
		providers = append(providers, coredb.SchemaProviderFunc(func() string { return schema }))
	}
	return providers, nil
}

// ApplySchemaRegistry initializes every domain schema in dependency order.
// It is shared by production startup and focused repository tests so they
// cannot drift to a private, incomplete schema setup.
func ApplySchemaRegistry(ctx context.Context, executor schemaExecutor, scenarioRoot string) error {
	schemas, statements, err := loadSchemaStatements(scenarioRoot)
	if err != nil {
		return err
	}
	for index, schema := range schemas {
		if _, err := executor.ExecContext(ctx, string(statements[index])); err != nil {
			return fmt.Errorf("initialize %s schema: %w", schema.Domain, err)
		}
	}
	return nil
}
