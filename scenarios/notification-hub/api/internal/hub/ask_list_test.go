package hub_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"

	"notification-hub/internal/hub"
)

var listNow = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

// listAsks raises asks for alice at chosen times.
type listAsks struct {
	t       *testing.T
	service *hub.Service
	db      *database.RoutedDB
	n       int
}

func newListAsks(t *testing.T) *listAsks {
	service, db := testService(t)
	return &listAsks{t: t, service: service, db: db}
}

// raise is one ask: age is how long ago it was raised, due when it is due
// from listNow (negative for already past), and state the state it ends in.
type raise struct {
	age        time.Duration
	due        time.Duration
	noDeadline bool
	state      string
	recipient  string
	source     string
	by         string
	conv       string
	urgency    string
	sender     string
	// direct is an ask sent by a caller, not raised as an event: its source is
	// the recipient and it has no event type.
	direct bool
}

func (l *listAsks) ask(r raise) hub.AskRecord {
	l.t.Helper()
	l.n++
	created := listNow.Add(-r.age)
	l.service.SetClockForTest(fixedClock{now: created})
	spec := decisionSpec(created, fmt.Sprintf("list-%03d", l.n))
	spec.Deadline = listNow.Add(r.due)
	if r.recipient != "" {
		spec.Recipient = r.recipient
	}
	if r.source != "" {
		spec.Source = r.source
	}
	if r.direct {
		spec.Source, spec.SourceEventType = spec.Recipient, ""
	}
	spec.CreatedBy, spec.CreatedConversation = r.by, r.conv
	spec.Urgency, spec.Sender = r.urgency, r.sender
	ask, _, err := l.service.AskWithSpec(context.Background(), spec)
	require.NoError(l.t, err)
	if r.noDeadline {
		l.set(ask.ID, `deadline = ''`)
	}
	if r.state != "" {
		l.set(ask.ID, `state = '`+r.state+`'`)
	}
	return ask
}

func (l *listAsks) set(id, assignment string) {
	l.t.Helper()
	_, err := l.db.ExecContext(context.Background(), `UPDATE asks SET `+assignment+` WHERE id = ?`, id)
	require.NoError(l.t, err)
}

func askIDs(asks []hub.AskRecord) []string {
	ids := make([]string, 0, len(asks))
	for _, ask := range asks {
		ids = append(ids, ask.ID)
	}
	return ids
}

