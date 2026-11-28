package search

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	"react-component-library/internal/catalogcoverage"
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

func TestCatalogSearchEvidenceExplainsCustomLayerReplacement(t *testing.T) {
	body := catalogBody(catalogcoverage.Asset{
		ID: "patterns.card", Name: "Card", Description: "A reusable card.",
	})
	for _, phrase := range []string{
		"Custom-layer replacement workflow",
		"search replacement candidates",
		"semantically matching RCL primitive",
		"Container layers require a container-capable match",
		"preserves layer identity and authored fallback content",
	} {
		require.Contains(t, body, phrase)
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

func TestLandingPageWorkspaceRanksBeforeArchetypeAndSuppressesGenericPageTypes(t *testing.T) {
	svc := newTestService(t)
	workspace := testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "ws-curated-marlow", "Marlow", "A shop landing page for an editorial product story.")
	landingType := testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.landing-page", "LandingPage", "Product marketing surface. Catalog id: templates.landing-page.")
	onboardingType := testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.onboarding-page", "OnboardingPage", "A multi-step onboarding page.")
	page := testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE, "pg-curated-marlow", "Marlow", "Marlow realizes templates.landing-page.")
	asset := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "patterns.hero", "Marketing Hero", "Hero for a landing page.")
	for _, doc := range []*Document{&workspace, &landingType, &onboardingType, &page, &asset} {
		doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	}
	seedDocuments(t, svc, workspace, landingType, onboardingType, page, asset)
	svc.WithSemantic(&semanticFixture{hits: []aisearch.SearchResult{
		{ID: onboardingType.ID, Score: .99},
		{ID: workspace.ID, Score: .95},
		{ID: landingType.ID, Score: .90},
		{ID: page.ID, Score: .85},
		{ID: asset.ID, Score: .80},
	}})

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "landing page", Limit: 10})
	require.NoError(t, err)
	titles := responseTitles(response)
	require.NotEmpty(t, titles)
	require.Equal(t, "Marlow", titles[0], "the curated workspace realization should precede its page archetype")
	require.Less(t, indexOfTitle(titles, "Marlow"), indexOfTitle(titles, "LandingPage"))
	require.NotContains(t, titles, "OnboardingPage", "a generic semantic page-type neighbor must not match this named outcome")
	require.Contains(t, titles, "Marlow", "matching workspace realizations remain discoverable")
	require.Contains(t, titles, "Marketing Hero", "composed assets remain discoverable after the outcome results")
	require.Less(t, indexOfTitle(titles, "Marlow"), indexOfTitle(titles, "Marketing Hero"))
}

func TestSearchExpandsAuthoredBlueprintDependenciesAndKeepsAssetQueriesAssetFirst(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	svc := newServiceAt(t, root, nil)
	database := svc.DB()
	createWorkspaceTables(t, database)
	_, err := database.Exec(`INSERT INTO workspaces (id, name, summary) VALUES ('marlow', 'Marlow', 'Editorial landing page for a product collection.');
INSERT INTO workspace_pages (id, workspace_id, name, page_type_id, spec_json) VALUES ('marlow-home', 'marlow', 'Home', 'templates.landing-page', '{"type":"page","children":[]}');`)
	require.NoError(t, err)
	_, _, err = svc.Reindex(context.Background(), true)
	require.NoError(t, err)

	response := search(t, svc, "landing page")
	require.GreaterOrEqual(t, len(response.Results), 3)
	require.Equal(t, "Marlow", response.Results[0].GetTitle(), "the matching curated workspace is the outcome-first result")
	require.Contains(t, responseTitles(response), "LandingPage")
	var hasRequiredAsset bool
	for _, result := range response.Results {
		require.NotEqual(t, "OnboardingPage", result.GetTitle(), "unrelated page types remain suppressed")
		if result.GetId() == "asset:primitives.section" {
			hasRequiredAsset = true
			require.Contains(t, result.GetRelations(), "page-type-requires:templates.landing-page")
			require.Contains(t, result.GetBadges(), "Required by LandingPage")
			require.Contains(t, result.GetSnippet(), "does not assert that the workspace renders the asset")
		}
		require.NotEqual(t, "asset:patterns.onboarding-flow", result.GetId(), "unrelated page type requirements are not expanded")
	}
	require.True(t, hasRequiredAsset, "the blueprint's authored required catalog asset is discoverable")

	buttons := search(t, svc, "button")
	require.NotEmpty(t, buttons.Results)
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_ASSET, buttons.Results[0].GetKind(), "direct asset queries retain asset-first ranking")
}

