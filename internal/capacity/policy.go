package capacity

import (
	"strings"
	"time"
)

const (
	policyBatch       = "batch"
	policyInteractive = "interactive"
)

const (
	policyService           = "service"
	policyKeyIdleYieldFloor = "idle_yield_floor"
)

const (
	bytesPerKiB = 1024
)

// Enforce modes (plan §8.5, §7 Phase 3). The lifecycle admission hook keys off
// these. Default is advisory.
const (
	// EnforceOff makes the admission hook a complete no-op: no claim is
	// recorded and the start path is byte-identical to legacy behavior. This is
	// the parity baseline (Phase 3 DoD).
	EnforceOff = "off"
	// EnforceAdvisory records a claim, runs Decide, logs/warns on the verdict,
	// and ALWAYS lets the caller proceed (caller chooses its own fallback). This
	// is the V1 default.
	EnforceAdvisory = "advisory"
	// EnforceOn enables reclaim actuation. Admission callers own whether a
	// non-grant verdict blocks execution. Gated; opt-in.
	EnforceOn = "on"
)

// EnvEnforce is the environment override for the enforce mode. When set to a
// recognized value it takes precedence over the stored policy.
const EnvEnforce = "VROOLI_CAPACITY_ENFORCE"

// Policy holds the tunable levers (plan §2 control-surface-tunable-levers-design).
// Every field is data-declared and editable via `vrooli capacity policy set`.
// No silent caps: each threshold is surfaced and tunable.
type Policy struct {
	// TrackingThreshold is the minimum observed VRAM (bytes) for a consumer to
	// be considered by reconciliation. Below this we don't track it.
	TrackingThreshold int64 `json:"tracking_threshold"`
	// IdleGrace is the dwell time a claim must remain idle (after the work-owner
	// reports idle) before it becomes reclaim-eligible.
	IdleGrace time.Duration `json:"idle_grace"`
	// DefaultHeartbeatTTL bounds claim liveness without a heartbeat.
	DefaultHeartbeatTTL time.Duration `json:"default_heartbeat_ttl"`
	// ReconcileWarnThreshold is how many bytes over its granted amount an
	// observed consumer may drift before reconciliation flags OVER_CLAIM.
	ReconcileWarnThreshold int64 `json:"reconcile_warn_threshold"`
	// Enforce is the admission-hook mode (off|advisory|on).
	Enforce string `json:"enforce"`
	// PreemptEnabled gates the last escalation rung (stop). Degradation is
	// always tried first; preempt only when this is true.
	PreemptEnabled bool `json:"preempt_enabled"`
	// AutoStopAllowlist names owners (csv) that reconciliation may auto-stop.
	// Empty (the default) means auto-stop is OFF for everyone — reconcile warns
	// only.
	AutoStopAllowlist []string `json:"auto_stop_allowlist"`
	// SweepInterval is the cadence the opportunistic resident-claim sweep is
	// debounced to (§8.6). The maintenance pass drives Sweep on lifecycle
	// activity; reads (admission/list/reconcile) re-sweep at most this often.
	SweepInterval time.Duration `json:"sweep_interval"`
	// DegradeDebounce is the minimum dwell between two degrade actuations of the
	// same target (§8.8 anti-thrash).
	DegradeDebounce time.Duration `json:"degrade_debounce"`
	// UpshiftHeadroom is the free VRAM (bytes) required before the actuator may
	// upshift a degraded idle claim back toward preferred (§8.8 hysteresis).
	UpshiftHeadroom int64 `json:"upshift_headroom"`
	// IdleYieldFloor is the LOWEST requester priority tier (rank) that may reclaim
	// an idle, yield_when_idle-opted claim (§8.3 idle-yield). When such a claim has
	// dwelt idle beyond idle_grace, a requester whose priority is at or above this
	// floor may reclaim it even at equal priority — relaxing the strict
	// lower-priority rule, but only for claims that explicitly opted in. Default
	// batch (the lowest tier), so an idle yield-opted claim yields to any GPU work.
	IdleYieldFloor int `json:"idle_yield_floor"`
	// TerminalRetention is how long a terminal (released/expired/preempted) claim
	// survives before terminal-claim GC prunes it (ledger hygiene). Active claims
	// are never pruned.
	TerminalRetention time.Duration `json:"terminal_retention"`
	// ObservedPeakHalflife is the decay half-life of the per-claim observed
	// high-water mark (§Phase 2). A larger value remembers peaks longer.
	ObservedPeakHalflife time.Duration `json:"observed_peak_halflife"`
	// DefaultIdleUnloadTTL is the fallback autonomous idle-unload dwell (§Phase 3)
	// applied to claims that do not declare their own idle_unload_ttl. 0 = off (the
	// safe default — autonomous unload is opt-in per resource).
	DefaultIdleUnloadTTL time.Duration `json:"default_idle_unload_ttl"`
	// RecommendHeadroomPct is the safety margin (percent) added above a claim's
	// observed peak when right-sizing recommends a smaller reservation (§Phase 4).
	// A recommendation is NEVER below observed_peak * (1 + pct/100).
	RecommendHeadroomPct int `json:"recommend_headroom"`
	// SwapPressureThresholdPct is the conservative occupancy fallback used for
	// RAM admission when fresh memory PSI is unavailable. Measured PSI uses the
	// canonical setpoint instead. 0 disables only this fallback.
	SwapPressureThresholdPct int `json:"swap_pressure_threshold"`
	// AccelReprobe decides what happens when host accelerator readiness
	// transitions from not-ready to ready. Resources that started before the
	// device appeared are, by then, running on the CPU with every status
	// surface green; without this they stay there until someone notices.
	//   off     — do nothing.
	//   report  — log the drifted resources and emit an event. The default:
	//             restarting is a lifecycle action, and turning one on by
	//             default would surprise an operator mid-request.
	//   restart — restart each drifted resource that is not actively working.
	AccelReprobe string `json:"accel_reprobe"`
}

