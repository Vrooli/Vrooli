// Package templates exposes the template transport boundary.
package templates

import domain "prompt-manager/internal/templates"

var (
	NewHandlers = domain.NewHandlers
	NewStore    = domain.NewStore
)