func TestCountAwaitingFollowsTheReadersAudience(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	l.ask(raise{age: 5 * time.Minute, by: "run:1", conv: "thread-1", source: "agent-manager"})
	l.ask(raise{age: 4 * time.Minute, by: "run:1", conv: "thread-1", source: "agent-manager", state: hub.AskEscalated})
	l.ask(raise{age: 3 * time.Minute, by: "run:2", conv: "thread-2", source: "other-service"})
	l.ask(raise{age: 2 * time.Minute, source: "event-service"}) // event-sourced: no verified sender
	l.ask(raise{age: time.Minute, by: "run:1", conv: "thread-1", state: hub.AskAnswered})
	l.ask(raise{age: time.Minute, recipient: "bob"})

	for name, tc := range map[string]struct {
		recipient string
		audience  hub.AskAudience
		want      int
	}{
		"the owner sees every pending and escalated ask of the recipient": {"alice", hub.OwnerAudience(), 4},
		"another recipient's owner counts only their own":                 {"bob", hub.OwnerAudience(), 1},
		"a run sees the asks it sent, not the answered one":               {"alice", hub.AskAudience{CreatedBy: "run:1"}, 2},
		"another run sees its own":                                        {"alice", hub.AskAudience{CreatedBy: "run:2"}, 1},
		"a run of the conversation sees what the conversation sent":       {"alice", hub.AskAudience{CreatedBy: "run:9", Conversation: "thread-1"}, 2},
		"a service sees its event-sourced asks":                           {"alice", hub.AskAudience{Source: "event-service"}, 1},
		"a service does not see another service's asks":                   {"alice", hub.AskAudience{Source: "nobody"}, 0},
		"a reader with no identity sees nothing":                          {"alice", hub.AskAudience{}, 0},
	} {
		got, err := l.service.CountAwaiting(ctx, tc.recipient, tc.audience)
		require.NoError(t, err, name)
		require.Equal(t, tc.want, got, name)
		page, err := l.service.ListAsksPageFor(ctx, tc.recipient, tc.audience, hub.AskListOptions{})
		require.NoError(t, err, name)
		require.Equal(t, tc.want, page.TotalAwaiting, "%s: the page carries the same count", name)
	}
	_, err := l.service.CountAwaiting(ctx, "", hub.OwnerAudience())
	require.ErrorIs(t, err, hub.ErrNotFound)

	// The count is of what awaits the reader, not of what the request filtered to.
	answered, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{StateFilter: hub.AskAnswered})
	require.NoError(t, err)
	require.Len(t, answered.Asks, 1)
	require.Equal(t, 4, answered.TotalAwaiting, "a state filter does not change the count")
	none, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{Query: "no such text anywhere"})
	require.NoError(t, err)
	require.Empty(t, none.Asks)
	require.Equal(t, 4, none.TotalAwaiting, "a search does not change the count")
	first, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{OpenOnly: true, Limit: 1})
	require.NoError(t, err)
	next, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{OpenOnly: true, Limit: 1, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Equal(t, 4, first.TotalAwaiting)
	require.Equal(t, 4, next.TotalAwaiting, "every page carries the exact count, not the rows left")
}

// sortedBy is the order a list must show, written out independently of the SQL.
// Each order is a comparison of the fields it names, ending in created_at and id.
func sortedBy(order hub.AskOrder, asks []hub.AskRecord) []string {
	urgency := func(a hub.AskRecord) int {
		return map[string]int{"critical": 0, "high": 1, "normal": 2, "low": 3}[a.Urgency]
	}
	deadline := func(a hub.AskRecord) string {
		if a.Deadline == "" {
			return "~" // after every timestamp
		}
		return a.Deadline
	}
	escalated := func(a hub.AskRecord) int {
		if a.State == hub.AskEscalated {
			return 0
		}
		return 1
	}
	// less compares on the order's leading fields; ok is false on a tie.
	less := func(a, b hub.AskRecord) (before, decided bool) {
		switch order {
		case hub.AskOrderAttention:
			if escalated(a) != escalated(b) {
				return escalated(a) < escalated(b), true
			}
			if urgency(a) != urgency(b) {
				return urgency(a) < urgency(b), true
			}
			if deadline(a) != deadline(b) {
				return deadline(a) < deadline(b), true
			}
		case hub.AskOrderDue:
			if deadline(a) != deadline(b) {
				return deadline(a) < deadline(b), true
			}
			if urgency(a) != urgency(b) {
				return urgency(a) < urgency(b), true
			}
		}
		return false, false
	}
	sorted := append([]hub.AskRecord(nil), asks...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if before, decided := less(a, b); decided {
			return before
		}
		oldestFirst := order == hub.AskOrderOldest
		if a.CreatedAt != b.CreatedAt {
			return (a.CreatedAt < b.CreatedAt) == oldestFirst
		}
		return (a.ID < b.ID) == oldestFirst
	})
	return askIDs(sorted)
}

// pageThrough reads every page of a list and returns the ids in the order read.
func (l *listAsks) pageThrough(opts hub.AskListOptions) (ids []string, last hub.AskListPage) {
	l.t.Helper()
	for pages := 0; ; pages++ {
		require.Less(l.t, pages, 50)
		page, err := l.service.ListAsksPage(context.Background(), "alice", opts)
		require.NoError(l.t, err)
		require.LessOrEqual(l.t, len(page.Asks), opts.Limit)
		ids = append(ids, askIDs(page.Asks)...)
		last = page
		if opts.Cursor = page.NextCursor; opts.Cursor == "" {
			return ids, last
		}
	}
}

