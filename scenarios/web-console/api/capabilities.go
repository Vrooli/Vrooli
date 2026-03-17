package main

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

type CapabilityStatus string

const (
	StatusAvailable   CapabilityStatus = "available"
	StatusUnavailable CapabilityStatus = "unavailable"
	StatusUnknown     CapabilityStatus = "unknown"
)

type CapabilityDef struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	DependencyKind DependencyKind `json:"dependencyKind"`
	DependencySlug string         `json:"dependencySlug"`
	Features       []string       `json:"features"`
}

type CapabilityState struct {
	CapabilityDef
	Status    CapabilityStatus `json:"status"`
	Message   string           `json:"message,omitempty"`
	CheckedAt string           `json:"checkedAt,omitempty"`
}

type StatusChecker interface {
	Check(ctx context.Context) (CapabilityStatus, string)
}

var knownCapabilities = []CapabilityDef{
	{
		ID:             "whisper-stt",
		Name:           "Whisper STT",
		Description:    "Speech-to-text transcription via Whisper",
		DependencyKind: DependencyResource,
		DependencySlug: "whisper",
		Features:       []string{"voice-input", "voice-streaming"},
	},
	{
		ID:             "kokoro-tts",
		Name:           "Kokoro TTS",
		Description:    "Text-to-speech synthesis via Kokoro",
		DependencyKind: DependencyResource,
		DependencySlug: "kokoro",
		Features:       []string{"voice-output"},
	},
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
}

type CapabilityRegistry struct {
	defs             []CapabilityDef
	checkers         map[string]StatusChecker
	livenessCheckers map[string]StatusChecker

	mu       sync.RWMutex
	cached   []CapabilityState
	cachedAt time.Time
	cacheTTL time.Duration
}

func NewCapabilityRegistry(defs []CapabilityDef, checkers map[string]StatusChecker, cacheTTL time.Duration) *CapabilityRegistry {
	return &CapabilityRegistry{
		defs:     defs,
		checkers: checkers,
		cacheTTL: cacheTTL,
	}
}

// SetLivenessCheckers registers lightweight health-only checkers used by
// ResolveLiveness. These skip expensive verification (e.g. test transcription)
// and only perform a fast HTTP liveness check.
func (r *CapabilityRegistry) SetLivenessCheckers(lc map[string]StatusChecker) {
	r.livenessCheckers = lc
}

func (r *CapabilityRegistry) Resolve(ctx context.Context) []CapabilityState {
	r.mu.RLock()
	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]CapabilityState, len(r.cached))
		copy(result, r.cached)
		r.mu.RUnlock()
		return result
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]CapabilityState, len(r.cached))
		copy(result, r.cached)
		return result
	}

	now := time.Now().UTC()
	states := make([]CapabilityState, len(r.defs))
	for i, def := range r.defs {
		state := CapabilityState{
			CapabilityDef: def,
			Status:        StatusUnknown,
			CheckedAt:     now.Format(time.RFC3339),
		}
		if checker, ok := r.checkers[def.ID]; ok {
			state.Status, state.Message = checker.Check(ctx)
		}
		states[i] = state
	}

	r.cached = states
	r.cachedAt = now

	result := make([]CapabilityState, len(states))
	copy(result, states)
	return result
}

// ResolveLiveness returns capability states using fast liveness-only checks.
// If the full-check cache is still fresh, it returns that directly (no extra
// work). When the cache is stale, it uses lightweight liveness checkers instead
// of the full checkers (which may include expensive operations like test
// transcription). The liveness results are NOT written to the main cache to
// avoid masking a broken-but-live service.
func (r *CapabilityRegistry) ResolveLiveness(ctx context.Context) []CapabilityState {
	// Return cached full-check results if still fresh.
	r.mu.RLock()
	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]CapabilityState, len(r.cached))
		copy(result, r.cached)
		r.mu.RUnlock()
		return result
	}
	r.mu.RUnlock()

	// Cache is stale -- use liveness checkers for a fast response.
	// Fall back to full checkers if no liveness checkers are configured.
	checkers := r.livenessCheckers
	if len(checkers) == 0 {
		return r.Resolve(ctx)
	}

	now := time.Now().UTC()
	states := make([]CapabilityState, len(r.defs))
	for i, def := range r.defs {
		state := CapabilityState{
			CapabilityDef: def,
			Status:        StatusUnknown,
			CheckedAt:     now.Format(time.RFC3339),
		}
		if checker, ok := checkers[def.ID]; ok {
			state.Status, state.Message = checker.Check(ctx)
		} else if checker, ok := r.checkers[def.ID]; ok {
			// No liveness checker for this cap -- fall back to full check.
			state.Status, state.Message = checker.Check(ctx)
		}
		states[i] = state
	}

	return states
}

func (r *CapabilityRegistry) IsAvailable(ctx context.Context, capabilityID string) bool {
	for _, s := range r.Resolve(ctx) {
		if s.ID == capabilityID {
			return s.Status == StatusAvailable
		}
	}
	return false
}