func indexOfTitle(titles []string, target string) int {
	for index, title := range titles {
		if title == target {
			return index
		}
	}
	return -1
}

// TestSearchOnboardingFlowWorkspaceFirst covers outcome-first ranking: a
// workspace is the thing a reader navigates to, so it precedes the assets that
// compose it.
func TestSearchOnboardingFlowWorkspaceFirst(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "workspace-harbor", "Harbor",
			"Curated workspace for an onboarding flow that helps new teams get set up."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "patterns.onboarding-flow", "OnboardingFlow",
			"Reusable onboarding flow pattern. Catalog id: patterns.onboarding-flow."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.onboarding-page", "OnboardingPage",
			"Onboarding surface. Catalog id: templates.onboarding-page."),
	)

	response := search(t, svc, "onboarding flow")
	require.NotEmpty(t, response.Results)
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_WORKSPACE, response.Results[0].GetKind())
	require.Equal(t, "Harbor", response.Results[0].GetTitle())
}

// TestSearchButtonAssetFirst is the asset-first fixture: the exact asset name
// must lead, ahead of related assets and a page type that mentions buttons.
func TestSearchButtonAssetFirst(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "workspace-button-guide", "Button patterns", "A guide page about choosing and using a button."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE, "page-button-guide", "Button guide", "How to use a button in a workspace."),
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

	assetOnly := search(t, svc, "button", searchv1.SearchKind_SEARCH_KIND_ASSET)
	require.Equal(t, "Button", assetOnly.Results[0].GetTitle(), "explicit asset filtering keeps the exact asset first")
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

func TestSemanticReadinessCacheCoalescesExpiresAndRecovers(t *testing.T) {
	var calls atomic.Int32
	var available atomic.Bool
	available.Store(true)
	semantic := &semanticIndex{readinessProbe: func(context.Context) bool {
		calls.Add(1)
		time.Sleep(25 * time.Millisecond)
		return available.Load()
	}}
	const concurrentChecks = 12
	var wg sync.WaitGroup
	results := make(chan bool, concurrentChecks)
	for range concurrentChecks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- semantic.Ready(context.Background())
		}()
	}
	wg.Wait()
	close(results)
	for ready := range results {
		require.True(t, ready)
	}
	require.EqualValues(t, 1, calls.Load(), "concurrent status reads should share a probe")
	require.True(t, semantic.Ready(context.Background()), "fresh snapshot should be reused")
	require.EqualValues(t, 1, calls.Load())

	semantic.readinessMu.Lock()
	semantic.readinessAt = time.Now().Add(-semanticReadinessTTL)
	semantic.readinessMu.Unlock()
	available.Store(false)
	require.False(t, semantic.Ready(context.Background()), "expired snapshot must be refreshed")
	require.EqualValues(t, 2, calls.Load())

	semantic.setError(errors.New("semantic search failed"))
	require.False(t, semantic.Ready(context.Background()), "search failure degrades readiness immediately")
	require.EqualValues(t, 2, calls.Load(), "known failure must not trigger a network probe")
	available.Store(true)
	semantic.setError(nil)
	require.True(t, semantic.Ready(context.Background()), "successful recovery must be visible on the next status read")
	require.EqualValues(t, 3, calls.Load())
}

