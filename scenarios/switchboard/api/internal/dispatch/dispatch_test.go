package dispatch

import (
	"context"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"switchboard/internal/channels"
	"switchboard/internal/ingress"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
	"testing"
	"time"
)

type runner struct{ calls int }

func (r *runner) Run(context.Context, string, []string, string) (string, error) {
	r.calls++
	return "reply", nil
}

// [REQ:SWBD-P0-012] [REQ:SWBD-P0-013] [REQ:SWBD-P1-002] [REQ:SWBD-P1-003] [REQ:SWBD-P1-004]
func TestDuplicateAndAgentMessagesNeverRun(t *testing.T) {
	r := &runner{}
	p := &Processor{Ingress: ingress.New(), Runner: r, Grant: trust.Grant{Scopes: []string{"read"}}}
	e := channels.Envelope{ChannelID: "x", RemoteMessageID: "1", AuthorKind: channels.AuthorHuman}
	out, err := p.Process(context.Background(), e, trust.Owner, trust.Owner, "a", false, true)
	require.NoError(t, err)
	require.Equal(t, OutcomeAccepted, out.Outcome)
	out, err = p.Process(context.Background(), e, trust.Owner, trust.Owner, "a", false, true)
	require.NoError(t, err)
	require.Equal(t, OutcomeDuplicate, out.Outcome)
	out, err = p.Process(context.Background(), channels.Envelope{ChannelID: "x", RemoteMessageID: "2", AuthorKind: channels.AuthorAgent}, trust.Owner, trust.Owner, "a", false, true)
	require.NoError(t, err)
	require.Equal(t, OutcomeSuppressed, out.Outcome)
	require.Equal(t, 1, r.calls)
}
func TestLowerTierCannotRunOwnerScope(t *testing.T) {
	r := &runner{}
	p := &Processor{Ingress: ingress.New(), Runner: r, Grant: trust.Grant{Scopes: []string{"owner"}}}
	out, err := p.Process(context.Background(), channels.Envelope{ChannelID: "x", RemoteMessageID: "1", AuthorKind: channels.AuthorHuman}, trust.Known, trust.Known, "a", false, true)
	require.NoError(t, err)
	require.Equal(t, OutcomeRefused, out.Outcome)
	require.Zero(t, r.calls)
}

func TestProcessPersistsThreadBeforeDispatch(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(func() string { return threads.Schema() })))
	r := &runner{}
	p := &Processor{Ingress: ingress.New(), Threads: threads.NewStore(database), Runner: r, Grant: trust.Grant{Scopes: []string{"read"}}}
	e := channels.Envelope{ChannelID: "telegram", ThreadKey: "chat-1", RemoteMessageID: "remote-1", AuthorKind: channels.AuthorHuman, ReceivedAt: time.Now()}
	out, err := p.Process(context.Background(), e, trust.Owner, trust.Owner, "agent", false, true)
	require.NoError(t, err)
	require.Equal(t, OutcomeAccepted, out.Outcome)
	var count int
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM switchboard_messages`).Scan(&count))
	require.Equal(t, 1, count)
}

// [REQ:SWBD-P1-005] [REQ:SWBD-P1-006]
func TestProcessEnforcesDurableThreadTurnBudget(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(func() string { return threads.Schema() })))
	store := threads.NewStore(database)
	e := channels.Envelope{ChannelID: "telegram", ThreadKey: "budget-chat", AuthorKind: channels.AuthorHuman, ReceivedAt: time.Now()}
	thread, err := store.Upsert(context.Background(), e, false)
	require.NoError(t, err)
	_, err = database.ExecContext(context.Background(), `UPDATE switchboard_threads SET turn_budget=1 WHERE id=?`, thread.ID)
	require.NoError(t, err)
	r := &runner{}
	p := &Processor{Ingress: ingress.New(), Threads: store, Runner: r, Grant: trust.Grant{Scopes: []string{"read"}}}
	e.RemoteMessageID = "budget-1"
	first, err := p.Process(context.Background(), e, trust.Owner, trust.Owner, "a", false, true)
	require.NoError(t, err)
	require.Equal(t, OutcomeAccepted, first.Outcome)
	e.RemoteMessageID = "budget-2"
	second, err := p.Process(context.Background(), e, trust.Owner, trust.Owner, "a", false, true)
	require.NoError(t, err)
	require.Equal(t, OutcomeSuppressed, second.Outcome)
	require.Contains(t, second.Reason, "budget exhausted")
	require.Equal(t, 1, r.calls)
}

// [REQ:SWBD-P1-005] [REQ:SWBD-P1-006]
func TestProcessNotifiesOwnerOnlyOnFirstBudgetSuppression(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(func() string { return threads.Schema() })))
	store := threads.NewStore(database)
	e := channels.Envelope{ChannelID: "telegram", ThreadKey: "notify-chat", AuthorKind: channels.AuthorHuman, ReceivedAt: time.Now()}
	thread, err := store.Upsert(context.Background(), e, false)
	require.NoError(t, err)
	_, err = database.ExecContext(context.Background(), `UPDATE switchboard_threads SET turn_budget=1 WHERE id=?`, thread.ID)
	require.NoError(t, err)
	notices := 0
	p := &Processor{
		Ingress: ingress.New(), Threads: store, Runner: &runner{},
		Grant:       trust.Grant{Scopes: []string{"read"}},
		NotifyOwner: func(context.Context, channels.Envelope, string) error { notices++; return nil },
	}
	for _, id := range []string{"notify-1", "notify-2", "notify-3"} {
		e.RemoteMessageID = id
		out, err := p.Process(context.Background(), e, trust.Owner, trust.Owner, "a", false, true)
		require.NoError(t, err)
		if id == "notify-1" {
			require.Equal(t, OutcomeAccepted, out.Outcome)
		} else {
			require.Equal(t, OutcomeSuppressed, out.Outcome)
		}
	}
	require.Equal(t, 1, notices)
}
