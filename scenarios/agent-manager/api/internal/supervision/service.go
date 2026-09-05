package supervision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	watches  *Repository
	events   eventlog.CohortRepository
	mu       sync.Mutex
	signals  map[string]chan struct{}
	kick     func()
	actions  *ActionService
	policies *PolicyStore
}

func (s *Service) SetSchedulerKick(kick func())            { s.kick = kick }
func (s *Service) SetActionService(actions *ActionService) { s.actions = actions }
func (s *Service) SetPolicyStore(policies *PolicyStore)    { s.policies = policies }

func NewService(watches *Repository, events eventlog.CohortRepository) *Service {
	return &Service{watches: watches, events: events, signals: map[string]chan struct{}{}}
}

func (s *Service) Create(ctx context.Context, req *domainpb.CreateCohortWatchRequest) (*domainpb.CohortWatch, bool, error) {
	if req == nil || req.GetSpec() == nil {
		return nil, false, errors.New("watch spec is required")
	}
	spec := proto.Clone(req.GetSpec()).(*domainpb.WatchSpec)
	if s.policies != nil {
		if strings.TrimSpace(spec.GetPolicyVersion()) == "" {
			active, err := s.policies.Active(ctx)
			if err != nil {
				return nil, false, fmt.Errorf("resolve active supervision policy: %w", err)
			}
			spec.PolicyVersion = active.Policy.Version
		} else if _, err := s.policies.Get(ctx, spec.GetPolicyVersion()); err != nil {
			return nil, false, fmt.Errorf("resolve supervision policy %q: %w", spec.GetPolicyVersion(), err)
		}
	}
	if s.policies != nil {
		record, err := s.policies.Get(ctx, spec.GetPolicyVersion())
		if err != nil {
			return nil, false, err
		}
		if record.State == "rejected" || record.State == "rolled_back" {
			return nil, false, errors.New("policy is not eligible for a new watch")
		}
		if spec.Triggers == nil {
			spec.Triggers = &domainpb.WatchTriggers{}
		}
		spec.Triggers.EventCount = record.Policy.EventCount
		spec.Triggers.FrictionScore = record.Policy.FrictionThreshold
		spec.Triggers.QuietTime = durationpb.New(time.Duration(record.Policy.QuietSeconds) * time.Second)
	}
	if spec.Triggers == nil {
		spec.Triggers = &domainpb.WatchTriggers{}
	}
	if spec.Triggers.GetEventCount() == 0 {
		spec.Triggers.EventCount = 64
	}
	// Terminal observation is a safety invariant. Proto3 bool presence cannot
	// distinguish an omitted value from an attempt to disable it.
	spec.Triggers.Terminal = true
	if spec.Triggers.GetFrictionScore() < 0 || spec.Triggers.GetFrictionScore() > 1 {
		return nil, false, errors.New("friction score must be between 0 and 1")
	}
	retention, err := s.events.RetentionState(ctx)
	if err != nil {
		return nil, false, err
	}
	watch, _, reused, err := s.watches.Create(ctx, spec, req.GetIdempotencyKey(), retention.Generation)
	if err == nil && s.kick != nil {
		s.kick()
	}
	return watch, reused, err
}

func (s *Service) Get(ctx context.Context, watchID string) (*domainpb.CohortWatch, error) {
	watch, _, err := s.watches.Get(ctx, watchID)
	return watch, err
}