func TestSemanticReadinessProbeTimeoutKeepsLexicalStatusResponsive(t *testing.T) {
	semantic := &semanticIndex{readinessProbe: func(ctx context.Context) bool {
		<-ctx.Done()
		return false
	}}
	started := time.Now()
	require.False(t, semantic.Ready(context.Background()))
	require.Less(t, time.Since(started), 2*time.Second, "a slow semantic health probe must not block lexical search status")
	require.False(t, semantic.Ready(context.Background()), "the timed-out readiness result should be reused")
}

func TestSemanticParaphraseAddsProjectedOutcomeAndHonorsKindFilter(t *testing.T) {
	svc := newTestService(t)
	doc := testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "workspace-checkout", "Checkout flow", "A guided purchase experience.")
	doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	asset := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "patterns.transaction", "Transaction path", "A reusable transaction path.")
	asset.ID = DocumentID(asset.Kind, asset.SubjectID)
	seedDocuments(t, svc, doc, asset)
	semantic := &semanticFixture{hits: []aisearch.SearchResult{{ID: doc.ID, Score: .9}, {ID: asset.ID, Score: .1}}}
	svc.WithSemantic(semantic)
	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{
		Query: "transaction path", Kinds: []searchv1.SearchKind{searchv1.SearchKind_SEARCH_KIND_ASSET}, Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, response.GetResults(), 1, "kind filtering must happen before the requested limit")
	require.Equal(t, "Transaction path", response.GetResults()[0].GetTitle())
	require.Equal(t, int32(1), response.GetTotal())
	require.NotNil(t, semantic.query.Filter)
	require.Equal(t, "kind", semantic.query.Filter.Must[0].Key)
	require.Equal(t, []any{"asset"}, semantic.query.Filter.Must[0].AnyOf)
}

func TestSemanticFusionFailureIsObservableAndKeepsLexicalResults(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control."))
	var logs strings.Builder
	svc.WithLogger(log.New(&logs, "", 0))
	svc.WithSemantic(&semanticFixture{err: errors.New("vector backend unavailable")})
	response := search(t, svc, "button")
	require.Equal(t, "Button", response.GetResults()[0].GetTitle())
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED, response.GetStatus().GetSemanticLeg())
	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED, status.GetSemanticLeg())
	require.Contains(t, logs.String(), "semantic retrieval leg degraded")
}

func TestUnprojectableSemanticHitMarksResponseDegraded(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control."))
	svc.WithSemantic(&semanticFixture{hits: []aisearch.SearchResult{{ID: "stale-vector-hit", Score: 1}}})
	response := search(t, svc, "button")
	require.NotEmpty(t, response.GetResults(), "lexical results remain usable")
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED, response.GetStatus().GetSemanticLeg())
}