// seedMixed raises asks that tie on every leading field of some order: same
// instant, same deadline, same urgency, and every state.
func (l *listAsks) seedMixed() []hub.AskRecord {
	var raised []hub.AskRecord
	for _, r := range []raise{
		{age: 5 * time.Minute, due: 3 * time.Hour, urgency: "normal", state: hub.AskEscalated},
		{age: 6 * time.Minute, due: time.Hour, urgency: "low", state: hub.AskEscalated},
		{age: 4 * time.Minute, due: 30 * time.Minute, urgency: "normal"},
		{age: 3 * time.Minute, due: 30 * time.Minute, urgency: "critical"},
		{age: 90 * time.Second, due: 45 * time.Minute, urgency: "high"},
		{age: 90 * time.Second, due: 45 * time.Minute, urgency: "high"}, // ties on everything but id
		{age: 2 * time.Minute, noDeadline: true, urgency: "high"},
		{age: time.Minute, noDeadline: true, urgency: "high"},
		{age: time.Minute, noDeadline: true, urgency: "low"},
		{age: 8 * time.Minute, due: -time.Hour, urgency: "low"}, // overdue: the earliest deadline
		{age: 7 * time.Minute, due: time.Minute, urgency: "critical", state: hub.AskAnswered},
		{age: 9 * time.Minute, due: 6 * 24 * time.Hour, urgency: "high", state: hub.AskDismissed},
		{age: 10 * time.Minute, due: 24 * time.Hour, urgency: "normal"},
	} {
		raised = append(raised, l.ask(r))
	}
	var stored []hub.AskRecord
	for _, ask := range raised {
		record, err := l.service.GetAsk(context.Background(), ask.ID)
		require.NoError(l.t, err)
		stored = append(stored, record)
	}
	return stored
}

func TestListAsksOrdersEveryWayAcrossPageBoundaries(t *testing.T) {
	l := newListAsks(t)
	all := l.seedMixed()
	var open []hub.AskRecord
	for _, ask := range all {
		if ask.State == hub.AskPending || ask.State == hub.AskEscalated {
			open = append(open, ask)
		}
	}
	for _, order := range []hub.AskOrder{"", hub.AskOrderNewest, hub.AskOrderOldest, hub.AskOrderAttention, hub.AskOrderDue} {
		effective := order
		if effective == "" {
			effective = hub.AskOrderNewest
		}
		for _, limit := range []int{1, 2, 3, 200} {
			got, _ := l.pageThrough(hub.AskListOptions{Limit: limit, Order: order})
			require.Equal(t, sortedBy(effective, all), got, "order %q limit %d: every ask once, in order", order, limit)
			got, page := l.pageThrough(hub.AskListOptions{OpenOnly: true, Limit: limit, Order: order})
			require.Equal(t, sortedBy(effective, open), got, "order %q limit %d, open only", order, limit)
			require.Equal(t, len(open), page.TotalAwaiting)
		}
	}
}

