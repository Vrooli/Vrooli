package selfhealthsnapshots

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the declarative DDL for the self-health snapshot store. It is
// idempotent (CREATE TABLE IF NOT EXISTS) so applyDomainSchemas can run it on
// every boot. A new table needs no column migration (storage-steer).
func Schema() string { return schemaSQL }
