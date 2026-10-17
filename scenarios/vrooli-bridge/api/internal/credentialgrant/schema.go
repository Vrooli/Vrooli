package credentialgrant

import _ "embed"

// schemaSQL is the credential-grant domain's declarative schema. The SQL file
// is the source of truth and is applied through modules.AllSchemas at boot.
//
//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }
