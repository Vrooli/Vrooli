package database

import _ "embed"

//go:embed system.sql
var systemSchemaSQL string

// SystemSchema returns the cross-cutting SQL that doesn't belong to any
// one domain — postgres extensions, custom types, cross-domain views,
// the schema_migrations table once brownfield migrations land.
//
// SQLite-backed scenarios (like the template) ship system.sql empty.
// Postgres scenarios commonly add `CREATE EXTENSION IF NOT EXISTS
// "uuid-ossp"` and similar preamble. Empty returns are fine —
// EnsureSchemas skips them.
//
// Tables owned by a single domain do NOT belong here. They live in
// internal/<dom>/schema.sql. If you find yourself adding a CREATE TABLE
// here, ask whether you actually want a `system` domain instead.
func SystemSchema() string { return systemSchemaSQL }
