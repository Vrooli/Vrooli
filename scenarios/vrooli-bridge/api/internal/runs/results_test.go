package runs_test

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/runs"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-P0-005] Integration (real sqlite): logs and artifact refs streamed
// from the node persist against the run id and survive being read back through
// a freshly-constructed service — proving durable, server-owned result
// collection rather than in-memory-only state.
func TestResults_LogsAndArtifactsPersist(t *testing.T) {
	d, clk := newSchemaDB(t)
	ctx := context.Background()

	// First service instance: create the run and ingest the node's events.
	repo := runs.NewSQLiteRepository(d, clk)
	svc := runs.NewService(repo, scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)))
	run, err := svc.Create(ctx, runs.CreateInput{NodeID: "n1", Scenario: "web-search", Verb: "scenario test", Args: []string{"web-search"}})
	require.NoError(t, err)

	events := []runs.RunEvent{
		{RunID: run.ID, Kind: runs.EventStatus, Sequence: 1, Status: "running"},
		{RunID: run.ID, Kind: runs.EventLog, Sequence: 2, LogChunk: "RUN web-search\n"},
		{RunID: run.ID, Kind: runs.EventLog, Sequence: 3, LogChunk: "PASS\n"},
		{RunID: run.ID, Kind: runs.EventArtifactRef, Sequence: 4, ArtifactRef: "dsh://node/n1/run/" + run.ID + "/report.json"},
		{RunID: run.ID, Kind: runs.EventExit, Sequence: 5, ExitCode: 0},
	}
	for _, ev := range events {
		accepted, err := svc.AppendEvent(ctx, ev)
		require.NoError(t, err)
		require.True(t, accepted)
	}

	// A fresh service over the SAME database (simulating a control-plane restart
	// / a different client) reads the full persisted history + the terminal run.
	svc2 := runs.NewService(runs.NewSQLiteRepository(d, clk), scheduletest.New(time.Now()))
	got, persistedEvents, err := svc2.Get(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, runs.StatusPassed, got.Status)
	require.Equal(t, []string{"dsh://node/n1/run/" + run.ID + "/report.json"}, got.ArtifactRefs)
	require.Len(t, persistedEvents, 5)

	// The log chunks are reconstructable in order.
	require.Equal(t, "RUN web-search\n", persistedEvents[1].LogChunk)
	require.Equal(t, "PASS\n", persistedEvents[2].LogChunk)
}
