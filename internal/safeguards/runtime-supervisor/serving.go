package runtimesupervisorsafeguard

import (
	"context"
	"fmt"
	"time"
)

// WaitServing blocks until the native root supervisor is active, runs the
// image with expectedSHA, and holds a fresh registry lease naming
// expectedBuild after renewing its managed fleet. It is the serving-image
// verification an operator baseline transition needs after Converge; an
// admitted activation uses ActivationEffects.VerifyCandidate instead.
func WaitServing(ctx context.Context, home, expectedSHA, expectedBuild string) error {
	return waitServing(ctx, ActivationEffects{}.operations(), home, expectedSHA, expectedBuild)
}

func waitServing(ctx context.Context, ops activationOps, home, expectedSHA, expectedBuild string) error {
	for {
		err := observeServing(ctx, ops, home, expectedSHA, expectedBuild)
		if err == nil {
			return nil
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("root supervisor did not verify the new serving image: %w (last observation: %v)", ctx.Err(), err)
		case <-timer.C:
		}
	}
}

func observeServing(ctx context.Context, ops activationOps, home, expectedSHA, expectedBuild string) error {
	state, err := ops.inspect(ctx)
	if err != nil {
		return err
	}
	if state.LoadState != "loaded" || state.ActiveState != "active" || state.SubState != "running" || state.PendingJob || state.PID <= 0 || state.NeedDaemonReload != "no" {
		return fmt.Errorf("native root owner is not active and settled")
	}
	image, err := ops.image(ctx, state.PID)
	if err != nil {
		return err
	}
	if image != expectedSHA {
		return fmt.Errorf("native root owner runs image %.12s, not %.12s", image, expectedSHA)
	}
	incarnation, err := ops.incarnation(state.PID)
	if err != nil {
		return err
	}
	lease, err := ops.lease(ctx, home)
	if err != nil {
		return err
	}
	now := time.Now()
	if lease.PID != state.PID || lease.Build != expectedBuild || lease.Incarnation != incarnation || !lease.Deadline.After(now) || lease.Heartbeat.After(now.Add(time.Second)) {
		return fmt.Errorf("root supervisor live image and fresh registry owner do not agree")
	}
	if !lease.FleetRenewed || lease.Heartbeat.Sub(lease.Started) < 250*time.Millisecond {
		return fmt.Errorf("root supervisor has not completed fresh fleet renewal")
	}
	return nil
}
