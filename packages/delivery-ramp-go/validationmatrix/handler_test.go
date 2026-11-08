package validationmatrix

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestHandlerCreatesAndReadsDurableMatrix(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	NewHandler(NewService(store, Executors{})).RegisterRoutes(router)
	body, err := json.Marshal(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	create := httptest.NewRequest(http.MethodPost, "/api/v1/validation/matrices", bytes.NewReader(body))
	created := httptest.NewRecorder()
	router.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var run MatrixRun
	if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRequest(http.MethodGet, "/api/v1/validation/matrices/"+run.RunID, nil)
	read := httptest.NewRecorder()
	router.ServeHTTP(read, get)
	if read.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", read.Code, read.Body.String())
	}
	var loaded MatrixRun
	if err := json.Unmarshal(read.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.RunID != run.RunID || loaded.Matrix.GetArtifactDigest() != "sha256:artifact" {
		t.Fatalf("handler lost matrix identity: %+v", loaded)
	}
}

func TestMatrixHTTPResponsesRedactLocalArtifactPaths(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	selection := baseSelection()
	selection.ArtifactPath = "/producer/private/app.apk"
	selection.ArtifactPaths = map[string]string{"android:local": "/producer/private/app.apk"}
	service := NewService(store, Executors{})
	router := mux.NewRouter()
	NewHandler(service).RegisterRoutes(router)
	body, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	router.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/validation/matrices", bytes.NewReader(body)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var run MatrixRun
	if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if !store.Update(run.RunID, func(stored *MatrixRun) {
		stored.Cells[0].Report = map[string]string{"review_recording_path": "/producer/private/review.mp4", "status": "captured"}
		stored.Cells[0].Cell.Evidence = []*domainv1.LayeredEvidence{{
			EvidenceId: "review", Uri: "file:///producer/private/evidence.mp4", Sha256: "sha256:evidence", Redacted: true,
		}}
		stored.Matrix.Cells = []*domainv1.ValidationCell{stored.Cells[0].Cell}
		if stored.Selection.Metadata == nil {
			stored.Selection.Metadata = make(map[string]string)
		}
		stored.Selection.Metadata["diagnostic_path"] = "/producer/private/debug.json"
	}) {
		t.Fatal("could not attach internal path evidence")
	}
	for _, path := range []string{"/api/v1/validation/matrices/" + run.RunID, "/api/v1/validation/matrices?scenario=" + selection.ScenarioName} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
		for _, privatePath := range [][]byte{
			[]byte("/producer/private/app.apk"), []byte("/producer/private/review.mp4"),
			[]byte("/producer/private/debug.json"), []byte("file:///producer/private/evidence.mp4"), []byte("artifact_paths"),
			[]byte("review_recording_path"), []byte("diagnostic_path"),
		} {
			if bytes.Contains(response.Body.Bytes(), privatePath) {
				t.Fatalf("GET %s exposed local artifact paths: %s", path, response.Body.String())
			}
		}
	}
	stored, ok := store.Get(run.RunID)
	if !ok || stored.Selection.ArtifactPath != selection.ArtifactPath || stored.Selection.ArtifactPaths["android:local"] != selection.ArtifactPath || stored.Cells[0].Report["review_recording_path"] != "/producer/private/review.mp4" || stored.Selection.Metadata["diagnostic_path"] != "/producer/private/debug.json" || stored.Cells[0].Cell.Evidence[0].Uri != "file:///producer/private/evidence.mp4" || stored.Matrix.Cells[0].Evidence[0].Uri != "file:///producer/private/evidence.mp4" {
		t.Fatalf("store did not retain internal paths for execution: %+v", stored)
	}
}

func TestAbortHandlerWaitsForOwnerTerminalState(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	service := NewService(store, Executors{Local: localFunc(func(ctx context.Context, _ CellRequest) CellResult {
		close(started)
		<-ctx.Done()
		return CellResult{}
	})})
	run, err := service.Create(baseSelection())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(run.RunID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("cell did not start")
	}

	router := mux.NewRouter()
	NewHandler(service).RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/validation/matrices/"+run.RunID+"/abort", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("abort status=%d body=%s", response.Code, response.Body.String())
	}
	var finished MatrixRun
	if err := json.Unmarshal(response.Body.Bytes(), &finished); err != nil {
		t.Fatal(err)
	}
	if finished.State != RunCancelled || finished.Cells[0].State != CellCancelled {
		t.Fatalf("abort handler returned before owner drained: run=%s cell=%s", finished.State, finished.Cells[0].State)
	}
}

func TestHandlerExposesProviderOwnedCatalogSnapshot(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	resolver := catalogFunc(func(_ context.Context, scenario string) (CatalogSnapshot, error) {
		if scenario != "demo" {
			t.Fatalf("unexpected scenario %q", scenario)
		}
		return CatalogSnapshot{Journeys: []JourneySelection{{JourneyID: "demo/login", DisplayName: "Login", Category: "existing-bas-case", ExecutionMode: "observer"}}}, nil
	})
	NewHandler(NewService(store, Executors{}, WithCatalogResolver(resolver))).RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/validation/catalog?scenario=demo", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", response.Code, response.Body.String())
	}
	var catalog CatalogSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Journeys) != 1 || catalog.Journeys[0].Category != "existing-bas-case" {
		t.Fatalf("unexpected catalog response: %+v", catalog)
	}
}

func TestHandlerExposesTypedProfileContracts(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	NewHandler(NewService(store, Executors{})).RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/validation/profiles", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("profiles status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Profiles []struct {
			Name                 string   `json:"name"`
			RequiredCapabilities []string `json:"required_capabilities"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Profiles) < 20 {
		t.Fatalf("profile catalog has %d entries; expected complete inventory", len(payload.Profiles))
	}
	for _, profile := range payload.Profiles {
		if profile.Name == "VALIDATION_ENVIRONMENT_PROFILE_HIGH_LATENCY" && len(profile.RequiredCapabilities) != 1 {
			t.Fatalf("high latency profile lacks typed capability contract: %+v", profile)
		}
	}
}