func TestSemanticCancellationIsNotSwallowedAsFallback(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control."))
	svc.WithSemantic(&semanticFixture{err: context.Canceled})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.Search(ctx, &searchv1.SearchRequest{Query: "button"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestProjectFusedBatchPreservesRankAndSkipsStaleHits(t *testing.T) {
	svc := newTestService(t)
	docs := make([]Document, 100)
	fused := make([]aisearch.FusedResult, 0, 101)
	for i := range docs {
		docs[i] = testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, fmt.Sprintf("asset.%03d", i), fmt.Sprintf("Asset %03d", i), "A reusable asset.")
		docs[i].ID = DocumentID(docs[i].Kind, docs[i].SubjectID)
	}
	seedDocuments(t, svc, docs...)
	for i := len(docs) - 1; i >= 0; i-- {
		fused = append(fused, aisearch.FusedResult{Result: aisearch.SearchResult{ID: docs[i].ID, Score: float64(i + 1)}})
	}
	fused = append(fused, aisearch.FusedResult{Result: aisearch.SearchResult{ID: "stale-vector-id", Score: 1}})

	got, err := svc.projectFused(context.Background(), fused)
	require.NoError(t, err)
	require.Len(t, got, len(docs))
	require.Equal(t, docs[len(docs)-1].ID, got[0].doc.ID, "projection must retain fused rank order")
	require.Equal(t, docs[0].ID, got[len(got)-1].doc.ID)
}

func TestSemanticOnlyGenericNeighborCannotDisplaceLexicalOutcome(t *testing.T) {
	svc := newTestService(t)
	workspace := testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "ws-curated-marlow", "Marlow", "A shop landing page, art directed for an editorial product story.")
	landingType := testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, "templates.landing-page", "LandingPage", "Landing page template for a product marketing surface.")
	page := testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE, "pg-curated-marlow", "Marlow", "Marlow templates.landing-page with a product story.")
	defaultPage := testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE, "pg-empty", "Page 1", "Page 1\n{}\nPage id: pg-empty")
	defaultPage.Subtitle = "Page · Untitled workspace"
	meaningfulPage := testDoc(searchv1.SearchKind_SEARCH_KIND_PAGE, "pg-campaign", "Campaign journal", "An editorial campaign launch with a product narrative and art-directed sections.")
	for _, doc := range []*Document{&workspace, &landingType, &page, &defaultPage, &meaningfulPage} {
		doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	}
	seedDocuments(t, svc, workspace, landingType, page, defaultPage, meaningfulPage)
	// The vector leg ranks the empty editor default as a broad neighbor, while
	// also returning a meaningful semantic-only page that remains discoverable.
	svc.WithSemantic(&semanticFixture{hits: []aisearch.SearchResult{
		{ID: defaultPage.ID, Score: .99},
		{ID: meaningfulPage.ID, Score: .95},
		{ID: workspace.ID, Score: .90},
		{ID: landingType.ID, Score: .85},
		{ID: page.ID, Score: .80},
	}})

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "landing page", Limit: 5})
	require.NoError(t, err)
	titles := responseTitles(response)
	require.NotContains(t, titles, "Page 1", "empty default pages must not be semantic-only results")
	require.Contains(t, titles, "Campaign journal", "meaningful semantic-only pages should remain discoverable")
	require.Equal(t, int32(4), response.GetTotal())
}

func responseTitles(response *searchv1.SearchResponse) []string {
	titles := make([]string, 0, len(response.GetResults()))
	for _, result := range response.GetResults() {
		titles = append(titles, result.GetTitle())
	}
	return titles
}

func TestUnavailableSemanticLegReportsDegradedAndKeepsFTSRanking(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc,
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control."),
		testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button-group", "Button group", "Grouped button actions."),
	)
	svc.WithSemantic(UnavailableSemantic(fmt.Errorf("qdrant is unavailable")))
	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED, status.GetSemanticLeg())
	response := search(t, svc, "button")
	require.Equal(t, "Button", response.GetResults()[0].GetTitle(), "FTS outcome-first ordering remains available during semantic startup failure")
}

func TestSemanticStartupFailureDegradesWithoutInterruptingFTS(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control."))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate startup before Ollama policy/config is reachable
	semantic, err := NewAISemantic(ctx, svc.DB(), aisearch.Config{EmbedRole: aisearch.DefaultEmbedRole}, aisearch.TuningConfig{Engine: aisearch.EngineHybrid}, "rcl-test")
	require.Error(t, err)
	svc.WithSemantic(semantic)
	status, statusErr := svc.Status(context.Background())
	require.NoError(t, statusErr)
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED, status.GetSemanticLeg())
	response := search(t, svc, "button")
	require.Equal(t, "Button", response.GetResults()[0].GetTitle())
}

func TestSemanticReconcileRunsWhenLexicalProjectionIsAlreadyCurrent(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control."))
	semantic := &semanticFixture{reconcileStarted: make(chan struct{}, 1)}
	svc.WithSemantic(semantic)
	_, _, err := svc.Reindex(context.Background(), false)
	require.NoError(t, err)
	select {
	case <-semantic.reconcileStarted:
	case <-time.After(time.Second):
		t.Fatal("semantic reconciliation did not start after lexical reindex")
	}
}

