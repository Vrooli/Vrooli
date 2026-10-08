package embedding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/cli-core/cliapp"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client registryconnect.RegistryServiceClient
}

type result struct {
	Operation string                             `json:"operation"`
	Status    string                             `json:"status"`
	Message   string                             `json:"message,omitempty"`
	Inventory []aisearch.EmbeddingInventoryEntry `json:"inventory,omitempty"`
	Plan      *aisearch.RetargetPlan             `json:"plan,omitempty"`
	Migration *migrationState                    `json:"migration,omitempty"`
	NextSteps []string                           `json:"next_steps,omitempty"`
}

type migrationState struct {
	ProviderID       string                `json:"provider_id"`
	JobID            string                `json:"job_id,omitempty"`
	LiveCollection   string                `json:"live_collection"`
	ShadowCollection string                `json:"shadow_collection"`
	OldCollection    string                `json:"old_collection,omitempty"`
	ActiveCollection string                `json:"active_collection"`
	Plan             aisearch.RetargetPlan `json:"plan"`
	CompareVerdict   string                `json:"compare_verdict,omitempty"`
	State            string                `json:"state"`
	UpdatedAt        string                `json:"updated_at"`
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: registryconnect.NewRegistryServiceClient(httpClient, baseURL)}
}

func (h *handlers) inventoryCall(ctx cliapp.OperationContext) (result, error) {
	entries, err := aisearch.InventoryQdrantCollections(context.Background(), qdrantURL(), os.Getenv("QDRANT_API_KEY"))
	if err != nil {
		return result{}, err
	}
	return result{Operation: "embedding inventory", Status: "ok", Inventory: entries, Message: fmt.Sprintf("inventoried %d Qdrant collection(s)", len(entries))}, nil
}

func (h *handlers) planCall(ctx cliapp.OperationContext) (result, error) {
	entries, err := aisearch.InventoryQdrantCollections(context.Background(), qdrantURL(), os.Getenv("QDRANT_API_KEY"))
	if err != nil {
		return result{}, err
	}
	target, err := resolveTarget(context.Background(), ctx.Flag("model"), ctx.Flag("role"), ctx.Flag("dimensions"))
	if err != nil {
		return result{}, err
	}
	stores := make([]aisearch.EmbeddingStore, 0, len(entries))
	var old aisearch.EmbeddingMetadata
	for _, entry := range entries {
		stores = append(stores, entry.Store)
		if old.Model == "" {
			old = entry.Metadata
		}
	}
	plan := aisearch.PlanEmbeddingRetargetForStores(old, target, stores)
	return result{Operation: "embedding retarget plan", Status: "ok", Plan: &plan, Message: fmt.Sprintf("classified %s for %d store(s)", plan.Compatibility, len(plan.StoreDetails))}, nil
}

