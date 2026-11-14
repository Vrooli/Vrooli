package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"storage-manager/internal/cleanup"
)

type OllamaModel struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest,omitempty"`
	ModifiedAt time.Time `json:"modified_at,omitempty"`
	SizeVRAM   int64     `json:"size_vram,omitempty"`
}

type OllamaModelInventory interface {
	ListModels(context.Context) ([]OllamaModel, error)
	ListRunningModels(context.Context) ([]OllamaModel, error)
}

// HTTPOllamaModelInventory is the inventory-only storage-manager boundary to
// Ollama. The public delete API accepts a model name without a digest/size
// precondition. Storage Manager has no supported owner operation that
// serializes deletion with every active request, role-policy writer, and
// lifecycle transition, so this adapter intentionally has no delete method.
// Preview and pre-apply observations do not establish that missing fence.
type HTTPOllamaModelInventory struct {
	BaseURL string
	HTTP    *http.Client
}

func NewHTTPOllamaModelInventory(baseURL string) *HTTPOllamaModelInventory {
	return &HTTPOllamaModelInventory{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *HTTPOllamaModelInventory) ListModels(ctx context.Context) ([]OllamaModel, error) {
	var response struct {
		Models []OllamaModel `json:"models"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/tags", nil, &response); err != nil {
		return nil, err
	}
	sort.Slice(response.Models, func(i, j int) bool { return response.Models[i].Name < response.Models[j].Name })
	return response.Models, nil
}

func (c *HTTPOllamaModelInventory) ListRunningModels(ctx context.Context) ([]OllamaModel, error) {
	var response struct {
		Models []struct {
			Name     string `json:"name"`
			Model    string `json:"model"`
			Size     int64  `json:"size"`
			SizeVRAM int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/ps", nil, &response); err != nil {
		return nil, err
	}
	out := make([]OllamaModel, 0, len(response.Models))
	for _, model := range response.Models {
		name := model.Name
		if name == "" {
			name = model.Model
		}
		out = append(out, OllamaModel{Name: name, Size: model.Size, SizeVRAM: model.SizeVRAM})
	}
	return out, nil
}

func (c *HTTPOllamaModelInventory) doJSON(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("ollama %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(message)))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode Ollama %s: %w", path, err)
		}
	}
	return nil
}

type ollamaRolePolicy struct {
	Model     string   `json:"model"`
	Fallbacks []string `json:"fallbacks"`
}

type ollamaModelPolicy struct {
	Roles map[string]ollamaRolePolicy `json:"roles"`
}

type ollamaModelLedger interface {
	Record(context.Context, time.Time, []OllamaModel) error
	RecordInventory(context.Context, time.Time, []OllamaModel) (ollamaInventoryTotals, error)
	Eligible(string, time.Time, time.Duration) bool
}

type OllamaModelRetentionProvider struct {
	client     OllamaModelInventory
	ledger     ollamaModelLedger
	policyPath string
	clock      cleanup.Clock
}

func NewOllamaModelRetentionProvider(client OllamaModelInventory, ledger ollamaModelLedger, policyPath string, clock cleanup.Clock) *OllamaModelRetentionProvider {
	return &OllamaModelRetentionProvider{client: client, ledger: ledger, policyPath: policyPath, clock: clock}
}

func (p *OllamaModelRetentionProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID: "ollama-model-retention", Name: "Ollama model retention (held without owner fence)", Version: "v3",
		OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafeWithOwner,
		// Preview is observational. Apply requires operator approval and also
		// fails closed until a supported atomic owner deletion contract exists.
		DefaultMode: cleanup.ProviderModeEnabled, DefaultApproval: cleanup.ApprovalModeOperator,
		SupportedPlatforms: []string{"linux", "darwin", "windows"}, RequiredPrivileges: []string{"ollama-api"},
		IrreversibleEffects: []string{"none; model retention is held until an atomic owner deletion contract exists"}, TestSubstitute: "fake-ollama-model-inventory",
	}
}

func (p *OllamaModelRetentionProvider) referencedModels() (map[string]bool, error) {
	data, err := os.ReadFile(p.policyPath)
	if err != nil {
		return nil, fmt.Errorf("read Ollama model policy: %w", err)
	}
	var policy ollamaModelPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("decode Ollama model policy: %w", err)
	}
	refs := map[string]bool{}
	for _, role := range policy.Roles {
		if model := strings.TrimSpace(role.Model); model != "" {
			refs[model] = true
		}
		for _, fallback := range role.Fallbacks {
			if model := strings.TrimSpace(fallback); model != "" {
				refs[model] = true
			}
		}
	}
	return refs, nil
}

func (p *OllamaModelRetentionProvider) observe(ctx context.Context, now time.Time) ([]OllamaModel, []OllamaModel, map[string]bool, ollamaInventoryTotals, error) {
	if p.client == nil || p.ledger == nil {
		return nil, nil, nil, ollamaInventoryTotals{}, fmt.Errorf("Ollama inventory or usage ledger unavailable")
	}
	models, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, nil, nil, ollamaInventoryTotals{}, err
	}
	running, err := p.client.ListRunningModels(ctx)
	if err != nil {
		return nil, nil, nil, ollamaInventoryTotals{}, err
	}
	if err := p.ledger.Record(ctx, now, running); err != nil {
		return nil, nil, nil, ollamaInventoryTotals{}, err
	}
	totals, err := p.ledger.RecordInventory(ctx, now, models)
	if err != nil {
		return nil, nil, nil, ollamaInventoryTotals{}, err
	}
	refs, err := p.referencedModels()
	if err != nil {
		return nil, nil, nil, ollamaInventoryTotals{}, err
	}
	return models, running, refs, totals, nil
}

func (p *OllamaModelRetentionProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	if !req.Policy.Enabled {
		return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, BlockedReason: "provider disabled by policy", RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
	}
	preview, err := p.preview(ctx, req.Scope.Now, req.Policy)
	if err != nil {
		return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, BlockedReason: err.Error(), RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
	}
	var bytes int64
	for _, item := range preview.Items {
		bytes = saturatingAddNonnegative(bytes, item.Bytes)
	}
	notices := append([]string(nil), preview.Warnings...)
	if len(preview.Items) == 0 {
		notices = append(notices, "estimated reclaimable bytes are zero because retention has no authorized deletion candidate; actual filesystem bytes reclaimed are unknown until an owner reports an authoritative delta")
	}
	return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, EstimatedBytes: bytes, ItemCount: len(preview.Items), RequiresApproval: true, ObservedAt: req.Scope.Now, Notices: notices}, nil
}

func (p *OllamaModelRetentionProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	if !req.Policy.Enabled {
		return cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, BlockedReason: "provider disabled by policy"}, nil
	}
	return p.preview(ctx, req.Scope.Now, req.Policy)
}

func (p *OllamaModelRetentionProvider) preview(ctx context.Context, now time.Time, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, MinAge: policy.MinAge}
	if now.IsZero() {
		if p.clock == nil {
			return out, fmt.Errorf("Ollama retention observation clock is unavailable")
		}
		now = p.clock.Now()
	}
	models, running, refs, totals, err := p.observe(ctx, now)
	if err != nil {
		out.BlockedReason = err.Error()
		return out, nil
	}
	runningSet := map[string]bool{}
	for _, model := range running {
		runningSet[model.Name] = true
	}
	var logicalBytes int64
	var explicitlyProtectedBytes int64
	observedModels := 0
	for _, model := range models {
		if model.Name == "" {
			continue
		}
		observedModels++
		logicalBytes = saturatingAddNonnegative(logicalBytes, model.Size)
		isRunning := runningSet[model.Name]
		isPolicyReferenced := refs[model.Name]
		if isRunning || isPolicyReferenced {
			explicitlyProtectedBytes = saturatingAddNonnegative(explicitlyProtectedBytes, model.Size)
		}
		if isRunning {
			continue
		}
		if isPolicyReferenced {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: protected by the active Ollama model policy", model.Name))
		}
		if model.Digest == "" {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: Ollama did not report a digest; retention identity is unknown", model.Name))
			continue
		}
		if isPolicyReferenced {
			continue
		}
		// This is deliberately a permanent fail-closed gate until Ollama exposes
		// a supported owner mutation contract that atomically covers role-policy
		// references, runtime activity/lifecycle and the observed model identity.
		// Storage Manager's preview and pre-apply reads are advisory only; they
		// do not serialize with request-time usage or every model writer.
		out.Warnings = append(out.Warnings, fmt.Sprintf("%s: held; exact install provenance is unavailable and Ollama exposes only name-based deletion, with no supported atomic retention fence", model.Name))
	}
	if observedModels > 0 {
		out.Warnings = append(out.Warnings, fmt.Sprintf("this preview observed %d Ollama model tags with %d bytes of reported logical model size at %s; shared blobs mean this is not unique filesystem usage or reclaimable-byte attribution", observedModels, logicalBytes, now.UTC().Format(time.RFC3339Nano)))
	}
	// All installed bytes remain retained until the serving owner can apply an
	// atomic reference/activity fence. The explicitly protected subtotal calls
	// out policy/running references, while the remainder is still held by the
	// fail-closed provider contract.
	out.Warnings = append(out.Warnings, fmt.Sprintf("Ollama persisted logical tag accounting: baseline=%d B current=%d B peak=%d B retained=%d B; retained includes every installed tag because no atomic owner deletion fence exists; snapshot observed %s. Explicit policy/running protected bytes in this preview=%d B at %s. Shared blobs make these tag totals non-physical and non-reclaimable", totals.BaselineLogicalBytes, totals.CurrentLogicalBytes, totals.PeakLogicalBytes, totals.CurrentLogicalBytes, totals.LastObservedAt.UTC().Format(time.RFC3339Nano), explicitlyProtectedBytes, now.UTC().Format(time.RFC3339Nano)))
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ID < out.Items[j].ID })
	return out, nil
}

func (p *OllamaModelRetentionProvider) Apply(_ context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.Metadata().Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.Metadata().ID, req.ProviderVersion, p.Metadata().Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s requires idempotency key", p.Metadata().ID)
	}
	if req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: p.Metadata().ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"operator approval required"}}, nil
	}
	result := cleanup.ApplyResult{ProviderID: p.Metadata().ID}
	result.SkippedItems = previewItemIDs(req.Preview.Items)
	if len(result.SkippedItems) > 0 {
		result.Warnings = append(result.Warnings, "Ollama model retention is disabled: no supported atomic owner fence covers role-policy references, runtime activity, lifecycle operations, and model identity at deletion time; no authoritative filesystem delta is available, so reclaimed bytes remain unknown and are reported as zero")
	}
	return result, nil
}

func (p *OllamaModelRetentionProvider) Verify(_ context.Context, req cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	if len(req.ApplyResult.SkippedItems) > 0 {
		return cleanup.VerifyResult{Verified: false, Message: "Ollama retention apply skipped one or more items"}, nil
	}
	if len(req.ApplyResult.AppliedItems) == 0 {
		return cleanup.VerifyResult{Verified: false, Message: "no Ollama model deletion was reported for verification"}, nil
	}
	return cleanup.VerifyResult{Verified: false, Message: "Ollama exposes no supported atomic retention deletion contract"}, nil
}

func saturatingAddNonnegative(total, value int64) int64 {
	if value <= 0 {
		return total
	}
	max := int64(^uint64(0) >> 1)
	if total > max-value {
		return max
	}
	return total + value
}
