// Package hostreq exposes Vrooli's typed host-requirement resolution to
// separately-versioned scenario modules. The implementation remains owned by
// internal/hostreq; this façade prevents consumers from copying deployment
// eligibility policy or invoking the control plane as a subprocess.
package hostreq

import internalhostreq "github.com/vrooli/vrooli/internal/hostreq"

type ResolveOptions = internalhostreq.ResolveOptions
type Resolution = internalhostreq.Resolution
type ResolvedRequirement = internalhostreq.ResolvedRequirement
type Eligibility = internalhostreq.Eligibility
type EligibilityVerdict = internalhostreq.EligibilityVerdict
type DeploymentTier = internalhostreq.DeploymentTier

const (
	TierDesktop           = internalhostreq.TierDesktop
	EligibilityEligible   = internalhostreq.EligibilityEligible
	EligibilityDegraded   = internalhostreq.EligibilityDegraded
	EligibilityIneligible = internalhostreq.EligibilityIneligible
	EligibilityUnknown    = internalhostreq.EligibilityUnknown
)

var (
	Resolve             = internalhostreq.Resolve
	EvaluateEligibility = internalhostreq.EvaluateEligibility
	CurrentPlatform     = internalhostreq.CurrentPlatform
)
