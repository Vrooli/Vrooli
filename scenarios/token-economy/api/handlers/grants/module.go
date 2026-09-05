// Package grants adapts grant authority for the authenticated access edge.
package grants

import (
	"token-economy/internal/grants"
)

// Schema re-exports the domain-owned schema for the central boot registry.
func Schema() string { return grants.Schema() }
