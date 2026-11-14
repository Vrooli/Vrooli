package conversations_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func dueIn(d time.Duration) func(*v1.AskRequest) {
	return func(msg *v1.AskRequest) { msg.Deadline = time.Now().Add(d).UTC().Format(time.RFC3339) }
}

// The inbox's count and order, read the way each kind of reader reads them.
func TestListAsksCarriesTheAwaitingCountForEachReader(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	f.ask(t, "run:valid-run-1", "r1-a")
	f.ask(t, "run:valid-run-1", "r1-b")
	f.ask(t, "run:valid-run-2", "r2-a")
	f.ask(t, "owner", "owner-a")
	f.eventAsk(t, "vrooli-autoheal", "event-a")

	for who, want := range map[string]int32{
		"owner":               5,
		"run:valid-run-1":     2,
		"run:valid-run-2":     1,
		"bearer:svc-autoheal": 1,
		"run:valid-run-3":     0,
	} {
		list, err := f.client.ListAsks(ctx, reading(who, &v1.ListAsksRequest{Limit: 1}))
		require.NoError(t, err, who)
		require.Equal(t, want, list.Msg.GetTotalAwaiting(), "%s: the count is the reader's, whatever the page size", who)
	}
	filtered, err := f.client.ListAsks(ctx, reading("run:valid-run-1", &v1.ListAsksRequest{Query: "nothing matches this"}))
	require.NoError(t, err)
	require.Empty(t, filtered.Msg.GetAsks())
	require.Equal(t, int32(2), filtered.Msg.GetTotalAwaiting(), "a search does not change the count")
}

func TestListAsksAttentionOrderOverTheWire(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	soon := f.ask(t, "run:valid-run-1", "soon", dueIn(time.Hour))
	later := f.ask(t, "run:valid-run-1", "later", dueIn(2*time.Hour))

	newest, err := f.client.ListAsks(ctx, reading("run:valid-run-1", &v1.ListAsksRequest{OpenOnly: true}))
	require.NoError(t, err)
	require.Equal(t, []string{later, soon}, ids(newest.Msg.GetAsks()), "the default order is newest first")
	explicit, err := f.client.ListAsks(ctx, reading("run:valid-run-1", &v1.ListAsksRequest{OpenOnly: true, Order: v1.ListAsksRequest_ORDER_NEWEST}))
	require.NoError(t, err)
	require.Equal(t, ids(newest.Msg.GetAsks()), ids(explicit.Msg.GetAsks()))

	attention := &v1.ListAsksRequest{OpenOnly: true, Limit: 1, Order: v1.ListAsksRequest_ORDER_ATTENTION}
	first, err := f.client.ListAsks(ctx, reading("run:valid-run-1", attention))
	require.NoError(t, err)
	require.Equal(t, []string{soon}, ids(first.Msg.GetAsks()), "the earlier deadline leads, though it is the older ask")
	require.NotEmpty(t, first.Msg.GetNextCursor())
	attention.Cursor = first.Msg.GetNextCursor()
	second, err := f.client.ListAsks(ctx, reading("run:valid-run-1", attention))
	require.NoError(t, err)
	require.Equal(t, []string{later}, ids(second.Msg.GetAsks()))
	require.Empty(t, second.Msg.GetNextCursor())

	for name, bad := range map[string]*v1.ListAsksRequest{
		"an order this build does not know": {OpenOnly: true, Order: v1.ListAsksRequest_Order(99)},
		"a newest cursor in attention":      {OpenOnly: true, Limit: 1, Order: v1.ListAsksRequest_ORDER_ATTENTION, Cursor: newestCursor(t, f)},
		"an urgency the Hub does not know":  {OpenOnly: true, Urgencies: []string{"urgent"}},
	} {
		_, err := f.client.ListAsks(ctx, reading("run:valid-run-1", bad))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
	}
}

func suggesting(name string) func(*v1.AskRequest) {
	return func(msg *v1.AskRequest) { msg.DefaultSender = &name }
}

// A caller's tooling can suggest a sender (the label of its session); the
// operator sees it only when the request declared none.
func TestAskDefaultSenderNamesAnUndeclaredAskOverTheWire(t *testing.T) {
	f := newReaderFixture(t)
	suggested := f.ask(t, "run:valid-run-1", "suggested", suggesting("research · steward-channel"))
	declared := f.ask(t, "run:valid-run-1", "declared", from("Treasury steward"), suggesting("research · steward-channel"))
	plain := f.ask(t, "run:valid-run-1", "plain")
	for id, want := range map[string]string{suggested: "research · steward-channel", declared: "Treasury steward", plain: "Agent run"} {
		got, err := f.client.GetAsk(context.Background(), reading("run:valid-run-1", &v1.GetAskRequest{AskId: id}))
		require.NoError(t, err)
		require.Equal(t, want, got.Msg.GetAsk().GetSender())
	}
}

