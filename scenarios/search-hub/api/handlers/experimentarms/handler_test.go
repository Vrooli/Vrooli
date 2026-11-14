package experimentarms

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	corestorage "github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
	internal "search-hub/internal/experimentarms"
)

func TestEmbeddingArmHTTPCreateReadAndCASLifecycle(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "http-arms.sqlite")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	namespace, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{FallbackScenario: "search-hub"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := internal.NewForNamespace(db, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	module := Module(store)
	module.Mount(router)

	create := `{"experiment_id":"trial-1","arm_id":"arm-1","provider_id":"docs","caller_id":"search-hub","build_id":"build-9","recipe":{"schema_version":"embedding-recipe-v1","model_id":"e5-small","model_revision":"commit-17","artifact_sha256":"` + strings.Repeat("a", 64) + `","tokenizer_revision":"tok-1","runtime":"onnx","runtime_build":"ort-1","precision":"fp32","query_instruction":"query: ","document_instruction":"passage: ","pooling":"mean","normalization":"l2","dimensions":384,"projection":"none","output_dtype":"float32","output_quantization":"none","truncation":"512","chunker_version":"chunk-v2","composer_version":"compose-v1"},"evidence":[{"kind":"e5-source-run","uri":"test-genie://search-hub/run-1","sha256":"` + strings.Repeat("b", 64) + `"},{"kind":"w10-caller-build","uri":"w10://search-hub/build","sha256":"` + strings.Repeat("c", 64) + `"}]}`
	response := callJSONAsOperator(t, router, http.MethodPost, basePath, create)
	if response.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var arm internal.Arm
	if err := json.Unmarshal(response.Body.Bytes(), &arm); err != nil {
		t.Fatal(err)
	}
	if arm.State != internal.StateDiscovered || arm.Version != 1 || arm.CallerID != "search-hub" || arm.OwnerID != namespace.Root()+".embedding-fleet" || len(arm.Evidence) != 2 {
		t.Fatalf("unexpected created arm response: %+v", arm)
	}

	read := callJSON(t, router, http.MethodGet, basePath+"/arm-1", "")
	if read.Code != http.StatusOK || !bytes.Contains(read.Body.Bytes(), []byte(arm.RecipeID)) {
		t.Fatalf("read failed or omitted recipe identity: status=%d body=%s", read.Code, read.Body.String())
	}
	transition := `{"expected_state":"discovered","expected_version":1,"to":"capability_verified","idempotency_key":"capability-1","receipt":{"kind":"capability","uri":"test://capability-1","sha256":"` + strings.Repeat("e", 64) + `"}}`
	changed := callJSONAsOperator(t, router, http.MethodPost, basePath+"/arm-1/transitions", transition)
	if changed.Code != http.StatusOK {
		t.Fatalf("transition status=%d body=%s", changed.Code, changed.Body.String())
	}
	conflict := callJSONAsOperator(t, router, http.MethodPost, basePath+"/arm-1/transitions", transition)
	if conflict.Code != http.StatusOK {
		t.Fatalf("idempotent transition retry status=%d body=%s", conflict.Code, conflict.Body.String())
	}
	stale := `{"expected_state":"discovered","expected_version":1,"to":"capability_verified","idempotency_key":"stale-2","receipt":{"kind":"capability","uri":"test://capability-1","sha256":"` + strings.Repeat("e", 64) + `"}}`
	staleResponse := callJSONAsOperator(t, router, http.MethodPost, basePath+"/arm-1/transitions", stale)
	if staleResponse.Code != http.StatusConflict {
		t.Fatalf("expected stale transition HTTP conflict, got %d: %s", staleResponse.Code, staleResponse.Body.String())
	}
}

func TestEmbeddingArmHTTPRejectsMalformedRecipeAndUnknownFields(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "http-bad-arms.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := internal.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(store).Mount(router)
	unknown := callJSON(t, router, http.MethodPost, basePath, `{"unknown":true}`)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown body field rejection, got %d", unknown.Code)
	}
	spoofedOwner := callJSON(t, router, http.MethodPost, basePath, `{"owner_id":"agent-manager.runs"}`)
	if spoofedOwner.Code != http.StatusBadRequest {
		t.Fatalf("arm creation must reject caller-supplied owner namespace, got %d", spoofedOwner.Code)
	}
	badRecipe := `{"experiment_id":"e","arm_id":"a","owner_id":"o","provider_id":"p","caller_id":"c","build_id":"b","recipe":{"model_id":"incomplete"},"evidence":[]}`
	bad := callJSONAsOperator(t, router, http.MethodPost, basePath, badRecipe)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected malformed recipe rejection, got %d: %s", bad.Code, bad.Body.String())
	}
}

