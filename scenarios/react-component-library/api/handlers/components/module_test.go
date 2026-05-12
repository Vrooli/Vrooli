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

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
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
	require.Len(t, components.Endpoints, 6, "components ships list, get, get-by-library-id, index, content get, content set")
}

func TestModule_ContentRoundTrip(t *testing.T) {
	r, root := setupModule(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "Button.tsx"), []byte(`/**
 * @libraryId react-component-library:Button
 * @displayName Button
 * @description Primary CTA.
 */
export const Button = () => null;
`), 0o600))

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

	written, err := os.ReadFile(filepath.Join(root, "Button.tsx"))
	require.NoError(t, err)
	require.Contains(t, string(written), "// rewritten")
}

func TestModule_IndexThenList(t *testing.T) {
	r, root := setupModule(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "Button.tsx"), []byte(`/**
 * @libraryId react-component-library:Button
 * @displayName Button
 * @description Primary CTA.
 * @version 1.0.0
 * @tags ["form"]
 */
export const Button = () => null;
`), 0o600))

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

func TestModule_GetReturnsNotFound(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, componentsconnect.ComponentsServiceGetComponentProcedure, `{"id":"ghost"}`)
	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"not_found"`)
}

// TestEndpoints_ParityWithProtoService — adding `rpc Foo(...)` to
// components.proto without a matching Endpoints entry would silently
// ship a .vrooli/endpoints.json that disagrees with the server.
func TestEndpoints_ParityWithProtoService(t *testing.T) {
	svc := componentsv1.File_react_component_library_v1_components_components_proto.
		Services().ByName("ComponentsService")
	require.NotNil(t, svc, "components proto must declare a ComponentsService service")

	byPath := make(map[string]int, len(components.Endpoints))
	for _, ep := range components.Endpoints {
		byPath[ep.Path]++
	}

	methods := svc.Methods()
	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		wantPath := "/" + string(svc.FullName()) + "/" + string(m.Name())
		count := byPath[wantPath]
		require.Equal(t, 1, count,
			"proto method %q (path %q) must have exactly one Endpoints entry; found %d",
			m.Name(), wantPath, count)
	}
}

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}
