// Package capacity (import alias capacityapp) is the application service behind
// the `vrooli capacity` CLI group. It wires the internal/capacity engine
// (claim ledger + Decide + Reconcile) to typed request/response shapes the CLI
// parses and renders. The engine is imported as `engine`.
package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	engine "github.com/vrooli/vrooli/internal/capacity"
)

// Store is the ledger surface the service needs (the engine's two repositories
// plus Close). It is an interface so tests inject an in-memory or temp store.
type Store interface {
	engine.ClaimRepository
	engine.PolicyRepository
	Close() error
}

// Service performs capacity operations for the CLI. The three seams (OpenStore,
// Source, Attributor) are injectable; nil values resolve production defaults.
type Service struct {
	OpenStore  func(ctx context.Context) (Store, error)
	Source     engine.CapacitySource
	Attributor engine.Attributor
	Clock      func() time.Time
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}

func (s Service) openStore(ctx context.Context) (Store, error) {
	if s.OpenStore != nil {
		return s.OpenStore(ctx)
	}
	return engine.NewSQLiteStore(ctx, engine.Config{})
}

func (s Service) source() engine.CapacitySource {
	if s.Source != nil {
		return s.Source
	}
	return engine.HostInventorySource{}
}

func (s Service) attributor() engine.Attributor {
	if s.Attributor != nil {
		return s.Attributor
	}
	return engine.NewDockerAttributor()
}

// ClaimView is the JSON/text projection of a claim.
type ClaimView struct {
	ClaimID        string  `json:"claim_id"`
	OwnerKind      string  `json:"owner_kind"`
	OwnerID        string  `json:"owner_id"`
	InstanceID     string  `json:"instance_id,omitempty"`
	ResourceKind   string  `json:"resource_kind"`
	GPUIndex       *int    `json:"gpu_index,omitempty"`
	AmountBytes    int64   `json:"amount_bytes"`
	PreferredBytes int64   `json:"preferred_bytes"`
	FloorBytes     int64   `json:"floor_bytes"`
	Priority       int     `json:"priority"`
	PriorityTier   string  `json:"priority_tier"`
	Protected      bool    `json:"protected"`
	Status         string  `json:"status"`
	ActivityState  string  `json:"activity_state"`
	Generation     int64   `json:"generation"`
	LastActiveAt   *string `json:"last_active_at,omitempty"`
}

func viewClaim(c engine.CapacityClaim) ClaimView {
	v := ClaimView{
		ClaimID:        c.ClaimID,
		OwnerKind:      c.OwnerKind,
		OwnerID:        c.OwnerID,
		InstanceID:     c.InstanceID,
		ResourceKind:   c.ResourceKind,
		GPUIndex:       c.GPUIndex,
		AmountBytes:    c.AmountBytes,
		PreferredBytes: c.PreferredBytes,
		FloorBytes:     c.FloorBytes,
		Priority:       c.Priority,
		PriorityTier:   engine.PriorityTierName(c.Priority),
		Protected:      c.Protected,
		Status:         c.Status,
		ActivityState:  c.ActivityState,
		Generation:     c.Generation,
	}
	if c.LastActiveAt != nil {
		ts := c.LastActiveAt.UTC().Format(time.RFC3339)
		v.LastActiveAt = &ts
	}
	return v
}

// ClaimRequest is the parsed `vrooli capacity claim` input.
type ClaimRequest struct {
	OwnerKind      string
	OwnerID        string
	InstanceID     string
	ResourceKind   string
	GPUIndex       *int
	PreferredBytes int64
	FloorBytes     int64
	PriorityTier   string
	Protected      bool
	ProfileJSON    string
	TTL            time.Duration
}

// ClaimOutput carries the granted claim plus the advisory verdict.
type ClaimOutput struct {
	Verdict engine.Verdict `json:"verdict"`
	Claim   ClaimView      `json:"claim"`
	Enforce string         `json:"enforce"`
}

