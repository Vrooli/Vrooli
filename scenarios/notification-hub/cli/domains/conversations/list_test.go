package conversations

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func TestListWithNoFlagsLeavesEveryChoiceToTheHub(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "list")
	require.NoError(t, err)
	require.Len(t, stub.lists, 1)
	req := stub.lists[0]
	require.Equal(t, v1.ListAsksRequest_ORDER_UNSPECIFIED, req.GetOrder(), "the Hub's default is newest first")
	require.Empty(t, req.GetStateFilter())
	require.Empty(t, req.GetUrgencies())
	require.Empty(t, req.GetSenders())
	require.False(t, req.GetOpenOnly())
}

func TestListSendsEveryFilterTheFlagsName(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "list",
		"--sort", "due", "--state", "answered", "--urgency", "high,critical", "--urgency", "low",
		"--sender", "Research Desk", "--sender", "Agent run", "--query", "backup", "--limit", "10", "--cursor", "c1")
	require.NoError(t, err)
	req := stub.lists[0]
	require.Equal(t, v1.ListAsksRequest_ORDER_DUE, req.GetOrder())
	require.Equal(t, "answered", req.GetStateFilter())
	require.Equal(t, []string{"high", "critical", "low"}, req.GetUrgencies())
	require.Equal(t, []string{"Research Desk", "Agent run"}, req.GetSenders())
	require.Equal(t, "backup", req.GetQuery())
	require.Equal(t, int32(10), req.GetLimit())
	require.Equal(t, "c1", req.GetCursor())
}

func TestEachSortNameAsksForItsOrder(t *testing.T) {
	for name, want := range map[string]v1.ListAsksRequest_Order{
		"attention": v1.ListAsksRequest_ORDER_ATTENTION,
		"due":       v1.ListAsksRequest_ORDER_DUE,
		"newest":    v1.ListAsksRequest_ORDER_NEWEST,
		"oldest":    v1.ListAsksRequest_ORDER_OLDEST,
		" Oldest ":  v1.ListAsksRequest_ORDER_OLDEST,
	} {
		stub := &hubStub{}
		_, _, err := runCommand(t, newTestService(t, stub), "list", "--sort", name)
		require.NoError(t, err, name)
		require.Equal(t, want, stub.lists[0].GetOrder(), name)
	}
}

func TestEveryStateTheHubFiltersByIsAccepted(t *testing.T) {
	for _, state := range []string{"all", "awaiting", "pending", "escalated", "answered", "defaulted", "expired", "withdrawn", "dismissed"} {
		stub := &hubStub{}
		_, _, err := runCommand(t, newTestService(t, stub), "list", "--state", state)
		require.NoError(t, err, state)
		require.Equal(t, state, stub.lists[0].GetStateFilter())
	}
}

func TestOpenIsTheShortcutForTheAwaitingState(t *testing.T) {
	stub, svc := newStubService(t)
	_, _, err := runCommand(t, svc, "list", "--open")
	require.NoError(t, err)
	_, _, err = runCommand(t, svc, "list", "--open", "--state", "awaiting")
	require.NoError(t, err)
	for _, req := range stub.lists {
		require.Equal(t, "awaiting", req.GetStateFilter())
	}

	_, _, err = runCommand(t, svc, "list", "--open", "--state", "answered")
	require.ErrorContains(t, err, "--open means --state awaiting")
	require.Len(t, stub.lists, 2, "a contradiction is refused before anything is sent")
}

func TestUrgencyValuesAreTidiedAndRepeatsDropped(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "list", "--urgency", " HIGH, Critical ", "--urgency", "high")
	require.NoError(t, err)
	require.Equal(t, []string{"high", "critical"}, stub.lists[0].GetUrgencies())
}

func TestASenderNameIsKeptWholeEvenWithACommaAndCollapsesBlanks(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "list", "--sender", "  Ops,  night   shift ")
	require.NoError(t, err)
	require.Equal(t, []string{"Ops, night shift"}, stub.lists[0].GetSenders())
}

func TestListRefusesAValueTheHubCannotUseAndNamesTheChoices(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"--sort", "priority"}, []string{`--sort must be one of attention, due, newest, oldest, got "priority"`}},
		{[]string{"--state", "open"}, []string{"--state must be one of all, awaiting, pending", `got "open"`}},
		{[]string{"--urgency", "high,urgent"}, []string{"--urgency must be one or more of low, normal, high, critical", `got "urgent"`}},
		{[]string{"--sender", "  "}, []string{"--sender needs a sender name", "By sender"}},
		{[]string{"--limit", "0"}, []string{"--limit must be a whole number from 1 to 200"}},
	} {
		stub := &hubStub{}
		_, _, err := runCommand(t, newTestService(t, stub), "list", tc.args...)
		require.Error(t, err, tc.args)
		for _, want := range tc.want {
			require.ErrorContains(t, err, want, tc.args)
		}
		require.Empty(t, stub.lists, "nothing is sent for %v", tc.args)
	}
}

