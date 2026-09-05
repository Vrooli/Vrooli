package stt

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema declares STT configuration and speaker enrollment metadata.
func Schema() string { return schemaSQL }
