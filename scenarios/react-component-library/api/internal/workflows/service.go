package workflows

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Service interface {
	Start(context.Context, StartInput) (Workflow, int, error)
	List(context.Context, string, string, bool, int) ([]Workflow, error)
	Get(context.Context, string) (Workflow, error)
	Refresh(context.Context, string) (Workflow, error)
	Stop(context.Context, string) (Workflow, error)
	Retry(context.Context, string, string) (Workflow, int, error)
	PromotionReadiness(context.Context, PromotionReadinessInput) (PromotionReadiness, error)
}

type service struct {
	repo       Repository
	dispatcher Dispatcher
	readiness  PromotionReadinessReader
}

func NewService(repo Repository, dispatcher Dispatcher, readiness ...PromotionReadinessReader) Service {
	s := &service{repo: repo, dispatcher: dispatcher}
	if len(readiness) > 0 {
		s.readiness = readiness[0]
	}
	return s
}

func (s *service) PromotionReadiness(ctx context.Context, in PromotionReadinessInput) (PromotionReadiness, error) {
	if s.readiness == nil {
		return PromotionReadiness{}, fmt.Errorf("promotion readiness reader not configured")
	}
	return s.readiness.PromotionReadiness(ctx, in)
}

func (s *service) Start(ctx context.Context, in StartInput) (Workflow, int, error) {
	if !validKind(in.Kind) {
		return Workflow{}, 0, fmt.Errorf("workflow kind must be extract or adopt")
	}
	if in.Kind == KindExtract && (strings.TrimSpace(in.SourceScenario) == "" || strings.TrimSpace(in.SourcePath) == "") {
		return Workflow{}, 0, fmt.Errorf("extract workflow requires source_scenario and source_path")
	}
	if in.Kind == KindAdopt && (strings.TrimSpace(in.AssetID) == "" || strings.TrimSpace(in.TargetScenario) == "") {
		return Workflow{}, 0, fmt.Errorf("adopt workflow requires asset_id and target_scenario")
	}
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.IdempotencyKey == "" {
		return Workflow{}, 0, fmt.Errorf("idempotency_key is required")
	}
	if prior, err := s.repo.FindActiveByIdempotency(ctx, in.IdempotencyKey); err == nil {
		return prior, 0, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Workflow{}, 0, err
	}
	w, err := s.repo.Create(ctx, Workflow{Kind: in.Kind, AssetID: in.AssetID, SourceScenario: in.SourceScenario, TargetScenario: in.TargetScenario, SourcePath: in.SourcePath, RequestedVersion: in.RequestedVersion, IdempotencyKey: in.IdempotencyKey, Status: StatusQueued})
	if err != nil {
		return Workflow{}, 0, err
	}
	dispatch, err := s.dispatcher.Start(ctx, in)
	if err != nil {
		w.Status = StatusUnavailable
		w.Error = err.Error()
		w, _ = s.repo.Update(ctx, w)
		return w, 0, nil
	}
	w.AgentManagerExecutionID, w.Status, w.Summary, w.Error = dispatch.ExecutionID, dispatch.Status, dispatch.Summary, dispatch.Error
	if w.Status == "" {
		w.Status = StatusQueued
	}
	w, err = s.repo.Update(ctx, w)
	if err != nil {
		return w, 0, err
	}
	// Persist the execution identity before entering the server-owned wait so
	// a caller disconnect cannot orphan the RCL record from its workflow.
	completed, err := s.dispatcher.Wait(ctx, w.AgentManagerExecutionID)
	if err != nil {
		w.Status, w.Error = StatusUnavailable, err.Error()
		w, _ = s.repo.Update(ctx, w)
		return w, 0, nil
	}
	w.Status, w.Summary, w.Error = completed.Status, completed.Summary, completed.Error
	if w.Status == "" {
		w.Status = StatusQueued
	}
	w, err = s.repo.Update(ctx, w)
	return w, 0, err
}

