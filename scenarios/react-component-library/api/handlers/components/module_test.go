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
	previewhandlers "react-component-library/handlers/preview"
	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/experience"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
)

type fakeExperienceReader struct {
	snapshot  experience.Snapshot
	component experience.Component
}

func (f *fakeExperienceReader) Get(_ context.Context, component experience.Component) (experience.Snapshot, error) {
	f.component = component
	return f.snapshot, nil
}

func setupModule(t *testing.T, opts ...components.ModuleOption) (*mux.Router, string) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalcomponents.Schema),
	))

	root := t.TempDir()
	svc, repo := components.BuildService(d, schedule.System(), root)
	internalcomponents.SetServiceJSONReader(svc, internalcomponents.NewFSServiceJSONReader(filepath.Dir(root)))
	opts = append(opts, components.WithPreviewService(previewhandlers.BuildServiceAtRoot(svc, nil, root)))
	m := components.ModuleFromService(svc, repo, root, log.New(io.Discard, "", 0), opts...)
	r := mux.NewRouter()
	m.Mount(r)
	return r, root
}

func TestModule_AuthoringWorkflowUsesLibraryIDAndPublishesOnlyAfterCheck(t *testing.T) {
	r, root := setupModule(t)

	rw := callConnect(r, componentsconnect.ComponentsServiceInitializeComponentProcedure, `{
		"slug":"Button",
		"libraryId":"react-component-library:Button",
		"displayName":"Button",
		"initialVersion":"1.0.0",
		"initialSource":"export function Button() { return <button>Save</button>; }"
	}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = callConnect(r, componentsconnect.ComponentsServiceBeginComponentVersionProcedure,
		`{"component":"react-component-library:Button","bump":"minor"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.1.0-draft.1"`)
	require.Contains(t, rw.Body.String(), `"previewPath":"/preview/react-component-library:Button/harness.html?version=1.1.0-draft.1"`)

	rw = callConnect(r, componentsconnect.ComponentsServiceCheckComponentVersionProcedure,
		`{"component":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"passed":true`)
	require.Contains(t, rw.Body.String(), `"stage":"preview"`)
	require.Contains(t, rw.Body.String(), `"verdict":"passed"`)

	rw = callConnect(r, componentsconnect.ComponentsServicePublishComponentVersionProcedure,
		`{"component":"react-component-library:Button","changelogMd":"Focused authoring workflow"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.1.0"`)
	require.FileExists(t, filepath.Join(root, "components", "Button", "versions", "1.1.0", "story.json"))
	manifest, err := os.ReadFile(filepath.Join(root, "components", "Button", "component.json"))
	require.NoError(t, err)
	require.Contains(t, string(manifest), `"latest": "1.1.0"`)
	require.NotContains(t, string(manifest), `1.1.0-draft.1`)
}

func TestModule_Shape(t *testing.T) {
	r, _ := setupModule(t)
	require.NotNil(t, r)
	require.Len(t, components.Endpoints, 18, "components ships registry, authoring, ingest, style fit, styles, content, versions, and story endpoints")
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

func TestModule_InitializeComponentCanonicalizesProvidedHeader(t *testing.T) {
	r, root := setupModule(t)

	rw := callConnect(r, componentsconnect.ComponentsServiceInitializeComponentProcedure, `{
		"slug":"code-block",
		"libraryId":"react-component-library:code-block",
		"displayName":"Code Block",
		"description":"Copyable fenced code block",
		"tags":["markdown","code"],
		"initialVersion":"0.1.0",
		"initialSource":"/**\n * @libraryId react-component-library:CodeBlock\n * @displayName Old Code Block\n * @description obsolete\n * @version 9.9.9\n * @tags [\"obsolete\"]\n * @deps npm/react-markdown\n */\nexport const CodeBlock = () => null;\n"
	}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `react-component-library:code-block`)

	sourcePath := filepath.Join(root, "components", "code-block", "versions", "0.1.0", "code-block.tsx")
	source, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	require.Contains(t, string(source), "@libraryId react-component-library:code-block")
	require.Contains(t, string(source), "@displayName Code Block")
	require.Contains(t, string(source), "@description Copyable fenced code block")
	require.Contains(t, string(source), "@version 0.1.0")
	require.Contains(t, string(source), "@tags [\"markdown\",\"code\"]")
	require.Contains(t, string(source), "@deps npm/react-markdown")
	require.FileExists(t, filepath.Join(root, "components", "code-block", "versions", "0.1.0", "story.json"))

	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:code-block"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
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

func TestModule_GetIncludesComponentExperienceWhenRequested(t *testing.T) {
	reader := &fakeExperienceReader{snapshot: experience.Snapshot{
		ContractID: "button", Title: "Button", EvidenceStatus: "available",
		Claims:   []experience.Claim{{ID: "visible", Tier: "machine", Statement: "Button is visible."}},
		Evidence: []experience.Evidence{{ClaimID: "visible", Verdict: "pass", CaptureRef: "http://captures/button.png"}},
	}}
	r, root := setupModule(t, components.WithExperienceReader(reader))
	writeButtonManifest(t, root, `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => null;
`)
	require.Equal(t, http.StatusOK, callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`).Code)

	rw := callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure, `{"libraryId":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	body := rw.Body.String()
	idStart := strings.Index(body, `"id":"`) + len(`"id":"`)
	idEnd := strings.Index(body[idStart:], `"`)
	id := body[idStart : idStart+idEnd]

	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentProcedure, `{"id":"`+id+`","includeExperience":true}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"contractId":"button"`)
	require.Contains(t, rw.Body.String(), `"evidenceStatus":"available"`)
	require.Contains(t, rw.Body.String(), `http://captures/button.png`)
	require.Equal(t, id, reader.component.ID)
}

func TestModule_IndexSurfacesStaleDesignStyleFinding(t *testing.T) {
	r, root := setupModule(t)
	writeButtonManifestWithStyles(t, root, `[{"styleId":"missing-style","affinity":"native"}]`)

	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `react-component-library:Button`)
	require.Contains(t, rw.Body.String(), `finding:stale_design_style`)
	require.Contains(t, rw.Body.String(), `missing-style`)
}

