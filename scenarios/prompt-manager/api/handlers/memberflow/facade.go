// Package memberflow exposes the member-flow transport boundary.
package memberflow

import domain "prompt-manager/internal/memberflow"

type (
	InboxAgingOptions           = domain.InboxAgingOptions
	OperatingGraphPromptSection = domain.OperatingGraphPromptSection
	OperatingModelService       = domain.OperatingModelService
)

var (
	NewHandlers                           = domain.NewHandlers
	OperatingGraphPromptSectionSourceLive = domain.OperatingGraphPromptSectionSourceLive
)
