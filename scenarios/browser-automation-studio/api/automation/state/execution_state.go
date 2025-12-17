// Package state provides namespace-aware variable management for workflow execution.
//
// The ExecutionState type separates runtime state into three namespaces with
// different mutability semantics:
//   - @store/ - Mutable runtime state, writable via setVariable/storeResult
//   - @params/ - Read-only input parameters from execution request or subflow call
//   - @env/ - Read-only environment configuration from project settings
//
// This package was extracted from automation/executor/flow_utils.go to provide
// clear responsibility boundaries and enable reuse across the automation stack.
package state

import (
	"strconv"
	"strings"
	"sync"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// ExecutionState provides namespace-aware variable management for workflow execution.
// It separates runtime state (@store/), input parameters (@params/), and environment
// configuration (@env/) into distinct namespaces with different mutability semantics.
//
// Thread Safety: ExecutionState uses a mutex to protect concurrent access to state.
// All public methods are safe to call from multiple goroutines.
type ExecutionState struct {
	mu     sync.RWMutex
	store  map[string]any // @store/ - mutable runtime state
	params map[string]any // @params/ - read-only input parameters
	env    map[string]any // @env/ - read-only environment configuration

	// Execution tracking
	nextIndex    int
	entryChecked bool
}

// New creates a new ExecutionState with the given initial values.
// All parameters can be nil; empty maps will be created.
func New(initialStore, initialParams, env map[string]any) *ExecutionState {
	state := &ExecutionState{
		store:  make(map[string]any),
		params: make(map[string]any),
		env:    make(map[string]any),
	}
	for k, v := range initialStore {
		state.store[k] = v
	}
	for k, v := range initialParams {
		state.params[k] = v
	}
	for k, v := range env {
		state.env[k] = v
	}
	return state
}

// NewFromStore creates an ExecutionState with only store values (legacy compatibility).
func NewFromStore(seed map[string]any) *ExecutionState {
	return New(seed, nil, nil)
}

// Namespace constants for reference resolution.
const (
	NamespaceStore  = "store"
	NamespaceParams = "params"
	NamespaceEnv    = "env"
)

// GetNamespace returns a copy of the specified namespace map.
// Returns nil if the namespace is not recognized.
func (s *ExecutionState) GetNamespace(namespace string) map[string]any {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch namespace {
	case NamespaceStore:
		return copyMap(s.store)
	case NamespaceParams:
		return copyMap(s.params)
	case NamespaceEnv:
		return copyMap(s.env)
	default:
		return nil
	}
}

// Get retrieves a value from the store namespace using the given key.
// Returns (value, true) if found, (nil, false) otherwise.
func (s *ExecutionState) Get(key string) (any, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.store[key]
	return v, ok
}

// Resolve looks up a value using dot/index path notation (e.g., user.name, items.0).
// Falls back to direct lookup if path resolution fails.
func (s *ExecutionState) Resolve(path string) (any, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resolveInternal(path)
}

func (s *ExecutionState) resolveInternal(path string) (any, bool) {
	// Try direct lookup first
	if v, ok := s.store[path]; ok {
		return v, true
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, false
	}

	current, ok := s.store[parts[0]]
	if !ok {
		return nil, false
	}

	for _, part := range parts[1:] {
		switch val := current.(type) {
		case map[string]any:
			current, ok = val[part]
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(val) {
				return nil, false
			}
			current = val[idx]
			ok = true
		default:
			return nil, false
		}
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// ResolveNamespaced looks up a value in the specified namespace using dot notation.
// Returns (value, true) if found, (nil, false) otherwise.
func (s *ExecutionState) ResolveNamespaced(namespace, path string) (any, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ns map[string]any
	switch namespace {
	case NamespaceStore:
		ns = s.store
	case NamespaceParams:
		ns = s.params
	case NamespaceEnv:
		ns = s.env
	default:
		return nil, false
	}

	return resolvePath(ns, path)
}

// Set stores a value in the @store/ namespace (the only mutable namespace).
func (s *ExecutionState) Set(key string, value any) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		s.store = make(map[string]any)
	}
	s.store[key] = value
}

// Merge merges the given map into the @store/ namespace.
func (s *ExecutionState) Merge(vars map[string]any) {
	if s == nil || vars == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range vars {
		s.store[k] = v
	}
}

// CopyStore returns a deep copy of the @store/ namespace.
func (s *ExecutionState) CopyStore() map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyMap(s.store)
}

// CopyParams returns a deep copy of the @params/ namespace.
func (s *ExecutionState) CopyParams() map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyMap(s.params)
}

// CopyEnv returns a deep copy of the @env/ namespace.
func (s *ExecutionState) CopyEnv() map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyMap(s.env)
}

// MarkEntryChecked marks that the entry selector check has been performed.
func (s *ExecutionState) MarkEntryChecked() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entryChecked = true
}

// HasCheckedEntry returns true if the entry selector check has been performed.
func (s *ExecutionState) HasCheckedEntry() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entryChecked
}

// SetNextIndexFromPlan updates the next available step index based on the plan.
func (s *ExecutionState) SetNextIndexFromPlan(plan contracts.ExecutionPlan) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	maxIdx := -1
	for _, instr := range plan.Instructions {
		if instr.Index > maxIdx {
			maxIdx = instr.Index
		}
	}
	maxIdx = maxInt(maxIdx, maxGraphIndex(plan.Graph))
	if maxIdx >= s.nextIndex {
		s.nextIndex = maxIdx + 1
	}
}

// AllocateIndexRange reserves a range of step indices for subflow execution.
// Returns the base index of the allocated range.
func (s *ExecutionState) AllocateIndexRange(count int) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	base := s.nextIndex
	s.nextIndex += count
	return base
}

// AllVars returns all store variables as a flat map (legacy compatibility).
// Deprecated: Use GetNamespace(NamespaceStore) instead.
func (s *ExecutionState) AllVars() map[string]any {
	return s.CopyStore()
}

// resolvePath resolves a dot-notation path against a map.
func resolvePath(m map[string]any, path string) (any, bool) {
	if m == nil || path == "" {
		return nil, false
	}

	// Try direct lookup first
	if v, ok := m[path]; ok {
		return v, true
	}

	// Handle dot notation
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, false
	}

	current, ok := m[parts[0]]
	if !ok {
		return nil, false
	}

	for _, part := range parts[1:] {
		switch val := current.(type) {
		case map[string]any:
			current, ok = val[part]
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(val) {
				return nil, false
			}
			current = val[idx]
			ok = true
		default:
			return nil, false
		}
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func copyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxGraphIndex(graph *contracts.PlanGraph) int {
	if graph == nil {
		return -1
	}
	maxIdx := -1
	for _, step := range graph.Steps {
		if step.Index > maxIdx {
			maxIdx = step.Index
		}
		if step.Loop != nil {
			if nested := maxGraphIndex(step.Loop); nested > maxIdx {
				maxIdx = nested
			}
		}
	}
	return maxIdx
}
