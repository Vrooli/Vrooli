package preferences_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"

	preferencesH "notification-hub/handlers/preferences"
	"notification-hub/internal/preferences"
)

const testSubject = "operator-subject"

// newServer wires the preferences module with a configured recipient and no
// verified owner identity, mirroring production read scoping on a bare mux.
func newServer(t *testing.T) (*mux.Router, *preferences.Storage) {
	t.Helper()
	db := databasetest.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), preferences.Schema())
	require.NoError(t, err)
	store := preferences.New(db)
	r := mux.NewRouter()
	preferencesH.ModuleWithRecipientResolver(store, func(context.Context) string { return testSubject }, nil).Mount(r)
	return r, store
}

func doBodyWithoutIdentity(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func doWithoutIdentity(t *testing.T, r http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func do(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Buffer
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewBuffer(payload)
	} else {
		reader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("X-Vrooli-Identity-Subject", testSubject)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

type catalogResponse struct {
	Sources []struct {
		SourceScenario string `json:"source_scenario"`
	} `json:"sources"`
	Preferences []struct {
		SourceScenario string `json:"source_scenario"`
		Category       string `json:"category"`
		Enabled        bool   `json:"enabled"`
		MinUrgency     string `json:"min_urgency"`
	} `json:"preferences"`
}

func TestCatalog_EmptyThenAfterSet(t *testing.T) {
	r, _ := newServer(t)

	// Empty: no sources, no preferences for this recipient.
	rec := doWithoutIdentity(t, r, http.MethodGet, "/api/v1/notifications/catalog")
	require.Equal(t, http.StatusOK, rec.Code)
	var empty catalogResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &empty))
	require.Empty(t, empty.Sources)
	require.Empty(t, empty.Preferences)

	// After a PUT, the recipient's stored preference appears in the catalog.
	put := do(t, r, http.MethodPut, "/api/v1/notifications/preferences", map[string]any{
		"source_scenario": "infra-health",
		"category":        "incident",
		"enabled":         false,
		"min_urgency":     "high",
	})
	require.Equal(t, http.StatusOK, put.Code)

	rec = doWithoutIdentity(t, r, http.MethodGet, "/api/v1/notifications/catalog")
	require.Equal(t, http.StatusOK, rec.Code)
	var after catalogResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &after))
	require.Len(t, after.Preferences, 1)
	require.Equal(t, "infra-health", after.Preferences[0].SourceScenario)
	require.Equal(t, "incident", after.Preferences[0].Category)
	require.False(t, after.Preferences[0].Enabled)
	require.Equal(t, "high", after.Preferences[0].MinUrgency)
}