func TestSemanticReindexTriggersCoalesceWhileReconciliationRuns(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control."))
	semantic := &blockingSemantic{started: make(chan int, 3), release: make(chan struct{}, 3)}
	svc.WithSemantic(semantic)

	_, _, err := svc.Reindex(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, 1, <-semantic.started)
	for range 3 {
		_, _, err = svc.Reindex(context.Background(), false)
		require.NoError(t, err)
	}

	semantic.release <- struct{}{}
	require.Equal(t, 2, <-semantic.started, "concurrent changes should schedule one follow-up pass")
	semantic.release <- struct{}{}
	select {
	case extra := <-semantic.started:
		t.Fatalf("coalesced triggers started an unnecessary pass %d", extra)
	case <-time.After(25 * time.Millisecond):
	}
	require.EqualValues(t, 2, semantic.calls.Load())
}

func TestProjectionSourceUsesStableContentHashes(t *testing.T) {
	svc := newTestService(t)
	doc := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control.")
	doc.UpdatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	seedDocuments(t, svc, doc)
	source := projectionSource{db: svc.DB()}
	first, err := source.LoadAll(context.Background())
	require.NoError(t, err)
	doc.UpdatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seedDocuments(t, svc, doc)
	second, err := source.LoadAll(context.Background())
	require.NoError(t, err)
	require.Equal(t, first[0].ID, second[0].ID)
	require.Equal(t, first[0].ContentHash, second[0].ContentHash, "timestamp-only changes must not re-embed content")
	doc.Body = "A different button purpose."
	seedDocuments(t, svc, doc)
	third, err := source.LoadAll(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, second[0].ContentHash, third[0].ContentHash, "meaningful source changes must reconcile")
}

func TestHybridReconcilerIndexesProjectionUpdatesAndDeletes(t *testing.T) {
	svc := newTestService(t)
	doc := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action control.")
	doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	seedDocuments(t, svc, doc)
	store := searchtest.NewVectorStore()
	embedder := &searchtest.Embedder{Vector: make([]float64, aisearch.DefaultVectorSize), AvailableValue: true}
	binding := aisearch.NewHybridBinding("workbench", "rcl:", store, projectionSource{db: svc.DB()},
		aisearch.NewIdentityChunker(), aisearch.NewIdentityComposer(), aisearch.NewBM25SparseEncoder())
	reconciler := aisearch.NewReconciler(embedder, []aisearch.SourceBinding{binding}, 1)
	index := &semanticIndex{
		store:         store,
		spec:          aisearch.CollectionSpec{Name: "test", DenseSize: aisearch.DefaultVectorSize, DenseDistance: aisearch.DefaultDenseDistance, Sparse: true, SparseModifier: aisearch.DefaultSparseModifier},
		reconciler:    reconciler,
		reconcileGate: make(chan struct{}, 1),
	}
	require.NoError(t, index.Reconcile(context.Background()))
	require.Len(t, store.Points, 1)
	for _, point := range store.Points {
		require.Equal(t, doc.ID, point.Payload["source_id"])
		require.NotNil(t, point.Sparse, "hybrid profile must persist the sparse retrieval vector")
	}
	doc.Body = "A secondary action surface."
	seedDocuments(t, svc, doc)
	require.NoError(t, index.Reconcile(context.Background()))
	for _, point := range store.Points {
		require.Contains(t, point.Payload["body"], doc.Body)
	}
	require.NoError(t, svc.ReplaceDocuments(context.Background(), nil))
	require.NoError(t, index.Reconcile(context.Background()))
	require.Empty(t, store.Points, "source-level deletion must remove semantic ghost points")
}

