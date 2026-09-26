// Package uiruntime is ui-health's BAS-driven runtime/render check group. It
// drives the target UI inside a host iframe through Browser Automation Studio's
// workflow engine and verifies the @vrooli/iframe-bridge handshake.
//
// It uses BAS's WorkflowsService.ExecuteAdhocWorkflow (not CaptureService — that
// rejects the about:blank host shell) with a thin timeline reader: the handshake
// assert + render screenshot ride in the RPC ExecutionTimeline; console/network
// are best-effort. Everything degrades gracefully — an unreachable BAS or UI
// yields skipped findings, never a hard failure.
package uiruntime

import (
	"context"

	"ui-health/internal/codefacts"
	"ui-health/internal/services/manifestvalidation"
)

// Input is what the validation handler hands the runtime group. It carries the
// already-resolved Code Facts so the group need not re-query.
type Input struct {
	Scenario    string
	ScenarioDir string
	Facts       codefacts.Facts
}

// Checker runs the runtime/render group and returns findings to fold into the
// single ui-health report. Check never returns an error: infrastructure absence
// (BAS down, UI not resolvable) is reported as skipped findings.
type Checker interface {
	Check(ctx context.Context, in Input) []manifestvalidation.Finding
}
