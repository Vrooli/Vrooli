package components_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	componentauthoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componentauthoring"
	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componentauthoring/componentauthoring_v1connect"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"

	"react-component-library/handlers/components"
	previewhandlers "react-component-library/handlers/preview"
	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/experience"
	"react-component-library/internal/versionledger"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
)

type fakeExperienceReader struct {
	snapshot  experience.Snapshot
	component experience.Component
}

type presenceReconcilerRecorder struct{ calls int }

func (r *presenceReconcilerRecorder) ReconcilePresence(context.Context, string, bool) error {
	r.calls++
	return nil
}

type presenceReconcilerFailure struct{ err error }

func (r presenceReconcilerFailure) ReconcilePresence(context.Context, string, bool) error {
	return r.err
}

type indexObserverRecorder struct {
	calls int
	err   error
}

func (o *indexObserverRecorder) Observe(_ context.Context, component internalcomponents.Component, _ internalcomponents.IndexManifestInput) error {
	o.calls++
	if component.LibraryID == "" {
		return errors.New("observer received an unindexed component")
	}
	return o.err
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
	internalcomponents.SetTypeScriptChecker(svc, internalcomponents.TypeScriptCheckerFunc(func(context.Context, string, string, string, string) internalcomponents.TypeScriptCheckResult {
		return internalcomponents.TypeScriptCheckResult{Passed: true, Message: "fixture TypeScript check passed"}
	}))
	internalcomponents.SetServiceJSONReader(svc, internalcomponents.NewFSServiceJSONReader(filepath.Dir(root)))
	opts = append(opts, components.WithPreviewService(previewhandlers.BuildServiceAtRoot(svc, nil, root)))
	m := components.ModuleFromService(svc, repo, root, log.New(io.Discard, "", 0), opts...)
	r := mux.NewRouter()
	m.Mount(r)
	return r, root
}

func TestComponentsReadsRequestRoutedDBPool(t *testing.T) {
	ctx := context.Background()
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "primary.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	ensureSchema := func(ctx context.Context, pool *sql.DB) error {
		return apidb.EnsureSchemas(ctx, pool, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema))
	}
	require.NoError(t, ensureSchema(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(ensureSchema)
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "components-routing", time.Minute))

	_, primaryRepo := components.BuildService(routed.Primary(), schedule.System(), t.TempDir())
	_, err = primaryRepo.Upsert(ctx, internalcomponents.UpsertInput{LibraryID: "rcl:routed", Slug: "Routed", DisplayName: "Routed"})
	require.NoError(t, err)
	root := t.TempDir()
	service, repo := components.BuildService(routed, schedule.System(), root)
	router := mux.NewRouter()
	components.ModuleFromService(service, repo, root, log.New(io.Discard, "", 0)).Mount(router)

	list := func(requestContext context.Context) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, componentsconnect.ComponentsServiceListComponentsProcedure, strings.NewReader(`{}`)).WithContext(requestContext)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	primaryResponse := list(ctx)
	require.Equal(t, http.StatusOK, primaryResponse.Code, primaryResponse.Body.String())
	require.Contains(t, primaryResponse.Body.String(), "rcl:routed")
	testResponse := list(apidb.WithTestMode(ctx))
	require.Equal(t, http.StatusOK, testResponse.Code, testResponse.Body.String())
	require.NotContains(t, testResponse.Body.String(), "rcl:routed", "request context must select the isolated routed pool")
}

type ingestScenarioReader map[string][]byte

func (r ingestScenarioReader) Read(_ context.Context, _ string, sourcePath string) ([]byte, error) {
	body, ok := r[sourcePath]
	if !ok {
		return nil, os.ErrNotExist
	}
	return body, nil
}

func setupIngestModule(t *testing.T, reader internalcomponents.ScenarioSourceReader) (*mux.Router, string) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalcomponents.Schema),
	))
	root := t.TempDir()
	svc, repo := components.BuildService(d, schedule.System(), root)
	internalcomponents.SetScenarioSourceReader(svc, reader)
	internalcomponents.SetTypeScriptChecker(svc, internalcomponents.TypeScriptCheckerFunc(func(context.Context, string, string, string, string) internalcomponents.TypeScriptCheckResult {
		return internalcomponents.TypeScriptCheckResult{Passed: true, Message: "fixture TypeScript check passed"}
	}))
	m := components.ModuleFromService(svc, repo, root, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)
	return r, root
}

