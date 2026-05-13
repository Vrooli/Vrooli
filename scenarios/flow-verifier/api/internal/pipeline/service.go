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
	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation"
	"flow-verifier/internal/flows/kinds/temporal"
	"flow-verifier/internal/flows/kinds/temporal/model"
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
//
// FailureReason narrows a Status="failed" entry into the recoverable
// categories the UI's "Needs generate" affordance keys off: empty means
// "no further detail / status is the whole story", "missing_artifacts"
// and "stale_artifacts" come from a typed *FreshnessError, other
// reasons are reserved for future substrate fixes (counterexample,
// lint, quint_failure, io). MissingArtifacts carries the structured
// list of artifact paths the UI lists in its Artifacts panel.
type RunEntry struct {
	FlowID           string
	FlowPath         string
	Root             string
	Mode             Mode
	Status           string
	Output           string
	ErrorMessage     string
	FailureReason    string
	MissingArtifacts []string
	StartedAt        time.Time
	FinishedAt       time.Time
}

// FailureReason* constants name the typed categories the recorder may
// stamp onto a failed RunEntry. Kept as exported strings (not iota) so
// they round-trip cleanly through JSON / SQLite.
const (
	FailureReasonMissingArtifacts = "missing_artifacts"
	FailureReasonStaleArtifacts   = "stale_artifacts"
	FailureReasonCounterexample   = "counterexample"
	FailureReasonLint             = "lint"
	FailureReasonQuintFailure     = "quint_failure"
	FailureReasonIO               = "io"
)

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
	specs, err := discovery.FindContracts(rootAbs)
	if err != nil {
		return VerifyResult{}, err
	}
	filtered := discovery.Filter(specs, opts.FlowID)
	if opts.FlowID != "" && len(filtered) == 0 {
		return VerifyResult{}, fmt.Errorf("unknown flow id %s", opts.FlowID)
	}
	var temporalSpecs, navigationSpecs []kind.Spec
	for _, s := range filtered {
		switch s.Kind() {
		case temporal.Name:
			temporalSpecs = append(temporalSpecs, s)
		case navigation.Name:
			navigationSpecs = append(navigationSpecs, s)
		default:
			return VerifyResult{}, fmt.Errorf("unsupported kind %q for flow %s", s.Kind(), s.FlowID())
		}
	}
	selected := temporal.FlowsFromSpecs(temporalSpecs)

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

	// Navigation kind: dispatched independently of the temporal pipeline.
	// Each navigation spec runs through kind.Verify; findings stream to
	// stdout and a single Recorder entry is written per spec.
	navErr := runNavigation(ctx, navigationSpecs, opts, rootAbs, teedStdout)
	if runErr != nil {
		return VerifyResult{}, runErr
	}
	if navErr != nil {
		return VerifyResult{}, navErr
	}
	return VerifyResult{Flows: selected, Mode: opts.Mode}, nil
}

func runNavigation(ctx context.Context, specs []kind.Spec, opts VerifyOptions, rootAbs string, stdout io.Writer) error {
	if len(specs) == 0 {
		return nil
	}
	navKind, ok := kind.Get(navigation.Name)
	if !ok {
		return fmt.Errorf("navigation kind not registered")
	}
	var firstErr error
	for _, s := range specs {
		started := time.Now().UTC()
		res, vErr := navKind.Verify(ctx, s)
		finished := time.Now().UTC()
		out := renderNavigationReport(s, res, vErr)
		_, _ = io.WriteString(stdout, out)

		status := "passed"
		errMsg := ""
		failureReason := ""
		if vErr != nil {
			status = "failed"
			errMsg = vErr.Error()
			failureReason = FailureReasonIO
		} else if !res.Passed {
			status = "failed"
			errMsg = "navigation invariants failed"
			failureReason = FailureReasonCounterexample
		}
		if opts.Recorder != nil {
			_ = opts.Recorder.Record(ctx, RunEntry{
				FlowID:        s.FlowID(),
				FlowPath:      s.ContractPath(),
				Root:          rootAbs,
				Mode:          opts.Mode,
				Status:        status,
				Output:        out,
				ErrorMessage:  errMsg,
				FailureReason: failureReason,
				StartedAt:     started,
				FinishedAt:    finished,
			})
		}
		if status == "failed" && firstErr == nil {
			if vErr != nil {
				firstErr = vErr
			} else {
				firstErr = fmt.Errorf("navigation flow %s: invariants failed", s.FlowID())
			}
		}
	}
	return firstErr
}

func renderNavigationReport(s kind.Spec, res kind.VerifyResult, vErr error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "navigation %s (%s)\n", s.FlowID(), s.ContractPath())
	if vErr != nil {
		fmt.Fprintf(&b, "  error: %v\n", vErr)
		return b.String()
	}
	passed, failed := 0, 0
	for _, f := range res.Findings {
		if f.Passed {
			passed++
		} else {
			failed++
		}
	}
	fmt.Fprintf(&b, "  findings: %d passed, %d failed\n", passed, failed)
	for _, f := range res.Findings {
		if !f.Passed {
			fmt.Fprintf(&b, "  - %s\n", f.Message)
		}
	}
	return b.String()
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
			if fe, ok := AsFreshnessError(runErr); ok {
				entry.MissingArtifacts = fe.Paths()
				switch fe.Kind {
				case FreshnessMissing:
					entry.FailureReason = FailureReasonMissingArtifacts
				case FreshnessStale:
					entry.FailureReason = FailureReasonStaleArtifacts
				}
			}
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
