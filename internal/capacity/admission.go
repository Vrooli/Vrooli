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
	"time"

	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostpressure"
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

	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return AdmissionResult{}, err
	}
	effective := policy.EffectiveEnforce(enforceEnv)
	if effective == EnforceOff {
		return AdmissionResult{Skipped: true, Reason: "enforcement disabled (policy off)", Enforce: EnforceOff}, nil
	}
	admission := AdmissionOptions{
		Source: opts.Source, MemoryPressure: opts.MemoryPressure, Attributor: opts.Attributor,
		Exec: opts.Exec, Policy: policy, Enforce: effective, Clock: opts.Clock,
	}
	// A strict no-reclaim startup serializes its decision and write against
	// every other admission, but it senses the host BEFORE taking the lock: it
	// used to hold the fleet-wide lock across hardware probing, and a lifecycle
	// restart held it for over 50 s.
	var sensing AdmissionSensing
	if opts.NoReclaim {
		var release func()
		sensing, release, err = SenseThenLock(ctx, store, admission)
		if err != nil {
			return AdmissionResult{}, err
		}
		defer release()
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
		admission.ExcludeClaims = ignored
		if opts.NoReclaim {
			verdict, err = DecideAdmission(ctx, store, decisionReq, admission, sensing)
		} else {
			verdict, err = EvaluateAdmission(ctx, store, decisionReq, admission)
		}
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

// AdmissionSensing is the host observation an admission decision rests on:
// one capacity snapshot and the liveness sweep run against it. It is gathered
// WITHOUT the admission lock - sensing shells out to nvidia-smi, docker and
// systemd and takes seconds on a loaded host - and handed to DecideAdmission,
// which takes the lock only around the ledger read, the decision and the write.
type AdmissionSensing struct {
	Snapshot hostinventory.Snapshot
	// Unavailable is non-empty when the host could not be sensed; the decision
	// is then a denial, in advisory mode too.
	Unavailable string
	swept       SweepResult
	sensedAt    time.Time
}

// SensedAt is when the snapshot was taken.
func (a AdmissionSensing) SensedAt() time.Time { return a.sensedAt }

// MaxAdmissionSensingAge bounds how stale sensing may be when the decision is
// made. A caller that queued longer than this for the admission lock senses
// again before it decides (see SenseThenLock).
const MaxAdmissionSensingAge = 10 * time.Second

// SenseAdmission reads the host and sweeps stale claims. It never holds the
// admission lock; every ledger write inside Sweep is its own short transaction.
func SenseAdmission(ctx context.Context, store ClaimRepository, opts AdmissionOptions) (AdmissionSensing, error) {
	source := opts.Source
	if source == nil {
		source = HostInventorySource{}
	}
	snapshot, err := source.Snapshot(ctx)
	if err != nil {
		return AdmissionSensing{Unavailable: err.Error()}, nil
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
		return AdmissionSensing{}, err
	}
	return AdmissionSensing{Snapshot: snapshot, swept: swept, sensedAt: time.Now()}, nil
}

// SenseThenLock senses outside the admission lock, takes the lock, and
// guarantees the sensing it returns is no older than MaxAdmissionSensingAge at
// the moment the lock is held. If the lock queue made it older, the sensing is
// redone and the lock retaken (bounded), so a long queue never turns into a
// decision on a stale host reading and the lock is still never held while
// sensing. The returned release must be called exactly once.
func SenseThenLock(ctx context.Context, store ClaimRepository, opts AdmissionOptions) (AdmissionSensing, func(), error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		sensing, err := SenseAdmission(ctx, store, opts)
		if err != nil {
			return AdmissionSensing{}, nil, err
		}
		release, err := LockAdmission(ctx, store)
		if err != nil {
			return AdmissionSensing{}, nil, err
		}
		if sensing.Unavailable != "" || time.Since(sensing.sensedAt) <= MaxAdmissionSensingAge || attempt == 2 {
			return sensing, release, nil
		}
		release()
		lastErr = fmt.Errorf("admission sensing aged past %s while queued for the admission lock", MaxAdmissionSensingAge)
	}
	return AdmissionSensing{}, nil, lastErr
}

// EvaluateAdmission is the shared native CLI/resource decision boundary. It
// senses once, preserves residents through Sweep, applies pressure policy and
// realizes only the reclaim cohort selected by Decide. Recording is separate
// from a grant; unavailable sensing is a denial, including in advisory mode.
// Callers that hold the admission lock use SenseThenLock + DecideAdmission so
// the lock never spans sensing.
func EvaluateAdmission(ctx context.Context, store ClaimRepository, req CapacityRequest, opts AdmissionOptions) (Verdict, error) {
	sensing, err := SenseAdmission(ctx, store, opts)
	if err != nil {
		return Verdict{}, err
	}
	return DecideAdmission(ctx, store, req, opts, sensing)
}

// DecideAdmission is the decision half of EvaluateAdmission: it reads the
// ledger, applies pressure policy and Decide to sensing taken earlier, and
// realizes the selected reclaim cohort.
func DecideAdmission(ctx context.Context, store ClaimRepository, req CapacityRequest, opts AdmissionOptions, sensing AdmissionSensing) (Verdict, error) {
	if sensing.Unavailable != "" {
		return Verdict{Kind: VerdictDeny, Reason: "capacity sensing unavailable: " + sensing.Unavailable}, nil
	}
	snapshot, swept := sensing.Snapshot, sensing.swept
	now := clock.Real{}.Now()
	if opts.Clock != nil {
		now = opts.Clock().UTC()
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