func (h *handlers) applyShadowCall(ctx cliapp.OperationContext) (result, error) {
	if !ctx.BoolFlag("shadow") {
		return result{}, errors.New("embedding retarget apply requires --shadow; live collections are never written directly")
	}
	providerID := strings.TrimSpace(ctx.Flag("provider-id"))
	if providerID == "" {
		return result{}, errors.New("--provider-id is required for a provider-owned shadow build")
	}
	entries, err := aisearch.InventoryQdrantCollections(context.Background(), qdrantURL(), os.Getenv("QDRANT_API_KEY"))
	if err != nil {
		return result{}, err
	}
	target, err := resolveTarget(context.Background(), ctx.Flag("model"), ctx.Flag("role"), ctx.Flag("dimensions"))
	if err != nil {
		return result{}, err
	}
	collection := strings.TrimSpace(ctx.Flag("collection"))
	if collection == "" && len(entries) == 1 {
		collection = entries[0].Store.Collection
	}
	if collection == "" {
		return result{}, errors.New("--collection is required when inventory contains multiple collections")
	}
	var old aisearch.EmbeddingMetadata
	for _, entry := range entries {
		if entry.Store.Collection == collection {
			old = entry.Metadata
			break
		}
	}
	if old.Model == "" {
		return result{}, fmt.Errorf("collection %q was not found in the inventory", collection)
	}
	store := aisearch.NewQdrantEmbeddingStore(collection)
	plan := aisearch.PlanEmbeddingRetargetForStores(old, target, []aisearch.EmbeddingStore{store})
	state := migrationState{ProviderID: providerID, LiveCollection: collection, ShadowCollection: collection + "__retarget_shadow", ActiveCollection: collection, Plan: plan, State: "shadow_build_requested", UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	providerResponse, err := h.invokeProviderControl(context.Background(), providerID, map[string]any{"action": "shadow", "provider_id": providerID, "collection": collection, "shadow_collection": state.ShadowCollection, "model": target.Model, "role": target.Role, "dimensions": target.Dimensions, "policy_schema_version": target.PolicySchemaVersion})
	if err != nil {
		return result{}, err
	}
	state.JobID = providerResponse.GetJobId()
	if state.JobID != "" {
		if err := h.waitMigration(context.Background(), providerID, state.JobID, &state); err != nil {
			return result{}, err
		}
	}
	state.State = "shadow_ready"
	if err := saveState(state); err != nil {
		return result{}, err
	}
	return result{Operation: "embedding retarget apply --shadow", Status: "accepted", Migration: &state, Message: "provider reindex endpoint accepted a shadow-only build; live collection was not written"}, nil
}

func (h *handlers) recordCompareCall(ctx cliapp.OperationContext) (result, error) {
	state, err := loadState()
	if err != nil {
		return result{}, err
	}
	verdict := strings.ToLower(strings.TrimSpace(ctx.Flag("verdict")))
	if verdict != "pass" && verdict != "fail" && verdict != "withheld" {
		return result{}, errors.New("--verdict must be pass, fail, or withheld")
	}
	state.CompareVerdict = verdict
	state.State = "shadow_compared"
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveState(state); err != nil {
		return result{}, err
	}
	return result{Operation: "embedding retarget record-compare", Status: "ok", Migration: &state, Message: "compare verdict recorded; cutover still requires pass"}, nil
}

func (h *handlers) abortCall(ctx cliapp.OperationContext) (result, error) {
	state, err := loadState()
	if err != nil {
		return result{}, err
	}
	if state.ShadowCollection == "" {
		return result{}, errors.New("migration has no shadow collection")
	}
	if err := deleteCollection(context.Background(), state.ShadowCollection); err != nil {
		return result{}, err
	}
	state.State = "aborted"
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveState(state); err != nil {
		return result{}, err
	}
	return result{Operation: "embedding retarget abort", Status: "ok", Migration: &state, Message: "shadow collection dropped; live collection remains active"}, nil
}

func (h *handlers) cutoverCall(ctx cliapp.OperationContext) (result, error) {
	state, err := loadState()
	if err != nil {
		return result{}, err
	}
	if err := requirePassingCompare(state); err != nil {
		return result{}, err
	}
	if _, err := h.invokeProviderControl(context.Background(), state.ProviderID, map[string]any{"action": "cutover", "provider_id": state.ProviderID, "collection": state.LiveCollection, "shadow_collection": state.ShadowCollection}); err != nil {
		return result{}, err
	}
	state.OldCollection = state.LiveCollection
	state.ActiveCollection = state.ShadowCollection
	state.State = "cutover"
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveState(state); err != nil {
		return result{}, err
	}
	return result{Operation: "embedding retarget cutover", Status: "ok", Migration: &state, Message: "provider pointer moved to the shadow; old collection retained for rollback"}, nil
}

func requirePassingCompare(state migrationState) error {
	if state.CompareVerdict != "pass" {
		return fmt.Errorf("refusing cutover: recorded evals compare verdict is %q; require pass", state.CompareVerdict)
	}
	return nil
}

func (h *handlers) rollbackCall(ctx cliapp.OperationContext) (result, error) {
	state, err := loadState()
	if err != nil {
		return result{}, err
	}
	if state.OldCollection == "" {
		return result{}, errors.New("no retained live collection is recorded; rollback is unavailable")
	}
	if _, err := h.invokeProviderControl(context.Background(), state.ProviderID, map[string]any{"action": "rollback", "provider_id": state.ProviderID, "collection": state.ActiveCollection, "rollback_collection": state.OldCollection}); err != nil {
		return result{}, err
	}
	state.ActiveCollection = state.OldCollection
	state.State = "rolled_back"
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := saveState(state); err != nil {
		return result{}, err
	}
	return result{Operation: "embedding retarget rollback", Status: "ok", Migration: &state, Message: "provider pointer restored without reindexing"}, nil
}

func (h *handlers) invokeProviderControl(ctx context.Context, providerID string, body map[string]any) (*registryv1.ExecuteEmbeddingMigrationResponse, error) {
	request := &registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: providerID, Action: stringValue(body, "action"), ShadowCollection: stringValue(body, "shadow_collection"), RollbackCollection: stringValue(body, "rollback_collection"), EmbeddingModel: stringValue(body, "model"), EmbeddingRole: stringValue(body, "role"), EmbeddingPolicySchemaVersion: stringValue(body, "policy_schema_version")}
	if value, ok := body["dimensions"].(int); ok {
		request.EmbeddingDimensions = int32(value)
	}
	resp, err := h.client.ExecuteEmbeddingMigration(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("execute provider %q embedding migration: %w", providerID, err)
	}
	return resp.Msg, nil
}