func (s *Service) List(ctx context.Context, req *domainpb.ListCohortWatchesRequest) (*domainpb.ListCohortWatchesResponse, error) {
	if req == nil {
		req = &domainpb.ListCohortWatchesRequest{}
	}
	watches, next, err := s.watches.List(ctx, req.GetFamilyExecutionId(), req.GetStatus(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	return &domainpb.ListCohortWatchesResponse{Watches: watches, NextPageToken: next}, nil
}

func (s *Service) Cancel(ctx context.Context, req *domainpb.CancelCohortWatchRequest) (*domainpb.CohortWatch, error) {
	if req == nil || strings.TrimSpace(req.GetWatchId()) == "" {
		return nil, errors.New("watch id is required")
	}
	watch, err := s.watches.Cancel(ctx, req.GetWatchId(), req.GetExpectedRevision())
	if err == nil {
		s.notify(req.GetWatchId())
	}
	return watch, err
}

func (s *Service) Wait(ctx context.Context, req *domainpb.WaitCohortWatchRequest) (*domainpb.WaitCohortWatchResponse, error) {
	if req == nil || strings.TrimSpace(req.GetWatchId()) == "" {
		return nil, errors.New("watch id is required")
	}
	timeout := 30 * time.Second
	if req.GetTimeout().IsValid() && req.GetTimeout().AsDuration() > 0 {
		timeout = req.GetTimeout().AsDuration()
		if timeout > 30*time.Second {
			timeout = 30 * time.Second
		}
	}
	signal := s.signal(req.GetWatchId())
	watch, err := s.Get(ctx, req.GetWatchId())
	if err != nil {
		return nil, err
	}
	if watch.GetRevision() > req.GetAfterRevision() || watch.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_ACTIVE {
		return &domainpb.WaitCohortWatchResponse{Watch: watch}, nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		watch, err = s.Get(ctx, req.GetWatchId())
		return &domainpb.WaitCohortWatchResponse{Watch: watch, TimedOut: true}, err
	case <-signal:
		watch, err = s.Get(ctx, req.GetWatchId())
		return &domainpb.WaitCohortWatchResponse{Watch: watch}, err
	}
}

func (s *Service) WaitTerminal(ctx context.Context, watchID string) (*domainpb.CohortWatch, error) {
	for {
		signal := s.signal(watchID)
		watch, err := s.Get(ctx, watchID)
		if err != nil {
			return nil, err
		}
		if watch.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_ACTIVE {
			return watch, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-signal:
		}
	}
}

func (s *Service) RequestAction(ctx context.Context, req *domainpb.RequestCohortWatchActionRequest) (*domainpb.RequestCohortWatchActionResponse, error) {
	if s.actions == nil {
		return nil, errors.New("cohort action service is unavailable")
	}
	return s.actions.Request(ctx, req)
}

func (s *Service) ListActions(ctx context.Context, req *domainpb.ListCohortWatchActionsRequest) (*domainpb.ListCohortWatchActionsResponse, error) {
	if s.actions == nil {
		return nil, errors.New("cohort action service is unavailable")
	}
	if req == nil || strings.TrimSpace(req.GetWatchId()) == "" {
		return nil, errors.New("watch id is required")
	}
	return s.actions.List(ctx, req.GetWatchId(), req.GetLimit())
}

func (s *Service) RecoverActions(ctx context.Context) (int, error) {
	if s.actions == nil {
		return 0, nil
	}
	return s.actions.RecoverPending(ctx)
}

func (s *Service) signal(watchID string) <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.signals[watchID] == nil {
		s.signals[watchID] = make(chan struct{})
	}
	return s.signals[watchID]
}

func (s *Service) notify(watchID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if signal := s.signals[watchID]; signal != nil {
		close(signal)
		delete(s.signals, watchID)
	}
}

// Inspect is read-only: it never advances the durable cursor. The scheduler
// advances only when its decision is committed atomically with the checkpoint.
func (s *Service) Inspect(ctx context.Context, req *domainpb.InspectCohortWatchRequest) (*domainpb.InspectCohortWatchResponse, error) {
	if req == nil || strings.TrimSpace(req.GetWatchId()) == "" {
		return nil, errors.New("watch id is required")
	}
	watch, checkpoint, err := s.watches.Get(ctx, req.GetWatchId())
	if err != nil {
		return nil, err
	}
	retention, err := s.events.RetentionState(ctx)
	if err != nil {
		return nil, err
	}
	response := &domainpb.InspectCohortWatchResponse{Watch: watch}
	if retention.Generation != checkpoint.RetentionGeneration {
		response.CursorResetRequired = true
		response.ResetReason = fmt.Sprintf("event retention generation changed from %d to %d; reconcile from durable run summaries", checkpoint.RetentionGeneration, retention.Generation)
		return response, nil
	}
	runIDs := make([]uuid.UUID, 0, len(watch.GetSpec().GetSubjects()))
	for _, subject := range watch.GetSpec().GetSubjects() {
		runID, err := uuid.Parse(subject.GetRunId())
		if err != nil {
			return nil, fmt.Errorf("watch subject run id %q: %w", subject.GetRunId(), err)
		}
		runIDs = append(runIDs, runID)
	}
	limit := int(req.GetEventLimit())
	if limit == 0 {
		limit = 64
	}
	events, err := s.events.ReadCohort(ctx, runIDs, checkpoint.RowID, limit)
	if err != nil {
		return nil, err
	}
	response.Events = make([]*domainpb.WatchEventEnvelope, 0, len(events))
	for _, event := range events {
		response.Events = append(response.Events, &domainpb.WatchEventEnvelope{EventId: event.ID.String(), RunId: event.RunID.String(), Sequence: event.Sequence, EventType: string(event.EventType), Timestamp: timestamppb.New(event.Timestamp)})
	}
	return response, nil
}
