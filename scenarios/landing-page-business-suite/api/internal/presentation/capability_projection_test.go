package presentation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func capabilityDocument() Document {
	a := testApp("web-console", "aquila", "aquila-page", VisibilityPublic, PublicationPublished, true)
	b := testApp("backdrop-studio", "backdrop-studio", "backdrop-page", VisibilityPublic, PublicationPublished, true)
	private := testApp("private-app", "private-app", "private-page", VisibilityPrivate, PublicationDraft, false)
	b.Capabilities = []Capability{{ID: "public-cap", Label: "Visible capability", Benefits: []string{"Configured public benefit"}, Status: CapabilityPreview, StatusLabel: "Preview", EvidenceRefs: []string{"fixture:public"}}}
	private.Capabilities = []Capability{{ID: "secret-cap", Label: "Secret capability", Benefits: []string{"Secret benefit"}, Status: CapabilityPreview, StatusLabel: "Preview", EvidenceRefs: []string{"fixture:private"}}}
	return testDocument(a, b, private)
}

func capabilityStrip(id, ref string) Block {
	return Block{ID: id, Kind: BlockCapabilityStrip, Version: 1, Variant: "inline", Content: CapabilityStripContent{Heading: "Configured strip", Items: []CapabilityItem{{CapabilityID: ref, Label: "Configured claim", Description: "Configured description"}}}}
}

func hasValidationCode(err error, code string) bool {
	var invalid *ValidationError
	if errors.As(err, &invalid) {
		for _, issue := range invalid.Issues {
			if issue.Code == code {
				return true
			}
		}
	}
	return false
}

func TestAppPageOwnershipIsUniqueAndCapabilityReferencesStayWithinOwner(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-014]
	document := capabilityDocument()
	document.Apps[1].PageID = document.Apps[0].PageID
	if err := Validate(document); !hasValidationCode(err, "ambiguous_page_owner") {
		t.Fatalf("shared page owner accepted: %v", err)
	}
	document = capabilityDocument()
	document.Pages[2].Blocks = append(document.Pages[2].Blocks, capabilityStrip("cross-app", "public-cap"))
	if err := Validate(document); !hasValidationCode(err, "unknown_capability_ref") {
		t.Fatalf("cross-app detail claim accepted: %v", err)
	}
}

func TestBundleProjectsOnlyPublicCapabilityNarrativesIndependentOfSpotlightCap(t *testing.T) { // [REQ:LP-PRES-005] [REQ:LP-PRES-009] [REQ:LP-PRES-014]
	document := capabilityDocument()
	document.Bundle.MaxAppSlides = 0
	page := &document.Pages[0]
	page.Blocks = append(page.Blocks, capabilityStrip("public-story", "public-cap"), capabilityStrip("secret-story", "secret-cap"))
	page.Display.Blocks["secret-story"] = BlockDisplay{Note: "Secret decoration"}
	page.Display.Shell.HeaderAction = &Action{Kind: ActionAnchor, Label: "Secret action", AccessibleLabel: "Secret action", Target: "#secret-story"}
	page.Navigation.Items = append(page.Navigation.Items, NavigationItem{Label: "Secret navigation", AccessibleLabel: "Secret navigation", Target: "#secret-story"})
	result, err := Resolve(document, ResolveRequest{Route: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.SelectedAppKeys) != 0 || len(result.Capabilities) != 1 || result.Capabilities[0].ID != "public-cap" {
		t.Fatalf("visible claims tied incorrectly to spotlight cap: %+v", result.Capabilities)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(data)), "secret") {
		t.Fatal("private capability narrative or navigation leaked")
	}
	if document.Pages[0].Display.Shell.HeaderAction == nil || len(document.Pages[0].Navigation.Items) != 2 {
		t.Fatal("projection mutated configured document")
	}
}

func TestAssetIdentityMatchesPublicCacheAndWireContract(t *testing.T) { // [REQ:LP-PRES-010]
	for _, hash := range []string{strings.Repeat("a", 32), strings.Repeat("a", 128), "sha256:" + strings.Repeat("a", 64), strings.Repeat("A", 64)} {
		document := capabilityDocument()
		document.Assets[0].ContentHash = hash
		if err := Validate(document); !hasValidationCode(err, "invalid_content_hash") {
			t.Fatalf("noncanonical SHA-256 accepted: %q (%v)", hash, err)
		}
	}
}

func TestPublicViewsUseCanonicalMembershipAndLocales(t *testing.T) { // [REQ:LP-PRES-009] [REQ:LP-PRES-011]
	document := capabilityDocument()
	document.Bundle.AppOrder = []string{"web-console", "private-app"} // backdrop is published but not a bundle member.
	views, err := PublicViews(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 4 {
		t.Fatalf("got %d views, want root and Aquila for two requested locales", len(views))
	}
	for _, view := range views {
		if view.Diagnostics.ResolvedRoute != "/" && view.Diagnostics.ResolvedRoute != "/apps/aquila" {
			t.Fatalf("non-public route escaped: %s", view.Diagnostics.ResolvedRoute)
		}
		if view.Diagnostics.Preview || view.Diagnostics.AppKey != "web-console" {
			t.Fatalf("wrong authority in canonical view: %+v", view.Diagnostics)
		}
	}
}
