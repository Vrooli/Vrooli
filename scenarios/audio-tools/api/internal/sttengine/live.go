package sttengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ControlPlaneRun is the narrow command seam used by the bounded live
// resolver. The concrete control-plane client owns binary discovery. The
// resolver may invoke it concurrently for independent inventory probes.
type ControlPlaneRun func(context.Context, ...string) ([]byte, error)

// LiveResolver caches control-plane observations so engine selection does not
// shell once per request. A start/stop lifecycle event can invalidate it.
type LiveResolver struct {
	run       ControlPlaneRun
	ttl       time.Duration
	available func(context.Context, string) bool

	mu         sync.Mutex
	checked    time.Time
	facts      map[string]EngineFacts
	refreshing bool
	revision   uint64
}

const (
	staleInventoryMaxAge    = 2
	staleLivenessBudget     = 500 * time.Millisecond
	backgroundRefreshBudget = 11 * time.Second
)

func NewLiveResolver(run ControlPlaneRun, ttl time.Duration) *LiveResolver {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &LiveResolver{run: run, ttl: ttl}
}

// SetAvailability adds the provider-owned liveness signal to the control-plane
// facts. It is optional so registry tests remain independent of live services.
func (r *LiveResolver) SetAvailability(fn func(context.Context, string) bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.available = fn
	r.checked = time.Time{}
	r.facts = nil
	r.revision++
	r.mu.Unlock()
}

func (r *LiveResolver) Invalidate() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.checked = time.Time{}
	r.facts = nil
	r.revision++
	r.mu.Unlock()
}

func (r *LiveResolver) Resolve(ctx context.Context, registry *Registry) (Resolution, error) {
	return r.resolve(ctx, registry, true)
}

// ResolveForAdmission never waits for a cold control-plane inventory read.
// It admits only from a fresh snapshot or a bounded-liveness-validated stale
// snapshot; otherwise it starts a coalesced background refresh and fails
// closed until host support and installation facts are known.
func (r *LiveResolver) ResolveForAdmission(ctx context.Context, registry *Registry) (Resolution, error) {
	return r.resolve(ctx, registry, false)
}

func (r *LiveResolver) resolve(ctx context.Context, registry *Registry, allowColdProbe bool) (Resolution, error) {
	if registry == nil {
		return Resolution{}, fmt.Errorf("sttengine: registry is nil")
	}
	if ctx == nil {
		return Resolution{}, fmt.Errorf("sttengine: request context is nil")
	}
	if r == nil {
		if !allowColdProbe {
			return registry.ResolveFacts(expiredInventoryFacts(registry))
		}
		facts, err := (*LiveResolver)(nil).observe(ctx, registry)
		resolution, resolveErr := registry.ResolveFacts(facts)
		if resolveErr != nil && err != nil {
			return resolution, err
		}
		return resolution, resolveErr
	}

	now := time.Now()
	r.mu.Lock()
	cached := cloneFacts(r.facts)
	age := now.Sub(r.checked)
	revision := r.revision
	fresh := cached != nil && age < r.ttl
	canUseStale := cached != nil && !fresh && age <= time.Duration(staleInventoryMaxAge)*r.ttl && r.available != nil
	r.mu.Unlock()

	var facts map[string]EngineFacts
	var err error
	switch {
	case fresh:
		facts = r.validateCached(ctx, registry, cached)
	case canUseStale:
		r.refreshAsync(ctx, registry)
		facts = r.validateCached(ctx, registry, cached)
	case cached != nil:
		// An over-age snapshot cannot prove that host policy and resource
		// installation facts still apply. Do not turn that uncertainty into a
		// synchronous control-plane wait (or admit from stale facts): refresh
		// asynchronously and fail closed until a fresh snapshot is published.
		r.refreshAsync(ctx, registry)
		facts = expiredInventoryFacts(registry)
	default:
		if !allowColdProbe {
			r.refreshAsync(ctx, registry)
			facts = expiredInventoryFacts(registry)
			break
		}
		facts, err = r.observe(ctx, registry)
		if err == nil {
			r.mu.Lock()
			if revision == r.revision {
				r.facts = cloneFacts(facts)
				r.checked = time.Now()
			}
			r.mu.Unlock()
		}
	}
	resolution, resolveErr := registry.ResolveFacts(facts)
	if resolveErr != nil && err != nil {
		return resolution, err
	}
	return resolution, resolveErr
}

