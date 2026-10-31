package control

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	internalflows "device-control/internal/flows"
)

var ErrLiveRunNotFound = errors.New("completed device run not found")

var (
	ErrLiveRunInProgress   = errors.New("device run is still in progress")
	ErrLiveRunNotEvictable = errors.New("device run is not an ordinary completed flow run")
	ErrLiveRunInconsistent = errors.New("completed device run cache is inconsistent")
)

// ForgetLiveRun drops only the process-local result, flow, and device index for
// an ordinary completed flow. It does not erase evidence, audits, saved flows,
// or retained copies. Readers that already copied a snapshot may still finish.
func (s *Service) ForgetLiveRun(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, hasResult := s.runs[id]
	_, hasFlow := s.flowRuns[id]
	device, hasDevice := s.runDevices[id]
	if !hasResult && !hasFlow && !hasDevice {
		return ErrLiveRunNotFound
	}
	if !hasResult || !hasFlow || !hasDevice || device == "" {
		for _, agent := range s.agents {
			if agent.PromotedFlowID == id {
				return ErrLiveRunNotEvictable
			}
		}
		return ErrLiveRunInconsistent
	}
	if result.Binding.LeaseID != "" {
		if _, active := s.activeCancels[result.Binding.LeaseID]; active {
			return ErrLiveRunInProgress
		}
	}
	for _, agent := range s.agents {
		if agent.PromotedFlowID == id {
			return ErrLiveRunNotEvictable
		}
	}

	// All three indexes are removed under one lock so inventory/export/retain
	// snapshots cannot observe a partially forgotten ordinary run.
	delete(s.runs, id)
	delete(s.flowRuns, id)
	delete(s.runDevices, id)
	return nil
}

// LiveRunIDs exposes only identifiers, so an owner can inventory process-local
// runs without disclosing targets, arguments, or evidence contents.
func (s *Service) LiveRunInventory() ([]string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.runs))
	for id := range s.runs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, len(s.activeCancels)
}

func (s *Service) RetainRun(ctx context.Context, id string, expiry time.Time) error {
	if s.retainedRuns == nil {
		return fmt.Errorf("durable run store unavailable")
	}
	s.mu.Lock()
	result, ok := s.runs[id]
	flow, found := s.flowRuns[id]
	device := s.runDevices[id]
	s.mu.Unlock()
	if !ok || !found || device == "" {
		return ErrLiveRunNotFound
	}
	return s.retainedRuns.Put(ctx, internalflows.RetainedRun{DeviceID: device, Flow: flow, Result: result, ExpiresAt: &expiry})
}

func (s *Service) RetainedRun(ctx context.Context, id string) (internalflows.RetainedRun, error) {
	if s.retainedRuns == nil {
		return internalflows.RetainedRun{}, fmt.Errorf("durable run store unavailable")
	}
	return s.retainedRuns.Get(ctx, id)
}

func (s *Service) RetainedRunIDs(ctx context.Context) ([]internalflows.RetainedRunSummary, error) {
	if s.retainedRuns == nil {
		return nil, fmt.Errorf("durable run store unavailable")
	}
	return s.retainedRuns.List(ctx)
}

func (s *Service) DeleteRetainedRun(ctx context.Context, id string) error {
	if s.retainedRuns == nil {
		return fmt.Errorf("durable run store unavailable")
	}
	return s.retainedRuns.Delete(ctx, id)
}
