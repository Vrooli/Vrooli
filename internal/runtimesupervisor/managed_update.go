package runtimesupervisor

import (
	"context"
	"fmt"
	"time"
)

// This worker is independent of fleet renewal. A resource operation may hold
// its own lifecycle lock or wait for native readiness; it must never delay the
// supervisor's scenario lease heartbeats. The durable owner handles process
// replacement and recovers the same request after this worker is lost.
func (s *Service) runManagedUpdateWorker(ctx context.Context) {
	if s.cfg.ManagedUpdateReconcile == nil {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		operation, cancel := context.WithTimeout(ctx, 4*time.Minute)
		err := s.reconcileManagedUpdatesOnce(operation)
		cancel()
		if err != nil && ctx.Err() == nil {
			s.logf("managed update recovery degraded: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) reconcileManagedUpdatesOnce(ctx context.Context) (err error) {
	defer func() {
		if problem := recover(); problem != nil {
			err = fmt.Errorf("managed update recovery panic contained: %v", problem)
		}
	}()
	return s.cfg.ManagedUpdateReconcile(ctx)
}
