package retention

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// BuiltinFactory builds the framework pruner for a target. The engine holds only
// the seam so it can be exercised without touching a filesystem or a database.
type BuiltinFactory func(spec Spec) (Pruner, error)

// EngineConfig configures one engine.
type EngineConfig struct {
	// Specs are the parsed budget declarations to enforce.
	Specs []Spec
	// Builtin builds pruners for budgets declaring pruner "builtin". Required
	// only if at least one spec declares it.
	Builtin BuiltinFactory
	// Registry resolves budgets declaring pruner "custom". A budget naming a
	// pruner nothing registered is an error at construction, never a silent
	// fallback to the builtin pruner.
	Registry *Registry
	// Now supplies the current time. Defaults to time.Now.
	Now func() time.Time
	// Observe, when set, is called with every cycle result as it completes, so
	// a caller can log it and raise a finding on BoundBytes without waiting for
	// the whole cycle.
	Observe func(Result, Spec)
}

// binding pairs a declared budget with the pruner that will enforce it.
type binding struct {
	spec   Spec
	pruner Pruner
}

// Engine enforces a set of declared budgets.
type Engine struct {
	bindings []binding
	now      func() time.Time
	observe  func(Result, Spec)
}

// NewEngine resolves every spec to a pruner and returns the engine that enforces
// them.
//
// Resolution happens once, at construction, so a budget naming an unregistered
// custom pruner fails at startup rather than at the first cycle in the middle of
// the night.
func NewEngine(cfg EngineConfig) (*Engine, error) {
	e := &Engine{
		now:     cfg.Now,
		observe: cfg.Observe,
	}
	if e.now == nil {
		e.now = time.Now
	}

	for _, spec := range cfg.Specs {
		pruner, err := resolvePruner(spec, cfg)
		if err != nil {
			return nil, err
		}
		e.bindings = append(e.bindings, binding{spec: spec, pruner: pruner})
	}
	return e, nil
}

func resolvePruner(spec Spec, cfg EngineConfig) (Pruner, error) {
	switch spec.Mode {
	case PrunerCustom:
		if cfg.Registry == nil {
			return nil, fmt.Errorf("budget %q: %w", spec.Budget.Name, ErrPrunerNotRegistered)
		}
		pruner, ok := cfg.Registry.Lookup(spec.Budget.Name)
		if !ok {
			return nil, fmt.Errorf("budget %q: %w", spec.Budget.Name, ErrPrunerNotRegistered)
		}
		return pruner, nil
	case PrunerBuiltin:
		if cfg.Builtin == nil {
			return nil, fmt.Errorf("budget %q: %w", spec.Budget.Name, ErrNoBuiltinFactory)
		}
		pruner, err := cfg.Builtin(spec)
		if err != nil {
			return nil, fmt.Errorf("budget %q builtin pruner: %w", spec.Budget.Name, err)
		}
		return pruner, nil
	default:
		return nil, fmt.Errorf("budget %q: pruner mode %q is not builtin or custom", spec.Budget.Name, spec.Mode)
	}
}

// Budgets returns the declared budgets this engine enforces, in cycle order.
func (e *Engine) Budgets() []Budget {
	out := make([]Budget, 0, len(e.bindings))
	for _, b := range e.bindings {
		out = append(out, b.spec.Budget)
	}
	return out
}

// Specs returns the parsed declarations this engine enforces, in cycle order.
func (e *Engine) Specs() []Spec {
	out := make([]Spec, 0, len(e.bindings))
	for _, b := range e.bindings {
		out = append(out, b.spec)
	}
	return out
}

// Run enforces every budget once and returns one result per budget attempted.
//
// A cancelled context stops the cycle and returns the results already produced
// with the current budget marked Incomplete, together with the context error.
// Partial progress is reported rather than discarded: on a target that takes
// hours to bring within budget, discarding a cancelled cycle's work would mean
// it never finishes.
func (e *Engine) Run(ctx context.Context) ([]Result, error) {
	results := make([]Result, 0, len(e.bindings))
	for _, b := range e.bindings {
		if err := ctx.Err(); err != nil {
			results = append(results, Result{Budget: b.spec.Budget.Name, BoundBy: BoundNone, Incomplete: true})
			return results, err
		}
		result, err := e.runOne(ctx, b)
		results = append(results, result)
		if e.observe != nil {
			e.observe(result, b.spec)
		}
		if err != nil {
			return results, fmt.Errorf("budget %q: %w", b.spec.Budget.Name, err)
		}
	}
	return results, nil
}

func (e *Engine) runOne(ctx context.Context, b binding) (Result, error) {
	before, err := b.pruner.Measure(ctx)
	if err != nil {
		return Result{Budget: b.spec.Budget.Name, Incomplete: true}, fmt.Errorf("measure: %w", err)
	}

	result, pruneErr := b.pruner.Prune(ctx, b.spec.Budget)
	result.Budget = b.spec.Budget.Name

	// The engine owns the reported measurements. A pruner may fill these with a
	// cheap byte-only reading for its own callers; the engine's are the full
	// Usage, item counts included, so a finding can say how many items remain.
	result.Before = before
	if pruneErr == nil {
		if after, measureErr := b.pruner.Measure(ctx); measureErr == nil {
			result.After = after
		}
	}
	if result.FreedBytes == 0 && result.After.Bytes < result.Before.Bytes {
		result.FreedBytes = result.Before.Bytes - result.After.Bytes
	}
	if pruneErr != nil {
		if errors.Is(pruneErr, context.Canceled) || errors.Is(pruneErr, context.DeadlineExceeded) {
			result.Incomplete = true
		}
		return result, pruneErr
	}

	result.BoundBy = normalizeBound(result, b.spec.Budget)
	return result, nil
}

// normalizeBound settles which bound determined the retained set.
//
// A pruner reports the bound it stopped on, because it is the only thing that
// knows. The engine overrides only in the case the pruner cannot see: the target
// is still over its byte ceiling. That is BoundBytes regardless of what the
// pruner managed to delete, because the size ceiling is what the retained set is
// now being judged against, and reporting it is what surfaces a producer
// outrunning its declared horizon.
func normalizeBound(r Result, b Budget) Bound {
	if r.OverBudget(b) {
		return BoundBytes
	}
	if r.BoundBy != BoundNone {
		return r.BoundBy
	}
	if r.Deleted > 0 && b.HasAgeBound() {
		return BoundAge
	}
	return BoundNone
}
