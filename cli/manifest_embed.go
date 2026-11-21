// Package manifest exposes the root CLI manifest to the control-plane binary.
package manifest

import _ "embed"

// raw is the single embedded copy of the root command contract.
//
//go:embed manifest.json
var raw []byte

// Bytes returns a copy of cli/manifest.json for manifest-driven command
// construction.
func Bytes() []byte { return append([]byte(nil), raw...) }
