package landing

import (
	"encoding/json"
	"strings"
	"testing"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"landing-page-business-suite-api/internal/presentation"
)

func TestPresentationTypedDocumentRoundTrip(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-014]
	document := presentation.Document{
		SchemaVersion: 1,
		Bundle:        presentation.Bundle{Key: "business-suite", Name: "Vrooli", AppOrder: []string{"web-console"}, PageID: "suite", EmptyPageID: "empty", DefaultLocale: "en", Locales: []string{"en"}},
		Apps:          []presentation.App{{Key: "web-console", Slug: "aquila", Name: "Aquila", Capabilities: []presentation.Capability{{ID: "speech", Label: "Local speech", Status: presentation.CapabilityComingSoon, LocalizedBenefits: map[string][]string{"fr": {"Une voix locale"}}}}}},
		Pages: []presentation.Page{{
			ID: "aquila", Locale: "en", Title: "Your agents. Your voice. Your work.",
			Blocks: []presentation.Block{
				{ID: "hero", Kind: presentation.BlockProductHero, Version: 1, Variant: "centered", Content: presentation.ProductHeroContent{
					AppKey: "web-console", Title: "Bring your agents into focus.", FixtureRef: "workspace",
					Actions: []presentation.Action{{Kind: presentation.ActionAnchor, Label: "Explore", AccessibleLabel: "Explore Aquila", Target: "#artifacts"}},
				}},
				{ID: "demo", Kind: presentation.BlockProductDemo, Version: 1, Variant: "recorded", Content: presentation.ProductDemoContent{
					Heading: "See the workflow.", Description: "A recorded product demonstration.", RendererRef: "video", PosterRef: "poster", AltText: "Recorded product demonstration",
					Playback: &presentation.ProductDemoPlayback{Provider: "youtube", ExternalURL: "https://youtu.be/dQw4w9WgXcQ", Layout: "split", PlayLabel: "Play demo", Caption: "A configured recording.", UnavailableLabel: "Video unavailable"},
				}},
			},
		}},
		Fixtures: []presentation.Fixture{{ID: "workspace", Kind: presentation.FixtureWorkspace, Workspace: &presentation.WorkspaceFixture{Title: "Aquila", Sessions: []string{"Review the changes"}}}},
		Assets:   []presentation.Asset{{ID: "poster", ReleaseRef: "release-poster", ContentHash: strings.Repeat("a", 64), Width: 1280, Height: 720, MIME: "image/png", Surface: "web-demo", CropPolicy: "center", Provenance: presentation.AssetProvenance{Provider: "test"}}},
		Strings:  map[string]map[string]string{"en": {"skip": "Skip to content", "summary": "AI response summary"}},
	}
	document.Pages[0].Display = presentation.PageDisplay{Shell: presentation.ShellDisplay{BrandName: "Aquila", BrandMark: "letter-a", SkipLabel: "Skip to content"}, Blocks: map[string]presentation.BlockDisplay{"hero": {Note: "Configured small print", HeadingBreaks: []int{12}}}, FixtureDisplay: map[string]presentation.FixtureDisplay{"workspace": {Mark: "letter-a", TabsLabel: "Session tabs", FileChanges: map[string]string{"plan.md": "+14"}}}}
	wire, err := PresentationDocumentProto(document)
	if err != nil {
		t.Fatal(err)
	}
	if wire.Pages[0].Blocks[0].Content.GetProductHero().GetFixtureRef() != "workspace" {
		t.Fatal("typed hero content was lost")
	}
	demo := wire.Pages[0].Blocks[1].Content.GetProductDemo()
	if demo.GetRendererRef() != "video" || demo.GetPosterRef() != "poster" || demo.GetPlayback().GetProvider() != "youtube" || demo.GetPlayback().GetExternalUrl() != "https://youtu.be/dQw4w9WgXcQ" || demo.GetPlayback().GetLayout() != "split" {
		t.Fatalf("typed recorded playback was lost: %+v", demo)
	}
	if wire.Fixtures[0].GetWorkspace().GetSessions()[0] != "Review the changes" {
		t.Fatal("configured workspace data was lost")
	}
	if wire.Strings["en"].Values["skip"] != "Skip to content" || wire.Apps[0].Capabilities[0].LocalizedBenefits["fr"].Values[0] != "Une voix locale" {
		t.Fatal("configured localized map values were lost")
	}
	roundTrip, err := PresentationDocumentFromProto(wire)
	if err != nil {
		t.Fatal(err)
	}
	hero, ok := roundTrip.Pages[0].Blocks[0].Content.(presentation.ProductHeroContent)
	if !ok || hero.AppKey != "web-console" || hero.Actions[0].AccessibleLabel != "Explore Aquila" {
		t.Fatalf("domain content changed during wire round trip: %+v", roundTrip.Pages[0].Blocks)
	}
	if roundTrip.Strings["en"]["summary"] != document.Strings["en"]["summary"] || roundTrip.Apps[0].Capabilities[0].LocalizedBenefits["fr"][0] != "Une voix locale" {
		t.Fatal("domain localized copy changed during wire round trip")
	}
	roundTripDemo, ok := roundTrip.Pages[0].Blocks[1].Content.(presentation.ProductDemoContent)
	if !ok || roundTripDemo.Playback == nil || roundTripDemo.Playback.Provider != "youtube" || roundTripDemo.Playback.PlayLabel != "Play demo" {
		t.Fatalf("domain recorded playback changed during wire round trip: %+v", roundTrip.Pages[0].Blocks[1].Content)
	}
	if roundTrip.Pages[0].Display.Shell.BrandName != "Aquila" || roundTrip.Pages[0].Display.Blocks["hero"].Note != "Configured small print" || roundTrip.Pages[0].Display.FixtureDisplay["workspace"].FileChanges["plan.md"] != "+14" {
		t.Fatal("typed display configuration was not preserved")
	}
}

