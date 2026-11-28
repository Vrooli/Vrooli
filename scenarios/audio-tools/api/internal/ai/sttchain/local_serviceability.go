package sttchain

import "context"

// serviceabilityObserver is implemented by local providers whose ordinary
// IsAvailable path may reconcile/start their backend. Health and inventory
// reads use this read-only seam instead.
type serviceabilityObserver interface {
	IsServiceable(context.Context) bool
}

// LocalEngineServiceable reports current liveness without causing resource
// lifecycle changes. A list or readiness request must not start a model just
// to describe it.
func (c *Chain) LocalEngineServiceable(ctx context.Context, engineID string) bool {
	if c == nil || !c.enableLocal {
		return false
	}
	provider := c.resolveLocalEngine(engineID)
	if provider == nil {
		return false
	}
	if observed, ok := provider.(serviceabilityObserver); ok {
		return observed.IsServiceable(ctx)
	}
	// Fail closed when an adapter has not implemented the observational
	// contract. Falling back to IsAvailable could turn a picker read into an
	// implicit resource start.
	return false
}
