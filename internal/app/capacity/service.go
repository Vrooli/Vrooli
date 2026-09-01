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
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/buildinfo"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

// Store is the ledger surface the service needs (the engine's two repositories
// plus Close). It is an interface so tests inject an in-memory or temp store.
type Store interface {
	engine.ClaimRepository
	engine.PolicyRepository
	GCTerminalClaims(ctx context.Context, olderThan time.Time) (engine.GCResult, error)
	Close() error
}

// Service performs capacity operations for the CLI. The three seams (OpenStore,
// Source, Attributor) are injectable; nil values resolve production defaults.
type Service struct {
	OpenStore  func(ctx context.Context) (Store, error)
	Source     engine.CapacitySource
	Attributor engine.Attributor
	// Exec is the adopter callback used when enforce mode must reclaim idle
	// lower-priority capacity. Nil uses the production resource-CLI executor;
	// tests inject a recorder so broker actuation remains hermetic.
	Exec  engine.ApplyExecutor
	Clock func() time.Time
	// SourceRoot is the Vrooli source root used to resolve a resource's declared
	// capacity block for claim-on-observe adoption (§Phase 6). Empty falls back to
	// the canonical source-root resolver; when neither resolves, adoption is a
	// no-op (the maintenance pass remains the always-on adoption driver).
	SourceRoot string
}

