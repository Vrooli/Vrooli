package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// ModelRetentionPolicy is config/model-retention-policy.json: how storage-
// manager decides when, and which, downloaded models leave disk.
type ModelRetentionPolicy struct {
	// CheckInterval is how often the controller re-decides.
	CheckInterval time.Duration
	// NotifyBelowFreePercent: below this much free space on a model store's
	// filesystem, preview and notify the operator.
	NotifyBelowFreePercent float64
	// EvictBelowFreePercent: below this, select eligible idle models for
	// eviction until projected free space is back to EvictUntilFreePercent.
	EvictBelowFreePercent float64
	EvictUntilFreePercent float64
	// IdleWindow: a model is eligible only when neither a request nor its
	// pull happened within this window.
	IdleWindow time.Duration
	// RegistryTagRefetchPenalty divides the ranking score of a model whose
	// reacquisition re-resolves a registry tag (and may not reproduce the same
	// bytes), so digest-pinned models of the same size go first.
	RegistryTagRefetchPenalty float64
	// Approval is "operator" (each eviction set is a Notification Hub
	// decision) or "automatic" (the controller applies selections itself).
	Approval string
	Stores   []ModelStoreConfig
}

// ModelStoreConfig names one model store and its owner command.
type ModelStoreConfig struct {
	Resource string `json:"resource"`
	Command  string `json:"command"`
}

type modelRetentionPolicyFile struct {
	CheckInterval             string             `json:"check_interval"`
	NotifyBelowFreePercent    float64            `json:"notify_below_free_percent"`
	EvictBelowFreePercent     float64            `json:"evict_below_free_percent"`
	EvictUntilFreePercent     float64            `json:"evict_until_free_percent"`
	IdleWindow                string             `json:"idle_window"`
	RegistryTagRefetchPenalty float64            `json:"registry_tag_refetch_penalty"`
	Approval                  string             `json:"approval"`
	Stores                    []ModelStoreConfig `json:"stores"`
}

// LoadModelRetentionPolicy reads and validates the policy file.
func LoadModelRetentionPolicy(path string) (ModelRetentionPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ModelRetentionPolicy{}, fmt.Errorf("read model retention policy: %w", err)
	}
	var raw modelRetentionPolicyFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return ModelRetentionPolicy{}, fmt.Errorf("decode model retention policy %s: %w", path, err)
	}
	interval, err := time.ParseDuration(raw.CheckInterval)
	if err != nil {
		return ModelRetentionPolicy{}, fmt.Errorf("model retention check_interval: %w", err)
	}
	idle, err := coreRetention.ParseAge(raw.IdleWindow)
	if err != nil {
		return ModelRetentionPolicy{}, fmt.Errorf("model retention idle_window: %w", err)
	}
	p := ModelRetentionPolicy{
		CheckInterval: interval, NotifyBelowFreePercent: raw.NotifyBelowFreePercent,
		EvictBelowFreePercent: raw.EvictBelowFreePercent, EvictUntilFreePercent: raw.EvictUntilFreePercent,
		IdleWindow: idle, RegistryTagRefetchPenalty: raw.RegistryTagRefetchPenalty,
		Approval: strings.TrimSpace(raw.Approval), Stores: raw.Stores,
	}
	return p, p.Validate()
}

// Validate rejects bands that cannot converge and unknown approval modes.
func (p ModelRetentionPolicy) Validate() error {
	switch {
	case p.CheckInterval < time.Minute:
		return fmt.Errorf("model retention check interval %s is below one minute", p.CheckInterval)
	case p.IdleWindow <= 0:
		return fmt.Errorf("model retention idle window must be positive")
	case p.EvictBelowFreePercent <= 0 || p.EvictBelowFreePercent > p.NotifyBelowFreePercent:
		return fmt.Errorf("evict_below_free_percent must be positive and at most notify_below_free_percent")
	case p.EvictUntilFreePercent < p.EvictBelowFreePercent || p.EvictUntilFreePercent >= 100:
		return fmt.Errorf("evict_until_free_percent must be at least evict_below_free_percent and below 100")
	case p.RegistryTagRefetchPenalty < 1:
		return fmt.Errorf("registry_tag_refetch_penalty must be at least 1")
	case p.Approval != ModelApprovalOperator && p.Approval != ModelApprovalAutomatic:
		return fmt.Errorf("model retention approval %q must be %q or %q", p.Approval, ModelApprovalOperator, ModelApprovalAutomatic)
	case len(p.Stores) == 0:
		return fmt.Errorf("model retention policy names no stores")
	}
	for _, s := range p.Stores {
		if strings.TrimSpace(s.Resource) == "" || strings.TrimSpace(s.Command) == "" {
			return fmt.Errorf("model retention store needs resource and command")
		}
	}
	return nil
}

const (
	ModelApprovalOperator  = "operator"
	ModelApprovalAutomatic = "automatic"
)

// ModelBand is where a filesystem's free space sits against the policy.
type ModelBand string

const (
	ModelBandOK     ModelBand = "ok"
	ModelBandNotify ModelBand = "notify"
	ModelBandEvict  ModelBand = "evict"
)

