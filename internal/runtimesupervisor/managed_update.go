package runtimesupervisor

import (
	"context"
	"fmt"
	"time"
)

// startReconcileWorker isolates a durable owner from fleet renewal and other
// owners. A reconcile may hold a lifecycle lock or wait for native readiness.
func (s *Service) startReconcileWorker(parent context.Context, name string, reconcile func(context.Context) error) func() {
	if reconcile == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runReconcileWorker(ctx, name, reconcile)
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.logf("%s worker shutdown remains unresolved; durable checkpoint retained", name)
		}
	}
}

func (s *Service) runReconcileWorker(ctx context.Context, name string, reconcile func(context.Context) error) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		operation, cancel := context.WithTimeout(ctx, 4*time.Minute)
		err := reconcileWorkerOnce(operation, name, reconcile)
		cancel()
		if err != nil && ctx.Err() == nil {
			s.logf("%s degraded: %v", name, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func reconcileWorkerOnce(ctx context.Context, name string, reconcile func(context.Context) error) (err error) {
	defer func() {
		if problem := recover(); problem != nil {
			err = fmt.Errorf("%s panic contained: %v", name, problem)
		}
	}()
	return reconcile(ctx)
}