func TestSemanticReconcileSerializesConcurrentReindexRequests(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var active atomic.Int32
	var overlapped atomic.Bool
	index := &semanticIndex{
		reconcileGate: make(chan struct{}, 1),
		reconcileFunc: func(context.Context) error {
			if active.Add(1) != 1 {
				overlapped.Store(true)
			}
			entered <- struct{}{}
			<-release
			active.Add(-1)
			return nil
		},
	}
	results := make(chan error, 2)
	go func() { results <- index.Reconcile(context.Background()) }()
	<-entered
	secondStarted := make(chan struct{})
	go func() { close(secondStarted); results <- index.Reconcile(context.Background()) }()
	<-secondStarted
	select {
	case <-entered:
		t.Fatal("second semantic reconcile overlapped the active run")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-results)
	require.NoError(t, <-results)
	require.False(t, overlapped.Load())
}

func TestCommittedSearchConfigParsesAndPinsProviderTuning(t *testing.T) {
	config, err := aisearch.LoadSearchFile(filepath.Join("..", "..", "..", ".vrooli", "search.json"))
	require.NoError(t, err)
	provider, ok := config.Provider("react-component-library.workbench")
	require.True(t, ok)
	require.Equal(t, aisearch.EngineHybrid, provider.ResolvedTuning().Engine)
	require.NotNil(t, provider.ConfigWritable)
	require.False(t, *provider.ConfigWritable)
}

type semanticFixture struct {
	hits             []aisearch.SearchResult
	err              error
	query            aisearch.SearchQuery
	reconcileCalls   int
	reconcileStarted chan struct{}
}

type blockingSemantic struct {
	calls   atomic.Int32
	started chan int
	release chan struct{}
}

func (*blockingSemantic) Enabled() bool              { return true }
func (*blockingSemantic) Ready(context.Context) bool { return true }
func (s *blockingSemantic) Reconcile(context.Context) error {
	call := int(s.calls.Add(1))
	s.started <- call
	<-s.release
	return nil
}

func (s *semanticFixture) Enabled() bool              { return true }
func (s *semanticFixture) Ready(context.Context) bool { return true }
func (s *semanticFixture) Reconcile(context.Context) error {
	s.reconcileCalls++
	if s.reconcileStarted != nil {
		select {
		case s.reconcileStarted <- struct{}{}:
		default:
		}
	}
	return nil
}

func (s *semanticFixture) SearchSemantic(_ context.Context, query aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	s.query = query
	return s.hits, s.err
}

func TestAskDoesNotSilentlyReturnLexicalResults(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."))
	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "button", Ask: true})
	require.Nil(t, response)
	require.ErrorIs(t, err, ErrAskUnavailable)
}

type failingSemantic struct{}

func (failingSemantic) Enabled() bool { return true }
func (failingSemantic) Reconcile(context.Context) error {
	return errors.New("fixture reconcile failure")
}

type recordingLogger struct {
	mu     sync.Mutex
	lines  []string
	called chan struct{}
}

func (l *recordingLogger) Printf(format string, args ...any) {
	l.mu.Lock()
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
	l.mu.Unlock()
	select {
	case l.called <- struct{}{}:
	default:
	}
}

func TestSemanticReconciliationUsesInjectedLogger(t *testing.T) {
	logger := &recordingLogger{called: make(chan struct{}, 1)}
	svc := (&Service{}).WithSemantic(failingSemantic{}).WithLogger(logger)
	svc.reconcileSemanticAsync()
	select {
	case <-logger.called:
	case <-time.After(time.Second):
		t.Fatal("injected logger was not called")
	}
	logger.mu.Lock()
	defer logger.mu.Unlock()
	require.Len(t, logger.lines, 1)
	require.Contains(t, logger.lines[0], "fixture reconcile failure")
}

type answererFunc func(context.Context, string, []Evidence) (Answer, error)

func (f answererFunc) Answer(ctx context.Context, query string, evidence []Evidence) (Answer, error) {
	return f(ctx, query, evidence)
}