// FilesystemSpace is one filesystem's capacity, keyed by device.
type FilesystemSpace struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	FreeBytes  int64  `json:"free_bytes"`
	TotalBytes int64  `json:"total_bytes"`
}

func (s FilesystemSpace) freePercent() float64 {
	if s.TotalBytes <= 0 {
		return 0
	}
	return float64(s.FreeBytes) * 100 / float64(s.TotalBytes)
}

// ModelStoreObservation is everything the decision reads about one store.
type ModelStoreObservation struct {
	Resource    string
	Inventory   modelretention.Inventory
	Usage       modelretention.UsageDocument
	BudgetBytes int64
	Space       FilesystemSpace
	Err         error
}

// ModelCandidate is one installed model in the ranking.
type ModelCandidate struct {
	Resource       string        `json:"resource"`
	Name           string        `json:"name"`
	Digest         string        `json:"digest"`
	Bytes          int64         `json:"bytes"`
	UniqueBytes    int64         `json:"unique_bytes"`
	LastActivity   time.Time     `json:"last_activity"`
	IdleFor        time.Duration `json:"idle_for_ns"`
	Eligible       bool          `json:"eligible"`
	Held           string        `json:"held,omitempty"`
	Score          float64       `json:"score"`
	Reacquire      []string      `json:"reacquire"`
	ReacquireExact bool          `json:"reacquire_exact"`
	SelectedFor    string        `json:"selected_for,omitempty"`
}

// ModelStoreSummary is one store's footprint against its budget.
type ModelStoreSummary struct {
	Resource        string `json:"resource"`
	StoreBytes      int64  `json:"store_bytes"`
	BudgetBytes     int64  `json:"budget_bytes"`
	OverBudgetBytes int64  `json:"over_budget_bytes"`
	Filesystem      string `json:"filesystem"`
	UsageTracked    bool   `json:"usage_tracked"`
	Error           string `json:"error,omitempty"`
}

// ModelFilesystemSummary is one filesystem's band.
type ModelFilesystemSummary struct {
	FilesystemSpace
	FreePercent          float64   `json:"free_percent"`
	Band                 ModelBand `json:"band"`
	ProjectedFreePercent float64   `json:"projected_free_percent"`
	Shortfall            string    `json:"shortfall,omitempty"`
}

// ModelEvictionDecision is storage-manager's verdict at one moment.
type ModelEvictionDecision struct {
	ObservedAt    time.Time                `json:"observed_at"`
	Band          ModelBand                `json:"band"`
	Filesystems   []ModelFilesystemSummary `json:"filesystems"`
	Stores        []ModelStoreSummary      `json:"stores"`
	Candidates    []ModelCandidate         `json:"candidates"`
	Selected      []ModelCandidate         `json:"selected"`
	SelectedBytes int64                    `json:"selected_bytes"`
}

// OverBudget lists stores above their budget.
func (d ModelEvictionDecision) OverBudget() []ModelStoreSummary {
	var out []ModelStoreSummary
	for _, s := range d.Stores {
		if s.OverBudgetBytes > 0 {
			out = append(out, s)
		}
	}
	return out
}

// Errors lists stores that could not be observed.
func (d ModelEvictionDecision) Errors() []ModelStoreSummary {
	var out []ModelStoreSummary
	for _, s := range d.Stores {
		if s.Error != "" {
			out = append(out, s)
		}
	}
	return out
}

func bandRank(b ModelBand) int {
	switch b {
	case ModelBandEvict:
		return 2
	case ModelBandNotify:
		return 1
	default:
		return 0
	}
}

