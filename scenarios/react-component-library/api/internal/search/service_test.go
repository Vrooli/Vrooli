package search

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

// newTestService opens a fresh file-backed SQLite database (unique DSN per test,
// MaxOpenConns(1) via the shared helper) and applies the production schema. The
// scenario root intentionally has no catalog so ReplaceDocuments tests are
// isolated from the repository.
func newTestService(t *testing.T) *Service {
	t.Helper()
	return newServiceAt(t, t.TempDir(), nil)
}

func newServiceAt(t *testing.T, scenarioRoot string, availability AvailabilityFunc) *Service {
	t.Helper()
	database := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	return NewService(database, scenarioRoot, availability)
}

func testDoc(kind searchv1.SearchKind, subjectID, title, body string) Document {
	return Document{
		SubjectID:    subjectID,
		Kind:         kind,
		Title:        title,
		Body:         body,
		Href:         "/test/" + subjectID,
		Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
	}
}

func seedDocuments(t *testing.T, svc *Service, docs ...Document) {
	t.Helper()
	require.NoError(t, svc.ReplaceDocuments(context.Background(), docs))
}

func search(t *testing.T, svc *Service, query string, kinds ...searchv1.SearchKind) *searchv1.SearchResponse {
	t.Helper()
	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: query, Kinds: kinds})
	require.NoError(t, err)
	require.NotNil(t, response)
	return response
}

// TestSearchLandingPage is the program's headline ranking fixture: the page type
// that realizes a landing page must beat generic assets, and an unrelated page
// type must not appear merely because every page type shares the word "page" in
// its catalog id.
func TestSearchLandingPage(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.landing-page", "LandingPage",
			"The configurable product marketing surface for a landing page. Catalog id: templates.landing-page."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.authentication-page", "AuthenticationPage",
			"Sign-in, registration and recovery surface. Catalog id: templates.authentication-page."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.pricing-page", "PricingPage",
			"Plan comparison and billing surface. Catalog id: templates.pricing-page."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "patterns.hero", "Marketing Hero",
			"Hero block for a landing page."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "patterns.hero-grid", "Hero Grid",
			"Grid block for a landing page."),
	)

	response := search(t, svc, "landing page")
	require.NotEmpty(t, response.Results)
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, response.Results[0].GetKind(),
		"a page type must outrank generic assets for 'landing page'")
	require.Equal(t, "LandingPage", response.Results[0].GetTitle())
	require.Equal(t, int32(3), response.Total, "only the three landing-page documents match")
	for _, result := range response.Results {
		require.NotEqual(t, "AuthenticationPage", result.GetTitle(),
			"unrelated page types must not be returned by an AND query")
		require.NotEqual(t, "PricingPage", result.GetTitle())
	}
}

// TestSearchOnboardingFlowWorkspaceFirst covers outcome-first ranking: a
// workspace is the thing a reader navigates to, so it precedes the assets that
// compose it.
func TestSearchOnboardingFlowWorkspaceFirst(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "workspace-harbor", "Onboarding flow",
			"Curated workspace for first-run onboarding and setup."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "patterns.onboarding-flow", "OnboardingFlow",
			"Reusable onboarding flow pattern. Catalog id: patterns.onboarding-flow."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.onboarding-page", "OnboardingPage",
			"Onboarding surface. Catalog id: templates.onboarding-page."),
	)

	response := search(t, svc, "onboarding flow")
	require.NotEmpty(t, response.Results)
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_WORKSPACE, response.Results[0].GetKind())
	require.Equal(t, "Onboarding flow", response.Results[0].GetTitle())
}

// TestSearchButtonAssetFirst is the asset-first fixture: the exact asset name
// must lead, ahead of related assets and a page type that mentions buttons.
func TestSearchButtonAssetFirst(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button",
			"Primary action control. Catalog id: controls.button."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button-group", "Button group",
			"Grouped actions. Catalog id: controls.button-group."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.form-page", "FormPage",
			"Form surface that uses a button for submission. Catalog id: templates.form-page."),
	)

	response := search(t, svc, "button")
	require.NotEmpty(t, response.Results)
	require.Equal(t, "Button", response.Results[0].GetTitle())
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_ASSET, response.Results[0].GetKind())
}

