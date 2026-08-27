package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// ResourceClaimSpec is the optional `capacity` block a resource declares in its
// resource.json so the lifecycle can claim on its behalf at admission. Absent
// block => the resource is not (yet) a capacity adopter and admission is a
// no-op (the dormant-by-default property that keeps Phase 3 parity-safe).
type ResourceClaimSpec struct {
	ResourceKind   string `json:"resource_kind"`
	PreferredBytes int64  `json:"preferred_bytes"`
	FloorBytes     int64  `json:"floor_bytes"`
	Priority       string `json:"priority"` // tier name
	GPUIndex       *int   `json:"gpu_index,omitempty"`
	Protected      bool   `json:"protected,omitempty"`
	// YieldWhenIdle opts the resource's claim into the idle-yield rule (§8.3): an
	// idle (beyond grace) claim yields its capacity to active work at/above the
	// idle_yield_floor. Default false keeps the strict-priority rule.
	YieldWhenIdle bool `json:"yield_when_idle,omitempty"`
	// IdleUnloadTTLSeconds opts the resource into autonomous idle-unload (§Phase 3):
	// once its claim has been idle this long the broker proactively unloads it to
	// floor (advisory logs / enforce actuates), accepting a cold start on next use.
	// 0 (the default) disables it for this resource — autonomous unload is opt-in.
	IdleUnloadTTLSeconds int `json:"idle_unload_ttl_seconds,omitempty"`
	// IdleGraceSeconds overrides the global demand-driven reclaim grace for this
	// resource. 0 keeps policy.idle_grace. Use this for warm audio/STT resources
	// that should remain resident briefly after recent use.
	IdleGraceSeconds int             `json:"idle_grace_seconds,omitempty"`
	TTLSeconds       int             `json:"ttl_seconds,omitempty"`
	Profile          *DegradeProfile `json:"profile,omitempty"`
}

// resourceManifestEnvelope is the minimal shape we decode from resource.json —
// we only care about the declared claim and never fail on the rest. The claim
// lives inside the acceleration block, so a resource cannot reserve VRAM
// without declaring the backend it needs it on.
type resourceManifestEnvelope struct {
	Acceleration *struct {
		Claim *ResourceClaimSpec `json:"claim"`
	} `json:"acceleration"`
}

// LoadResourceClaimSpec reads the optional capacity block from a resource's
// resource.json. ok is false (with nil error) when the file or block is absent.
func LoadResourceClaimSpec(root, resourceName string) (ResourceClaimSpec, bool, error) {
	path := filepath.Join(root, "resources", resourceName, "resource.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ResourceClaimSpec{}, false, nil
		}
		return ResourceClaimSpec{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	var env resourceManifestEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return ResourceClaimSpec{}, false, fmt.Errorf("parse %s acceleration.claim: %w", path, err)
	}
	if env.Acceleration == nil || env.Acceleration.Claim == nil {
		return ResourceClaimSpec{}, false, nil
	}
	return *env.Acceleration.Claim, true, nil
}

// AdmitStore is the ledger surface admission needs.
type AdmitStore interface {
	ClaimRepository
	PolicyRepository
	Close() error
}

// AdmitOptions configures a single resource admission. Seams are injectable for
// tests; nil values resolve production defaults.
type AdmitOptions struct {
	Root         string
	ResourceName string
	// EnforceEnv overrides the VROOLI_CAPACITY_ENFORCE read (tests).
	EnforceEnv string
	Source     CapacitySource
	// Attributor resolves observed GPU PIDs to owners for the presence-aware
	// admission sweep. nil resolves the production docker attributor.
	Attributor Attributor
	// Exec runs an adopter's degrade verb during enforce-mode actuation. nil
	// resolves the production CmdExecutor (shells the owner's resource CLI).
	Exec      ApplyExecutor
	OpenStore func(ctx context.Context) (AdmitStore, error)
	Clock     func() time.Time
}

// AdmissionResult reports what admission did. Skipped=true means no claim was
// recorded (no profile, or enforcement disabled) — the byte-identical path.
type AdmissionResult struct {
	Skipped bool    `json:"skipped"`
	Reason  string  `json:"reason,omitempty"`
	Enforce string  `json:"enforce,omitempty"`
	Verdict Verdict `json:"verdict,omitempty"`
	ClaimID string  `json:"claim_id,omitempty"`
}

