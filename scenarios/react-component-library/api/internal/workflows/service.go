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
}

type service struct {
	repo       Repository
	dispatcher Dispatcher
}

func NewService(repo Repository, dispatcher Dispatcher) Service {
	return &service{repo: repo, dispatcher: dispatcher}
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
	dispatch, err := s.dispatcher.Dispatch(ctx, in)
	if err != nil {
		w.Status = StatusUnavailable
		w.Error = err.Error()
		w, _ = s.repo.Update(ctx, w)
		return w, 0, nil
	}
	w.AgentManagerTaskID, w.AgentManagerRunID, w.Status = dispatch.TaskID, dispatch.RunID, dispatch.Status
	if w.Status == "" {
		w.Status = StatusQueued
	}
	w, err = s.repo.Update(ctx, w)
	return w, dispatch.QueueDepth, err
}
func (s *service) List(ctx context.Context, asset, target string, active bool, limit int) ([]Workflow, error) {
	return s.repo.List(ctx, asset, target, active, limit)
}
func (s *service) Get(ctx context.Context, id string) (Workflow, error) { return s.repo.Get(ctx, id) }
func (s *service) Refresh(ctx context.Context, id string) (Workflow, error) {
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return Workflow{}, err
	}
	if w.AgentManagerRunID == "" || !w.Status.Active() {
		return w, nil
	}
	snap, err := s.dispatcher.Snapshot(ctx, w.AgentManagerRunID, w.LastEventSequence)
	if err != nil {
		w.Status = StatusUnavailable
		w.Error = err.Error()
	} else {
		w.Status = snap.Status
		w.Summary = snap.Summary
		w.Error = snap.Error
		w.LastEventSequence = snap.LastEventSequence
	}
	return s.repo.Update(ctx, w)
}
func (s *service) Stop(ctx context.Context, id string) (Workflow, error) {
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return Workflow{}, err
	}
	if !w.Status.Active() || w.AgentManagerRunID == "" {
		return w, nil
	}
	snap, err := s.dispatcher.Stop(ctx, w.AgentManagerRunID)
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