func (s Service) sourceRoot() string {
	if strings.TrimSpace(s.SourceRoot) != "" {
		return s.SourceRoot
	}
	root, err := buildinfo.ResolveSourceRoot()
	if err != nil {
		// Source-root discovery is optional for claim adoption; failure preserves
		// the existing no-op behavior while using the canonical resolver.
		return ""
	}
	return root
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
	ClaimID        string `json:"claim_id"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	InstanceID     string `json:"instance_id,omitempty"`
	ResourceKind   string `json:"resource_kind"`
	GPUIndex       *int   `json:"gpu_index,omitempty"`
	AmountBytes    int64  `json:"amount_bytes"`
	PreferredBytes int64  `json:"preferred_bytes"`
	FloorBytes     int64  `json:"floor_bytes"`
	Priority       int    `json:"priority"`
	PriorityTier   string `json:"priority_tier"`
	Protected      bool   `json:"protected"`
	YieldWhenIdle  bool   `json:"yield_when_idle"`
	Status         string `json:"status"`
	ActivityState  string `json:"activity_state"`
	Generation     int64  `json:"generation"`
	CreatedAt      string `json:"created_at"`
	// UpdatedAt is the terminal-transition timestamp for terminal claims;
	// active claims use it as their last mutation time.
	UpdatedAt         string  `json:"updated_at"`
	LastActiveAt      *string `json:"last_active_at,omitempty"`
	ObservedBytes     int64   `json:"observed_bytes"`
	ObservedPeakBytes int64   `json:"observed_peak_bytes"`
	ObservedAt        *string `json:"observed_at,omitempty"`
	IdleUnloadTTL     string  `json:"idle_unload_ttl,omitempty"`
	IdleGrace         string  `json:"idle_grace,omitempty"`
	IdleReclaimState  string  `json:"idle_reclaim_state,omitempty"`
}

func viewClaim(c engine.CapacityClaim, policy engine.Policy, now time.Time) ClaimView {
	v := ClaimView{
		ClaimID:           c.ClaimID,
		OwnerKind:         c.OwnerKind,
		OwnerID:           c.OwnerID,
		InstanceID:        c.InstanceID,
		ResourceKind:      c.ResourceKind,
		GPUIndex:          c.GPUIndex,
		AmountBytes:       c.AmountBytes,
		PreferredBytes:    c.PreferredBytes,
		FloorBytes:        c.FloorBytes,
		Priority:          c.Priority,
		PriorityTier:      engine.PriorityTierName(c.Priority),
		CreatedAt:         c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         c.UpdatedAt.UTC().Format(time.RFC3339),
		Protected:         c.Protected,
		YieldWhenIdle:     c.YieldWhenIdle,
		Status:            c.Status,
		ActivityState:     c.ActivityState,
		Generation:        c.Generation,
		ObservedBytes:     c.ObservedBytes,
		ObservedPeakBytes: c.ObservedPeakBytes,
	}
	if c.LastActiveAt != nil {
		ts := c.LastActiveAt.UTC().Format(time.RFC3339)
		v.LastActiveAt = &ts
	}
	if c.ObservedAt != nil {
		ts := c.ObservedAt.UTC().Format(time.RFC3339)
		v.ObservedAt = &ts
	}
	if c.IdleUnloadTTL > 0 {
		v.IdleUnloadTTL = c.IdleUnloadTTL.String()
	}
	grace := c.IdleGrace
	if grace <= 0 {
		grace = policy.IdleGrace
	}
	if grace > 0 {
		v.IdleGrace = grace.String()
	}
	v.IdleReclaimState = idleReclaimState(c, grace, now)
	return v
}

func idleReclaimState(c engine.CapacityClaim, grace time.Duration, now time.Time) string {
	if !engine.IsActiveClaimStatus(c.Status) {
		return "terminal"
	}
	if c.ActivityState != engine.ActivityIdle {
		return "active"
	}
	since := c.LastActiveAt
	if since == nil {
		since = &c.CreatedAt
	}
	if grace > 0 && now.Before(since.Add(grace)) {
		return "warm_idle"
	}
	return "cold_idle"
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
	YieldWhenIdle  bool
	IdleUnloadTTL  time.Duration
	IdleGrace      time.Duration
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
		YieldWhenIdle:  req.YieldWhenIdle,
		IdleUnloadTTL:  req.IdleUnloadTTL,
		IdleGrace:      req.IdleGrace,
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
	var ledger []engine.CapacityClaim
	if snapErr != nil {
		verdict = engine.Verdict{Kind: engine.VerdictGrant, GrantedBytes: req.PreferredBytes, Warnings: []string{"capacity sensing unavailable: " + snapErr.Error()}}
	} else {
		var listErr error
		ledger, listErr = store.ListClaims(ctx, engine.ClaimFilter{ResourceKind: req.ResourceKind, Statuses: engine.ActiveClaimStatuses()})
		if listErr != nil {
			return ClaimOutput{}, listErr
		}
		verdict = engine.Decide(engReq, snapshot, ledger, policy, s.now())
	}
	// A CLI claim is a broker admission too, not merely a ledger insert. In
	// enforce mode, execute the declared adopter callbacks before recording the
	// requester so a grant that depends on reclaiming idle capacity is real.
	effective := policy.EffectiveEnforce(envEnforce())
	if effective == engine.EnforceOn && verdict.Granted() && verdict.ReclaimBytes > 0 {
		_, actuation, actErr := engine.EnforceReclaim(ctx, store, engReq.Priority, verdict, ledger, s.Exec, policy, effective, s.now())
		if actErr != nil {
			verdict.Warnings = append(verdict.Warnings, "capacity actuation error: "+actErr.Error())
		}
		for _, outcome := range actuation.Outcomes {
			if outcome.Err != "" {
				verdict.Warnings = append(verdict.Warnings, fmt.Sprintf("degrade of %q failed: %s", outcome.OwnerID, outcome.Err))
			}
		}
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
		YieldWhenIdle:  req.YieldWhenIdle,
		IdleUnloadTTL:  req.IdleUnloadTTL,
		IdleGrace:      req.IdleGrace,
		Status:         status,
		DegradeProfile: profile,
	}
	created, err := store.CreateClaim(ctx, claim, req.TTL)
	if err != nil {
		return ClaimOutput{}, err
	}
	return ClaimOutput{Verdict: verdict, Claim: viewClaim(created, policy, s.now()), Enforce: effective}, nil
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

// Resize changes an existing claim's amount in place. It is the operation an
// adopter uses when its real footprint moves, so the ledger keeps one row and
// one observed-usage history per resource lifetime instead of one row per
// change.
func (s Service) Resize(ctx context.Context, ref Ref) (ClaimView, error) {
	return s.mutate(ctx, func(store Store) (engine.CapacityClaim, error) {
		generation := ref.Generation
		if generation == 0 {
			current, err := store.GetClaim(ctx, ref.ClaimID)
			if err != nil {
				return engine.CapacityClaim{}, err
			}
			generation = current.Generation
		}
		return store.ResizeClaim(ctx, ref.ClaimID, generation, ref.AmountByte)
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
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return ClaimView{}, err
	}
	claim, err := fn(store)
	if err != nil {
		return ClaimView{}, err
	}
	return viewClaim(claim, policy, s.now()), nil
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
	AllHistory bool
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
	// Opportunistic, debounced resident-claim sweep (§8.6) so a list reflects
	// presence-refreshed/expired claims during active periods. Best-effort: a
	// sensing failure or cursor-less store is a silent no-op.
	if policy, perr := store.GetPolicy(ctx); perr == nil {
		_, _, _ = engine.MaybeSweep(ctx, store, s.source(), s.attributor(), policy, s.now())
	}
	filter := engine.ClaimFilter{OwnerID: req.OwnerID}
	if req.ActiveOnly || !req.AllHistory {
		filter.Statuses = engine.ActiveClaimStatuses()
	}
	claims, err := store.ListClaims(ctx, filter)
	if err != nil {
		return ListOutput{}, err
	}
	out := ListOutput{Claims: make([]ClaimView, 0, len(claims))}
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return ListOutput{}, err
	}
	now := s.now()
	for _, c := range claims {
		out.Claims = append(out.Claims, viewClaim(c, policy, now))
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
	// Opportunistic, debounced sweep (§8.6) reusing the snapshot we just took, so
	// reconcile classifies against presence-refreshed/expired claims rather than a
	// ledger that still lists a dead resident. Best-effort on a cursor-backed store.
	if cursor, ok := any(store).(engine.SweepCursorStore); ok {
		_, _, _ = engine.SweepIfDue(ctx, cursor, snapshot, s.attributor(), policy, s.now())
	}
	ledger, err := store.ListClaims(ctx, engine.ClaimFilter{Statuses: engine.ActiveClaimStatuses()})
	if err != nil {
		return ReconcileOutput{}, err
	}
	findings := engine.Reconcile(ctx, snapshot, ledger, s.attributor(), policy)
	// A declaration without a claim is actionable only when its resource is
	// actually resident. Installed-but-stopped resources cannot hold VRAM and
	// must not create a permanent warning; observed-but-unclaimed residents are
	// already covered by the adoption/reconcile path above.
	observedResources := make(map[string]bool)
	attr := s.attributor()
	for _, proc := range snapshot.GPUProcesses {
		if int64(proc.UsedBytes) < policy.TrackingThreshold {
			continue
		}
		attribution := attr.Attribute(ctx, proc.PID)
		for _, owner := range []string{
			attribution.OwnerID,
			engine.NormalizeOwnerName(attribution.ContainerName),
			engine.NormalizeProcessOwner(proc.ProcessName),
		} {
			if owner = strings.TrimSpace(owner); owner != "" && owner != engine.OwnerUnknown {
				observedResources[owner] = true
			}
		}
	}
	declared, err := engine.DeclaredGPUWithoutClaimFindings(s.sourceRoot(), ledger, func(resource string) bool {
		return s.resourceInstalled(resource) && observedResources[resource]
	})
	if err != nil {
		return ReconcileOutput{}, err
	}
	findings = append(findings, declared...)
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
	Adopted   []SweepView `json:"adopted,omitempty"`
	// IdleUnloadCandidates are claims idle beyond their idle_unload_ttl that the
	// broker WOULD autonomously unload (advisory). The CLI sweep verb only reports
	// them — actuation happens solely in the always-on maintenance pass under
	// enforce=on. Empty unless a resource declares idle_unload_ttl (or the
	// default_idle_unload_ttl lever is set).
	IdleUnloadCandidates []SweepView `json:"idle_unload_candidates,omitempty"`
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
	now := s.now()
	result, err := engine.Sweep(ctx, store, snapshot, s.attributor(), policy, now)
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
	// Claim-on-observe adoption (§Phase 6): adopt declared-but-unclaimed residents
	// into the ledger from their resource.json capacity block (idempotent,
	// declared-only, advisory-safe). Skipped when no source root resolves.
	if root := s.sourceRoot(); root != "" {
		loadSpec := func(name string) (engine.ResourceClaimSpec, bool, error) {
			return engine.LoadResourceClaimSpec(root, name)
		}
		if adopted, adoptErr := engine.AdoptObservedResidents(ctx, store, snapshot, s.attributor(), policy, loadSpec, now); adoptErr == nil {
			for _, c := range adopted {
				out.Adopted = append(out.Adopted, SweepView{ClaimID: c.ClaimID, OwnerID: c.OwnerID, Status: c.Status})
			}
		}
	}
	// Advisory idle-unload candidates (§Phase 3): report (never actuate) which
	// active claims are idle beyond their idle_unload_ttl. PlanIdleUnload is a pure
	// planner; the CLI verb only surfaces the would-unload — actuation lives in the
	// always-on maintenance pass under enforce=on.
	if active, listErr := store.ListClaims(ctx, engine.ClaimFilter{Statuses: engine.ActiveClaimStatuses()}); listErr == nil {
		for _, a := range engine.PlanIdleUnload(active, policy, now).Actions {
			out.IdleUnloadCandidates = append(out.IdleUnloadCandidates, SweepView{ClaimID: a.ClaimID, OwnerID: a.OwnerID, Status: a.ToStep})
		}
	}
	return out, nil
}

// RecommendRequest narrows the right-sizing scan.
type RecommendRequest struct {
	OwnerID string
}

// RecommendOutput is the advisory right-sizing suggestion set.
type RecommendOutput struct {
	Recommendations []engine.Recommendation `json:"recommendations"`
}

// Recommend compares active VRAM claims' declared reservations against their
// observed peaks and returns advisory right-sizing suggestions (§Phase 4). It
// never applies anything (contract C7) and stays silent for claims without
// enough observed data. It runs an opportunistic, debounced sweep first so the
// peaks it reads are fresh.
func (s Service) Recommend(ctx context.Context, req RecommendRequest) (RecommendOutput, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return RecommendOutput{}, err
	}
	defer store.Close()
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return RecommendOutput{}, err
	}
	// Best-effort fresh sample before reading peaks (cursor-backed + sensing only).
	_, _, _ = engine.MaybeSweep(ctx, store, s.source(), s.attributor(), policy, s.now())
	filter := engine.ClaimFilter{OwnerID: req.OwnerID, Statuses: engine.ActiveClaimStatuses()}
	claims, err := store.ListClaims(ctx, filter)
	if err != nil {
		return RecommendOutput{}, err
	}
	return RecommendOutput{Recommendations: engine.Recommend(claims, policy)}, nil
}

// GCOutput reports what terminal-claim GC pruned.
type GCOutput struct {
	Pruned         int    `json:"pruned"`
	Bytes          int64  `json:"bytes"`
	RetentionSpent string `json:"retention"`
}

// GC prunes terminal (released/expired/preempted) claims older than the
// terminal_retention lever. It is always safe — it frees no live capacity, only
// trimming dead history (plan §Phase 1). A retention of 0 disables GC.
func (s Service) GC(ctx context.Context) (GCOutput, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return GCOutput{}, err
	}
	defer store.Close()
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return GCOutput{}, err
	}
	out := GCOutput{RetentionSpent: policy.TerminalRetention.String()}
	if policy.TerminalRetention <= 0 {
		return out, nil
	}
	res, err := store.GCTerminalClaims(ctx, s.now().Add(-policy.TerminalRetention))
	if err != nil {
		return GCOutput{}, err
	}
	out.Pruned = res.Count
	out.Bytes = res.Bytes
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

// resourceInstalled reports whether a resource has anything on this host to
// hold a claim with. It is deliberately evidence-based rather than a
// configuration read: a resource listed as enabled but never installed still
// cannot reserve VRAM.
func (s Service) resourceInstalled(resource string) bool {
	root := strings.TrimSpace(s.sourceRoot())
	if root == "" || strings.TrimSpace(resource) == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}
	// A managed-service resource stages its artifact in the per-user store; a
	// compose or docker resource keeps its declared data directory. Either is
	// evidence the resource exists here.
	for _, candidate := range []string{
		filepath.Join(home, repocontractmeta.ProjectConfigDir, "artifacts", resource),
		filepath.Join(root, "resources", resource, "data"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}
