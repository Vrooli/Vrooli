package hostreq

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type DeploymentTier string

const TierDesktop DeploymentTier = "desktop"

type EligibilityVerdict string

const (
	EligibilityEligible   EligibilityVerdict = "eligible"
	EligibilityDegraded   EligibilityVerdict = "degraded"
	EligibilityIneligible EligibilityVerdict = "ineligible"
)

// Eligibility is a presentation-ready result. Consumers render this value;
// they must not reimplement the privilege/bundling policy.
type Eligibility struct {
	Requirement ResolvedRequirement `json:"requirement"`
	Tier        DeploymentTier      `json:"tier"`
	Platform    string              `json:"platform"`
	Verdict     EligibilityVerdict  `json:"verdict"`
	Reason      string              `json:"reason"`
}

// EvaluateEligibility applies the Tier 2 contract to one resolved requirement.
// present means a host-required object is available on the actual target;
// vendorable objects are supplied by the bundle and therefore pass it as true.
func EvaluateEligibility(requirement ResolvedRequirement, tier DeploymentTier, platform string, present bool) Eligibility {
	platform = strings.ToLower(strings.TrimSpace(platform))
	result := Eligibility{Requirement: requirement, Tier: tier, Platform: platform, Verdict: EligibilityEligible}
	if tier != TierDesktop {
		result.Reason = "not a desktop bundle target"
		return result
	}
	switch requirement.Bundling {
	case hostreqspec.BundlingProhibited:
		result.Verdict = EligibilityIneligible
		result.Reason = fmt.Sprintf("%s %q is prohibited from desktop bundles", requirement.Kind, requirement.Name)
	case hostreqspec.BundlingHostRequired:
		if present {
			result.Reason = fmt.Sprintf("host-required %s %q is present", requirement.Kind, requirement.Name)
			return result
		}
		if requirement.Required {
			result.Verdict = EligibilityIneligible
			result.Reason = fmt.Sprintf("required host %s %q is absent on %s", requirement.Kind, requirement.Name, platform)
		} else {
			result.Verdict = EligibilityDegraded
			result.Reason = fmt.Sprintf("optional host %s %q is absent on %s", requirement.Kind, requirement.Name, platform)
		}
	case hostreqspec.BundlingVendorable:
		if !present {
			result.Verdict = EligibilityIneligible
			result.Reason = fmt.Sprintf("vendorable %s %q has no artifact for %s", requirement.Kind, requirement.Name, platform)
		} else {
			result.Reason = fmt.Sprintf("vendorable %s %q has an artifact for %s", requirement.Kind, requirement.Name, platform)
		}
	default:
		result.Verdict = EligibilityIneligible
		result.Reason = fmt.Sprintf("%s %q has no bundling declaration", requirement.Kind, requirement.Name)
	}
	return result
}
