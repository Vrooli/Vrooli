package secrets

import _ "embed"

//go:embed schema.sql
var schemaSQL string

//go:embed migrations/001_resource_secret_metadata.sql
var resourceSecretMetadataMigration string

//go:embed desktop_schema.sql
var desktopSchemaSQL string

// Schema returns the secrets domain schema.
func Schema() string { return schemaSQL }

// ResourceSecretMetadataMigration returns the forward-only compatibility
// migration for databases created before the metadata columns existed.
func ResourceSecretMetadataMigration() string { return resourceSecretMetadataMigration }

// DesktopSchema returns the SQLite schema used by bundled desktop mode.
func DesktopSchema() string { return desktopSchemaSQL }
