// Package requirements implements native Go requirements synchronization.
// This file re-exports types from the types subpackage for backwards compatibility.
package requirements

import "test-genie/internal/requirements/types"

// Type aliases for evidence types
type (
	EvidenceRecord   = types.EvidenceRecord
	EvidenceMap      = types.EvidenceMap
	ManualValidation = types.ManualValidation
	PhaseResult      = types.PhaseResult
	VitestResult     = types.VitestResult
	EvidenceBundle   = types.EvidenceBundle
	ManualManifest   = types.ManualManifest
)

// Function re-exports
var (
	NewEvidenceBundle = types.NewEvidenceBundle
	NewManualManifest = types.NewManualManifest
)
