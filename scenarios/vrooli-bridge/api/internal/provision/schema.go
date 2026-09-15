package provision

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the provision domain's SQL contribution (the
// `provisioning_ops` table, the append-only `provision_events` table, and the
// `node_versions` history). Applied by database.EnsureSchemas via the
// modules.AllSchemas() registry. Forward-only declarative — re-runs are no-ops
// (CREATE TABLE IF NOT EXISTS). Never recreate the DB on a schema change.
func Schema() string { return schemaSQL }
