package preview_test

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

	previewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview/preview_v1connect"

	db "github.com/vrooli/api-core/databasetest"
	componentsH "react-component-library/handlers/components"
	previewH "react-component-library/handlers/preview"
	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"

	"github.com/vrooli/api-core/schedule"

	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
)

const buttonTSX = `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => <button>click</button>;
`

func setupModule(t *testing.T) (*mux.Router, string) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalcomponents.Schema),
	))

	root := t.TempDir()
	logger := log.New(io.Discard, "", 0)

	// Components module first — preview depends on its service surface
	// but the test wires its own shared service so both modules read
	// the same on-disk content.
	componentsModule := componentsH.ModuleWithRoot(d, schedule.System(), root, logger)
	svc, _ := componentsH.BuildService(d, schedule.System(), root)
	previewModule := previewH.Module(svc, logger)

	r := mux.NewRouter()
	componentsModule.Mount(r)
	previewModule.Mount(r)
	return r, root
}

func TestModule_Shape(t *testing.T) {
	r, _ := setupModule(t)
	require.NotNil(t, r)
	require.Len(t, previewH.Endpoints, 1, "preview ships GetPreviewBundle")
}

func TestModule_BundleRoundTrip(t *testing.T) {
	r, root := setupModule(t)

	writeButtonManifest(t, root, buttonTSX)

	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	id := extractFirstID(t, rw.Body.String())

	rw = callConnect(r, previewconnect.PreviewServiceGetPreviewBundleProcedure, `{"id":"`+id+`"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	body := rw.Body.String()
	require.Contains(t, body, `"js":`)
	require.Contains(t, body, `"sha256":`)
	require.Contains(t, body, `"sourcePath":"components/Button/versions/1.0.0/Button.tsx"`)
}

func TestModule_BundleNotFound(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, previewconnect.PreviewServiceGetPreviewBundleProcedure, `{"id":"ghost"}`)
	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
}

func TestModule_HarnessHTML(t *testing.T) {
	r, root := setupModule(t)
	writeButtonManifest(t, root, buttonTSX)
	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:Button"}`)
	id := extractFirstID(t, rw.Body.String())

	req, _ := http.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := rec.Body.String()
	require.Contains(t, body, `<!doctype html>`)
	require.Contains(t, body, `importmap`)
	require.Contains(t, body, `/preview/runtime/react-dom@18.3.1/client.js`)
	require.NotContains(t, body, `https://esm.sh/react`)
	require.Contains(t, body, `--color-primary`)
	require.Contains(t, body, `.bg-app-primary`)
	require.Contains(t, body, `data:text/javascript;base64,`)
	require.Contains(t, body, `preview-ready`)
	require.Contains(t, body, `id="root"`)
	// Cache-busting works because the inlined sha is unique per save.
	require.Contains(t, body, `name="bundle-sha256"`)
	// req TH-003: harness installs a theme-apply listener that sets
	// each token as a CSS custom property on :root.
	require.Contains(t, body, `rcl-theme-apply`)
	require.Contains(t, body, `documentElement.style.setProperty`)
	require.Contains(t, body, `rcl-theme-applied`)
}

