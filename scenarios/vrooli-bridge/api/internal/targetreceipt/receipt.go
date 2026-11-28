// Package targetreceipt defines truthful target-lane evidence projected from
// Bridge's durable remote-run history. Local contract evidence is deliberately
// unable to count as a real QEMU, VPS, desktop, or mobile lane verdict.
package targetreceipt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vrooli-bridge/internal/runs"
)

const (
	SchemaVersion = 1

	LaneLocalContract = "local-contract"
	ClassLocalFixture = "local-fixture"
	ProvenanceLocal   = "local-fixture"
)

var (
	ErrRunNotTerminal = errors.New("target receipt requires a terminal durable run")
	ErrMissingExit    = errors.New("target receipt requires a persisted exit event")
)

// Receipt is an immutable, machine-readable projection of a durable Bridge
// run. It carries an explicit provenance and eligibility bit so consumers can
// distinguish contract exercises from target certification.
type Receipt struct {
	SchemaVersion int       `json:"schema_version"`
	Lane          string    `json:"lane"`
	TargetID      string    `json:"target_id"`
	TargetClass   string    `json:"target_class"`
	Provenance    string    `json:"provenance"`
	RunID         string    `json:"run_id"`
	Scenario      string    `json:"scenario"`
	Verb          string    `json:"verb"`
	Status        string    `json:"status"`
	ExitCode      int32     `json:"exit_code"`
	EvidenceRefs  []string  `json:"evidence_refs"`
	CapturedAt    time.Time `json:"captured_at"`
	LaneEligible  bool      `json:"lane_eligible"`
}

// ReattachLocal reads the run and append-only event history from the existing
// durable Bridge run path and reconstructs an explicitly local-only receipt.
// The caller must not relabel this receipt as a real target lane.
func ReattachLocal(ctx context.Context, service runs.Service, runID string) (Receipt, error) {
	if service == nil {
		return Receipt{}, errors.New("runs service is required")
	}
	run, events, err := service.Get(ctx, strings.TrimSpace(runID))
	if err != nil {
		return Receipt{}, fmt.Errorf("read durable Bridge run: %w", err)
	}
	return FromLocalRun(run, events)
}

// FromLocalRun projects a persisted run into a local fixture receipt. It
// hard-codes local provenance and lane in this constructor: callers cannot
// pass a target lane and accidentally promote a local operation.
func FromLocalRun(run runs.Run, events []runs.RunEvent) (Receipt, error) {
	if strings.TrimSpace(run.ID) == "" || strings.TrimSpace(run.NodeID) == "" {
		return Receipt{}, errors.New("durable run and target identities are required")
	}
	if !run.Status.Terminal() {
		return Receipt{}, ErrRunNotTerminal
	}
	var exitFound bool
	for _, event := range events {
		if event.RunID != run.ID {
			return Receipt{}, fmt.Errorf("event run id %q does not match durable run %q", event.RunID, run.ID)
		}
		if event.Kind == runs.EventExit {
			exitFound = true
			if event.ExitCode != run.ExitCode {
				return Receipt{}, fmt.Errorf("exit event code %d does not match durable run code %d", event.ExitCode, run.ExitCode)
			}
		}
	}
	if !exitFound {
		return Receipt{}, ErrMissingExit
	}
	if run.Status == runs.StatusPassed && run.ExitCode != 0 {
		return Receipt{}, fmt.Errorf("passed durable run has nonzero exit code %d", run.ExitCode)
	}
	if run.Status == runs.StatusFailed && run.ExitCode == 0 {
		return Receipt{}, errors.New("failed durable run has zero exit code")
	}
	if run.FinishedAt.IsZero() {
		return Receipt{}, errors.New("durable terminal run has no finish timestamp")
	}
	return Receipt{
		SchemaVersion: SchemaVersion,
		Lane:          LaneLocalContract,
		TargetID:      run.NodeID,
		TargetClass:   ClassLocalFixture,
		Provenance:    ProvenanceLocal,
		RunID:         run.ID,
		Scenario:      run.Scenario,
		Verb:          run.Verb,
		Status:        run.Status.String(),
		ExitCode:      run.ExitCode,
		EvidenceRefs:  []string{"bridge-run://" + run.ID},
		CapturedAt:    run.FinishedAt.UTC(),
		LaneEligible:  false,
	}, nil
}
