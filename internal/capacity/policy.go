package capacity

import (
	"fmt"
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
}

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
