package registry

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the registry domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-
// only declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
//
// ⚠ Adding a column to an EXISTING table is NOT free under SQLite. Putting a
// new column only in CREATE TABLE IF NOT EXISTS silently skips it on any DB
// created before the column existed (the statement is a no-op when the table
// already exists) — exactly the bug that once broke all provider registration
// when control_token was added. To change the columns of an existing table,
// apply a one-shot migration that brings the existing DB to the declared shape
// (storage-steer §5: write /tmp/search-hub/migrate-*.sql with the
// `ALTER TABLE providers ADD COLUMN ...`, run it with the scenario stopped,
// then delete it). Do NOT recreate the DB. EnsureSchemas now runs a post-apply
// drift check (PRAGMA table_info vs the declared columns) and fails boot loudly
// if a declared column is missing, so a forgotten migration can no longer ship
// silently. See docs/internal/PROBLEMS.md "registry schema additive-column
// migration gap".
func Schema() string { return schemaSQL }
