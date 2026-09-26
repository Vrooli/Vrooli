package fleetscheduler

import (
	"context"
	"errors"
	"testing"

	"test-genie/internal/runmanager"
)

type fakeRunStarter struct {
	startErr error
	result   runmanager.StartResult
	status   runmanager.LiveStatus
	waitErr  error
}

func (f fakeRunStarter) Start(runmanager.StartOptions) (runmanager.StartResult, error) {
	return f.result, f.startErr
}

func (f fakeRunStarter) Wait(context.Context, string, string) (runmanager.LiveStatus, error) {
	return f.status, f.waitErr
}

func TestManagerLauncherMapsBusyError(t *testing.T) {
	l := NewManagerLauncher(fakeRunStarter{startErr: &runmanager.BusyError{Scenario: "x", RunID: "r1"}}, "comprehensive")
	_, err := l.Launch(context.Background(), "x")
	if !errors.Is(err, ErrScenarioBusy) {
		t.Fatalf("Launch() err = %v, want ErrScenarioBusy", err)
	}
}

func TestManagerLauncherLaunchAndAwait(t *testing.T) {
	starter := fakeRunStarter{
		result: runmanager.StartResult{RunID: "run-9"},
		status: runmanager.LiveStatus{Verdict: "passed"},
	}
	l := NewManagerLauncher(starter, "comprehensive")
	id, err := l.Launch(context.Background(), "x")
	if err != nil || id != "run-9" {
		t.Fatalf("Launch() = %q, %v", id, err)
	}
	st, err := l.Await(context.Background(), "x", id)
	if err != nil || st != "passed" {
		t.Fatalf("Await() = %q, %v", st, err)
	}
}

func TestManagerLauncherAwaitFallsBackToStatus(t *testing.T) {
	// No verdict (e.g. aborted) -> lifecycle status is returned.
	l := NewManagerLauncher(fakeRunStarter{status: runmanager.LiveStatus{Status: "aborted"}}, "")
	st, err := l.Await(context.Background(), "x", "r")
	if err != nil || st != "aborted" {
		t.Fatalf("Await() = %q, %v", st, err)
	}
}
