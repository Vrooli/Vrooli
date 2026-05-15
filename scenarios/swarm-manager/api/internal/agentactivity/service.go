package agentactivity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/idgen"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

var errNotFound = errors.New("agent activity not found")

type rawAgentService interface {
	IsEnabled() bool
	IsAvailable(ctx context.Context) bool
	ResolveURL(ctx context.Context) (string, error)
	GetProfileID() string
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
	StopRun(ctx context.Context, runID string) error
}

// initiativeSpawner is satisfied by agentmanager.AgentService and any test
// double that wants to participate in tracked initiative spawns. Kept
// separate from rawAgentService so existing stubs (capture, backlog) need
// no churn when agentactivity is extended.
type initiativeSpawner interface {
	SpawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error)
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

// Service is the single Swarm Manager seam for tracked agent-manager usage.
// All backlog/capture/scenario work spawns and continuations must flow through
// this service so durable activity records remain complete.
type Service struct {
	store             Store
	agentService      rawAgentService
	continuer         runContinuer
	differ            runDiffer
	eventReader       runEventReader
	initiativeSpawner initiativeSpawner
	sessionSpawner    sessionSpawner
	lanePolicy        LanePolicy
	eventDispatcher   dispatch.NodeDispatcher
	mu                sync.Mutex
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
	if spawner, ok := cfg.AgentService.(initiativeSpawner); ok {
		svc.initiativeSpawner = spawner
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

func (s *Service) SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	spec, err := specFromContext(ctx)
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	return s.spawnTracked(ctx, spec, req)
}

// SpawnInitiative tracks an initiative-scoped agent run (feedback rounds,
// initiative reviews) so the activity log carries the same provenance as
// backlog work. The Spec on the context must use OwnerInitiative.
func (s *Service) SpawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
	spec, err := specFromContext(ctx)
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	if spec.OwnerType != OwnerInitiative {
		return agentmanager.RunResult{}, fmt.Errorf("SpawnInitiative requires owner_type=initiative")
	}
	return s.spawnInitiativeTracked(ctx, spec, req)
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

func metadataWithProfileKey(metadata map[string]string, profileKey string) map[string]string {
	key := strings.TrimSpace(profileKey)
	if key == "" {
		return metadata
	}
	out := make(map[string]string, len(metadata)+1)
	for k, v := range metadata {
		out[k] = v
	}
	out["agent_profile_key"] = key
	return out
}

func (s *Service) StopRun(ctx context.Context, runID string) error {
	if s.agentService == nil {
		return agentmanager.ErrNotAvailable
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return fmt.Errorf("run id is required")
	}
	if err := s.agentService.StopRun(ctx, trimmed); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return err
	}
	now := nowRFC3339()
	changed := false
	for i := range records {
		if strings.TrimSpace(records[i].RunID) != trimmed || !isActiveStatus(records[i].Status) {
			continue
		}
		records[i].Status = StatusCancelled
		records[i].FinishedAt = now
		records[i].UpdatedAt = now
		changed = true
	}
	if !changed {
		return nil
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	for _, record := range records {
		if record.RunID == trimmed && record.Status == StatusCancelled {
			s.dispatchStatusUpdate(record)
		}
	}
	return nil
}

func (s *Service) Get(ctx context.Context, activityID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.refreshActiveLocked(ctx); err != nil {
		return Record{}, err
	}
	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
	}
	trimmed := strings.TrimSpace(activityID)
	for _, record := range records {
		if record.ActivityID == trimmed {
			return record, nil
		}
	}
	return Record{}, errNotFound
}

func (s *Service) List(ctx context.Context, filters ListFilters) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.refreshActiveLocked(ctx); err != nil {
		return nil, err
	}
	return s.listSnapshot(filters)
}

// ListSnapshot returns activity records from the persisted store without
// polling agent-manager. Graph projections use this snapshot path so opening
// the graph does not block on live run-state checks; the normal List endpoint
// keeps the refresh behavior for operators explicitly viewing activity state.
func (s *Service) ListSnapshot(_ context.Context, filters ListFilters) ([]Record, error) {
	return s.listSnapshot(filters)
}

