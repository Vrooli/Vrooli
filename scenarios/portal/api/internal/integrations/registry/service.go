package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/shared"

	"portal/internal/clock"
)

type definition struct {
	id          IntegrationID
	displayName string
	required    bool
	probe       Probe
}

type Service struct {
	clock clock.Clock
	store *Store

	mu            sync.Mutex
	lastMode      sharedv1.BehaviorMode
	overrideValue Override
	windows       map[IntegrationID]*window
	defs          []definition
}

type Config struct {
	Clock      clock.Clock
	Store      *Store
	Probes     map[IntegrationID]Probe
	WindowSize int
}

func NewService(cfg Config) *Service {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.System{}
	}
	probes := cfg.Probes
	if probes == nil {
		probes = DefaultProbes()
	}
	defs := []definition{
		{id: IntegrationSearchHub, displayName: "search-hub", probe: probes[IntegrationSearchHub]},
		{id: IntegrationOpenRouter, displayName: "OpenRouter", probe: probes[IntegrationOpenRouter]},
		{id: IntegrationAgentManager, displayName: "agent-manager", probe: probes[IntegrationAgentManager]},
		{id: IntegrationPromptMgr, displayName: "prompt-manager", probe: probes[IntegrationPromptMgr]},
	}
	windows := make(map[IntegrationID]*window, len(defs))
	for _, def := range defs {
		windows[def.id] = newWindow(cfg.WindowSize)
	}
	return &Service{
		clock:         clk,
		store:         cfg.Store,
		lastMode:      sharedv1.BehaviorMode_BEHAVIOR_MODE_OFF,
		overrideValue: OverrideAuto,
		windows:       windows,
		defs:          defs,
	}
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	override, err := s.loadOverride(ctx)
	if err != nil {
		return Status{}, err
	}
	statuses := make([]IntegrationStatus, 0, len(s.defs))
	now := s.clock.Now().UTC()
	for _, def := range s.defs {
		statuses = append(statuses, s.probe(ctx, def, now))
	}
	search := findStatus(statuses, IntegrationSearchHub)
	s.mu.Lock()
	decision := DecideMode(PolicyInput{
		PreviousMode: s.lastMode,
		Override:     override,
		Search:       search,
	})
	s.lastMode = decision.Mode
	s.mu.Unlock()
	return Status{
		Integrations: statuses,
		ActiveMode:   decision.Mode,
		Override:     override,
		Reason:       decision.Reason,
		EvaluatedAt:  now,
	}, nil
}

func (s *Service) SetOverride(ctx context.Context, override Override) (Status, error) {
	value := normalizeOverride(string(override))
	if s.store != nil {
		if err := s.store.SetOverride(ctx, value, s.clock.Now()); err != nil {
			return Status{}, err
		}
	}
	s.mu.Lock()
	s.overrideValue = value
	s.mu.Unlock()
	return s.Status(ctx)
}

func (s *Service) Observe(id IntegrationID, latency time.Duration, ok bool, degraded bool, reason string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	w := s.windows[id]
	s.mu.Unlock()
	if w == nil {
		return
	}
	w.add(Sample{
		At:       s.clock.Now().UTC(),
		Latency:  latency,
		OK:       ok,
		Degraded: degraded,
		Reason:   reason,
	})
}

func (s *Service) loadOverride(ctx context.Context) (Override, error) {
	if s.store == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.overrideValue, nil
	}
	return s.store.Override(ctx)
}

func (s *Service) probe(ctx context.Context, def definition, now time.Time) IntegrationStatus {
	start := s.clock.Now()
	result := ProbeResult{OK: false, Reason: "probe is not configured"}
	if def.probe != nil {
		probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		result = def.probe.Probe(probeCtx)
		cancel()
	}
	latency := s.clock.Now().Sub(start)
	if latency < 0 {
		latency = 0
	}
	stats := s.windows[def.id].add(Sample{
		At:       now,
		Latency:  latency,
		OK:       result.OK,
		Degraded: result.Degraded,
		Reason:   result.Reason,
	})
	state := sharedv1.IntegrationState_INTEGRATION_STATE_AVAILABLE
	switch {
	case !result.OK:
		state = sharedv1.IntegrationState_INTEGRATION_STATE_UNAVAILABLE
	case result.Degraded:
		state = sharedv1.IntegrationState_INTEGRATION_STATE_DEGRADED
	}
	reason := result.Reason
	if reason == "" {
		reason = fmt.Sprintf("%s is available", def.displayName)
	}
	return IntegrationStatus{
		ID:          def.id,
		DisplayName: def.displayName,
		State:       state,
		Stats:       stats,
		Reason:      reason,
		Required:    def.required,
	}
}

func findStatus(statuses []IntegrationStatus, id IntegrationID) IntegrationStatus {
	for _, status := range statuses {
		if status.ID == id {
			return status
		}
	}
	return IntegrationStatus{
		ID:     id,
		State:  sharedv1.IntegrationState_INTEGRATION_STATE_UNAVAILABLE,
		Reason: "integration status is missing",
	}
}
