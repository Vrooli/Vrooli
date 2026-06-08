package providers_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	"search-hub/internal/providers"
)

// TestMapResultsCLIHealthFixture is the Phase 3 proof that the generic mapping
// path turns a real cli-health SearchService.Search response into unified
// SearchHits using ONLY the cli-health.commands descriptor's ResultMapping — no
// provider-specific code. This is the no-conditional-monolith invariant made
// concrete: the same MapResults handles every provider; the descriptor differs.
func TestMapResultsCLIHealthFixture(t *testing.T) {
	body := readFixture(t, "cli_health_search.json")

	hits, err := providers.MapResults(cliHealthCommandsDescriptor(), body)
	require.NoError(t, err)
	require.Len(t, hits, 3)

	first := hits[0]
	require.Equal(t, "cli-health.commands", first.ProviderId)
	require.Equal(t, "cli-health", first.ProviderGroup)
	require.Equal(t, "command", first.Type)
	require.Equal(t, "scenario restart", first.Id)
	require.Equal(t, "scenario restart", first.Title)
	require.Equal(t, "Restart a scenario's API and UI processes through the lifecycle system.", first.Snippet)
	require.Equal(t, "scenario restart", first.Path)
	require.InDelta(t, 0.91, first.Score, 1e-9)
	require.Zero(t, first.RerankScore, "pre-rerank: rerank_score stays zero until Phase 6")

	// Order is preserved from the provider response (no re-ranking yet).
	require.Equal(t, "scenario logs", hits[1].Id)
	require.Equal(t, "test-genie execute", hits[2].Id)
	require.InDelta(t, 0.42, hits[2].Score, 1e-9)
}

func TestMapResultsScoreNormalization(t *testing.T) {
	cases := []struct {
		name  string
		scale registryv1.ScoreScale
		raw   float64
		want  float64
	}{
		{"cosine passthrough", registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1, 0.5, 0.5},
		{"cosine clamps high", registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1, 1.4, 1.0},
		{"percent divides", registryv1.ScoreScale_SCORE_SCALE_PERCENT_0_100, 80, 0.8},
		{"percent clamps", registryv1.ScoreScale_SCORE_SCALE_PERCENT_0_100, 140, 1.0},
		{"raw passthrough", registryv1.ScoreScale_SCORE_SCALE_RAW, 12.5, 12.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := mappingDescriptor(tc.scale, "", "")
			body := singleResult(t, tc.raw)
			hits, err := providers.MapResults(d, body)
			require.NoError(t, err)
			require.Len(t, hits, 1)
			require.InDelta(t, tc.want, hits[0].Score, 1e-9)
		})
	}
}

func TestMapResultsFilterField(t *testing.T) {
	// One endpoint, two leaves: keep only kind == "surface".
	d := mappingDescriptor(registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1, "kind", "surface")
	body := []byte(`{"results":[
		{"id":"a","title":"Login page","score":0.8,"kind":"surface"},
		{"id":"b","title":"Login widget","score":0.7,"kind":"widget"},
		{"id":"c","title":"Home page","score":0.6,"kind":"surface"}
	]}`)

	hits, err := providers.MapResults(d, body)
	require.NoError(t, err)
	require.Len(t, hits, 2)
	require.Equal(t, "a", hits[0].Id)
	require.Equal(t, "c", hits[1].Id)
}

func TestMapResultsPresenceField(t *testing.T) {
	// ui-health-style: one endpoint returns all surfaces; the widgets leaf keeps
	// only results where the `widget` object is populated (presence, not value —
	// there is no "widget" kind to equality-filter on).
	d := &registryv1.ProviderDescriptor{
		ProviderId:    "ui-health.widgets",
		ProviderGroup: "ui-health",
		Type:          "widget",
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath:   "results",
			IdField:       "filePath",
			TitleField:    "displayName",
			ScoreField:    "score",
			PathField:     "filePath",
			ScoreScale:    registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
			PresenceField: "widget",
		},
	}
	body := []byte(`{"results":[
		{"displayName":"Plain page","filePath":"a.tsx","score":0.8},
		{"displayName":"Chat widget","filePath":"b.tsx","score":0.7,"widget":{"slug":"chat","entry":"Chat"}},
		{"displayName":"Empty widget obj","filePath":"c.tsx","score":0.6,"widget":{}},
		{"displayName":"Null widget","filePath":"d.tsx","score":0.5,"widget":null}
	]}`)

	hits, err := providers.MapResults(d, body)
	require.NoError(t, err)
	require.Len(t, hits, 1, "only the populated-widget surface is kept (empty/null/absent are dropped)")
	require.Equal(t, "b.tsx", hits[0].Id)
	require.Equal(t, "Chat widget", hits[0].Title)
}

