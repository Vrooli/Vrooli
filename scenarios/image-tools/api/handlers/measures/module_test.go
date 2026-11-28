package measures

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internalmeasures "image-tools/internal/measures"

	"github.com/gorilla/mux"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestModelProfilesEndpointServesRecorderData(t *testing.T) {
	store := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), store, apidb.SchemaProviderFunc(internalmeasures.Schema)); err != nil {
		t.Fatal(err)
	}
	recorder := internalmeasures.NewRecorder(store)
	if err := recorder.RecordModelProfile(context.Background(), internalmeasures.ModelProfile{Operation: "background_removal", ModelID: "isnet", Profile: "warm", Status: "measured", DurationMS: 80, PeakMemoryB: 1000, Device: "cpu"}); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(recorder).Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/measures/models?operation=background_removal", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"model_id":"isnet"`) || !strings.Contains(response.Body.String(), `"latency_p95_ms":80`) {
		t.Fatalf("response %d: %s", response.Code, response.Body.String())
	}
}

func TestModelUsageEndpointServesSuccessfulModelUse(t *testing.T) {
	store := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), store, apidb.SchemaProviderFunc(internalmeasures.Schema)); err != nil {
		t.Fatal(err)
	}
	recorder := internalmeasures.NewRecorder(store)
	if err := recorder.RecordJobTrace(context.Background(), internalmeasures.JobTrace{JobID: "job-1", Operation: "caption", ModelID: "smolvlm-256m", Tier: "local-cpu", State: "succeeded"}); err != nil {
		t.Fatal(err)
	}
	if err := recorder.RecordJobTrace(context.Background(), internalmeasures.JobTrace{JobID: "job-2", Operation: "caption", ModelID: "smolvlm-256m", Tier: "local-cpu", State: "failed"}); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(recorder).Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/measures/usage?model_id=smolvlm-256m", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"successful_jobs":1`) || !strings.Contains(response.Body.String(), `"last_operation":"caption"`) {
		t.Fatalf("response %d: %s", response.Code, response.Body.String())
	}
}

func TestModuleDeclaresQueryEndpoint(t *testing.T) {
	if len(Endpoints) != 2 || Endpoints[0].ID != "model_usage" || Endpoints[0].Path != "/api/v1/measures/usage" || Endpoints[1].ID != "model_profiles" || Endpoints[1].Path != "/api/v1/measures/models" {
		t.Fatalf("unexpected endpoint descriptors: %+v", Endpoints)
	}
	endpoint := Endpoints[0]
	if endpoint.RESTException == nil || endpoint.RESTException.Reason != "ops_probe" {
		t.Fatalf("query endpoint must declare its deliberate REST transport: %+v", endpoint.RESTException)
	}
	if endpoint.RESTException.ProtoPayloads == nil || endpoint.RESTException.ProtoPayloads.Response.Conformance != "external_shape" {
		t.Fatalf("query response contract missing: %+v", endpoint.RESTException)
	}
}

type failingMeasuresDB struct{}

func (failingMeasuresDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, errors.New("database unavailable")
}

func (failingMeasuresDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("database unavailable")
}

func TestModelProfilesQueryUsesSharedErrorEnvelope(t *testing.T) {
	router := mux.NewRouter()
	Module(internalmeasures.NewRecorder(failingMeasuresDB{})).Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/measures/models", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type=%q, want JSON error envelope", got)
	}
	var envelope sharedv1.ErrorEnvelope
	if err := protojson.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode shared error envelope: %v; body=%s", err, response.Body.String())
	}
	if envelope.GetCode() != "internal" {
		t.Fatalf("error code=%q, want internal", envelope.GetCode())
	}
}
