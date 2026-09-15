package presentation

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func testApp(key, slug, page string, visibility Visibility, publication Publication, enabled bool) App {
	return App{Key: key, Slug: slug, Name: strings.Title(strings.ReplaceAll(slug, "-", " ")), Enabled: enabled, Visibility: visibility, Publication: publication, PageID: page, Tagline: "A configured tagline", Description: "A configured description"}
}

func testShellPage(id, locale, title string, blocks ...Block) Page {
	return Page{Display: testDisplay(), ID: id, Locale: locale, Title: title, Description: "A configured page description", Theme: Theme{Variant: "signal", Primary: "#263E38", Background: "#EAEADF", Accent: "#DF7958"}, Navigation: Navigation{Label: "Main navigation", Items: []NavigationItem{{Label: "Overview", AccessibleLabel: "Overview", Target: "#overview"}}}, Blocks: blocks, Footer: Footer{Label: "Footer navigation", Links: []NavigationItem{{Label: "Home", AccessibleLabel: "Home", Target: "/"}}}}
}

func testDisplay() PageDisplay {
	return PageDisplay{Shell: ShellDisplay{BrandName: "Configured brand", BrandMark: "suite", BrandTarget: "/", SkipLabel: "Skip to content", MenuLabel: "Menu", FooterBrandName: "Configured brand", FooterBrandMark: "suite", FooterBrandTarget: "/", UnavailableReason: "Not available", PreviewLabel: "Private preview"}, AssetLabels: map[string]AssetLabel{}, FixtureDisplay: map[string]FixtureDisplay{}, Blocks: map[string]BlockDisplay{}, Apps: map[string]AppDisplay{}}
}

func testAsset() Asset {
	return Asset{ID: "hero-art", ReleaseRef: "release-hero", ContentHash: strings.Repeat("a", 64), Width: 1440, Height: 720, MIME: "image/png", Surface: "web-hero", CropPolicy: "center", Provenance: AssetProvenance{Provider: "test-provider", JobRef: "job-hero", CandidateRef: "candidate-hero"}, OverlayRegions: []OverlayRegion{{Name: "copy", Width: 0.4, Height: 0.4, Measurement: LegibilityMeasurement{ContrastRatio: 7, MinimumContrastRatio: 4.5, Threshold: 4.5, Verdict: LegibilityPass, MeasurementRef: "measurement-hero"}}}, PublicURL: "/assets/hero-art.png", PrivateEvidenceRefs: []string{"asset-evidence"}}
}

func testAppBlocks(app App) []Block {
	return []Block{{ID: app.Key + "-hero", Kind: BlockProductHero, Version: SchemaVersion, Variant: "centered", Content: ProductHeroContent{AppKey: app.Key, Title: app.Name, Description: app.Description, VisualRef: "hero-art", AccessibilityLabel: app.Name + " product view", Actions: []Action{{Kind: ActionOpen, Label: "Open app", AccessibleLabel: "Open " + app.Name, Target: "/apps/" + app.Slug}}}}, {ID: app.Key + "-closing", Kind: BlockClosingAction, Version: SchemaVersion, Variant: "plain", Content: ClosingActionContent{Heading: "Continue with " + app.Name, Description: "The configured next step.", Actions: []Action{{Kind: ActionAnchor, Label: "Continue", AccessibleLabel: "Continue", Target: "#overview"}}}}}
}

func testDocument(apps ...App) Document {
	order := make([]string, 0, len(apps))
	bundleBlocks := []Block{}
	if len(apps) > 0 {
		bundleBlocks = append(bundleBlocks, Block{ID: "bundle-spotlights", Kind: BlockAppSpotlights, Version: SchemaVersion, Variant: "grid", Content: AppSpotlightsContent{Heading: "Configured apps", AppKeys: make([]string, 0, len(apps)), DetailLinkLabel: "Explore app"}})
	}
	pages := []Page{testShellPage("bundle", "en", "Vrooli Business Suite", bundleBlocks...), testShellPage("empty", "en", "No apps are currently available")}
	for _, app := range apps {
		order = append(order, app.Key)
		pages[0].Display.Apps[app.Key] = AppDisplay{VisualRef: "hero-art", Mark: "letter-a", Tone: "amber", DetailLabel: "Explore " + app.Name}
		for i := range pages[0].Blocks {
			if content, ok := pages[0].Blocks[i].Content.(AppSpotlightsContent); ok {
				content.AppKeys = append(content.AppKeys, app.Key)
				pages[0].Blocks[i].Content = content
			}
		}
		pages = append(pages, testShellPage(app.PageID, "en", app.Name, testAppBlocks(app)...))
	}
	return Document{
		SchemaVersion: SchemaVersion,
		Bundle:        Bundle{Key: "business-suite", Name: "Vrooli", AppOrder: order, MaxAppSlides: 2, PageID: "bundle", EmptyPageID: "empty", DefaultLocale: "en", Locales: []string{"en", "fr"}},
		Apps:          apps,
		Pages:         pages,
		Assets:        []Asset{testAsset()},
	}
}