func TestListAsksAttentionLeadsWithEscalatedThenTheMostUrgent(t *testing.T) {
	l := newListAsks(t)
	// The operator's case: urgency outranks a nearer deadline, and an escalated
	// ask outranks urgency.
	dueSoonNormal := l.ask(raise{age: time.Hour, due: 2 * time.Hour, urgency: "normal"})
	dueLateHigh := l.ask(raise{age: 2 * time.Hour, due: 5 * 24 * time.Hour, urgency: "high"})
	dueLateCritical := l.ask(raise{age: 3 * time.Hour, due: 6 * 24 * time.Hour, urgency: "critical"})
	escalatedLow := l.ask(raise{age: 4 * time.Hour, due: 3 * time.Hour, urgency: "low", state: hub.AskEscalated})
	ids, _ := l.pageThrough(hub.AskListOptions{OpenOnly: true, Limit: 50, Order: hub.AskOrderAttention})
	require.Equal(t, []string{escalatedLow.ID, dueLateCritical.ID, dueLateHigh.ID, dueSoonNormal.ID}, ids)

	due, _ := l.pageThrough(hub.AskListOptions{OpenOnly: true, Limit: 50, Order: hub.AskOrderDue})
	require.Equal(t, []string{dueSoonNormal.ID, escalatedLow.ID, dueLateHigh.ID, dueLateCritical.ID}, due, "due soonest ignores urgency until deadlines tie")

	// An ask stored with no urgency counts as normal.
	unset := l.ask(raise{age: time.Minute, due: 2 * time.Hour})
	l.set(unset.ID, `urgency = ''`)
	ids, _ = l.pageThrough(hub.AskListOptions{OpenOnly: true, Limit: 50, Order: hub.AskOrderAttention})
	require.Equal(t, []string{escalatedLow.ID, dueLateCritical.ID, dueLateHigh.ID, unset.ID, dueSoonNormal.ID}, ids, "a normal ask raised later leads an earlier one with the same deadline")
}

func TestListAsksCursorHoldsWhileTheInboxChanges(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	var seeded []hub.AskRecord
	for i := range 7 {
		seeded = append(seeded, l.ask(raise{age: time.Duration(10-i) * time.Minute, due: time.Duration(i+1) * time.Hour}))
	}
	opts := hub.AskListOptions{OpenOnly: true, Limit: 3, Order: hub.AskOrderAttention}
	first, err := l.service.ListAsksPage(ctx, "alice", opts)
	require.NoError(t, err)
	require.Equal(t, askIDs(seeded[:3]), askIDs(first.Asks))

	// Meanwhile an urgent ask arrives (it sorts before the cursor), one is answered
	// before it was reached, and a later one arrives.
	l.ask(raise{age: time.Second, due: time.Minute, state: hub.AskEscalated})
	l.set(seeded[4].ID, `state = 'answered'`)
	tail := l.ask(raise{age: time.Second, due: 24 * time.Hour})

	opts.Cursor = first.NextCursor
	second, err := l.service.ListAsksPage(ctx, "alice", opts)
	require.NoError(t, err)
	require.Equal(t, []string{seeded[3].ID, seeded[5].ID, seeded[6].ID}, askIDs(second.Asks),
		"the next page continues after the last row seen: nothing repeats, the answered ask is gone")
	require.Equal(t, tail.ID, func() string {
		opts.Cursor = second.NextCursor
		third, err := l.service.ListAsksPage(ctx, "alice", opts)
		require.NoError(t, err)
		require.Len(t, third.Asks, 1)
		require.Empty(t, third.NextCursor)
		return third.Asks[0].ID
	}(), "an ask that sorts after the cursor is still reached")
}

