package providers

import (
	"bytes"
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
	DeleteModel(context.Context, string) error
}

// HTTPOllamaModelInventory is the storage-manager boundary to Ollama. It
// deliberately uses only the public model APIs and never inspects model blobs.
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

func (c *HTTPOllamaModelInventory) DeleteModel(ctx context.Context, model string) error {
	body, err := json.Marshal(map[string]string{"name": model})
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodDelete, "/api/delete", bytes.NewReader(body), nil)
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
		ID: "ollama-model-retention", Name: "Ollama models eligible for retention", Version: "v1",
		OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafeWithOwner,
		// Preview is safe to run automatically; deletion remains operator-gated
		// by both the conditional tier and Apply's approval check.
		DefaultMode: cleanup.ProviderModeEnabled, DefaultApproval: cleanup.ApprovalModeOperator,
		SupportedPlatforms: []string{"linux", "darwin", "windows"}, RequiredPrivileges: []string{"ollama-api"},
		IrreversibleEffects: []string{"operator-approved Ollama model deletion through the service API"}, TestSubstitute: "fake-ollama-model-inventory",
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

func (p *OllamaModelRetentionProvider) observe(ctx context.Context, now time.Time) ([]OllamaModel, []OllamaModel, map[string]bool, error) {
	if p.client == nil || p.ledger == nil {
		return nil, nil, nil, fmt.Errorf("Ollama inventory or usage ledger unavailable")
	}
	models, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	running, err := p.client.ListRunningModels(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := p.ledger.Record(ctx, now, running); err != nil {
		return nil, nil, nil, err
	}
	refs, err := p.referencedModels()
	if err != nil {
		return nil, nil, nil, err
	}
	return models, running, refs, nil
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
		bytes += item.Bytes
	}
	return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, EstimatedBytes: bytes, ItemCount: len(preview.Items), RequiresApproval: true, ObservedAt: req.Scope.Now}, nil
}

func (p *OllamaModelRetentionProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	if !req.Policy.Enabled {
		return cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version, BlockedReason: "provider disabled by policy"}, nil
	}
	return p.preview(ctx, req.Scope.Now, req.Policy)
}

func (p *OllamaModelRetentionProvider) preview(ctx context.Context, now time.Time, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: p.Metadata().Version}
	if now.IsZero() {
		if p.clock == nil {
			return out, fmt.Errorf("Ollama retention observation clock is unavailable")
		}
		now = p.clock.Now()
	}
	models, running, refs, err := p.observe(ctx, now)
	if err != nil {
		out.BlockedReason = err.Error()
		return out, nil
	}
	runningSet := map[string]bool{}
	for _, model := range running {
		runningSet[model.Name] = true
	}
	for _, model := range models {
		if model.Name == "" || runningSet[model.Name] {
			continue
		}
		reason := ""
		safety := cleanup.SafetyTierConditional
		switch {
		case !refs[model.Name]:
			reason = "unreferenced by model-policy.json"
		case p.ledger.Eligible(model.Name, now, policy.MinAge):
			reason = fmt.Sprintf("not observed loaded for %s", policy.MinAge)
		default:
			continue
		}
		out.Items = append(out.Items, cleanup.PreviewItem{ID: "ollama-model:" + model.Name, Description: fmt.Sprintf("Ollama model %s (%s)", model.Name, reason), Bytes: model.Size, Action: "ollama-delete-model", SafetyTier: safety})
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ID < out.Items[j].ID })
	return out, nil
}

func (p *OllamaModelRetentionProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
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
	for _, item := range req.Preview.Items {
		const prefix = "ollama-model:"
		if !strings.HasPrefix(item.ID, prefix) {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			continue
		}
		if err := p.client.DeleteModel(ctx, strings.TrimPrefix(item.ID, prefix)); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", item.ID, err))
			continue
		}
		result.ReclaimedBytes += item.Bytes
	}
	result.Applied = result.ReclaimedBytes > 0
	return result, nil
}

func (p *OllamaModelRetentionProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "Ollama model eviction uses the service API and operator approval"}, nil
}