// Claim runs the admission decision and records the resulting claim. In
// advisory mode the claim is recorded at the requested preferred amount even on
// a non-grant verdict (the caller proceeds and chooses its own fallback); the
// verdict and its warnings are always returned.
func (s Service) Claim(ctx context.Context, req ClaimRequest) (ClaimOutput, error) {
	if strings.TrimSpace(req.OwnerID) == "" {
		return ClaimOutput{}, fmt.Errorf("--owner-id is required")
	}
	if req.ResourceKind == "" {
		req.ResourceKind = engine.ResourceKindVRAM
	}
	var profile *engine.DegradeProfile
	if strings.TrimSpace(req.ProfileJSON) != "" {
		profile = &engine.DegradeProfile{}
		if err := json.Unmarshal([]byte(req.ProfileJSON), profile); err != nil {
			return ClaimOutput{}, fmt.Errorf("parse --profile: %w", err)
		}
	}
	engReq := engine.CapacityRequest{
		OwnerKind:      req.OwnerKind,
		OwnerID:        req.OwnerID,
		InstanceID:     req.InstanceID,
		ResourceKind:   req.ResourceKind,
		GPUIndex:       req.GPUIndex,
		PreferredBytes: req.PreferredBytes,
		FloorBytes:     req.FloorBytes,
		Priority:       engine.ParsePriorityTier(req.PriorityTier),
		Protected:      req.Protected,
		Profile:        profile,
		TTL:            req.TTL,
	}

	store, err := s.openStore(ctx)
	if err != nil {
		return ClaimOutput{}, err
	}
	defer store.Close()

	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return ClaimOutput{}, err
	}
	snapshot, snapErr := s.source().Snapshot(ctx)
	// A snapshot error is not fatal in advisory mode — we still record the claim,
	// but we surface the sensing failure as a warning via a deny-ish verdict.
	var verdict engine.Verdict
	if snapErr != nil {
		verdict = engine.Verdict{Kind: engine.VerdictGrant, GrantedBytes: req.PreferredBytes, Warnings: []string{"capacity sensing unavailable: " + snapErr.Error()}}
	} else {
		ledger, listErr := store.ListClaims(ctx, engine.ClaimFilter{ResourceKind: req.ResourceKind, Statuses: engine.ActiveClaimStatuses()})
		if listErr != nil {
			return ClaimOutput{}, listErr
		}
		verdict = engine.Decide(engReq, snapshot, ledger, policy, s.now())
	}

	amount := req.PreferredBytes
	status := engine.StatusGranted
	if verdict.Granted() {
		amount = verdict.GrantedBytes
		if verdict.Kind == engine.VerdictDegrade {
			status = engine.StatusDegraded
		}
	}

	claim := engine.CapacityClaim{
		OwnerKind:      req.OwnerKind,
		OwnerID:        req.OwnerID,
		InstanceID:     req.InstanceID,
		ResourceKind:   req.ResourceKind,
		GPUIndex:       req.GPUIndex,
		AmountBytes:    amount,
		PreferredBytes: req.PreferredBytes,
		FloorBytes:     req.FloorBytes,
		Priority:       engReq.Priority,
		Protected:      req.Protected,
		Status:         status,
		DegradeProfile: profile,
	}
	created, err := store.CreateClaim(ctx, claim, req.TTL)
	if err != nil {
		return ClaimOutput{}, err
	}
	return ClaimOutput{Verdict: verdict, Claim: viewClaim(created), Enforce: policy.EffectiveEnforce(envEnforce())}, nil
}

// Ref identifies a single claim by ID, with the generation for guarded ops.
type Ref struct {
	ClaimID    string
	Generation int64
	TTL        time.Duration
	State      string // for activity
	ToStep     string // for degrade
	AmountByte int64  // for degrade
	Reason     string // for preempt
}

// Heartbeat renews a claim's liveness.
func (s Service) Heartbeat(ctx context.Context, ref Ref) (ClaimView, error) {
	return s.mutate(ctx, func(store Store) (engine.CapacityClaim, error) {
		return store.HeartbeatClaim(ctx, ref.ClaimID, ref.Generation, ref.TTL)
	})
}