func TestSemanticLegIsHonest(t *testing.T) {
	svc := newTestService(t)
	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED, status.GetSemanticLeg(),
		"no semantic leg is injected, so the status must not claim one")

	ready, err := svc.WithSemantic(enabledSemantic{}).Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_READY, ready.GetSemanticLeg())

	disabled, err := svc.WithSemantic(disabledSemantic{}).Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED, disabled.GetSemanticLeg())
}

type enabledSemantic struct{}

func (enabledSemantic) Enabled() bool { return true }

type disabledSemantic struct{}

func (disabledSemantic) Enabled() bool { return false }

func TestQuotedPhraseRequiresAdjacency(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "forms.validation", "Validation Summary",
			"The validation error appears below the field."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "forms.rules", "Validation Rules",
			"Validation rules catch an error before submission."),
	)

	response := search(t, svc, `"validation error"`)
	require.Equal(t, int32(1), response.Total, "a quoted phrase matches adjacent tokens only")
	require.Equal(t, "Validation Summary", response.Results[0].GetTitle())
}

func TestNoMatchReturnsEmptyWithIndexedStatus(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."))

	response := search(t, svc, "zzzz-not-in-the-corpus")
	require.Empty(t, response.Results)
	require.Equal(t, int32(0), response.Total)
	require.NotNil(t, response.Status)
	require.True(t, response.Status.GetIndexed(), "no-match is distinct from an empty index")
	require.Equal(t, int32(1), response.Status.GetDocumentCount())
}

func TestEmptyIndexReportsNotIndexed(t *testing.T) {
	svc := newTestService(t)
	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.False(t, status.GetIndexed())
	require.Equal(t, int32(0), status.GetDocumentCount())
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED, status.GetSemanticLeg())

	response := search(t, svc, "")
	require.Empty(t, response.Results)
	require.Equal(t, int32(0), response.Total)
}

func TestKindFilterAppliesBeforeLimit(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button-group", "Button group", "Grouped actions."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.form-page", "FormPage", "A page with a button."),
	)

	all := search(t, svc, "button")
	require.Equal(t, int32(3), all.Total)

	assetsOnly := search(t, svc, "button", searchv1.SearchKind_SEARCH_KIND_ASSET)
	require.Equal(t, int32(2), assetsOnly.Total, "total is the matched count after the kind filter")
	require.Len(t, assetsOnly.Results, 2)
	for _, result := range assetsOnly.Results {
		require.Equal(t, searchv1.SearchKind_SEARCH_KIND_ASSET, result.GetKind())
	}
	foundKinds := false
	for _, facet := range assetsOnly.Facets {
		if facet.GetKey() != "kind" {
			continue
		}
		foundKinds = true
		for _, value := range facet.GetValues() {
			require.NotEqual(t, kindPageType, value.GetValue())
		}
	}
	require.True(t, foundKinds)
}

func TestSuggestUsesIndexedTitlesAndPrefixes(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "ws-onboarding", "Onboarding flow", "First run."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "ws-onboarding-checklist", "Onboarding checklist", "Setup list."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "ws-checkout", "Checkout flow", "Payment."),
	)

	response, err := svc.Suggest(context.Background(), &searchv1.SuggestRequest{Query: "onb", Limit: 10})
	require.NoError(t, err)
	require.Len(t, response.Suggestions, 2)
	require.Equal(t, "Onboarding flow", response.Suggestions[0], "the shorter prefix match sorts first")
	require.Equal(t, "Onboarding checklist", response.Suggestions[1])

	contains, err := svc.Suggest(context.Background(), &searchv1.SuggestRequest{Query: "flow", Limit: 10})
	require.NoError(t, err)
	require.Contains(t, contains.Suggestions, "Onboarding flow")
	require.Contains(t, contains.Suggestions, "Checkout flow")
}