func TestPresentationTypedContentRejectsMismatchedDiscriminator(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-012]
	wire := &shared.ProductPresentationDocument{Pages: []*shared.PresentationPage{{Blocks: []*shared.PresentationBlock{{
		Kind: "product-hero", Content: &shared.PresentationBlockContent{Value: &shared.PresentationBlockContent_VoiceStory{VoiceStory: &shared.PresentationVoiceStory{Heading: "Wrong kind"}}},
	}}}}}
	if _, err := PresentationDocumentFromProto(wire); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched oneof accepted: %v", err)
	}
	if _, err := PresentationDocumentFromProto(nil); err == nil {
		t.Fatal("nil document accepted")
	}
}

func TestResolvedPresentationWirePreservesIdentityAndOmitsPrivateEvidence(t *testing.T) { // [REQ:LP-PRES-009] [REQ:LP-PRES-012]
	value := presentation.ResolveResult{
		SchemaVersion: 1, Mode: presentation.ModeSingleApp, Scope: presentation.ScopeApp, AppKey: "web-console",
		Page:         presentation.ResolvedPage{ID: "aquila", Locale: "en", Blocks: []presentation.ResolvedBlock{{ID: "voice", Kind: presentation.BlockVoiceStory, Version: 1, Variant: "transcript", Content: presentation.VoiceStoryContent{Heading: "Speak your mind.", ProviderQualification: "AI summaries use your configured provider."}}}},
		Capabilities: []presentation.ResolvedCapability{{ID: "remote", Label: "Remote computers", Status: presentation.CapabilityComingSoon, StatusLabel: "Coming soon"}},
		Diagnostics:  presentation.Diagnostics{RequestedRoute: "/", ResolvedRoute: "/", RequestedVariant: "campaign-one", ResolvedVariant: "campaign-one", ResolvedRevision: strings.Repeat("a", 64), BlockDigest: "sha256:" + strings.Repeat("b", 64)},
	}
	wire, err := ResolvedPresentationProto(value)
	if err != nil {
		t.Fatal(err)
	}
	if wire.GetDiagnostics().GetResolvedVariant() != "campaign-one" || wire.GetAppKey() != "web-console" || wire.GetCapabilities()[0].GetStatus() != "coming-soon" {
		t.Fatal("identity or roadmap state was altered")
	}
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(wire)
	if err != nil || !json.Valid(data) {
		t.Fatalf("invalid public wire payload: %v", err)
	}
	for _, field := range []string{"preservation_ref", "owner_qualification", "private_evidence_refs", "measurement_ref"} {
		if strings.Contains(string(data), field) {
			t.Fatalf("public wire contains %s", field)
		}
	}
}
