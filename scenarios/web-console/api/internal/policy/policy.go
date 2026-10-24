// Package policy defines session expiration policy types and evaluation.
//
// DOC: docs/concepts/GLOSSARY.md#core-terms
// [REQ:P1-001a] Expiration Policy Engine
package policy

import (
	"fmt"
	"time"
)

// Mode defines the session expiration behavior.
type Mode string

const (
	Never  Mode = "never"
	Preset Mode = "preset"
	Custom Mode = "custom"
)

// presetDurations maps preset keys to durations.
// CROSS-LANGUAGE COUPLING: Must match POLICY_OPTIONS in ui/src/consts/policy-options.ts.
var presetDurations = map[string]time.Duration{
	"1h":  time.Hour,
	"8h":  8 * time.Hour,
	"24h": 24 * time.Hour,
}

const (
	customDurationMin = time.Minute
	customDurationMax = 7 * 24 * time.Hour
)

// Policy controls when a session expires.
type Policy struct {
	Mode     Mode   `json:"mode"`
	Duration string `json:"duration,omitempty"`
}

// Default returns the default never-expire policy.
func Default() Policy {
	return Policy{Mode: Never}
}

// Validate checks that a policy is well-formed.
func Validate(p Policy) error {
	switch p.Mode {
	case Never:
		return nil
	case Preset:
		if _, ok := presetDurations[p.Duration]; !ok {
			return fmt.Errorf("invalid preset duration %q; valid: 1h, 8h, 24h", p.Duration)
		}
		return nil
	case Custom:
		if p.Duration == "" {
			return fmt.Errorf("custom policy requires a duration")
		}
		d, err := time.ParseDuration(p.Duration)
		if err != nil {
			return fmt.Errorf("invalid custom duration %q: %w", p.Duration, err)
		}
		if d < customDurationMin {
			return fmt.Errorf("custom duration must be at least %s", customDurationMin)
		}
		if d > customDurationMax {
			return fmt.Errorf("custom duration must be at most %s", customDurationMax)
		}
		return nil
	default:
		return fmt.Errorf("invalid policy mode %q; valid: never, preset, custom", p.Mode)
	}
}

// ResolveTTL returns the duration for a policy. Returns 0 to mean "never expire".
func ResolveTTL(p Policy) time.Duration {
	switch p.Mode {
	case Preset:
		if d, ok := presetDurations[p.Duration]; ok {
			return d
		}
		return 0
	case Custom:
		d, err := time.ParseDuration(p.Duration)
		if err != nil {
			return 0
		}
		return d
	default:
		return 0
	}
}

// IsExpired checks if a session has expired given its creation time and policy.
// [REQ:P1-001a] Expiration Policy Engine — <10ms evaluation
func IsExpired(createdAt time.Time, p Policy) bool {
	ttl := ResolveTTL(p)
	if ttl == 0 {
		return false
	}
	return time.Since(createdAt) > ttl
}
