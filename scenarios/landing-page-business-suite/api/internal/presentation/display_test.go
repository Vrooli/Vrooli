package presentation

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestLP_PRES_004_007_DisplayIsImmutableConfiguredContent(t *testing.T) {
	document := richDocument()
	document.Pages[2].Display.Blocks["hero"] = BlockDisplay{Note: "Your configured welcome", HeadingBreaks: []int{10}}
	root, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := Resolve(document, ResolveRequest{Route: "/apps/aquila"})
	if err != nil {
		t.Fatal(err)
	}
	if root.Diagnostics.BlockDigest != detail.Diagnostics.BlockDigest || !reflect.DeepEqual(root.Page.Display, detail.Page.Display) {
		t.Fatal("root/detail display parity was lost")
	}
	root.Page.Display.Blocks["hero"] = BlockDisplay{Note: "Changed by a response consumer"}
	root.Page.Display.FixtureDisplay["workspace-demo"] = FixtureDisplay{Mark: "play"}
	again, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if again.Page.Display.Blocks["hero"].Note != "Your configured welcome" || again.Page.Display.FixtureDisplay["workspace-demo"].Mark != "letter-a" {
		t.Fatal("resolved display shares mutable backing state")
	}
	document.Pages[2].Display.Shell.BrandName = "Different configured brand"
	changed, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Diagnostics.BlockDigest == detail.Diagnostics.BlockDigest {
		t.Fatal("content digest omitted configured display copy")
	}
}

func TestLP_PRES_005_006_HeroCapacityIsIndependentOfSpotlightCap(t *testing.T) {
	appA := testApp("web-console", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true)
	appB := testApp("backdrop-studio", "backdrop-studio", "backdrop-page", VisibilityPublic, PublicationPublished, true)
	document := testDocument(appA, appB)
	hero := Block{ID: "bundle-hero", Kind: BlockBundleHero, Version: 1, Variant: "editorial", Content: BundleHeroContent{Title: "A configured suite", Description: "Two useful products", AccessibilityLabel: "Explore the suite", HeroItems: []HeroItem{
		{AppKey: appB.Key, VisualRef: "hero-art", ExhibitKind: "artwork", DetailLabel: "Design"},
		{AppKey: appA.Key, VisualRef: "hero-art", ExhibitKind: "screenshot", DetailLabel: "Work"},
	}, Actions: []Action{{Kind: ActionAnchor, Label: "Explore", AccessibleLabel: "Explore apps", Target: "#bundle-spotlights"}}}}
	document.Pages[0].Blocks = append([]Block{hero}, document.Pages[0].Blocks...)
	for _, cap := range []int{0, 1, 2, 10} {
		document.Bundle.MaxAppSlides = cap
		result, err := Resolve(document, ResolveRequest{Route: "/"})
		if err != nil {
			t.Fatal(err)
		}
		items := result.Page.Blocks[0].Content.(BundleHeroContent).HeroItems
		if len(items) != 2 || items[0].AppKey != appB.Key || items[1].AppKey != appA.Key {
			t.Fatalf("k=%d altered configured hero grouping/order: %#v", cap, items)
		}
		slides := result.Page.Blocks[1].Content.(AppSpotlightsContent).AppKeys
		want := cap
		if want > 2 {
			want = 2
		}
		if len(slides) != want || len(result.SelectedAppKeys) != want {
			t.Fatalf("k=%d produced %d app slides", cap, len(slides))
		}
		if len(result.Spotlights) != 2 || len(result.Page.Display.Apps) != 2 {
			t.Fatal("hero lost its typed profile or configured display resource")
		}
	}
}

func TestLP_PRES_007_012_DisplayProjectionDoesNotExposePrivateProfiles(t *testing.T) {
	document := richDocument()
	private := testApp("private-app", "private-app", "private-page", VisibilityPrivate, PublicationDraft, false)
	document.Apps = append(document.Apps, private)
	document.Bundle.AppOrder = append(document.Bundle.AppOrder, private.Key)
	document.Pages = append(document.Pages, testShellPage(private.PageID, "en", "Preserved private page", testAppBlocks(private)...))
	fixture := document.Fixtures[0]
	fixture.ID = "private-exhibit"
	document.Fixtures = append(document.Fixtures, fixture)
	display := &document.Pages[2].Display
	display.Apps[private.Key] = AppDisplay{FixtureRef: fixture.ID, Mark: "play", Tone: "sage", DetailLabel: "SECRET PRIVATE COPY"}
	display.FixtureDisplay[fixture.ID] = FixtureDisplay{Mark: "play", Avatar: "Hidden", Time: "Now", TabsLabel: "Secret tabs", TerminalLabel: "Secret terminal", MessagesLabel: "Secret messages"}
	display.Shell.HeaderAction = &Action{Kind: ActionDownload, Label: "PRIVATE DOWNLOAD", AccessibleLabel: "Private download", AppKey: private.Key}
	public, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{private.Key, fixture.ID, "SECRET PRIVATE COPY", "PRIVATE DOWNLOAD", "Secret tabs"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("public display leaked %q", secret)
		}
	}
	preview, err := Resolve(document, ResolveRequest{Route: "/", PreviewRevision: "draft-1", PreviewAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Mode != ModeSingleApp || preview.Diagnostics.BlockDigest != public.Diagnostics.BlockDigest {
		t.Fatal("draft root preview changed public membership/content")
	}
	privatePreview, err := Resolve(document, ResolveRequest{Route: "/apps/private-app", PreviewRevision: "draft-1", PreviewAuthorized: true})
	if err != nil || privatePreview.AppKey != private.Key || !privatePreview.Diagnostics.NoStore {
		t.Fatalf("explicit private app preview failed: %v", err)
	}
}

func TestLP_PRES_004_014_DisplayValidationRejectsUnrenderableConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Document)
		code   string
	}{
		{"missing shell copy", func(d *Document) { d.Pages[2].Display.Shell.SkipLabel = "" }, "must not be empty"},
		{"unknown mark", func(d *Document) { d.Pages[2].Display.Shell.BrandMark = "custom-jsx" }, "registered product mark"},
		{"unsafe header action", func(d *Document) {
			d.Pages[2].Display.Shell.HeaderAction = &Action{Kind: ActionOpen, Label: "Open", AccessibleLabel: "Open", Target: "javascript:alert(1)"}
		}, "safe"},
		{"unknown block", func(d *Document) { d.Pages[2].Display.Blocks["missing"] = BlockDisplay{Note: "Orphan"} }, "reference a block"},
		{"missing fixture display", func(d *Document) { delete(d.Pages[2].Display.FixtureDisplay, "workspace-demo") }, "requires configured display"},
		{"unsupported renderer variant", func(d *Document) { d.Pages[2].Blocks[0].Variant = "split" }, "not registered"},
		{"unsupported theme", func(d *Document) { d.Pages[2].Theme.Variant = "dark" }, "signal or studio"},
		{"invalid heading offsets", func(d *Document) { d.Pages[2].Display.Blocks["hero"] = BlockDisplay{HeadingBreaks: []int{10, 2}} }, "strictly increasing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := richDocument()
			tt.change(&document)
			err := Validate(document)
			if err == nil || !strings.Contains(err.Error(), tt.code) {
				t.Fatalf("invalid display accepted or wrong error: %v", err)
			}
		})
	}
}