// Accelerator re-probe modes.
const (
	AccelReprobeOff     = "off"
	AccelReprobeReport  = "report"
	AccelReprobeRestart = "restart"
)

// DefaultPolicy returns the conservative V1 defaults: advisory enforcement,
// preempt disabled, auto-stop off.
func DefaultPolicy() Policy {
	return Policy{
		TrackingThreshold:      256 * 1024 * bytesPerKiB, // 256 MiB
		IdleGrace:              DefaultIdleGrace,
		DefaultHeartbeatTTL:    DefaultHeartbeatTTL,
		ReconcileWarnThreshold: 512 * 1024 * bytesPerKiB, // 512 MiB drift
		Enforce:                EnforceAdvisory,
		PreemptEnabled:         false,
		AutoStopAllowlist:      nil,
		SweepInterval:          DefaultSweepInterval,
		DegradeDebounce:        DefaultDegradeDebounce,
		UpshiftHeadroom:        DefaultUpshiftHeadroom,
		IdleYieldFloor:         PriorityBatch,
		TerminalRetention:      DefaultTerminalRetention,
		ObservedPeakHalflife:   DefaultObservedPeakHalflife,
		RecommendHeadroomPct:   DefaultRecommendHeadroomPct,

		SwapPressureThresholdPct: DefaultSwapPressureThreshold,
		AccelReprobe:             AccelReprobeReport,
	}
}

// EffectiveEnforce resolves the enforce mode honoring the env override.
func (p Policy) EffectiveEnforce(envValue string) string {
	v := strings.TrimSpace(envValue)
	switch v {
	case EnforceOff, EnforceAdvisory, EnforceOn:
		return v
	default:
		if strings.TrimSpace(p.Enforce) == "" {
			return EnforceAdvisory
		}
		return p.Enforce
	}
}

// IsAutoStopAllowed reports whether reconciliation may auto-stop the named owner.
func (p Policy) IsAutoStopAllowed(ownerID string) bool {
	for _, o := range p.AutoStopAllowlist {
		if o == ownerID {
			return true
		}
	}
	return false
}
