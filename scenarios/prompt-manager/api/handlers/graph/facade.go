// Package graph exposes the graph REST/Connect transport boundary.
package graph

import domain "prompt-manager/internal/graph"

var (
	DefaultScoreFns                    = domain.DefaultScoreFns
	NewBuilder                         = domain.NewBuilder
	NewCLIDetector                     = domain.NewCLIDetector
	NewCLIHealthCommandValidator       = domain.NewCLIHealthCommandValidator
	NewConnectMount                    = domain.NewConnectMount
	NewHandlers                        = domain.NewHandlers
	NewHealthConfigStore               = domain.NewHealthConfigStore
	NewIndexStore                      = domain.NewIndexStore
	NewOperatingMapStore               = domain.NewOperatingMapStore
	NewScanner                         = domain.NewScanner
	NewScenarioCompletenessCLIProvider = domain.NewScenarioCompletenessCLIProvider
)
