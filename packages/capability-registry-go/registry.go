// Package capabilityregistry defines the scenario-independent capability
// registry contract. Scenario packages provide concrete definitions and
// reachability checkers; this package owns validation, state projection, and
// the stable description consumed by APIs and conformance tooling.
package capabilityregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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

// Criticality states whether the absence of a capability matters to the
// owning service. It is declared by a catalogue and is never inferred from
// a provider probe.
type Criticality string

const (
	CriticalityRequired Criticality = "required"
	CriticalityOptional Criticality = "optional"
)

const (
	ActionKindNone            ActionKind = ""
	ActionKindOperatorCommand ActionKind = "operator_command"
	ActionKindScenarioStart   ActionKind = "scenario_start"
	ActionKindScenarioRestart ActionKind = "scenario_restart"
	ActionKindOwnerGuidance   ActionKind = "owner_guidance"
)

type Def struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	DependencyKind DependencyKind  `json:"dependencyKind"`
	DependencySlug string          `json:"dependencySlug"`
	Features       []string        `json:"features"`
	Platform       PlatformVerdict `json:"platform,omitempty"`
	// Criticality defaults to optional. A catalogue must explicitly declare
	// required when a capability is allowed to gate service health.
	Criticality     Criticality `json:"criticality,omitempty"`
	ActionKind      ActionKind  `json:"actionKind,omitempty"`
	ActionLabel     string      `json:"actionLabel,omitempty"`
	OperatorCommand string      `json:"operatorCommand,omitempty"`
}

// ResolvedCriticality returns the declared criticality, applying the safe
// optional default for older catalogues.
func (d Def) ResolvedCriticality() Criticality {
	if d.Criticality == "" {
		return CriticalityOptional
	}
	return d.Criticality
}

type State struct {
	Def
	Status                 Status            `json:"status"`
	Message                string            `json:"message,omitempty"`
	ReasonCode             string            `json:"reasonCode,omitempty"`
	ActionKind             ActionKind        `json:"actionKind,omitempty"`
	ActionLabel            string            `json:"actionLabel,omitempty"`
	OperatorCommand        string            `json:"operatorCommand,omitempty"`
	CheckedAt              string            `json:"checkedAt,omitempty"`
	FeatureStatus          map[string]string `json:"featureStatus,omitempty"`
	FeatureReason          map[string]string `json:"featureReason,omitempty"`
	FeatureOperatorCommand map[string]string `json:"featureOperatorCommand,omitempty"`
	// ProviderStatus preserves the individual provider verdicts that make up
	// a feature rollup. Consumers may use this to choose a preferred provider
	// without confusing an available fallback with the preferred provider.
	ProviderStatus map[string]string `json:"providerStatus,omitempty"`
	ProviderFeatures map[string]string `json:"providerFeatures,omitempty"`
}

type Checker interface {
	Check(context.Context) (Status, string)
}

type ResultChecker interface {
	CheckResult(context.Context) CheckResult
}

type tierCache struct {
	states []State
	at     time.Time
}

type CheckResult struct {
	Status                 Status
	Message                string
	ReasonCode             string
	ActionKind             ActionKind
	ActionLabel            string
	OperatorCommand        string
	FeatureStatus          map[string]string
	FeatureReason          map[string]string
	FeatureOperatorCommand map[string]string
	ProviderStatus         map[string]string
	ProviderFeatures       map[string]string
}

// CapabilityRollup is the serviceability view of one logical capability.
// Providers are retained so callers can expose their individual state while
// using the same any-available serviceability rule.
type CapabilityRollup struct {
	Name                 string
	Providers            []State
	Serviceable          bool
	UnavailableProviders []string
}

// Serviceable reports whether at least one provider is available.
func Serviceable(states []State) bool {
	for _, state := range states {
		if state.Status == StatusAvailable {
			return true
		}
	}
	return false
}

// RollupByCapability groups provider states using the caller-supplied
// grouping function. The shared registry deliberately does not know any
// scenario's capability vocabulary.
func RollupByCapability(states []State, groups func(State) []string) []CapabilityRollup {
	if groups == nil {
		return nil
	}
	byName := make(map[string]*CapabilityRollup)
	var order []string
	for _, state := range states {
		seen := make(map[string]struct{})
		for _, name := range groups(state) {
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			rollup, ok := byName[name]
			if !ok {
				rollup = &CapabilityRollup{Name: name}
				byName[name] = rollup
				order = append(order, name)
			}
			rollup.Providers = append(rollup.Providers, state)
			if state.Status == StatusUnavailable {
				rollup.UnavailableProviders = append(rollup.UnavailableProviders, state.ID)
			}
		}
	}
	out := make([]CapabilityRollup, 0, len(order))
	for _, name := range order {
		rollup := *byName[name]
		rollup.Serviceable = Serviceable(rollup.Providers)
		out = append(out, rollup)
	}
	return out
}

