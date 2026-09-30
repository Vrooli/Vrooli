// Package search exposes the deterministic-search transport boundary.
package search

import domain "prompt-manager/internal/search"

var (
	NewAgentSearchService = domain.NewAgentSearchService
	NewHandlers           = domain.NewHandlers
	NewService            = domain.NewService
	NewTeamSearchService  = domain.NewTeamSearchService
)
