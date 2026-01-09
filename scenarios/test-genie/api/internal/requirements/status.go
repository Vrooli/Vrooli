// Package requirements implements native Go requirements synchronization.
// This file re-exports types from the types subpackage for backwards compatibility.
package requirements

import "test-genie/internal/requirements/types"

// Type aliases for status types
type DeclaredStatus = types.DeclaredStatus
type LiveStatus = types.LiveStatus
type ValidationStatus = types.ValidationStatus
type Criticality = types.Criticality
type ValidationType = types.ValidationType

// Status constants
const (
	StatusPending        = types.StatusPending
	StatusPlanned        = types.StatusPlanned
	StatusInProgress     = types.StatusInProgress
	StatusComplete       = types.StatusComplete
	StatusNotImplemented = types.StatusNotImplemented
)

const (
	LivePassed  = types.LivePassed
	LiveFailed  = types.LiveFailed
	LiveSkipped = types.LiveSkipped
	LiveNotRun  = types.LiveNotRun
	LiveUnknown = types.LiveUnknown
)

const (
	ValStatusNotImplemented = types.ValStatusNotImplemented
	ValStatusPlanned        = types.ValStatusPlanned
	ValStatusImplemented    = types.ValStatusImplemented
	ValStatusFailing        = types.ValStatusFailing
)

const (
	CriticalityP0 = types.CriticalityP0
	CriticalityP1 = types.CriticalityP1
	CriticalityP2 = types.CriticalityP2
)

const (
	ValTypeTest       = types.ValTypeTest
	ValTypeAutomation = types.ValTypeAutomation
	ValTypeManual     = types.ValTypeManual
	ValTypeLighthouse = types.ValTypeLighthouse
)

// Function re-exports
var (
	NormalizeLiveStatus       = types.NormalizeLiveStatus
	NormalizeDeclaredStatus   = types.NormalizeDeclaredStatus
	NormalizeValidationStatus = types.NormalizeValidationStatus
	NormalizeCriticality      = types.NormalizeCriticality
	NormalizeValidationType   = types.NormalizeValidationType
	DeriveValidationStatus    = types.DeriveValidationStatus
	DeriveRequirementStatus   = types.DeriveRequirementStatus
	DeriveLiveRollup          = types.DeriveLiveRollup
	DeriveDeclaredRollup      = types.DeriveDeclaredRollup
	IsTerminalStatus          = types.IsTerminalStatus
	IsCritical                = types.IsCritical
)
