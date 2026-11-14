package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"storage-manager/internal/census"
	"storage-manager/internal/providers"

	"github.com/gorilla/mux"
	corestorage "github.com/vrooli/api-core/storage"
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
	if len(inventory.Owners) != 3 || ownerByID(inventory.Owners, "demo") == nil {
		t.Fatalf("inventory = %#v", inventory)
	}
}

func ownerByID(owners []corestorage.OwnerManifest, id string) *corestorage.OwnerManifest {
	for i := range owners {
		if owners[i].ID == id {
			return &owners[i]
		}
	}
	return nil
}

type fakeOllamaInventory struct {
	models []providers.OllamaModel
}

type fakeQdrantReader struct {
	inspection providers.QdrantGenerationInspection
	err        error
}

func (f fakeQdrantReader) InspectQdrantGenerations(context.Context) (providers.QdrantGenerationInspection, error) {
	return f.inspection, f.err
}

func TestQdrantEndpointBoundsAndClassifiesOwnerGenerations(t *testing.T) {
	router := mux.NewRouter()
	Module(ModuleDeps{RepoRoot: t.TempDir(), QdrantReader: fakeQdrantReader{inspection: providers.QdrantGenerationInspection{
		Owner: "agent-manager.runs", Namespace: "conversation-search", Alias: "agent-manager_conversation-search", ActiveCollection: "active",
		Generations: []providers.QdrantGeneration{
			{CollectionName: "active", State: "active", Bytes: 10},
			{CollectionName: "old", State: "retired", Bytes: 20},
			{CollectionName: "unknown", State: "quarantined", Bytes: 30},
		},
		Protected: []string{"active"}, Quarantined: []string{"unknown"}, Eligible: []string{"old"},
	}}}).Mount(router)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/storage/qdrant?state=retired&limit=1", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var out qdrantStorageReport
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Generations) != 1 || out.Generations[0].CollectionName != "old" {
		t.Fatalf("generations = %+v", out.Generations)
	}
	if out.DispositionBytes[census.DispositionOwnedManaged] != 20 {
		t.Fatalf("dispositions = %+v", out.DispositionBytes)
	}
}

func (f fakeOllamaInventory) ListModels(context.Context) ([]providers.OllamaModel, error) {
	return f.models, nil
}

func (f fakeOllamaInventory) ListRunningModels(context.Context) ([]providers.OllamaModel, error) {
	return nil, nil
}

type errorOllamaInventory struct{}

func (errorOllamaInventory) ListModels(context.Context) ([]providers.OllamaModel, error) {
	return nil, errors.New("Ollama inventory unavailable")
}

func (errorOllamaInventory) ListRunningModels(context.Context) ([]providers.OllamaModel, error) {
	return nil, errors.New("Ollama inventory unavailable")
}

