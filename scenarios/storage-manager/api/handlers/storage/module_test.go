package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	corestorage "github.com/vrooli/api-core/storage"
	"storage-manager/internal/providers"
)

func TestCensusEndpointReturnsClosedAccounting(t *testing.T) {
	if !hasEndpoint("storage_census", "/api/v1/census") {
		t.Fatal("census endpoint descriptor missing")
	}
	if !hasEndpoint("storage_census_history", "/api/v1/census/history") {
		t.Fatal("census history endpoint descriptor missing")
	}
	if !hasEndpoint("storage_retention_owners", "/api/v1/retention/owners") {
		t.Fatal("retention owner endpoint descriptor missing")
	}
	if !hasEndpoint("storage_adoption", "/api/v1/adoption") || !hasEndpoint("storage_infra_health", "/api/v1/infra-health/storage") {
		t.Fatal("adoption/infra-health endpoint descriptors missing")
	}
}

func TestInventoryEndpointIsAdvertised(t *testing.T) {
	if !hasEndpoint("storage_inventory", "/api/v1/storage/inventory") {
		t.Fatal("inventory endpoint descriptor missing")
	}
}

func TestInventoryEndpointReturnsTypedOwnerRecords(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"service":{"name":"demo"},"storage":{"entries":{"data":{"rung":"owned","path":"db","kind":"file","class":"data"}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(ModuleDeps{RepoRoot: root}).Mount(router)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/inventory", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var inventory corestorage.OwnerInventory
	if err := json.NewDecoder(res.Body).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Owners) != 1 || inventory.Owners[0].ID != "demo" {
		t.Fatalf("inventory = %#v", inventory)
	}
}

type fakeOllamaInventory struct {
	models []providers.OllamaModel
}

func (f fakeOllamaInventory) ListModels(context.Context) ([]providers.OllamaModel, error) {
	return f.models, nil
}
func (f fakeOllamaInventory) ListRunningModels(context.Context) ([]providers.OllamaModel, error) {
	return nil, nil
}
func (f fakeOllamaInventory) DeleteModel(context.Context, string) error { return nil }

func TestInventoryEndpointAttributesOllamaModelsAndRemainder(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "ollama", "model-policy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"roles":{"code.local":{"model":"primary:latest","fallbacks":["fallback:latest"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	modelRoot := filepath.Join(root, "models")
	if err := os.MkdirAll(modelRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelRoot, "shared-blob"), []byte("physical-remainder"), 0o644); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(ModuleDeps{
		RepoRoot: root, OllamaModelRoot: modelRoot,
		OllamaInventory: fakeOllamaInventory{models: []providers.OllamaModel{
			{Name: "orphan:latest", Digest: "sha256:orphan", Size: 5},
			{Name: "primary:latest", Digest: "sha256:primary", Size: 7},
		}},
	}).Mount(router)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/storage/inventory", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var out struct {
		OllamaModels OllamaStorageInventory `json:"ollama_models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.OllamaModels.Models) != 2 || out.OllamaModels.Models[0].Name != "orphan:latest" {
		t.Fatalf("models = %+v", out.OllamaModels.Models)
	}
	if out.OllamaModels.Models[0].PolicyReachable || !out.OllamaModels.Models[1].PolicyReachable {
		t.Fatalf("reachability = %+v", out.OllamaModels.Models)
	}
	if out.OllamaModels.UnattributedBytes == 0 || len(out.OllamaModels.UnattributedPaths) != 1 {
		t.Fatalf("physical remainder = %+v", out.OllamaModels)
	}
}

func TestPlacementAPIIsPreviewFirstAndCopyVerified(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(ModuleDeps{RepoRoot: root}).Mount(router)
	body := bytes.NewBufferString(`{"entry":"cache","source":"` + source + `","destination":"` + destination + `"}`)
	planReq := httptest.NewRequest(http.MethodPost, "/api/v1/placement/plan", body)
	planRes := httptest.NewRecorder()
	router.ServeHTTP(planRes, planReq)
	if planRes.Code != http.StatusOK {
		t.Fatalf("plan status = %d, body=%s", planRes.Code, planRes.Body.String())
	}
	var plan struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(planRes.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	applyBody := bytes.NewBufferString(`{"plan_id":"` + plan.ID + `","approved":true}`)
	applyReq := httptest.NewRequest(http.MethodPost, "/api/v1/placement/migrate", applyBody)
	applyRes := httptest.NewRecorder()
	router.ServeHTTP(applyRes, applyReq)
	if applyRes.Code != http.StatusOK {
		t.Fatalf("apply status = %d, body=%s", applyRes.Code, applyRes.Body.String())
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source remains: %v", err)
	}
}

func hasEndpoint(id, path string) bool {
	for _, endpoint := range Endpoints {
		if endpoint.ID == id && strings.Contains(endpoint.Path, path) {
			return true
		}
	}
	return false
}
