package main

import (
	"context"
	"errors"
	"fmt"
)

type apiCleanupPlan struct {
	drainExports      func(context.Context) error
	cancelCheckpoints context.CancelFunc
	checkpointsDone   <-chan struct{}
	schedulerStop     func(context.Context) error
	workflowShutdown  func(context.Context) error
	cancelReconciler  context.CancelFunc
	reconcilerDone    <-chan struct{}
	stopSidecar       func(context.Context) error
	closeDatabase     func() error
}

// runAPICleanup is the production shutdown composition used by server.Run.
// Dependencies below the workflow/reconciler join boundary are deliberately
// left live when an owner misses the cleanup deadline; startup recovery is the
// only safe continuation once its process has exited.
func runAPICleanup(ctx context.Context, plan apiCleanupPlan) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var failures []error
	if plan.drainExports != nil {
		if err := plan.drainExports(ctx); err != nil {
			failures = append(failures, fmt.Errorf("drain exports: %w", err))
		}
	}
	if plan.cancelCheckpoints != nil {
		plan.cancelCheckpoints()
	}
	if plan.checkpointsDone != nil {
		if err := waitForOwner(ctx, plan.checkpointsDone, "profile checkpoints"); err != nil {
			failures = append(failures, err)
		}
	}
	if plan.schedulerStop != nil {
		if err := plan.schedulerStop(ctx); err != nil {
			failures = append(failures, fmt.Errorf("stop scheduler: %w", err))
		}
	}
	// Always close workflow admission after scheduled admission has stopped,
	// even when scheduler draining failed. Do not close driver/DB owners unless
	// both owners report a complete join.
	if plan.workflowShutdown != nil {
		if err := plan.workflowShutdown(ctx); err != nil {
			failures = append(failures, fmt.Errorf("join workflow executions: %w", err))
		}
	}
	if len(failures) != 0 {
		return errors.Join(failures...)
	}
	if plan.cancelReconciler != nil {
		plan.cancelReconciler()
	}
	if plan.reconcilerDone != nil {
		if err := waitForOwner(ctx, plan.reconcilerDone, "session reconciler"); err != nil {
			return err
		}
	}
	if plan.stopSidecar != nil {
		if err := plan.stopSidecar(ctx); err != nil {
			return fmt.Errorf("stop sidecar: %w", err)
		}
	}
	if plan.closeDatabase != nil {
		if err := plan.closeDatabase(); err != nil {
			return fmt.Errorf("close database: %w", err)
		}
	}
	return nil
}

func waitForOwner(ctx context.Context, done <-chan struct{}, owner string) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for %s: %w", owner, ctx.Err())
	}
}
