package stripe

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the stripe domain's SQL contribution (checkout_sessions,
// subscriptions, subscription_schedules), applied by database.EnsureSchemas via
// the modules registry.
func Schema() string { return schemaSQL }
