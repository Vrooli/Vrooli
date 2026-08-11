package telemetry

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the telemetry outbox schema for the central database.
func Schema() string { return schema }