func TestModule_AuthoringWorkflowUsesLibraryIDAndPublishesOnlyAfterCheck(t *testing.T) {
	r, root := setupModule(t)

	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
		"slug":"Button",
		"libraryId":"react-component-library:Button",
		"displayName":"Button",
		"initialVersion":"1.0.0",
		"initialSource":"export function Button() { return <button>Save</button>; }"
	}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = callConnect(r, authoringconnect.ComponentAuthoringServiceBeginComponentVersionProcedure,
		`{"component":"react-component-library:Button","bump":"minor"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.1.0-draft.1"`)
	require.Contains(t, rw.Body.String(), `"previewPath":"/preview/react-component-library:Button/harness.html?version=1.1.0-draft.1"`)

	rw = callConnect(r, authoringconnect.ComponentAuthoringServiceCheckComponentVersionProcedure,
		`{"component":"react-component-library:Button"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"passed":true`)
	require.Contains(t, rw.Body.String(), `"stage":"preview"`)
	require.Contains(t, rw.Body.String(), `"verdict":"passed"`)
	require.Contains(t, rw.Body.String(), `"stage":"typescript"`)

	rw = callConnect(r, authoringconnect.ComponentAuthoringServicePublishComponentVersionProcedure,
		`{"component":"react-component-library:Button","changelogMd":"Focused authoring workflow"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.1.0"`)
	require.FileExists(t, filepath.Join(root, "components", "Button", "versions", "1.1.0", "story.json"))
	manifest, err := os.ReadFile(filepath.Join(root, "components", "Button", "component.json"))
	require.NoError(t, err)
	require.Contains(t, string(manifest), `"latest": "1.1.0"`)
	require.NotContains(t, string(manifest), `1.1.0-draft.1`)
}

func TestModule_UpdateComponentManifestAppliesGovernedMetadata(t *testing.T) {
	r, root := setupModule(t)
	initialize := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
		"slug":"CallToAction",
		"libraryId":"react-component-library:CallToAction",
		"displayName":"Call to action",
		"initialVersion":"1.0.0",
		"initialSource":"export function CallToAction() { return <button>Continue</button>; }"
	}`)
	require.Equal(t, http.StatusOK, initialize.Code, initialize.Body.String())

	updated := callConnect(r, authoringconnect.ComponentAuthoringServiceUpdateComponentManifestProcedure, `{
		"componentId":"react-component-library:CallToAction",
		"displayName":"Continue action",
		"description":"A clear next step.",
		"tags":["actions","primary"],
		"retiredMajorAliases":["0"],
		"dependencies":[]
	}`)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())
	require.Contains(t, updated.Body.String(), `"displayName":"Continue action"`)

	manifest, err := os.ReadFile(filepath.Join(root, "components", "CallToAction", "component.json"))
	require.NoError(t, err)
	var persisted struct {
		DisplayName         string   `json:"displayName"`
		Description         string   `json:"description"`
		Tags                []string `json:"tags"`
		RetiredMajorAliases []string `json:"retiredMajorAliases"`
	}
	require.NoError(t, json.Unmarshal(manifest, &persisted))
	require.Equal(t, "Continue action", persisted.DisplayName)
	require.Equal(t, "A clear next step.", persisted.Description)
	require.Equal(t, []string{"actions", "primary"}, persisted.Tags)
	require.Equal(t, []string{"0"}, persisted.RetiredMajorAliases)
}

func TestModule_Shape(t *testing.T) {
	r, _ := setupModule(t)
	require.NotNil(t, r)
	require.Len(t, components.Endpoints, 22, "the module documents every registry and authoring endpoint once")
	require.Equal(t, 12, componentsv1.File_react_component_library_v1_components_components_proto.Services().ByName("ComponentsService").Methods().Len())
	require.Equal(t, 10, componentauthoringv1.File_react_component_library_v1_componentauthoring_component_authoring_proto.Services().ByName("ComponentAuthoringService").Methods().Len())
}

func TestModule_IndexComponentsReconcilesUnlessSuppressed(t *testing.T) {
	recorder := &presenceReconcilerRecorder{}
	r, _ := setupModule(t, components.WithPresenceReconciler(recorder))

	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Equal(t, 1, recorder.calls)

	rw = callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{"noReconcile":true}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Equal(t, 1, recorder.calls, "the explicit no_reconcile diagnostic path must skip tier movement")
}

func TestModule_IndexComponentsReportsDeferredAndFailedPresenceReconciliation(t *testing.T) {
	t.Run("deferred tier moves preserve successful index response", func(t *testing.T) {
		reconciler := presenceReconcilerFailure{err: versionledger.ErrPresenceReconciliationIncomplete{Deferred: []string{"Button@1.0.0"}}}
		r, root := setupModule(t, components.WithPresenceReconciler(reconciler))
		writeButtonManifest(t, root, `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => null;
