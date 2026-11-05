package landing

import (
	"bytes"
	"testing"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/landing"
	"landing-page-business-suite-api/internal/presentation"
)

func TestLandingConfigProtoRejectsLegacyPayload(t *testing.T) {
	if _, err := LandingConfigProto(&landing.LandingConfigResponse{}); err == nil {
		t.Fatal("legacy response was encoded without a typed presentation")
	}
}

func TestLandingConfigProtoTypedPresentationOmitsLegacyMarketingFields(t *testing.T) { // [REQ:LP-PRES-011]
	response, err := LandingConfigProto(&landing.LandingConfigResponse{
		Pricing:   &commerce.PricingOverview{},
		Downloads: []delivery.App{{AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{}}},
		Presentation: &presentation.ResolveResult{
			Page:        presentation.ResolvedPage{Display: presentation.PageDisplay{Shell: presentation.ShellDisplay{BrandName: "Configured", BrandMark: "suite", BrandTarget: "/", SkipLabel: "Skip", MenuLabel: "Menu", FooterBrandName: "Configured", FooterBrandMark: "suite", FooterBrandTarget: "/", UnavailableReason: "Unavailable", PreviewLabel: "Preview"}}},
			Diagnostics: presentation.Diagnostics{AssignmentSource: "weighted_visitor", WeightFingerprint: "weights-v1"},
			Actions:     []presentation.ResolvedAction{{Key: `["purchase","web-console","price-pro",""]`, Status: presentation.ResolvedActionReady, Href: "/checkout?price_id=price-pro", AppKey: "web-console", PlanRef: "price-pro"}},
		},
	})
	if err != nil {
		t.Fatalf("LandingConfigProto() error = %v", err)
	}
	assertLegacyLandingFieldsRetired(t, response)
	if response.GetPresentation() == nil || response.GetPresentation().GetPage().GetDisplay().GetShell().GetBrandName() != "Configured" {
		t.Fatalf("typed presentation missing or display was not mapped: %+v", response.GetPresentation())
	}
	if got := response.GetPresentation().GetDiagnostics().GetAssignmentSource(); got != "weighted_visitor" || response.GetPresentation().GetDiagnostics().GetWeightFingerprint() != "weights-v1" {
		t.Fatalf("typed assignment diagnostics = source:%q fingerprint:%q", got, response.GetPresentation().GetDiagnostics().GetWeightFingerprint())
	}
	if len(response.GetPresentation().GetActions()) != 1 || response.GetPresentation().GetActions()[0].GetStatus() != "ready" || response.GetPresentation().GetActions()[0].GetHref() != "/checkout?price_id=price-pro" {
		t.Fatalf("typed action was not mapped: %+v", response.GetPresentation().GetActions())
	}
}

func TestLandingConfigProtoRetiredLegacyFieldsAreAbsentFromDescriptorAndWire(t *testing.T) {
	response := &lpbsv1.LandingConfigResponse{}
	fields := response.ProtoReflect().Descriptor().Fields()
	retired := []struct {
		number protoreflect.FieldNumber
		name   protoreflect.Name
	}{
		{1, "variant"}, {2, "sections"}, {5, "header"},
		{6, "branding"}, {8, "coupon_mappings"}, {9, "intro_offers"},
	}
	for _, field := range retired {
		if fields.ByNumber(field.number) != nil || fields.ByName(field.name) != nil {
			t.Errorf("retired field %d/%q remains in the descriptor", field.number, field.name)
		}
	}

	wire := []byte{}
	for _, field := range retired {
		wire = protowire.AppendTag(wire, field.number, protowire.BytesType)
		wire = protowire.AppendBytes(wire, []byte{0x01})
	}
	decoded := &lpbsv1.LandingConfigResponse{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatalf("unmarshal retired-field wire: %v", err)
	}
	if !bytes.Equal(decoded.ProtoReflect().GetUnknown(), wire) {
		t.Fatalf("retired wire fields were interpreted instead of preserved as unknown: got %x want %x", decoded.ProtoReflect().GetUnknown(), wire)
	}
}

func assertLegacyLandingFieldsRetired(t *testing.T, response *lpbsv1.LandingConfigResponse) {
	t.Helper()
	fields := response.ProtoReflect().Descriptor().Fields()
	for _, field := range []struct {
		number protoreflect.FieldNumber
		name   protoreflect.Name
	}{
		{1, "variant"}, {2, "sections"}, {5, "header"},
		{6, "branding"}, {8, "coupon_mappings"}, {9, "intro_offers"},
	} {
		if fields.ByNumber(field.number) != nil || fields.ByName(field.name) != nil {
			t.Errorf("typed response descriptor still exposes legacy field %d/%q", field.number, field.name)
		}
	}
}

func TestLandingConfigProtoTypedPresentationSanitizesDeliveryOwner(t *testing.T) { // [REQ:LP-PRES-011]
	artifactID := int64(17)
	response, err := LandingConfigProto(&landing.LandingConfigResponse{
		Downloads: []delivery.App{{
			ID: 9, BundleKey: "business-suite", AppKey: "web-console", Name: "owner-private name",
			InstallSteps: []string{"/srv/private/install.sh"}, UpdateAPIKey: "secret-update-key",
			Metadata: map[string]interface{}{"web_url": "/apps/aquila", "secret": "BAS prose"},
			Platforms: []delivery.Asset{{
				ID: 11, BundleKey: "business-suite", AppKey: "web-console", Platform: "linux",
				ArtifactURL: "https://private.example/download", ArtifactSource: "managed", ArtifactID: &artifactID,
				ReleaseVersion: "1.2.3", ReleaseNotes: "private release notes", Checksum: "sha256:abc",
				VariantKey: "internal-arm64", ArtifactFilename: "private.tar.gz", ArtifactSizeBytes: 99,
				ArtifactCount: 2, Metadata: map[string]interface{}{"source_path": "/srv/private"}, RequiresEntitlement: true,
			}},
		}},
		Presentation: &presentation.ResolveResult{},
	})
	if err != nil {
		t.Fatalf("LandingConfigProto() error = %v", err)
	}
	app := response.GetDownloads()[0]
	if app.GetName() != "" || len(app.GetInstallSteps()) != 0 || app.GetUpdateApiKey() != "" || app.GetId() != 9 {
		t.Fatalf("typed delivery retained owner fields: %+v", app)
	}
	if app.GetMetadata().GetFields()["web_url"].GetStringValue() != "/apps/aquila" || len(app.GetMetadata().GetFields()) != 1 {
		t.Fatalf("typed delivery web launch projection = %+v", app.GetMetadata())
	}
	asset := app.GetPlatforms()[0]
	if asset.GetId() != 11 || asset.GetArtifactId() != artifactID || asset.GetPlatform() != "linux" || asset.GetReleaseVersion() != "1.2.3" || asset.GetChecksum() != "sha256:abc" || !asset.GetRequiresEntitlement() {
		t.Fatalf("typed delivery facts were not retained: %+v", asset)
	}
	if asset.GetArtifactUrl() != "" || asset.GetArtifactSource() != "" || asset.GetReleaseNotes() != "" || asset.GetVariantKey() != "" || asset.GetArtifactFilename() != "" || asset.GetArtifactSizeBytes() != 0 || asset.GetArtifactCount() != 0 || len(asset.GetMetadata().GetFields()) != 0 {
		t.Fatalf("typed delivery retained private artifact fields: %+v", asset)
	}
}