type Registry struct {
	defs             []Def
	checkers         map[string]Checker
	livenessCheckers map[string]Checker
	mu               sync.RWMutex
	full             tierCache
	liveness         tierCache
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

func (r *Registry) SetLivenessCheckers(checkers map[string]Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.livenessCheckers = checkers
}

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
	if r.full.states != nil && (r.cacheTTL <= 0 || r.now().Sub(r.full.at) < r.cacheTTL) {
		out := append([]State(nil), r.full.states...)
		r.mu.RUnlock()
		return out
	}
	defs := append([]Def(nil), r.defs...)
	checkers := cloneCheckers(r.checkers)
	r.mu.RUnlock()
	states := resolveStates(ctx, defs, checkers, fullCheckerBudget, r.now)
	r.mu.Lock()
	if r.full.states != nil && (r.cacheTTL <= 0 || r.now().Sub(r.full.at) < r.cacheTTL) {
		out := append([]State(nil), r.full.states...)
		r.mu.Unlock()
		return out
	}
	r.full = tierCache{states: states, at: r.now()}
	out := append([]State(nil), states...)
	r.mu.Unlock()
	return out
}

func (r *Registry) ResolveLiveness(ctx context.Context) []State {
	r.mu.RLock()
	fresh := r.liveness.states != nil && (r.cacheTTL <= 0 || r.now().Sub(r.liveness.at) < r.cacheTTL)
	cached := append([]State(nil), r.liveness.states...)
	defs := append([]Def(nil), r.defs...)
	checkers := cloneCheckers(r.livenessCheckers)
	r.mu.RUnlock()
	if fresh {
		return cached
	}
	states := resolveStates(ctx, defs, checkers, livenessCheckerBudget, r.now)
	r.mu.Lock()
	if r.liveness.states != nil && (r.cacheTTL <= 0 || r.now().Sub(r.liveness.at) < r.cacheTTL) {
		out := append([]State(nil), r.liveness.states...)
		r.mu.Unlock()
		return out
	}
	r.liveness = tierCache{states: states, at: r.now()}
	out := append([]State(nil), states...)
	r.mu.Unlock()
	return out
}

func (r *Registry) ResolveForce(ctx context.Context) []State {
	r.mu.Lock()
	r.full = tierCache{}
	defs := append([]Def(nil), r.defs...)
	checkers := cloneCheckers(r.checkers)
	r.mu.Unlock()
	states := resolveStates(ctx, defs, checkers, fullCheckerBudget, r.now)
	r.mu.Lock()
	r.full = tierCache{states: states, at: r.now()}
	out := append([]State(nil), states...)
	r.mu.Unlock()
	return out
}

func resolveStates(ctx context.Context, defs []Def, checkers map[string]Checker, budget time.Duration, now func() time.Time) []State {
	checkedAt := now().UTC().Format(time.RFC3339)
	states := make([]State, len(defs))
	for i, def := range defs {
		state := stateFor(def, checkedAt)
		if def.Platform.Support == PlatformUnsupported {
			state.Status = StatusUnavailable
			state.Message = "unavailable by design"
			if def.Platform.Reason != "" {
				state.Message += ": " + def.Platform.Reason
			}
			states[i] = state
			continue
		}
		if checker := checkers[def.ID]; checker != nil {
			if err := ctx.Err(); err != nil {
				state.Message = "not evaluated: deadline reached"
				states[i] = state
				continue
			}
			apply(&state, checkWithBudget(ctx, checker, budget))
		}
		states[i] = state
	}
	return states
}

func cloneCheckers(in map[string]Checker) map[string]Checker {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]Checker, len(in))
	for id, checker := range in {
		out[id] = checker
	}
	return out
}

// IsProviderLive observes one provider using only its cheap liveness checker.
// A short freshness window avoids repeating the probe for every paragraph in
// a playback queue while keeping request-path decisions independent from the
// display cache and from expensive readiness checks.
func (r *Registry) IsProviderLive(ctx context.Context, id string) bool {
	const freshness = 2 * time.Second
	now := r.now()
	r.mu.RLock()
	if now.Sub(r.liveness.at) < freshness {
		for _, state := range r.liveness.states {
			if state.ID == id {
				r.mu.RUnlock()
				return state.Status == StatusAvailable
			}
		}
	}
	checker := r.livenessCheckers[id]
	r.mu.RUnlock()
	if checker == nil {
		return false
	}
	return checkWithBudget(ctx, checker, livenessCheckerBudget).Status == StatusAvailable
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
	r.full = tierCache{}
	r.liveness = tierCache{}
}

func (r *Registry) CacheTTL() time.Duration { return r.cacheTTL }

func check(ctx context.Context, checker Checker) CheckResult {
	if rich, ok := checker.(ResultChecker); ok {
		return rich.CheckResult(ctx)
	}
	status, message := checker.Check(ctx)
	return CheckResult{Status: status, Message: message}
}

const (
	livenessCheckerBudget = 3 * time.Second
	fullCheckerBudget     = 30 * time.Second
)

func checkWithBudget(parent context.Context, checker Checker, budget time.Duration) CheckResult {
	checkCtx, cancel := context.WithTimeout(parent, budget)
	defer cancel()
	return check(checkCtx, checker)
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
	if result.FeatureStatus != nil {
		state.FeatureStatus = make(map[string]string, len(result.FeatureStatus))
		for feature, status := range result.FeatureStatus {
			state.FeatureStatus[feature] = status
		}
	}
	if result.FeatureReason != nil {
		state.FeatureReason = maps.Clone(result.FeatureReason)
	}
	if result.FeatureOperatorCommand != nil {
		state.FeatureOperatorCommand = maps.Clone(result.FeatureOperatorCommand)
	}
	if result.ProviderStatus != nil {
		state.ProviderStatus = maps.Clone(result.ProviderStatus)
	}
	if result.ProviderFeatures != nil {
		state.ProviderFeatures = maps.Clone(result.ProviderFeatures)
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
