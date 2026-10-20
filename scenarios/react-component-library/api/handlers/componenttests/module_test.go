package componenttests_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"react-component-library/handlers/components"
	componenttests "react-component-library/handlers/componenttests"
	internalcomponents "react-component-library/internal/components"
	domain "react-component-library/internal/componenttests"
	localdb "react-component-library/internal/database"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	testsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests/componenttests_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type passingExecutor struct{}

func (passingExecutor) ExecuteStory(context.Context, string, string, string) (domain.StoryExecution, error) {
	return domain.StoryExecution{Passed: true, AccessibilityJSON: `{"contract":"bas-accessibility-snapshot/v1"}`, Artifacts: []domain.Artifact{{Reference: "test://capture"}}}, nil
}

type countingFixture struct{ calls atomic.Int32 }

func (f *countingFixture) Validate(context.Context) error {
	f.calls.Add(1)
	return nil
}

type unavailableExecutor struct {
	called atomic.Bool
}

func (e *unavailableExecutor) ExecuteStory(context.Context, string, string, string) (domain.StoryExecution, error) {
	e.called.Store(true)
	return domain.StoryExecution{}, domain.ExecutorUnavailableError{Err: io.ErrUnexpectedEOF}
}

func (*unavailableExecutor) Probe(context.Context) error {
	return domain.ExecutorUnavailableError{Err: io.ErrUnexpectedEOF}
}

func TestModuleFailsValidationBeforeSchedulingWhenExecutorUnavailable(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "components", "Button", "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Button", "component.json"), []byte(`{"libraryId":"rcl:Button","catalogId":"controls.button","displayName":"Button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "Button.tsx"), []byte("export const Button = () => null;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "story.json"), []byte(`{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"idle","name":"Idle","args":{},"expect":[]}]}`), 0o644))
	assets, repo := components.BuildService(database, schedule.System(), root)
	router := mux.NewRouter()
	components.ModuleFromService(assets, repo, root, log.New(io.Discard, "", 0)).Mount(router)
	executor := &unavailableExecutor{}
	componenttests.ModuleWithExecutor(database, assets, root, executor, log.New(io.Discard, "", 0)).Mount(router)

	provider := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library"}`)
	require.Equal(t, http.StatusOK, provider.Code, provider.Body.String())
	require.Contains(t, provider.Body.String(), `"status":"VALIDATION_STATUS_FAILED"`)
	require.Contains(t, provider.Body.String(), `COMPONENT_TEST_RUNNER_UNAVAILABLE:`)
	require.False(t, executor.called.Load())
}

func TestValidateScenarioReportsSelectedBASRouteWithoutCapture(t *testing.T) {
	var healthGets atomic.Int32
	var capturePosts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			healthGets.Add(1)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/browser_automation_studio.v1.capture.CaptureService/Capture":
			capturePosts.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	root := t.TempDir()
	assets, repo := components.BuildService(database, schedule.System(), root)
	router := mux.NewRouter()
	components.ModuleFromService(assets, repo, root, log.New(io.Discard, "", 0)).Mount(router)
	executor := domain.BASCaptureExecutor{BASBaseURL: server.URL, HTTPClient: server.Client()}
	componenttests.ModuleWithExecutor(database, assets, root, executor, log.New(io.Discard, "", 0)).Mount(router)

	provider := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library"}`)
	require.Equal(t, http.StatusOK, provider.Code, provider.Body.String())
	var response scenariovalidationv1.ValidateScenarioResponse
	require.NoError(t, protojson.Unmarshal(provider.Body.Bytes(), &response), provider.Body.String())
	require.NotNil(t, response.GetNativeDetail(), "provider response must include the same executor's read-only BAS selection")
	var detail structpb.Struct
	require.NoError(t, anypb.UnmarshalTo(response.GetNativeDetail(), &detail, proto.UnmarshalOptions{}))
	fields := detail.AsMap()
	require.Equal(t, server.URL, fields["bas_base_url"])
	require.Equal(t, "/browser_automation_studio.v1.capture.CaptureService/Capture", fields["capture_path"])
	require.NotEmpty(t, fields["resolved_at"])
	require.Equal(t, "healthy", fields["health_status"])
	require.Equal(t, true, fields["inspection_only"])
	require.Equal(t, false, fields["execution_requested"])
	require.EqualValues(t, 1, healthGets.Load())
	require.Zero(t, capturePosts.Load(), "validation read must never execute a browser capture")
}