func TestListAsksRefusesWhatItCannotRead(t *testing.T) {
	l := newListAsks(t)
	l.ask(raise{age: 2 * time.Minute, sender: "Research Desk", urgency: "high"})
	l.ask(raise{age: time.Minute, sender: "Treasury", urgency: "normal"})
	attention, err := l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{Limit: 1, Order: hub.AskOrderAttention})
	require.NoError(t, err)
	require.NotEmpty(t, attention.NextCursor)

	// A cursor belongs to the search, reader, order and filters it came from.
	for name, bad := range map[string]hub.AskListOptions{
		"an attention cursor in newest order":   {Cursor: attention.NextCursor},
		"an attention cursor in due order":      {Order: hub.AskOrderDue, Cursor: attention.NextCursor},
		"an attention cursor under a filter":    {Order: hub.AskOrderAttention, Urgencies: []string{"high"}, Cursor: attention.NextCursor},
		"an attention cursor under a sender":    {Order: hub.AskOrderAttention, Senders: []string{"Treasury"}, Cursor: attention.NextCursor},
		"an attention cursor in another search": {Order: hub.AskOrderAttention, Query: "x", Cursor: attention.NextCursor},
		"an unknown order":                      {Order: "random"},
		"an unknown urgency":                    {Urgencies: []string{"urgent"}},
		"a sender that is not text":             {Senders: []string{"a\x00b"}},
		"a sender longer than a name may be":    {Senders: []string{strings.Repeat("x", 61)}},
		"a state filter the Hub does not know":  {StateFilter: "bogus"},
		"a cursor that is not a cursor":         {Cursor: "garbage"},
		"open only against a closed state":      {OpenOnly: true, StateFilter: hub.AskAnswered},
	} {
		_, err := l.service.ListAsksPage(context.Background(), "alice", bad)
		require.ErrorIs(t, err, hub.ErrInvalidArgument, name)
	}
	// Order no longer needs an open state: every order reads every state.
	for _, order := range []hub.AskOrder{hub.AskOrderAttention, hub.AskOrderDue, hub.AskOrderOldest} {
		_, err := l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{StateFilter: hub.AskAnswered, Order: order})
		require.NoError(t, err, string(order))
	}
}

func TestListAsksFiltersByUrgencyAndSenderOnTheServer(t *testing.T) {
	l := newListAsks(t)
	desk := func(age time.Duration, urgency string) hub.AskRecord {
		return l.ask(raise{age: age, due: time.Hour, urgency: urgency, sender: "Research Desk"})
	}
	d1, d2 := desk(9*time.Minute, "normal"), desk(8*time.Minute, "normal")
	d3 := desk(7*time.Minute, "high")
	t1 := l.ask(raise{age: 6 * time.Minute, due: time.Hour, urgency: "high", sender: "Treasury"})
	t2 := l.ask(raise{age: 5 * time.Minute, due: time.Hour, urgency: "critical", sender: "Treasury"})
	closed := l.ask(raise{age: 4 * time.Minute, due: time.Hour, urgency: "high", sender: "Treasury", state: hub.AskAnswered})

	ids := func(opts hub.AskListOptions) []string {
		opts.Limit = 50
		got, _ := l.pageThrough(opts)
		return got
	}
	require.ElementsMatch(t, []string{d3.ID, t1.ID}, ids(hub.AskListOptions{OpenOnly: true, Urgencies: []string{"high"}}))
	require.ElementsMatch(t, []string{d3.ID, t1.ID, t2.ID}, ids(hub.AskListOptions{OpenOnly: true, Urgencies: []string{"high", "critical"}}), "values of one filter combine with OR")
	require.ElementsMatch(t, []string{t1.ID, t2.ID}, ids(hub.AskListOptions{OpenOnly: true, Senders: []string{"Treasury"}}))
	require.ElementsMatch(t, []string{t1.ID}, ids(hub.AskListOptions{OpenOnly: true, Senders: []string{"Treasury"}, Urgencies: []string{"high"}}), "the two filters combine with AND")
	require.ElementsMatch(t, []string{t1.ID, t2.ID, closed.ID}, ids(hub.AskListOptions{Senders: []string{"Treasury"}}), "without a state filter the answered ask is included")
	require.Empty(t, ids(hub.AskListOptions{OpenOnly: true, Senders: []string{"Nobody"}}))
	require.ElementsMatch(t, []string{d1.ID, d2.ID}, ids(hub.AskListOptions{OpenOnly: true, Query: "research desk", Urgencies: []string{"normal"}}), "search matches the sender's name")

	// Filtered pages stay exact across page boundaries.
	var got []string
	cursor := ""
	for {
		page, err := l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{OpenOnly: true, Senders: []string{"Treasury"}, Limit: 1, Cursor: cursor})
		require.NoError(t, err)
		got = append(got, askIDs(page.Asks)...)
		if cursor = page.NextCursor; cursor == "" {
			break
		}
	}
	require.ElementsMatch(t, []string{t1.ID, t2.ID}, got)
}

