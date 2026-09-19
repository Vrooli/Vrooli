package main

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationseed"
)

func TestSEOServiceSitemapUsesPublishedPublicPresentationRoutes(t *testing.T) {
	store := newSEOTestConfigStore(t)
	document := seoTestDocument(t)
	document.Apps[0].Publication = presentation.PublicationPublished
	publishSEOTestPresentation(t, store, "control", document)

	branding := store.GetBranding()
	canonical := "https://trusted.example"
	branding.CanonicalBaseURL = &canonical
	if err := store.SaveBranding(branding); err != nil {
		t.Fatal(err)
	}

	sitemap, err := NewSEOService(store).SitemapXML("https://attacker.example")
	if err != nil {
		t.Fatal(err)
	}
	var documentXML struct {
		URLs []struct {
			Location string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal([]byte(sitemap), &documentXML); err != nil {
		t.Fatalf("sitemap XML decode failed: %v\n%s", err, sitemap)
	}
	locations := make(map[string]bool, len(documentXML.URLs))
	for _, entry := range documentXML.URLs {
		locations[entry.Location] = true
	}
	if !locations["https://trusted.example/"] || !locations["https://trusted.example/apps/aquila"] {
		t.Fatalf("published public routes missing: %#v", locations)
	}
	for _, privatePath := range []string{"/apps/browser-automation-studio", "/apps/backdrop-studio"} {
		if locations["https://trusted.example"+privatePath] {
			t.Fatalf("private seed route leaked: %#v", locations)
		}
	}
	if locations["https://attacker.example/"] {
		t.Fatalf("request host leaked into sitemap: %#v", locations)
	}
}

func TestSEOServiceSitemapIntersectsSelectablePublishedVariantMembership(t *testing.T) {
	store := newSEOTestConfigStore(t)
	base := seoTestDocument(t)
	base.Apps[0].Publication = presentation.PublicationPublished
	publishSEOTestPresentation(t, store, "control", base)
	if err := store.SaveVariant("alternate", &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: "alternate", Name: "Alternate", Weight: 100, Status: "active",
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}}); err != nil {
		t.Fatal(err)
	}
	alternate := seoTestDocument(t)
	alternate.Apps[0].Publication = presentation.PublicationPublished
	alternate.Apps[1].Enabled = true
	alternate.Apps[1].Visibility = presentation.VisibilityPublic
	alternate.Apps[1].Publication = presentation.PublicationPublished
	publishSEOTestPresentation(t, store, "alternate", alternate)

	branding := store.GetBranding()
	canonical := "https://trusted.example"
	branding.CanonicalBaseURL = &canonical
	if err := store.SaveBranding(branding); err != nil {
		t.Fatal(err)
	}
	sitemap, err := NewSEOService(store).SitemapXML("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sitemap, "/apps/aquila") || strings.Contains(sitemap, "/apps/browser-automation-studio") {
		t.Fatalf("sitemap did not conservatively intersect memberships: %s", sitemap)
	}
}

func TestSEOServiceSitemapIsUnavailableWhenSelectablePresentationIsMissing(t *testing.T) {
	store := newSEOTestConfigStore(t)
	branding := store.GetBranding()
	canonical := "https://trusted.example"
	branding.CanonicalBaseURL = &canonical
	if err := store.SaveBranding(branding); err != nil {
		t.Fatal(err)
	}
	_, err := NewSEOService(store).SitemapXML("")
	if !errors.Is(err, content.ErrPublishedRoutesMissing) {
		t.Fatalf("SitemapXML error = %v, want missing published routes", err)
	}
}

func TestSEOServiceSitemapFailsClosedWhenNoVariantIsSelectable(t *testing.T) {
	store := experimentation.NewConfigStore(t.TempDir(), filepath.Join(t.TempDir(), "branding.json"), nil)
	if err := store.SaveVariant("archived", &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: "archived", Name: "Archived", Weight: 100, Status: "archived",
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}}); err != nil {
		t.Fatal(err)
	}
	branding := store.GetBranding()
	canonical := "https://trusted.example"
	branding.CanonicalBaseURL = &canonical
	if err := store.SaveBranding(branding); err != nil {
		t.Fatal(err)
	}
	_, err := NewSEOService(store).SitemapXML("")
	if !errors.Is(err, content.ErrPublishedRoutesMissing) {
		t.Fatalf("SitemapXML error = %v, want no selectable published routes", err)
	}
}

func TestPublishedRouteAdapterHonorsCanceledLeaseContext(t *testing.T) {
	store := newPublishedSEOConfigStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (seoConfigStoreAdapter{store: store}).PublishedPublicRoutes(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PublishedPublicRoutes() error = %v, want canceled lease context", err)
	}
}

func newSEOTestConfigStore(t *testing.T) *experimentation.ConfigStore {
	t.Helper()
	store := experimentation.NewConfigStore(t.TempDir(), filepath.Join(t.TempDir(), "branding.json"), nil)
	if err := store.SaveVariant("control", &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: "control", Name: "Control", Weight: 100, Status: "active",
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}}); err != nil {
		t.Fatal(err)
	}
	store.SetPresentationStorage(filerouting.New(storage.Paths{ConfigDir: t.TempDir()}), func(context.Context, presentation.Document, *presentation.Document) error { return nil })
	return store
}

func newPublishedSEOConfigStore(t *testing.T) *experimentation.ConfigStore {
	t.Helper()
	store := newSEOTestConfigStore(t)
	document := seoTestDocument(t)
	document.Apps[0].Publication = presentation.PublicationPublished
	publishSEOTestPresentation(t, store, "control", document)
	return store
}

func publishSEOTestPresentation(t *testing.T, store *experimentation.ConfigStore, variant string, document presentation.Document) {
	t.Helper()
	state, err := store.SavePresentationDraft(context.Background(), variant, document, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishPresentation(context.Background(), variant, state.DraftRevision, state.Generation); err != nil {
		t.Fatal(err)
	}
}

func seoTestDocument(t *testing.T) presentation.Document {
	t.Helper()
	paths := []string{
		filepath.Join("..", ".vrooli", "presentation-seeds", "recommended-signal-studio.json"),
		filepath.Join("scenarios", "landing-page-business-suite", ".vrooli", "presentation-seeds", "recommended-signal-studio.json"),
		filepath.Join("internal", "presentationseed", "recommended-signal-studio.json"),
	}
	var data []byte
	var err error
	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("read presentation seed: %v", err)
	}
	document, err := presentationseed.DecodeAndValidate(data)
	if err != nil {
		t.Fatal(err)
	}
	return document
}
