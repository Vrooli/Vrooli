package health

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/fallback"
)

// ModelProber is the subset of runner.Runner used by the periodic probe.
// Kept as a small local interface so this package does not import the
// runner adapter package, avoiding cycles.
type ModelProber interface {
	ProbeModel(ctx context.Context, modelID string) error
}

// RunnerProberLookup resolves a runner key to its prober, or returns nil
// when the runner is not registered in this process.
type RunnerProberLookup func(runnerType string) ModelProber

// RegistrySnapshot is the minimal immutable catalog view the probe needs.
type RegistrySnapshot interface {
	// Get returns the registry snapshot's iterable contents. Each entry
	// pairs a runner_type key with the list of model IDs configured for it.
	IterModels(yield func(runnerType string, modelIDs []string) bool)
}

// ProbeConfig configures the periodic probe loop. Interval<=0 runs the
// startup sweep only.
type ProbeConfig struct {
	Interval time.Duration

	// Retention controls audit-row eviction. <=0 disables eviction.
	Retention time.Duration

	// EvictionInterval controls how often EvictByRetention runs in the
	// background. Defaults to 24h when zero.
	EvictionInterval time.Duration
}

// DefaultProbeConfig returns the canonical defaults: probe every 30
// minutes, evict audit rows older than 90 days every 24 hours.
func DefaultProbeConfig() ProbeConfig {
	return ProbeConfig{
		Interval:         30 * time.Minute,
		Retention:        DefaultRetention,
		EvictionInterval: 24 * time.Hour,
	}
}

// Probe drives the periodic health-probe loop. It writes audit rows on
// every probe outcome (ok or failed) so the SQLite-backed Store has a
// continuous record across restarts.
//
// The probe:
//   - writes to the persisted Store
//   - emits typed observations (Status + fallback.Reason) instead of
//     freeform message strings
//   - probes runners themselves via runner availability checks (the
//     model probe success; this is explicit)
type Probe struct {
	store    *Store
	resolve  RunnerProberLookup
	registry RegistrySnapshot
	classify fallback.Classifier
	config   ProbeConfig

	once  sync.Once
	cease context.CancelFunc
}

// NewProbe wires a probe.
func NewProbe(store *Store, registry RegistrySnapshot, resolve RunnerProberLookup, classify fallback.Classifier, config ProbeConfig) *Probe {
	if classify == nil {
		classify = fallback.NewTextClassifier()
	}
	return &Probe{
		store:    store,
		registry: registry,
		resolve:  resolve,
		classify: classify,
		config:   config,
	}
}

// RunOnce performs a single probe sweep. Safe to call concurrently with
// itself; the Store serialises writes internally.
func (p *Probe) RunOnce(ctx context.Context) {
	if p == nil || p.store == nil || p.registry == nil || p.resolve == nil {
		return
	}
	p.registry.IterModels(func(runnerKey string, models []string) bool {
		prober := p.resolve(runnerKey)
		if prober == nil {
			// Runner not registered in this process; skip rather than
			// flapping its status across restarts.
			return true
		}
		for _, modelID := range models {
			id := strings.TrimSpace(modelID)
			if id == "" {
				continue
			}
			err := prober.ProbeModel(ctx, id)
			if err == nil {
				if recErr := p.store.RecordModel(ctx, runnerKey, id, StatusOK, "", "", "probe"); recErr != nil {
					slog.Warn("health.probe: record ok failed", "err", recErr, "runner", runnerKey, "model", id)
				}
				continue
			}
			ce := p.classify.Classify(fallback.ClassifyInput{
				RunnerType: runnerKey,
				Cause:      err,
				Stderr:     err.Error(),
			})
			reason, message := fallback.ReasonUnknown, err.Error()
			if ce != nil {
				reason = ce.Reason
				message = ce.Message
			}
			if recErr := p.store.RecordModel(ctx, runnerKey, id, StatusFailed, string(reason), message, "probe"); recErr != nil {
				slog.Warn("health.probe: record failed failed", "err", recErr, "runner", runnerKey, "model", id)
			}
		}
		return true
	})
}

// Start launches the probe and the eviction loop in goroutines. Cancel
// the context to stop both. Safe to call multiple times — only the first
// call actually starts the goroutines.
func (p *Probe) Start(ctx context.Context) {
	if p == nil {
		return
	}
	p.once.Do(func() {
		runCtx, cancel := context.WithCancel(ctx)
		p.cease = cancel

		go p.runProbeLoop(runCtx)
		if p.config.Retention > 0 {
			go p.runEvictionLoop(runCtx)
		}
	})
}

// Stop cancels the probe and eviction loops if they were started.
func (p *Probe) Stop() {
	if p == nil || p.cease == nil {
		return
	}
	p.cease()
}

func (p *Probe) runProbeLoop(ctx context.Context) {
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
}

func (p *Probe) runEvictionLoop(ctx context.Context) {
	interval := p.config.EvictionInterval
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	// Prime: evict on startup so a long-stopped server doesn't keep
	// stale audit rows around any longer than necessary.
	if _, err := p.store.EvictByRetention(ctx, p.config.Retention); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("health.probe: prime eviction failed", "err", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := p.store.EvictByRetention(ctx, p.config.Retention); err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn("health.probe: eviction failed", "err", err)
			}
		}
	}
}
