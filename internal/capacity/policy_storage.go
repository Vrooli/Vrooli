package capacity

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// This file owns the persisted policy key/value codec and its local helpers.

// AccelReprobeModes is the closed set of accel_reprobe values.
var AccelReprobeModes = []string{AccelReprobeOff, AccelReprobeReport, AccelReprobeRestart}

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
	policyKeyIdleYieldFloor,
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
	case policyKeyIdleYieldFloor:
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
//
//nolint:gocyclo // claim policy resolves an explicit resource-kind and state decision matrix.
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
	case policyKeyIdleYieldFloor:
		v := strings.TrimSpace(value)
		switch v {
		case policyInteractive, policyService, policyBatch:
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

// ranks render as the numeric value).
func PriorityTierName(priority int) string {
	switch {
	case priority >= PriorityInteractive:
		return policyInteractive
	case priority >= PriorityService:
		return policyService
	default:
		return policyBatch
	}
}

// ParsePriorityTier maps a tier name (or numeric string) to its integer rank.
// An empty or unknown name defaults to batch (the lowest, safest tier).
func ParsePriorityTier(name string) int {
	switch name {
	case policyInteractive:
		return PriorityInteractive
	case policyService:
		return PriorityService
	case policyBatch, "":
		return PriorityBatch
	default:
		return PriorityBatch
	}
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

// SetPolicyKey validates and persists a single policy key, returning the
// resulting effective policy.
func (s *SQLiteStore) SetPolicyKey(ctx context.Context, key, value string) (Policy, error) {
	current, err := s.GetPolicy(ctx)
	if err != nil {
		return Policy{}, err
	}
	next, err := current.withKey(key, value)
	if err != nil {
		return Policy{}, err
	}
	// Re-derive the canonical stored value from the validated policy so a value
	// like "60s" round-trips consistently.
	stored, err := next.Get(key)
	if err != nil {
		return Policy{}, err
	}
	now := s.now()
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `
INSERT INTO capacity_policy (key, value, updated_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			key, stored, formatTime(now))
		if execErr != nil {
			return fmt.Errorf("set capacity policy %s: %w", key, execErr)
		}
		return nil
	})
	if err != nil {
		return Policy{}, err
	}
	return next, nil
}

// RecordSweepAt stamps the time of the most recent sweep so the opportunistic
// callers can debounce to policy.SweepInterval.
func (s *SQLiteStore) RecordSweepAt(ctx context.Context, at time.Time) error {
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `
INSERT INTO capacity_policy (key, value, updated_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			sweepCursorKey, formatTime(at.UTC()), formatTime(s.now()))
		if execErr != nil {
			return fmt.Errorf("record capacity sweep cursor: %w", execErr)
		}
		return nil
	})
}