// Activity reports the work-owner activity state. When the caller does not
// supply a generation (Generation==0, which is never a live claim generation),
// the current generation is resolved first: the work-owner is the truth source
// for its own activity, so last-writer-wins is correct and consumers should not
// have to fight the optimistic-concurrency guard for a simple active/idle report.
func (s Service) Activity(ctx context.Context, ref Ref) (ClaimView, error) {
	return s.mutate(ctx, func(store Store) (engine.CapacityClaim, error) {
		generation := ref.Generation
		if generation == 0 {
			current, err := store.GetClaim(ctx, ref.ClaimID)
			if err != nil {
				return engine.CapacityClaim{}, err
			}
			generation = current.Generation
		}
		return store.ReportActivity(ctx, ref.ClaimID, generation, ref.State)
	})
}

// Degrade steps a claim down to a profile rung.
func (s Service) Degrade(ctx context.Context, ref Ref) (ClaimView, error) {
	return s.mutate(ctx, func(store Store) (engine.CapacityClaim, error) {
		amount := ref.AmountByte
		if amount == 0 {
			// Resolve the amount from the claim's profile by label.
			current, err := store.GetClaim(ctx, ref.ClaimID)
			if err != nil {
				return engine.CapacityClaim{}, err
			}
			resolved, ok := resolveStepAmount(current, ref.ToStep)
			if !ok {
				return engine.CapacityClaim{}, fmt.Errorf("step %q not found in claim profile", ref.ToStep)
			}
			amount = resolved
			ref.Generation = current.Generation
		}
		return store.DegradeClaim(ctx, ref.ClaimID, ref.Generation, ref.ToStep, amount)
	})
}

// Release terminates a claim cleanly.
func (s Service) Release(ctx context.Context, ref Ref) (ClaimView, error) {
	return s.mutate(ctx, func(store Store) (engine.CapacityClaim, error) {
		return store.ReleaseClaim(ctx, ref.ClaimID)
	})
}

func (s Service) mutate(ctx context.Context, fn func(Store) (engine.CapacityClaim, error)) (ClaimView, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return ClaimView{}, err
	}
	defer store.Close()
	claim, err := fn(store)
	if err != nil {
		return ClaimView{}, err
	}
	return viewClaim(claim), nil
}

func resolveStepAmount(c engine.CapacityClaim, label string) (int64, bool) {
	if c.DegradeProfile == nil {
		return 0, false
	}
	for _, st := range c.DegradeProfile.Steps {
		if st.Label == label {
			return st.AmountBytes, true
		}
	}
	return 0, false
}

// ListRequest narrows the claim listing.
type ListRequest struct {
	OwnerID    string
	ActiveOnly bool
}

// ListOutput is the claim listing.
type ListOutput struct {
	Claims []ClaimView `json:"claims"`
}

// List returns claims, optionally filtered to one owner / active only.
func (s Service) List(ctx context.Context, req ListRequest) (ListOutput, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return ListOutput{}, err
	}
	defer store.Close()
	filter := engine.ClaimFilter{OwnerID: req.OwnerID}
	if req.ActiveOnly {
		filter.Statuses = engine.ActiveClaimStatuses()
	}
	claims, err := store.ListClaims(ctx, filter)
	if err != nil {
		return ListOutput{}, err
	}
	out := ListOutput{Claims: make([]ClaimView, 0, len(claims))}
	for _, c := range claims {
		out.Claims = append(out.Claims, viewClaim(c))
	}
	return out, nil
}

// ReconcileOutput is the reconciliation finding set.
type ReconcileOutput struct {
	Findings []engine.Finding `json:"findings"`
}

