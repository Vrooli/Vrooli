package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/config"
)

var ErrWorkflowServiceShuttingDown = errors.New("workflow service is shutting down")
var ErrWorkflowExecutionCapacity = errors.New("workflow execution capacity is full")

// cancelAndJoinExecutionOwners is the single owner for stopping local runner
// contexts and joining their cleanup. It cancels every selected runner before
// waiting so service shutdown cannot serialize cancellation across executions.
func cancelAndJoinExecutionOwners(ctx context.Context, controls map[uuid.UUID]executionControl) error {
	for _, control := range controls {
		if control.cancel != nil {
			control.cancel()
		}
	}
	for id, control := range controls {
		if control.done == nil {
			continue
		}
		select {
		case <-control.done:
		case <-ctx.Done():
			return fmt.Errorf("join execution %s: %w", id, ctx.Err())
		}
	}
	return nil
}

// beginExecutionAdmission protects the interval between accepting an execution
// and registering its local runner control. Shutdown closes this gate before
// taking the runner snapshot, so no admitted execution can appear behind it.
func (s *WorkflowService) beginExecutionAdmission() (releaseAdmission func(), releaseSlot func(), err error) {
	if s == nil {
		return nil, nil, errors.New("workflow service not configured")
	}
	s.executionLifecycleMu.Lock()
	defer s.executionLifecycleMu.Unlock()
	if s.executionClosing {
		return nil, nil, ErrWorkflowServiceShuttingDown
	}
	limit := s.maxConcurrentExecutions
	if limit < 1 {
		limit = config.DefaultMaxConcurrentExecutions
	}
	if s.executionSlotsInUse >= limit {
		return nil, nil, fmt.Errorf("%w: limit=%d", ErrWorkflowExecutionCapacity, limit)
	}
	if s.executionAdmissions == 0 {
		s.executionAdmissionsDrained = make(chan struct{})
	}
	s.executionAdmissions++
	s.executionSlotsInUse++
	var admissionOnce, slotOnce sync.Once
	return func() {
			admissionOnce.Do(func() {
				s.executionLifecycleMu.Lock()
				defer s.executionLifecycleMu.Unlock()
				s.executionAdmissions--
				if s.executionAdmissions == 0 {
					close(s.executionAdmissionsDrained)
				}
			})
		}, func() {
			slotOnce.Do(func() {
				s.executionLifecycleMu.Lock()
				defer s.executionLifecycleMu.Unlock()
				s.executionSlotsInUse--
			})
		}, nil
}

// Shutdown permanently closes execution admission, waits for any execution
// being admitted to register its owner control, then cancels and joins every
// local runner. Call before stopping the browser sidecar and while persistence
// is still available. A deadline leaves the unresolved runner to startup
// recovery rather than pretending its cleanup completed.
func (s *WorkflowService) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.executionLifecycleMu.Lock()
	s.executionClosing = true
	var admissionsDrained <-chan struct{}
	if s.executionAdmissions == 0 {
		closed := make(chan struct{})
		close(closed)
		admissionsDrained = closed
	} else {
		admissionsDrained = s.executionAdmissionsDrained
	}
	s.executionLifecycleMu.Unlock()
	select {
	case <-admissionsDrained:
	case <-ctx.Done():
		return fmt.Errorf("wait for execution admission to drain: %w", ctx.Err())
	}

	controls := make(map[uuid.UUID]executionControl)
	s.executionCancels.Range(func(key, value any) bool {
		id, idOK := key.(uuid.UUID)
		control, controlOK := value.(executionControl)
		if idOK && controlOK {
			controls[id] = control
		}
		return true
	})
	if err := cancelAndJoinExecutionOwners(ctx, controls); err != nil {
		return fmt.Errorf("join executions during shutdown: %w", err)
	}
	return nil
}
