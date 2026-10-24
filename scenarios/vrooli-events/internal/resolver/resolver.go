// Package resolver provides EmittingResolver — a decorator that wraps a
// discovery Resolver to emit events to vrooli-events on each resolve call
// and enforce sender-side policy from a live-synced cache.
//
// [REQ:DI-001] EmittingResolver decorator
// [REQ:DI-003] Sender-side policy cache
//
// DOC: docs/guides/integrating-a-scenario.md
// DOC: docs/internal/SEAMS.md#architecture-alignment
package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/emitter"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/fallback"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// Resolver is the interface for discovering scenario ports/URLs.
// Mirrors the contract of packages/api-core/discovery.Resolver.
type Resolver interface {
	ResolveScenarioPort(ctx context.Context, scenario, portKey string) (int, error)
	ResolveScenarioURL(ctx context.Context, scenario, portKey string) (string, error)
}

// PolicyDeniedError is returned when a policy rule blocks a resolve call.
type PolicyDeniedError struct {
	RuleID int64  `json:"rule_id"`
	Reason string `json:"reason"`
}

func (e *PolicyDeniedError) Error() string {
	return fmt.Sprintf("policy denied: rule %d: %s", e.RuleID, e.Reason)
}

// Config configures the EmittingResolver.
type Config struct {
	Inner          Resolver         // underlying resolver (required)
	Emitter        *emitter.Emitter // event emitter (required)
	SourceScenario string           // name of this scenario
	PolicyRules    []policy.Rule    // initial policy snapshot (may be nil)
	DefaultMode    fallback.Mode    // fail-open or fail-closed when no rules
}

// EmittingResolver wraps an existing Resolver, emitting events on each call
// and enforcing locally-cached policies before delegating.
type EmittingResolver struct {
	inner          Resolver
	emitter        *emitter.Emitter
	sourceScenario string
	defaultMode    fallback.Mode

	mu    sync.RWMutex
	rules []policy.Rule
}

// NewEmittingResolver creates an EmittingResolver with an optional initial
// policy snapshot. Pass nil PolicyRules for an empty cache (fail-open by default).
func NewEmittingResolver(cfg Config) *EmittingResolver {
	mode := cfg.DefaultMode
	if mode == "" {
		mode = fallback.ModeFailOpen
	}
	rules := cfg.PolicyRules
	if rules == nil {
		rules = []policy.Rule{}
	}
	return &EmittingResolver{
		inner:          cfg.Inner,
		emitter:        cfg.Emitter,
		sourceScenario: cfg.SourceScenario,
		defaultMode:    mode,
		rules:          rules,
	}
}

// ResolveScenarioPort resolves a port, checking policy first and emitting an event after.
func (r *EmittingResolver) ResolveScenarioPort(ctx context.Context, scenario, portKey string) (int, error) {
	if err := r.checkPolicy(scenario); err != nil {
		return 0, err
	}

	start := time.Now()
	port, err := r.inner.ResolveScenarioPort(ctx, scenario, portKey)
	r.emitResolve(scenario, portKey, port, time.Since(start), err)
	return port, err
}

// ResolveScenarioURL resolves a URL, checking policy first and emitting an event after.
func (r *EmittingResolver) ResolveScenarioURL(ctx context.Context, scenario, portKey string) (string, error) {
	if err := r.checkPolicy(scenario); err != nil {
		return "", err
	}

	start := time.Now()
	url, err := r.inner.ResolveScenarioURL(ctx, scenario, portKey)
	r.emitResolve(scenario, portKey, 0, time.Since(start), err)
	return url, err
}

// UpdateRules replaces the cached policy rules. Thread-safe.
// Called by the SSE subscription loop when policy changes arrive.
func (r *EmittingResolver) UpdateRules(rules []policy.Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = rules
}

// ApplyEvent applies a single PolicyEvent (from SSE) to the cached rules.
func (r *EmittingResolver) ApplyEvent(evt policy.PolicyEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch evt.Action {
	case "deleted":
		for i, rule := range r.rules {
			if rule.ID == evt.RuleID {
				r.rules = append(r.rules[:i], r.rules[i+1:]...)
				break
			}
		}
	case "created":
		if evt.Rule != nil {
			r.rules = append(r.rules, *evt.Rule)
		}
	case "updated":
		if evt.Rule != nil {
			for i, rule := range r.rules {
				if rule.ID == evt.RuleID {
					r.rules[i] = *evt.Rule
					return
				}
			}
			// Not found — add it
			r.rules = append(r.rules, *evt.Rule)
		}
	}
}

// Rules returns a copy of the cached rules (for testing/health).
func (r *EmittingResolver) Rules() []policy.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make([]policy.Rule, len(r.rules))
	copy(cp, r.rules)
	return cp
}

// checkPolicy evaluates cached rules against the target scenario.
func (r *EmittingResolver) checkPolicy(target string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.rules) == 0 {
		return nil // no rules → allow
	}

	for _, rule := range r.rules {
		if !rule.Enabled {
			continue
		}
		if rule.RuleType != policy.RuleTypeAccess {
			continue
		}
		if rule.SourceScenario != "*" && rule.SourceScenario != r.sourceScenario {
			continue
		}
		if rule.TargetScenario != "*" && rule.TargetScenario != target {
			continue
		}
		if rule.Effect == policy.EffectDeny {
			return &PolicyDeniedError{
				RuleID: rule.ID,
				Reason: "denied by access control rule",
			}
		}
		return nil // explicit allow
	}

	return nil // no matching rule → allow
}

// emitResolve fires a resolve event asynchronously.
func (r *EmittingResolver) emitResolve(target, portKey string, port int, dur time.Duration, resolveErr error) {
	if r.emitter == nil {
		return
	}

	success := resolveErr == nil
	errMsg := ""
	if resolveErr != nil {
		errMsg = resolveErr.Error()
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"target_scenario": target,
		"port_key":        portKey,
		"resolved_port":   port,
		"duration_ms":     dur.Milliseconds(),
		"success":         success,
		"error":           errMsg,
	})

	r.emitter.Emit(emitter.EventPayload{
		EventType:      r.sourceScenario + ".discovery.resolve.v1",
		SourceScenario: r.sourceScenario,
		TargetScenario: target,
		Payload:        payload,
	})
}
