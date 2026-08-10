// Package deployability exposes the control-plane deployability contracts to
// scenario modules without requiring them to import an internal package.
// The implementation and ownership remain in internal/deployability.
package deployability

import core "github.com/vrooli/vrooli/internal/deployability"

type HostOS = core.HostOS
type Bundling = core.Bundling
type DeliveryTier = core.DeliveryTier
type Verdict = core.Verdict
type ResourceRequirements = core.ResourceRequirements
type PlatformDeclaration = core.PlatformDeclaration
type CapabilityDeclaration = core.CapabilityDeclaration
type DependencyDeclaration = core.DependencyDeclaration
type TargetDeclaration = core.TargetDeclaration
type ResolutionInput = core.ResolutionInput
type Reason = core.Reason
type DependencyResult = core.DependencyResult
type Resolution = core.Resolution
type CapabilityImplementation = core.CapabilityImplementation
type CapabilityResolutionStatus = core.CapabilityResolutionStatus
type CapabilityResolution = core.CapabilityResolution
type InstanceLiteral = core.InstanceLiteral
type SwapSource = core.SwapSource
type SwapAlternative = core.SwapAlternative
type ResourceSwapSuggestion = core.ResourceSwapSuggestion
type ToolAcquisitionDeclaration = core.ToolAcquisitionDeclaration

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

var Resolve = core.Resolve
var ResolveCapability = core.ResolveCapability
var SuggestResourceSwaps = core.SuggestResourceSwaps
var ExtractDeclaredAlternatives = core.ExtractDeclaredAlternatives
var FindInstanceLiterals = core.FindInstanceLiterals
var ValidateMacOSAcquisition = core.ValidateMacOSAcquisition
