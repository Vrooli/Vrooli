// Package skills exposes the skills and experiment transport boundary.
package skills

import domain "prompt-manager/internal/skills"

var (
	NewAgentManagerIdentityVerifier = domain.NewAgentManagerIdentityVerifier
	NewExperimentHandlers           = domain.NewExperimentHandlers
	NewHTTPWorkPublisherFromEnv     = domain.NewHTTPWorkPublisherFromEnv
	NewHandlers                     = domain.NewHandlers
	NewMetricsAdapter               = domain.NewMetricsAdapter
	NewReadRecorder                 = domain.NewReadRecorder
	NewStoreAdapter                 = domain.NewStoreAdapter
	NewUsageReporter                = domain.NewUsageReporter
	NewVariantHandlers              = domain.NewVariantHandlers
)
