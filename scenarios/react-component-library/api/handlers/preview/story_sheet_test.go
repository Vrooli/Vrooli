package preview_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
)

func writeButtonStoryContract(t *testing.T, root string) {
	t.Helper()
	versionDir := filepath.Join(root, "components", "Button", "versions", "1.0.0")
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "story.tsx"), []byte(`export function Default() { return <button>Ready</button>; }
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "story.json"), []byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [
    {"id": "anatomy", "name": "Anatomy", "role": "anatomy", "mode": "live", "args": {}, "composition": {"specimen": {"module": "./story.tsx", "export": "Default"}}},
    {"id": "loading", "name": "Loading", "role": "axis", "axis": "loading", "mode": "live", "args": {}, "composition": {"specimen": {"module": "./story.tsx", "export": "Default"}}}
  ]
}`), 0o600))
}

func TestHarnessStorySheetFiltersRolesAndExpandsViewports(t *testing.T) {
	router, root := setupModule(t)
	writeButtonManifest(t, root, buttonTSX)
	writeButtonStoryContract(t, root)
	indexed := callConnect(router, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, indexed.Code, indexed.Body.String())
	lookup := callConnect(router, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure, `{"libraryId":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, lookup.Code, lookup.Body.String())
	id := extractFirstID(t, lookup.Body.String())

	request := httptest.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html?role=anatomy&viewport=desktop,mobile&theme=dark", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	body := response.Body.String()
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Contains(t, body, "Button stories")
	require.Contains(t, body, `data-rcl-theme="dark"`)
	require.Contains(t, body, `data-story-id="anatomy:desktop"`)
	require.Contains(t, body, `data-story-id="anatomy:mobile"`)
	require.NotContains(t, body, `data-story-id="loading"`)
	require.Contains(t, body, `story=anatomy`)
	require.Contains(t, body, `version=1.0.0`)
}

func TestHarnessStorySheetAxisIncludesAnatomy(t *testing.T) {
	router, root := setupModule(t)
	writeButtonManifest(t, root, buttonTSX)
	writeButtonStoryContract(t, root)
	indexed := callConnect(router, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, indexed.Code, indexed.Body.String())
	lookup := callConnect(router, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure, `{"libraryId":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, lookup.Code, lookup.Body.String())
	id := extractFirstID(t, lookup.Body.String())

	request := httptest.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html?axis=loading", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `data-story-id="anatomy"`)
	require.Contains(t, response.Body.String(), `data-story-id="loading"`)

	request = httptest.NewRequest(http.MethodGet, "/preview/"+id+"/harness.html?axis=missing", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), `unknown axis "missing"`)
}
