package modelregistry

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

// ModelHealthStatus is the public status label for a single (runner, modelID) entry.
type ModelHealthStatus string

const (
	// ModelHealthUnknown means the model has not been probed and has no runtime signal.
	ModelHealthUnknown ModelHealthStatus = "unknown"
	// ModelHealthOK means the last observation (probe or runtime execution) succeeded.
	ModelHealthOK ModelHealthStatus = "ok"
	// ModelHealthFailed means the most recent observation classified the model as unavailable.
	ModelHealthFailed ModelHealthStatus = "failed"
)

// ModelHealth is a point-in-time snapshot of a single model's health.
type ModelHealth struct {
	Status      ModelHealthStatus `json:"status"`
	LastChecked time.Time         `json:"lastChecked"`
	Message     string            `json:"message,omitempty"`
}

// HealthSnapshot is the JSON-shaped payload returned by the handler.
type HealthSnapshot struct {
	Runners map[string]map[string]ModelHealth `json:"runners"`
}

// healthKey composes a (runner, modelID) key for the internal map.
type healthKey struct {
	runner  string
	modelID string
}

// HealthStore holds the in-memory health map. Not persisted: on restart, the
// probe ticker repopulates it. Marking is thread-safe.
type HealthStore struct {
	mu      sync.RWMutex
	entries map[healthKey]ModelHealth
	// runnerTypes lists the runner keys we know about (from the registry), used when
	// producing snapshots so runners with no entries still appear as an empty map.
	runnerTypes []string
}

// NewHealthStore returns an empty store.
func NewHealthStore() *HealthStore {
	return &HealthStore{entries: make(map[healthKey]ModelHealth)}
}

// RegisterRunners seeds the list of runner keys whose models are being tracked.
// Called once at startup by the orchestrator.
func (s *HealthStore) RegisterRunners(runners []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := make(map[string]struct{}, len(runners))
	out := make([]string, 0, len(runners))
	for _, r := range runners {
		trimmed := strings.TrimSpace(r)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	s.runnerTypes = out
}

// Mark updates the health of a single model. An empty modelID is accepted but
// stored under the empty key — it represents the runner-default sentinel.
func (s *HealthStore) Mark(runnerType string, modelID string, status ModelHealthStatus, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[healthKey{runner: runnerType, modelID: modelID}] = ModelHealth{
		Status:      status,
		LastChecked: time.Now().UTC(),
		Message:     strings.TrimSpace(message),
	}
}

// Snapshot returns a copy of the current health map, keyed by runner then modelID.
// Runners that were registered but have no entries appear with an empty inner map,
// so consumers always see the same top-level shape.
func (s *HealthStore) Snapshot() HealthSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := HealthSnapshot{Runners: make(map[string]map[string]ModelHealth, len(s.runnerTypes))}
	for _, r := range s.runnerTypes {
		out.Runners[r] = make(map[string]ModelHealth)
	}
	for k, v := range s.entries {
		if _, ok := out.Runners[k.runner]; !ok {
			out.Runners[k.runner] = make(map[string]ModelHealth)
		}
		out.Runners[k.runner][k.modelID] = v
	}
	return out
}

// ModelProber is the subset of runner.Runner used by the probe. Keeping the
// dependency on a small local interface means this package does not import the
// runner adapter package, avoiding cycles.
type ModelProber interface {
	ProbeModel(ctx context.Context, modelID string) error
}

// RunnerProberLookup resolves a runner key to its prober, or returns nil when
// the runner is not registered.
type RunnerProberLookup func(runnerType string) ModelProber

// ProbeConfig configures the periodic health probe loop.
type ProbeConfig struct {
	// Interval between full probe sweeps. Zero or negative disables the ticker
	// (the startup probe still runs once).
	Interval time.Duration
}

// DefaultProbeConfig returns the canonical defaults: sweep every 30 minutes.
func DefaultProbeConfig() ProbeConfig {
	return ProbeConfig{Interval: 30 * time.Minute}
}

// HealthProbe drives the periodic probe loop. It is passive: it reads the current
// registry and invokes the supplied prober for each (runner, modelID) pair.
type HealthProbe struct {
	registry *Store
	health   *HealthStore
	resolve  RunnerProberLookup
	config   ProbeConfig
}

// NewHealthProbe wires a probe from its dependencies.
func NewHealthProbe(registry *Store, health *HealthStore, resolve RunnerProberLookup, config ProbeConfig) *HealthProbe {
	return &HealthProbe{
		registry: registry,
		health:   health,
		resolve:  resolve,
		config:   config,
	}
}

// RunOnce performs a single probe sweep over every model in the registry.
// It is safe to call from multiple goroutines; the health store is the only
// shared mutable state and it serializes writes internally.
func (p *HealthProbe) RunOnce(ctx context.Context) {
	if p.registry == nil || p.health == nil || p.resolve == nil {
		return
	}
	snapshot := p.registry.Get()
	if snapshot == nil {
		return
	}
	for runnerKey, runnerCfg := range snapshot.Runners {
		prober := p.resolve(runnerKey)
		if prober == nil {
			// Runner not registered in this process; leave its entries untouched
			// so a restart doesn't flap their status.
			continue
		}
		for _, model := range runnerCfg.Models {
			id := strings.TrimSpace(model.ID)
			if id == "" {
				continue
			}
			if err := prober.ProbeModel(ctx, id); err != nil {
				p.health.Mark(runnerKey, id, ModelHealthFailed, err.Error())
				continue
			}
			p.health.Mark(runnerKey, id, ModelHealthOK, "")
		}
	}
}

// Start launches the probe in a goroutine. It runs once immediately, then on
// the configured interval. Cancel the context to stop the loop.
func (p *HealthProbe) Start(ctx context.Context) {
	go func() {
		p.RunOnce(ctx)
		if p.config.Interval <= 0 {
			return
		}
		ticker := time.NewTicker(p.config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.RunOnce(ctx)
			}
		}
	}()
	log.Printf("[model-health] probe started (interval=%v)", p.config.Interval)
}