// Reconcile classifies observed GPU consumers against the ledger.
func (s Service) Reconcile(ctx context.Context) (ReconcileOutput, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return ReconcileOutput{}, err
	}
	defer store.Close()
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return ReconcileOutput{}, err
	}
	snapshot, err := s.source().Snapshot(ctx)
	if err != nil {
		return ReconcileOutput{}, fmt.Errorf("capacity sensing unavailable: %w", err)
	}
	ledger, err := store.ListClaims(ctx, engine.ClaimFilter{Statuses: engine.ActiveClaimStatuses()})
	if err != nil {
		return ReconcileOutput{}, err
	}
	findings := engine.Reconcile(ctx, snapshot, ledger, s.attributor(), policy)
	return ReconcileOutput{Findings: findings}, nil
}

// SweepView is the JSON/text projection of one swept claim.
type SweepView struct {
	ClaimID string `json:"claim_id"`
	OwnerID string `json:"owner_id"`
	Status  string `json:"status"`
}

// SweepOutput reports the presence-driven sweep result.
type SweepOutput struct {
	Refreshed []SweepView `json:"refreshed"`
	Expired   []SweepView `json:"expired"`
}

// Sweep runs the resident-claim heartbeat driver: it renews the heartbeat of
// active non-op claims whose owner is still observed holding GPU memory, then
// expires every active claim whose deadline lapsed with no observed owner. This
// is the engine-gap closer for resident model-server containers that cannot
// heartbeat themselves. It mutates the ledger but never enforces.
func (s Service) Sweep(ctx context.Context) (SweepOutput, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return SweepOutput{}, err
	}
	defer store.Close()
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return SweepOutput{}, err
	}
	snapshot, err := s.source().Snapshot(ctx)
	if err != nil {
		return SweepOutput{}, fmt.Errorf("capacity sensing unavailable: %w", err)
	}
	result, err := engine.Sweep(ctx, store, snapshot, s.attributor(), policy, s.now())
	if err != nil {
		return SweepOutput{}, err
	}
	out := SweepOutput{
		Refreshed: make([]SweepView, 0, len(result.Refreshed)),
		Expired:   make([]SweepView, 0, len(result.Expired)),
	}
	for _, c := range result.Refreshed {
		out.Refreshed = append(out.Refreshed, SweepView{ClaimID: c.ClaimID, OwnerID: c.OwnerID, Status: c.Status})
	}
	for _, c := range result.Expired {
		out.Expired = append(out.Expired, SweepView{ClaimID: c.ClaimID, OwnerID: c.OwnerID, Status: c.Status})
	}
	return out, nil
}

// PolicyEntry is one key/value lever for rendering.
type PolicyEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PolicyOutput is the full policy as ordered entries.
type PolicyOutput struct {
	Entries []PolicyEntry `json:"entries"`
}

// PolicyGet returns one key (when Key set) or all levers.
func (s Service) PolicyGet(ctx context.Context, key string) (PolicyOutput, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return PolicyOutput{}, err
	}
	defer store.Close()
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return PolicyOutput{}, err
	}
	return policyOutput(policy, key)
}

// PolicySet validates and persists one lever, returning the resulting policy.
func (s Service) PolicySet(ctx context.Context, key, value string) (PolicyOutput, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return PolicyOutput{}, err
	}
	defer store.Close()
	policy, err := store.SetPolicyKey(ctx, key, value)
	if err != nil {
		return PolicyOutput{}, err
	}
	return policyOutput(policy, "")
}

func policyOutput(policy engine.Policy, key string) (PolicyOutput, error) {
	if strings.TrimSpace(key) != "" {
		val, err := policy.Get(key)
		if err != nil {
			return PolicyOutput{}, err
		}
		return PolicyOutput{Entries: []PolicyEntry{{Key: key, Value: val}}}, nil
	}
	out := PolicyOutput{}
	for _, k := range engine.PolicyKeys {
		val, err := policy.Get(k)
		if err != nil {
			return PolicyOutput{}, err
		}
		out.Entries = append(out.Entries, PolicyEntry{Key: k, Value: val})
	}
	return out, nil
}

func envEnforce() string {
	return strings.TrimSpace(os.Getenv(engine.EnvEnforce))
}
