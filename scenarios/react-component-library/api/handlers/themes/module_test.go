package themes_test

import (
	"context"
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

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}
