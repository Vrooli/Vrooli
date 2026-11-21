package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/tuning"
)

// ResourceClaimSpec is the optional `acceleration.claim` block declared in
// resource.json so the lifecycle can claim on its behalf at admission. Absent
// block => the resource is not (yet) a capacity adopter and admission is a
// no-op (the dormant-by-default property that keeps Phase 3 parity-safe).
type ResourceClaimSpec struct {
	ResourceKind   string `json:"resource_kind"`
	PreferredBytes int64  `json:"default_preferred_bytes"`
	FloorBytes     int64  `json:"floor_bytes"`
	Priority       string `json:"default_priority"` // tier seed, overridable by operator state
	Confidence     string `json:"confidence"`
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

// LoadResourceClaimSpec reads the optional acceleration.claim block from a resource's
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
	LockAdmission(context.Context) (func(), error)
	Close() error
}

// AdmitOptions configures a single resource admission. Seams are injectable for
// tests; nil values resolve production defaults.
type AdmitOptions struct {
	// NoReclaim makes this startup a required admission from already-free capacity.
	NoReclaim    bool
	Root         string
	ResourceName string
	// EnforceEnv overrides the VROOLI_CAPACITY_ENFORCE read (tests).
	EnforceEnv string
	Source     CapacitySource
	// MemoryPressure injects fresh PSI evidence; nil uses the production reader only with the default source.
	MemoryPressure func() hostpressure.Reading
	// Attributor resolves observed GPU PIDs to owners for the presence-aware
	// admission sweep. nil resolves the production docker attributor.
	Attributor Attributor
	// Exec runs an adopter's degrade verb during enforce-mode actuation. nil
	// resolves the production CmdExecutor (shells the owner's resource CLI).
	Exec      ApplyExecutor
	OpenStore func(ctx context.Context) (AdmitStore, error)
	Clock     func() time.Time
	// Log receives advisory admission/lease errors. nil discards diagnostics.
	Log io.Writer
}

// AdmissionResult reports what admission did. Skipped=true means no claim was
// recorded (no profile, or enforcement disabled).
type AdmissionResult struct {
	Skipped bool          `json:"skipped"`
	Reason  string        `json:"reason,omitempty"`
	Enforce string        `json:"enforce,omitempty"`
	Verdict Verdict       `json:"verdict,omitempty"`
	ClaimID string        `json:"claim_id,omitempty"`
	ttl     time.Duration // Same manifest snapshot as the admitted request.
}

