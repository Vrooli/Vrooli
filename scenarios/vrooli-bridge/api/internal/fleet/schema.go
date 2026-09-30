package fleet

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the fleet domain's SQL contribution (the `rollouts` table and
// the per-node `rollout_results` ledger). Applied by database.EnsureSchemas via
// the modules.AllSchemas() registry. Forward-only declarative — re-runs are
// no-ops (CREATE TABLE IF NOT EXISTS). Never recreate the DB on a schema change.
func Schema() string { return schemaSQL }