func (s *service) List(ctx context.Context, asset, target string, active bool, limit int) ([]Workflow, error) {
	workflows, err := s.repo.List(ctx, asset, target, active, limit)
	if err != nil {
		return nil, err
	}
	refresher, ok := s.dispatcher.(RefreshingDispatcher)
	if !ok {
		return workflows, nil
	}
	for index := range workflows {
		workflow := workflows[index]
		if !workflow.Status.Active() || workflow.AgentManagerExecutionID == "" {
			continue
		}
		// Listing is a best-effort read path. A temporary Agent Manager outage
		// must not make durable RCL history disappear; the next read can retry
		// reconciliation. A successful terminal refresh is persisted so active
		// filters stop exposing executions that have already ended.
		snapshot, refreshErr := refresher.Refresh(ctx, workflow.AgentManagerExecutionID)
		if refreshErr != nil {
			continue
		}
		workflow.Status, workflow.Summary, workflow.Error = snapshot.Status, snapshot.Summary, snapshot.Error
		if !workflow.Status.Active() {
			updated, updateErr := s.repo.Update(ctx, workflow)
			if updateErr != nil {
				return nil, updateErr
			}
			workflow = updated
		}
		workflows[index] = workflow
	}
	return workflows, nil
}
func (s *service) Get(ctx context.Context, id string) (Workflow, error) {
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return Workflow{}, err
	}
	return s.reconcileActive(ctx, w)
}
func (s *service) Refresh(ctx context.Context, id string) (Workflow, error) {
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return Workflow{}, err
	}
	return s.reconcileActive(ctx, w)
}

// reconcileActive makes reads recovery-aware. A client can disconnect after
// Agent Manager has persisted a terminal result but before RCL's synchronous
// waiter persists its projection (manual review is one example). The waiter
// records that boundary as unavailable, so an execution-backed unavailable
// row must be reconciled just like an active row. Without a read-time
// reconciliation, the durable RCL row would remain stale forever.
func (s *service) reconcileActive(ctx context.Context, w Workflow) (Workflow, error) {
	if (w.Status.Active() || w.Status == StatusUnavailable) && w.AgentManagerExecutionID != "" {
		if refresher, ok := s.dispatcher.(RefreshingDispatcher); ok {
			snapshot, refreshErr := refresher.Refresh(ctx, w.AgentManagerExecutionID)
			if refreshErr != nil {
				return Workflow{}, refreshErr
			}
			w.Status, w.Summary, w.Error = snapshot.Status, snapshot.Summary, snapshot.Error
			if !w.Status.Active() {
				updated, err := s.repo.Update(ctx, w)
				if err != nil {
					return Workflow{}, err
				}
				w = updated
			}
		}
	}
	return w, nil
}

func (s *service) Stop(ctx context.Context, id string) (Workflow, error) {
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return Workflow{}, err
	}
	if !w.Status.Active() || w.AgentManagerExecutionID == "" {
		return w, nil
	}
	snap, err := s.dispatcher.Stop(ctx, w.AgentManagerExecutionID)
	if err != nil {
		return Workflow{}, err
	}
	w.Status = snap.Status
	if w.Status == "" {
		w.Status = StatusStopped
	}
	w.Summary = snap.Summary
	w.Error = snap.Error
	w.LastEventSequence = snap.LastEventSequence
	return s.repo.Update(ctx, w)
}

func (s *service) Retry(ctx context.Context, id, key string) (Workflow, int, error) {
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return Workflow{}, 0, err
	}
	if w.Status.Active() {
		return w, 0, nil
	}
	if strings.TrimSpace(key) == "" {
		return Workflow{}, 0, fmt.Errorf("idempotency_key is required")
	}
	return s.Start(ctx, StartInput{Kind: w.Kind, AssetID: w.AssetID, SourceScenario: w.SourceScenario, TargetScenario: w.TargetScenario, SourcePath: w.SourcePath, RequestedVersion: w.RequestedVersion, IdempotencyKey: key})
}
func validKind(k Kind) bool { return k == KindExtract || k == KindAdopt }
