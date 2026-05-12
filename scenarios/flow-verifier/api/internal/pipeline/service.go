// service.go exposes the high-level Verify entry point that CLI and HTTP
// handlers call: discover flows under root, optionally filter to one
// flow id, then drive Run with the chosen mode.
package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"flow-verifier/internal/flows/discovery"
	"flow-verifier/internal/flows/model"
)

// Recorder is the seam pipeline.Verify uses to persist one history row
// per flow per invocation. Implementations live outside pipeline
// (handlers/verifications wires it to internal/runs.Service); pipeline
// only needs Record to fire-and-forget.
type Recorder interface {
	Record(ctx context.Context, entry RunEntry) error
}

// RunEntry is the per-flow record handed to a Recorder. ID/timestamps
// may be left zero; the recorder is free to fill them.
type RunEntry struct {
	FlowID       string
	FlowPath     string
	Root         string
	Mode         Mode
	Status       string
	Output       string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
}

// VerifyOptions configures a Verify invocation.
type VerifyOptions struct {
	Root     string
	FlowID   string
	Mode     Mode
	Stdout   io.Writer
	Recorder Recorder // optional; nil disables history persistence
}

// VerifyResult summarises what Verify did.
type VerifyResult struct {
	Flows []model.Flow
	Mode  Mode
}

// Verify discovers flows under Root (filtered by FlowID if non-empty),
// compiles them, and runs the pipeline in the requested mode. If a
// Recorder is supplied, one entry is recorded per flow with the flow's
// terminal status. A pipeline-level failure (no flows discovered, quint
// missing, etc.) is reported via the returned error and not recorded
// per-flow.
func Verify(ctx context.Context, opts VerifyOptions) (VerifyResult, error) {
	rootAbs, err := filepath.Abs(opts.Root)
	if err != nil {
		return VerifyResult{}, err
	}
	flows, err := discovery.FindContracts(rootAbs)
	if err != nil {
		return VerifyResult{}, err
	}
	selected := discovery.Filter(flows, opts.FlowID)
	if opts.FlowID != "" && len(selected) == 0 {
		return VerifyResult{}, fmt.Errorf("unknown flow id %s", opts.FlowID)
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	var perFlow *bytes.Buffer
	var teedStdout io.Writer = stdout
	if opts.Recorder != nil {
		perFlow = &bytes.Buffer{}
		teedStdout = io.MultiWriter(stdout, perFlow)
	}

	started := make(map[string]time.Time, len(selected))
	for _, flow := range selected {
		started[flow.FlowID] = time.Now().UTC()
	}

	runErr := Run(ctx, Options{
		Root:   rootAbs,
		Flows:  selected,
		Mode:   opts.Mode,
		Stdout: teedStdout,
	})

	if opts.Recorder != nil {
		recordPerFlow(ctx, opts.Recorder, selected, started, opts.Mode, rootAbs, perFlow.String(), runErr)
	}
	if runErr != nil {
		return VerifyResult{}, runErr
	}
	return VerifyResult{Flows: selected, Mode: opts.Mode}, nil
}

// recordPerFlow translates a single Verify invocation into one Recorder
// entry per flow. pipeline.Run iterates flows in order and stops at the
// first failure, so any flow that emitted output before the error is
// recorded as passed, the flow that triggered the error is recorded as
// failed, and any remaining flows are skipped (not recorded).
func recordPerFlow(ctx context.Context, rec Recorder, flows []model.Flow, started map[string]time.Time, mode Mode, root string, output string, runErr error) {
	finished := time.Now().UTC()
	// Detect which flow the error belongs to by checking the trailing
	// flow id mentioned in output. pipeline.Run prints "fresh <id>"
	// or "wrote <path>" before advancing. If the error is non-nil and
	// references a flow id, that flow's entry is "failed"; flows
	// processed before it are "passed". This stays a heuristic — Phase
	// E records the minimum the UI needs; deeper per-flow attribution
	// is a follow-up.
	failingFlow := ""
	if runErr != nil && len(flows) > 0 {
		failingFlow = guessFailingFlow(output, flows)
	}
	for _, flow := range flows {
		entry := RunEntry{
			FlowID:     flow.FlowID,
			FlowPath:   flow.ContractPath,
			Root:       root,
			Mode:       mode,
			Output:     output,
			StartedAt:  started[flow.FlowID],
			FinishedAt: finished,
		}
		switch {
		case runErr == nil:
			entry.Status = "passed"
		case failingFlow == "" || failingFlow == flow.FlowID:
			entry.Status = "failed"
			entry.ErrorMessage = runErr.Error()
			failingFlow = flow.FlowID
		default:
			// Earlier flow than the failing one: it succeeded before
			// pipeline aborted on the next one.
			entry.Status = "passed"
		}
		_ = rec.Record(ctx, entry)
		if entry.Status == "failed" {
			// Any flow after the failing one was not processed; do not
			// emit speculative "skipped" rows. Phase E keeps this
			// minimal — Phase F can add a "skipped" status if the UI
			// needs it.
			return
		}
	}
}

// guessFailingFlow walks output for the last flow id mentioned; pipeline
// prints "fresh <id>" or "wrote <path-containing-id>" before advancing,
// so the last mentioned id is the one that was actively being processed
// when Run returned an error. Falls back to the first flow.
func guessFailingFlow(output string, flows []model.Flow) string {
	lastSeen := ""
	for _, flow := range flows {
		if strings.Contains(output, flow.FlowID) {
			lastSeen = flow.FlowID
		}
	}
	if lastSeen == "" && len(flows) > 0 {
		return flows[0].FlowID
	}
	return lastSeen
}