func expiredInventoryFacts(registry *Registry) map[string]EngineFacts {
	facts := make(map[string]EngineFacts, len(registry.Engines()))
	for _, engine := range registry.Engines() {
		if engine.Kind == KindLocalResource {
			facts[engine.ID] = EngineFacts{
				ServiceabilityObserved: true,
				Serviceable:            false,
				SignalUnavailable:      true,
				Reason:                 "control-plane inventory unavailable or exceeded freshness bound",
			}
			continue
		}
		facts[engine.ID] = EngineFacts{PlatformSupported: true, Installed: true, Running: true, Healthy: true, WorkloadCapable: true, Reason: "non-local engine is request-gated"}
	}
	return facts
}

// validateCached refreshes the fast provider-owned liveness signal for cached
// inventory. The control plane remains authoritative for host support and
// installation facts; provider liveness prevents a stopped model from being
// admitted from a still-fresh snapshot.
func (r *LiveResolver) validateCached(ctx context.Context, registry *Registry, cached map[string]EngineFacts) map[string]EngineFacts {
	available := r.currentAvailability()
	if available == nil {
		return cached
	}
	type liveness struct {
		id          string
		serviceable bool
	}
	results := make(chan liveness, len(registry.Engines()))
	var wg sync.WaitGroup
	for _, engine := range registry.Engines() {
		if engine.Kind != KindLocalResource {
			continue
		}
		engine := engine
		wg.Add(1)
		go func() {
			defer wg.Done()
			probeCtx, cancel := context.WithTimeout(ctx, staleLivenessBudget)
			defer cancel()
			results <- liveness{id: engine.ID, serviceable: available(probeCtx, engine.ID)}
		}()
	}
	wg.Wait()
	close(results)
	validated := cloneFacts(cached)
	for result := range results {
		facts := validated[result.id]
		facts.ServiceabilityObserved = true
		facts.Serviceable = result.serviceable
		facts.Running = result.serviceable
		facts.Healthy = result.serviceable
		if !result.serviceable {
			facts.Reason = "provider liveness unavailable; cached inventory not admitted"
		}
		validated[result.id] = facts
	}
	return validated
}

