package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"storage-manager/internal/cleanup"
)

// ModelStore is one model store's owner boundary: its retention view and its
// one fenced eviction operation. Storage-manager decides; the owner deletes.
type ModelStore interface {
	Resource() string
	Inventory(context.Context) (modelretention.Inventory, error)
	Evict(context.Context, ModelEvictionRequest) (modelretention.EvictResult, error)
}

// ModelEvictionRequest is what the decision saw, so the owner can refuse a
// model whose identity changed since.
type ModelEvictionRequest struct {
	Name, Digest string
	Size         int64
	Reason       string
	RequestedBy  string
}

// CommandRunner runs an owner command and returns its stdout.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// CommandModelStore drives `<command> models retention|evict --json`.
type CommandModelStore struct {
	Name    string
	Command string
	Run     CommandRunner
}

func (s CommandModelStore) Resource() string { return s.Name }

func (s CommandModelStore) run(ctx context.Context, args ...string) ([]byte, error) {
	if s.Run != nil {
		return s.Run(ctx, s.Command, args...)
	}
	cmd := exec.CommandContext(ctx, s.Command, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w: %s", s.Command, strings.Join(args, " "), err, strings.TrimSpace(lastLine(stderr.String())))
	}
	return out, nil
}

func (s CommandModelStore) Inventory(ctx context.Context) (modelretention.Inventory, error) {
	out, err := s.run(ctx, "models", "retention", "--json")
	if err != nil {
		return modelretention.Inventory{}, err
	}
	var inv modelretention.Inventory
	if err := decodeOwnerJSON(out, &inv); err != nil {
		return modelretention.Inventory{}, fmt.Errorf("decode %s retention inventory: %w", s.Name, err)
	}
	if inv.Schema != modelretention.InventorySchema {
		return modelretention.Inventory{}, fmt.Errorf("%s retention inventory schema %q, want %q", s.Name, inv.Schema, modelretention.InventorySchema)
	}
	return inv, nil
}

func (s CommandModelStore) Evict(ctx context.Context, req ModelEvictionRequest) (modelretention.EvictResult, error) {
	args := []string{"models", "evict", req.Name, "--digest", req.Digest, "--size", strconv.FormatInt(req.Size, 10), "--json"}
	if req.Reason != "" {
		args = append(args, "--reason", req.Reason)
	}
	if req.RequestedBy != "" {
		args = append(args, "--requested-by", req.RequestedBy)
	}
	out, runErr := s.run(ctx, args...)
	var result modelretention.EvictResult
	if err := decodeOwnerJSON(out, &result); err != nil {
		if runErr != nil {
			return result, runErr
		}
		return result, fmt.Errorf("decode %s eviction result: %w", s.Name, err)
	}
	if result.Refused != "" {
		return result, nil
	}
	return result, runErr
}

// decodeOwnerJSON tolerates a rebuild notice printed before the document.
func decodeOwnerJSON(out []byte, v any) error {
	trimmed := bytes.TrimSpace(out)
	if !bytes.HasPrefix(trimmed, []byte("{")) {
		idx := bytes.Index(trimmed, []byte("\n{"))
		if idx < 0 {
			return errors.New("no JSON document in owner output")
		}
		trimmed = trimmed[idx+1:]
	}
	return json.Unmarshal(trimmed, v)
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// ModelStoreEnvironment supplies the host facts the decision needs.
type ModelStoreEnvironment struct {
	// UsagePath returns the store's model-usage.json.
	UsagePath func(resource string) (string, error)
	// Space measures the filesystem holding path.
	Space func(path string) (FilesystemSpace, error)
	// Budget returns the store's declared byte budget (0 when none).
	Budget func(resource string) (int64, error)
}

// ModelStoreRetentionProvider decides disk eviction across model stores and
// applies an approved selection through each owner's fenced operation.
type ModelStoreRetentionProvider struct {
	policy ModelRetentionPolicy
	stores []ModelStore
	env    ModelStoreEnvironment
	clock  cleanup.Clock

	mu   sync.Mutex
	last ModelEvictionDecision
}

func NewModelStoreRetentionProvider(policy ModelRetentionPolicy, stores []ModelStore, env ModelStoreEnvironment, clock cleanup.Clock) *ModelStoreRetentionProvider {
	return &ModelStoreRetentionProvider{policy: policy, stores: stores, env: env, clock: clock}
}

const modelStoreProviderID = "model-store-retention"

func (p *ModelStoreRetentionProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID: modelStoreProviderID, Name: "Model store disk retention", Version: "v1",
		OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafeWithOwner,
		// Every selection is an operator decision by default; each owner's
		// eviction re-checks residency, in-flight requests, identity and
		// policy under its fence.
		DefaultMode: cleanup.ProviderModeEnabled, DefaultApproval: cleanup.ApprovalModeOperator,
		SupportedPlatforms: []string{"linux", "darwin"}, RequiredPrivileges: []string{"model-store-owner-cli"},
		IrreversibleEffects: []string{"deletes downloaded model weights; the owner records the reacquire command and expected size first, and the model is re-downloaded on demand"},
		TestSubstitute:      "fake-model-store",
	}
}