`)
		rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		require.Contains(t, rw.Body.String(), "1 version tier move(s) deferred")
		require.Contains(t, rw.Body.String(), "Button@1.0.0")
	})

	t.Run("unexpected reconciliation failure fails the request", func(t *testing.T) {
		r, root := setupModule(t, components.WithPresenceReconciler(presenceReconcilerFailure{err: errors.New("tier store offline")}))
		writeButtonManifest(t, root, `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => null;
`)
		rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
		require.Equal(t, http.StatusBadRequest, rw.Code, rw.Body.String())
		require.Contains(t, rw.Body.String(), `"code":"failed_precondition"`)
		require.Contains(t, rw.Body.String(), "tier store offline")
	})
}

func TestModule_IndexComponentsSurfacesPostUpsertObserverFailures(t *testing.T) {
	observer := &indexObserverRecorder{err: errors.New("dependency projection unavailable")}
	r, root := setupModule(t, components.WithIndexObserver(observer))
	writeButtonManifest(t, root, `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => null;
`)
	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Equal(t, 1, observer.calls)
	require.Contains(t, rw.Body.String(), "dependency projection unavailable")
}

func TestModule_PersistPreviewFrameCreatesVersionPinnedDraftStory(t *testing.T) {
	r, root := setupModule(t)
	writePageFrameTestCatalog(t, root)
	writePageManifest(t, root)

	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = callConnect(r, componentsconnect.ComponentsServiceListPreviewFramesProcedure, `{"componentId":"react-component-library:Page","version":"1.0.0","storyId":"default"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"asset":"navigation.page"`)

	rw = callConnect(r, componentsconnect.ComponentsServicePersistPreviewFrameProcedure, `{
  "componentId":"react-component-library:Page",
  "version":"1.0.0",
  "storyId":"default",
  "asset":"navigation.page",
  "frameVersion":"1.0.0",
  "region":"navigation",
  "capability":"navigation"
}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.0.1-draft.1"`)
	require.Contains(t, rw.Body.String(), `navigation.page`)
	require.FileExists(t, filepath.Join(root, "components", "Page", "versions", "1.0.1-draft.1", "story.json"))
}

