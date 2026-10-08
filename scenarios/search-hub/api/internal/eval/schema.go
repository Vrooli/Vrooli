package eval

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the eval domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-only
// declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
//
// ⚠ Adding a column to an EXISTING table is NOT free under SQLite: a column
// added only in CREATE TABLE IF NOT EXISTS reaches a freshly-created DB but
// silently never lands on one created before the column existed. To change the
// columns of an existing table, apply a one-shot migration (storage-steer §5:
// /tmp/search-hub/migrate-*.sql with the `ALTER TABLE ... ADD COLUMN`, run with
// the scenario stopped, then delete it) — do NOT recreate the DB. EnsureSchemas
// runs a post-apply drift check that fails boot loudly if a declared column is
// missing. See internal/registry/schema.go and docs/internal/PROBLEMS.md
// "registry schema additive-column migration gap".
func Schema() string { return schemaSQL }
