package agentactivity

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/dispatch"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

var errNotFound = errors.New("agent activity not found")

type rawAgentService interface {
	IsEnabled() bool
	IsAvailable(ctx context.Context) bool
	ResolveURL(ctx context.Context) (string, error)
	GetProfileID() string
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
	StopRun(ctx context.Context, runID string) error
}

type sessionSpawner interface {
	SpawnSession(ctx context.Context, req agentmanager.SessionSpawnRequest) (agentmanager.RunResult, error)
}

type runContinuer interface {
	ContinueRun(ctx context.Context, runID string, message string) error
}

type runDiffer interface {
	GetRunDiff(ctx context.Context, runID string) (agentmanager.RunDiff, error)
}

type runEventReader interface {
	GetRunEvents(ctx context.Context, runID string, opts agentmanager.RunEventsOptions) ([]*domainpb.RunEvent, bool, error)
}

type ServiceConfig struct {
	StorePath    string
	AgentService rawAgentService
	// LanePolicy returns the concurrency cap for each phase-kind lane. If
	// nil, the service spawns without lane gating (legacy / test fallback);
	// production wiring always supplies one.
	LanePolicy LanePolicy
}

// Service projects Agent Manager Run activity for human-led sessions. Structured
// backlog, capture, execution, review, and milestone work uses declared
// workflows and must never acquire a raw Run continuation through this seam.
type Service struct {
	store           Store
	agentService    rawAgentService
	continuer       runContinuer
	differ          runDiffer
	eventReader     runEventReader
	sessionSpawner  sessionSpawner
	lanePolicy      LanePolicy
	eventDispatcher dispatch.NodeDispatcher
	mu              sync.Mutex
}

func NewService(cfg ServiceConfig) *Service {
	svc := &Service{
		store:        NewStore(cfg.StorePath),
		agentService: cfg.AgentService,
		lanePolicy:   cfg.LanePolicy,
	}
	if continuer, ok := cfg.AgentService.(runContinuer); ok {
		svc.continuer = continuer
	}
	if differ, ok := cfg.AgentService.(runDiffer); ok {
		svc.differ = differ
	}
	if reader, ok := cfg.AgentService.(runEventReader); ok {
		svc.eventReader = reader
	}
	if spawner, ok := cfg.AgentService.(sessionSpawner); ok {
		svc.sessionSpawner = spawner
	}
	return svc
}

func (s *Service) SetEventDispatcher(d dispatch.NodeDispatcher) {
	s.eventDispatcher = d
}

// SetLanePolicy wires the lane policy after construction. Useful when
// settings (the canonical LanePolicy implementation) is constructed after
// the activity service in the bootstrap order.
func (s *Service) SetLanePolicy(p LanePolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lanePolicy = p
}

// LaneActiveCounts returns active-record counts per canonical lane. This
// satisfies the execution.ActivityLaneReader seam so GovernanceStatus can
// render the four-lane utilization view without execution importing the
// agentactivity store directly.
func (s *Service) LaneActiveCounts() (map[Lane]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	return LaneActiveCounts(records), nil
}

// checkLaneCapacityLocked returns ErrLaneSaturated when the lane derived
// from spec is at or above the policy cap. records must be the current
// store snapshot (callers already hold s.mu and have just loaded). On any
// derivation error (unrecognized purpose / phaseKind), the spawn is
// rejected — that path indicates a wiring bug that should be loud.
func (s *Service) checkLaneCapacityLocked(spec Spec, records []Record) error {
	if s.lanePolicy == nil {
		return nil
	}
	lane, err := LaneOf(spec.Purpose, spec.PhaseKind)
	if err != nil {
		return err
	}
	limit := s.lanePolicy.LimitFor(lane)
	if limit <= 0 {
		return nil
	}
	if LaneActiveCount(records, lane) >= limit {
		return fmt.Errorf("%w: lane=%s purpose=%s phase_kind=%s", ErrLaneSaturated, lane, spec.Purpose, spec.PhaseKind)
	}
	return nil
}

func (s *Service) IsEnabled() bool {
	return s.agentService != nil && s.agentService.IsEnabled()
}

func (s *Service) IsAvailable(ctx context.Context) bool {
	if s.agentService == nil {
		return false
	}
	return s.agentService.IsAvailable(ctx)
}

func (s *Service) ResolveURL(ctx context.Context) (string, error) {
	if s.agentService == nil {
		return "", agentmanager.ErrNotAvailable
	}
	return s.agentService.ResolveURL(ctx)
}

func (s *Service) GetProfileID() string {
	if s.agentService == nil {
		return ""
	}
	return s.agentService.GetProfileID()
}

func (s *Service) SpawnSession(ctx context.Context, req agentmanager.SessionSpawnRequest) (agentmanager.RunResult, error) {
	spec, err := specFromContext(ctx)
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	if spec.OwnerType != OwnerSession {
		return agentmanager.RunResult{}, fmt.Errorf("SpawnSession requires owner_type=session")
	}
	return s.spawnSessionTracked(ctx, spec, req)
}

func (s *Service) ContinueRun(ctx context.Context, runID string, message string) error {
	spec, err := specFromContext(ctx)
	if err != nil {
		return err
	}
	if spec.OwnerType != OwnerSession {
		return fmt.Errorf("ContinueRun requires owner_type=session")
	}
	return s.continueTracked(ctx, spec, runID, message)
}

func (s *Service) GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error) {
	if s.agentService == nil {
		return agentmanager.RunState{}, agentmanager.ErrNotAvailable
	}
	return s.agentService.GetRunState(ctx, runID)
}

func (s *Service) GetRunEvents(ctx context.Context, runID string, opts agentmanager.RunEventsOptions) ([]*domainpb.RunEvent, bool, error) {
	if s.eventReader == nil {
		return nil, false, agentmanager.ErrNotAvailable
	}
	return s.eventReader.GetRunEvents(ctx, runID, opts)
}

func (s *Service) GetRunDiff(ctx context.Context, runID string) (agentmanager.RunDiff, error) {
	if s.differ == nil {
		return agentmanager.RunDiff{}, fmt.Errorf("run diff not available")
	}
	return s.differ.GetRunDiff(ctx, runID)
}
