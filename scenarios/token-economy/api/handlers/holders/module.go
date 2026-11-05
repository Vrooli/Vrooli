// Package holders adapts authenticated holder views for the access edge.
package holders

import (
	"token-economy/internal/holders"
)

// Schema re-exports the domain-owned schema for the central boot registry.
func Schema() string { return holders.Schema() }
