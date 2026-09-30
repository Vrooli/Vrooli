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
// resolver. The concrete control-plane client owns binary discovery.
type ControlPlaneRun func(context.Context, ...string) ([]byte, error)

// LiveResolver caches control-plane observations so engine selection does not
// shell once per request. A start/stop lifecycle event can invalidate it.
type LiveResolver struct {
	run       ControlPlaneRun
	ttl       time.Duration
	available func(context.Context, string) bool

	mu      sync.Mutex
	checked time.Time
	facts   map[string]EngineFacts
}

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
	r.mu.Unlock()
}

func (r *LiveResolver) Invalidate() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.checked = time.Time{}
	r.facts = nil
	r.mu.Unlock()
}

func (r *LiveResolver) Resolve(ctx context.Context, registry *Registry) (Resolution, error) {
	if registry == nil {
		return Resolution{}, fmt.Errorf("sttengine: registry is nil")
	}
	facts, err := r.observe(ctx, registry)
	resolution, resolveErr := registry.ResolveFacts(facts)
	if resolveErr != nil && err != nil {
		return resolution, err
	}
	return resolution, resolveErr
}

func (r *LiveResolver) observe(ctx context.Context, registry *Registry) (map[string]EngineFacts, error) {
	if r == nil || r.run == nil {
		facts := make(map[string]EngineFacts, len(registry.Engines()))
		for _, engine := range registry.Engines() {
			facts[engine.ID] = EngineFacts{SignalUnavailable: true, Reason: "control-plane unavailable"}
		}
		return facts, fmt.Errorf("sttengine: control-plane unavailable")
	}
	r.mu.Lock()
	if r.facts != nil && time.Since(r.checked) < r.ttl {
		facts := cloneFacts(r.facts)
		r.mu.Unlock()
		return facts, nil
	}
	r.mu.Unlock()

	facts := make(map[string]EngineFacts, len(registry.Engines()))
	runProbe := func(args ...string) ([]byte, error) {
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return r.run(probeCtx, args...)
	}
	var firstErr error
	for _, engine := range registry.Engines() {
		if engine.Kind != KindLocalResource || strings.TrimSpace(engine.Resource) == "" {
			facts[engine.ID] = EngineFacts{PlatformSupported: true, Installed: true, Running: true, Healthy: true, WorkloadCapable: true, Reason: "non-local engine is request-gated"}
			continue
		}
		status, err := runProbe("resource", "status", engine.Resource, "--json")
		if err != nil {
			facts[engine.ID] = EngineFacts{SignalUnavailable: true, Reason: "control-plane status unavailable"}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		var s struct {
			Installed  bool   `json:"installed"`
			Running    bool   `json:"running"`
			Healthy    bool   `json:"healthy"`
			StatusCode string `json:"status_code"`
			Resource   struct {
				ObservedMode string `json:"observed_mode"`
				StatusCode   string `json:"status_code"`
			} `json:"resource"`
		}
		if err := json.Unmarshal(status, &s); err != nil {
			facts[engine.ID] = EngineFacts{SignalUnavailable: true, Reason: "control-plane status malformed"}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if r.available != nil && !r.available(ctx, engine.ID) {
			s.Running = false
			s.Healthy = false
		}
		accelerated := strings.TrimSpace(s.Resource.ObservedMode) != "" && s.Resource.ObservedMode != "cpu"
		signalUnavailable := false
		accel, accelErr := runProbe("resource", "acceleration", "explain", engine.Resource, "--json")
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
		facts[engine.ID] = EngineFacts{
			PlatformSupported: statusCode != "unsupported_platform",
			Installed:         s.Installed, Running: s.Running, Healthy: s.Healthy,
			WorkloadCapable:   len(engine.Strategies) > 0,
			Accelerated:       accelerated,
			SignalUnavailable: signalUnavailable,
			Reason:            fmt.Sprintf("control-plane resource status: status_code=%s observed_mode=%s", statusCode, s.Resource.ObservedMode),
		}
	}
	r.mu.Lock()
	r.facts = cloneFacts(facts)
	r.checked = time.Now()
	r.mu.Unlock()
	return facts, firstErr
}

func cloneFacts(in map[string]EngineFacts) map[string]EngineFacts {
	out := make(map[string]EngineFacts, len(in))
	for id, facts := range in {
		out[id] = facts
	}
	return out
}
