// Package sensitivity owns the document-to-routing policy.  It deliberately
// returns a ceiling rather than choosing a derivation tier: tier selection is
// the derivation domain's responsibility.
package sensitivity

import (
	"fmt"

	gatewaypb "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

// Policy describes the strongest routing profiles a privacy class may use.
// The list is ordered from most restrictive to least restrictive only for
// presentation; callers must use Allows rather than relying on enum order.
type Policy struct {
	Allowed map[gatewaypb.Profile]bool
}

var policies = map[documentpb.PrivacyClass]Policy{
	documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC: {
		Allowed: map[gatewaypb.Profile]bool{
			gatewaypb.Profile_PROFILE_LOCAL_ONLY:        true,
			gatewaypb.Profile_PROFILE_LOCAL_FIRST:       true,
			gatewaypb.Profile_PROFILE_REMOTE_ONLY:       true,
			gatewaypb.Profile_PROFILE_QUALITY_FIRST:     true,
			gatewaypb.Profile_PROFILE_CHEAP_FIRST:       true,
			gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE: true,
		},
	},
	documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL: {
		Allowed: map[gatewaypb.Profile]bool{
			gatewaypb.Profile_PROFILE_LOCAL_ONLY:        true,
			gatewaypb.Profile_PROFILE_LOCAL_FIRST:       true,
			gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE: true,
		},
	},
	documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL: {
		Allowed: map[gatewaypb.Profile]bool{
			gatewaypb.Profile_PROFILE_LOCAL_ONLY:        true,
			gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE: true,
		},
	},
	documentpb.PrivacyClass_PRIVACY_CLASS_SECRET: {
		Allowed: map[gatewaypb.Profile]bool{
			gatewaypb.Profile_PROFILE_LOCAL_ONLY: true,
		},
	},
}

func (p Policy) Allows(profile gatewaypb.Profile) bool {
	return p.Allowed[profile]
}

// PolicyFor returns a copy of the policy for class. Unknown classes fail
// closed instead of silently becoming public.
func PolicyFor(class documentpb.PrivacyClass) (Policy, error) {
	policy, ok := policies[class]
	if !ok {
		return Policy{}, fmt.Errorf("no sensitivity policy for privacy class %s", class)
	}
	return policy, nil
}

// Ceiling is the highest derivation tier permitted by class. It does not
// select a handler chain.
func Ceiling(class documentpb.PrivacyClass) (documentpb.Tier, error) {
	switch class {
	case documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC,
		documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL:
		return documentpb.Tier_TIER_THREE, nil
	case documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		documentpb.PrivacyClass_PRIVACY_CLASS_SECRET:
		return documentpb.Tier_TIER_TWO, nil
	default:
		return documentpb.Tier_TIER_UNSPECIFIED, fmt.Errorf("unknown privacy class %s", class)
	}
}
