package attestation

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the attestation ledger SQL contribution.
func Schema() string { return schemaSQL }
