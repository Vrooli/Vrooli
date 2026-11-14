package attached

import _ "embed"

// schemaSQL is the attached-device domain's declarative schema. Keeping the
// schema beside the repository makes the domain's storage footprint explicit;
// modules.AllSchemas registers it during API bootstrap.
//
//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }
