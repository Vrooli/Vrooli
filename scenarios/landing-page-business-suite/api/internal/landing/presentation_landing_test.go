package landing

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
)

func presentationLandingDocument(action *presentation.Action) presentation.Document {
	shell := presentation.ShellDisplay{
		BrandName: "Configured Suite", BrandMark: "suite", BrandTarget: "/",
		SkipLabel: "Skip to content", MenuLabel: "Menu", FooterBrandName: "Configured Suite",
		FooterBrandMark: "suite", FooterBrandTarget: "/", UnavailableReason: "Unavailable",
		PreviewLabel: "Private preview", HeaderAction: action,
	}
	page := presentation.Page{
		Display: presentation.PageDisplay{Shell: shell, AssetLabels: map[string]presentation.AssetLabel{"hero-art": {Alt: "Configured hero artwork"}}, FixtureDisplay: map[string]presentation.FixtureDisplay{}, Blocks: map[string]presentation.BlockDisplay{}, Apps: map[string]presentation.AppDisplay{}},
		ID:      "aquila-page", Locale: "en", Title: "Aquila", Description: "Configured app page",
		Theme:      presentation.Theme{Variant: "signal", Primary: "#263E38", Background: "#EAEADF", Accent: "#DF7958"},
		Navigation: presentation.Navigation{Label: "Navigation", Items: []presentation.NavigationItem{}},
		Footer:     presentation.Footer{Label: "Footer", Links: []presentation.NavigationItem{}},
		Blocks: []presentation.Block{
			{ID: "aquila-hero", Kind: presentation.BlockProductHero, Version: presentation.SchemaVersion, Variant: "centered", Content: presentation.ProductHeroContent{AppKey: "web-console", Title: "Aquila", Description: "Configured app", VisualRef: "hero-art", AccessibilityLabel: "Aquila product view", Actions: []presentation.Action{{Kind: presentation.ActionOpen, Label: "Open", AccessibleLabel: "Open Aquila", Target: "/apps/aquila"}}}},
			{ID: "aquila-closing", Kind: presentation.BlockClosingAction, Version: presentation.SchemaVersion, Variant: "plain", Content: presentation.ClosingActionContent{Heading: "Continue", Description: "Continue with Aquila", Actions: []presentation.Action{{Kind: presentation.ActionAnchor, Label: "Continue", AccessibleLabel: "Continue", Target: "#aquila-hero"}}}},
		},
	}
	page.Display.Apps["web-console"] = presentation.AppDisplay{VisualRef: "hero-art", Mark: "letter-a", Tone: "amber", DetailLabel: "Explore Aquila"}
	page.Display.Blocks["aquila-hero"] = presentation.BlockDisplay{AccessibilityLabel: "Aquila product view"}
	page.Display.Blocks["aquila-closing"] = presentation.BlockDisplay{}
	return presentation.Document{
		SchemaVersion: presentation.SchemaVersion,
		Bundle:        presentation.Bundle{Key: "business-suite", Name: "Business Suite", AppOrder: []string{"web-console"}, MaxAppSlides: 1, PageID: "aquila-page", EmptyPageID: "aquila-page", DefaultLocale: "en", Locales: []string{"en"}},
		Apps:          []presentation.App{{Key: "web-console", Slug: "aquila", Name: "Aquila", Enabled: true, Visibility: presentation.VisibilityPublic, Publication: presentation.PublicationPublished, PageID: "aquila-page", Tagline: "A configured app", Description: "A configured app"}},
		Pages:         []presentation.Page{page}, Assets: []presentation.Asset{{ID: "hero-art", ReleaseRef: "release-hero", ContentHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Width: 1440, Height: 720, MIME: "image/png", Surface: "web-hero", CropPolicy: "center", Provenance: presentation.AssetProvenance{Provider: "test", JobRef: "job", CandidateRef: "candidate"}, OverlayRegions: []presentation.OverlayRegion{{Name: "copy", Width: 0.4, Height: 0.4, Measurement: presentation.LegibilityMeasurement{ContrastRatio: 7, MinimumContrastRatio: 4.5, Threshold: 4.5, Verdict: presentation.LegibilityPass, MeasurementRef: "measurement"}}}, PublicURL: "/assets/hero-art.png", PrivateEvidenceRefs: []string{"asset-evidence"}}}, Fixtures: []presentation.Fixture{},
	}
}