// AdmitResource derives the lifecycle request from the resource manifest.
// Ordinary startup remains advisory; explicit NoReclaim requires a fresh grant
// before changing the reservation. Missing declarations and disabled policy are
// reported as skipped, never certified as successful admission.
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
	if opts.NoReclaim {
		release, err := LockAdmission(ctx, store)
		if err != nil {
			return AdmissionResult{}, err
		}
		defer release()
	}

	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return AdmissionResult{}, err
	}
	effective := policy.EffectiveEnforce(enforceEnv)
	if effective == EnforceOff {
		return AdmissionResult{Skipped: true, Reason: "enforcement disabled (policy off)", Enforce: EnforceOff}, nil
	}

	req := spec.toRequest(opts.ResourceName)
	req.NoReclaim = opts.NoReclaim
	existing, err := store.ListClaims(ctx, ClaimFilter{
		OwnerKind: OwnerKindResource, OwnerID: opts.ResourceName,
		ResourceKind: req.ResourceKind, Statuses: ActiveClaimStatuses(),
	})
	if err != nil {
		return AdmissionResult{}, err
	}
	var reusable *CapacityClaim
	var replaced []string
	for _, claim := range existing {
		if reusable == nil && claimMatchesResourceRequest(claim, req) && claim.Status != StatusReserved && (!opts.NoReclaim || claim.AmountBytes == req.PreferredBytes) {
			c := claim
			reusable = &c
		} else {
			replaced = append(replaced, claim.ClaimID)
		}
	}
	var verdict Verdict
	if reusable != nil && !opts.NoReclaim {
		// Residency renewal is not a new placement. Do not reclaim someone else's
		// capacity merely to renew accounting for an already admitted resident.
		verdict = reusedClaimVerdict(*reusable)
	} else {
		ignored := append([]string(nil), replaced...)
		decisionReq := req
		if reusable != nil {
			ignored = append(ignored, reusable.ClaimID)
		}
		if opts.NoReclaim {
			// Startup executes the selected manifest, not an unimplemented lower-rung
			// recommendation. Require its full budget; explicit claim callers can still
			// request and execute smaller rungs through their own owner path.
			decisionReq.FloorBytes, decisionReq.Profile = req.PreferredBytes, nil
		}
		verdict, err = EvaluateAdmission(ctx, store, decisionReq, AdmissionOptions{
			Source: opts.Source, MemoryPressure: opts.MemoryPressure, Attributor: opts.Attributor,
			Exec: opts.Exec, Policy: policy, Enforce: effective, Clock: opts.Clock, ExcludeClaims: ignored,
		})
		if err != nil {
			return AdmissionResult{}, err
		}
		if opts.NoReclaim && !verdict.Granted() {
			return AdmissionResult{Enforce: effective, Verdict: verdict}, fmt.Errorf("no-reclaim admission refused: %s", verdict.Reason)
		}
		if reusable != nil {
			warnings := verdict.Warnings
			verdict = reusedClaimVerdict(*reusable)
			verdict.Warnings = warnings
		}
	}

	var recorded CapacityClaim
	if reusable != nil {
		recorded, err = store.HeartbeatClaim(ctx, reusable.ClaimID, reusable.Generation, normalizeAdmitTTL(req.TTL))
	} else {
		recorded, err = store.CreateClaim(ctx, req.Claim(verdict), req.TTL)
	}
	if err != nil {
		return AdmissionResult{}, err
	}
	// Keep the old accounting until its successor is recorded or renewed. A
	// failed decision/write must not erase an existing owner's reservation.
	for _, id := range replaced {
		if _, err := store.ReleaseClaim(ctx, id); err != nil {
			return AdmissionResult{Enforce: effective, Verdict: verdict, ClaimID: recorded.ClaimID}, fmt.Errorf("claim %s recorded; release prior capacity claim %s: %w", recorded.ClaimID, id, err)
		}
	}
	reason := ""
	if reusable != nil {
		reason = "reused existing active claim"
	}
	return AdmissionResult{Enforce: effective, Verdict: verdict, ClaimID: recorded.ClaimID, Reason: reason, ttl: normalizeAdmitTTL(req.TTL)}, nil
}

func reusedClaimVerdict(claim CapacityClaim) Verdict {
	v := Verdict{Kind: VerdictGrant, GrantedBytes: claim.AmountBytes, Step: "preferred"}
	if claim.AmountBytes < claim.PreferredBytes {
		v.Kind, v.Step = VerdictDegrade, "floor"
		v.Reason = "reused existing degraded reservation"
	}
	if claim.DegradeProfile != nil {
		for _, step := range claim.DegradeProfile.Steps {
			if step.AmountBytes == claim.AmountBytes {
				v.Step = step.Label
				break
			}
		}
	}
	return v
}

// AdmissionOptions binds the shared sensing/decision pipeline. ExcludeClaims is
// only the exact reservations this owner is replacing, never other owners.
type AdmissionOptions struct {
	Source         CapacitySource
	MemoryPressure func() hostpressure.Reading
	Attributor     Attributor
	Exec           ApplyExecutor
	Policy         Policy
	Enforce        string
	Clock          func() time.Time
	ExcludeClaims  []string
}

