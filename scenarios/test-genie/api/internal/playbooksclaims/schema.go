// Package playbooksclaims owns the per-scenario lease + heartbeat record
// guarding concurrent test-genie playbooks runs.
package playbooksclaims

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema returns the declarative DDL for the playbooks_claims table.
// It is idempotent (CREATE TABLE IF NOT EXISTS) and safe to apply on every boot.
func Schema() string { return schemaSQL }
