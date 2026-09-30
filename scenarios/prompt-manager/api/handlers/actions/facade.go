// Package actions exposes the actions transport boundary while the legacy
// REST adapter is retired in favor of Connect.
package actions

import domain "prompt-manager/internal/actions"

type SemanticActionHit = domain.SemanticActionHit

var (
	NewCLIHealthCommandResolver = domain.NewCLIHealthCommandResolver
	NewHandlers                 = domain.NewHandlers
	NewService                  = domain.NewService
)
