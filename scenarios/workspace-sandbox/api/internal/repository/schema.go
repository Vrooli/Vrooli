// Package repository provides database operations for sandboxes.
package repository

import _ "embed"

//go:embed schema.sql
var SchemaSQL string
