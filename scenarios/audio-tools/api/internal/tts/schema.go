package tts

import _ "embed"

//go:embed schema.sql
var schemaSQL string

// Schema declares TTS configuration and emitted playback events.
func Schema() string { return schemaSQL }