func TestPreferenceWriteStillRequiresIdentity(t *testing.T) {
	r, _ := newServer(t)
	rec := doBodyWithoutIdentity(t, r, http.MethodPut, "/api/v1/notifications/preferences", map[string]any{
		"source_scenario": "marketing",
		"enabled":         true,
	})
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPutPreference_HappyPath(t *testing.T) {
	r, _ := newServer(t)
	rec := do(t, r, http.MethodPut, "/api/v1/notifications/preferences", map[string]any{
		"source_scenario":  "marketing",
		"category":         "",
		"enabled":          true,
		"min_urgency":      "normal",
		"channel_override": "email",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var row struct {
		SourceScenario  string `json:"source_scenario"`
		ChannelOverride string `json:"channel_override"`
		Enabled         bool   `json:"enabled"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &row))
	require.Equal(t, "marketing", row.SourceScenario)
	require.Equal(t, "email", row.ChannelOverride)
	require.True(t, row.Enabled)

	// A source-wide ("") preference is listed back for the recipient.
	list := doWithoutIdentity(t, r, http.MethodGet, "/api/v1/notifications/preferences")
	require.Equal(t, http.StatusOK, list.Code)
	var listed struct {
		Preferences []struct {
			SourceScenario string `json:"source_scenario"`
			Category       string `json:"category"`
		} `json:"preferences"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &listed))
	require.Len(t, listed.Preferences, 1)
	require.Equal(t, "marketing", listed.Preferences[0].SourceScenario)
	require.Equal(t, "", listed.Preferences[0].Category)
}

func TestPutPreference_EmptySourceIsBadRequest(t *testing.T) {
	r, _ := newServer(t)
	rec := do(t, r, http.MethodPut, "/api/v1/notifications/preferences", map[string]any{
		"source_scenario": "",
		"enabled":         true,
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPutPreference_InvalidBodyIsBadRequest(t *testing.T) {
	r, _ := newServer(t)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewBufferString("{not json"))
	req.Header.Set("X-Vrooli-Identity-Subject", testSubject)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// seedTwoSources declares two producing scenarios, each with one category, and
// stores a preference row for each, so a source filter has something to cut.
func seedTwoSources(t *testing.T, r http.Handler, store *preferences.Storage) {
	t.Helper()
	ctx := context.Background()
	const now = "2026-10-28T00:00:00Z"
	for _, src := range []struct{ source, category string }{
		{"web-console", "session.input_needed"},
		{"git-control-tower", "review"},
	} {
		require.NoError(t, store.UpsertDeclaredSource(ctx, src.source, src.source, "", "", now))
		require.NoError(t, store.UpsertDeclaredCategory(ctx, src.source, src.category, src.category, "", true, "normal", now))
		put := do(t, r, http.MethodPut, "/api/v1/notifications/preferences", map[string]any{
			"source_scenario": src.source,
			"category":        src.category,
			"enabled":         false,
		})
		require.Equal(t, http.StatusOK, put.Code)
	}
}

func catalogSources(t *testing.T, r http.Handler, path string) (sources, prefSources []string) {
	t.Helper()
	rec := doWithoutIdentity(t, r, http.MethodGet, path)
	require.Equal(t, http.StatusOK, rec.Code)
	var body catalogResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	for _, s := range body.Sources {
		sources = append(sources, s.SourceScenario)
	}
	for _, p := range body.Preferences {
		prefSources = append(prefSources, p.SourceScenario)
	}
	return sources, prefSources
}

func TestCatalog_SourceFilterScopesToOneScenario(t *testing.T) {
	r, store := newServer(t)
	seedTwoSources(t, r, store)

	// Unfiltered: the hub's own Settings page sees every source.
	sources, prefs := catalogSources(t, r, "/api/v1/notifications/catalog")
	require.ElementsMatch(t, []string{"web-console", "git-control-tower"}, sources)
	require.ElementsMatch(t, []string{"web-console", "git-control-tower"}, prefs)

	// Filtered: a scenario's settings page sees only its own source and rows.
	sources, prefs = catalogSources(t, r, "/api/v1/notifications/catalog?source_scenario=web-console")
	require.Equal(t, []string{"web-console"}, sources)
	require.Equal(t, []string{"web-console"}, prefs)
}

func TestCatalog_SourceFilterAcceptsListsAndUnknownSources(t *testing.T) {
	r, store := newServer(t)
	seedTwoSources(t, r, store)

	sources, _ := catalogSources(t, r, "/api/v1/notifications/catalog?source_scenario=web-console,git-control-tower")
	require.ElementsMatch(t, []string{"web-console", "git-control-tower"}, sources)

	sources, _ = catalogSources(t, r, "/api/v1/notifications/catalog?source_scenario=web-console&source_scenario=git-control-tower")
	require.ElementsMatch(t, []string{"web-console", "git-control-tower"}, sources)

	// A scenario that has declared nothing yet gets an empty catalog, never
	// every other scenario's categories.
	sources, prefs := catalogSources(t, r, "/api/v1/notifications/catalog?source_scenario=not-registered")
	require.Empty(t, sources)
	require.Empty(t, prefs)
}

func TestPreferences_SourceFilter(t *testing.T) {
	r, store := newServer(t)
	seedTwoSources(t, r, store)

	rec := doWithoutIdentity(t, r, http.MethodGet, "/api/v1/notifications/preferences?source_scenario=git-control-tower")
	require.Equal(t, http.StatusOK, rec.Code)
	var listed struct {
		Preferences []struct {
			SourceScenario string `json:"source_scenario"`
		} `json:"preferences"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Preferences, 1)
	require.Equal(t, "git-control-tower", listed.Preferences[0].SourceScenario)
}
