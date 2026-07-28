package componenttests_test

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
	testsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests/componenttests_v1connect"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"react-component-library/handlers/components"
	componenttests "react-component-library/handlers/componenttests"
	"react-component-library/internal/clock"
	internalcomponents "react-component-library/internal/components"
	domain "react-component-library/internal/componenttests"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/testutil/db"
)

type passingExecutor struct{}

func (passingExecutor) ExecuteStory(context.Context, string, string, string) (domain.StoryExecution, error) {
	return domain.StoryExecution{Passed: true}, nil
}

func TestModuleRunsAndListsDurableContractReport(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "components", "Button", "versions", "1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Button", "component.json"), []byte(`{"libraryId":"rcl:Button","displayName":"Button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "Button.tsx"), []byte("/**\n * @libraryId rcl:Button\n * @version 1.0.0\n */\nexport const Button = () => null;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "story.json"), []byte(`{"schemaVersion":1,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"idle","name":"Idle","args":{},"expect":[{"kind":"role","role":"button","name":"Button"}]}]}`), 0o644))
	assets, repo := components.BuildService(database, clock.System{}, root)
	router := mux.NewRouter()
	components.ModuleFromService(assets, repo, root, log.New(io.Discard, "", 0)).Mount(router)
	componenttests.ModuleWithExecutor(database, assets, root, passingExecutor{}, log.New(io.Discard, "", 0)).Mount(router)
	index := call(router, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, index.Code, index.Body.String())
	provider := call(router, validationconnect.ScenarioValidationServiceValidateScenarioProcedure, `{"scenario":"react-component-library","includeExecution":true}`)
	require.Equal(t, http.StatusOK, provider.Code, provider.Body.String())
	require.Contains(t, provider.Body.String(), `"status":"VALIDATION_STATUS_PASSED"`)
	byID := call(router, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure, `{"libraryId":"rcl:Button"}`)
	require.Equal(t, http.StatusOK, byID.Code, byID.Body.String())
	require.Contains(t, byID.Body.String(), `"id":"`)
	// Connect JSON response has the id near the start; use an intentionally stable
	// test request from the indexed service output instead of any filesystem path.
	start := strings.Index(byID.Body.String(), `"id":"`) + 6
	end := start + strings.Index(byID.Body.String()[start:], `"`)
	id := byID.Body.String()[start:end]
	run := call(router, testsconnect.ComponentTestsServiceRunComponentTestProcedure, `{"componentId":"`+id+`","version":"1.0.0","includeClosure":true}`)
	require.Equal(t, http.StatusOK, run.Code, run.Body.String())
	require.Contains(t, run.Body.String(), `"verdict":"passed"`)
	runIDStart := strings.Index(run.Body.String(), `"id":"`) + len(`"id":"`)
	runIDEnd := runIDStart + strings.Index(run.Body.String()[runIDStart:], `"`)
	runID := run.Body.String()[runIDStart:runIDEnd]
	rerun := call(router, testsconnect.ComponentTestsServiceRerunComponentTestProcedure, `{"reportId":"`+runID+`"}`)
	require.Equal(t, http.StatusOK, rerun.Code, rerun.Body.String())
	require.Contains(t, rerun.Body.String(), `"verdict":"passed"`)
	list := call(router, testsconnect.ComponentTestsServiceListComponentTestReportsProcedure, `{"componentId":"`+id+`"}`)
	require.Equal(t, http.StatusOK, list.Code, list.Body.String())
	require.Contains(t, list.Body.String(), `"reports"`)
}

func call(router *mux.Router, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