func (s *Service) listSnapshot(filters ListFilters) ([]Record, error) {
	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		if matchesFilters(record, filters) {
			filtered = append(filtered, record)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].RequestedAt == filtered[j].RequestedAt {
			return filtered[i].ActivityID > filtered[j].ActivityID
		}
		return filtered[i].RequestedAt > filtered[j].RequestedAt
	})

	return filtered, nil
}

// HasActiveAgent checks whether a backlog item has an active agent
// (pending, starting, running, or needs_review). Used by non-spawn guards
// like WorkshopDeleteRound and WorkshopReset to prevent mutations while
// an agent is working. Returns false on store errors (safe-side: allow
// operations to proceed if the store is broken).
func (s *Service) HasActiveAgent(ctx context.Context, ownerKind, ownerName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return false
	}
	_ = s.refreshActiveForOwnerLocked(ctx, records, ownerKind, ownerName)
	records, _ = s.store.Load()

	for _, rec := range records {
		if rec.OwnerType != OwnerBacklog || rec.OwnerKind != ownerKind || rec.OwnerName != ownerName {
			continue
		}
		if isPendingStale(rec) {
			continue
		}
		if isActiveStatus(rec.Status) {
			return true
		}
	}
	return false
}

func (s *Service) spawnTracked(
	ctx context.Context,
	spec Spec,
	req agentmanager.BacklogSpawnRequest,
) (agentmanager.RunResult, error) {
	if s.agentService == nil || !s.agentService.IsEnabled() {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	// Guard: at most one active agent per backlog item.
	if spec.OwnerType == OwnerBacklog {
		// Best-effort refresh of this item's active records to clear stale state.
		if refreshErr := s.refreshActiveForOwnerLocked(ctx, records, spec.OwnerKind, spec.OwnerName); refreshErr != nil {
			slog.Warn("backlog item guard: refresh failed, proceeding with stale data", "err", refreshErr)
		}
		// Re-load after refresh may have saved updated statuses.
		records, err = s.store.Load()
		if err != nil {
			s.mu.Unlock()
			return agentmanager.RunResult{}, err
		}

		staleCleaned := false
		for i, rec := range records {
			if rec.OwnerType != OwnerBacklog || rec.OwnerKind != spec.OwnerKind || rec.OwnerName != spec.OwnerName {
				continue
			}
			// Auto-fail stale pending records (no RunID, older than TTL).
			if isPendingStale(rec) {
				records[i].Status = StatusFailed
				records[i].FailureReason = "pending spawn timed out"
				records[i].FinishedAt = nowRFC3339()
				records[i].UpdatedAt = records[i].FinishedAt
				staleCleaned = true
				continue
			}
			if isActiveStatus(rec.Status) {
				s.mu.Unlock()
				return agentmanager.RunResult{}, fmt.Errorf("%w: %s/%s already has an active agent (activity=%s, status=%s, purpose=%s)",
					ErrBacklogItemBusy, rec.OwnerKind, rec.OwnerName, rec.ActivityID, rec.Status, rec.Purpose)
			}
		}
		if staleCleaned {
			if err := s.store.Save(records); err != nil {
				s.mu.Unlock()
				return agentmanager.RunResult{}, err
			}
		}
	}

	// Lane gate. Backlog spawns predate P2 — if PhaseKind is unset, fall
	// back to the per-Purpose default lane via LaneOf. Saturation returns
	// ErrLaneSaturated so QueueBacklog can leave the queued item pending.
	if err := s.checkLaneCapacityLocked(spec, records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	now := nowRFC3339()
	record := Record{
		ActivityID:      idgen.Generate(),
		OwnerType:       spec.OwnerType,
		OwnerKind:       spec.OwnerKind,
		OwnerName:       spec.OwnerName,
		OwnerTitle:      spec.OwnerTitle,
		ExecutionID:     spec.ExecutionID,
		PhaseKind:       spec.PhaseKind,
		Purpose:         spec.Purpose,
		InteractionType: InteractionSpawn,
		Status:          StatusPending,
		RequestedAt:     now,
		RequestedBy:     spec.RequestedBy,
		Metadata:        metadataWithProfileKey(spec.Metadata, req.ProfileKey),
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	runResult, spawnErr := s.agentService.SpawnBacklog(ctx, req)

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err = s.store.Load()
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	idx := indexByID(records, record.ActivityID)
	if idx < 0 {
		return agentmanager.RunResult{}, fmt.Errorf("agent activity disappeared during spawn")
	}

	updated := records[idx]
	updated.UpdatedAt = nowRFC3339()
	if spawnErr != nil {
		updated.Status = StatusFailed
		updated.FailureReason = spawnErr.Error()
		updated.FinishedAt = updated.UpdatedAt
		records[idx] = updated
		if err := s.store.Save(records); err != nil {
			return agentmanager.RunResult{}, err
		}
		s.dispatchStatusUpdate(updated)
		return agentmanager.RunResult{}, spawnErr
	}

	updated.TaskID = strings.TrimSpace(runResult.TaskID)
	updated.RunID = strings.TrimSpace(runResult.RunID)
	updated.Status = StatusStarting
	updated.StartedAt = updated.UpdatedAt
	records[idx] = updated
	if err := s.store.Save(records); err != nil {
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(updated)
	return runResult, nil
}

// spawnInitiativeTracked is the initiative analogue of spawnTracked. There
// is no per-owner exclusivity guard — feedback rounds gate themselves via
// the on-disk feedback lock, which is checked before this is called.
func (s *Service) spawnInitiativeTracked(
	ctx context.Context,
	spec Spec,
	req agentmanager.InitiativeSpawnRequest,
) (agentmanager.RunResult, error) {
	if s.initiativeSpawner == nil {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}
	if s.agentService == nil || !s.agentService.IsEnabled() {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	// Lane gate. Initiative-scoped phase runs declare PhaseKind explicitly
	// (the phase runner reads it from PhaseDefinition.Kind), so saturation
	// here surfaces ErrLaneSaturated to the operating-mode round refresher,
	// which defers via the pending_auto_start sidecar (P4) on retry.
	if err := s.checkLaneCapacityLocked(spec, records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	now := nowRFC3339()
	record := Record{
		ActivityID:      idgen.Generate(),
		OwnerType:       spec.OwnerType,
		OwnerKind:       spec.OwnerKind,
		OwnerName:       spec.OwnerName,
		OwnerTitle:      spec.OwnerTitle,
		ExecutionID:     spec.ExecutionID,
		PhaseKind:       spec.PhaseKind,
		Purpose:         spec.Purpose,
		InteractionType: InteractionSpawn,
		Status:          StatusPending,
		RequestedAt:     now,
		RequestedBy:     spec.RequestedBy,
		Metadata:        metadataWithProfileKey(spec.Metadata, req.ProfileKey),
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	runResult, spawnErr := s.initiativeSpawner.SpawnInitiative(ctx, req)

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err = s.store.Load()
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	idx := indexByID(records, record.ActivityID)
	if idx < 0 {
		return agentmanager.RunResult{}, fmt.Errorf("agent activity disappeared during spawn")
	}

	updated := records[idx]
	updated.UpdatedAt = nowRFC3339()
	if spawnErr != nil {
		updated.Status = StatusFailed
		updated.FailureReason = spawnErr.Error()
		updated.FinishedAt = updated.UpdatedAt
		records[idx] = updated
		if err := s.store.Save(records); err != nil {
			return agentmanager.RunResult{}, err
		}
		s.dispatchStatusUpdate(updated)
		return agentmanager.RunResult{}, spawnErr
	}

	updated.TaskID = strings.TrimSpace(runResult.TaskID)
	updated.RunID = strings.TrimSpace(runResult.RunID)
	updated.Status = StatusStarting
	updated.StartedAt = updated.UpdatedAt
	records[idx] = updated
	if err := s.store.Save(records); err != nil {
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(updated)
	return runResult, nil
}

func (s *Service) spawnSessionTracked(
	ctx context.Context,
	spec Spec,
	req agentmanager.SessionSpawnRequest,
) (agentmanager.RunResult, error) {
	if s.sessionSpawner == nil {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}
	if s.agentService == nil || !s.agentService.IsEnabled() {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	// Lane gate. Session-scoped spawns (interactive operator sessions)
	// honor the same lanes — without phaseKind they fall back to the
	// per-Purpose default; saturation returns ErrLaneSaturated to the
	// caller (no enqueue path).
	if err := s.checkLaneCapacityLocked(spec, records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}

	now := nowRFC3339()
	record := Record{
		ActivityID:      idgen.Generate(),
		OwnerType:       spec.OwnerType,
		OwnerKind:       spec.OwnerKind,
		OwnerName:       spec.OwnerName,
		OwnerTitle:      spec.OwnerTitle,
		ExecutionID:     spec.ExecutionID,
		PhaseKind:       spec.PhaseKind,
		Purpose:         spec.Purpose,
		InteractionType: InteractionSpawn,
		Status:          StatusPending,
		RequestedAt:     now,
		RequestedBy:     spec.RequestedBy,
		Metadata:        metadataWithProfileKey(spec.Metadata, req.ProfileKey),
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	runResult, spawnErr := s.sessionSpawner.SpawnSession(ctx, req)

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err = s.store.Load()
	if err != nil {
		return agentmanager.RunResult{}, err
	}
	idx := indexByID(records, record.ActivityID)
	if idx < 0 {
		return agentmanager.RunResult{}, fmt.Errorf("agent activity disappeared during spawn")
	}

	updated := records[idx]
	updated.UpdatedAt = nowRFC3339()
	if spawnErr != nil {
		updated.Status = StatusFailed
		updated.FailureReason = spawnErr.Error()
		updated.FinishedAt = updated.UpdatedAt
		records[idx] = updated
		if err := s.store.Save(records); err != nil {
			return agentmanager.RunResult{}, err
		}
		s.dispatchStatusUpdate(updated)
		return agentmanager.RunResult{}, spawnErr
	}

	updated.TaskID = strings.TrimSpace(runResult.TaskID)
	updated.RunID = strings.TrimSpace(runResult.RunID)
	updated.Status = StatusStarting
	updated.StartedAt = updated.UpdatedAt
	records[idx] = updated
	if err := s.store.Save(records); err != nil {
		return agentmanager.RunResult{}, err
	}
	s.dispatchStatusUpdate(updated)
	return runResult, nil
}

func (s *Service) continueTracked(ctx context.Context, spec Spec, runID string, message string) error {
	if s.continuer == nil {
		return fmt.Errorf("run continuation not available")
	}
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return fmt.Errorf("run id is required")
	}

	s.mu.Lock()
	records, err := s.store.Load()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	now := nowRFC3339()
	record := Record{
		ActivityID:      idgen.Generate(),
		OwnerType:       spec.OwnerType,
		OwnerKind:       spec.OwnerKind,
		OwnerName:       spec.OwnerName,
		OwnerTitle:      spec.OwnerTitle,
		ExecutionID:     spec.ExecutionID,
		PhaseKind:       spec.PhaseKind,
		Purpose:         spec.Purpose,
		InteractionType: InteractionContinue,
		RunID:           trimmedRunID,
		Status:          StatusPending,
		RequestedAt:     now,
		RequestedBy:     spec.RequestedBy,
		Metadata:        spec.Metadata,
		UpdatedAt:       now,
	}
	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return err
	}
	s.dispatchStatusUpdate(record)
	s.mu.Unlock()

	continueErr := s.continuer.ContinueRun(ctx, trimmedRunID, message)

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err = s.store.Load()
	if err != nil {
		return err
	}
	idx := indexByID(records, record.ActivityID)
	if idx < 0 {
		return fmt.Errorf("agent activity disappeared during continuation")
	}
	updated := records[idx]
	updated.UpdatedAt = nowRFC3339()
	if continueErr != nil {
		updated.Status = StatusFailed
		updated.FailureReason = continueErr.Error()
		updated.FinishedAt = updated.UpdatedAt
	} else {
		updated.Status = StatusRunning
		updated.StartedAt = updated.UpdatedAt
	}
	records[idx] = updated
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(updated)
	return continueErr
}
