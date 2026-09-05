package authorization

import (
	_ "embed"
	"strings"
)

//go:embed schema.sql
var schema string

func Schema() string { return strings.TrimSpace(schema) }
