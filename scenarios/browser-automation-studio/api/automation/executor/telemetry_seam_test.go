package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// fakeTelemetrySession is a minimal EngineSession whose Run returns a
// successful outcome, letting us assert the telemetry seam fires.
type fakeTelemetrySession struct {
	runs int
}

func (s *fakeTelemetrySession) Run(_ context.Context, _ contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	s.runs++
	return contracts.StepOutcome{Success: true}, nil
}
func (s *fakeTelemetrySession) Reset(context.Context) error { return nil }
func (s *fakeTelemetrySession) Close(context.Context) error { return nil }
func (s *fakeTelemetrySession) GetStorageState(context.Context) (json.RawMessage, error) {
	return nil, nil
}

// recordingCollector captures hook invocations to verify ordering.
type recordingCollector struct {
	before int
	after  int
	order  []string
}

func (c *recordingCollector) BeforeStep(ctx context.Context, _ contracts.CompiledInstruction) context.Context {
	c.before++
	c.order = append(c.order, "before")
	return ctx
}

func (c *recordingCollector) AfterStep(context.Context, contracts.CompiledInstruction, contracts.StepOutcome, error) {
	c.after++
	c.order = append(c.order, "after")
}

func TestRunWithRetries_DefaultsToNoOpCollector(t *testing.T) {
	// A SimpleExecutor built via NewSimpleExecutor must run cleanly with the
	// inert default collector — i.e. the no-op path is the behavior-preserving
	// default for P1.
	e := NewSimpleExecutor(nil)
	sess := &fakeTelemetrySession{}
	outcome, err := e.runWithRetries(context.Background(), Request{}, sess, contracts.CompiledInstruction{NodeID: "n1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !outcome.Success {
		t.Fatalf("expected success outcome")
	}
	if sess.runs != 1 {
		t.Fatalf("expected 1 driver run, got %d", sess.runs)
	}

	// The zero-value executor (no constructor) must also default to NoOp.
	if _, ok := (&SimpleExecutor{}).telemetryCollector().(driver.NoOpCollector); !ok {
		t.Fatalf("zero-value executor must default to NoOpCollector")
	}
}

func TestRunWithRetries_InvokesCollectorAroundStep(t *testing.T) {
	c := &recordingCollector{}
	e := NewSimpleExecutor(nil).WithTelemetryCollector(c)
	sess := &fakeTelemetrySession{}

	_, err := e.runWithRetries(context.Background(), Request{}, sess, contracts.CompiledInstruction{NodeID: "n1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.before != 1 || c.after != 1 {
		t.Fatalf("expected one before+after, got before=%d after=%d", c.before, c.after)
	}
	if len(c.order) != 2 || c.order[0] != "before" || c.order[1] != "after" {
		t.Fatalf("expected before then after, got %v", c.order)
	}
}

func TestWithTelemetryCollector_NilResetsToNoOp(t *testing.T) {
	e := NewSimpleExecutor(nil).WithTelemetryCollector(&recordingCollector{})
	e.WithTelemetryCollector(nil)
	if _, ok := e.telemetryCollector().(driver.NoOpCollector); !ok {
		t.Fatalf("nil collector must reset to NoOpCollector")
	}
}
