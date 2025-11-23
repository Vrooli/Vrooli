package engine

import (
	"os"
	"strings"
)

// SelectionConfig controls how the executor chooses an engine implementation.
type SelectionConfig struct {
	DefaultEngine string // Fallback when no override/request provided.
	Override      string // Explicit override (e.g., env/feature flag).
	FeatureFlag   string // Feature flag name that enables new executor path.
	ShadowMode    string // "on" enables dual-run shadow execution.
}

// FromEnv builds SelectionConfig from environment variables. The variables are
// intentionally simple to avoid widening handler contracts:
//
//	ENGINE: global default engine name (e.g., browserless)
//	ENGINE_OVERRIDE: hard override for all executions (e.g., desktop)
//	ENGINE_FEATURE_FLAG: opt-in flag name; blank defaults to "on", set to "off" to disable
//	ENGINE_SHADOW_MODE: "on" to run new executor alongside legacy without impacting status
func FromEnv() SelectionConfig {
	return SelectionConfig{
		DefaultEngine: strings.TrimSpace(os.Getenv("ENGINE")),
		Override:      strings.TrimSpace(os.Getenv("ENGINE_OVERRIDE")),
		FeatureFlag:   strings.TrimSpace(os.Getenv("ENGINE_FEATURE_FLAG")),
		ShadowMode:    strings.TrimSpace(os.Getenv("ENGINE_SHADOW_MODE")),
	}
}

// Resolve chooses the engine name given a per-execution request. Override, when
// present, always wins to keep rollout deterministic.
func (c SelectionConfig) Resolve(requested string) string {
	switch {
	case c.Override != "":
		return c.Override
	case strings.TrimSpace(requested) != "":
		return strings.TrimSpace(requested)
	default:
		return c.DefaultEngine
	}
}

// FeatureEnabled reports whether the new executor path should be used. A blank
// feature flag implies opt-in enabled to simplify local development. Set to
// "off" or "false" to force-disable.
func (c SelectionConfig) FeatureEnabled() bool {
	flag := strings.TrimSpace(c.FeatureFlag)
	if flag == "" {
		return true
	}
	if strings.EqualFold(flag, "off") || strings.EqualFold(flag, "false") {
		return false
	}
	return strings.EqualFold(flag, "on")
}

// ShadowEnabled determines whether to run the new executor without impacting
// primary execution status.
func (c SelectionConfig) ShadowEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(c.ShadowMode), "on")
}
