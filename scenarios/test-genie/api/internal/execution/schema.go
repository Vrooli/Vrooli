package execution

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns declarative DDL for the compact execution headers and
// normalized phase-history projection. It is idempotent (CREATE TABLE IF NOT
// EXISTS) and safe to apply on every boot. Existing databases are evolved by
// Migrate; legacy result documents are never read by runtime code.
func Schema() string { return schemaSQL }
