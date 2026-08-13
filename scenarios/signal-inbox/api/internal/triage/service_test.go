package triage

import (
	"context"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/signals"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
)

func newTriage(t *testing.T) (*Service, signals.Service) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(signals.Schema), apidb.SchemaProviderFunc(Schema)))
	clock := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	return NewService(NewSQLiteRepository(database), clock), signals.NewService(signals.NewSQLiteRepository(database, clock), clock)
}

func capture(t *testing.T, journal signals.Service) signals.Signal {
	t.Helper()
	result, err := journal.Capture(context.Background(), signals.CaptureInput{Text: "durable triage evidence"})
	require.NoError(t, err)
	return result.Signal
}

func TestDispositionRejectsUndeclaredTransitionAndNeverChangesJournal(t *testing.T) {
	t.Log("[REQ:SIG-P0-007]")
	svc, journal := newTriage(t)
	signal := capture(t, journal)
	_, err := svc.Set(context.Background(), signal.ID, Done, nil)
	require.ErrorAs(t, err, new(ErrInvalidTransition))
	_, err = svc.Set(context.Background(), signal.ID, Dropped, nil)
	require.NoError(t, err)
	stored, err := journal.Get(context.Background(), signal.ID)
	require.NoError(t, err)
	require.Equal(t, signal.ContentHash, stored.ContentHash)
}

func TestAnnotationsAccumulateOutcomeIdentifiers(t *testing.T) {
	t.Log("[REQ:SIG-P0-006]")
	svc, journal := newTriage(t)
	signal := capture(t, journal)
	first, err := svc.Annotate(context.Background(), signal.ID, Operator, "useful for later", nil)
	require.NoError(t, err)
	_, err = svc.Annotate(context.Background(), signal.ID, Agent, "created capability", &Outcome{Kind: OutcomeScenario, TargetID: "signal-inbox"})
	require.NoError(t, err)
	_, annotations, err := svc.Get(context.Background(), signal.ID)
	require.NoError(t, err)
	require.Len(t, annotations, 2)
	require.Equal(t, first.ID, annotations[0].ID)
	require.Equal(t, "signal-inbox", annotations[1].Outcome.TargetID)
}
