package outcomeledger

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the run_ledger_facts and usage_mode_periods tables this package owns.
func Schema() string { return schemaSQL }