// EvaluateAdmission is the shared native CLI/resource decision boundary. It
// senses once, preserves residents through Sweep, applies pressure policy and
// realizes only the reclaim cohort selected by Decide. Recording is separate
// from a grant; unavailable sensing is a denial, including in advisory mode.
func EvaluateAdmission(ctx context.Context, store ClaimRepository, req CapacityRequest, opts AdmissionOptions) (Verdict, error) {
	source := opts.Source
	if source == nil {
		source = HostInventorySource{}
	}
	snapshot, err := source.Snapshot(ctx)
	if err != nil {
		return Verdict{Kind: VerdictDeny, Reason: "capacity sensing unavailable: " + err.Error()}, nil
	}
	now := clock.Real{}.Now()
	if opts.Clock != nil {
		now = opts.Clock().UTC()
	}
	attr := opts.Attributor
	if attr == nil {
		attr = NewDockerAttributor()
	}
	swept, err := Sweep(ctx, store, snapshot, attr, opts.Policy, now)
	if err != nil {
		return Verdict{}, err
	}
	ledger, err := store.ListClaims(ctx, ClaimFilter{ResourceKind: req.ResourceKind, Statuses: ActiveClaimStatuses()})
	if err != nil {
		return Verdict{}, err
	}
	ledger = slices.DeleteFunc(ledger, func(c CapacityClaim) bool {
		return slices.Contains(opts.ExcludeClaims, c.ClaimID) && c.OwnerKind == req.OwnerKind && c.OwnerID == req.OwnerID
	})
	policy := opts.Policy
	pressure := opts.MemoryPressure
	if pressure == nil && opts.Source == nil {
		pressure = ReadHostMemoryPressure
	}
	if req.ResourceKind == ResourceKindRAM {
		policy, err = MemoryPressurePolicy(policy, pressure)
		if err != nil {
			return Verdict{Kind: VerdictDeny, Reason: err.Error()}, nil
		}
	}
	verdict := Decide(req, snapshot, ledger, policy, now)
	if len(swept.Deferred) > 0 {
		verdict.Warnings = append(verdict.Warnings, fmt.Sprintf("preserved %d resident claim(s) whose GPU presence is unverified", len(swept.Deferred)))
	}
	if len(snapshot.GPUs) > 0 && (snapshot.ProbeStatuses["nvidia_gpu"] != "ok" || snapshot.ProbeStatuses["nvidia_gpu_processes"] != "ok") {
		verdict.Warnings = append(verdict.Warnings, "GPU process absence is unverified; resident claims were preserved")
	}
	return AdmitReclaim(ctx, store, req.Priority, verdict, ledger, opts.Exec, policy, opts.Enforce, now), nil
}