func TestModule_ValidateStyleFitRoundTrip(t *testing.T) {
	r, root := setupModule(t)
	writeButtonManifestWithStyles(t, root, `[{"styleId":"vrooli-default","affinity":"native","reason":"token-native baseline"}]`)
	writeScenarioServiceJSON(t, root, "target-app", "vrooli-default")

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

	rw = callConnect(r, componentsconnect.ComponentsServiceValidateStyleFitProcedure,
		`{"componentId":"`+id+`","scenario":"target-app","version":"1.0.0"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"kind":"STYLE_FIT_VERDICT_KIND_OK"`)
	require.Contains(t, rw.Body.String(), `"scenarioStyle":"vrooli-default"`)
	require.Contains(t, rw.Body.String(), `token-native baseline`)
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

func writeButtonManifestWithStyles(t *testing.T, root, designStyles string) {
	t.Helper()
	dir := filepath.Join(root, "components", "Button")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.json"), []byte(`{
  "libraryId": "react-component-library:Button",
  "displayName": "Button",
  "description": "Primary CTA.",
  "tags": ["form"],
  "designStyles": `+designStyles+`,
  "latest": "1.0.0",
  "deprecatedVersions": []
}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "versions", "1.0.0", "Button.tsx"), []byte(`/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => null;
`), 0o600))
}

func writeScenarioServiceJSON(t *testing.T, root, scenario, styleID string) {
	t.Helper()
	dir := filepath.Join(root, "..", scenario, ".vrooli")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "service.json"), []byte(`{
  "generation": {
    "design": {
      "id": "`+styleID+`"
    }
  }
}`), 0o600))
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
