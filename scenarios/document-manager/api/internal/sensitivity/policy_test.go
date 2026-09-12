package sensitivity

import (
	"testing"

	"github.com/stretchr/testify/require"
	gatewaypb "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

func TestPolicyTableRejectsWeakerProfiles(t *testing.T) {
	classes := []documentpb.PrivacyClass{
		documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC,
		documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		documentpb.PrivacyClass_PRIVACY_CLASS_SECRET,
	}
	profiles := []gatewaypb.Profile{
		gatewaypb.Profile_PROFILE_LOCAL_ONLY,
		gatewaypb.Profile_PROFILE_LOCAL_FIRST,
		gatewaypb.Profile_PROFILE_REMOTE_ONLY,
		gatewaypb.Profile_PROFILE_QUALITY_FIRST,
		gatewaypb.Profile_PROFILE_CHEAP_FIRST,
		gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE,
	}
	for _, class := range classes {
		policy, err := PolicyFor(class)
		require.NoError(t, err)
		for _, profile := range profiles {
			expected := class == documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC ||
				(class == documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL && (profile == gatewaypb.Profile_PROFILE_LOCAL_ONLY || profile == gatewaypb.Profile_PROFILE_LOCAL_FIRST || profile == gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE)) ||
				(class == documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL && (profile == gatewaypb.Profile_PROFILE_LOCAL_ONLY || profile == gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE)) ||
				(class == documentpb.PrivacyClass_PRIVACY_CLASS_SECRET && profile == gatewaypb.Profile_PROFILE_LOCAL_ONLY)
			require.Equalf(t, expected, policy.Allows(profile), "[REQ:DOC-P0-013] policy mismatch for %s and %s", class, profile)
		}
	}
}

func TestCeilingDoesNotSelectTier(t *testing.T) { // [REQ:DOC-P0-012]
	tier, err := Ceiling(documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL)
	require.NoError(t, err)
	require.Equal(t, documentpb.Tier_TIER_TWO, tier)
}
