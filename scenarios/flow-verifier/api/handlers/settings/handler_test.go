package settings_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	handlers "flow-verifier/handlers/settings"
	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/server"
	"flow-verifier/internal/settings"
	"flow-verifier/internal/testutil/assertx"
	"flow-verifier/internal/testutil/db"
	"flow-verifier/internal/testutil/httpx"
	"flow-verifier/internal/testutil/mocks"

	_ "modernc.org/sqlite"

	apidb "github.com/vrooli/api-core/database"

	"github.com/stretchr/testify/require"
)

// newSettingsLive wires the settings module behind a real httptest
// server with an empty in-memory SQLite database and a fake clock.
func newSettingsLive(t *testing.T) (*httpx.LiveServer, *settings.Service, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(settings.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC))
	svc := settings.NewService(settings.NewSQLiteRepository(d, clk))
	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
		handlers.ModuleWithService(svc),
	)
	return httpx.NewLiveServer(t, srv), svc, clk
}

func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

// TestModule_Shape pins the public contract of the module.
func TestModule_Shape(t *testing.T) {
	live, svc, _ := newSettingsLive(t)
	_ = live
	mod := handlers.ModuleWithService(svc)
	require.Equal(t, "settings", mod.Name)
	require.NotNil(t, mod.Mount)
	require.Len(t, mod.Endpoints, 2, "GET and PUT both descripted")
}

// TestGet_DefaultsWhenEmpty: a freshly migrated DB returns the
// hard-coded defaults rather than 404.
func TestGet_DefaultsWhenEmpty(t *testing.T) {
	live, _, _ := newSettingsLive(t)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/settings", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustDecodeJSON[settings.Settings](t, body)
	require.Equal(t, settings.PrincipalLocal, got.PrincipalID)
	require.Equal(t, settings.ThemeSystem, got.Theme)
	require.Equal(t, settings.FontScaleMd, got.FontScale)
	require.False(t, got.ReducedMotion)
	require.Equal(t, ".", got.DefaultRoot)
	require.Equal(t, settings.DensityComfortable, got.Density)
	require.Equal(t, 320, got.SidebarWidth)
	require.Equal(t, "flowId", got.InventoryFilters.Sort.Key)
}

// TestPut_PartialMerge: PUT body with only theme leaves other fields at
// their defaults, returns 200, and a follow-up GET reflects the change.
func TestPut_PartialMerge(t *testing.T) {
	live, _, _ := newSettingsLive(t)
	dark := settings.ThemeDark
	patch := settings.Patch{Theme: &dark}

	resp, body := live.Do(t, http.MethodPut, "/api/v1/settings", jsonBody(t, patch))
	assertx.AssertStatus(t, resp, http.StatusOK)
	got := assertx.MustDecodeJSON[settings.Settings](t, body)
	require.Equal(t, settings.ThemeDark, got.Theme)
	require.Equal(t, settings.FontScaleMd, got.FontScale, "fontScale unchanged")
	require.Equal(t, 320, got.SidebarWidth, "sidebarWidth unchanged")

	resp, body = live.Do(t, http.MethodGet, "/api/v1/settings", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)
	got = assertx.MustDecodeJSON[settings.Settings](t, body)
	require.Equal(t, settings.ThemeDark, got.Theme, "GET after PUT reflects the change")
}

// TestPut_InvalidEnum400: unknown enum value returns 400 + invalid_request.
func TestPut_InvalidEnum400(t *testing.T) {
	live, _, _ := newSettingsLive(t)
	// raw body — can't use the typed Patch here because Theme is a
	// string-typed enum and we want to send a bogus value as if from a
	// misbehaving client.
	body := bytes.NewReader([]byte(`{"theme":"hot-pink"}`))
	resp, _ := live.Do(t, http.MethodPut, "/api/v1/settings", body)
	assertx.AssertStatus(t, resp, http.StatusBadRequest)
}

// TestPut_MalformedJSON400: junk body returns 400.
func TestPut_MalformedJSON400(t *testing.T) {
	live, _, _ := newSettingsLive(t)
	body := bytes.NewReader([]byte(`{not-json`))
	resp, _ := live.Do(t, http.MethodPut, "/api/v1/settings", body)
	assertx.AssertStatus(t, resp, http.StatusBadRequest)
}

// TestPut_UnknownFields400: extra fields are rejected (DisallowUnknownFields).
func TestPut_UnknownFields400(t *testing.T) {
	live, _, _ := newSettingsLive(t)
	body := bytes.NewReader([]byte(`{"unknownKey":"x"}`))
	resp, _ := live.Do(t, http.MethodPut, "/api/v1/settings", body)
	assertx.AssertStatus(t, resp, http.StatusBadRequest)
}

// TestPersistenceAcrossDBReopen: writes via the production driver path,
// closes the DB, reopens against the same on-disk file, and confirms
// the row is intact. This is the canary the plan calls out for
// "settings survive scenario restart".
func TestPersistenceAcrossDBReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.db")
	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_txlock=immediate"

	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC))

	// First open: write a setting.
	d1, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	d1.SetMaxOpenConns(1)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d1,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(settings.Schema),
	))
	svc1 := settings.NewService(settings.NewSQLiteRepository(d1, clk))
	dark := settings.ThemeDark
	width := 400
	_, err = svc1.Upsert(context.Background(), settings.Patch{Theme: &dark, SidebarWidth: &width})
	require.NoError(t, err)
	require.NoError(t, d1.Close())

	// Reopen: read back through a fresh handle/service.
	d2, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	d2.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = d2.Close() })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d2,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(settings.Schema),
	))
	svc2 := settings.NewService(settings.NewSQLiteRepository(d2, clk))
	got, err := svc2.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, settings.ThemeDark, got.Theme)
	require.Equal(t, 400, got.SidebarWidth)
}