// DecideModelEvictions is the pure decision. It ranks every installed model,
// then selects eligible ones first to bring each over-budget store under its
// budget and then, on any filesystem below the evict band, until projected
// free space reaches the evict-until level. Disk eviction is independent of
// in-memory unload: a loaded model is never a candidate.
func DecideModelEvictions(policy ModelRetentionPolicy, stores []ModelStoreObservation, now time.Time) ModelEvictionDecision {
	d := ModelEvictionDecision{ObservedAt: now.UTC(), Band: ModelBandOK}
	spaces := map[string]FilesystemSpace{}
	var fsOrder []string
	for _, store := range stores {
		summary := ModelStoreSummary{Resource: store.Resource, BudgetBytes: store.BudgetBytes, Filesystem: store.Space.ID, UsageTracked: !store.Usage.TrackingSince.IsZero()}
		if store.Err != nil {
			summary.Error = store.Err.Error()
			d.Stores = append(d.Stores, summary)
			continue
		}
		summary.StoreBytes = store.Inventory.StoreBytes
		if store.BudgetBytes > 0 && summary.StoreBytes > store.BudgetBytes {
			summary.OverBudgetBytes = summary.StoreBytes - store.BudgetBytes
		}
		d.Stores = append(d.Stores, summary)
		if _, ok := spaces[store.Space.ID]; !ok && store.Space.ID != "" {
			spaces[store.Space.ID] = store.Space
			fsOrder = append(fsOrder, store.Space.ID)
		}
		for _, m := range store.Inventory.Models {
			d.Candidates = append(d.Candidates, rankModel(policy, store, m, now))
		}
	}
	sort.SliceStable(d.Candidates, func(i, j int) bool {
		a, b := d.Candidates[i], d.Candidates[j]
		if a.Eligible != b.Eligible {
			return a.Eligible
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.IdleFor != b.IdleFor {
			return a.IdleFor > b.IdleFor
		}
		return a.Resource+"/"+a.Name < b.Resource+"/"+b.Name
	})
	storeFS := map[string]string{}
	for _, s := range d.Stores {
		storeFS[s.Resource] = s.Filesystem
	}
	freed := map[string]int64{} // filesystem -> selected unique bytes
	selectFor := func(i int, reason string) {
		d.Candidates[i].SelectedFor = reason
		freed[storeFS[d.Candidates[i].Resource]] += d.Candidates[i].UniqueBytes
		d.SelectedBytes += d.Candidates[i].UniqueBytes
	}
	for _, s := range d.Stores {
		need := s.OverBudgetBytes
		for i := range d.Candidates {
			c := d.Candidates[i]
			if need <= 0 {
				break
			}
			if c.Resource != s.Resource || !c.Eligible || c.SelectedFor != "" || c.UniqueBytes <= 0 {
				continue
			}
			selectFor(i, "budget")
			need -= c.UniqueBytes
		}
	}
	for _, id := range fsOrder {
		space := spaces[id]
		summary := ModelFilesystemSummary{FilesystemSpace: space, FreePercent: space.freePercent(), Band: ModelBandOK}
		switch {
		case summary.FreePercent < policy.EvictBelowFreePercent:
			summary.Band = ModelBandEvict
		case summary.FreePercent < policy.NotifyBelowFreePercent:
			summary.Band = ModelBandNotify
		}
		if summary.Band == ModelBandEvict {
			target := int64(policy.EvictUntilFreePercent / 100 * float64(space.TotalBytes))
			for i := range d.Candidates {
				if space.FreeBytes+freed[id] >= target {
					break
				}
				c := d.Candidates[i]
				if storeFS[c.Resource] != id || !c.Eligible || c.SelectedFor != "" || c.UniqueBytes <= 0 {
					continue
				}
				selectFor(i, "free-space")
			}
			if space.FreeBytes+freed[id] < target {
				summary.Shortfall = fmt.Sprintf("free space stays below %.0f%% after every eligible model is selected; %d more bytes are needed from other owners", policy.EvictUntilFreePercent, target-space.FreeBytes-freed[id])
			}
		}
		if space.TotalBytes > 0 {
			summary.ProjectedFreePercent = float64(space.FreeBytes+freed[id]) * 100 / float64(space.TotalBytes)
		}
		if bandRank(summary.Band) > bandRank(d.Band) {
			d.Band = summary.Band
		}
		d.Filesystems = append(d.Filesystems, summary)
	}
	for _, c := range d.Candidates {
		if c.SelectedFor != "" {
			d.Selected = append(d.Selected, c)
		}
	}
	return d
}

func rankModel(policy ModelRetentionPolicy, store ModelStoreObservation, m modelretention.InventoryModel, now time.Time) ModelCandidate {
	c := ModelCandidate{
		Resource: store.Resource, Name: m.Name, Digest: m.Digest, Bytes: m.Bytes, UniqueBytes: m.UniqueBytes,
		Reacquire: m.Reacquire, ReacquireExact: m.ReacquireExact,
	}
	last := store.Usage.LastActivity(m.Name)
	if m.ModifiedAt.After(last) {
		last = m.ModifiedAt
	}
	c.LastActivity = last
	if !last.IsZero() {
		c.IdleFor = now.Sub(last)
	}
	penalty := 1.0
	if !m.ReacquireExact {
		penalty = policy.RegistryTagRefetchPenalty
	}
	c.Score = float64(m.UniqueBytes) / penalty
	switch {
	case m.Required:
		c.Held = "required: " + m.RequiredReason
	case m.Blocked != "":
		c.Held = "owner refuses: " + m.Blocked
	case m.Loaded:
		c.Held = "loaded in memory"
	case strings.TrimSpace(m.Digest) == "":
		c.Held = "no digest reported"
	case store.Usage.TrackingSince.IsZero():
		c.Held = "usage unknown: the store's activity edge has not recorded traffic yet"
	case c.IdleFor < policy.IdleWindow && last.Equal(store.Usage.TrackingSince):
		c.Held = fmt.Sprintf("usage tracked for only %s; idle window is %s", roundDays(c.IdleFor), roundDays(policy.IdleWindow))
	case c.IdleFor < policy.IdleWindow:
		c.Held = fmt.Sprintf("used or pulled %s ago; idle window is %s", roundDays(c.IdleFor), roundDays(policy.IdleWindow))
	case m.UniqueBytes <= 0:
		c.Held = "frees no bytes: every blob is shared with another tag"
	default:
		c.Eligible = true
	}
	return c
}

func roundDays(d time.Duration) string {
	if d < time.Hour {
		return d.Round(time.Minute).String()
	}
	if d < 48*time.Hour {
		return d.Round(time.Hour).String()
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
