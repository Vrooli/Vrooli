package database

import (
	"context"
	"database/sql"
	"fmt"
)

// SchemaProvider is implemented by code units that own SQL they want
// applied to a scenario's database at boot. Domain packages embed
// schema.sql via go:embed and return its contents from a Schema()
// function; SchemaProviderFunc adapts that function to the interface
// so the slice in main.go can list providers without each domain
// declaring a wrapper type.
//
// The pattern is: each internal/<dom>/ ships schema.sql (forward-only
// declarative — IF NOT EXISTS / DO blocks for idempotency) and a
// Schema() function. Cross-cutting infrastructure (postgres extensions,
// custom types, cross-domain views) lives in a "system" home (e.g.,
// internal/database/system.sql) which exposes its own SystemSchema().
// EnsureSchemas applies them in registration order; empty schemas skip.
type SchemaProvider interface {
	Schema() string
}

// SchemaProviderFunc adapts a bare func() string to SchemaProvider so
// domain packages can stay free of declaration boilerplate.
type SchemaProviderFunc func() string

func (f SchemaProviderFunc) Schema() string { return f() }

// SchemaExecer is the minimal database surface EnsureSchemas needs.
// *sql.DB satisfies it directly; tests can supply a fake without
// pulling in a real driver.
type SchemaExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Compile-time guarantee that *sql.DB satisfies SchemaExecer.
var _ SchemaExecer = (*sql.DB)(nil)

// EnsureSchemas applies each provider's schema to db in the order given.
// Schemas must be idempotent (CREATE TABLE IF NOT EXISTS, etc.) — each
// boot calls EnsureSchemas and a successful apply must be a no-op on
// the next call.
//
// Empty schemas (Schema() returns "") are skipped silently so a system
// home that has no cross-cutting bits doesn't need a placeholder
// statement.
//
// Errors include the provider's index (1-based) and the underlying
// error. Provider type names are not included because providers are
// often SchemaProviderFunc-wrapped functions whose type is uninformative.
func EnsureSchemas(ctx context.Context, db SchemaExecer, providers ...SchemaProvider) error {
	for i, p := range providers {
		sqlText := p.Schema()
		if sqlText == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, sqlText); err != nil {
			return fmt.Errorf("apply schema provider %d: %w", i+1, err)
		}
	}
	return nil
}