func claimMatchesResourceRequest(claim CapacityClaim, req CapacityRequest) bool {
	return claim.ResourceKind == req.ResourceKind &&
		(req.ResourceKind != ResourceKindVRAM || sameGPU(claim.GPUIndex, req.GPUIndex)) &&
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
		GPUIndex:       nil,
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

type clockAdapter func() time.Time

func (c clockAdapter) Now() time.Time { return c() }

// WithResourceAdmission keeps the exact manifest-derived claim alive throughout
// a bounded start/restart operation, including CPU-only verification/loading.
// Ordinary starts remain advisory. Explicit NoReclaim refuses startup when
// admission or the initial startup lease cannot be established.
// Completion hands residency to the resource companion with one final renewal.
// Failure/cancellation does not mean the supervised process stopped, so it never
// releases the claim. Explicit verified stop owns release.
func WithResourceAdmission(ctx context.Context, opts AdmitOptions, run func(context.Context) error) error {
	return withResourceAdmissionTicks(ctx, opts, nil, run)
}

func withResourceAdmissionTicks(ctx context.Context, opts AdmitOptions, ticks <-chan time.Time, run func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	runOwner := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return run(ctx)
	}
	result, err := AdmitResource(ctx, opts)
	if err != nil {
		admissionLog(opts, "admission: %v", err)
		if opts.NoReclaim {
			return err
		}
		return runOwner()
	}
	for _, warning := range result.Verdict.Warnings {
		admissionLog(opts, "admission: %s", warning)
	}
	if result.Skipped || result.ClaimID == "" {
		if opts.NoReclaim {
			return fmt.Errorf("no-reclaim admission unavailable: %s", result.Reason)
		}
		return runOwner()
	}
	admissionLog(opts, "admission claim=%s verdict=%s granted_bytes=%d no_reclaim=%t", result.ClaimID, result.Verdict.Kind, result.Verdict.GrantedBytes, opts.NoReclaim)
	store, err := openAdmitStore(ctx, opts)
	if err != nil {
		admissionLog(opts, "open startup lease: %v", err)
		if opts.NoReclaim {
			return err
		}
		return runOwner()
	}
	defer store.Close()
	ttl := result.ttl

	interval := ttl / 3
	if interval <= 0 {
		interval = time.Nanosecond
	}
	if ticks == nil {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		ticks = ticker.C
	}
	renew := func(parent context.Context) {
		callCtx, cancel := context.WithTimeout(parent, interval)
		defer cancel()
		current, getErr := store.GetClaim(callCtx, result.ClaimID)
		if getErr == nil && (current.OwnerKind != OwnerKindResource || current.OwnerID != opts.ResourceName) {
			getErr = fmt.Errorf("claim identity no longer matches resource")
		}
		if getErr == nil {
			_, getErr = store.HeartbeatClaim(callCtx, current.ClaimID, current.Generation, ttl)
		}
		if getErr != nil && parent.Err() == nil {
			admissionLog(opts, "renew startup claim %s: %v", result.ClaimID, getErr)
		}
	}
	leaseCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-ticks:
				renew(leaseCtx)
			}
		}
	}()
	// Join before closing the store; no stranded renewal goroutine after return.
	var once sync.Once
	stop := func() { once.Do(func() { cancel(); <-done }) }
	defer stop()
	runErr := runOwner()
	stop()
	if ctx.Err() == nil {
		renew(ctx)
	}
	return runErr
}

// ReleaseResourceAdmissionAfterStop captures this owner's active claims before
// a stop performed under the caller's lifecycle lock. Only a successful verified
// terminal stop releases those IDs. Restart and uncertain stop never call this.
func ReleaseResourceAdmissionAfterStop(ctx context.Context, opts AdmitOptions, stop func() error) error {
	_, declared, err := LoadResourceClaimSpec(opts.Root, opts.ResourceName)
	if err != nil {
		admissionLog(opts, "read stop claims: %v", err)
	}
	if !declared || err != nil {
		return stop()
	}
	store, err := openAdmitStore(ctx, opts)
	if err != nil {
		admissionLog(opts, "open stop claims: %v", err)
		return stop()
	}
	defer store.Close()
	claims, err := store.ListClaims(ctx, ClaimFilter{OwnerKind: OwnerKindResource, OwnerID: opts.ResourceName, Statuses: ActiveClaimStatuses()})
	if err != nil {
		admissionLog(opts, "capture stop claims: %v", err)
	}
	if stopErr := stop(); stopErr != nil {
		return stopErr
	}
	// A successful stop may finish escalation after caller cancellation. Ledger
	// cleanup is bounded and retains normal stop authority, not a new model task.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tuning.CredentialServiceTimeout())
	defer cancel()
	for _, claim := range claims {
		current, getErr := store.GetClaim(cleanupCtx, claim.ClaimID)
		if getErr == nil && (current.OwnerKind != OwnerKindResource || current.OwnerID != opts.ResourceName) {
			getErr = fmt.Errorf("claim identity no longer matches resource")
		}
		if getErr == nil {
			_, getErr = store.ReleaseClaim(cleanupCtx, claim.ClaimID)
		}
		if getErr != nil {
			admissionLog(opts, "release stopped claim %s: %v", claim.ClaimID, getErr)
		}
	}
	return nil
}

func admissionLog(opts AdmitOptions, format string, args ...any) {
	if opts.Log != nil {
		fmt.Fprintf(opts.Log, "capacity (%s): "+format+"\n", append([]any{opts.ResourceName}, args...)...)
	}
}
