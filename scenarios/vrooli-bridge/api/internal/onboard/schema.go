package onboard

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the onboard domain's SQL contribution (the `onboarding_ops`
// table and the append-only `onboarding_step_events` history). Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry. Forward-only
// declarative — re-runs are no-ops (CREATE TABLE IF NOT EXISTS). Never recreate
// the DB on a schema change.
func Schema() string { return schemaSQL }
