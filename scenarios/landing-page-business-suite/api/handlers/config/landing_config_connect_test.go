package landing

import (
	"math"
	"testing"

	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/landing"
)

func stringPointer(value string) *string { return &value }

func TestLandingConfigProtoPreservesPublicOptionalAndNestedFields(t *testing.T) {
	amountOff, redeemBy, artifactID := int64(250), int64(1735689600), int64(7)
	durationInMonths, maxRedemptions, sectionID := 3, 12, 42
	supportEmail, comingSoon := "support@example.com", true
	response, err := LandingConfigProto(&landing.LandingConfigResponse{
		Variant:        landing.LandingVariantSummary{ID: 9, Slug: "control", Name: "Control", Axes: map[string]string{"audience": "pro"}},
		Sections:       []landing.LandingSection{{SectionType: "hero", Content: map[string]interface{}{"headline": "Ship safely"}, Order: 1, Enabled: true}},
		Header:         experimentation.LandingHeaderConfig{Nav: experimentation.HeaderNavConfig{Links: []experimentation.HeaderNavLink{{ID: "plans", Type: "section", Label: "Plans", SectionID: &sectionID, VisibleOn: experimentation.HeaderVisibilityConfig{Desktop: true, Mobile: true}}}}},
		Branding:       &landing.LandingBranding{SiteName: "Business Suite", SupportEmail: &supportEmail, ComingSoonEnabled: &comingSoon},
		CouponMappings: map[string]string{"price_pro": "intro_pro"},
		IntroOffers:    []landing.IntroOffer{{ID: "intro_pro", Name: stringPointer("Pro intro"), AmountOff: amountOff, Duration: "repeating", DurationInMonths: &durationInMonths, MaxRedemptions: &maxRedemptions, RedeemBy: redeemBy, Valid: true, IsIntroCoupon: true, IntroTier: stringPointer("pro")}},
		Downloads:      []delivery.App{{ID: 3, BundleKey: "business_suite", AppKey: "desktop", Name: "Desktop", IconURL: "https://example.com/icon.png", UpdatePolicy: map[string]interface{}{"channel": "stable"}, Metadata: map[string]interface{}{"category": "desktop"}, Platforms: []delivery.Asset{{ID: 5, BundleKey: "business_suite", AppKey: "desktop", Platform: "darwin", ArtifactURL: "https://example.com/app.dmg", ArtifactSource: "managed", ArtifactID: &artifactID, VariantKey: "arm64", ArtifactFilename: "app.dmg", ArtifactSizeBytes: 99, ArtifactCount: 1, Metadata: map[string]interface{}{"signed": true}}}}},
	})
	if err != nil {
		t.Fatalf("LandingConfigProto() error = %v", err)
	}
	if got := response.GetSections()[0].GetContent().GetFields()["headline"].GetStringValue(); got != "Ship safely" {
		t.Fatalf("section content = %q", got)
	}
	if got := response.GetHeader().GetNav().GetLinks()[0].GetSectionId(); got != int32(sectionID) {
		t.Fatalf("header section ID = %d", got)
	}
	if response.GetBranding().GetSupportEmail() != supportEmail || !response.GetBranding().GetComingSoonEnabled() {
		t.Fatalf("branding lost optional values: %#v", response.GetBranding())
	}
	offer := response.GetIntroOffers()[0]
	if offer.GetAmountOff() != amountOff || offer.GetDurationInMonths() != int32(durationInMonths) || offer.GetMaxRedemptions() != int32(maxRedemptions) || offer.GetRedeemBy() != redeemBy {
		t.Fatalf("intro offer lost fields: %#v", offer)
	}
	asset := response.GetDownloads()[0].GetPlatforms()[0]
	if asset.GetArtifactId() != artifactID || asset.GetArtifactSource() != "managed" || asset.GetArtifactFilename() != "app.dmg" || asset.GetArtifactSizeBytes() != 99 {
		t.Fatalf("download asset lost fields: %#v", asset)
	}
}

func TestLandingConfigInt32RejectsOutOfRangeValues(t *testing.T) {
	if _, err := LandingConfigInt32(math.MaxInt32+1, "test"); err == nil {
		t.Fatal("accepted a value above int32 range")
	}
	if _, err := LandingConfigInt32(math.MinInt32-1, "test"); err == nil {
		t.Fatal("accepted a value below int32 range")
	}
}
