// Package requirements implements native Go requirements synchronization.
// This file re-exports types from the types subpackage for backwards compatibility.
package requirements

import "test-genie/internal/requirements/types"

// Type aliases for evidence types
type EvidenceRecord = types.EvidenceRecord
type EvidenceMap = types.EvidenceMap
type ManualValidation = types.ManualValidation
type PhaseResult = types.PhaseResult
type VitestResult = types.VitestResult
type EvidenceBundle = types.EvidenceBundle
type ManualManifest = types.ManualManifest

// Function re-exports
var (
	NewEvidenceBundle = types.NewEvidenceBundle
	NewManualManifest = types.NewManualManifest
)