func TestEmbeddingGenerationRouteFailsClosedWithoutLifecycleRuntime(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "closed-generation.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := internal.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(store).Mount(router)
	response := callJSONAsOperator(t, router, http.MethodPost, basePath+"/arm-1/generation", `{"recipe":{"model_id":"caller-model"},"peak_bytes":1,"snapshot_id":"caller-snapshot"}`)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "unavailable") {
		t.Fatalf("generation route must remain closed when lifecycle runtime is absent: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEmbeddingArmMutationsRequireVerifiedOperationAuthority(t *testing.T) {
	router := mux.NewRouter()
	Module(nil).Mount(router)
	body := `{"experiment_id":"trial","arm_id":"arm-1","provider_id":"docs","caller_id":"search-hub","build_id":"build","recipe":{"schema_version":"embedding-recipe-v1","model_id":"e5-small","model_revision":"commit-17","artifact_sha256":"` + strings.Repeat("a", 64) + `","tokenizer_revision":"tok-1","runtime":"onnx","runtime_build":"ort-1","precision":"fp32","query_instruction":"query: ","document_instruction":"passage: ","pooling":"mean","normalization":"l2","dimensions":384,"projection":"none","output_dtype":"float32","output_quantization":"none","truncation":"512","chunker_version":"chunk-v2","composer_version":"compose-v1"},"evidence":[]}`

	unauthenticated := callJSON(t, router, http.MethodPost, basePath, body)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated create status=%d body=%s", unauthenticated.Code, unauthenticated.Body.String())
	}
	wrongGrant := callJSONAsAgent(t, router, http.MethodPost, basePath, body, "embedding_experiment.read")
	if wrongGrant.Code != http.StatusForbidden {
		t.Fatalf("create with unrelated operation grant status=%d body=%s", wrongGrant.Code, wrongGrant.Body.String())
	}
}

func TestPromotionAndRollbackDecisionsRequireIndependentHuman(t *testing.T) {
	for _, to := range []string{"promoting", "rolling_back", "deleted"} {
		t.Run(to, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, basePath+"/arm-1/transitions", nil)
			request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{
				Kind: identity.ActorAgent, Subject: "agent-run", Verified: true, Scopes: []string{writeCapability},
				Grant: &identity.OperationGrant{ID: "grant", Approver: "operator", Operations: []string{writeOperation}, Targets: []string{"search-hub:embedding-arm/arm-1"}, ExpiresAt: time.Now().Add(time.Hour)},
			}))
			response := httptest.NewRecorder()
			if authorizeMutation(response, request, "arm-1", requiresIndependentHuman(internal.State(to))) {
				t.Fatal("agent grant authorized a human-only lifecycle decision")
			}
			if response.Code != http.StatusForbidden {
				t.Fatalf("decision status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func callJSON(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func callJSONAsOperator(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator:test", Verified: true, Scopes: []string{writeCapability},
	}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func callJSONAsAgent(t *testing.T, router http.Handler, method, path, body, operation string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "agent-run", Verified: true, Scopes: []string{writeCapability},
		Grant: &identity.OperationGrant{ID: "grant", Approver: "operator", Operations: []string{operation}, Targets: []string{"search-hub:embedding-arm/arm-1"}, ExpiresAt: time.Now().Add(time.Hour)},
	}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