func TestLP_PRES_003_005_006_007_008_012_014_015_ResolutionMatrix(t *testing.T) {
	appA := testApp("web-console", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true)
	appB := testApp("backdrop-studio", "backdrop-studio", "backdrop-page", VisibilityPublic, PublicationPublished, true)
	private := testApp("browser-automation-studio", "browser-automation-studio", "bas-page", VisibilityPrivate, PublicationDraft, false)

	tests := []struct {
		name     string
		document Document
		request  ResolveRequest
		mode     Mode
		scope    Scope
		appKey   string
		selected []string
	}{
		{"zero public apps", testDocument(testApp("web-console", "aquila", "aquila-page", VisibilityPrivate, PublicationDraft, false)), ResolveRequest{Route: "/"}, ModeEmpty, ScopeBundle, "", []string{}},
		{"one public app", testDocument(appA), ResolveRequest{Route: "/"}, ModeSingleApp, ScopeApp, "web-console", []string{"web-console"}},
		{"bundle preserves configured order and cap", testDocument(appB, appA, private), ResolveRequest{Route: "/"}, ModeBundle, ScopeBundle, "", []string{"backdrop-studio", "web-console"}},
		{"zero spotlight cap remains bundle", func() Document { d := testDocument(appB, appA); d.Bundle.MaxAppSlides = 0; return d }(), ResolveRequest{Route: "/"}, ModeBundle, ScopeBundle, "", []string{}},
		{"detail is app scope", testDocument(appA, appB), ResolveRequest{Route: "/apps/aquila"}, ModeAppDetail, ScopeApp, "web-console", []string{"web-console"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Resolve(tt.document, tt.request)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if result.Mode != tt.mode || result.Scope != tt.scope || result.AppKey != tt.appKey {
				t.Fatalf("identity = (%s, %s, %q), want (%s, %s, %q)", result.Mode, result.Scope, result.AppKey, tt.mode, tt.scope, tt.appKey)
			}
			if !reflect.DeepEqual(result.SelectedAppKeys, tt.selected) {
				t.Fatalf("selected = %#v, want %#v", result.SelectedAppKeys, tt.selected)
			}
			if len(result.Spotlights) != len(tt.selected) || len(result.Spotlights) > tt.document.Bundle.MaxAppSlides && result.Mode == ModeBundle {
				t.Fatalf("spotlights = %d, selected = %#v", len(result.Spotlights), tt.selected)
			}
			if result.Diagnostics.EligibleAppKeys == nil {
				t.Fatal("eligible app keys must be present in diagnostics")
			}
		})
	}

	parityDocument := richDocument()
	root, err := Resolve(parityDocument, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := Resolve(parityDocument, ResolveRequest{Route: "/apps/aquila"})
	if err != nil {
		t.Fatal(err)
	}
	if root.Diagnostics.BlockDigest != detail.Diagnostics.BlockDigest {
		t.Fatalf("single/detail digest mismatch: %s != %s", root.Diagnostics.BlockDigest, detail.Diagnostics.BlockDigest)
	}
	if len(root.Page.Blocks) == 0 || len(detail.Page.Blocks) == 0 {
		t.Fatal("single-app root and detail must resolve the configured app blocks")
	}
	if len(root.Capabilities) == 0 || len(root.Assets) == 0 || len(root.Fixtures) == 0 {
		t.Fatal("parity fixture must exercise capabilities and referenced resources")
	}
	if !reflect.DeepEqual(root.Page.Blocks, detail.Page.Blocks) || !reflect.DeepEqual(root.Capabilities, detail.Capabilities) || !reflect.DeepEqual(root.Assets, detail.Assets) || !reflect.DeepEqual(root.Fixtures, detail.Fixtures) {
		t.Fatal("single-app root/detail content, capability, and resource parity mismatch")
	}
}

func TestLP_PRES_003_006_007_PrivateRoutesAndPublicProjection(t *testing.T) {
	document := testDocument(
		testApp("aquila", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true),
		testApp("bas", "browser-automation-studio", "bas-page", VisibilityPrivate, PublicationDraft, true),
	)
	if _, err := Resolve(document, ResolveRequest{Route: "/apps/browser-automation-studio"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("private route error = %v, want ErrNotFound", err)
	}
	if _, err := Resolve(document, ResolveRequest{Route: "/apps/unknown"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown route error = %v, want ErrNotFound", err)
	}

	preview, err := Resolve(document, ResolveRequest{Route: "/apps/browser-automation-studio", PreviewRevision: "draft-v1", PreviewAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Diagnostics.Preview || !preview.Diagnostics.NoIndex || !preview.Diagnostics.NoStore {
		t.Fatalf("preview guardrails missing: %#v", preview.Diagnostics)
	}
	if _, err := Resolve(document, ResolveRequest{Route: "/apps/browser-automation-studio", PreviewRevision: "draft-v1"}); !errors.Is(err, ErrPreviewUnauthorized) {
		t.Fatalf("unauthorized preview error = %v", err)
	}

	public, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "browser-automation-studio") || strings.Contains(string(b), "preservation") || strings.Contains(string(b), "evidence_refs") {
		t.Fatalf("public response leaked private data: %s", b)
	}
	if preview.AppKey != "bas" || len(preview.Page.Blocks) == 0 {
		t.Fatalf("preview did not expose the preserved disabled private profile: %#v", preview)
	}
}

func TestLP_PRES_003_ConfiguredVariantAssignmentIsNotInvented(t *testing.T) {
	document := testDocument(testApp("web-console", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true))
	result, err := Resolve(document, ResolveRequest{Route: "/", Variant: "signal", ResolvedVariant: "afterhours", VariantAssignment: "visitor-a"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Diagnostics.RequestedVariant != "signal" || result.Diagnostics.ResolvedVariant != "afterhours" || result.Diagnostics.Fallback {
		t.Fatalf("variant assignment was altered: %#v", result.Diagnostics)
	}
	result, err = Resolve(document, ResolveRequest{Route: "/", Variant: "operator-owned"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Diagnostics.ResolvedVariant != "operator-owned" || result.Diagnostics.Fallback {
		t.Fatalf("arbitrary configured variant was rejected or relabeled: %#v", result.Diagnostics)
	}
}

func TestLP_PRES_008_014_ResolutionDoesNotAliasDocument(t *testing.T) {
	document := richDocument()
	first, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated resolution is not deterministic")
	}
	firstPage := first.Page.Blocks[0].Content.(ProductHeroContent)
	firstPage.Actions[0].Label = "mutated response"
	first.Page.Blocks[0].Content = firstPage
	first.Fixtures[0].Workspace.Title = "mutated fixture"
	first.Assets[0].OverlayRegions[0].Measurement.Verdict = LegibilityFail
	if document.Fixtures[0].Workspace.Title == "mutated fixture" || document.Assets[0].OverlayRegions[0].Measurement.Verdict == LegibilityFail {
		t.Fatal("response mutation changed the configured document")
	}
	if second.Page.Blocks[0].Content.(ProductHeroContent).Actions[0].Label == "mutated response" || second.Fixtures[0].Workspace.Title == "mutated fixture" || second.Assets[0].OverlayRegions[0].Measurement.Verdict == LegibilityFail {
		t.Fatal("response mutation changed a repeated resolution")
	}
}

func TestLP_PRES_005_SpotlightBudgetIsPageWide(t *testing.T) {
	page := testShellPage("bundle", "en", "Bundle",
		Block{ID: "spotlights-one", Kind: BlockAppSpotlights, Version: SchemaVersion, Variant: "grid", Content: AppSpotlightsContent{Heading: "Apps", AppKeys: []string{"web-console", "backdrop-studio"}, DetailLinkLabel: "Open"}},
		Block{ID: "spotlights-two", Kind: BlockAppSpotlights, Version: SchemaVersion, Variant: "stacked", Content: AppSpotlightsContent{Heading: "More apps", AppKeys: []string{"web-console", "backdrop-studio"}, DetailLinkLabel: "Open"}},
	)
	resolved, err := projectPage(page, map[string]bool{"web-console": true, "backdrop-studio": true}, []string{"web-console", "backdrop-studio"}, ModeBundle, false, map[string]bool{"aquila": true, "backdrop-studio": true})
	if err != nil {
		t.Fatal(err)
	}
	first := resolved.Blocks[0].Content.(AppSpotlightsContent)
	second := resolved.Blocks[1].Content.(AppSpotlightsContent)
	if !reflect.DeepEqual(first.AppKeys, []string{"web-console", "backdrop-studio"}) || len(second.AppKeys) != 0 {
		t.Fatalf("spotlight budget was applied per block: first=%#v second=%#v", first.AppKeys, second.AppKeys)
	}
}

func TestLP_PRES_012_ContentDigestPropagatesEncodingErrors(t *testing.T) {
	_, err := ContentDigest(ResolvedPage{Blocks: []ResolvedBlock{{Kind: BlockVoiceStory, Content: VoiceStoryContent{Waveform: []float64{math.NaN()}}}}})
	if err == nil {
		t.Fatal("ContentDigest returned a digest for non-JSON content")
	}
}

func TestLP_PRES_014_FixtureResourcesResolveAndRedactEvidence(t *testing.T) {
	document := richDocument()
	document.Fixtures = append(document.Fixtures, Fixture{ID: "backdrop-demo", Kind: FixtureBackdrop, Backdrop: &BackdropFixture{Title: "Backdrop", Label: "Backdrop", Selected: "Hero", Styles: []string{"Signal"}, Surface: "web-hero", Palette: "dark", Panel: "preview", Caption: "Configured artwork", Export: "PNG", Options: []string{"Desktop"}, Badge: "Ready", AssetRefs: []string{"hero-art"}}})
	page := &document.Pages[2]
	page.Display.FixtureDisplay["backdrop-demo"] = FixtureDisplay{Mark: "landscape"}
	page.Blocks[0].Content = ProductHeroContent{AppKey: "web-console", Title: "All agents", Description: "One view", FixtureRef: "backdrop-demo", AccessibilityLabel: "Illustrative workspace", Actions: []Action{{Kind: ActionOpen, Label: "Open", AccessibleLabel: "Open", Target: "/apps/aquila"}}}
	result, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fixtures) != 1 || result.Fixtures[0].ID != "backdrop-demo" || len(result.Assets) != 1 || result.Assets[0].ID != "hero-art" {
		t.Fatalf("fixture resource closure = fixtures %#v assets %#v", result.Fixtures, result.Assets)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(data)
	for _, private := range []string{"private-capture-1", "job-1", "candidate-1", "measure-1", "private_evidence_refs", "measurement_ref"} {
		if strings.Contains(serialized, private) {
			t.Fatalf("public resource leaked %q: %s", private, serialized)
		}
	}
}

func TestLP_PRES_007_PublicActionsAndWrongProductHeroAreNotLeaked(t *testing.T) {
	document := testDocument(
		testApp("web-console", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true),
		testApp("bas", "browser-automation-studio", "bas-page", VisibilityPrivate, PublicationDraft, false),
	)
	page := &document.Pages[2]
	page.Blocks = append(page.Blocks, Block{ID: "private-action", Kind: BlockClosingAction, Version: SchemaVersion, Variant: "plain", Content: ClosingActionContent{Heading: "Configured close", Description: "Configured close", Actions: []Action{{Kind: ActionOpen, Label: "Private", AccessibleLabel: "Private", AppKey: "bas"}}}})
	if err := Validate(document); err != nil {
		t.Fatal(err)
	}
	result, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range result.Page.Blocks {
		if block.ID == "private-action" {
			closing, ok := block.Content.(ClosingActionContent)
			if !ok || len(closing.Actions) != 0 {
				t.Fatalf("private action survived public projection: %#v", closing.Actions)
			}
		}
	}

	document.Pages[2].Blocks[0].Content = ProductHeroContent{AppKey: "bas", Title: "Private", Description: "Private", VisualRef: "hero-art", AccessibilityLabel: "Private", Actions: []Action{{Kind: ActionOpen, Label: "Open", AccessibleLabel: "Open", Target: "/apps/browser-automation-studio"}}}
	if err := Validate(document); !hasIssueCode(err, "wrong_product_hero_app_ref") {
		t.Fatalf("wrong app hero validation error = %v", err)
	}
}

func TestLP_PRES_004_009_014_StrictTypedDecodeAndRichFixtureShape(t *testing.T) {
	rich := richDocument()
	if err := Validate(rich); err != nil {
		t.Fatalf("rich document should validate: %v", err)
	}

	data, err := json.Marshal(rich)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeDocument(data)
	if err != nil {
		t.Fatalf("DecodeDocument() error = %v", err)
	}
	if _, ok := decoded.Pages[2].Blocks[0].Content.(ProductHeroContent); !ok {
		t.Fatalf("decoded content type = %T, want ProductHeroContent", decoded.Pages[2].Blocks[0].Content)
	}

	var unknown struct{}
	_ = unknown
	bad := strings.Replace(string(data), `"fixture_ref":"workspace-demo"`, `"unknown_prop":"no"`, 1)
	if _, err := DecodeDocument([]byte(bad)); err == nil {
		t.Fatal("unknown typed content property was accepted")
	}
	badTop := strings.TrimSuffix(string(data), "}") + `,"unexpected":true}`
	if _, err := DecodeDocument([]byte(badTop)); err == nil {
		t.Fatal("unknown document property was accepted")
	}
}

func TestLP_PRES_003_004_008_009_012_014_015_ValidationRejectsUnsafeOrUnqualified(t *testing.T) {
	document := richDocument()
	cases := []struct {
		name   string
		mutate func(*Document)
		code   string
	}{
		{"duplicate app key", func(d *Document) { d.Apps = append(d.Apps, d.Apps[0]) }, "duplicate_app_key"},
		{"unknown app order ref", func(d *Document) { d.Bundle.AppOrder[0] = "missing" }, "unknown_app_ref"},
		{"unqualified available capability", func(d *Document) { d.Apps[0].Capabilities[0].OwnerQualification = nil }, "unverified_owner_qualification"},
		{"geometry without measurement", func(d *Document) { d.Assets[0].OverlayRegions[0].Measurement = LegibilityMeasurement{} }, "invalid_measurement"},
		{"negative cap", func(d *Document) { d.Bundle.MaxAppSlides = -1 }, "invalid_cap"},
		{"unsafe action", func(d *Document) {
			page := &d.Pages[2]
			page.Blocks[0].Content = ProductHeroContent{Title: "x", Description: "x", FixtureRef: "workspace-demo", AccessibilityLabel: "x", Actions: []Action{{Kind: ActionOpen, Label: "x", AccessibleLabel: "x", Target: "javascript:alert(1)"}}}
		}, "unsafe_target"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			copy := document
			copy.Apps = append([]App(nil), document.Apps...)
			copy.Bundle.AppOrder = append([]string(nil), document.Bundle.AppOrder...)
			copy.Assets = append([]Asset(nil), document.Assets...)
			tt.mutate(&copy)
			err := Validate(copy)
			if !hasIssueCode(err, tt.code) {
				t.Fatalf("Validate() error = %v, want code %q", err, tt.code)
			}
		})
	}
}

func TestLP_PRES_004_LocaleSelectionForSamePageID(t *testing.T) {
	document := testDocument(testApp("web-console", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true))
	frPage := document.Pages[2]
	frPage.Locale, frPage.Title, frPage.Description = "fr", "Accueil français", "Description française"
	document.Pages = append(document.Pages, frPage)
	fr, err := Resolve(document, ResolveRequest{Route: "/", Locale: "fr-FR"})
	if err != nil {
		t.Fatal(err)
	}
	if fr.Page.Locale != "en" || fr.Diagnostics.FallbackReason != "locale_unavailable" {
		t.Fatalf("unsupported locale selection = (%s, %q)", fr.Page.Locale, fr.Diagnostics.FallbackReason)
	}
	fr, err = Resolve(document, ResolveRequest{Route: "/", Locale: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if fr.Page.Locale != "fr" || fr.Page.Title != "Accueil français" {
		t.Fatalf("exact locale selection = %#v", fr.Page)
	}
}

func hasIssueCode(err error, code string) bool {
	var validation *ValidationError
	if !errors.As(err, &validation) {
		return false
	}
	for _, issue := range validation.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func richDocument() Document {
	app := testApp("web-console", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true)
	app.Capabilities = []Capability{{ID: "local-speech", Label: "Local speech", Benefits: []string{"Speak and listen"}, Status: CapabilityAvailable, EvidenceRefs: []string{"receipt-speech"}, OwnerQualification: &OwnerQualification{Owner: "audio-owner", EvidenceRef: "receipt-speech", ReleaseRef: "release-1", Qualified: true}}}
	asset := Asset{ID: "hero-art", ReleaseRef: "release-1", ContentHash: strings.Repeat("a", 64), Width: 1440, Height: 720, MIME: "image/png", Surface: "web-hero", CropPolicy: "center", Provenance: AssetProvenance{Provider: "backdrop", JobRef: "job-1", CandidateRef: "candidate-1"}, OverlayRegions: []OverlayRegion{{Name: "copy", Width: 0.4, Height: 0.4, Measurement: LegibilityMeasurement{ContrastRatio: 7, MinimumContrastRatio: 4.5, Threshold: 4.5, Verdict: LegibilityPass, MeasurementRef: "measure-1"}}}, PublicURL: "/assets/hero-art.png", PrivateEvidenceRefs: []string{"private-capture-1"}}
	fixture := Fixture{ID: "workspace-demo", Kind: FixtureWorkspace, Workspace: &WorkspaceFixture{Title: "Aquila", Group: "Website refresh", GroupLabel: "Your work", Groups: []string{"Website refresh"}, SessionsLabel: "Sessions", Sessions: []string{"Build"}, Role: "Builder", Model: "Model", Reviewer: "Reviewer", ReviewerModel: "Model", Branch: "main", Prompt: "Prompt", Answer: "Answer", Files: []string{"Hero.tsx"}, FileLabel: "Changes", Diff: []string{"Clearer"}, Command: "$ test", Checks: []string{"Pass"}, Ready: "Ready", Composer: "Compose", ReturnLabel: "Summary", ReturnTitle: "Useful", ReviewMessage: "Reviewed", MessageLabel: "You", ReplyLabel: "Agent", Status: "Connected", Today: "Today", Keyboard: []string{"enter"}}}
	hero := Block{ID: "hero", Kind: BlockProductHero, Version: SchemaVersion, Variant: "centered", Content: ProductHeroContent{AppKey: "web-console", Title: "All agents", Description: "One view", VisualRef: "hero-art", FixtureRef: "workspace-demo", AccessibilityLabel: "Illustrative workspace", Actions: []Action{{Kind: ActionOpen, Label: "Open", AccessibleLabel: "Open", Target: "/apps/aquila"}}}}
	voice := Block{ID: "voice", Kind: BlockVoiceStory, Version: SchemaVersion, Variant: "waveform", Content: VoiceStoryContent{Heading: "Speak", Body: "Voice", Features: []VoiceFeature{{Title: "Local", Description: "Speech", CapabilityID: "local-speech"}}, Note: "Configured provider", InputLabel: "Input", Transcript: "Hello", SummaryLabel: "Summary", SummaryTitle: "Short", SummaryItems: []string{"Useful"}, OutputLabel: "Output", DemoNote: "Illustrative", ProviderQualification: "Configured", Waveform: []float64{0.2, 0.8}, CapabilityIDs: []string{"local-speech"}}}
	closing := Block{ID: "closing", Kind: BlockClosingAction, Version: SchemaVersion, Variant: "plain", Content: ClosingActionContent{Heading: "Continue", Description: "Continue", Actions: []Action{{Kind: ActionAnchor, Label: "Continue", AccessibleLabel: "Continue", Target: "#details"}}}}
	page := Page{ID: "aquila-page", Locale: "en", Title: "Aquila", Description: "Workspace", Theme: Theme{Variant: "signal", Primary: "#263E38", Background: "#EAEADF", Accent: "#DF7958"}, Navigation: Navigation{Label: "Main", Items: []NavigationItem{{Label: "Details", AccessibleLabel: "Details", Target: "#details"}}}, Blocks: []Block{hero, voice, closing}, Footer: Footer{Label: "Footer", Links: []NavigationItem{{Label: "Home", AccessibleLabel: "Home", Target: "/"}}}}
	d := testDocument(app)
	d.Assets = []Asset{asset}
	d.Fixtures = []Fixture{fixture}
	page.Display = testDisplay()
	page.Display.FixtureDisplay["workspace-demo"] = FixtureDisplay{Mark: "letter-a", Avatar: "JD", Time: "Now", TabsLabel: "Sessions", TerminalLabel: "Terminal", MessagesLabel: "Messages"}
	d.Pages[2] = page
	return d
}
