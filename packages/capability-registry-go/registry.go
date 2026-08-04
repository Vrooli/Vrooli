// Package capabilityregistry defines the scenario-independent capability
// registry contract. Scenario packages provide concrete definitions and
// reachability checkers; this package owns validation, state projection, and
// the stable description consumed by APIs and conformance tooling.
package capabilityregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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

type ActionKind string

type PlatformSupport string

const (
	PlatformSupported   PlatformSupport = "supported"
	PlatformDegraded    PlatformSupport = "degraded"
	PlatformUnsupported PlatformSupport = "unsupported"
)

type PlatformVerdict struct {
	Support PlatformSupport `json:"support,omitempty"`
	Reason  string          `json:"reason,omitempty"`
}

const (
	ActionKindNone            ActionKind = ""
	ActionKindOperatorCommand ActionKind = "operator_command"
	ActionKindScenarioStart   ActionKind = "scenario_start"
	ActionKindScenarioRestart ActionKind = "scenario_restart"
	ActionKindOwnerGuidance   ActionKind = "owner_guidance"
)

type Def struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	DependencyKind  DependencyKind  `json:"dependencyKind"`
	DependencySlug  string          `json:"dependencySlug"`
	Features        []string        `json:"features"`
	Platform        PlatformVerdict `json:"platform,omitempty"`
	ActionKind      ActionKind      `json:"actionKind,omitempty"`
	ActionLabel     string          `json:"actionLabel,omitempty"`
	OperatorCommand string          `json:"operatorCommand,omitempty"`
}

type State struct {
	Def
	Status          Status     `json:"status"`
	Message         string     `json:"message,omitempty"`
	ReasonCode      string     `json:"reasonCode,omitempty"`
	ActionKind      ActionKind `json:"actionKind,omitempty"`
	ActionLabel     string     `json:"actionLabel,omitempty"`
	OperatorCommand string     `json:"operatorCommand,omitempty"`
	CheckedAt       string     `json:"checkedAt,omitempty"`
}

type Checker interface {
	Check(context.Context) (Status, string)
}

type ResultChecker interface {
	CheckResult(context.Context) CheckResult
}

type CheckResult struct {
	Status          Status
	Message         string
	ReasonCode      string
	ActionKind      ActionKind
	ActionLabel     string
	OperatorCommand string
}

type Registry struct {
	defs             []Def
	checkers         map[string]Checker
	livenessCheckers map[string]Checker
	mu               sync.RWMutex
	cached           []State
	cachedAt         time.Time
	cacheTTL         time.Duration
	now              func() time.Time
}

func New(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return NewWithClock(defs, checkers, cacheTTL, time.Now)
}

func NewWithClock(defs []Def, checkers map[string]Checker, cacheTTL time.Duration, now func() time.Time) *Registry {
	if now == nil {
		now = time.Now
	}
	return &Registry{defs: append([]Def(nil), defs...), checkers: checkers, cacheTTL: cacheTTL, now: now}
}

func (r *Registry) SetLivenessCheckers(checkers map[string]Checker) { r.livenessCheckers = checkers }

func (r *Registry) Validate() error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}
	seen := make(map[string]struct{}, len(r.defs))
	for i, def := range r.defs {
		if def.ID == "" {
			return fmt.Errorf("definition %d has no id", i)
		}
		if def.Description == "" {
			return fmt.Errorf("definition %q has no description", def.ID)
		}
		if def.DependencyKind != DependencyScenario && def.DependencyKind != DependencyResource {
			return fmt.Errorf("definition %q has invalid dependency kind", def.ID)
		}
		if def.DependencySlug == "" {
			return fmt.Errorf("definition %q has no dependency slug", def.ID)
		}
		if _, duplicate := seen[def.ID]; duplicate {
			return fmt.Errorf("definition %q is duplicated", def.ID)
		}
		seen[def.ID] = struct{}{}
		checker, ok := r.checkers[def.ID]
		if !ok || isNil(checker) {
			return fmt.Errorf("definition %q has no reachability checker", def.ID)
		}
		if resultChecker, ok := checker.(ResultChecker); ok && isNil(resultChecker) {
			return fmt.Errorf("definition %q has a nil checker", def.ID)
		}
	}
	for id, checker := range r.checkers {
		if isNil(checker) {
			return fmt.Errorf("checker %q is nil", id)
		}
	}
	return nil
}

// ValidateStates checks the part of the contract that only exists after a
// reachability check has run. An unavailable dependency must explain how an
// operator can recover it; otherwise the registry would report a dead end.
func (r *Registry) ValidateStates(states []State) error {
	for _, state := range states {
		if state.Status == StatusUnavailable && state.ActionKind == ActionKindNone {
			return fmt.Errorf("state %q is unavailable without an operator action", state.ID)
		}
	}
	return nil
}

// Describe returns stable JSON containing definitions and current states. The
// definition order is intentionally preserved because it is the UI order.
func (r *Registry) Describe(ctx context.Context) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	states := r.Resolve(ctx)
	if err := r.ValidateStates(states); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Definitions []Def   `json:"definitions"`
		States      []State `json:"states"`
	}{r.defs, states})
}