// AdmitResource is the advisory admission hook the lifecycle calls before a
// resource starts (plan §7 Phase 3). It is ALWAYS advisory in V1: it claims +
// warns + records but NEVER blocks the start (a returned error is operational,
// not a veto — the caller logs it and proceeds). When the resource declares no
// capacity block, or enforcement is off, it is a complete no-op.
//
//nolint:gocyclo // admission evaluates resource identity, budget, lease, and contention outcomes explicitly.
func AdmitResource(ctx context.Context, opts AdmitOptions) (AdmissionResult, error) {
	enforceEnv := opts.EnforceEnv
	if strings.TrimSpace(enforceEnv) == "" {
		enforceEnv = strings.TrimSpace(os.Getenv(EnvEnforce))
	}
	// Fast parity path: explicit env-off short-circuits before touching disk/DB.
	if enforceEnv == EnforceOff {
		return AdmissionResult{Skipped: true, Reason: "enforcement disabled (env off)", Enforce: EnforceOff}, nil
	}

	spec, ok, err := LoadResourceClaimSpec(opts.Root, opts.ResourceName)
	if err != nil {
		return AdmissionResult{}, err
	}
	if !ok {
		return AdmissionResult{Skipped: true, Reason: "resource declares no capacity block"}, nil
	}

	store, err := openAdmitStore(ctx, opts)
	if err != nil {
		return AdmissionResult{}, err
	}
	defer store.Close()

	now := admitNow(opts)
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return AdmissionResult{}, err
	}
	effective := policy.EffectiveEnforce(enforceEnv)
	if effective == EnforceOff {
		return AdmissionResult{Skipped: true, Reason: "enforcement disabled (policy off)", Enforce: EnforceOff}, nil
	}

	req := spec.toRequest(opts.ResourceName)
	verdict := Verdict{Kind: VerdictGrant, GrantedBytes: req.PreferredBytes}
	var admissionWarnings []string
	existing, listErr := store.ListClaims(ctx, ClaimFilter{
		OwnerKind:    OwnerKindResource,
		OwnerID:      opts.ResourceName,
		ResourceKind: req.ResourceKind,
		Statuses:     ActiveClaimStatuses(),
	})
	if listErr != nil {
		return AdmissionResult{}, listErr
	}
	var reusable *CapacityClaim
	for _, claim := range existing {
		claim := claim
		if claimMatchesResourceRequest(claim, req) && reusable == nil {
			reusable = &claim
			continue
		}
		if _, err := store.ReleaseClaim(ctx, claim.ClaimID); err != nil {
			return AdmissionResult{}, fmt.Errorf("release stale capacity claim %s: %w", claim.ClaimID, err)
		}
		admissionWarnings = append(admissionWarnings, fmt.Sprintf("released stale capacity claim %s for manifest refresh", claim.ClaimID))
	}
	if reusable != nil {
		reused := *reusable
		if refreshed, hbErr := store.HeartbeatClaim(ctx, reused.ClaimID, reused.Generation, normalizeAdmitTTL(req.TTL)); hbErr == nil {
			reused = refreshed
		}
		verdict.Warnings = append(verdict.Warnings, admissionWarnings...)
		return AdmissionResult{Enforce: effective, Verdict: verdict, ClaimID: reused.ClaimID, Reason: "reused existing active claim"}, nil
	}

	source := opts.Source
	if source == nil {
		source = HostInventorySource{}
	}
	attr := opts.Attributor
	if attr == nil {
		attr = NewDockerAttributor()
	}
	var ledger []CapacityClaim
	snapshot, snapErr := source.Snapshot(ctx)
	if snapErr != nil {
		// Sensing unavailable: do NOT expire — a presence miss must never strand a
		// live resident. Decide on the optimistic preferred grant.
		verdict.Warnings = append(verdict.Warnings, "capacity sensing unavailable: "+snapErr.Error())
	} else {
		// Presence-aware self-clean (opportunistic sweep, §8.6): refresh resident
		// claims still observed on the GPU, then expire dead ones — so a crashed
		// prior owner's claim doesn't count against this start, but a live resident
		// whose heartbeat lapsed is rescued rather than wrongly expired.
		if _, err := Sweep(ctx, store, snapshot, attr, policy, now); err != nil {
			return AdmissionResult{}, err
		}
		listed, listErr := store.ListClaims(ctx, ClaimFilter{ResourceKind: req.ResourceKind, Statuses: ActiveClaimStatuses()})
		if listErr != nil {
			return AdmissionResult{}, listErr
		}
		ledger = listed
		verdict = Decide(req, snapshot, ledger, policy, now)
	}
	verdict.Warnings = append(verdict.Warnings, admissionWarnings...)

	// Enforce-mode actuation (§8.8): when the grant depends on reclaiming idle
	// lower-priority capacity, plan + actuate the degrade ladder BEFORE recording
	// the new claim, so the space is actually freed. Advisory/off only log the
	// plan. Actuation failure is non-fatal — it surfaces as a warning and the
	// claim is still recorded (the requester proceeds, honestly degraded).
	if effective == EnforceOn && verdict.Granted() && verdict.ReclaimBytes > 0 {
		_, actResult, actErr := EnforceReclaim(ctx, store, req.Priority, verdict, ledger, opts.Exec, policy, effective, now)
		if actErr != nil {
			verdict.Warnings = append(verdict.Warnings, "capacity actuation error: "+actErr.Error())
		}
		for _, oc := range actResult.Outcomes {
			if oc.Err != "" {
				verdict.Warnings = append(verdict.Warnings, fmt.Sprintf("degrade of %q failed: %s", oc.OwnerID, oc.Err))
			}
		}
	}

	amount := req.PreferredBytes
	status := StatusGranted
	if verdict.Granted() {
		amount = verdict.GrantedBytes
		if verdict.Kind == VerdictDegrade {
			status = StatusDegraded
		}
	}
	claim := CapacityClaim{
		OwnerKind:      OwnerKindResource,
		OwnerID:        opts.ResourceName,
		ResourceKind:   req.ResourceKind,
		GPUIndex:       req.GPUIndex,
		AmountBytes:    amount,
		PreferredBytes: req.PreferredBytes,
		FloorBytes:     req.FloorBytes,
		Priority:       req.Priority,
		Protected:      req.Protected,
		YieldWhenIdle:  req.YieldWhenIdle,
		IdleUnloadTTL:  req.IdleUnloadTTL,
		IdleGrace:      req.IdleGrace,
		Status:         status,
		DegradeProfile: req.Profile,
	}
	created, err := store.CreateClaim(ctx, claim, req.TTL)
	if err != nil {
		return AdmissionResult{}, err
	}
	return AdmissionResult{Enforce: effective, Verdict: verdict, ClaimID: created.ClaimID}, nil
}