func TestInventoryEndpointAttributesOllamaModelsAndLogicalFileRemainder(t *testing.T) {
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
	provenance := `{"schema_version":1,"models":[` +
		`{"name":"primary:latest","digest":"sha256:primary","registry":"registry.ollama.ai","repository":"library/primary","tag":"latest","manifest_digest":"sha256:primary","verified_at":"2026-11-11T17:30:00Z"},` +
		`{"name":"orphan:latest","digest":"sha256:old","registry":"registry.ollama.ai","repository":"library/orphan","tag":"latest","manifest_digest":"sha256:old","verified_at":"2026-11-11T17:30:00Z"}]}`
	if err := os.WriteFile(filepath.Join(root, "ollama-install-provenance.json"), []byte(provenance), 0o600); err != nil {
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
	if !out.OllamaModels.TotalModelBytesKnown || out.OllamaModels.TotalModelBytes != 12 {
		t.Fatalf("model logical total = known:%v bytes:%d, want known:12", out.OllamaModels.TotalModelBytesKnown, out.OllamaModels.TotalModelBytes)
	}
	if out.OllamaModels.Models[0].PolicyReachable || !out.OllamaModels.Models[1].PolicyReachable {
		t.Fatalf("reachability = %+v", out.OllamaModels.Models)
	}
	if out.OllamaModels.Models[0].InstallProvenanceStatus != "unknown" || out.OllamaModels.Models[1].InstallProvenanceStatus != "matched" {
		t.Fatalf("digest-matched provenance status = %+v", out.OllamaModels.Models)
	}
	for _, model := range out.OllamaModels.Models {
		if model.Regenerable {
			t.Fatalf("provenance was incorrectly treated as retention authorization: %+v", model)
		}
	}
	if !strings.Contains(out.OllamaModels.Models[1].RegenerableReason, "same-artifact restoration and retention authorization are not proven") {
		t.Fatalf("matched provenance overclaimed regenerability: %+v", out.OllamaModels.Models[1])
	}
	if out.OllamaModels.PhysicalBytesKnown || out.OllamaModels.PhysicalBytes != nil || out.OllamaModels.UnattributedBytes != nil {
		t.Fatalf("inventory claimed an allocated-byte measurement: %+v", out.OllamaModels)
	}
	if !out.OllamaModels.LogicalFileLengthBytesKnown || !out.OllamaModels.UnattributedLogicalFileLengthKnown || out.OllamaModels.LogicalFileLengthBytes != int64(len("physical-remainder")) || out.OllamaModels.UnattributedLogicalFileLengthBytes != int64(len("physical-remainder"))-12 || len(out.OllamaModels.UnattributedPaths) != 1 {
		t.Fatalf("logical file-length remainder = %+v", out.OllamaModels)
	}
	if !strings.Contains(out.OllamaModels.AccountingNote, "allocated blocks") {
		t.Fatalf("missing physical-accounting limitation: %+v", out.OllamaModels)
	}
}

func TestInventoryEndpointMarksPolicyReachabilityUnknownWhenPolicyIsUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name       string
		policyBody string
		writeFile  bool
	}{
		{name: "missing policy"},
		{name: "malformed policy", policyBody: `{"roles":`, writeFile: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			policyPath := filepath.Join(root, "resources", "ollama", "model-policy.json")
			if tc.writeFile {
				if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(policyPath, []byte(tc.policyBody), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			router := mux.NewRouter()
			Module(ModuleDeps{
				RepoRoot: root, OllamaModelRoot: t.TempDir(),
				OllamaInventory: fakeOllamaInventory{models: []providers.OllamaModel{{Name: "installed:latest", Size: 10}}},
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
			if out.OllamaModels.PolicyReachabilityKnown || out.OllamaModels.RoleAccountingKnown {
				t.Fatalf("unavailable policy was treated as known: %+v", out.OllamaModels)
			}
			if len(out.OllamaModels.Models) != 1 || out.OllamaModels.Models[0].PolicyReachable || out.OllamaModels.Models[0].PolicyReachabilityKnown {
				t.Fatalf("row made a false negative reachability claim: %+v", out.OllamaModels.Models)
			}
		})
	}
}

func TestInventoryEndpointMarksNormalizedDuplicateModelTagsAmbiguous(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "resources", "ollama", "model-policy.json")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, []byte(`{"roles":{"chat":{"model":"model:latest"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(ModuleDeps{
		RepoRoot: root, OllamaModelRoot: t.TempDir(),
		OllamaInventory: fakeOllamaInventory{models: []providers.OllamaModel{
			{Name: " model:latest ", Size: 10},
			{Name: "model:latest", Size: 100},
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
	if out.OllamaModels.TotalModelBytesKnown || out.OllamaModels.RoleAccountingKnown || !out.OllamaModels.PolicyReachabilityKnown {
		t.Fatalf("duplicate normalized tag accounting/reachability states = %+v", out.OllamaModels)
	}
	if len(out.OllamaModels.Roles) != 1 || out.OllamaModels.Roles[0].LogicalModelBytesKnown || len(out.OllamaModels.Roles[0].AmbiguousModels) != 1 || out.OllamaModels.Roles[0].AmbiguousModels[0] != "model:latest" {
		t.Fatalf("duplicate model size did not remain ambiguous in role attribution: %+v", out.OllamaModels.Roles)
	}
	for _, model := range out.OllamaModels.Models {
		if model.Name != "model:latest" || !model.PolicyReachable || !model.PolicyReachabilityKnown {
			t.Fatalf("normalized tag or policy reachability was not exposed consistently: %+v", model)
		}
	}
}

func TestOllamaInventoryLogicalTotalsRejectNegativeAndOverflow(t *testing.T) {
	max := int64(^uint64(0) >> 1)
	for _, models := range [][]providers.OllamaModel{
		{{Name: "negative:latest", Size: -1}},
		{{Name: "large-a:latest", Size: max}, {Name: "large-b:latest", Size: 1}},
	} {
		root := t.TempDir()
		out := buildOllamaStorageInventory(context.Background(), fakeOllamaInventory{models: models}, root, filepath.Join(root, "model-policy.json"))
		if out.TotalModelBytesKnown || out.InventoryError == "" {
			t.Fatalf("invalid logical model total was accepted: %+v", out)
		}
		if !out.LogicalFileLengthBytesKnown || out.UnattributedLogicalFileLengthKnown {
			t.Fatalf("invalid model total did not keep remainder unknown: %+v", out)
		}
	}

	unknownRoot := buildOllamaStorageInventory(context.Background(), fakeOllamaInventory{}, filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "model-policy.json"))
	if unknownRoot.LogicalFileLengthBytesKnown || unknownRoot.UnattributedLogicalFileLengthKnown || unknownRoot.InventoryError == "" {
		t.Fatalf("unreadable root was reported as zero bytes: %+v", unknownRoot)
	}
	if unknownRoot.PhysicalBytesKnown || unknownRoot.PhysicalBytes != nil {
		t.Fatalf("unreadable root exposed physical bytes: %+v", unknownRoot)
	}
}

func TestOllamaRoleAccountingIsDeduplicatedPerRoleAndUnknownForMissingModels(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "model-policy.json")
	policy := `{"roles":{"chat":{"model":"shared:latest","fallbacks":["chat-only:latest","shared:latest"]},"embed":{"model":"shared:latest"},"missing":{"model":"not-installed:latest"}}}`
	if err := os.WriteFile(policyPath, []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	models := []providers.OllamaModel{
		{Name: "shared:latest", Size: 100},
		{Name: "chat-only:latest", Size: 25},
	}
	roles, err := buildOllamaRoleAccounting(policyPath, models)
	if err == nil {
		t.Fatal("missing installed role model was treated as complete accounting")
	}
	if len(roles) != 3 || roles[0].Name != "chat" || roles[1].Name != "embed" || roles[2].Name != "missing" {
		t.Fatalf("roles are not complete and stable-sorted: %+v", roles)
	}
	if !roles[0].LogicalModelBytesKnown || roles[0].LogicalModelBytes != 125 || len(roles[0].Models) != 2 {
		t.Fatalf("duplicate same-role tag was not counted once: %+v", roles[0])
	}
	if !roles[1].LogicalModelBytesKnown || roles[1].LogicalModelBytes != 100 {
		t.Fatalf("shared tag was not attributed independently to embed: %+v", roles[1])
	}
	if roles[2].LogicalModelBytesKnown || len(roles[2].MissingModels) != 1 || roles[2].MissingModels[0] != "not-installed:latest" {
		t.Fatalf("missing model was not explicit/unknown: %+v", roles[2])
	}
}

func TestOllamaInventoryUnavailableDoesNotReportKnownZeroTotals(t *testing.T) {
	root := t.TempDir()
	out := buildOllamaStorageInventory(context.Background(), errorOllamaInventory{}, root, filepath.Join(root, "model-policy.json"))
	if out.TotalModelBytesKnown || out.LogicalFileLengthBytesKnown || out.UnattributedLogicalFileLengthKnown || out.InventoryError == "" {
		t.Fatalf("unavailable model inventory looked like known zero accounting: %+v", out)
	}
}

func TestAddNonnegativeBytesSaturatesAndRejectsInvalidValues(t *testing.T) {
	max := int64(^uint64(0) >> 1)
	if got, ok := addNonnegativeBytes(max-2, 3); got != max || ok {
		t.Fatalf("overflow = (%d, %v), want (max, false)", got, ok)
	}
	if got, ok := addNonnegativeBytes(10, -1); got != 10 || ok {
		t.Fatalf("negative = (%d, %v), want (10, false)", got, ok)
	}
	if got, ok := addNonnegativeBytes(10, 5); got != 15 || !ok {
		t.Fatalf("valid sum = (%d, %v), want (15, true)", got, ok)
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
