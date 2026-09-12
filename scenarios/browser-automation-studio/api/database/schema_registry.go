package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	coredb "github.com/vrooli/api-core/database"
	billingSchema "github.com/vrooli/browser-automation-studio/internal/billing"
	coreSchema "github.com/vrooli/browser-automation-studio/internal/core"
	lifecycleSchema "github.com/vrooli/browser-automation-studio/internal/lifecycle"
	recordingSchema "github.com/vrooli/browser-automation-studio/internal/recording"
	uxmetricsSchema "github.com/vrooli/browser-automation-studio/internal/uxmetrics"
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
	{Domain: "core", File: "core/schema.sql"},
	{Domain: "recording", File: "recording/schema.sql"},
	{Domain: "billing", File: "billing/schema.sql"},
	{Domain: "uxmetrics", File: "uxmetrics/schema.sql"},
	{Domain: "lifecycle", File: "lifecycle/schema.sql"},
}

func loadSchemaStatements(scenarioRoot string) ([]SchemaDefinition, [][]byte, error) {
	schemaRoot := filepath.Join(scenarioRoot, "api", "internal")
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
	_ = scenarioRoot
	return []coredb.SchemaProvider{
		coredb.SchemaProviderFunc(coreSchema.Schema),
		coredb.SchemaProviderFunc(recordingSchema.Schema),
		coredb.SchemaProviderFunc(billingSchema.Schema),
		coredb.SchemaProviderFunc(uxmetricsSchema.Schema),
		coredb.SchemaProviderFunc(lifecycleSchema.Schema),
	}, nil
}

// ApplySchemaRegistry initializes every domain schema in dependency order.
// It is shared by production startup and focused repository tests so they
// cannot drift to a private, incomplete schema setup.
func ApplySchemaRegistry(ctx context.Context, executor schemaExecutor, scenarioRoot string) error {
	providers, err := SchemaProviders(scenarioRoot)
	if err != nil {
		return err
	}
	for index, schema := range SchemaRegistry {
		if _, err := executor.ExecContext(ctx, providers[index].Schema()); err != nil {
			return fmt.Errorf("initialize %s schema: %w", schema.Domain, err)
		}
	}
	return nil
}
