package engine

import (
	"os"
	"strings"
)

// SelectionConfig controls how the executor chooses an engine implementation.
type SelectionConfig struct {
	DefaultEngine string // Fallback when no override/request provided.
	Override      string // Explicit override (e.g., env/feature flag).
}

// FromEnv builds SelectionConfig from environment variables. The variables are
// intentionally simple to avoid widening handler contracts:
//
//	ENGINE: global default engine name (default: playwright)
//	ENGINE_OVERRIDE: hard override for all executions (e.g., desktop)
func FromEnv() SelectionConfig {
	return SelectionConfig{
		DefaultEngine: strings.TrimSpace(os.Getenv("ENGINE")),
		Override:      strings.TrimSpace(os.Getenv("ENGINE_OVERRIDE")),
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
