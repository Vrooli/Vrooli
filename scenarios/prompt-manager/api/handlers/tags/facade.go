// Package tags exposes the tags transport boundary.
package tags

import domain "prompt-manager/internal/tags"

var (
	NewHandlers   = domain.NewHandlers
	NewRepository = domain.NewRepository
)
