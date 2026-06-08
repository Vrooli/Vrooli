package eval

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the eval domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-only
// declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
//
// ⚠ Adding a column to an EXISTING table is NOT idempotent under SQLite (no
// `ADD COLUMN IF NOT EXISTS`; a duplicate `ADD COLUMN` errors) and EnsureSchemas
// execs this file as one statement with no error tolerance — so an added column
// only reaches a freshly-created DB, not one created before the column existed.
// See internal/registry/schema.go and docs/internal/PROBLEMS.md "registry schema
// additive-column migration gap"; the brownfield additive-migration path is the
// real fix (escalate per storage-steer §6) — until then recreate the local DB.
func Schema() string { return schemaSQL }
