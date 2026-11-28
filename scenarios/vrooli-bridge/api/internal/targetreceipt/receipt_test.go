package targetreceipt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"vrooli-bridge/internal/runs"
)

func passingRun() (runs.Run, []runs.RunEvent) {
	finished := time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)
	run := runs.Run{ID: "run-local-1", NodeID: "local-fixture-1", Scenario: "vrooli-bridge", Verb: "scenario status", Status: runs.StatusPassed, FinishedAt: finished}
	events := []runs.RunEvent{
		{RunID: run.ID, Sequence: 1, Kind: runs.EventStatus, Status: "running"},
		{RunID: run.ID, Sequence: 2, Kind: runs.EventExit, ExitCode: 0},
	}
	return run, events
}

func TestFromLocalRunAlwaysMarksReceiptIneligibleForRealLane(t *testing.T) {
	run, events := passingRun()
	receipt, err := FromLocalRun(run, events)
	require.NoError(t, err)
	require.Equal(t, LaneLocalContract, receipt.Lane)
	require.Equal(t, ClassLocalFixture, receipt.TargetClass)
	require.Equal(t, ProvenanceLocal, receipt.Provenance)
	require.False(t, receipt.LaneEligible)
}

func TestFromLocalRunRequiresTerminalDurableExitEvidence(t *testing.T) {
	run, events := passingRun()
	run.Status = runs.StatusRunning
	_, err := FromLocalRun(run, events)
	require.ErrorIs(t, err, ErrRunNotTerminal)

	run, events = passingRun()
	_, err = FromLocalRun(run, events[:1])
	require.ErrorIs(t, err, ErrMissingExit)
}

func TestFromLocalRunRejectsMismatchedEventIdentityOrExitCode(t *testing.T) {
	run, events := passingRun()
	events[0].RunID = "other-run"
	_, err := FromLocalRun(run, events)
	require.ErrorContains(t, err, "does not match durable run")

	run, events = passingRun()
	events[1].ExitCode = 1
	_, err = FromLocalRun(run, events)
	require.ErrorContains(t, err, "does not match durable run code")
}

func TestFromLocalRunRejectsContradictoryVerdict(t *testing.T) {
	run, events := passingRun()
	run.Status = runs.StatusFailed
	_, err := FromLocalRun(run, events)
	require.ErrorContains(t, err, "failed durable run has zero exit code")

	run, events = passingRun()
	run.ExitCode = 1
	events[1].ExitCode = 1
	_, err = FromLocalRun(run, events)
	require.ErrorContains(t, err, "passed durable run has nonzero exit code")
}
