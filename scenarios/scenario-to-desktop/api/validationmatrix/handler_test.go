package validationmatrix

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
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