// Policy is the loaded retention policy.
func (p *ModelStoreRetentionProvider) Policy() ModelRetentionPolicy { return p.policy }

func (p *ModelStoreRetentionProvider) now() time.Time {
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Now()
}

// Decide observes every store and returns the current decision.
func (p *ModelStoreRetentionProvider) Decide(ctx context.Context) ModelEvictionDecision {
	now := p.now()
	observations := make([]ModelStoreObservation, 0, len(p.stores))
	for _, store := range p.stores {
		observations = append(observations, p.observe(ctx, store))
	}
	d := DecideModelEvictions(p.policy, observations, now)
	p.mu.Lock()
	p.last = d
	p.mu.Unlock()
	return d
}

// Last returns the most recent decision without observing again.
func (p *ModelStoreRetentionProvider) Last() ModelEvictionDecision {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.last
}

func (p *ModelStoreRetentionProvider) observe(ctx context.Context, store ModelStore) ModelStoreObservation {
	obs := ModelStoreObservation{Resource: store.Resource()}
	inv, err := store.Inventory(ctx)
	if err != nil {
		obs.Err = err
		return obs
	}
	obs.Inventory = inv
	if p.env.UsagePath != nil {
		path, err := p.env.UsagePath(obs.Resource)
		if err == nil {
			obs.Usage, err = modelretention.ReadUsage(path)
		}
		if err != nil {
			obs.Err = fmt.Errorf("read %s model usage: %w", obs.Resource, err)
			return obs
		}
	}
	if p.env.Space != nil {
		if obs.Space, err = p.env.Space(inv.ModelRoot); err != nil {
			obs.Err = fmt.Errorf("measure %s filesystem: %w", obs.Resource, err)
			return obs
		}
	}
	if p.env.Budget != nil {
		if obs.BudgetBytes, err = p.env.Budget(obs.Resource); err != nil {
			obs.Err = fmt.Errorf("read %s budget: %w", obs.Resource, err)
			return obs
		}
	}
	return obs
}

// modelItemIdentity is the PreviewItem identity: exactly what was decided.
type modelItemIdentity struct {
	Resource string `json:"resource"`
	Name     string `json:"name"`
	Digest   string `json:"digest"`
	Size     int64  `json:"size"`
}

// ModelItemID is the stable preview item id for one candidate.
func ModelItemID(c ModelCandidate) string {
	return modelStoreProviderID + ":" + c.Resource + ":" + c.Name + "@" + c.Digest
}

func modelPreviewItem(c ModelCandidate) cleanup.PreviewItem {
	identity, _ := json.Marshal(modelItemIdentity{Resource: c.Resource, Name: c.Name, Digest: c.Digest, Size: c.Bytes})
	return cleanup.PreviewItem{
		ID: ModelItemID(c), Path: "model-store://" + c.Resource + "/" + c.Name,
		Description: fmt.Sprintf("%s model %s, idle %s, selected for %s; reacquire: %s", c.Resource, c.Name, roundDays(c.IdleFor), c.SelectedFor, strings.Join(c.Reacquire, " ")),
		Bytes:       c.UniqueBytes, Action: "evict", SafetyTier: cleanup.SafetyTierSafeWithOwner, Identity: string(identity),
		ModTime: c.LastActivity,
	}
}