func publishedPresentationLandingFixture(t *testing.T, document presentation.Document) *experimentation.ConfigStore {
	t.Helper()
	store, _ := publishedPresentationLandingFixtureWithRoots(t, document)
	return store
}

func publishedPresentationLandingFixtureWithRoots(t *testing.T, document presentation.Document) (*experimentation.ConfigStore, *filerouting.RoutedRoots) {
	t.Helper()
	variantsDir := t.TempDir()
	store := experimentation.NewConfigStore(variantsDir, "", nil)
	if err := store.SaveVariant("control", &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: "control", Name: "Control", Weight: 100, Status: "active",
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}}); err != nil {
		t.Fatalf("SaveVariant() error = %v", err)
	}
	roots := filerouting.New(storage.Paths{ConfigDir: t.TempDir()})
	store.SetPresentationStorage(roots, func(context.Context, presentation.Document) error { return nil })
	state, err := store.SavePresentationDraft(context.Background(), "control", document, 0)
	if err != nil {
		t.Fatalf("SavePresentationDraft() error = %v", err)
	}
	if _, err := store.PublishPresentation(context.Background(), "control", state.DraftRevision, state.Generation); err != nil {
		t.Fatalf("PublishPresentation() error = %v", err)
	}
	return store, roots
}

func TestCorruptActivePublicationDoesNotImplicitlyRollback(t *testing.T) { // [REQ:LP-PRES-012]
	ctx := context.Background()
	priorDocument := presentationLandingDocument(nil)
	priorDocument.Bundle.Key = "previous-bundle"
	store, roots := publishedPresentationLandingFixtureWithRoots(t, priorDocument)
	prior, err := store.GetPresentationState(ctx, "control")
	if err != nil {
		t.Fatal(err)
	}
	draft, err := store.SavePresentationDraft(ctx, "control", presentationLandingDocument(nil), prior.Generation)
	if err != nil {
		t.Fatal(err)
	}
	active, err := store.PublishPresentation(ctx, "control", draft.DraftRevision, draft.Generation)
	if err != nil {
		t.Fatal(err)
	}
	root, err := roots.PickRequired(ctx, storage.ClassConfig)
	if err != nil {
		t.Fatal(err)
	}
	corruptPath := filepath.Join(root, "presentations", "control", "revisions", active.ActiveRevision+".json")
	if err := os.WriteFile(corruptPath, []byte(`{"corrupted":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ownerReads := 0
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
		ownerReads++
		return nil, nil, nil
	}}
	for _, route := range []string{"/", "/apps/aquila"} {
		response, err := service.GetLandingConfigForRequest(ctx, "control", route, "en", "")
		if response != nil || !errors.Is(err, experimentation.ErrPresentationCorrupt) {
			t.Fatalf("corrupt active publication silently substituted a retained page for %s: response=%+v err=%v", route, response, err)
		}
	}
	if ownerReads != 0 {
		t.Fatalf("invalid publication reached commerce owners: %d", ownerReads)
	}
	retained, err := store.GetPresentationRevision(ctx, "control", prior.ActiveRevision)
	if err != nil || retained.Document.Bundle.Key != "previous-bundle" {
		t.Fatalf("prior recovery revision was lost: %+v %v", retained, err)
	}
	after, err := store.GetPresentationState(ctx, "control")
	if err != nil || after.Generation != active.Generation || after.ActiveRevision != active.ActiveRevision || len(after.PublishedRevisions) != 2 {
		t.Fatalf("failed public read mutated publication state: %+v %v", after, err)
	}
}

type presentationExposureRecorder struct {
	calls  int
	ctx    context.Context
	args   []string
	result bool
}

func (r *presentationExposureRecorder) RecordPresentationExposure(ctx context.Context, visitorID, variantSlug, revision, route, locale, blockDigest, weightFingerprint string) (bool, error) {
	r.calls++
	r.ctx = ctx
	r.args = []string{visitorID, variantSlug, revision, route, locale, blockDigest, weightFingerprint}
	return r.result, nil
}

func testPresentationOwnerJoin(_ context.Context, _ string) (*commerce.PricingOverview, []delivery.App, error) {
	return &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}}, []delivery.App{{BundleKey: "business-suite", AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{}}}, nil
}

func TestLocaleFallbackIsReportedWithoutChangingAppScope(t *testing.T) { // [REQ:LP-PRES-012]
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(nil))
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: testPresentationOwnerJoin}
	for _, route := range []string{"/", "/apps/aquila"} {
		response, err := service.GetLandingConfigForRequest(context.Background(), "control", route, "fr", "")
		if err != nil {
			t.Fatal(err)
		}
		diagnostics := response.Presentation.Diagnostics
		if !response.Fallback || !diagnostics.Fallback || diagnostics.FallbackReason != "locale_unavailable" {
			t.Fatalf("locale fallback was concealed: response flag=%v, diagnostics=%+v", response.Fallback, diagnostics)
		}
		if diagnostics.Locale != "en" || diagnostics.AppKey != "web-console" || diagnostics.ResolvedRoute != route || diagnostics.RequestedVariant != "control" || diagnostics.ResolvedVariant != "control" {
			t.Fatalf("fallback changed app, route or variant identity: %+v", diagnostics)
		}
	}
}

func TestGetLandingConfigForRequestPinsPublishedRevisionAcrossRootAndDetail(t *testing.T) { // [REQ:LP-PRES-011]
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(nil))
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
		return &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}}, []delivery.App{{BundleKey: "business-suite", AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{}}, {BundleKey: "business-suite", AppKey: "browser-automation-studio", Name: "Private BAS", Platforms: []delivery.Asset{}}}, nil
	}}

	root, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
	if err != nil {
		t.Fatalf("root resolution error = %v", err)
	}
	detail, err := service.GetLandingConfigForRequest(context.Background(), "control", "/apps/aquila", "en", "")
	if err != nil {
		t.Fatalf("detail resolution error = %v", err)
	}
	if root.Presentation == nil || detail.Presentation == nil {
		t.Fatal("published presentation was not returned")
	}
	if root.Presentation.Diagnostics.ResolvedRevision != detail.Presentation.Diagnostics.ResolvedRevision {
		t.Fatalf("root/detail revisions differ: %q != %q", root.Presentation.Diagnostics.ResolvedRevision, detail.Presentation.Diagnostics.ResolvedRevision)
	}
	if root.Presentation.Diagnostics.BlockDigest != detail.Presentation.Diagnostics.BlockDigest {
		t.Fatalf("root/detail content digest differs: %q != %q", root.Presentation.Diagnostics.BlockDigest, detail.Presentation.Diagnostics.BlockDigest)
	}
	if len(root.Downloads) != 1 || root.Downloads[0].AppKey != "web-console" {
		t.Fatalf("root delivery join was not scoped to eligible public keys: %#v", root.Downloads)
	}
	if len(detail.Downloads) != 1 || detail.Downloads[0].AppKey != "web-console" {
		t.Fatalf("detail delivery join was not scoped to current app: %#v", detail.Downloads)
	}
}

func TestRetainedPublicationDoesNotReadmitDeletedOrArchivedVariant(t *testing.T) { // [REQ:LP-PRES-009]
	for _, revoke := range []string{"delete", "archive"} {
		t.Run(revoke, func(t *testing.T) {
			store := publishedPresentationLandingFixture(t, presentationLandingDocument(nil))
			service := &LandingConfigService{configStore: store, presentationOwnerJoin: testPresentationOwnerJoin}
			before, err := store.GetPublishedPresentation(context.Background(), "control")
			if err != nil {
				t.Fatal(err)
			}
			if revoke == "delete" {
				if err := store.DeleteVariant("control"); err != nil {
					t.Fatal(err)
				}
			} else {
				snapshot, err := store.GetVariant("control")
				if err != nil {
					t.Fatal(err)
				}
				copySnapshot := *snapshot
				copySnapshot.Variant.Status = "archived"
				if err := store.SaveVariant("control", &copySnapshot); err != nil {
					t.Fatal(err)
				}
			}
			for _, route := range []string{"/", "/apps/aquila"} {
				response, err := service.GetLandingConfigForRequest(context.Background(), "control", route, "en", "")
				want := presentation.ErrUnavailable
				if route != "/" {
					want = presentation.ErrNotFound
				}
				if response != nil || !errors.Is(err, want) {
					t.Fatalf("revoked %s leaked public publication: response=%v error=%v", route, response, err)
				}
			}
			retained, err := store.GetPublishedPresentation(context.Background(), "control")
			if err != nil || retained.Revision != before.Revision {
				t.Fatalf("revocation destroyed immutable history: %v", err)
			}
		})
	}
}

func TestGetLandingConfigForRequestDoesNotUseGenericFallbackForDetailWithoutPublication(t *testing.T) { // [REQ:LP-PRES-009]
	store := experimentation.NewConfigStore("", "", nil)
	service := &LandingConfigService{configStore: store}
	if _, err := service.GetLandingConfigForRequest(context.Background(), "control", "/apps/aquila", "en", ""); !errors.Is(err, presentation.ErrNotFound) {
		t.Fatalf("unpublished detail error = %v, want ErrNotFound", err)
	}
}

func TestGetLandingConfigForRequestDoesNotUseGenericFallbackForRootWithoutPublication(t *testing.T) { // [REQ:LP-PRES-009]
	store := experimentation.NewConfigStore("", "", nil)
	service := &LandingConfigService{configStore: store}
	_, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
	if !errors.Is(err, presentation.ErrUnavailable) {
		t.Fatalf("unpublished root error = %v, want ErrUnavailable", err)
	}
}

func TestGetLandingConfigForRequestFailsClosedWhenNoVariantIsSelectable(t *testing.T) {
	store := experimentation.NewConfigStore("", "", nil)
	service := &LandingConfigService{configStore: store}
	if _, err := service.GetLandingConfigForRequest(context.Background(), "", "/", "en", "visitor-1"); !errors.Is(err, presentation.ErrUnavailable) {
		t.Fatalf("no-selectable root error = %v, want ErrUnavailable", err)
	}
}

func TestGetLandingConfigForRequestFailsClosedForPrivateOrUnknownDetail(t *testing.T) { // [REQ:LP-PRES-009]
	document := presentationLandingDocument(nil)
	privatePage := document.Pages[0]
	privatePage.ID = "private-page"
	privatePage.Blocks = append([]presentation.Block(nil), document.Pages[0].Blocks...)
	privatePage.Display.Apps = map[string]presentation.AppDisplay{"private": {VisualRef: "hero-art", Mark: "letter-a", Tone: "amber", DetailLabel: "Explore Private"}}
	privatePage.Blocks[0].Content = presentation.ProductHeroContent{AppKey: "private", Title: "Private", Description: "Private app", VisualRef: "hero-art", AccessibilityLabel: "Private product view", Actions: []presentation.Action{{Kind: presentation.ActionOpen, Label: "Open", AccessibleLabel: "Open Private", Target: "/apps/private"}}}
	document.Pages = append(document.Pages, privatePage)
	document.Apps = append(document.Apps, presentation.App{Key: "private", Slug: "private", Name: "Private", Enabled: true, Visibility: presentation.VisibilityPrivate, Publication: presentation.PublicationDraft, PageID: "private-page", Tagline: "Private app", Description: "Private"})
	store := publishedPresentationLandingFixture(t, document)
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: testPresentationOwnerJoin}
	for _, route := range []string{"/apps/private", "/apps/unknown"} {
		if _, err := service.GetLandingConfigForRequest(context.Background(), "control", route, "en", ""); !errors.Is(err, presentation.ErrNotFound) {
			t.Fatalf("route %q error = %v, want ErrNotFound", route, err)
		}
	}
}

func TestGetLandingConfigForRequestUsesOwnerKeysForHeaderActions(t *testing.T) { // [REQ:LP-PRES-011]
	artifactID := int64(11)
	plan := &commerce.PlanOption{StripePriceId: "price-pro", BundleKey: "business-suite", DisplayEnabled: true}
	readyDownload := delivery.App{BundleKey: "business-suite", AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{{BundleKey: "business-suite", AppKey: "web-console", Platform: "linux", ReleaseVersion: "1.0.0", ArtifactID: &artifactID}}}
	for _, test := range []struct {
		name       string
		action     presentation.Action
		pricing    *commerce.PricingOverview
		downloads  []delivery.App
		wantStatus presentation.ActionStatus
	}{
		{name: "unreleased download row is unavailable", action: presentation.Action{Kind: presentation.ActionDownload, Label: "Download", AccessibleLabel: "Download", AppKey: "web-console"}, pricing: &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}}, downloads: []delivery.App{{BundleKey: "business-suite", AppKey: "web-console", Name: "Aquila"}, {BundleKey: "business-suite", AppKey: "browser-automation-studio", Name: "BAS"}}, wantStatus: presentation.ResolvedActionUnavailable},
		{name: "unknown purchase plan is unavailable", action: presentation.Action{Kind: presentation.ActionPurchase, Label: "Buy", AccessibleLabel: "Buy", PlanRef: "price-pro"}, pricing: &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}}, downloads: []delivery.App{readyDownload}, wantStatus: presentation.ResolvedActionUnavailable},
		{name: "supported download row is ready", action: presentation.Action{Kind: presentation.ActionDownload, Label: "Download", AccessibleLabel: "Download", AppKey: "web-console"}, pricing: &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}}, downloads: []delivery.App{readyDownload}, wantStatus: presentation.ResolvedActionReady},
		{name: "displayed matching plan is ready", action: presentation.Action{Kind: presentation.ActionPurchase, Label: "Buy", AccessibleLabel: "Buy", PlanRef: "price-pro"}, pricing: &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}, Monthly: []*commerce.PlanOption{plan}}, downloads: nil, wantStatus: presentation.ResolvedActionReady},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := presentationLandingDocument(&test.action)
			store := publishedPresentationLandingFixture(t, document)
			service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
				return test.pricing, test.downloads, nil
			}}
			response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
			if err != nil {
				t.Fatalf("error = %v, want readable page with action observation", err)
			}
			if response.Presentation == nil || len(response.Presentation.Actions) < 1 {
				t.Fatalf("actions = %#v, want header and block observations", response.Presentation)
			}
			if response.Presentation.Actions[0].Status != test.wantStatus {
				t.Fatalf("action status = %q, want %q", response.Presentation.Actions[0].Status, test.wantStatus)
			}
			if test.action.Kind == presentation.ActionDownload && test.wantStatus == presentation.ResolvedActionReady && response.Presentation.Actions[0].Href != "/apps/aquila/download" {
				t.Fatalf("download action does not reach app chooser: %q", response.Presentation.Actions[0].Href)
			}
		})
	}
}

func TestGetLandingConfigForRequestRejectsMismatchedOwnerBundles(t *testing.T) {
	artifactID := int64(11)
	canonical := "business-suite"
	for _, test := range []struct {
		name      string
		action    presentation.Action
		pricing   *commerce.PricingOverview
		downloads []delivery.App
	}{
		{
			name:    "pricing bundle differs from presentation",
			action:  presentation.Action{Kind: presentation.ActionPurchase, Label: "Buy", AccessibleLabel: "Buy", PlanRef: "price-pro"},
			pricing: &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "other-bundle"}, Monthly: []*commerce.PlanOption{{StripePriceId: "price-pro", BundleKey: "other-bundle", DisplayEnabled: true}}},
		},
		{
			name:    "plan bundle differs from pricing bundle",
			action:  presentation.Action{Kind: presentation.ActionPurchase, Label: "Buy", AccessibleLabel: "Buy", PlanRef: "price-pro"},
			pricing: &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: canonical}, Monthly: []*commerce.PlanOption{{StripePriceId: "price-pro", BundleKey: "other-bundle", DisplayEnabled: true}}},
		},
		{
			name:      "download bundle differs from presentation",
			action:    presentation.Action{Kind: presentation.ActionDownload, Label: "Download", AccessibleLabel: "Download", AppKey: "web-console"},
			pricing:   &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: canonical}},
			downloads: []delivery.App{{BundleKey: "other-bundle", AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{{AppKey: "web-console", Platform: "linux", ReleaseVersion: "1.0.0", ArtifactID: &artifactID}}}},
		},
		{
			name:      "platform bundle differs from its app",
			action:    presentation.Action{Kind: presentation.ActionDownload, Label: "Download", AccessibleLabel: "Download", AppKey: "web-console"},
			pricing:   &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: canonical}},
			downloads: []delivery.App{{BundleKey: canonical, AppKey: "web-console", Platforms: []delivery.Asset{{BundleKey: "private-bundle", AppKey: "web-console", Platform: "linux", ReleaseVersion: "1.0.0", ArtifactID: &artifactID}}}},
		},
		{
			name:      "platform app differs from its parent",
			action:    presentation.Action{Kind: presentation.ActionDownload, Label: "Download", AccessibleLabel: "Download", AppKey: "web-console"},
			pricing:   &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: canonical}},
			downloads: []delivery.App{{BundleKey: canonical, AppKey: "web-console", Platforms: []delivery.Asset{{BundleKey: canonical, AppKey: "private-app", Platform: "linux", ReleaseVersion: "1.0.0", ArtifactID: &artifactID}}}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := publishedPresentationLandingFixture(t, presentationLandingDocument(&test.action))
			service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
				return test.pricing, test.downloads, nil
			}}
			response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
			if err != nil || response == nil || response.Presentation == nil {
				t.Fatalf("owner mismatch discarded marketing page: response=%#v error=%v", response, err)
			}
			if response.Pricing != nil || len(response.Downloads) != 0 {
				t.Fatalf("mismatched owner data was projected publicly: pricing=%#v downloads=%#v", response.Pricing, response.Downloads)
			}
			if len(response.Presentation.Actions) == 0 || response.Presentation.Actions[0].Status != presentation.ResolvedActionUnavailable {
				t.Fatalf("mismatched owner action status = %#v, want unavailable", response.Presentation.Actions)
			}
		})
	}
}

func TestGetLandingConfigForRequestUsesValidatedOwnerWebURLForAppOpen(t *testing.T) {
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(&presentation.Action{Kind: presentation.ActionOpen, Label: "Open", AccessibleLabel: "Open Aquila", AppKey: "web-console"}))
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
		return &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}}, []delivery.App{{BundleKey: "business-suite", AppKey: "web-console", Name: "Aquila", Metadata: map[string]interface{}{"web_url": "/app/web-console"}, Platforms: []delivery.Asset{}}}, nil
	}}
	response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
	if err != nil {
		t.Fatalf("web-only owner join error = %v", err)
	}
	if len(response.Downloads) != 1 || response.Downloads[0].Metadata["web_url"] != "/app/web-console" {
		t.Fatalf("validated web URL was not preserved in public owner row: %#v", response.Downloads)
	}
	var found bool
	for _, action := range response.Presentation.Actions {
		if action.AppKey == "web-console" {
			found = true
			if action.Status != presentation.ResolvedActionReady || action.Href != "/app/web-console" {
				t.Fatalf("web-only app open action = %#v, want ready owner URL", action)
			}
		}
	}
	if !found {
		t.Fatalf("web-only app open action was not projected: %#v", response.Presentation.Actions)
	}
}

func TestGetLandingConfigForRequestStripsUnsafeOwnerWebURLAndKeepsInstallerTruthful(t *testing.T) {
	for _, raw := range []string{"//evil.example/app", `/\\evil`, "javascript:alert(1)", "http://example.test/app", "/app/../admin"} {
		t.Run(raw, func(t *testing.T) {
			store := publishedPresentationLandingFixture(t, presentationLandingDocument(&presentation.Action{Kind: presentation.ActionOpen, Label: "Open", AccessibleLabel: "Open Aquila", AppKey: "web-console"}))
			service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
				return &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}}, []delivery.App{{BundleKey: "business-suite", AppKey: "web-console", Name: "Aquila", Metadata: map[string]interface{}{"web_url": raw}, Platforms: []delivery.Asset{}}}, nil
			}}
			response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
			if err != nil {
				t.Fatalf("unsafe web URL caused page failure: %v", err)
			}
			if _, ok := response.Downloads[0].Metadata["web_url"]; ok {
				t.Fatalf("unsafe web URL leaked into public metadata: %#v", response.Downloads[0].Metadata)
			}
			if response.Presentation.Actions[0].Status != presentation.ResolvedActionUnavailable {
				t.Fatalf("unsafe web URL made app open ready: %#v", response.Presentation.Actions)
			}
		})
	}
}

func TestGetLandingConfigForRequestKeepsPageReadableWhenOwnerIsUnavailable(t *testing.T) { // [REQ:LP-PRES-011]
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(&presentation.Action{Kind: presentation.ActionPurchase, Label: "Buy", AccessibleLabel: "Buy", PlanRef: "price-pro"}))
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
		return nil, nil, errors.New("pricing provider unavailable")
	}}
	response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
	if err != nil || response == nil || response.Presentation == nil {
		t.Fatalf("owner outage discarded page: response=%#v error=%v", response, err)
	}
	if len(response.Presentation.Actions) < 1 || response.Presentation.Actions[0].Status != presentation.ResolvedActionUnavailable || response.Presentation.Actions[0].Reason != "Unavailable" {
		t.Fatalf("owner outage action observation = %#v", response.Presentation.Actions)
	}
}

func TestGetLandingConfigForRequestKeepsDownloadsWhenPricingOwnerIsUnavailable(t *testing.T) {
	artifactID := int64(42)
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(&presentation.Action{Kind: presentation.ActionDownload, Label: "Get Aquila", AccessibleLabel: "Get Aquila", AppKey: "web-console"}))
	service := &LandingConfigService{
		configStore: store,
		presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
			return nil, nil, errors.New("pricing catalog unavailable")
		},
		presentationDownloadJoin: func(context.Context, string) ([]delivery.App, error) {
			return []delivery.App{{BundleKey: "business-suite", AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{{BundleKey: "business-suite", AppKey: "web-console", Platform: "linux", ReleaseVersion: "0.0.1", ArtifactID: &artifactID}}}}, nil
		},
	}

	response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "")
	if err != nil || response == nil || len(response.Downloads) != 1 {
		t.Fatalf("download owner was discarded with pricing outage: response=%#v error=%v", response, err)
	}
	if response.Pricing != nil {
		t.Fatalf("pricing should remain unavailable: %#v", response.Pricing)
	}
	if response.Presentation.Actions[0].Status != presentation.ResolvedActionReady {
		t.Fatalf("download action = %#v, want ready", response.Presentation.Actions[0])
	}
}

func TestGetLandingConfigForRequestDoesNotExposeEmptyVisitor(t *testing.T) { // [REQ:LP-PRES-011]
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(nil))
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: testPresentationOwnerJoin}

	response, err := service.GetLandingConfigForRequest(context.Background(), "", "/", "en", "")
	if err != nil || response == nil || response.Presentation == nil {
		t.Fatalf("empty visitor resolution failed: response=%#v error=%v", response, err)
	}
	if response.Presentation.Diagnostics.RequestedVariant != "" {
		t.Fatalf("requested variant = %q, want raw empty request", response.Presentation.Diagnostics.RequestedVariant)
	}
	if response.Presentation.Diagnostics.AssignmentSource != "" || response.Presentation.Diagnostics.WeightFingerprint != "" {
		t.Fatalf("anonymous resolution carried assignment facts: source=%q fingerprint=%q", response.Presentation.Diagnostics.AssignmentSource, response.Presentation.Diagnostics.WeightFingerprint)
	}

	identified, err := service.GetLandingConfigForRequest(context.Background(), "", "/", "en", "visitor-1")
	if err != nil {
		t.Fatalf("identified visitor resolution failed: %v", err)
	}
	if identified.Presentation.Diagnostics.AssignmentSource != string(PresentationAssignmentWeightedVisitor) || identified.Presentation.Diagnostics.WeightFingerprint != experimentation.WeightFingerprint(store.ListVariants()) {
		t.Fatalf("identified resolution assignment facts = source:%q fingerprint:%q", identified.Presentation.Diagnostics.AssignmentSource, identified.Presentation.Diagnostics.WeightFingerprint)
	}
}

func TestGetLandingConfigForRequestMarksExplicitVariantWithoutExposureFacts(t *testing.T) {
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(nil))
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: testPresentationOwnerJoin}
	response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "visitor-1")
	if err != nil {
		t.Fatalf("explicit variant resolution failed: %v", err)
	}
	if response.Presentation.Diagnostics.AssignmentSource != string(PresentationAssignmentExplicitURL) || response.Presentation.Diagnostics.WeightFingerprint != "" {
		t.Fatalf("explicit resolution assignment facts = source:%q fingerprint:%q", response.Presentation.Diagnostics.AssignmentSource, response.Presentation.Diagnostics.WeightFingerprint)
	}
}

func TestGetLandingConfigForRequestSanitizesOwnerPricingBeforePublicProjection(t *testing.T) {
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(nil))
	privatePlan := &commerce.PlanOption{PlanName: "private", BundleKey: "business-suite", StripePriceId: "price-private", DisplayEnabled: false}
	publicPlan := &commerce.PlanOption{PlanName: "Public", BundleKey: "business-suite", StripePriceId: "price-public", DisplayEnabled: true, AmountCents: 1200, Metadata: map[string]*common.JsonValue{"private": {}}}
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: func(context.Context, string) (*commerce.PricingOverview, []delivery.App, error) {
		return &commerce.PricingOverview{Bundle: &commerce.BundleProduct{BundleKey: "business-suite"}, Monthly: []*commerce.PlanOption{privatePlan, publicPlan}}, nil, nil
	}}
	response, err := service.GetLandingConfigForRequest(context.Background(), "control", "/", "en", "visitor-1")
	if err != nil {
		t.Fatalf("typed pricing resolution failed: %v", err)
	}
	if len(response.Pricing.Monthly) != 1 || response.Pricing.Monthly[0].StripePriceId != "price-public" || response.Pricing.Monthly[0].Metadata != nil {
		t.Fatalf("typed pricing was not sanitized before projection: %#v", response.Pricing.Monthly)
	}
}

func TestRecordPresentationExposureValidatesPinnedPublicProof(t *testing.T) {
	store := publishedPresentationLandingFixture(t, presentationLandingDocument(nil))
	recorder := &presentationExposureRecorder{result: true}
	service := &LandingConfigService{configStore: store, presentationOwnerJoin: testPresentationOwnerJoin}
	service.UsePresentationExposureRecorder(recorder)
	ctx := context.WithValue(context.Background(), struct{}{}, "lease")
	response, err := service.GetLandingConfigForRequest(ctx, "", "/", "en", "visitor-1")
	if err != nil || response == nil || response.Presentation == nil {
		t.Fatalf("read-only public resolution failed: response=%#v error=%v", response, err)
	}
	diagnostics := response.Presentation.Diagnostics
	recorded, err := service.RecordPresentationExposure(ctx, PresentationExposureRequest{
		VisitorID: "visitor-1", VariantSlug: diagnostics.ResolvedVariant, Revision: diagnostics.ResolvedRevision,
		Route: diagnostics.ResolvedRoute, Locale: diagnostics.Locale, BlockDigest: diagnostics.BlockDigest,
		WeightFingerprint: experimentation.WeightFingerprint(store.ListVariants()), Source: PresentationAssignmentWeightedVisitor,
	})
	if err != nil || !recorded || recorder.calls != 1 || recorder.ctx != ctx {
		t.Fatalf("valid exposure = recorded:%v error:%v calls:%d ctx:%v", recorded, err, recorder.calls, recorder.ctx == ctx)
	}
	if _, err := service.RecordPresentationExposure(ctx, PresentationExposureRequest{
		VisitorID: "visitor-1", VariantSlug: diagnostics.ResolvedVariant, Revision: diagnostics.ResolvedRevision,
		Route: diagnostics.ResolvedRoute, Locale: diagnostics.Locale, BlockDigest: diagnostics.BlockDigest,
		WeightFingerprint: experimentation.WeightFingerprint(store.ListVariants()), Source: PresentationAssignmentExplicitURL,
	}); !errors.Is(err, presentation.ErrUnavailable) {
		t.Fatalf("explicit URL exposure error = %v, want ErrUnavailable", err)
	}
	if recorder.calls != 1 {
		t.Fatalf("explicit URL exposure reached recorder: %d calls", recorder.calls)
	}
}

func TestPublicOwnerSnapshotRefIsStableAndExcludesPrivateDeliveryFields(t *testing.T) {
	first := int64(7)
	second := int64(99)
	left := []delivery.App{{AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{{ID: first, AppKey: "web-console", Platform: "linux", ReleaseVersion: "1.0.0", ArtifactURL: "private-a", ArtifactID: &first, Checksum: "sha256:a", Metadata: map[string]interface{}{"secret": "one"}}}}}
	right := []delivery.App{{AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{{ID: second, AppKey: "web-console", Platform: "linux", ReleaseVersion: "1.0.0", ArtifactURL: "private-b", ArtifactID: &second, Checksum: "sha256:a", Metadata: map[string]interface{}{"secret": "two"}}}}}
	if got, want := publicOwnerSnapshotRef(nil, left), publicOwnerSnapshotRef(nil, right); got != want {
		t.Fatalf("private delivery fields changed public snapshot ref: %q != %q", got, want)
	}

	changed := []delivery.App{{AppKey: "web-console", Name: "Aquila", Platforms: []delivery.Asset{{Platform: "linux", ReleaseVersion: "2.0.0", Checksum: "sha256:a"}}}}
	if publicOwnerSnapshotRef(nil, left) == publicOwnerSnapshotRef(nil, changed) {
		t.Fatal("public release change did not change snapshot ref")
	}
}
