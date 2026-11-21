// This file records an already-sanitized browser lifecycle/action payload on
// the run that owns the browser capability. It never parses page content and
// never reads BAS internals.
package orchestration

import (
	"context"
	"fmt"

	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
)

// RecordBrowserEvidence appends a sanitized run-reported lifecycle/action
// payload to the originating run. The transcript is not BAS attestation;
// callers must not use these counters as trusted billing measurements.
func (o *Orchestrator) RecordBrowserEvidence(ctx context.Context, runID uuid.UUID, payload eventlog.Payload) error {
	run, err := o.runs.Get(ctx, runID)
	if err != nil {
		return err
	}
	if run == nil || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil || !run.ResolvedConfig.Features.BrowserCapability.Enabled {
		return fmt.Errorf("browser capability is not enabled for run")
	}
	event, err := eventlog.BuildEvent(runID, payload)
	if err != nil {
		return err
	}
	return o.appendAndBroadcastEvents(ctx, runID, event)
}