func TestBrowseEmptyQueryIsCuratedAndOrdered(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_COMMAND, "command.library", "Library", "Browse assets."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.landing-page", "LandingPage", "Landing page."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "workspace-harbor", "Harbor", "Workspace."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_DISPLAY, "display-desktop", "Desktop display", "A display."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_STATE, "state-empty", "Empty state", "A state."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_EVIDENCE, "finding-1", "evidence_freshness", "Stale evidence."),
	)

	response := search(t, svc, "   ")
	require.Len(t, response.Results, 4)
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_WORKSPACE, response.Results[0].GetKind())
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, response.Results[1].GetKind())
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_ASSET, response.Results[2].GetKind())
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_COMMAND, response.Results[3].GetKind())
	for _, result := range response.Results {
		require.NotEqual(t, searchv1.SearchKind_SEARCH_KIND_EVIDENCE, result.GetKind(),
			"evidence is a review result, not a browse destination")
		require.NotEqual(t, searchv1.SearchKind_SEARCH_KIND_DISPLAY, result.GetKind())
		require.NotEqual(t, searchv1.SearchKind_SEARCH_KIND_STATE, result.GetKind())
	}

	// Displays and states remain reachable through an explicit filter.
	displayOnly := search(t, svc, "", searchv1.SearchKind_SEARCH_KIND_DISPLAY)
	require.Len(t, displayOnly.Results, 1)
	require.Equal(t, "Desktop display", displayOnly.Results[0].GetTitle())
}

func TestDeduplicatesByKindAndSubject(t *testing.T) {
	svc := newTestService(t)
	lower := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action.")
	higher := lower
	higher.ID = "asset:duplicate-row"
	higher.Body = "Primary action button button button."
	seedDocuments(t, svc, lower, higher)

	response := search(t, svc, "button")
	require.Equal(t, int32(1), response.Total)
	require.Len(t, response.Results, 1)
}

func TestAndFallsBackToOr(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "workspace-harbor", "Onboarding flow",
			"First-run onboarding."),
	)

	response := search(t, svc, "onboarding wizard")
	require.Equal(t, int32(1), response.Total, "AND found nothing, so OR must return the onboarding workspace")
	require.Equal(t, "Onboarding flow", response.Results[0].GetTitle())
}

func TestQuerySyntaxIsTreatedAsData(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.landing-page", "LandingPage", "Landing surface."),
	)

	queries := []string{
		`button (")`,
		`"landing page" (draft)`,
		`button OR * NEAR("landing"`,
		`foo:bar -baz +qux`,
	}
	for _, query := range queries {
		response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: query})
		require.NoErrorf(t, err, "query %q must not produce an FTS error", query)
		require.NotNil(t, response)
	}
	require.Equal(t, "Button", search(t, svc, `button (")`).Results[0].GetTitle())
}

func TestReindexCatalogIsIdempotentAndProjectsAvailability(t *testing.T) {
	root := t.TempDir()
	writeCatalogAsset(t, root, "templates", "landing-page.json", `{
		"kind": "catalog-asset",
		"asset": {
			"id": "templates.landing-page",
			"name": "LandingPage",
			"kind": "page-template",
			"domain": "templates",
			"description": "The configurable product marketing surface for a landing page.",
			"targets": ["react-vite"],
			"target": {"priority": "P1", "maturity": "verified"}
		}
	}`)
	writeCatalogAsset(t, root, "controls", "button.json", `{
		"kind": "catalog-asset",
		"asset": {
			"id": "controls.button",
			"name": "Button",
			"kind": "component",
			"domain": "controls",
			"description": "Primary action control.",
			"targets": ["react-vite"],
			"target": {"priority": "P0", "maturity": "production-ready"}
		}
	}`)

	svc := newServiceAt(t, root, nil)
	firstCount, firstAt, err := svc.Reindex(context.Background(), true)
	require.NoError(t, err)
	// Two catalog assets plus the nine static commands.
	require.Equal(t, 11, firstCount)

	secondCount, secondAt, err := svc.Reindex(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, firstCount, secondCount, "reindex must be idempotent")
	require.NotZero(t, secondAt)

	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.True(t, status.GetIndexed())
	require.Equal(t, int32(11), status.GetDocumentCount())
	require.False(t, status.GetStale())

	// The catalog source marks assets planned when no availability resolver is
	// injected; it never pretends the catalog is built.
	button := search(t, svc, "button", searchv1.SearchKind_SEARCH_KIND_ASSET)
	require.NotEmpty(t, button.Results)
	require.Equal(t, searchv1.SearchAvailability_SEARCH_AVAILABILITY_PLANNED, button.Results[0].GetAvailability())

	// A non-forced rebuild of a current index is a no-op and preserves the
	// watermark.
	thirdCount, thirdAt, err := svc.Reindex(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, firstCount, thirdCount)
	require.Equal(t, secondAt, thirdAt)
	require.False(t, firstAt.IsZero())
}

