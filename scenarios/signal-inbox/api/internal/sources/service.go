package sources

import (
	"context"
	"errors"
	"io"
	"math"
	"sort"
	"time"

	"signal-inbox/internal/signals"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type Service struct {
	repo     Repository
	captures CaptureService
	clock    schedule.Clock
	adapters map[string]Adapter
}

func NewService(repo Repository, captures CaptureService, clk schedule.Clock, adapters ...Adapter) (*Service, error) {
	service := &Service{repo: repo, captures: captures, clock: clk, adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		descriptor := adapter.Descriptor()
		if descriptor.ID == "" || descriptor.Kind == "" || descriptor.RiskTier == RiskUnspecified || descriptor.RiskTier > RiskTier2 {
			return nil, ErrInvalidDescriptor{"adapter id, kind, and declared risk tier are required"}
		}
		if _, exists := service.adapters[descriptor.ID]; exists {
			return nil, ErrInvalidDescriptor{"duplicate adapter id " + descriptor.ID}
		}
		service.adapters[descriptor.ID] = adapter
	}
	return service, nil
}

func (s *Service) State(ctx context.Context, id string) (State, error) {
	adapter, ok := s.adapters[id]
	if !ok {
		return State{}, ErrUnknownAdapter{ID: id}
	}
	state, found, err := s.repo.GetState(ctx, id)
	descriptor := adapter.Descriptor()
	if err != nil {
		return State{}, err
	}
	if found {
		state.Kind = descriptor.Kind
		return state, nil
	}
	state = State{AdapterID: id, Kind: descriptor.Kind, RiskTier: descriptor.RiskTier, Enabled: descriptor.RiskTier == RiskTier0}
	return state, s.repo.PutState(ctx, state)
}

func (s *Service) SetEnabled(ctx context.Context, id string, enabled bool) (State, error) {
	state, err := s.State(ctx, id)
	if err != nil {
		return State{}, err
	}
	state.Enabled, state.DisabledReason = enabled, ""
	return state, s.repo.PutState(ctx, state)
}

func (s *Service) List(ctx context.Context) ([]State, error) {
	ids := make([]string, 0, len(s.adapters))
	for id := range s.adapters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	states := make([]State, 0, len(ids))
	for _, id := range ids {
		state, err := s.State(ctx, id)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func (s *Service) Import(ctx context.Context, id string, input io.Reader) (ImportResult, error) {
	adapter, ok := s.adapters[id]
	if !ok {
		return ImportResult{}, ErrUnknownAdapter{ID: id}
	}
	state, err := s.State(ctx, id)
	if err != nil {
		return ImportResult{}, err
	}
	if !state.Enabled {
		return ImportResult{}, ErrAdapterDisabled{ID: id, Reason: state.DisabledReason}
	}
	started := s.clock.Now().UTC()
	entries, err := adapter.Parse(ctx, input) // exactly one adapter call; imports never retry.
	if err != nil {
		return s.finishFailure(ctx, state, started, err)
	}
	result := ImportResult{RunID: uuid.NewString(), AdapterID: id}
	for _, entry := range entries {
		captured, captureErr := s.captures.Capture(signals.WithInferenceDeferred(ctx), entry)
		if captureErr != nil {
			incrementCount(&result.Failed)
			return s.finishFailure(ctx, state, started, captureErr)
		}
		if captured.Duplicate {
			incrementCount(&result.Duplicated)
		} else {
			incrementCount(&result.Created)
		}
	}
	state.LastRunAt, state.LastError = s.clock.Now().UTC(), ""
	if err := s.repo.PutState(ctx, state); err != nil {
		return ImportResult{}, err
	}
	if err := s.repo.AppendRun(ctx, result, started, s.clock.Now().UTC()); err != nil {
		return ImportResult{}, err
	}
	return result, nil
}

func (s *Service) finishFailure(ctx context.Context, state State, started time.Time, cause error) (ImportResult, error) {
	state.LastRunAt, state.LastError = s.clock.Now().UTC(), cause.Error()
	var anomaly ErrAnomalousResponse
	if errors.As(cause, &anomaly) {
		state.Enabled, state.DisabledReason = false, anomaly.Reason
	}
	_ = s.repo.PutState(ctx, state)
	result := ImportResult{RunID: uuid.NewString(), AdapterID: state.AdapterID, Failed: 1}
	_ = s.repo.AppendRun(ctx, result, started, s.clock.Now().UTC())
	return result, cause
}

func incrementCount(value *uint32) {
	if *value < math.MaxUint32 {
		*value++
	}
}
