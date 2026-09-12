package download

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the download domain's SQL contribution (download_apps and
// download_assets), applied by database.EnsureSchemas via the modules registry.
func Schema() string { return schemaSQL }