func TestReindexUsesInjectedAvailability(t *testing.T) {
	root := t.TempDir()
	writeCatalogAsset(t, root, "controls", "button.json", `{
		"kind": "catalog-asset",
		"asset": {
			"id": "controls.button",
			"name": "Button",
			"kind": "component",
			"domain": "controls",
			"description": "Primary action control.",
			"targets": ["react-vite"]
		}
	}`)
	svc := newServiceAt(t, root, func(_ context.Context, catalogID string) (bool, string, error) {
		if catalogID == "controls.button" {
			return true, "built 1.2.0", nil
		}
		return false, "", nil
	})
	count, _, err := svc.Reindex(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, 10, count, "one catalog asset plus nine commands")

	response := search(t, svc, "button")
	require.NotEmpty(t, response.Results)
	require.Equal(t, searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE, response.Results[0].GetAvailability())
	require.Contains(t, response.Results[0].GetBadges(), "built 1.2.0")
}

func TestReindexFailureRetainsPriorIndexAndRecordsError(t *testing.T) {
	// The service root has no catalog directory, so the rebuild fails after the
	// previous projection was seeded.
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."))

	_, _, err := svc.Reindex(context.Background(), true)
	require.Error(t, err)

	status, statusErr := svc.Status(context.Background())
	require.NoError(t, statusErr)
	require.True(t, status.GetIndexed(), "a failed rebuild keeps the prior index queryable")
	require.Equal(t, int32(1), status.GetDocumentCount())
	require.True(t, status.GetStale())
	require.NotEmpty(t, status.GetLastError())

	require.Equal(t, int32(1), search(t, svc, "button").Total)
}

func TestReindexIndexesWorkspaceTablesWhenPresent(t *testing.T) {
	root := t.TempDir()
	writeCatalogAsset(t, root, "controls", "button.json", `{
		"kind": "catalog-asset",
		"asset": {
			"id": "controls.button",
			"name": "Button",
			"kind": "component",
			"domain": "controls",
			"description": "Primary action control.",
			"targets": ["react-vite"]
		}
	}`)
	svc := newServiceAt(t, root, nil)
	createWorkspaceTables(t, svc.DB())
	seedWorkspaceRows(t, svc.DB())

	count, _, err := svc.Reindex(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, 14, count, "nine commands, one catalog asset, workspace, page, display and state")

	workspace := search(t, svc, "Onboarding flow", searchv1.SearchKind_SEARCH_KIND_WORKSPACE)
	require.NotEmpty(t, workspace.Results)
	require.Equal(t, "Onboarding flow", workspace.Results[0].GetTitle())
	require.Equal(t, "/design?workspace=w1", workspace.Results[0].GetHref())
	require.Contains(t, workspace.Results[0].GetRelations(), "workspace:w1")

	page := search(t, svc, "Welcome", searchv1.SearchKind_SEARCH_KIND_PAGE)
	require.NotEmpty(t, page.Results)
	require.Equal(t, "/design?workspace=w1&page=p1", page.Results[0].GetHref())
	require.Contains(t, page.Results[0].GetRelations(), "page:p1")

	display := search(t, svc, "Desktop", searchv1.SearchKind_SEARCH_KIND_DISPLAY)
	require.NotEmpty(t, display.Results)
	require.Equal(t, "/design?workspace=w1&page=p1&display=d1", display.Results[0].GetHref())

	state := search(t, svc, "Step 1", searchv1.SearchKind_SEARCH_KIND_STATE)
	require.NotEmpty(t, state.Results)
	require.Equal(t, "/design?workspace=w1&page=p1&display=d1&state=s1", state.Results[0].GetHref())
	require.Contains(t, state.Results[0].GetRelations(), "display:d1")
}