func listPage() *v1.ListAsksResponse {
	return &v1.ListAsksResponse{
		Asks: []*v1.Ask{
			{Id: "a1", State: "escalated", Question: "Free the disk?", Sender: "Storage manager", Urgency: "critical", Deadline: "2026-11-14T18:00:00Z"},
			{Id: "a2", State: "pending", Question: "Ship it?", Sender: "Agent run", Urgency: "normal"},
			{Id: "a3", State: "answered", Question: "Rename?", Answer: "yes", Sender: "Research Desk", Urgency: "high", Deadline: "2026-11-10T00:00:00Z"},
		},
		NextCursor:    "cursor-2",
		TotalAwaiting: 2,
		Facets: &v1.AskFacets{
			Urgencies: []*v1.AskFacetCount{{Value: "critical", Count: 1}, {Value: "high", Count: 1}, {Value: "normal", Count: 1}},
			Senders:   []*v1.AskFacetCount{{Value: "Agent run", Count: 1}, {Value: "Research Desk", Count: 1}, {Value: "Storage manager", Count: 1}},
		},
	}
}

func TestListRowsTellAsksApartBySenderUrgencyAndDeadline(t *testing.T) {
	stdout, _, err := runCommand(t, newTestService(t, &hubStub{listResp: listPage()}), "list")
	require.NoError(t, err)
	for _, want := range []string{
		"a1 [escalated] Free the disk? | from Storage manager, critical, due 2026-11-14T18:00:00Z",
		"a2 [pending] Ship it? | from Agent run\n",
		"a3 [answered] Rename? -> yes | from Research Desk, high\n",
	} {
		require.Contains(t, stdout+"\n", want)
	}
	require.NotContains(t, stdout, "Agent run, normal", "the ordinary urgency is not printed on every row")
	require.NotContains(t, stdout, "due 2026-11-10", "a closed ask has no deadline to act on")
}

func TestListTextSummarisesTheFacetsAndPointsToTheNextPage(t *testing.T) {
	stdout, _, err := runCommand(t, newTestService(t, &hubStub{listResp: listPage()}), "list")
	require.NoError(t, err)
	require.Contains(t, stdout, "3 asks")
	require.Contains(t, stdout, "By urgency: critical 1, high 1, normal 1")
	require.Contains(t, stdout, "By sender: Agent run 1, Research Desk 1, Storage manager 1")
	require.Contains(t, stdout, "--cursor cursor-2")

	page := listPage()
	page.NextCursor = ""
	page.Facets = nil
	stdout, _, err = runCommand(t, newTestService(t, &hubStub{listResp: page}), "list")
	require.NoError(t, err)
	require.NotContains(t, stdout, "By urgency")
	require.NotContains(t, stdout, "--cursor", "no next page, no hint")
}

func TestListJSONCarriesTheSenderTheFacetsAndTheCursor(t *testing.T) {
	stdout, _, err := runCommand(t, newTestService(t, &hubStub{listResp: listPage()}), "list", "--json")
	require.NoError(t, err)
	var parsed struct {
		Asks []struct {
			ID     string `json:"id"`
			Sender string `json:"sender"`
		} `json:"asks"`
		NextCursor string `json:"next_cursor"`
		Facets     struct {
			Urgencies []struct {
				Value string `json:"value"`
				Count int    `json:"count"`
			} `json:"urgencies"`
			Senders []struct {
				Value string `json:"value"`
				Count int    `json:"count"`
			} `json:"senders"`
		} `json:"facets"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed), "stdout is pure JSON: %q", stdout)
	require.Equal(t, "Storage manager", parsed.Asks[0].Sender)
	require.Equal(t, "cursor-2", parsed.NextCursor)
	require.Equal(t, "critical", parsed.Facets.Urgencies[0].Value)
	require.Equal(t, 1, parsed.Facets.Urgencies[0].Count)
	require.Equal(t, "Research Desk", parsed.Facets.Senders[1].Value)
	require.False(t, strings.Contains(stdout, "By sender"), "JSON has no text summary")
}

func TestGetShowsWhoIsAsking(t *testing.T) {
	stdout, _, err := runCommand(t, newTestService(t, &hubStub{getAsk: &v1.Ask{Id: "a", State: "pending", Question: "Q?", Sender: "Research Desk"}}), "get", "a")
	require.NoError(t, err)
	require.Contains(t, stdout, "From: Research Desk")
}

func TestAskSendsTheFromNameAsTheSender(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--from", "  Research   Desk ")...)
	require.NoError(t, err)
	require.NotNil(t, stub.asks[0].Sender)
	require.Equal(t, "Research Desk", stub.asks[0].GetSender())

	stub = &hubStub{}
	_, _, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk()...)
	require.NoError(t, err)
	require.Nil(t, stub.asks[0].Sender, "without --from the Hub names the sender, or an open ask keeps its own")
}

func TestAskRefusesAFromNameTheHubWouldRefuseBeforeSendingAnything(t *testing.T) {
	for name, tc := range map[string]struct{ from, want string }{
		"empty":   {" ", "--from needs a name of 1 to 60 characters"},
		"long":    {strings.Repeat("x", 61), "--from is 61 characters"},
		"control": {"bad\x07name", "control characters"},
	} {
		stub := &hubStub{}
		_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--from", tc.from)...)
		require.ErrorContains(t, err, tc.want, name)
		require.Empty(t, stub.asks, name)
	}
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--from", strings.Repeat("é", 60))...)
	require.NoError(t, err, "60 characters, counted as characters not bytes")
}

func TestTheSenderTakesPartInTheDerivedKeySoTwoSendersAsksStayApart(t *testing.T) {
	stub, svc := newStubService(t)
	for _, from := range []string{"Research Desk", "Treasury steward"} {
		_, _, err := runCommand(t, svc, "ask", minimalAsk("--from", from)...)
		require.NoError(t, err)
	}
	require.NotEqual(t, stub.asks[0].GetIdempotencyKey(), stub.asks[1].GetIdempotencyKey())
}