func TestModule_HarnessUsesSelectedKitAndRejectsMissingCompiledUtilities(t *testing.T) {
	r, root := setupModule(t)
	writeButtonManifest(t, root, buttonTSX)
	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:Button"}`)
	id := extractFirstID(t, rw.Body.String())

	cases := []struct {
		kit    string
		radius string
	}{
		{kit: "vrooli-default", radius: "--radius-control: 0.375rem"},
		{kit: "vrooli-command-display", radius: "--radius-control: 10px"},
		{kit: "vrooli-conversion-landing", radius: "--radius-control: 9999px"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html?kit="+tc.kit, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, tc.kit)
		require.Contains(t, rec.Body.String(), tc.radius, tc.kit)
	}

	req := httptest.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html?kit=missing-kit", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestModule_HarnessAcceptsLibraryID(t *testing.T) {
	r, root := setupModule(t)
	writeButtonManifest(t, root, buttonTSX)
	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	req, _ := http.NewRequest(http.MethodGet, "/preview/react-component-library:Button/harness.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `components/Button/versions/1.0.0/Button.tsx`)
	require.Contains(t, rec.Body.String(), `name="component-id" content="react-component-library:Button"`)
}

func TestModule_RuntimeReactServesVendoredESM(t *testing.T) {
	r, _ := setupModule(t)
	req, _ := http.NewRequest(http.MethodGet, "/preview/runtime/react@18.3.1/index.js", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Header().Get("Content-Type"), "application/javascript")
	require.Contains(t, rec.Body.String(), "export")
	require.Contains(t, rec.Body.String(), "createElement")
	require.NotContains(t, rec.Body.String(), "https://esm.sh")
}

func TestModule_RuntimeReactJSXRuntimeServesNamedExports(t *testing.T) {
	r, _ := setupModule(t)
	req, _ := http.NewRequest(http.MethodGet, "/preview/runtime/react@18.3.1/jsx-runtime.js", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := rec.Body.String()
	require.Contains(t, body, "export {")
	require.Contains(t, body, "  jsx,")
	require.Contains(t, body, "  jsxs")
	require.Contains(t, body, "  Fragment,")
	require.NotContains(t, body, "https://esm.sh")
}

func TestModule_RuntimeReactDOMClientServesNamedExports(t *testing.T) {
	r, _ := setupModule(t)
	req, _ := http.NewRequest(http.MethodGet, "/preview/runtime/react-dom@18.3.1/client.js", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := rec.Body.String()
	require.Contains(t, body, "export {")
	require.Contains(t, body, "  createRoot,")
	require.Contains(t, body, "  hydrateRoot")
	require.Contains(t, body, `if (name === "react")`)
	require.NotContains(t, body, "https://esm.sh")
}

func TestModule_HarnessNotFound(t *testing.T) {
	r, _ := setupModule(t)
	req, _ := http.NewRequest(http.MethodGet, "/preview/ghost/harness.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestModule_HarnessBundleError(t *testing.T) {
	r, root := setupModule(t)

	// Component file passes the header gate but is syntactically broken,
	// so the indexer succeeds and esbuild later rejects it.
	writeComponentManifest(t, root, "Broken", "react-component-library:Broken", `/**
 * @libraryId react-component-library:Broken
 * @version 1.0.0
 */
export const Broken = () => <div
`)
	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:Broken"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	id := extractFirstID(t, rw.Body.String())

	req, _ := http.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code, "bundler errors must not report preview readiness")
	require.Contains(t, rec.Body.String(), "bundle failed")
}

// Proto/Connect parity for the preview domain is now enforced globally
// by TestProtoConnectParity in api/internal/modules/registry_test.go.

func extractFirstID(t *testing.T, body string) string {
	t.Helper()
	idStart := strings.Index(body, `"id":"`)
	require.NotEqual(t, -1, idStart, "no id in body %q", body)
	idStart += len(`"id":"`)
	idEnd := strings.Index(body[idStart:], `"`)
	require.NotEqual(t, -1, idEnd)
	return body[idStart : idStart+idEnd]
}

func writeButtonManifest(t *testing.T, root, source string) {
	t.Helper()
	writeComponentManifest(t, root, "Button", "react-component-library:Button", source)
}

func writeComponentManifest(t *testing.T, root, slug, libraryID, source string) {
	t.Helper()
	dir := filepath.Join(root, "components", slug)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.json"), []byte(`{
  "libraryId": "`+libraryID+`",
  "displayName": "`+slug+`",
  "description": "",
  "tags": [],
  "latest": "1.0.0",
  "deprecatedVersions": []
}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "versions", "1.0.0", slug+".tsx"), []byte(source), 0o600))
}

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}