func newestCursor(t *testing.T, f *readerFixture) string {
	t.Helper()
	page, err := f.client.ListAsks(context.Background(), reading("run:valid-run-1", &v1.ListAsksRequest{OpenOnly: true, Limit: 1}))
	require.NoError(t, err)
	require.NotEmpty(t, page.Msg.GetNextCursor())
	return page.Msg.GetNextCursor()
}

func urgent(level string) func(*v1.AskRequest) {
	return func(msg *v1.AskRequest) { msg.Urgency = &level }
}

func from(name string) func(*v1.AskRequest) {
	return func(msg *v1.AskRequest) { msg.Sender = &name }
}

// What the operator sees: asks sorted by what needs them, named by who sent
// them, narrowed by urgency and sender, with the counts those choices show.
func TestListAsksSortsFiltersAndCountsOverTheWire(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	normalSoon := f.ask(t, "run:valid-run-1", "normal-soon", urgent("normal"), dueIn(time.Hour), from("Treasury steward"))
	highLate := f.ask(t, "run:valid-run-1", "high-late", urgent("high"), dueIn(5*24*time.Hour), from("Treasury steward"))
	criticalLate := f.ask(t, "run:valid-run-2", "critical-late", urgent("critical"), dueIn(6*24*time.Hour))
	desk := f.eventAsk(t, "vrooli-autoheal", "event-a")

	list := func(who string, req *v1.ListAsksRequest) *v1.ListAsksResponse {
		t.Helper()
		resp, err := f.client.ListAsks(ctx, reading(who, req))
		require.NoError(t, err)
		return resp.Msg
	}
	open := func(req *v1.ListAsksRequest) *v1.ListAsksRequest { req.OpenOnly = true; return req }

	require.Equal(t, []string{criticalLate, highLate, normalSoon, desk},
		ids(list("owner", open(&v1.ListAsksRequest{Order: v1.ListAsksRequest_ORDER_ATTENTION})).GetAsks()),
		"urgency outranks a nearer deadline")
	require.Equal(t, []string{normalSoon, desk, highLate, criticalLate},
		ids(list("owner", open(&v1.ListAsksRequest{Order: v1.ListAsksRequest_ORDER_DUE})).GetAsks()))
	require.Equal(t, []string{normalSoon, highLate, criticalLate, desk},
		ids(list("owner", open(&v1.ListAsksRequest{Order: v1.ListAsksRequest_ORDER_OLDEST})).GetAsks()))

	page := list("owner", open(&v1.ListAsksRequest{Order: v1.ListAsksRequest_ORDER_ATTENTION}))
	senders := map[string]string{}
	for _, ask := range page.GetAsks() {
		senders[ask.GetId()] = ask.GetSender()
	}
	require.Equal(t, map[string]string{
		normalSoon: "Treasury steward", highLate: "Treasury steward",
		criticalLate: "Agent run", desk: "Vrooli Autoheal",
	}, senders, "a declared sender is shown; the others are named from what the Hub verified")

	facet := func(counts []*v1.AskFacetCount) []string {
		out := make([]string, len(counts))
		for i, c := range counts {
			out[i] = c.GetValue() + "=" + string(rune('0'+c.GetCount()))
		}
		return out
	}
	require.Equal(t, []string{"critical=1", "high=1", "normal=2"}, facet(page.GetFacets().GetUrgencies()))
	require.Equal(t, []string{"Treasury steward=2", "Agent run=1", "Vrooli Autoheal=1"}, facet(page.GetFacets().GetSenders()))

	narrowed := list("owner", open(&v1.ListAsksRequest{Senders: []string{"Treasury steward"}, Urgencies: []string{"high", "critical"}}))
	require.Equal(t, []string{highLate}, ids(narrowed.GetAsks()))
	require.Equal(t, []string{"critical=0", "high=1", "normal=1"}, facet(narrowed.GetFacets().GetUrgencies()), "the urgency counts apply the sender filter, not their own; critical was selected, so it is listed at zero")
	require.Equal(t, []string{"Agent run=1", "Treasury steward=1"}, facet(narrowed.GetFacets().GetSenders()), "the sender counts apply the urgency filter, not their own")

	// A run's facets count only what that run may read.
	own := list("run:valid-run-1", open(&v1.ListAsksRequest{}))
	require.Equal(t, []string{"Treasury steward=2"}, facet(own.GetFacets().GetSenders()))
}
