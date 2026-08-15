package telemetry

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the telemetry domain's SQL contribution. Query telemetry is
// kept in its own domain so it can be added, migrated, or removed independently
// of the cross-cutting metrics handlers that read it.
func Schema() string { return schemaSQL }
