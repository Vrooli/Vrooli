package embedding

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/cli-core/cliapp"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
)

// TestRequirePassingCompareRejectsMissingOrNonPassingVerdicts [REQ:REQ-P1-018]
func TestRequirePassingCompareRejectsMissingOrNonPassingVerdicts(t *testing.T) {
	for _, verdict := range []string{"", "withheld", "fail"} {
		if err := requirePassingCompare(migrationState{CompareVerdict: verdict}); err == nil {
			t.Fatalf("verdict %q was accepted without a passing comparison", verdict)
		}
	}
	if err := requirePassingCompare(migrationState{CompareVerdict: "pass"}); err != nil {
		t.Fatalf("passing comparison rejected: %v", err)
	}
}

func TestAbortRefusesUnsupportedDeletion(t *testing.T) {
	_, err := (&handlers{}).abortCall(nil)
	if err == nil || !strings.Contains(err.Error(), "no data was changed") {
		t.Fatalf("abort error=%v, want explicit no-mutation refusal", err)
	}
}

type migrationRegistryClient struct {
	submission *registryv1.ExecuteEmbeddingMigrationResponse
	status     string
	requests   []*registryv1.ExecuteEmbeddingMigrationRequest
}

func (f *migrationRegistryClient) ExecuteEmbeddingMigration(_ context.Context, req *connect.Request[registryv1.ExecuteEmbeddingMigrationRequest]) (*connect.Response[registryv1.ExecuteEmbeddingMigrationResponse], error) {
	f.requests = append(f.requests, req.Msg)
	if req.Msg.GetAction() == "status" {
		return connect.NewResponse(&registryv1.ExecuteEmbeddingMigrationResponse{JobId: req.Msg.GetJobId(), State: f.status}), nil
	}
	return connect.NewResponse(f.submission), nil
}

func (*migrationRegistryClient) RegisterProvider(context.Context, *connect.Request[registryv1.RegisterProviderRequest]) (*connect.Response[registryv1.RegisterProviderResponse], error) {
	return nil, nil
}
func (*migrationRegistryClient) ListProviders(context.Context, *connect.Request[registryv1.ListProvidersRequest]) (*connect.Response[registryv1.ListProvidersResponse], error) {
	return nil, nil
}
func (*migrationRegistryClient) ListMaturityTargets(context.Context, *connect.Request[registryv1.ListMaturityTargetsRequest]) (*connect.Response[registryv1.ListMaturityTargetsResponse], error) {
	return nil, nil
}
func (*migrationRegistryClient) DeregisterProvider(context.Context, *connect.Request[registryv1.DeregisterProviderRequest]) (*connect.Response[registryv1.DeregisterProviderResponse], error) {
	return nil, nil
}

var _ registry_v1connect.RegistryServiceClient = (*migrationRegistryClient)(nil)

func TestApplyShadowRequiresTerminalProviderSuccessBeforeReady(t *testing.T) {
	tests := []struct {
		name            string
		submission      *registryv1.ExecuteEmbeddingMigrationResponse
		status          string
		wantErr         bool
		wantState       string
		wantStatusPolls int
	}{
		{name: "succeeded job becomes ready", submission: &registryv1.ExecuteEmbeddingMigrationResponse{JobId: "job-1"}, status: "succeeded", wantState: "shadow_ready", wantStatusPolls: 1},
		{name: "missing job id fails closed", submission: &registryv1.ExecuteEmbeddingMigrationResponse{State: "building"}, wantErr: true, wantState: "shadow_build_failed"},
		{name: "unknown job state never becomes ready", submission: &registryv1.ExecuteEmbeddingMigrationResponse{JobId: "job-2"}, status: "shadow_ready", wantErr: true, wantState: "shadow_status_unknown", wantStatusPolls: 1},
		{name: "failed job is retained as failed", submission: &registryv1.ExecuteEmbeddingMigrationResponse{JobId: "job-3"}, status: "failed", wantErr: true, wantState: "shadow_build_failed", wantStatusPolls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SEARCH_HUB_EMBEDDING_STATE_FILE", t.TempDir()+"/migration.json")
			recipeInput := aisearch.EmbeddingRecipeInput{SchemaVersion: "embedding-recipe-v1", ModelID: "new@sha256:abc", ModelRevision: "revision-1", ArtifactSHA256: strings.Repeat("a", 64), TokenizerRevision: "tokenizer-1", Runtime: "onnx", RuntimeBuild: "runtime-1", Precision: "fp32", QueryInstruction: "query", DocumentInstruction: "document", Pooling: "mean", Normalization: "l2", Dimensions: 768, OutputDType: "float32", Truncation: "512", ChunkerVersion: "chunk-v1", ComposerVersion: "composer-v1"}
			recipeBytes, err := json.Marshal(recipeInput)
			if err != nil {
				t.Fatal(err)
			}
			recipeFile := filepath.Join(t.TempDir(), "recipe.json")
			if err := os.WriteFile(recipeFile, recipeBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			client := &migrationRegistryClient{submission: tt.submission, status: tt.status}
			h := &handlers{
				client: client,
				inventory: func(context.Context, string, string) ([]aisearch.EmbeddingInventoryEntry, error) {
					return []aisearch.EmbeddingInventoryEntry{{Store: aisearch.EmbeddingStore{ID: "qdrant:live", Kind: aisearch.EmbeddingStoreQdrant, Collection: "live"}, Metadata: aisearch.EmbeddingMetadata{Model: "old", Role: "embedding.default", Dimensions: 768, PolicySchemaVersion: "v1"}}}, nil
				},
				resolveTarget: func(context.Context, string, string, string) (aisearch.EmbeddingMetadata, error) {
					return aisearch.EmbeddingMetadata{Model: "new@sha256:abc", Role: "embedding.default", Dimensions: 768, PolicySchemaVersion: "v2"}, nil
				},
			}
			ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
				Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
					{Name: "shadow", Bool: true},
					{Name: "provider-id"},
					{Name: "collection"},
					{Name: "model"},
					{Name: "role"},
					{Name: "dimensions"},
					{Name: "recipe-file"},
				}},
				BoolFlags: map[string]bool{"shadow": true},
				Flags:     map[string]string{"provider-id": "architecture-cartographer.domain-map", "collection": "live", "recipe-file": recipeFile},
			})
			got, err := h.applyShadowCall(ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("applyShadowCall() error=%v, wantErr=%v", err, tt.wantErr)
			}
			state, loadErr := loadState()
			if loadErr != nil || state.State != tt.wantState {
				t.Fatalf("persisted state=%q err=%v, want %q", state.State, loadErr, tt.wantState)
			}
			if len(client.requests) != 1+tt.wantStatusPolls {
				t.Fatalf("provider requests=%d, want %d", len(client.requests), 1+tt.wantStatusPolls)
			}
			if client.requests[0].GetAction() != "shadow" || client.requests[0].GetEmbeddingModel() != "new@sha256:abc" || client.requests[0].GetEmbeddingDimensions() != 768 || client.requests[0].GetEmbeddingRecipeId() == "" || client.requests[0].GetEmbeddingRecipeCanonical() == "" {
				t.Fatalf("shadow request omitted target contract: %+v", client.requests[0])
			}
			if _, err := aisearch.EmbeddingRecipeFromCanonical([]byte(client.requests[0].GetEmbeddingRecipeCanonical()), client.requests[0].GetEmbeddingRecipeId()); err != nil {
				t.Fatalf("shadow request recipe identity is not verifiable: %v", err)
			}
			if state.RecipeID != client.requests[0].GetEmbeddingRecipeId() || state.RecipeCanonical != client.requests[0].GetEmbeddingRecipeCanonical() {
				t.Fatalf("migration state did not retain submitted recipe identity: state=%+v request=%+v", state, client.requests[0])
			}
			if !tt.wantErr && (got.Status != "ready" || got.Migration == nil || got.Migration.State != "shadow_ready") {
				t.Fatalf("successful result=%+v; want ready after succeeded provider status", got)
			}
		})
	}
}