func (h *handlers) waitMigration(ctx context.Context, providerID, jobID string, state *migrationState) error {
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		resp, err := h.client.ExecuteEmbeddingMigration(ctx, connect.NewRequest(&registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: providerID, Action: "status", JobId: jobID}))
		if err != nil {
			return fmt.Errorf("read provider migration job %q: %w", jobID, err)
		}
		state.State = resp.Msg.GetState()
		switch resp.Msg.GetState() {
		case "succeeded":
			return nil
		case "failed", "cancelled":
			return fmt.Errorf("provider migration job %q ended %s: %s", jobID, resp.Msg.GetState(), resp.Msg.GetError())
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("provider migration job %q did not finish within 30 minutes", jobID)
}

func stringValue(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return value
}

func (h *handlers) report(_ cliapp.OperationContext, value result) cliapp.MutationReport {
	resultLine := value.Message
	if resultLine == "" {
		resultLine = value.Status
	}
	return cliapp.MutationReport{Result: []string{resultLine}, Changes: []string{value.Operation + " status=" + value.Status}, NextCommand: value.NextSteps}
}

func resolveTarget(ctx context.Context, model, role, dimensions string) (aisearch.EmbeddingMetadata, error) {
	role = strings.TrimSpace(role)
	model = strings.TrimSpace(model)
	if model == "" && role == "" {
		role = aisearch.DefaultEmbedRole
	}
	var resolved ollamapolicy.ResolvedRole
	var err error
	if model != "" {
		resolved, err = (ollamapolicy.Resolver{}).ResolveModel(ctx, model)
	} else {
		resolved, err = (ollamapolicy.Resolver{}).ResolveRole(ctx, role)
	}
	if err != nil {
		return aisearch.EmbeddingMetadata{}, err
	}
	if dimensions != "" {
		var parsed int
		if _, scanErr := fmt.Sscanf(dimensions, "%d", &parsed); scanErr != nil || parsed <= 0 {
			return aisearch.EmbeddingMetadata{}, fmt.Errorf("invalid --dimensions %q", dimensions)
		}
		resolved.EmbeddingDimensions = parsed
	}
	policy, err := aisearch.EmbeddingPolicyFromResolved(resolved)
	if err != nil {
		return aisearch.EmbeddingMetadata{}, err
	}
	return aisearch.EmbeddingMetadata(policy), nil
}

func qdrantURL() string {
	if value := strings.TrimSpace(os.Getenv("QDRANT_URL")); value != "" {
		return value
	}
	return aisearch.DefaultQdrantURL
}

func statePath() string {
	if value := strings.TrimSpace(os.Getenv("SEARCH_HUB_EMBEDDING_STATE_FILE")); value != "" {
		return value
	}
	root := strings.TrimSpace(os.Getenv("VROOLI_SOURCE_ROOT"))
	if root == "" {
		root, _ = os.Getwd()
	}
	return filepath.Join(root, ".vrooli", "state", "search-hub", "embedding-migration.json")
}

func loadState() (migrationState, error) {
	blob, err := os.ReadFile(statePath())
	if err != nil {
		return migrationState{}, fmt.Errorf("read embedding migration state %q: %w", statePath(), err)
	}
	var state migrationState
	if err := json.Unmarshal(blob, &state); err != nil {
		return migrationState{}, fmt.Errorf("decode embedding migration state: %w", err)
	}
	return state, nil
}

func saveState(state migrationState) error {
	path := statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create embedding migration state directory: %w", err)
	}
	blob, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode embedding migration state: %w", err)
	}
	return os.WriteFile(path, append(blob, '\n'), 0o600)
}

func deleteCollection(ctx context.Context, collection string) error {
	endpoint := strings.TrimRight(qdrantURL(), "/") + "/collections/" + url.PathEscape(collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	if key := strings.TrimSpace(os.Getenv("QDRANT_API_KEY")); key != "" {
		req.Header.Set("api-key", key)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("drop shadow collection %q: %w", collection, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("drop shadow collection %q: HTTP %d", collection, resp.StatusCode)
	}
	return nil
}
