package preferences_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testkit "github.com/vrooli/repo-contract-go/repocontracttest"

	"notification-hub/internal/preferences"
)

// newRepo builds a temporary repo with a valid repo-contract and the named
// scenario stubs (each gets a `.vrooli/service.json`). It returns the repo root.
func newRepo(t *testing.T, scenarios ...string) string {
	t.Helper()
	fixture := testkit.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	for _, name := range scenarios {
		fixture.WriteScenarioStub(t, name)
	}
	return fixture.Root
}

// writeNotificationsManifest writes a `.vrooli/notifications.json` under a
// scenario in the fixture repo.
func writeNotificationsManifest(t *testing.T, repoRoot, scenario, raw string) {
	t.Helper()
	testkit.WriteRawJSON(t, filepath.Join(repoRoot, "scenarios", scenario, ".vrooli", "notifications.json"), raw, 0o644)
}

// writeServiceDisplayName overwrites a scenario stub's service.json with a
// displayName so branding fallbacks are exercised.
func writeServiceDisplayName(t *testing.T, repoRoot, scenario, displayName string) {
	t.Helper()
	testkit.WriteJSON(t, filepath.Join(repoRoot, "scenarios", scenario, ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{"name": scenario, "displayName": displayName},
	})
}

func TestLoadManifestIsLenient(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notifications.json")
	testkit.WriteRawJSON(t, path, `{
		"displayName": "Agent Manager",
		"icon": "agent-manager",
		"deepLinkBase": "https://host/runs",
		"unknownField": 123,
		"categories": [
			{"key": "run-failed", "displayName": "Run failed", "description": "A run ended in error.", "defaultEnabled": true, "defaultMinUrgency": "high"},
			{"key": "   ", "displayName": "dropped"},
			{"displayName": "no key, dropped"}
		]
	}`, 0o644)

	m, err := preferences.LoadManifest(path)
	require.NoError(t, err)
	require.Equal(t, "Agent Manager", m.DisplayName)
	require.Equal(t, "agent-manager", m.Icon)
	require.Equal(t, "https://host/runs", m.DeepLinkBase)
	require.Len(t, m.Categories, 1, "empty-key categories are dropped")
	require.Equal(t, "run-failed", m.Categories[0].Key)
	require.True(t, m.Categories[0].DefaultEnabled)
	require.Equal(t, "high", m.Categories[0].DefaultMinUrgency)
}

func TestIngestDeclaredManifestsOnlyIngestsDeclaringScenarios(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	// "agent-manager" declares a manifest; "silent-scenario" does not.
	repoRoot := newRepo(t, "agent-manager", "silent-scenario")
	writeNotificationsManifest(t, repoRoot, "agent-manager", `{
		"displayName": "Agent Manager",
		"icon": "am-icon",
		"deepLinkBase": "https://host/runs",
		"categories": [
			{"key": "run-failed", "displayName": "Run failed", "defaultEnabled": true, "defaultMinUrgency": "high"}
		]
	}`)

	require.NoError(t, store.IngestDeclaredManifests(ctx, repoRoot, now()))

	sources, err := store.ListCatalogForRepo(ctx, repoRoot)
	require.NoError(t, err)
	require.Len(t, sources, 1, "only the declaring scenario appears")
	src := sources[0]
	require.Equal(t, "agent-manager", src.SourceScenario)
	require.True(t, src.Declared)
	require.Equal(t, "Agent Manager", src.DisplayName)
	require.Equal(t, "am-icon", src.Icon)
	require.Equal(t, "https://host/runs", src.DeepLinkBase)
	require.Len(t, src.Categories, 1)
	require.Equal(t, "run-failed", src.Categories[0].Category)
	require.True(t, src.Categories[0].Declared)
	require.True(t, src.Categories[0].DefaultEnabled)
	require.Equal(t, "high", src.Categories[0].DefaultMinUrgency)
}

func TestIngestDeclaredManifestsFallsBackToServiceDisplayName(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	repoRoot := newRepo(t, "planner")
	writeServiceDisplayName(t, repoRoot, "planner", "Personal Planner")
	// Manifest omits displayName and icon; both fall back.
	writeNotificationsManifest(t, repoRoot, "planner", `{
		"categories": [{"key": "reminder"}]
	}`)

	require.NoError(t, store.IngestDeclaredManifests(ctx, repoRoot, now()))

	sources, err := store.ListCatalogForRepo(ctx, repoRoot)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	require.Equal(t, "Personal Planner", sources[0].DisplayName, "displayName falls back to service.json")
	require.Equal(t, "planner", sources[0].Icon, "icon falls back to the slug")
}

func TestIngestDeclaredManifestsMissingRepoRootIsNonFatal(t *testing.T) {
	store := newStore(t)
	require.NoError(t, store.IngestDeclaredManifests(context.Background(), t.TempDir(), now()))
}

func TestListCatalogFiltersBogusObservedSources(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	repoRoot := newRepo(t, "agent-manager", "planner")
	// Declared source (not observed in events).
	require.NoError(t, store.UpsertDeclaredSource(ctx, "agent-manager", "Agent Manager", "am", "", now()))
	// Observed real scenario -> branding inferred.
	require.NoError(t, store.ObserveCatalog(ctx, "planner", "reminder", now()))
	store.InferObservedBranding(ctx, repoRoot, "planner", now())
	// Observed bogus source (a manual probe) -> must be hidden.
	require.NoError(t, store.ObserveCatalog(ctx, "smoketest", "probe", now()))

	sources, err := store.ListCatalogForRepo(ctx, repoRoot)
	require.NoError(t, err)

	byName := map[string]preferences.Source{}
	for _, s := range sources {
		byName[s.SourceScenario] = s
	}
	require.Contains(t, byName, "agent-manager", "declared source is always shown")
	require.Contains(t, byName, "planner", "observed real-scenario source is shown")
	require.NotContains(t, byName, "smoketest", "observed bogus source is hidden")

	require.True(t, byName["agent-manager"].Declared)
	require.False(t, byName["planner"].Declared)
}

func TestInferObservedBrandingNeverClobbersDeclared(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	repoRoot := newRepo(t, "planner")
	writeServiceDisplayName(t, repoRoot, "planner", "Service Name")
	require.NoError(t, store.UpsertDeclaredSource(ctx, "planner", "Declared Name", "declared-icon", "", now()))

	// Inference must not overwrite declared branding.
	store.InferObservedBranding(ctx, repoRoot, "planner", now())

	name, icon, _, ok := store.ResolveBrand(ctx, "planner")
	require.True(t, ok)
	require.Equal(t, "Declared Name", name)
	require.Equal(t, "declared-icon", icon)
}

func TestObserveCatalogKeepsEmptyCategorySource(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	repoRoot := newRepo(t, "planner")
	require.NoError(t, store.ObserveCatalog(ctx, "planner", "", now()))

	sources, err := store.ListCatalogForRepo(ctx, repoRoot)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	require.Equal(t, "planner", sources[0].SourceScenario)
	require.Empty(t, sources[0].Categories, "a source observed with an empty category has no category rows")
}