func TestAskUsesBoundedIndexedEvidenceAndReturnsVerifiedCitations(t *testing.T) {
	svc := newTestService(t)
	doc := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action.")
	doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	seedDocuments(t, svc, doc)
	svc.WithAnswerer(answererFunc(func(_ context.Context, query string, evidence []Evidence) (Answer, error) {
		require.Equal(t, "what is a button", query)
		require.Len(t, evidence, 1)
		require.Equal(t, doc.ID, evidence[0].ID)
		require.Equal(t, doc.Body, evidence[0].Body)
		return Answer{Text: "A button is a primary action control.", Citations: []string{doc.ID}}, nil
	}))

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "what is a button", Ask: true})
	require.NoError(t, err)
	require.NotNil(t, response.GetAnswer())
	require.Equal(t, "A button is a primary action control.", response.GetAnswer().GetText())
	require.Equal(t, []string{doc.ID}, []string{response.GetAnswer().GetCitations()[0].GetResultId()})
	require.Equal(t, doc.ID, response.GetResults()[0].GetId())
}

func TestAskRejectsCitationOutsideSuppliedResults(t *testing.T) {
	svc := newTestService(t)
	doc := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action.")
	doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	seedDocuments(t, svc, doc)
	svc.WithAnswerer(answererFunc(func(context.Context, string, []Evidence) (Answer, error) {
		return Answer{Text: "Unsupported claim.", Citations: []string{"invented:result"}}, nil
	}))

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "button", Ask: true})
	require.Nil(t, response)
	require.ErrorIs(t, err, ErrAskInvalidAnswer)
}

func TestAskRejectsOpaqueResultIdentifierAsAnswer(t *testing.T) {
	svc := newTestService(t)
	doc := testDoc(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, "ws_curated_marlow", "Marlow", "Editorial-style landing page workspace.")
	doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	seedDocuments(t, svc, doc)
	svc.WithAnswerer(answererFunc(func(context.Context, string, []Evidence) (Answer, error) {
		return Answer{Text: doc.ID, Citations: []string{doc.ID}}, nil
	}))

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "landing page", Ask: true})
	require.Nil(t, response)
	require.ErrorIs(t, err, ErrAskInvalidAnswer)
}

func TestAskRejectsDuplicateCitationsWithoutGatewaySchemaSupport(t *testing.T) {
	svc := newTestService(t)
	doc := testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action.")
	doc.ID = DocumentID(doc.Kind, doc.SubjectID)
	seedDocuments(t, svc, doc)
	svc.WithAnswerer(answererFunc(func(context.Context, string, []Evidence) (Answer, error) {
		return Answer{Text: "A primary action control.", Citations: []string{doc.ID, doc.ID}}, nil
	}))

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "button", Ask: true})
	require.Nil(t, response)
	require.ErrorIs(t, err, ErrAskInvalidAnswer)
}

func TestAskRejectsAnswerWithoutCitations(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."))
	svc.WithAnswerer(answererFunc(func(context.Context, string, []Evidence) (Answer, error) {
		return Answer{Text: "A primary action control."}, nil
	}))

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "button", Ask: true})
	require.Nil(t, response)
	require.ErrorIs(t, err, ErrAskInvalidAnswer)
}

func TestAskBoundsEvidenceCountAndContent(t *testing.T) {
	svc := newTestService(t)
	docs := make([]Document, 10)
	for i := range docs {
		docs[i] = testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, fmt.Sprintf("controls.button-%02d", i), fmt.Sprintf("Button %02d", i), strings.Repeat("evidence ", 400))
		docs[i].ID = DocumentID(docs[i].Kind, docs[i].SubjectID)
	}
	seedDocuments(t, svc, docs...)
	svc.WithAnswerer(answererFunc(func(_ context.Context, _ string, evidence []Evidence) (Answer, error) {
		require.Len(t, evidence, maxAskEvidence)
		for _, item := range evidence {
			require.LessOrEqual(t, len(item.Body), maxEvidencePerResult)
			require.LessOrEqual(t, len(item.Title), maxEvidenceTitle)
			require.LessOrEqual(t, len(item.Subtitle), maxEvidenceSubtitle)
			require.LessOrEqual(t, len(item.ID), maxEvidenceID)
		}
		return Answer{Text: "These are buttons.", Citations: []string{evidence[0].ID}}, nil
	}))

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "button", Ask: true})
	require.NoError(t, err)
	require.Len(t, response.GetResults(), maxAskEvidence)
}