func TestReindexIndexesFindingsWhenPresent(t *testing.T) {
	root := t.TempDir()
	writeCatalogAsset(t, root, "controls", "button.json", `{
		"kind": "catalog-asset",
		"asset": {
			"id": "controls.button",
			"name": "Button",
			"kind": "component",
			"domain": "controls",
			"description": "Primary action control.",
			"targets": ["react-vite"]
		}
	}`)
	svc := newServiceAt(t, root, nil)
	_, err := svc.DB().Exec(`CREATE TABLE findings (
		id TEXT PRIMARY KEY,
		subject_kind TEXT NOT NULL DEFAULT '',
		subject_id TEXT NOT NULL DEFAULT '',
		gate TEXT NOT NULL DEFAULT '',
		severity TEXT NOT NULL DEFAULT '',
		evidence_json TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL DEFAULT '',
		owner TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT ''
	)`)
	require.NoError(t, err)
	// Only open blocker/high findings are searchable evidence. A 14k-row queue
	// would otherwise drown every lexical query in findings.
	_, err = svc.DB().Exec(`INSERT INTO findings (id, subject_kind, subject_id, gate, severity, evidence_json, status, owner, created_at)
		VALUES
		('f1', 'asset', 'controls.button', 'catalog.evidence_freshness', 'blocker', '{"ageDays":42}', 'open', 'catalog', '2026-11-19T10:00:00Z'),
		('f2', 'asset', 'controls.button', 'catalog.evidence_freshness', 'medium', '{}', 'open', 'catalog', '2026-11-19T10:00:00Z'),
		('f3', 'asset', 'controls.button', 'catalog.evidence_freshness', 'blocker', '{}', 'fixed', 'catalog', '2026-11-19T10:00:00Z')`)
	require.NoError(t, err)

	_, _, err = svc.Reindex(context.Background(), true)
	require.NoError(t, err)

	evidence := search(t, svc, "evidence freshness", searchv1.SearchKind_SEARCH_KIND_EVIDENCE)
	require.Len(t, evidence.Results, 1, "medium and resolved findings stay out of search")
	require.Equal(t, "/review?finding=f1", evidence.Results[0].GetHref())
	require.Contains(t, evidence.Results[0].GetBadges(), "blocker")
}

// TestLexicalQueriesAreFast guards the p95 budget with a deliberately generous
// ceiling: 50 queries over the full-catalog scale corpus must fit in 250 ms.
// It catches accidental per-query work (loading files, rebuilding FTS) without
// being flaky on a loaded CI machine.
func TestLexicalQueriesAreFast(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("wall-clock budget is not meaningful under the race detector")
	}
	svc := newTestService(t)
	docs := make([]Document, 0, 500)
	for i := 0; i < 500; i++ {
		docs = append(docs, testDoc(
			searchv1.SearchKind_SEARCH_KIND_ASSET,
			fmt.Sprintf("asset.%03d", i),
			fmt.Sprintf("Asset %03d", i),
			"Reusable component for landing page onboarding flow button form field validation navigation card grid.",
		))
	}
	seedDocuments(t, svc, docs...)

	queries := []string{"button", "landing page", "onboarding flow", "form field", "navigation card", "asset 1", "zzzz absent", `"validation error"`, "grid", "flow"}
	started := time.Now()
	for i := 0; i < 50; i++ {
		response, err := svc.Search(context.Background(), &searchv1.SearchRequest{
			Query: queries[i%len(queries)],
			Limit: 20,
		})
		require.NoError(t, err)
		require.NotNil(t, response)
	}
	elapsed := time.Since(started)
	t.Logf("50 lexical queries over 500 documents took %s", elapsed)
	if elapsed > 250*time.Millisecond {
		t.Fatalf("50 lexical queries took %s, above the 250ms regression ceiling", elapsed)
	}
}

