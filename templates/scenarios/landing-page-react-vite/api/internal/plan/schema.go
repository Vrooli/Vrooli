package plan

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the plan domain's SQL contribution (bundle_products and
// bundle_prices), applied by database.EnsureSchemas via the modules registry.
func Schema() string { return schemaSQL }
