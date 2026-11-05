package process

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Incarnation reads the OS identity of a currently present PID. A missing or
// unsupported identity is an error, never evidence that a recorded launch exited.
func Incarnation(pid int) (string, error) { return processIncarnation(pid) }

// LaunchRefID binds the registry's opaque process reference to one launch.
func LaunchRefID(generation string) string { return "launch-" + generation }

// PinRecord opens a stable handle and verifies the recorded launch while it is
// alive. The caller owns Close. A previously dead PID cannot supply exit proof.
func PinRecord(record Record) (Handle, error) {
	return pinRecord(record, OpenHandle, Incarnation, func(pid int) (string, error) {
		env, err := ReadEnvironment(pid)
		return env["VROOLI_COMPONENT_GENERATION"], err
	})
}

func pinRecord(record Record, open func(int) (Handle, error), identity func(int) (string, error), generation func(int) (string, error)) (Handle, error) {
	if record.PID <= 0 || record.Generation == "" || record.Incarnation == "" {
		return nil, errors.New("process launch identity is incomplete")
	}
	before, err := identity(record.PID)
	if err != nil || before != record.Incarnation {
		return nil, errors.New("recorded process incarnation is unavailable or mismatched before pin")
	}
	beforeGeneration, err := generation(record.PID)
	if err != nil || beforeGeneration != record.Generation {
		return nil, errors.New("recorded launch generation is unavailable or mismatched before pin")
	}
	handle, err := open(record.PID)
	if err != nil {
		return nil, fmt.Errorf("pin recorded process: %w", err)
	}
	actual, err := identity(record.PID)
	if err != nil || actual != before || actual != record.Incarnation {
		_ = handle.Close()
		return nil, errors.New("recorded process incarnation is unavailable or mismatched")
	}
	afterGeneration, err := generation(record.PID)
	if err != nil || afterGeneration != record.Generation {
		_ = handle.Close()
		return nil, errors.New("recorded launch generation is unavailable or mismatched after pin")
	}
	alive, err := handle.Alive()
	if err != nil || !alive {
		_ = handle.Close()
		return nil, errors.New("recorded process was not alive when pinned")
	}
	return handle, nil
}

// AwaitExit observes only the previously pinned incarnation. A context timeout
// or an inspection error never becomes successful component completion.
func AwaitExit(ctx context.Context, handle Handle) error {
	if handle == nil {
		return errors.New("stable process handle is required")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		alive, err := handle.Alive()
		if err != nil {
			return fmt.Errorf("observe pinned process exit: %w", err)
		}
		if !alive {
			return nil
		}
		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
