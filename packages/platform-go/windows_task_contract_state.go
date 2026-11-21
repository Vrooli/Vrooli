package platform

import (
	"context"
	"time"
)

// TASK_STATE is a finite native enum. UNKNOWN is a valid observation, but only
// DISABLED and READY explicitly prove that no instances are queued or running.
const (
	windowsTaskUnknown int32 = iota
	windowsTaskDisabled
	windowsTaskQueued
	windowsTaskReady
	windowsTaskRunning
)

func validWindowsTaskState(state int32) bool {
	return state >= windowsTaskUnknown && state <= windowsTaskRunning
}

func windowsTaskQuiescent(snapshot WindowsTaskSnapshot) (bool, error) {
	if !validWindowsTaskState(snapshot.State) {
		return false, ErrOperationIdentity
	}
	for _, instance := range snapshot.Instances {
		if !validWindowsTaskState(instance.State) {
			return false, ErrOperationIdentity
		}
	}
	if snapshot.State == windowsTaskUnknown {
		return false, ErrOperationOutcomeLost
	}
	return len(snapshot.Instances) == 0 && (snapshot.State == windowsTaskDisabled || snapshot.State == windowsTaskReady), nil
}

func windowsTaskNeedsRun(snapshot WindowsTaskSnapshot) (bool, error) {
	quiet, err := windowsTaskQuiescent(snapshot)
	if err != nil {
		return false, err
	}
	if !snapshot.Enabled || len(snapshot.Instances) > 1 {
		return false, ErrOperationIdentity
	}
	if snapshot.State == windowsTaskDisabled {
		return false, ErrOperationIdentity
	}
	// A queued or running action is already submitted, even if GetInstances
	// currently exposes none. Await it rather than issuing a second Run.
	return quiet, nil
}

func waitWindowsTaskQuiescent(ctx context.Context, query func() (WindowsTaskSnapshot, error)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot, err := query()
		if err != nil {
			return err
		}
		quiet, err := windowsTaskQuiescent(snapshot)
		if err != nil || quiet {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

type windowsOperatorTaskControlEffects struct {
	stop  func(context.Context, WindowsTaskSnapshot) error
	query func() (WindowsTaskSnapshot, error)
	run   func() error
	await func(context.Context) error
}

// This is the ordinary installed supervisor's actual stop/start/restart policy.
// Native callbacks retain installed definition/account/image and action proofs.
func controlWindowsOperatorTaskWith(ctx context.Context, verb string, snapshot WindowsTaskSnapshot, e windowsOperatorTaskControlEffects) error {
	if verb != "stop" && verb != "start" && verb != "restart" {
		return ErrUnsupported
	}
	if verb == "stop" || verb == "restart" {
		if err := e.stop(ctx, snapshot); err != nil {
			return err
		}
		if verb == "stop" {
			return nil
		}
		var err error
		snapshot, err = e.query()
		if err != nil {
			return err
		}
	}
	run, err := windowsTaskNeedsRun(snapshot)
	if err != nil {
		return err
	}
	if run {
		if err := e.run(); err != nil {
			return err
		}
	}
	return e.await(ctx)
}

// Both role retirement and privileged boot removal independently requery the
// exact owned definition immediately before deletion and prove absence after it.
func deleteWindowsQuiescentTask(query func() (WindowsTaskSnapshot, error), remove func(string) error, missing func(error) bool) error {
	snapshot, err := query()
	if missing(err) {
		return nil
	}
	if err != nil {
		return err
	}
	quiet, err := windowsTaskQuiescent(snapshot)
	if err != nil {
		return err
	}
	if snapshot.Enabled || !quiet {
		return ErrOperationNotQuiescent
	}
	if err := remove(snapshot.Name); err != nil {
		return err
	}
	_, err = query()
	if missing(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return ErrOperationNotQuiescent
}
