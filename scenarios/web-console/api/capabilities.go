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
}

type CapabilityRegistry struct {
	defs     []CapabilityDef
	checkers map[string]StatusChecker

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

func (r *CapabilityRegistry) IsAvailable(ctx context.Context, capabilityID string) bool {
	for _, s := range r.Resolve(ctx) {
		if s.ID == capabilityID {
			return s.Status == StatusAvailable
		}
	}
	return false
}
