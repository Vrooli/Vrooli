// Package skills exposes the skills transport boundary.
package skills

import domain "prompt-manager/internal/skills"

var (
	NewAgentManagerIdentityVerifier = domain.NewAgentManagerIdentityVerifier
	NewHandlers                     = domain.NewHandlers
	NewMetricsAdapter               = domain.NewMetricsAdapter
	NewReadRecorder                 = domain.NewReadRecorder
	NewStoreAdapter                 = domain.NewStoreAdapter
	NewUsageReporter                = domain.NewUsageReporter
)
