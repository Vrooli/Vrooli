package ai

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// EventEmitter is the seam Service uses to publish AI usage events.
// Production wires this to the internal/events Logger.
type EventEmitter interface {
	Emit(eventType, sessionID string, details map[string]string)
}

// Event-type constants. Kept here so Service does not import
// internal/events (which would create an upward dependency).
const (
	eventAIGenerate = "ai.generate"
	eventAISuggest  = "ai.suggest"
)

// Service is the canonical AI capability. It runs the provider chain
// against config-driven enable/timeout/health policy and implements the
// handlers/ai.Backend interface so handlers can talk to it directly.
type Service struct {
	chain     *Chain
	configs   ConfigStore
	sysCtx    *SystemContext
	events    EventEmitter
	genCount  *atomic.Int64
	suggCount *atomic.Int64
}

// NewService constructs a Service from its collaborators. genCount and
// suggCount are atomic counters owned by the caller (the metrics aggregator)
// so /metrics can read them without going through this package.
func NewService(chain *Chain, configs ConfigStore, sysCtx *SystemContext, events EventEmitter, genCount, suggCount *atomic.Int64) *Service {
	return &Service{
		chain:     chain,
		configs:   configs,
		sysCtx:    sysCtx,
		events:    events,
		genCount:  genCount,
		suggCount: suggCount,
	}
}

// Chain returns the underlying provider chain. Used by tests and by
// callers that need raw chain access.
func (s *Service) Chain() *Chain { return s.chain }

// Configs returns the underlying config store.
func (s *Service) Configs() ConfigStore { return s.configs }

// SystemContext returns the captured environment context.
func (s *Service) SystemContext() *SystemContext { return s.sysCtx }

// Execute runs the provider chain with the given prompts, respecting
// per-provider enable/timeout config and recording health metrics.
func (s *Service) Execute(ctx context.Context, systemPrompt, userPrompt string) (result, provider string, err error) {
	configs := s.configs.GetConfigs()

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		var p Provider
		for _, cp := range s.chain.Providers() {
			if cp.Name() == cfg.Name {
				p = cp
				break
			}
		}
		if p == nil {
			continue
		}

		timeout := time.Duration(cfg.TimeoutSec) * time.Second
		pCtx, cancel := context.WithTimeout(ctx, timeout)

		start := time.Now()
		res, pErr := p.Generate(pCtx, systemPrompt, userPrompt)
		elapsed := time.Since(start)
		cancel()

		if pErr != nil {
			s.configs.RecordError(cfg.Name)
			err = pErr
			continue
		}

		s.configs.RecordSuccess(cfg.Name, elapsed)
		return res, cfg.Name, nil
	}

	if err != nil {
		return "", "", err
	}
	return "", "", fmt.Errorf("no enabled providers configured")
}

// ExecuteCommand satisfies handlers/ai.Backend.
func (s *Service) ExecuteCommand(ctx context.Context, userPrompt string) (string, string, error) {
	return s.Execute(ctx, BuildCommandSystemPrompt(s.sysCtx), userPrompt)
}

// ExecuteSuggest satisfies handlers/ai.Backend.
func (s *Service) ExecuteSuggest(ctx context.Context, userPrompt string) (string, string, error) {
	return s.Execute(ctx, BuildSuggestSystemPrompt(s.sysCtx), userPrompt)
}

// EmitGenerate satisfies handlers/ai.Backend.
func (s *Service) EmitGenerate(provider, prompt string) {
	if s.events == nil {
		return
	}
	s.events.Emit(eventAIGenerate, "", map[string]string{
		"provider": provider,
		"prompt":   prompt,
	})
}

// EmitSuggest satisfies handlers/ai.Backend.
func (s *Service) EmitSuggest(provider, prompt string, count int) {
	if s.events == nil {
		return
	}
	s.events.Emit(eventAISuggest, "", map[string]string{
		"provider": provider,
		"prompt":   prompt,
		"count":    fmt.Sprintf("%d", count),
	})
}

// IncrGenerations satisfies handlers/ai.Backend.
func (s *Service) IncrGenerations() {
	if s.genCount != nil {
		s.genCount.Add(1)
	}
}

// IncrSuggestions satisfies handlers/ai.Backend.
func (s *Service) IncrSuggestions() {
	if s.suggCount != nil {
		s.suggCount.Add(1)
	}
}

// GetConfigs satisfies handlers/ai.Backend.
func (s *Service) GetConfigs() []Config { return s.configs.GetConfigs() }

// GetHealth satisfies handlers/ai.Backend.
func (s *Service) GetHealth() []Health { return s.configs.GetHealth() }

// UpdateProviderConfig satisfies handlers/ai.Backend.
func (s *Service) UpdateProviderConfig(name string, enabled bool, priority, timeoutSec, maxRetries int) bool {
	return s.configs.UpdateConfig(name, enabled, priority, timeoutSec, maxRetries)
}
