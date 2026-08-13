package variants

import (
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"
	"landing-page-business-suite/cli/internal/support"
)

func TestLegacyUpdateRequestPreservesLegacyAxesAndSEO(t *testing.T) {
	request, err := legacyUpdateRequest("control", []byte(`{
		"name":"Updated control",
		"axes":{"persona":"founder","conversionStyle":"demo_led"},
		"seo_config":{"title":"Control SEO","canonical_path":"/control"}
	}`))
	if err != nil {
		t.Fatalf("legacyUpdateRequest() error = %v", err)
	}
	if request.GetSlug() != "control" || request.GetName() != "Updated control" {
		t.Fatalf("request identity = %#v", request)
	}
	if got := request.GetAxes().GetValues(); got["persona"] != "founder" || got["conversionStyle"] != "demo_led" {
		t.Fatalf("request axes = %#v", got)
	}
	if got := request.GetSeoConfig(); got.GetTitle() != "Control SEO" || got.GetCanonicalPath() != "/control" {
		t.Fatalf("request SEO = %#v", got)
	}
}

func TestLegacyUpdateRequestRejectsMissingSlugAndInvalidPayload(t *testing.T) {
	if _, err := legacyUpdateRequest("", []byte(`{}`)); err == nil {
		t.Fatal("expected missing slug error")
	}
	if _, err := legacyUpdateRequest("control", []byte(`{`)); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestLegacyImportRequestPreservesSnapshotFileShape(t *testing.T) {
	request, err := legacyImportRequest("import-me", []byte(`{
		"variant": {
			"slug": "import-me",
			"name": "Imported",
			"description": "Portable configuration",
			"weight": 25,
			"status": "active",
			"axes": {"persona": "operator"},
			"seo_config": {"title": "Imported SEO"}
		},
		"sections": [{
			"section_type": "hero",
			"content": {"heading": "Welcome"},
			"order": 1,
			"enabled": true
		}]
	}`))
	if err != nil {
		t.Fatalf("legacyImportRequest() error = %v", err)
	}
	snapshot := request.GetSnapshot()
	if request.GetSlug() != "import-me" || snapshot.GetName() != "Imported" || snapshot.GetAxes()["persona"] != "operator" {
		t.Fatalf("snapshot identity = %#v", snapshot)
	}
	if snapshot.GetSeoConfig().GetTitle() != "Imported SEO" || len(snapshot.GetSections()) != 1 {
		t.Fatalf("snapshot metadata = %#v", snapshot)
	}
	section := snapshot.GetSections()[0]
	if section.GetSectionType() != "hero" || section.GetContent().GetFields()["heading"].GetStringValue() != "Welcome" {
		t.Fatalf("snapshot section = %#v", section)
	}
}

func TestLegacyImportRequestRejectsMissingAndMismatchedPayloadSlug(t *testing.T) {
	if _, err := legacyImportRequest("", []byte(`{}`)); err == nil {
		t.Fatal("expected missing positional slug error")
	}
	if _, err := legacyImportRequest("import-me", []byte(`{"variant":{"slug":"other"}}`)); err == nil {
		t.Fatal("expected mismatched payload slug error")
	}
	if _, err := legacyImportRequest("import-me", []byte(`{"sections":[]}`)); err == nil {
		t.Fatal("expected missing variant object error")
	}
}

func TestRegisterExposesValidCommandGroup(t *testing.T) {
	group := Register(support.Dependencies{})
	if err := testutil.ValidateCommandGroup(group); err != nil {
		t.Fatalf("ValidateCommandGroup() error = %v", err)
	}
}
