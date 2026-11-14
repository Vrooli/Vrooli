package embedding

import (
	"context"
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
				}},
				BoolFlags: map[string]bool{"shadow": true},
				Flags:     map[string]string{"provider-id": "architecture-cartographer.domain-map", "collection": "live"},
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
			if client.requests[0].GetAction() != "shadow" || client.requests[0].GetEmbeddingModel() != "new@sha256:abc" || client.requests[0].GetEmbeddingDimensions() != 768 {
				t.Fatalf("shadow request omitted target contract: %+v", client.requests[0])
			}
			if !tt.wantErr && (got.Status != "ready" || got.Migration == nil || got.Migration.State != "shadow_ready") {
				t.Fatalf("successful result=%+v; want ready after succeeded provider status", got)
			}
		})
	}
}
