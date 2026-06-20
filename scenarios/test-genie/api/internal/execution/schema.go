package execution

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the declarative DDL for the suite_executions table owned by the
// execution domain. It is idempotent (CREATE TABLE IF NOT EXISTS) and safe to
// apply on every boot. Existing (pre-terminal_outcome) databases are evolved by
// Migrate, never recreated.
func Schema() string { return schemaSQL }