func TestValidateScenarioInspectionOnlyDoesNotRunPopulatedCatalog(t *testing.T) {
	var healthGets atomic.Int32
	var capturePosts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			healthGets.Add(1)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/browser_automation_studio.v1.capture.CaptureService/Capture":
			capturePosts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"executionId":"capture-1","durationMs":42,"domHtml":"<pre id=\"rcl-story-result\" hidden>{\"passed\":true,\"failures\":[],\"performance\":{\"mountMs\":12.5,\"commitCount\":2,\"nodeCount\":8}}</pre>","accessibilityJson":"{\"contract\":\"bas-accessibility-snapshot/v1\"}","artifacts":[{"type":"CAPTURE_TYPE_SCREENSHOT","reference":"artifact://screenshot-1","sizeBytes":1234}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "catalog", "assets"), 0o755))
	sourceRoot := filepath.Join(root, "library")
	versionRoot := filepath.Join(sourceRoot, "components", "Button", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(versionRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceRoot, "components", "Button", "component.json"), []byte(`{"libraryId":"rcl:Button","catalogId":"controls.button","displayName":"Button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionRoot, "Button.tsx"), []byte("export const Button = () => null;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionRoot, "story.json"), []byte(`{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"idle","name":"Idle","args":{},"expect":[]}]}`), 0o644))
	assets, repo := components.BuildService(database, schedule.System(), sourceRoot)
	router := mux.NewRouter()
	components.ModuleFromService(assets, repo, sourceRoot, log.New(io.Discard, "", 0)).Mount(router)
	fixture := &countingFixture{}
	executor := domain.BASCaptureExecutor{RCLBaseURL: "http://rcl.test", BASBaseURL: server.URL, HTTPClient: server.Client()}
	componenttests.ModuleWithExecutorAndFixture(database, assets, sourceRoot, executor, fixture, log.New(io.Discard, "", 0)).Mount(router)
	index := call(router, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, index.Code, index.Body.String())
	byLibraryID := call(router, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure, `{"libraryId":"rcl:Button"}`)
	require.Equal(t, http.StatusOK, byLibraryID.Code, byLibraryID.Body.String())
	idStart := strings.Index(byLibraryID.Body.String(), `"id":"`) + len(`"id":"`)
	require.Greater(t, idStart, len(`"id":"`))
	idEnd := idStart + strings.Index(byLibraryID.Body.String()[idStart:], `"`)
	componentID := byLibraryID.Body.String()[idStart:idEnd]

	inspection := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library"}`)
	require.Equal(t, http.StatusOK, inspection.Code, inspection.Body.String())
	var inspected scenariovalidationv1.ValidateScenarioResponse
	require.NoError(t, protojson.Unmarshal(inspection.Body.Bytes(), &inspected), inspection.Body.String())
	// Assert the no-effect boundary before status so a causal baseline exposes
	// any accidental catalog execution, Capture POST, report, or fixture write.
	require.Zero(t, capturePosts.Load())
	require.Zero(t, fixture.calls.Load())
	require.EqualValues(t, 1, healthGets.Load())
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED, inspected.GetStatus())
	var detail structpb.Struct
	require.NoError(t, anypb.UnmarshalTo(inspected.GetNativeDetail(), &detail, proto.UnmarshalOptions{}))
	require.Equal(t, true, detail.AsMap()["inspection_only"])
	require.Equal(t, false, detail.AsMap()["execution_requested"])
	require.Contains(t, inspected.GetAssessment().GetCapabilities()[0].GetCurrentSummary(), "not evaluated")
	before := call(router, testsconnect.ComponentTestsServiceListComponentTestReportsProcedure, `{"componentId":"`+componentID+`"}`)
	require.Equal(t, http.StatusOK, before.Code, before.Body.String())
	var emptyReports componenttestsv1.ListComponentTestReportsResponse
	require.NoError(t, protojson.Unmarshal(before.Body.Bytes(), &emptyReports), before.Body.String())
	require.Empty(t, emptyReports.GetReports())

	// The existing explicit execution mode still enters the full catalog path.
	execution := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library","includeExecution":true}`)
	require.Equal(t, http.StatusOK, execution.Code, execution.Body.String())
	require.EqualValues(t, 1, capturePosts.Load())
	require.EqualValues(t, 1, fixture.calls.Load())
	var executed scenariovalidationv1.ValidateScenarioResponse
	require.NoError(t, protojson.Unmarshal(execution.Body.Bytes(), &executed), execution.Body.String())
	require.NoError(t, anypb.UnmarshalTo(executed.GetNativeDetail(), &detail, proto.UnmarshalOptions{}))
	require.Equal(t, false, detail.AsMap()["inspection_only"])
	require.Equal(t, true, detail.AsMap()["execution_requested"])
	after := call(router, testsconnect.ComponentTestsServiceListComponentTestReportsProcedure, `{"componentId":"`+componentID+`"}`)
	require.Equal(t, http.StatusOK, after.Code, after.Body.String())
	var savedReports componenttestsv1.ListComponentTestReportsResponse
	require.NoError(t, protojson.Unmarshal(after.Body.Bytes(), &savedReports), after.Body.String())
	require.NotEmpty(t, savedReports.GetReports())
}

