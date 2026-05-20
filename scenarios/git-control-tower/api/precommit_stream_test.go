package main

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeStreamingRunner struct {
	lines    []string
	exitCode int
	delay    time.Duration
}

func (r fakeStreamingRunner) Run(ctx context.Context, req CommandRunRequest) (CommandRunResult, error) {
	res, err := r.RunStream(ctx, req, nil)
	return res, err
}

func (r fakeStreamingRunner) RunStream(ctx context.Context, _ CommandRunRequest, onLine func(stream, line string)) (CommandRunResult, error) {
	var stdout strings.Builder
	for _, line := range r.lines {
		if r.delay > 0 {
			select {
			case <-ctx.Done():
				return CommandRunResult{Stdout: stdout.String()}, ctx.Err()
			case <-time.After(r.delay):
			}
		}
		stdout.WriteString(line + "\n")
		if onLine != nil {
			onLine("stdout", line)
		}
	}
	if r.exitCode != 0 {
		return CommandRunResult{Stdout: stdout.String()}, fakeExitError{code: r.exitCode}
	}
	return CommandRunResult{Stdout: stdout.String()}, nil
}

type capturingEmitter struct {
	mu     sync.Mutex
	events []PrecommitStreamEvent
}

func (c *capturingEmitter) Emit(event PrecommitStreamEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

func (c *capturingEmitter) snapshot() []PrecommitStreamEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]PrecommitStreamEvent, len(c.events))
	copy(out, c.events)
	return out
}

func TestRunStreamEmitsStartedProgressFinishedOnSuccess(t *testing.T) {
	svc := newTestPrecommitServiceWithRunner(t, fakeStreamingRunner{lines: []string{"step1", "step2"}})
	ctx := context.Background()
	if _, err := svc.Save(ctx, "/tmp/repo", PrecommitConfig{Enabled: true, Command: "noop", WorkingDirectory: "/tmp/repo", TimeoutSeconds: 30, RunBeforeCommit: true, AllowOverride: true}); err != nil {
		t.Fatalf("save: %v", err)
	}
	emitter := &capturingEmitter{}
	result, err := svc.RunStream(ctx, "/tmp/repo", PrecommitRunRequest{}, emitter)
	if err != nil {
		t.Fatalf("run stream: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("expected passed, got %s", result.Status)
	}
	events := emitter.snapshot()
	if len(events) < 2 {
		t.Fatalf("expected at least started+finished, got %d", len(events))
	}
	if events[0].Type != "started" {
		t.Fatalf("first event must be started, got %s", events[0].Type)
	}
	last := events[len(events)-1]
	if last.Type != "finished" {
		t.Fatalf("last event must be finished, got %s", last.Type)
	}
	if last.Result == nil || last.Result.Status != "passed" {
		t.Fatalf("finished event missing passed result: %+v", last)
	}
	sawProgress := false
	for _, ev := range events {
		if ev.Type == "progress" {
			sawProgress = true
			if len(ev.Tail) == 0 {
				t.Fatalf("progress event should carry tail lines, got empty")
			}
		}
	}
	if !sawProgress {
		t.Fatalf("expected at least one progress event")
	}
}

func TestRunStreamEmitsFinishedFailedOnNonZeroExit(t *testing.T) {
	svc := newTestPrecommitServiceWithRunner(t, fakeStreamingRunner{lines: []string{"oops"}, exitCode: 7})
	ctx := context.Background()
	if _, err := svc.Save(ctx, "/tmp/repo", PrecommitConfig{Enabled: true, Command: "noop", WorkingDirectory: "/tmp/repo", TimeoutSeconds: 30, RunBeforeCommit: true, AllowOverride: true}); err != nil {
		t.Fatalf("save: %v", err)
	}
	emitter := &capturingEmitter{}
	result, err := svc.RunStream(ctx, "/tmp/repo", PrecommitRunRequest{}, emitter)
	if err != nil {
		t.Fatalf("run stream: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if result.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %d", result.ExitCode)
	}
	events := emitter.snapshot()
	last := events[len(events)-1]
	if last.Type != "finished" || last.Result == nil || last.Result.Status != "failed" {
		t.Fatalf("expected finished+failed event, got %+v", last)
	}
}

func TestTailBufferKeepsLastN(t *testing.T) {
	tb := newTailBuffer(3)
	for i, line := range []string{"a", "b", "c", "d", "e"} {
		tb.add(line)
		_ = i
	}
	got := tb.snapshot()
	want := []string{"c", "d", "e"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
