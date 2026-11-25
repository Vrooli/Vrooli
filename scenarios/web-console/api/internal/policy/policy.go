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
	// Idle archives a session once its agent has done no work for Duration
	// and nothing is still running. It is never measured from CreatedAt; the
	// sweeper asks the session's harness evidence (see api/session_idle.go).
	Idle Mode = "idle"
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
	// An idle window under 15 minutes would archive agents between turns.
	idleDurationMin = 15 * time.Minute
	idleDurationMax = 7 * 24 * time.Hour
)

// Policy controls when a session expires.
type Policy struct {
	Mode     Mode   `json:"mode"`
	Duration string `json:"duration,omitempty"`
}

// programmaticIdleDuration is how long a machine-started session may go
// without agent work before it archives. It is intentionally longer than one
// turn and shorter than a working day: an executor nobody is watching must not
// outlive its run, while a research run that keeps writing its transcript is
// never touched.
const programmaticIdleDuration = 6 * time.Hour

// Default returns the default never-expire policy, which is what an
// interactive session keeps: a tab the operator opened stays until they close
// it, however long they leave it alone.
func Default() Policy {
	return Policy{Mode: Never}
}

// ProgrammaticDefault returns the policy a machine-started session starts
// under when its creator does not choose one. A programmatic session has no
// operator watching the tab, so it is archived once its agent has done no work
// for the idle window instead of living forever.
//
// A policy is persisted with the session at creation, so changing this value
// never rewrites a session that already started: every session keeps the
// policy it was created with.
func ProgrammaticDefault() Policy {
	return Policy{Mode: Idle, Duration: programmaticIdleDuration.String()}
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
	case Idle:
		if p.Duration == "" {
			return fmt.Errorf("idle policy requires a duration")
		}
		d, err := time.ParseDuration(p.Duration)
		if err != nil {
			return fmt.Errorf("invalid idle duration %q: %w", p.Duration, err)
		}
		if d < idleDurationMin {
			return fmt.Errorf("idle duration must be at least %s", idleDurationMin)
		}
		if d > idleDurationMax {
			return fmt.Errorf("idle duration must be at most %s", idleDurationMax)
		}
		return nil
	default:
		return fmt.Errorf("invalid policy mode %q; valid: never, preset, custom, idle", p.Mode)
	}
}

// IdleAfter returns how long a session under the idle policy must go without
// agent work before it is archived. ok is false for every other mode.
func IdleAfter(p Policy) (time.Duration, bool) {
	if p.Mode != Idle {
		return 0, false
	}
	d, err := time.ParseDuration(p.Duration)
	if err != nil || d <= 0 {
		return 0, false
	}
	return d, true
}

// ResolveTTL returns the age-based lifetime of a policy. Returns 0 to mean
// "never expire by age" (never and idle).
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