func TestValidateScenarioReportsBASHealthFailureWithoutSensitiveNoise(t *testing.T) {
	var healthGets atomic.Int32
	var capturePosts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			healthGets.Add(1)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("private BAS diagnostic token=server-secret"))
			return
		}
		if r.Method == http.MethodPost {
			capturePosts.Add(1)
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)
	parsed.User = url.UserPassword("operator", "url-secret")
	parsed.Fragment = "fragment-secret"

	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	root := t.TempDir()
	assets, repo := components.BuildService(database, schedule.System(), root)
	router := mux.NewRouter()
	components.ModuleFromService(assets, repo, root, log.New(io.Discard, "", 0)).Mount(router)
	executor := domain.BASCaptureExecutor{BASBaseURL: parsed.String(), HTTPClient: server.Client()}
	componenttests.ModuleWithExecutor(database, assets, root, executor, log.New(io.Discard, "", 0)).Mount(router)

	provider := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library"}`)
	require.Equal(t, http.StatusOK, provider.Code, provider.Body.String())
	require.Contains(t, provider.Body.String(), `"status":"VALIDATION_STATUS_FAILED"`)
	require.Contains(t, provider.Body.String(), `"health_status":"unhealthy"`)
	require.Contains(t, provider.Body.String(), `"http_status_code":503`)
	require.Contains(t, provider.Body.String(), `"selection_exact":false`)
	for _, secret := range []string{"url-secret", "fragment-secret", "server-secret", "token="} {
		require.NotContains(t, provider.Body.String(), secret)
	}
	require.EqualValues(t, 1, healthGets.Load())
	require.Zero(t, capturePosts.Load(), "a failed health inspection must never try CaptureService")
}

