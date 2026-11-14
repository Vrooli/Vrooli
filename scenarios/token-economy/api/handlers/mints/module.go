// Package mints adapts the token-type domain for the authenticated access edge.
package mints

import (
	"token-economy/internal/mints"
)

// Schema re-exports the domain-owned schema for the central boot registry.
func Schema() string { return mints.Schema() }