// refreshAsync coalesces expired-inventory refreshes. A bounded control-plane
// probe can therefore update the shared snapshot without blocking a resident
// model's next interactive admission.
func (r *LiveResolver) refreshAsync(parent context.Context, registry *Registry) {
	if parent == nil || parent.Err() != nil {
		return
	}
	r.mu.Lock()
	if r.refreshing {
		r.mu.Unlock()
		return
	}
	r.refreshing = true
	revision := r.revision
	r.mu.Unlock()
	go func() {
		defer func() {
			r.mu.Lock()
			r.refreshing = false
			r.mu.Unlock()
		}()
		// Refresh must outlive the request that triggered it, but retain its
		// tracing and diagnostic values and enforce a strict upper bound.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), backgroundRefreshBudget)
		defer cancel()
		facts, err := r.observe(ctx, registry)
		if err != nil {
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		if revision != r.revision {
			return
		}
		r.facts = cloneFacts(facts)
		r.checked = time.Now()
	}()
}

func (r *LiveResolver) observe(ctx context.Context, registry *Registry) (map[string]EngineFacts, error) {
	if r == nil || r.run == nil {
		facts := make(map[string]EngineFacts, len(registry.Engines()))
		for _, engine := range registry.Engines() {
			facts[engine.ID] = EngineFacts{SignalUnavailable: true, Reason: "control-plane unavailable"}
		}
		return facts, fmt.Errorf("sttengine: control-plane unavailable")
	}
	facts := make(map[string]EngineFacts, len(registry.Engines()))
	runProbe := func(args ...string) ([]byte, error) {
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return r.run(probeCtx, args...)
	}
	// Resource status and acceleration probes are independent. Run them
	// concurrently under the same per-probe bound so a cold engine inventory
	// costs at most one probe budget per engine.
	type observation struct {
		id    string
		facts EngineFacts
		err   error
	}
	observations := make(chan observation, len(registry.Engines()))
	var wg sync.WaitGroup
	for _, engine := range registry.Engines() {
		if engine.Kind != KindLocalResource || strings.TrimSpace(engine.Resource) == "" {
			facts[engine.ID] = EngineFacts{PlatformSupported: true, Installed: true, Running: true, Healthy: true, WorkloadCapable: true, Reason: "non-local engine is request-gated"}
			continue
		}
		engine := engine
		wg.Add(1)
		go func() {
			defer wg.Done()
			type probeResult struct {
				body []byte
				err  error
			}
			statusResult := make(chan probeResult, 1)
			accelerationResult := make(chan probeResult, 1)
			go func() {
				body, err := runProbe("resource", "status", engine.Resource, "--json")
				statusResult <- probeResult{body: body, err: err}
			}()
			go func() {
				body, err := runProbe("resource", "acceleration", "explain", engine.Resource, "--json")
				accelerationResult <- probeResult{body: body, err: err}
			}()
			statusProbe := <-statusResult
			accelerationProbe := <-accelerationResult
			if statusProbe.err != nil {
				observations <- observation{id: engine.ID, facts: EngineFacts{SignalUnavailable: true, Reason: "control-plane status unavailable"}, err: statusProbe.err}
				return
			}
			var s struct {
				Installed  bool   `json:"installed"`
				Running    bool   `json:"running"`
				Healthy    bool   `json:"healthy"`
				Status     string `json:"status"`
				StatusCode string `json:"status_code"`
				Resource   struct {
					ObservedMode string `json:"observed_mode"`
					StatusCode   string `json:"status_code"`
				} `json:"resource"`
			}
			if err := json.Unmarshal(statusProbe.body, &s); err != nil {
				observations <- observation{id: engine.ID, facts: EngineFacts{SignalUnavailable: true, Reason: "control-plane status malformed"}, err: err}
				return
			}
			// Attach-only mode: the provider's own health probe is the
			// serviceability truth when lifecycle ownership is external. A
			// manually started (or externally managed) resource can be healthy
			// even though the control plane never installed or started it; a
			// control-plane "unsupported" platform verdict is about the managed
			// artifact, not about a server the operator deliberately attached.
			attached := false
			available := r.currentAvailability()
			if available != nil {
				livenessCtx, cancel := context.WithTimeout(ctx, staleLivenessBudget)
				serviceable := available(livenessCtx, engine.ID)
				cancel()
				if serviceable {
					if !s.Installed || !s.Running || !s.Healthy {
						s.Installed, s.Running, s.Healthy = true, true, true
						attached = true
					}
				} else {
					s.Running = false
					s.Healthy = false
				}
			}
			accelerated := strings.TrimSpace(s.Resource.ObservedMode) != "" && s.Resource.ObservedMode != "cpu"
			signalUnavailable := false
			accel, accelErr := accelerationProbe.body, accelerationProbe.err
			if accelErr == nil {
				var a struct {
					Selected  string `json:"selected"`
					Placement struct {
						State string `json:"state"`
					} `json:"placement"`
				}
				if json.Unmarshal(accel, &a) == nil {
					accelerated = a.Placement.State == "ok" && a.Selected != "" && a.Selected != "cpu"
				} else {
					signalUnavailable = true
				}
			} else {
				signalUnavailable = true
			}
			statusCode := s.StatusCode
			if statusCode == "" {
				statusCode = s.Resource.StatusCode
			}
			// Resource status has existed in both the lifecycle envelope (`status`)
			// and the capability envelope (`status_code`). Treat a healthy lifecycle
			// result as the same positive platform signal when an older control-plane
			// shape omits both status-code fields; do not infer support from presence
			// alone or override an explicit unsupported_platform result.
			if statusCode == "" && (strings.EqualFold(s.Status, "healthy") || strings.EqualFold(s.Status, "ok")) {
				statusCode = "ok"
			}
			platformSupported := statusCode != "unsupported_platform"
			reason := fmt.Sprintf("control-plane resource status: status_code=%s observed_mode=%s", statusCode, s.Resource.ObservedMode)
			if attached {
				platformSupported = true
				reason = "attached: provider health confirmed; lifecycle ownership is external (" + reason + ")"
			}
			observations <- observation{id: engine.ID, facts: EngineFacts{
				PlatformSupported: platformSupported,
				Installed:         s.Installed, Running: s.Running, Healthy: s.Healthy,
				WorkloadCapable:   len(engine.Strategies) > 0,
				Accelerated:       accelerated,
				Attached:          attached,
				SignalUnavailable: signalUnavailable,
				Reason:            reason,
			}}
		}()
	}
	wg.Wait()
	close(observations)
	var firstErr error
	for result := range observations {
		facts[result.id] = result.facts
		if firstErr == nil && result.err != nil {
			firstErr = result.err
		}
	}
	return facts, firstErr
}

func cloneFacts(in map[string]EngineFacts) map[string]EngineFacts {
	if in == nil {
		return nil
	}
	out := make(map[string]EngineFacts, len(in))
	for id, facts := range in {
		out[id] = facts
	}
	return out
}

func (r *LiveResolver) currentAvailability() func(context.Context, string) bool {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.available
}
