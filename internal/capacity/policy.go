package capacity

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
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
	// EnforceOn honors the verdict (block/degrade/queue). Gated; opt-in.
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
	// SwapPressureThresholdPct is the percentage of swap in use above which a
	// RAM request is denied rather than granted. Swap usage is a lagging signal
	// of memory exhaustion that AvailableBytes misses. 0 disables the check.
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

// AccelReprobeModes is the closed set of accel_reprobe values.
var AccelReprobeModes = []string{AccelReprobeOff, AccelReprobeReport, AccelReprobeRestart}

// DefaultPolicy returns the conservative V1 defaults: advisory enforcement,
// preempt disabled, auto-stop off.
func DefaultPolicy() Policy {
	return Policy{
		TrackingThreshold:      256 * 1024 * 1024, // 256 MiB
		IdleGrace:              DefaultIdleGrace,
		DefaultHeartbeatTTL:    DefaultHeartbeatTTL,
		ReconcileWarnThreshold: 512 * 1024 * 1024, // 512 MiB drift
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

// PolicyKeys are the recognized tunable keys, in stable order.
var PolicyKeys = []string{
	"tracking_threshold",
	"idle_grace",
	"default_heartbeat_ttl",
	"reconcile_warn_threshold",
	"enforce",
	"preempt_enabled",
	"auto_stop_allowlist",
	"sweep_interval",
	"degrade_debounce",
	"upshift_headroom",
	"idle_yield_floor",
	"terminal_retention",
	"observed_peak_halflife",
	"default_idle_unload_ttl",
	"recommend_headroom",
	"swap_pressure_threshold",
	"accel_reprobe",
}

// Get returns the string value of a policy key (for `policy get`).
func (p Policy) Get(key string) (string, error) {
	switch key {
	case "tracking_threshold":
		return strconv.FormatInt(p.TrackingThreshold, 10), nil
	case "idle_grace":
		return p.IdleGrace.String(), nil
	case "default_heartbeat_ttl":
		return p.DefaultHeartbeatTTL.String(), nil
	case "reconcile_warn_threshold":
		return strconv.FormatInt(p.ReconcileWarnThreshold, 10), nil
	case "enforce":
		return p.Enforce, nil
	case "preempt_enabled":
		return strconv.FormatBool(p.PreemptEnabled), nil
	case "auto_stop_allowlist":
		return strings.Join(p.AutoStopAllowlist, ","), nil
	case "sweep_interval":
		return p.SweepInterval.String(), nil
	case "degrade_debounce":
		return p.DegradeDebounce.String(), nil
	case "upshift_headroom":
		return strconv.FormatInt(p.UpshiftHeadroom, 10), nil
	case "idle_yield_floor":
		return PriorityTierName(p.IdleYieldFloor), nil
	case "terminal_retention":
		return p.TerminalRetention.String(), nil
	case "observed_peak_halflife":
		return p.ObservedPeakHalflife.String(), nil
	case "default_idle_unload_ttl":
		return p.DefaultIdleUnloadTTL.String(), nil
	case "recommend_headroom":
		return strconv.Itoa(p.RecommendHeadroomPct), nil
	case "swap_pressure_threshold":
		return strconv.Itoa(p.SwapPressureThresholdPct), nil
	case "accel_reprobe":
		return p.EffectiveAccelReprobe(), nil
	default:
		return "", fmt.Errorf("%w: unknown policy key %q", ErrInvalidClaim, key)
	}
}

// withKey returns a copy of the policy with one key set from its string value,
// validating the value. Unknown keys and malformed values are rejected.
func (p Policy) withKey(key, value string) (Policy, error) {
	out := p
	switch key {
	case "tracking_threshold":
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || n < 0 {
			return p, fmt.Errorf("%w: tracking_threshold must be a non-negative integer", ErrInvalidClaim)
		}
		out.TrackingThreshold = n
	case "idle_grace":
		d, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || d < 0 {
			return p, fmt.Errorf("%w: idle_grace must be a non-negative duration", ErrInvalidClaim)
		}
		out.IdleGrace = d
	case "default_heartbeat_ttl":
		d, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || d <= 0 {
			return p, fmt.Errorf("%w: default_heartbeat_ttl must be a positive duration", ErrInvalidClaim)
		}
		out.DefaultHeartbeatTTL = d
	case "reconcile_warn_threshold":
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || n < 0 {
			return p, fmt.Errorf("%w: reconcile_warn_threshold must be a non-negative integer", ErrInvalidClaim)
		}
		out.ReconcileWarnThreshold = n
	case "enforce":
		v := strings.TrimSpace(value)
		if v != EnforceOff && v != EnforceAdvisory && v != EnforceOn {
			return p, fmt.Errorf("%w: enforce must be one of off|advisory|on", ErrInvalidClaim)
		}
		out.Enforce = v
	case "preempt_enabled":
		b, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return p, fmt.Errorf("%w: preempt_enabled must be a boolean", ErrInvalidClaim)
		}
		out.PreemptEnabled = b
	case "auto_stop_allowlist":
		out.AutoStopAllowlist = splitCSV(value)
	case "sweep_interval":
		d, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || d <= 0 {
			return p, fmt.Errorf("%w: sweep_interval must be a positive duration", ErrInvalidClaim)
		}
		out.SweepInterval = d
	case "degrade_debounce":
		d, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || d < 0 {
			return p, fmt.Errorf("%w: degrade_debounce must be a non-negative duration", ErrInvalidClaim)
		}
		out.DegradeDebounce = d
	case "upshift_headroom":
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || n < 0 {
			return p, fmt.Errorf("%w: upshift_headroom must be a non-negative integer", ErrInvalidClaim)
		}
		out.UpshiftHeadroom = n
	case "idle_yield_floor":
		v := strings.TrimSpace(value)
		switch v {
		case "interactive", "service", "batch":
			out.IdleYieldFloor = ParsePriorityTier(v)
		default:
			return p, fmt.Errorf("%w: idle_yield_floor must be one of interactive|service|batch", ErrInvalidClaim)
		}
	case "terminal_retention":
		d, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || d < 0 {
			return p, fmt.Errorf("%w: terminal_retention must be a non-negative duration", ErrInvalidClaim)
		}
		out.TerminalRetention = d
	case "observed_peak_halflife":
		d, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || d <= 0 {
			return p, fmt.Errorf("%w: observed_peak_halflife must be a positive duration", ErrInvalidClaim)
		}
		out.ObservedPeakHalflife = d
	case "default_idle_unload_ttl":
		d, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || d < 0 {
			return p, fmt.Errorf("%w: default_idle_unload_ttl must be a non-negative duration (0 = off)", ErrInvalidClaim)
		}
		out.DefaultIdleUnloadTTL = d
	case "swap_pressure_threshold":
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < 0 || n > 100 {
			return p, fmt.Errorf("%w: swap_pressure_threshold must be an integer percent between 0 and 100", ErrInvalidClaim)
		}
		out.SwapPressureThresholdPct = n
	case "accel_reprobe":
		mode := strings.ToLower(strings.TrimSpace(value))
		if !slices.Contains(AccelReprobeModes, mode) {
			return p, fmt.Errorf("%w: accel_reprobe must be one of %v", ErrInvalidClaim, AccelReprobeModes)
		}
		out.AccelReprobe = mode
	case "recommend_headroom":
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < 0 {
			return p, fmt.Errorf("%w: recommend_headroom must be a non-negative integer percent", ErrInvalidClaim)
		}
		out.RecommendHeadroomPct = n
	default:
		return p, fmt.Errorf("%w: unknown policy key %q", ErrInvalidClaim, key)
	}
	return out, nil
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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// EffectiveAccelReprobe resolves an unset accel_reprobe to the report default,
// so a policy row written before this key existed reads as report rather than
// as off.
func (p Policy) EffectiveAccelReprobe() string {
	mode := strings.ToLower(strings.TrimSpace(p.AccelReprobe))
	if !slices.Contains(AccelReprobeModes, mode) {
		return AccelReprobeReport
	}
	return mode
}