func TestModuleRunsAndListsDurableContractReport(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "catalog", "assets"), 0o755))
	sourceRoot := filepath.Join(root, "library")
	require.NoError(t, os.MkdirAll(filepath.Join(sourceRoot, "components", "Button", "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceRoot, "components", "Button", "component.json"), []byte(`{"libraryId":"rcl:Button","catalogId":"controls.button","displayName":"Button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceRoot, "components", "Button", "versions", "1.0.0", "Button.tsx"), []byte("/**\n * @libraryId rcl:Button\n * @version 1.0.0\n */\nexport const Button = () => null;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceRoot, "components", "Button", "versions", "1.0.0", "story.json"), []byte(`{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"idle","name":"Idle","args":{},"expect":[{"kind":"role","role":"button","name":"Button"}]}]}`), 0o644))
	assets, repo := components.BuildService(database, schedule.System(), sourceRoot)
	router := mux.NewRouter()
	components.ModuleFromService(assets, repo, sourceRoot, log.New(io.Discard, "", 0)).Mount(router)
	componenttests.ModuleWithExecutor(database, assets, sourceRoot, passingExecutor{}, log.New(io.Discard, "", 0)).Mount(router)
	index := call(router, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, index.Code, index.Body.String())
	provider := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library","includeExecution":true}`)
	require.Equal(t, http.StatusOK, provider.Code, provider.Body.String())
	require.Contains(t, provider.Body.String(), `"status":"VALIDATION_STATUS_PASSED"`)
	description := call(router, validationconnect.ScenarioValidationServiceDescribeProviderProcedure, `{}`)
	require.Equal(t, http.StatusOK, description.Code, description.Body.String())
	require.Contains(t, description.Body.String(), `"provider":"react-component-library"`)
	require.Contains(t, description.Body.String(), `"phase":"component-tests"`)
	byID := call(router, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure, `{"libraryId":"rcl:Button"}`)
	require.Equal(t, http.StatusOK, byID.Code, byID.Body.String())
	require.Contains(t, byID.Body.String(), `"id":"`)
	// Connect JSON response has the id near the start; use an intentionally stable
	// test request from the indexed service output instead of any filesystem path.
	start := strings.Index(byID.Body.String(), `"id":"`) + 6
	end := start + strings.Index(byID.Body.String()[start:], `"`)
	id := byID.Body.String()[start:end]
	run := call(router, testsconnect.ComponentTestsServiceRunComponentTestProcedure, `{"componentId":"controls.button","version":"1.0.0","includeClosure":true}`)
	require.Equal(t, http.StatusOK, run.Code, run.Body.String())
	require.Contains(t, run.Body.String(), `"verdict":"passed"`)
	runIDStart := strings.Index(run.Body.String(), `"id":"`) + len(`"id":"`)
	runIDEnd := runIDStart + strings.Index(run.Body.String()[runIDStart:], `"`)
	runID := run.Body.String()[runIDStart:runIDEnd]
	get := call(router, testsconnect.ComponentTestsServiceGetComponentTestReportProcedure, `{"id":"`+runID+`"}`)
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
	var durable componenttestsv1.GetComponentTestReportResponse
	require.NoError(t, protojson.Unmarshal(get.Body.Bytes(), &durable), get.Body.String())
	require.NotNil(t, durable.GetReport())
	require.Equal(t, runID, durable.GetReport().GetId(), "the exact report returned by the save path must be retrievable")
	require.Equal(t, "passed", durable.GetReport().GetVerdict())
	require.True(t, durable.GetReport().GetIncludeClosure())
	require.NotEmpty(t, durable.GetReport().GetResults())
	rerun := call(router, testsconnect.ComponentTestsServiceRerunComponentTestProcedure, `{"reportId":"`+runID+`"}`)
	require.Equal(t, http.StatusOK, rerun.Code, rerun.Body.String())
	require.Contains(t, rerun.Body.String(), `"verdict":"passed"`)
	list := call(router, testsconnect.ComponentTestsServiceListComponentTestReportsProcedure, `{"componentId":"`+id+`"}`)
	require.Equal(t, http.StatusOK, list.Code, list.Body.String())
	require.Contains(t, list.Body.String(), `"reports"`)
	secondProvider := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library","includeExecution":true}`)
	require.Equal(t, http.StatusOK, secondProvider.Code, secondProvider.Body.String())
	require.Contains(t, secondProvider.Body.String(), `"reused_versions":1`)
	require.Contains(t, secondProvider.Body.String(), `"skipped_versions":1`)
}

func call(router *mux.Router, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
