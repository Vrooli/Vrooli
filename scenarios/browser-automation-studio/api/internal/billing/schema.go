package billing

import _ "embed"

import monetization "github.com/vrooli/vrooli/packages/monetization-go"

//go:embed schema.sql
var schemaSQL string

// Schema returns the billing domain schema.
func Schema() string { return schemaSQL + "\n" + monetization.SQLiteSchema }
