package themes_test

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	themesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/themes/themes_v1connect"

	themesH "react-component-library/handlers/themes"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/themes"

	db "github.com/vrooli/api-core/databasetest"
)

type fakeDesignReader struct {
	byScenario map[string][]byte
}

type themesServiceStub struct {
	listErr, getErr, resolveErr error
}

func (s themesServiceStub) ListBuiltins(context.Context) ([]themes.Theme, error) {
	return []themes.Theme{{ID: "stub", Name: "Stub", Source: "builtin", Tokens: map[string]string{"--color-primary": "#123456"}}}, s.listErr
}

func (s themesServiceStub) GetBuiltin(context.Context, string) (themes.Theme, error) {
	return themes.Theme{}, s.getErr
}

func (s themesServiceStub) ResolveFromScenario(context.Context, string) (themes.Theme, error) {
	return themes.Theme{}, s.resolveErr
}

func (themesServiceStub) EnsureBuiltinsSeeded(context.Context) error { return nil }

func (f fakeDesignReader) Read(_ context.Context, scenario string) ([]byte, error) {
	return f.byScenario[scenario], nil
}

func setupModule(t *testing.T) (*mux.Router, themes.Service) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(themes.Schema),
	))
	designs := fakeDesignReader{byScenario: map[string][]byte{
		"target": []byte("---\nname: Target Theme\ncolors:\n  primary: \"#123456\"\n---\n"),
	}}
	svc := themesH.BuildService(d, designs)
	require.NoError(t, svc.EnsureBuiltinsSeeded(context.Background()))
	m := themesH.ModuleFromService(svc, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)
	return r, svc
}

func TestModule_Shape(t *testing.T) {
	r, _ := setupModule(t)
	require.NotNil(t, r)
	require.Len(t, themesH.Endpoints, 3)
}

func TestModule_ListAndGetBuiltin(t *testing.T) {
	r, _ := setupModule(t)

	rw := callConnect(r, themesconnect.ThemesServiceListBuiltinThemesProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"id":"light"`)

	rw = callConnect(r, themesconnect.ThemesServiceGetBuiltinThemeProcedure, `{"id":"light"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"name":"Light"`)
}

func TestModule_GetMissingBuiltinReturnsNotFound(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, themesconnect.ThemesServiceGetBuiltinThemeProcedure, `{"id":"missing"}`)
	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"not_found"`)
}

func TestModule_GetThemeFromScenarioProjectsTheme(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, themesconnect.ThemesServiceGetThemeFromScenarioProcedure, `{"scenario_id":"target"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"id":"scenario:target"`)
	require.Contains(t, rw.Body.String(), `"name":"Target Theme"`)
	require.Contains(t, rw.Body.String(), `"--color-primary":"#123456"`)
	require.Contains(t, rw.Body.String(), `"source":"scenario:target"`)
}

func TestModule_MapsServiceFailuresToConnectStatus(t *testing.T) {
	tests := []struct {
		name, path, body string
		service          themesServiceStub
		status           int
	}{
		{
			name: "list internal error", path: themesconnect.ThemesServiceListBuiltinThemesProcedure,
			body: `{}`, service: themesServiceStub{listErr: errors.New("database unavailable")}, status: http.StatusInternalServerError,
		},
		{
			name: "builtin missing", path: themesconnect.ThemesServiceGetBuiltinThemeProcedure,
			body: `{"id":"missing"}`, service: themesServiceStub{getErr: themes.ErrThemeNotFound{ID: "missing"}}, status: http.StatusNotFound,
		},
		{
			name: "invalid scenario theme", path: themesconnect.ThemesServiceGetThemeFromScenarioProcedure,
			body: `{"scenario_id":"target"}`, service: themesServiceStub{resolveErr: themes.ErrInvalidDesignMD{Scenario: "target", Reason: "no theme tokens"}}, status: http.StatusBadRequest,
		},
		{
			name: "missing scenario theme", path: themesconnect.ThemesServiceGetThemeFromScenarioProcedure,
			body: `{"scenario_id":"target"}`, service: themesServiceStub{resolveErr: themes.ErrScenarioDesignMDMissing{Scenario: "target", Cause: errors.New("not found")}}, status: http.StatusNotFound,
		},
		{
			name: "unexpected scenario error", path: themesconnect.ThemesServiceGetThemeFromScenarioProcedure,
			body: `{"scenario_id":"target"}`, service: themesServiceStub{resolveErr: errors.New("reader failed")}, status: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := themesH.ModuleFromService(tt.service, log.New(io.Discard, "", 0))
			r := mux.NewRouter()
			m.Mount(r)
			rw := callConnect(r, tt.path, tt.body)
			require.Equal(t, tt.status, rw.Code, rw.Body.String())
		})
	}
}

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}
