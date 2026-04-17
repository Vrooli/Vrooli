package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

type (
	SupportClass   = hostreqkit.SupportClass
	ExecutionState = hostreqkit.ExecutionState
)

const (
	SupportSupported     = hostreqkit.SupportSupported
	SupportUnsupported   = hostreqkit.SupportUnsupported
	SupportNotApplicable = hostreqkit.SupportNotApplicable
	SupportManualOnly    = hostreqkit.SupportManualOnly
)

const (
	ExecutionPending              = hostreqkit.ExecutionPending
	ExecutionAlreadyPresent       = hostreqkit.ExecutionAlreadyPresent
	ExecutionWouldInstall         = hostreqkit.ExecutionWouldInstall
	ExecutionWouldApply           = hostreqkit.ExecutionWouldApply
	ExecutionInstalled            = hostreqkit.ExecutionInstalled
	ExecutionApplied              = hostreqkit.ExecutionApplied
	ExecutionManualActionRequired = hostreqkit.ExecutionManualActionRequired
	ExecutionUnsupported          = hostreqkit.ExecutionUnsupported
	ExecutionNotApplicable        = hostreqkit.ExecutionNotApplicable
	ExecutionFailed               = hostreqkit.ExecutionFailed
)

type ItemStatus = hostreqkit.ItemStatus

type (
	ToolStatus      = hostreqkit.ToolStatus
	SafeguardStatus = hostreqkit.SafeguardStatus
)

type Report = hostreqkit.Report