func TestListAsksFacetsCountEveryFilterButTheirOwn(t *testing.T) {
	l := newListAsks(t)
	for _, r := range []raise{
		{age: 9 * time.Minute, urgency: "normal", sender: "Research Desk"},
		{age: 8 * time.Minute, urgency: "normal", sender: "Research Desk"},
		{age: 7 * time.Minute, urgency: "high", sender: "Research Desk"},
		{age: 6 * time.Minute, urgency: "high", sender: "Treasury"},
		{age: 5 * time.Minute, urgency: "critical", sender: "Treasury"},
		{age: 4 * time.Minute, urgency: "low", sender: "Backup manager"},
		{age: 3 * time.Minute, urgency: "high", sender: "Treasury", state: hub.AskAnswered},
	} {
		l.ask(r)
	}
	facets := func(opts hub.AskListOptions) hub.AskFacets {
		opts.Limit = 1
		page, err := l.service.ListAsksPage(context.Background(), "alice", opts)
		require.NoError(t, err)
		return page.Facets
	}
	counts := func(list []hub.AskFacetCount) []string {
		out := make([]string, len(list))
		for i, c := range list {
			out[i] = fmt.Sprintf("%s=%d", c.Value, c.Count)
		}
		return out
	}

	open := facets(hub.AskListOptions{OpenOnly: true})
	require.Equal(t, []string{"critical=1", "high=2", "normal=2", "low=1"}, counts(open.Urgencies), "urgencies in rank order")
	require.Equal(t, []string{"Research Desk=3", "Treasury=2", "Backup manager=1"}, counts(open.Senders), "senders by count, then name")

	all := facets(hub.AskListOptions{})
	require.Equal(t, []string{"Research Desk=3", "Treasury=3", "Backup manager=1"}, counts(all.Senders), "no state filter counts the answered ask too")

	// Choosing high leaves the urgency counts alone and narrows the senders.
	high := facets(hub.AskListOptions{OpenOnly: true, Urgencies: []string{"high"}})
	require.Equal(t, counts(open.Urgencies), counts(high.Urgencies), "a facet ignores its own filter")
	require.Equal(t, []string{"Research Desk=1", "Treasury=1"}, counts(high.Senders), "the other facet applies it")

	// Choosing a sender does the same the other way round.
	treasury := facets(hub.AskListOptions{OpenOnly: true, Senders: []string{"Treasury"}})
	require.Equal(t, []string{"critical=1", "high=1"}, counts(treasury.Urgencies))
	require.Equal(t, counts(open.Senders), counts(treasury.Senders))

	// A selected value is listed even when it holds nothing.
	empty := facets(hub.AskListOptions{OpenOnly: true, Urgencies: []string{"low"}, Senders: []string{"Treasury"}})
	require.Equal(t, []string{"critical=1", "high=1", "low=0"}, counts(empty.Urgencies), "low is selected, and no Treasury ask is low")
	require.Equal(t, []string{"Backup manager=1", "Treasury=0"}, counts(empty.Senders))

	// Search narrows the counts, and every page carries them.
	searched := facets(hub.AskListOptions{OpenOnly: true, Query: "treasury"})
	require.Equal(t, []string{"Treasury=2"}, counts(searched.Senders))
	next, err := l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{OpenOnly: true, Limit: 1, Cursor: func() string {
		page, err := l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{OpenOnly: true, Limit: 1})
		require.NoError(t, err)
		return page.NextCursor
	}()})
	require.NoError(t, err)
	require.Equal(t, counts(open.Urgencies), counts(next.Facets.Urgencies), "later pages carry the facets too")
}

