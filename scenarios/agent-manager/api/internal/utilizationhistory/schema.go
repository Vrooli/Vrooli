package utilizationhistory

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema is the domain-owned schema for durable physical-utilization samples.
func Schema() string { return schemaSQL }