func TestReadRecipeFileRequiresCompleteValidatedInput(t *testing.T) {
	if _, err := readRecipeFile(""); err == nil {
		t.Fatal("empty recipe path was accepted")
	}
	path := filepath.Join(t.TempDir(), "recipe.json")
	if err := os.WriteFile(path, []byte(`{"model_id":"model"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRecipeFile(path); err == nil {
		t.Fatal("incomplete recipe was accepted")
	}
	large := aisearch.EmbeddingRecipeInput{SchemaVersion: "embedding-recipe-v1", ModelID: "model@sha256:abc", ModelRevision: "revision-1", ArtifactSHA256: strings.Repeat("a", 64), TokenizerRevision: "tokenizer-1", Runtime: "onnx", RuntimeBuild: "runtime-1", Precision: "fp32", QueryInstruction: strings.Repeat("q", 17*1024), DocumentInstruction: "document", Pooling: "mean", Normalization: "l2", Dimensions: 384, OutputDType: "float32", Truncation: "512", ChunkerVersion: "chunk-v1", ComposerVersion: "composer-v1"}
	largeBytes, err := json.Marshal(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, largeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRecipeFile(path); err == nil || !strings.Contains(err.Error(), "16 KiB") {
		t.Fatalf("oversized recipe error=%v, want the migration size limit", err)
	}
}

func TestBindTargetRecipeRejectsPolicyMismatch(t *testing.T) {
	recipe, err := aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{SchemaVersion: "embedding-recipe-v1", ModelID: "model@sha256:abc", ModelRevision: "revision-1", ArtifactSHA256: strings.Repeat("a", 64), TokenizerRevision: "tokenizer-1", Runtime: "onnx", RuntimeBuild: "runtime-1", Precision: "fp32", QueryInstruction: "query", DocumentInstruction: "document", Pooling: "mean", Normalization: "l2", Dimensions: 384, OutputDType: "float32", Truncation: "512", ChunkerVersion: "chunk-v1", ComposerVersion: "composer-v1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bindTargetRecipe(aisearch.EmbeddingMetadata{Model: "different-model", Dimensions: 384}, recipe); err == nil {
		t.Fatal("recipe for a different model was accepted")
	}
	if _, err := bindTargetRecipe(aisearch.EmbeddingMetadata{Model: "model@sha256:abc", Dimensions: 768}, recipe); err == nil {
		t.Fatal("recipe with a different dimension was accepted")
	}
	bound, err := bindTargetRecipe(aisearch.EmbeddingMetadata{Model: "model@sha256:abc", Dimensions: 384}, recipe)
	if err != nil {
		t.Fatalf("matching recipe rejected: %v", err)
	}
	if bound.RecipeID != recipe.ID() || bound.RecipeCanonical != string(recipe.Canonical()) {
		t.Fatalf("recipe identity was not bound: %+v", bound)
	}
}
