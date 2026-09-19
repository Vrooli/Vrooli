package authoring

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"

	"experience-manager/internal/spec"

	db "github.com/vrooli/api-core/databasetest"
)

func TestAuthoringRoundTripAppliesParserCleanPage(t *testing.T) { // [REQ:EXPERIEN-P0-004]
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, handle, apidb.SchemaProviderFunc(Schema)))

	scenarioDir := writeScenario(t)
	service := Service{
		Repo: NewSQLiteRepository(handle),
		Now:  func() time.Time { return time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC) },
	}
	session, err := service.StartSession(ctx, "", scenarioDir)
	require.NoError(t, err)

	_, draft, err := service.SubmitPage(ctx, session.ID, validPageForm("authored-page"))
	require.NoError(t, err)
	require.Equal(t, "pages/authored-page.json", draft.Path)

	preview, err := service.Preview(ctx, session.ID)
	require.NoError(t, err)
	require.Equal(t, "demo", preview.Report.Scenario)
	require.Empty(t, errorFindings(preview.Report.Findings))
	require.NoFileExists(t, filepath.Join(scenarioDir, "experience", "pages", "authored-page.json"))

	applied, err := service.Apply(ctx, session.ID)
	require.NoError(t, err)
	require.Empty(t, errorFindings(applied.Report.Findings))
	require.FileExists(t, filepath.Join(scenarioDir, "experience", "pages", "authored-page.json"))

	report, err := spec.ParseScenario(scenarioDir)
	require.NoError(t, err)
	require.Empty(t, errorFindings(report.Findings))
}

func TestSuggestBindingsIncludesExistingSpecBindings(t *testing.T) { // [REQ:EXPERIEN-P0-004]
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, handle, apidb.SchemaProviderFunc(Schema)))

	scenarioDir := writeScenario(t)
	service := Service{Repo: NewSQLiteRepository(handle)}
	suggestions, err := service.SuggestBindings(ctx, "", scenarioDir, "existing", 10)
	require.NoError(t, err)
	require.Len(t, suggestions, 1)
	require.Equal(t, "main", suggestions[0].ElementID)
	require.Equal(t, "existing-main", suggestions[0].TestID)
	require.Equal(t, "spec", suggestions[0].Source)
}

func TestCompareVariantsRendersSideBySide(t *testing.T) { // [REQ:EXPERIEN-P1-001]
	ctx := context.Background()
	scenarioDir := writeScenario(t)
	service := Service{}
	result, err := service.CompareVariants(ctx, "", scenarioDir, "existing", "wireframe", []*contractv1.SpecVariant{
		{Id: "compact", Title: "Compact", Page: validPageForm("existing")},
		{Id: "evidence", Title: "Evidence Forward", Page: variantPageForm("existing", "Evidence Forward")},
	})
	require.NoError(t, err)
	require.Equal(t, "demo", result.Scenario)
	require.Len(t, result.Variants, 2)
	require.Contains(t, result.HTML, "data-variant=\"compact\"")
	require.FileExists(t, result.ArtifactPath)
}

func TestPromoteVariantAppliesParserCleanPage(t *testing.T) { // [REQ:EXPERIEN-P1-001]
	ctx := context.Background()
	scenarioDir := writeScenario(t)
	service := Service{}
	result, err := service.PromoteVariant(ctx, "", scenarioDir, "existing", &contractv1.SpecVariant{
		Id:    "evidence",
		Title: "Evidence Forward",
		Page:  variantPageForm("existing", "Evidence Forward"),
	})
	require.NoError(t, err)
	require.Equal(t, "PASSED", statusFromFindings(result.Report.Findings))
	require.NotEmpty(t, result.Variant.HTML)
	require.Contains(t, readFile(t, filepath.Join(scenarioDir, "experience", "pages", "existing.json")), "Evidence Forward")
}

func TestPromoteVariantRejectsDifferentPageID(t *testing.T) {
	ctx := context.Background()
	scenarioDir := writeScenario(t)
	service := Service{}
	_, err := service.PromoteVariant(ctx, "", scenarioDir, "existing", &contractv1.SpecVariant{
		Id:   "wrong",
		Page: validPageForm("other-page"),
	})
	require.ErrorContains(t, err, "must match target page")
	require.NotContains(t, readFile(t, filepath.Join(scenarioDir, "experience", "pages", "existing.json")), "Authored Page")
}

func validPageForm(id string) *contractv1.PageForm {
	return &contractv1.PageForm{
		Id:         id,
		Title:      "Authored Page",
		Purpose:    "Authored page exists to prove the studio round trip is parser clean.",
		Routes:     []string{"/authored"},
		Status:     "draft",
		Priorities: []*contractv1.PriorityForm{{Statement: "The main region is the clear starting point."}},
		States:     []*contractv1.StateForm{{Id: "default", Description: "Default authored state."}},
		Elements:   []*contractv1.ElementForm{{Id: "main", Role: "region", Name: "Main authored region"}},
		Claims: []*contractv1.ClaimForm{{
			Id:        "main-present",
			Type:      "element-present",
			Statement: "The main authored region is present.",
			Tier:      "machine",
			Elements:  []string{"main"},
			States:    []string{"default"},
		}},
		Bindings:      []*contractv1.BindingForm{{ElementId: "main", Testid: "authored-main"}},
		SketchRegions: []*contractv1.SketchRegionForm{{Id: "main-region", Elements: []string{"main"}}},
	}
}

func variantPageForm(id, title string) *contractv1.PageForm {
	form := validPageForm(id)
	form.Title = title
	form.Purpose = "Promoted page variant exists to prove workshop promotion remains parser clean."
	form.Priorities[0].Statement = "Evidence is visible before secondary controls."
	form.Elements[0].Name = title + " region"
	return form
}

func writeScenario(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "demo")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "experience", "pages"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "experience", "index.json"), []byte(`{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "description": "Demo experience contract for authoring tests.",
  "pages": [{"id": "existing", "path": "pages/existing.json", "title": "Existing", "status": "draft"}],
  "journeys": []
}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "experience", "pages", "existing.json"), []byte(`{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {
    "id": "existing",
    "title": "Existing",
    "routes": ["/existing"],
    "purpose": "Existing page keeps the base experience contract parser clean."
  },
  "priorities": [{"statement": "The existing page has a main region."}],
  "states": [{"id": "default", "description": "Default state."}],
  "elements": [{"id": "main", "role": "region", "name": "Existing main"}],
  "claims": [{
    "id": "main-present",
    "type": "element-present",
    "statement": "The existing main region is present.",
    "tier": "machine",
    "elements": ["main"],
    "states": ["default"]
  }],
  "bindings": {"elements": {"main": {"testid": "existing-main"}}},
  "sketch": {"regions": [{"id": "main-region", "elements": ["main"]}]}
}
`), 0o644))
	return dir
}

func errorFindings(findings []spec.Finding) []spec.Finding {
	var out []spec.Finding
	for _, finding := range findings {
		if finding.Severity == spec.SeverityError {
			out = append(out, finding)
		}
	}
	return out
}

func statusFromFindings(findings []spec.Finding) string {
	if len(errorFindings(findings)) > 0 {
		return "FAILED"
	}
	return "PASSED"
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