// DecisionLines renders the ranking for previews, notices and logs.
func DecisionLines(d ModelEvictionDecision) []string {
	var lines []string
	for _, fs := range d.Filesystems {
		line := fmt.Sprintf("filesystem %s: %.1f%% free (%d of %d bytes), band %s, projected %.1f%% after selection", fs.Path, fs.FreePercent, fs.FreeBytes, fs.TotalBytes, fs.Band, fs.ProjectedFreePercent)
		if fs.Shortfall != "" {
			line += "; " + fs.Shortfall
		}
		lines = append(lines, line)
	}
	for _, s := range d.Stores {
		switch {
		case s.Error != "":
			lines = append(lines, fmt.Sprintf("store %s: unobserved: %s", s.Resource, s.Error))
		case s.BudgetBytes > 0:
			lines = append(lines, fmt.Sprintf("store %s: %d bytes of %d budget (%d over)", s.Resource, s.StoreBytes, s.BudgetBytes, s.OverBudgetBytes))
		default:
			lines = append(lines, fmt.Sprintf("store %s: %d bytes, no budget", s.Resource, s.StoreBytes))
		}
	}
	for i, c := range d.Candidates {
		state := "eligible"
		if c.SelectedFor != "" {
			state = "SELECTED for " + c.SelectedFor
		} else if !c.Eligible {
			state = "held: " + c.Held
		}
		lines = append(lines, fmt.Sprintf("%d. %s/%s unique=%d bytes=%d idle=%s score=%.0f %s", i+1, c.Resource, c.Name, c.UniqueBytes, c.Bytes, roundDays(c.IdleFor), c.Score, state))
	}
	return lines
}

func (p *ModelStoreRetentionProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	meta := p.Metadata()
	out := cleanup.Estimate{ProviderID: meta.ID, ProviderVersion: meta.Version, RequiresApproval: true, ObservedAt: req.Scope.Now}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	d := p.Decide(ctx)
	out.EstimatedBytes = d.SelectedBytes
	out.ItemCount = len(d.Selected)
	for _, s := range d.Errors() {
		out.Notices = append(out.Notices, fmt.Sprintf("store %s unobserved: %s", s.Resource, s.Error))
	}
	return out, nil
}

func (p *ModelStoreRetentionProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	meta := p.Metadata()
	out := cleanup.Preview{ProviderID: meta.ID, ProviderVersion: meta.Version}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	d := p.Decide(ctx)
	for _, c := range d.Selected {
		out.Items = append(out.Items, modelPreviewItem(c))
	}
	out.Warnings = DecisionLines(d)
	// One model is one indivisible owner operation.
	out.AllowSingleOvershoot = true
	return out, nil
}

// ModelEvictionOutcome is one item's apply result.
type ModelEvictionOutcome struct {
	Item       string `json:"item"`
	Resource   string `json:"resource"`
	Model      string `json:"model"`
	Evicted    bool   `json:"evicted"`
	FreedBytes int64  `json:"freed_bytes"`
	Refused    string `json:"refused,omitempty"`
	Error      string `json:"error,omitempty"`
	Reacquire  string `json:"reacquire,omitempty"`
}