func claimMatchesResourceRequest(claim CapacityClaim, req CapacityRequest) bool {
	return claim.ResourceKind == req.ResourceKind &&
		claim.GPUIndex == req.GPUIndex &&
		claim.PreferredBytes == req.PreferredBytes &&
		claim.FloorBytes == req.FloorBytes &&
		claim.Priority == req.Priority &&
		claim.Protected == req.Protected &&
		claim.YieldWhenIdle == req.YieldWhenIdle &&
		claim.IdleUnloadTTL == req.IdleUnloadTTL &&
		claim.IdleGrace == req.IdleGrace &&
		reflect.DeepEqual(claim.DegradeProfile, req.Profile)
}

// normalizeAdmitTTL resolves the heartbeat TTL for a reused claim: a declared
// positive TTL is honored, otherwise the engine default (the sweep keeps
// residents alive regardless, but a renewed deadline gives the next sweep slack).
func normalizeAdmitTTL(ttl time.Duration) time.Duration {
	if ttl > 0 {
		return ttl
	}
	return DefaultHeartbeatTTL
}

func (spec ResourceClaimSpec) toRequest(resourceName string) CapacityRequest {
	kind := spec.ResourceKind
	if kind == "" {
		kind = ResourceKindVRAM
	}
	ttl := time.Duration(spec.TTLSeconds) * time.Second
	return CapacityRequest{
		OwnerKind:      OwnerKindResource,
		OwnerID:        resourceName,
		ResourceKind:   kind,
		GPUIndex:       spec.GPUIndex,
		PreferredBytes: spec.PreferredBytes,
		FloorBytes:     spec.FloorBytes,
		Priority:       ParsePriorityTier(spec.Priority),
		Protected:      spec.Protected,
		YieldWhenIdle:  spec.YieldWhenIdle,
		IdleUnloadTTL:  time.Duration(spec.IdleUnloadTTLSeconds) * time.Second,
		IdleGrace:      time.Duration(spec.IdleGraceSeconds) * time.Second,
		Profile:        spec.Profile,
		TTL:            ttl,
	}
}

func openAdmitStore(ctx context.Context, opts AdmitOptions) (AdmitStore, error) {
	if opts.OpenStore != nil {
		return opts.OpenStore(ctx)
	}
	cfg := Config{}
	if opts.Clock != nil {
		cfg.Clock = clockAdapter(opts.Clock)
	}
	return NewSQLiteStore(ctx, cfg)
}

func admitNow(opts AdmitOptions) time.Time {
	if opts.Clock != nil {
		return opts.Clock().UTC()
	}
	return time.Now().UTC()
}

type clockAdapter func() time.Time

func (c clockAdapter) Now() time.Time { return c() }