func writeCatalogAsset(t *testing.T, scenarioRoot, domain, filename, contents string) {
	t.Helper()
	dir := filepath.Join(scenarioRoot, "catalog", "assets", domain)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(contents), 0o644))
}

func createWorkspaceTables(t *testing.T, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`
CREATE TABLE workspaces (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  provenance TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  binding_scenario TEXT NOT NULL DEFAULT '',
  binding_page TEXT NOT NULL DEFAULT '',
  theme_json TEXT NOT NULL DEFAULT '{}',
  options_json TEXT NOT NULL DEFAULT '{}',
  revision INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE workspace_pages (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  name TEXT NOT NULL,
  route TEXT NOT NULL DEFAULT '',
  page_type_id TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  spec_json TEXT NOT NULL DEFAULT '{}'
);
CREATE TABLE workspace_displays (
  id TEXT PRIMARY KEY,
  page_id TEXT NOT NULL,
  device TEXT NOT NULL DEFAULT '',
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE workspace_states (
  id TEXT PRIMARY KEY,
  display_id TEXT NOT NULL,
  name TEXT NOT NULL,
  args_json TEXT NOT NULL DEFAULT '{}',
  scenario_data_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT '',
  capture_id TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0
);`)
	require.NoError(t, err)
}

func seedWorkspaceRows(t *testing.T, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`
INSERT INTO workspaces (id, name, provenance, tags_json, binding_scenario, binding_page, updated_at)
VALUES ('w1', 'Onboarding flow', 'curated', '["onboarding","first-run"]', 'harbor', 'welcome', '2026-11-19T09:00:00Z');
INSERT INTO workspace_pages (id, workspace_id, name, route, page_type_id, sort_order, spec_json)
VALUES ('p1', 'w1', 'Welcome', '/welcome', 'templates.onboarding-page', 0, '{"purpose":"greet the operator"}');
INSERT INTO workspace_displays (id, page_id, device, width, height, sort_order)
VALUES ('d1', 'p1', 'desktop', 1440, 900, 0);
INSERT INTO workspace_states (id, display_id, name, args_json, scenario_data_json, status, sort_order)
VALUES ('s1', 'd1', 'Step 1', '{"step":1}', '{}', 'draft', 0);`)
	require.NoError(t, err)
}

func TestSchemaIsIdempotent(t *testing.T) {
	database := databasetest.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(Schema)))
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(Schema)))
	require.True(t, strings.Contains(Schema(), "search_docs"))
}

// A projection change must invalidate an otherwise current-looking index.
// Without this, a fixed href or field shape would never reach a running
// workbench because the source content did not change.
func TestStatusTreatsAnOlderProjectionAsStaleAndRebuilds(t *testing.T) {
	root := t.TempDir()
	writeCatalogAsset(t, root, "controls", "button.json", `{"kind":"catalog-asset","asset":{"id":"controls.button","name":"Button","kind":"component","domain":"controls","description":"Primary action control.","targets":["react-vite"],"target":{"priority":"P0","maturity":"production-ready"}}}`)
	service := newServiceAt(t, root, nil)
	ctx := context.Background()
	count, _, err := service.Reindex(ctx, true)
	require.NoError(t, err)
	require.Greater(t, count, 0)
	status, err := service.Status(ctx)
	require.NoError(t, err)
	require.False(t, status.GetStale())

	_, err = service.db.ExecContext(ctx, `UPDATE search_state SET value = '1' WHERE key = ?`, stateProjectionVersion)
	require.NoError(t, err)
	status, err = service.Status(ctx)
	require.NoError(t, err)
	require.True(t, status.GetStale(), "an index from an older projection is stale")

	rebuilt, _, err := service.Reindex(ctx, false)
	require.NoError(t, err)
	require.Greater(t, rebuilt, 0, "a non-forced reindex must rebuild when the projection version moved")
	status, err = service.Status(ctx)
	require.NoError(t, err)
	require.False(t, status.GetStale())
}
