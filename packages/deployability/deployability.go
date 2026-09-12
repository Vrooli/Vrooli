// Package deployability exposes the control-plane deployability contracts to
// scenario modules without requiring them to import an internal package.
// The implementation and ownership remain in internal/deployability.
package deployability

import core "github.com/vrooli/vrooli/internal/deployability"

type (
	HostOS                         = core.HostOS
	Bundling                       = core.Bundling
	DeliveryTier                   = core.DeliveryTier
	Verdict                        = core.Verdict
	ResourceRequirements           = core.ResourceRequirements
	GPURequirement                 = core.GPURequirement
	PlatformDeclaration            = core.PlatformDeclaration
	CapabilityDeclaration          = core.CapabilityDeclaration
	DependencyDeclaration          = core.DependencyDeclaration
	TargetDeclaration              = core.TargetDeclaration
	ResolutionInput                = core.ResolutionInput
	Reason                         = core.Reason
	DependencyResult               = core.DependencyResult
	Resolution                     = core.Resolution
	CapabilityImplementation       = core.CapabilityImplementation
	CapabilityDeclarer             = core.CapabilityDeclarer
	CapabilityResolutionStatus     = core.CapabilityResolutionStatus
	CapabilityResolution           = core.CapabilityResolution
	ConformanceFinding             = core.ConformanceFinding
	ConformanceReport              = core.ConformanceReport
	ConformanceTarget              = core.ConformanceTarget
	Evidence                       = core.Evidence
	ManifestRule                   = core.ManifestRule
	ScenarioManifestReport         = core.ScenarioManifestReport
	InstanceLiteral                = core.InstanceLiteral
	ManifestDeclaration            = core.ManifestDeclaration
	PlatformObservation            = core.PlatformObservation
	PlatformStatus                 = core.PlatformStatus
	Qualification                  = core.Qualification
	SwapSource                     = core.SwapSource
	SwapAlternative                = core.SwapAlternative
	ResourceSwapSuggestion         = core.ResourceSwapSuggestion
	AcquisitionCoverageDeclaration = core.AcquisitionCoverageDeclaration
	UnknownPlatformStatusError     = core.UnknownPlatformStatusError
)

const (
	HostOSLinux   = core.HostOSLinux
	HostOSMacOS   = core.HostOSMacOS
	HostOSWindows = core.HostOSWindows

	BundlingVendorable   = core.BundlingVendorable
	BundlingHostRequired = core.BundlingHostRequired
	BundlingProhibited   = core.BundlingProhibited

	TierLocal      = core.TierLocal
	TierDesktop    = core.TierDesktop
	TierMobile     = core.TierMobile
	TierSaaS       = core.TierSaaS
	TierEnterprise = core.TierEnterprise

	VerdictEligible   = core.VerdictEligible
	VerdictDegraded   = core.VerdictDegraded
	VerdictIneligible = core.VerdictIneligible
	VerdictUnknown    = core.VerdictUnknown

	CapabilityImplemented = core.CapabilityImplemented
	CapabilityUnwired     = core.CapabilityUnwired
	CapabilityPeerless    = core.CapabilityPeerless
)

var (
	Resolve                     = core.Resolve
	ResolveCapability           = core.ResolveCapability
	SuggestResourceSwaps        = core.SuggestResourceSwaps
	ExtractDeclaredAlternatives = core.ExtractDeclaredAlternatives
	FindInstanceLiterals        = core.FindInstanceLiterals
	ValidateAcquisitionCoverage = core.ValidateAcquisitionCoverage
)
