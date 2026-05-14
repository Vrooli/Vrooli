package components_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"

	"react-component-library/handlers/components"
	"react-component-library/internal/clock"
	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/testutil/db"
)

func setupModule(t *testing.T) (*mux.Router, string) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalcomponents.Schema),
	))

	root := t.TempDir()
	m := components.ModuleWithRoot(d, clock.System{}, root, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)
	return r, root
}

func TestModule_Shape(t *testing.T) {
	r, _ := setupModule(t)
	require.NotNil(t, r)
	require.Len(t, components.Endpoints, 11, "components ships registry, source authoring, content, and version endpoints")
}

func TestModule_InitializeComponentRoundTrip(t *testing.T) {
	r, root := setupModule(t)

	rw := callConnect(r, componentsconnect.ComponentsServiceInitializeComponentProcedure, `{
		"slug":"Header",
		"libraryId":"react-component-library:Header",
		"displayName":"Header",
		"description":"Scenario header",
		"tags":["layout"],
		"initialVersion":"0.1.0"
	}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `react-component-library:Header`)
	require.Contains(t, rw.Body.String(), `components/Header/component.json`)
	require.FileExists(t, filepath.Join(root, "components", "Header", "component.json"))
	require.FileExists(t, filepath.Join(root, "components", "Header", "versions", "0.1.0", "Header.tsx"))

	rw = callConnect(r, componentsconnect.ComponentsServiceListComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `react-component-library:Header`)
}

func TestModule_ContentRoundTrip(t *testing.T) {
	r, root := setupModule(t)

	writeButtonManifest(t, root, `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => null;
`)

	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	body := rw.Body.String()
	idStart := strings.Index(body, `"id":"`) + len(`"id":"`)
	idEnd := strings.Index(body[idStart:], `"`)
	id := body[idStart : idStart+idEnd]
	require.NotEmpty(t, id)

	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentContentProcedure,
		`{"id":"`+id+`"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), "@libraryId")
	require.Contains(t, rw.Body.String(), `"sha256"`)

	rw = callConnect(r, componentsconnect.ComponentsServiceUpdateComponentContentProcedure,
		`{"id":"`+id+`","content":"// rewritten\nexport const Button = () => null;\n"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"sha256"`)

	written, err := os.ReadFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "Button.tsx"))
	require.NoError(t, err)
	require.Contains(t, string(written), "// rewritten")
}

func TestModule_IndexThenList(t *testing.T) {
	r, root := setupModule(t)

	writeButtonManifest(t, root, `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 * @tags ["form"]
 */
export const Button = () => null;
`)

	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `react-component-library:Button`)

	rw = callConnect(r, componentsconnect.ComponentsServiceListComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `react-component-library:Button`)

	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"displayName":"Button"`)
}

func writeButtonManifest(t *testing.T, root, source string) {
	t.Helper()
	dir := filepath.Join(root, "components", "Button")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.json"), []byte(`{
  "libraryId": "react-component-library:Button",
  "displayName": "Button",
  "description": "Primary CTA.",
  "tags": ["form"],
  "latest": "1.0.0",
  "deprecatedVersions": []
}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "versions", "1.0.0", "Button.tsx"), []byte(source), 0o600))
}

func TestModule_GetReturnsNotFound(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, componentsconnect.ComponentsServiceGetComponentProcedure, `{"id":"ghost"}`)
	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"not_found"`)
}

// Proto/Connect parity for the components domain is now enforced
// globally by TestProtoConnectParity in
// api/internal/modules/registry_test.go.

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}
