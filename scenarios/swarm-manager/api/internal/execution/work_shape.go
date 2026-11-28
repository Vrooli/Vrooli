package execution

import (
	"context"
	"strings"

	"swarm-manager/internal/transitionrunner"
)

// SetPlanWorkGuard installs the owning work-shape boundary before the service
// starts serving requests. It applies to direct API/automatic queue admission
// as well as execution-record transitions; a board action is not a guard.
func (s *Service) SetPlanWorkGuard(guard func(context.Context, string) error) {
	s.planWorkGuard = guard
}

func (s *Service) checkPlanWork(ctx context.Context, kind, name string) error {
	if s.planWorkGuard == nil {
		return nil
	}
	return s.planWorkGuard(ctx, strings.ToLower(strings.TrimSpace(kind))+"/"+strings.TrimSpace(name))
}

func (s *Service) guardPlanInput(builder transitionrunner.InputBuilder) transitionrunner.InputBuilder {
	return func(ctx context.Context, executionID string) (transitionrunner.Snapshot, error) {
		// Input builders run both inside and outside the execution mutex. Read
		// the durable store directly, like the existing builders; Get would
		// refresh executions and recursively acquire that mutex.
		records, index, err := s.loadRecordLocked(executionID)
		if err != nil {
			return transitionrunner.Snapshot{}, err
		}
		record := records[index]
		if err := s.checkPlanWork(ctx, record.BacklogKind, record.BacklogName); err != nil {
			return transitionrunner.Snapshot{}, err
		}
		return builder(ctx, executionID)
	}
}