func TestListAsksSenderFacetIsBoundedAndKeepsSelectedSenders(t *testing.T) {
	l := newListAsks(t)
	for i := range 25 {
		l.ask(raise{age: time.Duration(i+1) * time.Minute, sender: fmt.Sprintf("Sender %02d", i)})
	}
	page, err := l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{OpenOnly: true, Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Facets.Senders, 20, "the sender facet is bounded")
	// A sender beyond the bound that the request selected is still listed.
	page, err = l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{OpenOnly: true, Limit: 1, Senders: []string{"Sender 24"}})
	require.NoError(t, err)
	var names []string
	for _, c := range page.Facets.Senders {
		names = append(names, c.Value)
	}
	require.Contains(t, names, "Sender 24")
	require.Len(t, page.Asks, 1)
}

func TestAwaitingQueriesReadTheOpenAsksIndex(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	for i := range 40 {
		state := hub.AskAnswered
		if i < 5 {
			state = ""
		}
		l.ask(raise{age: time.Duration(i) * time.Minute, due: time.Hour, state: state})
	}
	// Plan the queries over a realistic row count, as SQLite would after ANALYZE.
	_, err := l.db.ExecContext(ctx, `ANALYZE`)
	require.NoError(t, err)
	plan := func(query string, args ...any) string {
		rows, err := l.db.QueryContext(ctx, `EXPLAIN QUERY PLAN `+query, args...)
		require.NoError(t, err)
		defer rows.Close()
		var lines []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
			lines = append(lines, detail)
		}
		require.NoError(t, rows.Err())
		return strings.Join(lines, "\n")
	}

	count := plan(hub.AwaitingCountSQL, "alice")
	t.Logf("count plan:\n%s", count)
	require.Contains(t, count, "idx_asks_open_attention", "the count reads the open asks only:\n%s", count)

	for _, order := range []hub.AskOrder{hub.AskOrderAttention, hub.AskOrderDue, hub.AskOrderNewest, hub.AskOrderOldest} {
		query, args := hub.AskListSQL("awaiting", order)
		got := plan(query, args...)
		t.Logf("page plan (%s):\n%s", order, got)
		require.Contains(t, got, "idx_asks_open_attention", "%s: the page reads the open asks only:\n%s", order, got)
		require.NotContains(t, got, "SCAN a\n", "%s: no scan of every ask:\n%s", order, got)
	}
}

func TestPushCarriesTheCurrentAwaitingCount(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	service.SetClockForTest(fixedClock{now: listNow})

	first, _, err := service.AskWithSpec(ctx, decisionSpec(listNow, "push-1"))
	require.NoError(t, err)
	waitDelivered(t, service, first)
	second, _, err := service.AskWithSpec(ctx, decisionSpec(listNow, "push-2"))
	require.NoError(t, err)
	waitDelivered(t, service, second)
	// Someone else's pending ask does not count toward alice's badge.
	bobAsk := decisionSpec(listNow, "push-bob")
	bobAsk.Recipient = "bob"
	_, _, err = service.AskWithSpec(ctx, bobAsk)
	require.NoError(t, err)

	counts := func() []int {
		sender.mu.Lock()
		defer sender.mu.Unlock()
		var out []int
		for _, message := range sender.messages {
			require.NotNil(t, message.Awaiting, "a web push for the hub's own app carries the count")
			out = append(out, *message.Awaiting)
		}
		return out
	}
	require.Equal(t, []int{1, 2}, counts(), "each push carries the count when it was sent, its own ask included")

	// Once nothing awaits, a push says 0, which clears the badge, rather than omitting the key.
	_, err = db.ExecContext(ctx, `UPDATE asks SET state = 'answered'`)
	require.NoError(t, err)
	require.NoError(t, service.Process(ctx, insertPending(t, db, "alice", "an update", "public")))
	require.Eventually(t, func() bool { return pushCount(sender) == 3 }, asyncWait, 10*time.Millisecond)
	require.Equal(t, []int{1, 2, 0}, counts())
}