func TestMapResultsNestedPaths(t *testing.T) {
	// swarm-manager-style payload: title nested under payload.title.
	d := &registryv1.ProviderDescriptor{
		ProviderId:    "swarm-manager.records",
		ProviderGroup: "swarm-manager",
		Type:          "record",
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results",
			IdField:     "id",
			TitleField:  "payload.title",
			ScoreField:  "score",
			PathField:   "id",
			ScoreScale:  registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
	}
	body := []byte(`{"results":[{"id":"rec-1","score":0.66,"payload":{"title":"How to restart"}}]}`)

	hits, err := providers.MapResults(d, body)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "rec-1", hits[0].Id)
	require.Equal(t, "How to restart", hits[0].Title)
	require.Equal(t, "rec-1", hits[0].Path)
}

func TestMapResultsErrors(t *testing.T) {
	d := mappingDescriptor(registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1, "", "")

	t.Run("malformed json", func(t *testing.T) {
		_, err := providers.MapResults(d, []byte("{not json"))
		require.Error(t, err)
	})

	t.Run("results_path not an array", func(t *testing.T) {
		_, err := providers.MapResults(d, []byte(`{"results":{"oops":true}}`))
		require.Error(t, err)
	})

	t.Run("missing results yields empty, not error", func(t *testing.T) {
		hits, err := providers.MapResults(d, []byte(`{"results":[]}`))
		require.NoError(t, err)
		require.Empty(t, hits)
	})

	t.Run("absent results_path yields empty, not error (no-match response)", func(t *testing.T) {
		// A provider (e.g. ui-health) that omits the results key entirely on a
		// zero-result query must map to an honest empty group, never a degraded
		// mapping error. Regression guard for the F.5/G.4/I.4 adapter bug.
		hits, err := providers.MapResults(d, []byte(`{"total":0}`))
		require.NoError(t, err)
		require.Empty(t, hits)
	})

	t.Run("null results_path yields empty, not error", func(t *testing.T) {
		hits, err := providers.MapResults(d, []byte(`{"results":null}`))
		require.NoError(t, err)
		require.Empty(t, hits)
	})

	t.Run("numeric id coerced to string", func(t *testing.T) {
		hits, err := providers.MapResults(d, []byte(`{"results":[{"id":42,"title":"t","score":0.5}]}`))
		require.NoError(t, err)
		require.Len(t, hits, 1)
		require.Equal(t, "42", hits[0].Id)
	})
}

// cliHealthCommandsDescriptor builds the cli-health.commands descriptor the
// fixture test maps against. It mirrors the result_mapping in cli-health's
// `.vrooli/search.json` (id/title/path = "name", snippet = "description"). It is
// inlined here because search-hub no longer ships any provider descriptor — the
// generic adapter is exercised against a representative descriptor, and the real
// one now lives in cli-health's own SSOT and is self-registered at boot.
func cliHealthCommandsDescriptor() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "cli-health.commands",
		ProviderGroup: "cli-health",
		Bucket:        registryv1.Bucket_BUCKET_DO,
		Type:          "command",
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath:  "results",
			IdField:      "name",
			TitleField:   "name",
			ScoreField:   "score",
			SnippetField: "description",
			PathField:    "name",
			ScoreScale:   registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
	}
}

// mappingDescriptor builds a minimal descriptor whose result items live at
// results[] with flat id/title/score fields, plus an optional filter.
func mappingDescriptor(scale registryv1.ScoreScale, filterField, filterValue string) *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "fixture.leaf",
		ProviderGroup: "fixture",
		Type:          "thing",
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results",
			IdField:     "id",
			TitleField:  "title",
			ScoreField:  "score",
			PathField:   "id",
			ScoreScale:  scale,
			FilterField: filterField,
			FilterValue: filterValue,
		},
	}
}

func singleResult(t *testing.T, score float64) []byte {
	t.Helper()
	return []byte(`{"results":[{"id":"x","title":"X","score":` +
		strconv.FormatFloat(score, 'f', -1, 64) + `}]}`)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return b
}
