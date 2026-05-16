// Package capabilities models the capability registry: the set of optional
// resources (Whisper, Kokoro, Ollama, etc.) the web-console can use and
// whether each is currently reachable. The HTTP transport lives in
// handlers/capabilities; this package owns the in-process state and the
// Checker contract.
package capabilities

import (
	"context"
	"sync"
	"time"
)

type DependencyKind string

const (
	DependencyScenario DependencyKind = "scenario"
	DependencyResource DependencyKind = "resource"
)

type Status string

const (
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
	StatusUnknown     Status = "unknown"
)

type Def struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	DependencyKind DependencyKind `json:"dependencyKind"`
	DependencySlug string         `json:"dependencySlug"`
	Features       []string       `json:"features"`
}

type State struct {
	Def
	Status    Status `json:"status"`
	Message   string `json:"message,omitempty"`
	CheckedAt string `json:"checkedAt,omitempty"`
}

type Checker interface {
	Check(ctx context.Context) (Status, string)
}

// Known is the built-in capability catalogue that ships with the
// web-console scenario. Callers may pass a different slice to NewRegistry
// (tests do); production wiring in main.go uses this value.
var Known = []Def{
	{
		ID:             "ollama",
		Name:           "Ollama",
		Description:    "Local LLM inference for AI command generation",
		DependencyKind: DependencyResource,
		DependencySlug: "ollama",
		Features:       []string{"ai-command-generation"},
	},
	{
		ID:             "openrouter",
		Name:           "OpenRouter",
		Description:    "Cloud LLM API for AI command generation",
		DependencyKind: DependencyResource,
		DependencySlug: "openrouter",
		Features:       []string{"ai-command-generation"},
	},
	// Connected scenarios — the single source of truth for cross-scenario
	// integrations web-console expects to discover and adopt. Each entry
	// declares what features it would unlock once available. The local Whisper
	// and Kokoro resource entries above remain registered during the
	// audio-tools extraction prep; they collapse to the audio-tools entry
	// after web-console adopts audio-tools.
	{
		ID:             "audio-tools",
		Name:           "Audio Tools",
		Description:    "Shared audio capability scenario: STT, TTS, summarization, provider routing, BYOK/LPBS/local tiers, adoptable UI",
		DependencyKind: DependencyScenario,
		DependencySlug: "audio-tools",
		Features: []string{
			"voice-input",
			"voice-streaming",
			"voice-speaker-verification",
			"voice-enrollment",
			"voice-output",
			"tts-summarization",
			"tts-cache",
			"tts-paragraph-split",
			"audio-provider-routing",
		},
	},
}

type Registry struct {
	defs             []Def
	checkers         map[string]Checker
	livenessCheckers map[string]Checker

	mu       sync.RWMutex
	cached   []State
	cachedAt time.Time
	cacheTTL time.Duration
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return &Registry{
		defs:     defs,
		checkers: checkers,
		cacheTTL: cacheTTL,
	}
}

// SetLivenessCheckers registers lightweight health-only checkers used by
// ResolveLiveness. These skip expensive verification (e.g. test transcription)
// and only perform a fast HTTP liveness check.
func (r *Registry) SetLivenessCheckers(lc map[string]Checker) {
	r.livenessCheckers = lc
}

func (r *Registry) Resolve(ctx context.Context) []State {
	r.mu.RLock()
	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]State, len(r.cached))
		copy(result, r.cached)
		r.mu.RUnlock()
		return result
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]State, len(r.cached))
		copy(result, r.cached)
		return result
	}

	now := time.Now().UTC()
	states := make([]State, len(r.defs))
	for i, def := range r.defs {
		state := State{
			Def:       def,
			Status:    StatusUnknown,
			CheckedAt: now.Format(time.RFC3339),
		}
		if checker, ok := r.checkers[def.ID]; ok {
			state.Status, state.Message = checker.Check(ctx)
		}
		states[i] = state
	}

	r.cached = states
	r.cachedAt = now

	result := make([]State, len(states))
	copy(result, states)
	return result
}

// ResolveLiveness returns capability states using fast liveness-only checks.
// If the full-check cache is still fresh, it returns that directly (no extra
// work). When the cache is stale, it uses lightweight liveness checkers instead
// of the full checkers (which may include expensive operations like test
// transcription). The liveness results are NOT written to the main cache to
// avoid masking a broken-but-live service.
func (r *Registry) ResolveLiveness(ctx context.Context) []State {
	r.mu.RLock()
	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]State, len(r.cached))
		copy(result, r.cached)
		r.mu.RUnlock()
		return result
	}
	r.mu.RUnlock()

	checkers := r.livenessCheckers
	if len(checkers) == 0 {
		return r.Resolve(ctx)
	}

	now := time.Now().UTC()
	states := make([]State, len(r.defs))
	for i, def := range r.defs {
		state := State{
			Def:       def,
			Status:    StatusUnknown,
			CheckedAt: now.Format(time.RFC3339),
		}
		if checker, ok := checkers[def.ID]; ok {
			state.Status, state.Message = checker.Check(ctx)
		} else if checker, ok := r.checkers[def.ID]; ok {
			state.Status, state.Message = checker.Check(ctx)
		}
		states[i] = state
	}

	return states
}

func (r *Registry) IsAvailable(ctx context.Context, capabilityID string) bool {
	for _, s := range r.Resolve(ctx) {
		if s.ID == capabilityID {
			return s.Status == StatusAvailable
		}
	}
	return false
}
