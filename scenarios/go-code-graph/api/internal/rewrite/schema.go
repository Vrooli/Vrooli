package rewrite

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the DDL the rewrite domain registers with EnsureSchemas.
// Today it materializes two tables: `rewrite_plans` (persistent PlanStore
// backing) and `rewrite_operation_log` (REQ-P1-002 Operation Log).
func Schema() string { return schemaSQL }