func TestModule_PersistPreviewFrameRejectsInvalidRegionBeforeCreatingDraft(t *testing.T) {
	r, root := setupModule(t)
	writePageFrameTestCatalog(t, root)
	writePageManifest(t, root)
	require.Equal(t, http.StatusOK, callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`).Code)

	rw := callConnect(r, componentsconnect.ComponentsServicePersistPreviewFrameProcedure, `{
  "componentId":"react-component-library:Page",
  "version":"1.0.0",
  "storyId":"default",
  "asset":"navigation.page",
  "frameVersion":"1.0.0",
  "region":"not-a-region",
  "capability":"navigation"
	}`)
	require.Equal(t, http.StatusBadRequest, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"code":"failed_precondition"`)
	_, err := os.Stat(filepath.Join(root, "components", "Page", "versions", "1.0.1-draft.1"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestModule_InitializeComponentRoundTrip(t *testing.T) {
	r, root := setupModule(t)

	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
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

func TestModule_CreateComponentVersionDraftRoute(t *testing.T) {
	r, root := setupModule(t)
	initialized := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
		"slug":"toast",
		"libraryId":"react-component-library:toast",
		"displayName":"Toast",
		"initialVersion":"1.0.0"
	}`)
	require.Equal(t, http.StatusOK, initialized.Code, initialized.Body.String())

	created := callConnect(r, authoringconnect.ComponentAuthoringServiceCreateComponentVersionProcedure, `{
		"componentId":"react-component-library:toast",
		"version":"1.1.0-draft.1",
		"fromVersion":"1.0.0",
		"intent":"COMPONENT_VERSION_INTENT_DRAFT",
		"source":"export function Toast() { return <aside role=\"status\">Saved</aside>; }"
	}`)
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	require.Contains(t, created.Body.String(), `"version":"1.1.0-draft.1"`)
	require.Contains(t, created.Body.String(), `"libraryId":"react-component-library:toast"`)
	require.FileExists(t, filepath.Join(root, "components", "toast", "versions", "1.1.0-draft.1", "toast.tsx"))
	require.Contains(t, created.Body.String(), `"sourcePath":"components/toast/versions/1.1.0-draft.1/toast.tsx"`)
}

func TestModule_IngestComponentCreatesDraftThroughConnect(t *testing.T) {
	r, root := setupIngestModule(t, ingestScenarioReader{
		"ui/src/components/Notice.tsx": []byte(`export function Notice() { return <aside role="status">Saved</aside>; }`),
	})
	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceIngestComponentProcedure, `{
		"scenario":"web-console",
		"sourceFile":"ui/src/components/Notice.tsx",
		"slug":"notice",
		"displayName":"Notice",
		"tags":["feedback"]
	}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"libraryId":"react-component-library:notice"`)
	require.Contains(t, rw.Body.String(), `"draftVersion":"0.1.0-draft.1"`)
	require.Contains(t, rw.Body.String(), `"checklistPath":"docs/guides/de-scenario-ification-checklist.md"`)
	require.FileExists(t, filepath.Join(root, "components", "notice", "versions", "0.1.0-draft.1", "Notice.tsx"))
}

func TestModule_IngestComponentRejectsUnacceptedBehaviorLoss(t *testing.T) {
	r, root := setupIngestModule(t, ingestScenarioReader{
		"ui/src/components/DrawerShell.tsx": []byte(`import { useFocusTrap } from "@/hooks/useFocusTrap"; export function DrawerShell() { useFocusTrap(); return <div role="dialog" aria-modal="true" />; }`),
		"ui/src/hooks/useFocusTrap.ts":      []byte(`export function useFocusTrap() { window.addEventListener("keydown", () => {}); }`),
	})
	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceIngestComponentProcedure, `{
		"scenario":"web-console",
		"sourceFile":"ui/src/components/DrawerShell.tsx",
		"slug":"drawer-shell",
		"displayName":"Drawer Shell"
	}`)
	require.Equal(t, http.StatusBadRequest, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"failed_precondition"`)
	require.Contains(t, rw.Body.String(), "addEventListener")
	require.NoFileExists(t, filepath.Join(root, "components", "drawer-shell", "component.json"))
}

func TestModule_InitializeComponentCanonicalizesProvidedHeader(t *testing.T) {
	r, root := setupModule(t)

	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
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

	rw = callConnect(r, authoringconnect.ComponentAuthoringServiceGetComponentContentProcedure,
		`{"id":"`+id+`"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), "@libraryId")
	require.Contains(t, rw.Body.String(), `"sha256"`)

	rw = callConnect(r, componentsconnect.ComponentsServiceListComponentVersionsProcedure,
		`{"componentId":"`+id+`"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.0.0"`)
	rw = callConnect(r, componentsconnect.ComponentsServiceGetComponentVersionContentProcedure,
		`{"componentId":"`+id+`","version":"1.0.0","path":"Button.tsx"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), "@libraryId")
	rw = callConnect(r, componentsconnect.ComponentsServiceListComponentStoriesProcedure,
		`{"componentId":"`+id+`","version":"1.0.0"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	rw = callConnect(r, componentsconnect.ComponentsServiceListDesignStylesProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = callConnect(r, authoringconnect.ComponentAuthoringServiceBeginComponentVersionProcedure,
		`{"component":"react-component-library:Button","bump":"patch"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.0.1-draft.1"`)

	rw = callConnect(r, authoringconnect.ComponentAuthoringServiceUpdateComponentContentProcedure,
		`{"id":"`+id+`","content":"// rewritten\nexport const Button = () => null;\n"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"sha256"`)

	written, err := os.ReadFile(filepath.Join(root, "components", "Button", "versions", "1.0.1-draft.1", "Button.tsx"))
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

func writePageManifest(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "components", "Page")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.json"), []byte(`{
  "libraryId": "react-component-library:Page",
  "catalogId": "navigation.page",
  "displayName": "Page",
  "latest": "1.0.0",
  "deprecatedVersions": []
}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "versions", "1.0.0", "Page.tsx"), []byte(`/**
 * @libraryId react-component-library:Page
 * @version 1.0.0
 */
export const Page = () => null;
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "versions", "1.0.0", "story.json"), []byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id": "default", "name": "Default", "args": {}}]
}`), 0o600))
}

func writePageFrameTestCatalog(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(filepath.Dir(root), "catalog", "assets", "navigation")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "page.json"), []byte(`{
  "kind": "catalog-asset",
  "asset": {"id": "navigation.page", "kind": "navigation", "targets": ["react-vite"]},
  "regions": [
    {"id": "navigation", "accepts": "navigation"},
    {"id": "content"}
  ],
  "expects": [{"capability": "router-adapter"}]
}`), 0o600))
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

func TestManifestRetirementRejectsSetAndClear(t *testing.T) {
	handler := components.NewAuthoringConnectHandler(components.AuthoringDeps{})
	_, err := handler.UpdateComponentManifest(t.Context(), connect.NewRequest(&componentsv1.UpdateComponentManifestRequest{RetiredMajorAliases: []string{"1"}, ClearRetiredMajorAliases: true}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

type retirementHandlerRepo struct {
	internalcomponents.Repository
	component internalcomponents.Component
	deleted   bool
}

func (r *retirementHandlerRepo) GetByLibraryID(context.Context, string) (internalcomponents.Component, error) {
	return r.component, nil
}

func (r *retirementHandlerRepo) ListVersions(context.Context, string, int) ([]internalcomponents.ComponentVersion, error) {
	return nil, nil
}

func (r *retirementHandlerRepo) ListStories(context.Context, internalcomponents.StoryQuery) ([]internalcomponents.ComponentStory, error) {
	return nil, nil
}

func (r *retirementHandlerRepo) DeleteRetiredComponent(context.Context, string, string) error {
	r.deleted = true
	return nil
}

func TestRetireHandlerPreflightThenApply(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios/react-component-library/library")
	manifest := filepath.Join(source, "components/Old/component.json")
	catalog := filepath.Join(root, "scenarios/react-component-library/catalog/assets/navigation/old.json")
	for path, data := range map[string]string{manifest: `{"libraryId":"library:Old","catalogId":"navigation.old"}`, catalog: `{"kind":"catalog-asset","asset":{"id":"navigation.old","kind":"component","target":{"maturity":"deprecated"}}}`} {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(data), 0o644))
	}
	repo := &retirementHandlerRepo{component: internalcomponents.Component{ID: "old", LibraryID: "library:Old", CatalogID: "navigation.old", Slug: "Old", ManifestPath: "components/Old/component.json"}}
	h := components.NewAuthoringConnectHandler(components.AuthoringDeps{Repo: repo, SourceRoot: source})
	_, err := h.RetireComponent(t.Context(), connect.NewRequest(&componentsv1.RetireComponentRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	checked, err := h.RetireComponent(t.Context(), connect.NewRequest(&componentsv1.RetireComponentRequest{LibraryId: "library:Old"}))
	require.NoError(t, err)
	require.Empty(t, checked.Msg.Error)
	require.True(t, checked.Msg.Ready)
	require.False(t, checked.Msg.Retired)
	require.False(t, repo.deleted)
	require.FileExists(t, manifest)
	require.FileExists(t, catalog)
	applied, err := h.RetireComponent(t.Context(), connect.NewRequest(&componentsv1.RetireComponentRequest{LibraryId: "library:Old", Apply: true}))
	require.NoError(t, err)
	require.Empty(t, applied.Msg.Error)
	require.True(t, applied.Msg.Retired)
	require.True(t, repo.deleted)
	require.FileExists(t, applied.Msg.SnapshotPath)
	require.FileExists(t, applied.Msg.CatalogArchivePath)
	require.NoFileExists(t, manifest)
	require.NoFileExists(t, catalog)
}

type failingExperienceReader struct{ err error }

func (f failingExperienceReader) Get(context.Context, experience.Component) (experience.Snapshot, error) {
	return experience.Snapshot{}, f.err
}

func TestModule_GetComponentIncludesExperienceFailsExplicitlyWhenUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name       string
		option     components.ModuleOption
		wantCode   string
		wantStatus int
	}{
		{name: "reader not configured", wantCode: "failed_precondition", wantStatus: http.StatusBadRequest},
		{name: "reader failure", option: components.WithExperienceReader(failingExperienceReader{err: os.ErrPermission}), wantCode: "internal", wantStatus: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r *mux.Router
			var root string
			if tc.option == nil {
				r, root = setupModule(t)
			} else {
				r, root = setupModule(t, tc.option)
			}
			writeButtonManifest(t, root, `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 */
export const Button = () => null;
`)
			require.Equal(t, http.StatusOK, callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`).Code)
			lookup := callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure, `{"libraryId":"react-component-library:Button"}`)
			require.Equal(t, http.StatusOK, lookup.Code, lookup.Body.String())
			body := lookup.Body.String()
			idStart := strings.Index(body, `"id":"`) + len(`"id":"`)
			idEnd := strings.Index(body[idStart:], `"`)
			id := body[idStart : idStart+idEnd]
			require.NotEmpty(t, id)

			response := callConnect(r, componentsconnect.ComponentsServiceGetComponentProcedure, `{"id":"`+id+`","includeExperience":true}`)
			require.Equal(t, tc.wantStatus, response.Code, response.Body.String())
			require.Contains(t, response.Body.String(), `"code":"`+tc.wantCode+`"`)
		})
	}
}

func TestModule_ListPreviewFramesReportsCompatibilityAndFixtureChoices(t *testing.T) {
	r, root := setupModule(t)
	writePageFrameTestCatalog(t, root)
	writePageManifest(t, root)

	catalogRoot := filepath.Join(filepath.Dir(root), "catalog", "assets")
	writeCatalogFrame := func(category, name, body string) {
		t.Helper()
		dir := filepath.Join(catalogRoot, category)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600))
	}
	writeCatalogFrame("navigation", "app-shell", `{"kind":"catalog-asset","asset":{"id":"navigation.app-shell","kind":"navigation","targets":["react-vite"]},"regions":[{"id":"content","accepts":"data-display"}],"expects":[{"capability":"data-source"}]}`)
	writeCatalogFrame("controls", "card-frame", `{"kind":"catalog-asset","asset":{"id":"controls.card-frame","kind":"component","targets":["react-vite"]},"regions":[{"id":"content"}]}`)
	writeCatalogFrame("controls", "unsupported-frame", `{"kind":"catalog-asset","asset":{"id":"controls.unsupported-frame","kind":"component","targets":["other-target"]},"regions":[{"id":"content"}]}`)
	writeCatalogFrame("fixtures", "records", `{"kind":"catalog-asset","asset":{"id":"fixtures.records","kind":"fixture"},"fixture":{"satisfies":{"capability":"data-source"}}}`)

	appShellDir := filepath.Join(root, "components", "AppShell")
	require.NoError(t, os.MkdirAll(filepath.Join(appShellDir, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appShellDir, "component.json"), []byte(`{"libraryId":"react-component-library:AppShell","catalogId":"navigation.app-shell","displayName":"AppShell","latest":"1.0.0"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(appShellDir, "versions", "1.0.0", "AppShell.tsx"), []byte(`/**
 * @libraryId react-component-library:AppShell
 * @version 1.0.0
 */
export const AppShell = () => null;
`), 0o600))

	rw := callConnect(r, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	rw = callConnect(r, componentsconnect.ComponentsServiceListPreviewFramesProcedure, `{"componentId":"react-component-library:Page"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"asset":"navigation.page"`)
	require.Contains(t, rw.Body.String(), `"asset":"navigation.app-shell"`)
	require.Contains(t, rw.Body.String(), `"fixture":"fixtures.records"`)
	require.Contains(t, rw.Body.String(), `"diagnosticCode":"subject_capability_mismatch"`)
	require.Contains(t, rw.Body.String(), `"diagnosticCode":"not_canonical_frame"`)
	require.Contains(t, rw.Body.String(), `"diagnosticCode":"unsupported_target"`)
}
