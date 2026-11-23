package inference

import (
	"context"
	"sync"
	"time"

	"ai-gateway/internal/providers"
)

// routeTrace collects what the provider walk did for one typed request so its
// route event records the candidate that ran and the time the provider took.
// The walk may try several candidates and, for embeddings, several batches;
// the trace keeps the last candidate tried and sums the time across calls.
type routeTrace struct {
	mu           sync.Mutex
	attempts     int
	candidates   map[string]struct{}
	provider     string
	resourceRole string
	model        string
	locality     string
	executionNS  int64
	runtimeNS    int64
	admissionNS  int64
	runtimeKnown bool
}

type routeTraceKey struct{}

func withRouteTrace(ctx context.Context) (context.Context, *routeTrace) {
	trace := &routeTrace{candidates: map[string]struct{}{}}
	return context.WithValue(ctx, routeTraceKey{}, trace), trace
}

// executeTraced runs one provider call and adds it to the request's trace.
// Every typed path that dispatches inference goes through it, so no provider
// call escapes the route record.
func executeTraced(ctx context.Context, adapter providers.Adapter, resolved providers.ResolvedRole, request providers.ExecutionRequest) (providers.ExecutionResult, error) {
	started := time.Now()
	execution, err := adapter.Execute(ctx, request)
	if trace, _ := ctx.Value(routeTraceKey{}).(*routeTrace); trace != nil {
		trace.attempt(adapter.Provider, request.Role, resolved, time.Since(started), execution.Timing)
	}
	return execution, err
}

func (t *routeTrace) attempt(provider, resourceRole string, resolved providers.ResolvedRole, took time.Duration, timing providers.ProviderTiming) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.attempts++
	t.candidates[provider+"/"+resourceRole] = struct{}{}
	t.provider, t.resourceRole = provider, resourceRole
	t.model, t.locality = resolved.Model, resolved.Locality
	t.executionNS += max(int64(took), 0)
	if timing.RuntimeNS > 0 {
		t.runtimeKnown = true
		t.runtimeNS += timing.RuntimeNS
		t.admissionNS += max(timing.AdmissionWaitNS, 0)
	}
}

// routeTraceSnapshot is a trace's settled values.
type routeTraceSnapshot struct {
	attempts, candidates                    int
	provider, resourceRole, model, locality string
	executionNS, runtimeNS, admissionNS     int64
	runtimeKnown                            bool
}

func (t *routeTrace) snapshot() routeTraceSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return routeTraceSnapshot{
		attempts: t.attempts, candidates: len(t.candidates),
		provider: t.provider, resourceRole: t.resourceRole, model: t.model, locality: t.locality,
		executionNS: t.executionNS, runtimeNS: t.runtimeNS, admissionNS: t.admissionNS, runtimeKnown: t.runtimeKnown,
	}
}
