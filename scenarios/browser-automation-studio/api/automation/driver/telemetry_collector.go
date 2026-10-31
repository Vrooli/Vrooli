package driver

import (
	"context"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// TelemetryCollector is the instrumentation seam the executor invokes
// around step execution. It is the Go-side counterpart to the
// playwright-driver instrumentation hook: a place for cross-cutting
// telemetry (perf timing, tracing, counters) to attach without the
// executor knowing what is being collected.
//
// The default wiring uses NoOpCollector, so today these hooks are inert.
// P2 of the performance-health buildout supplies a real collector that
// records per-step timing/perf signals into the capture pipeline.
//
// Hook contract:
//   - BeforeStep runs immediately before the step's driver call. The
//     returned context is threaded into the step (allowing the collector
//     to stash span handles, deadlines, etc.).
//   - AfterStep runs immediately after the step completes (success or
//     failure), receiving the outcome and any error from the driver call.
type TelemetryCollector interface {
	BeforeStep(ctx context.Context, instr contracts.CompiledInstruction) context.Context
	AfterStep(ctx context.Context, instr contracts.CompiledInstruction, outcome contracts.StepOutcome, err error)
}

// NoOpCollector is the default TelemetryCollector. Every hook is inert; it
// returns the context unchanged and records nothing.
type NoOpCollector struct{}

// BeforeStep returns ctx unchanged.
func (NoOpCollector) BeforeStep(ctx context.Context, _ contracts.CompiledInstruction) context.Context {
	return ctx
}

// AfterStep does nothing.
func (NoOpCollector) AfterStep(context.Context, contracts.CompiledInstruction, contracts.StepOutcome, error) {
}
