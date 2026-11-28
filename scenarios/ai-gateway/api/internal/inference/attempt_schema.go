package inference

import _ "embed"

//go:embed schema.sql
var attemptSchema string

func AttemptSchema() string { return attemptSchema }
