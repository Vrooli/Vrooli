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

	componentsH "react-component-library/handlers/components"
	previewH "react-component-library/handlers/preview"
	"react-component-library/internal/clock"
	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/testutil/db"

	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
)

const buttonTSX = `/**
 * @libraryId react-component-library:Button
 * @displayName Button
 * @description Primary CTA.
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
	componentsModule := componentsH.ModuleWithRoot(d, clock.System{}, root, logger)
	svc, _ := componentsH.BuildService(d, clock.System{}, root)
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

	require.NoError(t, os.WriteFile(filepath.Join(root, "Button.tsx"), []byte(buttonTSX), 0o600))

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
	require.Contains(t, body, `"sourcePath":"Button.tsx"`)
}

func TestModule_BundleNotFound(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, previewconnect.PreviewServiceGetPreviewBundleProcedure, `{"id":"ghost"}`)
	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
}

func TestModule_HarnessHTML(t *testing.T) {
	r, root := setupModule(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "Button.tsx"), []byte(buttonTSX), 0o600))
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
	require.Contains(t, body, `react-dom@18.3.1/client`)
	require.Contains(t, body, `data:text/javascript;base64,`)
	require.Contains(t, body, `preview-ready`)
	require.Contains(t, body, `id="root"`)
	// Cache-busting works because the inlined sha is unique per save.
	require.Contains(t, body, `name="bundle-sha256"`)
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
	require.NoError(t, os.WriteFile(filepath.Join(root, "Broken.tsx"), []byte(`/**
 * @libraryId react-component-library:Broken
 */
export const Broken = () => <div
`), 0o600))
	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:Broken"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	id := extractFirstID(t, rw.Body.String())

	req, _ := http.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "bundler errors render in-iframe as 200 HTML")
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

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}
