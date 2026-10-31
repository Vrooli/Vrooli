package commerce

import _ "embed"

// OperationsSchema is declarative runtime DDL. api-core/database reconciles
// additive columns as part of EnsureSchemas, so a newly declared column is
// added to an existing table at boot without a hand-run ALTER or destructive
// migration. This is the greenfield-with-data tier: retain one-shot operator
// scripts for data moves, and introduce versioned migrations only once a
// production schema evolution requires them.
//
//go:embed operations_schema.sql
var operationsSchema string

func OperationsSchema() string { return operationsSchema }
