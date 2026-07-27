// Package analytics owns storage declarations for conversion analytics.
package analytics

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the analytics-owned declarative SQL schema.
func Schema() string { return schema }