func TestAskFailureDoesNotBecomeLexicalAnswer(t *testing.T) {
	svc := newTestService(t)
	seedDocuments(t, svc, testDoc(searchv1.SearchKind_SEARCH_KIND_ASSET, "controls.button", "Button", "Primary action."))
	svc.WithAnswerer(answererFunc(func(context.Context, string, []Evidence) (Answer, error) {
		return Answer{}, fmt.Errorf("gateway unavailable")
	}))

	response, err := svc.Search(context.Background(), &searchv1.SearchRequest{Query: "button", Ask: true})
	require.Nil(t, response)
	require.ErrorContains(t, err, "gateway unavailable")
	lexical := search(t, svc, "button")
	require.NotEmpty(t, lexical.GetResults(), "ordinary lexical search remains available after Ask failure")
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
	require.Contains(t, workspace.Results[0].GetRelations(), "page-type:templates.onboarding-page")
	summary := search(t, svc, "guided launch", searchv1.SearchKind_SEARCH_KIND_WORKSPACE)
	require.NotEmpty(t, summary.Results, "workspace summaries must be searchable")
	require.Equal(t, "Onboarding flow", summary.Results[0].GetTitle())

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

// TestLexicalQueriesAreFast guards the lexical p95 budget at a corpus larger
// than the served catalog. It catches accidental per-query work (loading files,
// rebuilding FTS) while measuring each query independently so aggregate suite
// scheduling does not distort the latency threshold.
func TestLexicalQueriesAreFast(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("wall-clock budget is not meaningful under the race detector")
	}
	svc := newTestService(t)
	docs := make([]Document, 0, 1000)
	for i := 0; i < 1000; i++ {
		docs = append(docs, testDoc(
			searchv1.SearchKind_SEARCH_KIND_ASSET,
			fmt.Sprintf("asset.%03d", i),
			fmt.Sprintf("Asset %03d", i),
			"Reusable component for landing page onboarding flow button form field validation navigation card grid.",
		))
	}
	seedDocuments(t, svc, docs...)

	queries := []string{"button", "landing page", "onboarding flow", "form field", "navigation card", "asset 1", "zzzz absent", `"validation error"`, "grid", "flow"}
	latencies := make([]time.Duration, 0, 50)
	for i := 0; i < 50; i++ {
		started := time.Now()
		response, err := svc.Search(context.Background(), &searchv1.SearchRequest{
			Query: queries[i%len(queries)],
			Limit: 20,
		})
		require.NoError(t, err)
		require.NotNil(t, response)
		latencies = append(latencies, time.Since(started))
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95 := latencies[(len(latencies)*95+99)/100-1]
	t.Logf("lexical p95 over 50 queries and 1,000 documents: %s", p95)
	if p95 > 100*time.Millisecond {
		t.Fatalf("lexical p95 was %s, above the 100ms budget", p95)
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
  summary TEXT NOT NULL DEFAULT '',
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
INSERT INTO workspaces (id, name, summary, provenance, tags_json, binding_scenario, binding_page, updated_at)
VALUES ('w1', 'Onboarding flow', 'A guided launch surface for new operators.', 'curated', '["onboarding","first-run"]', 'harbor', 'welcome', '2026-11-19T09:00:00Z');
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

	_, err = service.DB().ExecContext(ctx, `UPDATE search_state SET value = '1' WHERE key = ?`, stateProjectionVersion)
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