// ApplyApproved evicts the approved candidates that are still eligible now.
// Approval covers the exact identities decided; a model used, reloaded,
// re-pulled or made required since is skipped, never re-chosen.
func (p *ModelStoreRetentionProvider) ApplyApproved(ctx context.Context, approved []ModelCandidate, reason, requestedBy string) []ModelEvictionOutcome {
	current := p.Decide(ctx)
	byID := map[string]ModelCandidate{}
	for _, c := range current.Candidates {
		byID[ModelItemID(c)] = c
	}
	stores := map[string]ModelStore{}
	for _, s := range p.stores {
		stores[s.Resource()] = s
	}
	var outcomes []ModelEvictionOutcome
	for _, want := range approved {
		id := ModelItemID(want)
		outcome := ModelEvictionOutcome{Item: id, Resource: want.Resource, Model: want.Name}
		now, ok := byID[id]
		switch {
		case !ok:
			outcome.Refused = "no longer installed with the approved digest"
		case !now.Eligible:
			outcome.Refused = "no longer eligible: " + now.Held
		case now.Bytes != want.Bytes:
			outcome.Refused = "size changed since approval"
		}
		store := stores[want.Resource]
		if outcome.Refused == "" && store == nil {
			outcome.Refused = "store is not configured"
		}
		if outcome.Refused != "" {
			outcomes = append(outcomes, outcome)
			continue
		}
		result, err := store.Evict(ctx, ModelEvictionRequest{Name: want.Name, Digest: want.Digest, Size: want.Bytes, Reason: reason, RequestedBy: requestedBy})
		outcome.Evicted, outcome.FreedBytes, outcome.Refused = result.Evicted, result.FreedBytes, result.Refused
		outcome.Reacquire = strings.Join(result.Reacquire, " ")
		if err != nil {
			outcome.Error = err.Error()
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes
}

func (p *ModelStoreRetentionProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	meta := p.Metadata()
	if req.ProviderVersion != meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", meta.ID, req.ProviderVersion, meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s requires idempotency key", meta.ID)
	}
	result := cleanup.ApplyResult{ProviderID: meta.ID}
	if req.ApprovalMode != cleanup.ApprovalModeOperator {
		result.SkippedItems = previewItemIDs(req.Preview.Items)
		result.Warnings = []string{"operator approval required"}
		return result, nil
	}
	var approved []ModelCandidate
	for _, item := range req.Preview.Items {
		var identity modelItemIdentity
		if err := json.Unmarshal([]byte(item.Identity), &identity); err != nil || identity.Name == "" {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": unreadable identity")
			continue
		}
		approved = append(approved, ModelCandidate{Resource: identity.Resource, Name: identity.Name, Digest: identity.Digest, Bytes: identity.Size})
	}
	for _, o := range p.ApplyApproved(ctx, approved, "cleanup plan "+req.PlanID, "storage-manager cleanup apply") {
		if o.Evicted {
			result.AppliedItems = append(result.AppliedItems, o.Item)
			result.ReclaimedBytes = saturatingAddNonnegative(result.ReclaimedBytes, o.FreedBytes)
			continue
		}
		result.SkippedItems = append(result.SkippedItems, o.Item)
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s%s", o.Item, o.Refused, o.Error))
	}
	result.Applied = len(result.AppliedItems) > 0
	return result, nil
}

// Verify confirms every applied model is gone from its owner's inventory.
func (p *ModelStoreRetentionProvider) Verify(ctx context.Context, req cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	if len(req.ApplyResult.AppliedItems) == 0 {
		return cleanup.VerifyResult{Verified: false, Message: "no model eviction was reported for verification"}, nil
	}
	present := map[string]bool{}
	for _, store := range p.stores {
		inv, err := store.Inventory(ctx)
		if err != nil {
			return cleanup.VerifyResult{Verified: false, Message: fmt.Sprintf("%s inventory unavailable: %v", store.Resource(), err)}, nil
		}
		for _, m := range inv.Models {
			present[ModelItemID(ModelCandidate{Resource: store.Resource(), Name: m.Name, Digest: m.Digest})] = true
		}
	}
	for _, id := range req.ApplyResult.AppliedItems {
		if present[id] {
			return cleanup.VerifyResult{Verified: false, Message: id + " is still installed"}, nil
		}
	}
	if len(req.ApplyResult.SkippedItems) > 0 {
		return cleanup.VerifyResult{Verified: false, Message: "evicted models are gone; some selected models were skipped"}, nil
	}
	return cleanup.VerifyResult{Verified: true, Message: "every evicted model is gone from its store"}, nil
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
