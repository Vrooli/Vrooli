package registry

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the registry domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-
// only declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS).
//
// ⚠ Adding a column to an EXISTING table is NOT idempotent under SQLite:
// `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` is a syntax error, and a plain
// `ADD COLUMN` on a table that already has the column fails with "duplicate
// column name" — and EnsureSchemas execs this file as one statement with no
// per-statement error tolerance, so either form crashes boot. Putting the
// column only in CREATE TABLE (the control_token case) silently skips it on any
// DB created before the column existed (CREATE TABLE IF NOT EXISTS is a no-op),
// which is exactly the bug that broke all provider registration once
// control_token was added. Until the brownfield additive-migration path lands
// (see docs/internal/PROBLEMS.md "registry schema additive-column migration
// gap"), adding a column requires recreating the local SQLite DB — it rebuilds
// from self-registration. Escalate per storage-steer §6 rather than hand-rolling
// an ALTER here.
func Schema() string { return schemaSQL }
