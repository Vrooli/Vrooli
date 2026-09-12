// Package schema registers the scenario's domain-owned runtime schemas.
package schema

import (
	_ "embed"
)

//go:embed system.sql
var systemSchema string

// System returns the declarative schema owned by shared runtime substrate.
// Domain schemas are composed by the API composition root, not by this
// substrate package, so the substrate never depends on product domains.
func System() string { return systemSchema }