func (r *Registry) Resolve(ctx context.Context) []State {
	r.mu.RLock()
	if r.cached != nil && (r.cacheTTL <= 0 || r.now().Sub(r.cachedAt) < r.cacheTTL) {
		out := append([]State(nil), r.cached...)
		r.mu.RUnlock()
		return out
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cached != nil && (r.cacheTTL <= 0 || r.now().Sub(r.cachedAt) < r.cacheTTL) {
		return append([]State(nil), r.cached...)
	}
	now := r.now().UTC().Format(time.RFC3339)
	states := make([]State, len(r.defs))
	for i, def := range r.defs {
		state := stateFor(def, now)
		if def.Platform.Support == PlatformUnsupported {
			state.Status = StatusUnavailable
			state.Message = "unavailable by design"
			if def.Platform.Reason != "" {
				state.Message += ": " + def.Platform.Reason
			}
			states[i] = state
			continue
		}
		if checker, ok := r.checkers[def.ID]; ok && checker != nil {
			apply(&state, check(ctx, checker))
		}
		states[i] = state
	}
	r.cached = states
	r.cachedAt = r.now()
	return append([]State(nil), states...)
}

func (r *Registry) ResolveLiveness(ctx context.Context) []State {
	r.mu.RLock()
	fresh := r.cached != nil && (r.cacheTTL <= 0 || r.now().Sub(r.cachedAt) < r.cacheTTL)
	cached := append([]State(nil), r.cached...)
	r.mu.RUnlock()
	if fresh {
		return cached
	}
	if len(r.livenessCheckers) == 0 {
		return r.Resolve(ctx)
	}
	states := make([]State, len(r.defs))
	for i, def := range r.defs {
		state := stateFor(def, r.now().UTC().Format(time.RFC3339))
		if def.Platform.Support == PlatformUnsupported {
			state.Status = StatusUnavailable
			state.Message = "unavailable by design"
			if def.Platform.Reason != "" {
				state.Message += ": " + def.Platform.Reason
			}
			states[i] = state
			continue
		}
		// A configured liveness map is intentionally authoritative. Missing
		// entries remain unknown rather than silently running an expensive full
		// checker from the wrong health tier.
		checker := r.livenessCheckers[def.ID]
		if checker != nil {
			apply(&state, check(ctx, checker))
		}
		states[i] = state
	}
	return states
}

func (r *Registry) ResolveForce(ctx context.Context) []State {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cached = nil
	now := r.now().UTC().Format(time.RFC3339)
	states := make([]State, len(r.defs))
	for i, def := range r.defs {
		state := stateFor(def, now)
		if def.Platform.Support == PlatformUnsupported {
			state.Status = StatusUnavailable
			state.Message = "unavailable by design"
			if def.Platform.Reason != "" {
				state.Message += ": " + def.Platform.Reason
			}
		} else if checker := r.checkers[def.ID]; checker != nil {
			apply(&state, check(ctx, checker))
		}
		states[i] = state
	}
	r.cached, r.cachedAt = states, r.now()
	return append([]State(nil), states...)
}

func (r *Registry) IsAvailable(ctx context.Context, id string) bool {
	for _, state := range r.Resolve(ctx) {
		if state.ID == id {
			return state.Status == StatusAvailable
		}
	}
	return false
}

func (r *Registry) Invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cached = nil
	r.cachedAt = time.Time{}
}

func (r *Registry) CacheTTL() time.Duration { return r.cacheTTL }

func check(ctx context.Context, checker Checker) CheckResult {
	if rich, ok := checker.(ResultChecker); ok {
		return rich.CheckResult(ctx)
	}
	status, message := checker.Check(ctx)
	return CheckResult{Status: status, Message: message}
}

func apply(state *State, result CheckResult) {
	state.Status, state.Message, state.ReasonCode = result.Status, result.Message, result.ReasonCode
	if result.ActionKind != ActionKindNone {
		state.ActionKind = result.ActionKind
	}
	if result.ActionLabel != "" {
		state.ActionLabel = result.ActionLabel
	}
	if result.OperatorCommand != "" {
		state.OperatorCommand = result.OperatorCommand
	}
}

func stateFor(def Def, checkedAt string) State {
	actionKind := def.ActionKind
	actionLabel := def.ActionLabel
	operatorCommand := def.OperatorCommand
	if actionKind == ActionKindNone {
		actionKind = ActionKindOwnerGuidance
		actionLabel = "Review dependency health"
		command := "resource"
		if def.DependencyKind == DependencyScenario {
			command = "scenario"
		}
		operatorCommand = fmt.Sprintf("vrooli %s status %s --json", command, def.DependencySlug)
	}
	return State{
		Def:             def,
		Status:          StatusUnknown,
		ActionKind:      actionKind,
		ActionLabel:     actionLabel,
		OperatorCommand: operatorCommand,
		CheckedAt:       checkedAt,
	}
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
