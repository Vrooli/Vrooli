package conversations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliutil"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

// labelled is a sender source that always resolves to label, and counts its calls.
func labelled(label string, calls *atomic.Int32) senderSource {
	return func(context.Context) (string, error) {
		calls.Add(1)
		return label, nil
	}
}

func askWithSource(t *testing.T, source senderSource, args ...string) *hubStub {
	t.Helper()
	stub := &hubStub{}
	svc := newTestService(t, stub)
	svc.senderSource = source
	_, _, err := runCommand(t, svc, "ask", args...)
	require.NoError(t, err)
	require.Len(t, stub.asks, 1)
	return stub
}

func TestAnAskWithNoFromIsSentFromTheResolvedSessionLabel(t *testing.T) {
	var calls atomic.Int32
	stub := askWithSource(t, labelled("research · steward-channel-ux-tooling", &calls), minimalAsk()...)
	require.Equal(t, "research · steward-channel-ux-tooling", stub.asks[0].GetDefaultSender())
	require.Nil(t, stub.asks[0].Sender, "the label is a default, not a declared sender: the Hub uses it only to name a new ask")
}

func TestFromWinsOverTheResolvedLabelAndTheLabelIsNotEvenLookedUp(t *testing.T) {
	var calls atomic.Int32
	stub := askWithSource(t, labelled("session label", &calls), minimalAsk("--from", "Treasury steward")...)
	require.Equal(t, "Treasury steward", stub.asks[0].GetSender())
	require.Nil(t, stub.asks[0].DefaultSender, "a declared name leaves no room for a default")
	require.Zero(t, calls.Load(), "an explicit name needs no lookup, and costs no time")
}

func TestAnAskIsSentWithoutADefaultWhenTheLookupFailsAndNothingIsSaid(t *testing.T) {
	failing := func(context.Context) (string, error) { return "", errors.New("agent-manager is down") }
	stub := &hubStub{}
	svc := newTestService(t, stub)
	svc.senderSource = failing
	stdout, stderr, err := runCommand(t, svc, "ask", minimalAsk()...)
	require.NoError(t, err)
	require.Nil(t, stub.asks[0].Sender)
	require.Nil(t, stub.asks[0].DefaultSender, "the Hub names the sender itself")
	require.NotContains(t, stdout+stderr, "agent-manager is down", "a convenience that fails is silent")
	require.NotContains(t, stdout+stderr, "Warning")
}

func TestAnAskIsNeverHeldUpLongerThanTheBudgetByASlowLookup(t *testing.T) {
	restore := senderBudget
	senderBudget = 50 * time.Millisecond
	t.Cleanup(func() { senderBudget = restore })
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	slow := func(context.Context) (string, error) {
		<-release // ignores the context, as a port lookup that shells out does
		return "too late", nil
	}
	stub := &hubStub{}
	svc := newTestService(t, stub)
	svc.senderSource = slow
	started := time.Now()
	_, _, err := runCommand(t, svc, "ask", minimalAsk()...)
	require.NoError(t, err)
	require.Less(t, time.Since(started), 2*time.Second)
	require.Nil(t, stub.asks[0].DefaultSender, "an answer that arrives after the budget is not used")
}

func TestTheLookupRunsOncePerProcess(t *testing.T) {
	var calls atomic.Int32
	stub := &hubStub{}
	svc := newTestService(t, stub)
	svc.senderSource = labelled("Backup watcher", &calls)
	for range 3 {
		_, _, err := runCommand(t, svc, "ask", minimalAsk("--idempotency-key", "k")...)
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), calls.Load())
	for _, ask := range stub.asks {
		require.Equal(t, "Backup watcher", ask.GetDefaultSender())
	}
}

func TestAProcessWithNoAgentIdentityHasNoDefaultAndMakesNoRead(t *testing.T) {
	// No inherited token and no broker socket in this temp dir: no credential.
	t.Setenv(cliutil.EnvIdentityToken, "")
	t.Setenv("TMPDIR", t.TempDir())
	_, err := senderOfThisRun(context.Background())
	require.ErrorContains(t, err, "no agent identity")
}

func TestAResolvedLabelIsRepairedNotRefused(t *testing.T) {
	for name, tc := range map[string]struct{ label, want string }{
		"blanks collapsed":        {"  research \t steward  ", "research steward"},
		"control characters gone": {"bad\x07na\x00me\x1b", "badname"},
		"newline is a blank":      {"two\nlines", "two lines"},
		"invalid utf-8 dropped":   {"ok\xffgo", "okgo"},
		"empty stays empty":       {" \x07 ", ""},
	} {
		require.Equal(t, tc.want, tidySender(tc.label), name)
	}

	long := strings.Repeat("é", 80)
	got := tidySender(long)
	require.Equal(t, strings.Repeat("é", 60), got, "cut at 60 characters, on a character boundary")
	require.True(t, strings.HasSuffix(tidySender(strings.Repeat("a", 59)+" "+strings.Repeat("b", 10)), "a"), "no trailing blank after the cut")

	var calls atomic.Int32
	stub := askWithSource(t, labelled(long, &calls), minimalAsk()...)
	require.Equal(t, strings.Repeat("é", 60), stub.asks[0].GetDefaultSender(), "the ask carries a default the Hub accepts")
}

func TestOneRoleKeepsOneSenderAcrossDays(t *testing.T) {
	for label, want := range map[string]string{
		"research · steward-channel-ux-tooling-20261114":   "research · steward-channel-ux-tooling",
		"research · steward-channel-ux-tooling-20261113-3": "research · steward-channel-ux-tooling",
		"research · steward-test-o8-20261112-2":            "research · steward-test-o8",
		"research · steward-free-disk-root-20261113-r2-4":  "research · steward-free-disk-root",
		"research · steward-free-host-20261114":            "research · steward-free-host",
		"Portfolio steward free-host-20261114T0401":        "Portfolio steward free-host",
		"Portfolio steward free-host-20261114T040105Z-2":   "Portfolio steward free-host",
		"Treasury steward":        "Treasury steward",
		"research · steward-v2":   "research · steward-v2",
		"release-2026":            "release-2026",
		"nightly-20261114 backup": "nightly-20261114 backup",
		"research · steward-channel-ux-tooling-202611140001": "research · steward-channel-ux-tooling-202611140001",
		"  research · steward-free-host-20261114  ":          "research · steward-free-host",
	} {
		require.Equal(t, want, roleOfSessionLabel(label), label)
	}
	require.Equal(t, roleOfSessionLabel("research · steward-channel-ux-tooling-20261113-3"), roleOfSessionLabel("research · steward-channel-ux-tooling-20261114"),
		"two days of one role are one sender")
}

func TestAGenericOrEmptySessionLabelIsNoLabel(t *testing.T) {
	for _, label := range []string{"This machine", "this machine", "  This   Machine ", "", "  ", "\x07", "-20261114"} {
		require.Empty(t, roleOfSessionLabel(label), "%q", label)
	}
}

// chain runs senderOfRun with a run and a session label, and reports what the
// session read was asked.
func chain(run runRecord, runErr error, sessionLabel string, sessionErr error) (name string, sessionRead string, err error) {
	read := func(context.Context, string) (runRecord, error) { return run, runErr }
	sessions := func(_ context.Context, req *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error) {
		sessionRead = req.Msg.GetId()
		if sessionErr != nil {
			return nil, sessionErr
		}
		return connect.NewResponse(&sessionsv1.GetResponse{Session: &sessionsv1.Session{Id: req.Msg.GetId(), DisplayLabel: sessionLabel}}), nil
	}
	name, err = senderOfRun(context.Background(), "run-1", read, sessions)
	return name, sessionRead, err
}

func TestTheSenderIsTheSessionLabelThenTheRunsOwnHandSetLabel(t *testing.T) {
	// A named session wins, and is read by the id the run names.
	name, read, err := chain(runRecord{SessionID: "s-9", Label: "Portfolio steward free-host-20261114T0401", LabelSource: "manual"}, nil, "research · steward-free-host-20261114", nil)
	require.NoError(t, err)
	require.Equal(t, "research · steward-free-host", name)
	require.Equal(t, "s-9", read)

	// A generic session label falls through to a label somebody set on the run.
	name, _, err = chain(runRecord{SessionID: "s-9", Label: "Portfolio steward free-host-20261114T0401", LabelSource: "manual"}, nil, "This machine", nil)
	require.NoError(t, err)
	require.Equal(t, "Portfolio steward free-host", name, "the time of day goes with the date")

	// So does a session that cannot be read, or a run with no session.
	name, _, err = chain(runRecord{SessionID: "s-9", Label: "Backup watcher", LabelSource: "manual"}, nil, "", connect.NewError(connect.CodeNotFound, nil))
	require.NoError(t, err)
	require.Equal(t, "Backup watcher", name)
	name, read, err = chain(runRecord{Label: "Backup watcher", LabelSource: "manual"}, nil, "unused", nil)
	require.NoError(t, err)
	require.Equal(t, "Backup watcher", name)
	require.Empty(t, read, "no session id, no session read")
}

func TestTheOpaqueLabelsAgentManagerMakesUpAreNoName(t *testing.T) {
	// The run of an attached Claude Code session in a UI Web Console session: its
	// own label is "claude-code session <hash>" (label_source harness), and its
	// session is "This machine".
	for _, source := range []string{"harness", "placeholder", ""} {
		_, _, err := chain(runRecord{SessionID: "s-9", Label: "claude-code session claude-code-2a6bdf29f3f97818", LabelSource: source}, nil, "This machine", nil)
		require.ErrorContains(t, err, "no label that names an agent", source)
	}
}

func TestARunThatCannotBeReadHasNoSender(t *testing.T) {
	_, read, err := chain(runRecord{}, errors.New("404"), "x", nil)
	require.ErrorContains(t, err, "404")
	require.Empty(t, read)
}

func TestTheRunReadTakesTheSessionIdAndLabelFromAgentManagersOpenRunRoute(t *testing.T) {
	var path string
	body := `{"run":{"id":"r1","web_console_session_id":"sess-1","label":"Portfolio steward","label_source":"manual"}}`
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		require.Empty(t, r.Header.Get(cliutil.HeaderAgentIdentityToken), "the run read needs no credential and is sent none")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)

	run, err := readRun(context.Background(), "r1")
	require.NoError(t, err)
	require.Equal(t, runRecord{SessionID: "sess-1", Label: "Portfolio steward", LabelSource: "manual"}, run)
	require.Equal(t, "/api/v1/runs/r1", path)

	body = `{"run":{"webConsoleSessionId":"sess-2","label":"x","labelSource":"harness"}}`
	run, err = readRun(context.Background(), "r1")
	require.NoError(t, err)
	require.Equal(t, runRecord{SessionID: "sess-2", Label: "x", LabelSource: "harness"}, run, "the protojson spelling is read too")

	body = `{"run":{"id":"r1"}}`
	run, err = readRun(context.Background(), "r1")
	require.NoError(t, err)
	require.Equal(t, runRecord{}, run, "a run with nothing to name it is not an error here")

	status, body = http.StatusNotFound, `not found`
	_, err = readRun(context.Background(), "r1")
	require.ErrorContains(t, err, "404")

	status, body = http.StatusOK, `{not json`
	_, err = readRun(context.Background(), "r1")
	require.Error(t, err)
}

func TestAControlCharacterInTheLabelNeverReachesTheHub(t *testing.T) {
	var calls atomic.Int32
	stub := askWithSource(t, labelled("Research\x07 Desk", &calls), minimalAsk()...)
	require.Equal(t, "Research Desk", stub.asks[0].GetDefaultSender())
}

func TestTheResolvedDefaultTakesNoPartInTheDerivedIdempotencyKey(t *testing.T) {
	var calls atomic.Int32
	failing := func(context.Context) (string, error) { return "", errors.New("down") }
	withLabel := askWithSource(t, labelled("Research Desk", &calls), minimalAsk()...)
	without := askWithSource(t, failing, minimalAsk()...)
	require.Equal(t, "Research Desk", withLabel.asks[0].GetDefaultSender())
	require.Nil(t, without.asks[0].DefaultSender)
	require.Equal(t, withLabel.asks[0].GetIdempotencyKey(), without.asks[0].GetIdempotencyKey(),
		"a retry that resolved the label once and failed to the next time is the same request, not a new ask")
}

func TestARevisionNeverRenamesFromTheDefaultSoTheKeptLineStaysTrue(t *testing.T) {
	var calls atomic.Int32
	stub := &hubStub{askResp: askUpdated()}
	svc := newTestService(t, stub)
	svc.senderSource = labelled("Research Desk", &calls)
	_, stderr, err := runCommand(t, svc, "ask", "--subject-key", "release", "--technical-details", "Run 42.")
	require.NoError(t, err)
	require.Nil(t, stub.asks[0].Sender, "a default is not a declared sender, so a revision declares none")
	require.Equal(t, "Research Desk", stub.asks[0].GetDefaultSender(), "the Hub uses it only if this turns out to create the ask")
	require.Contains(t, stderr, "the sender", "the open ask keeps its sender, and the line says so")

	stub = &hubStub{askResp: askUpdated()}
	svc = newTestService(t, stub)
	svc.senderSource = labelled("Research Desk", &calls)
	_, stderr, err = runCommand(t, svc, "ask", "--subject-key", "release", "--technical-details", "Run 42.", "--from", "Release bot")
	require.NoError(t, err)
	require.NotContains(t, stderr, "the sender", "a declared --from renames, so no sender was kept")
	require.Nil(t, stub.asks[0].DefaultSender)
}

func TestDryRunSaysWhoTheOperatorWouldSeeAndWhereTheNameCameFrom(t *testing.T) {
	for source, want := range map[string]string{
		"declared":  "Sender the operator would see: Release bot (your --from)",
		"default":   "Sender the operator would see: Release bot (your Web Console session label)",
		"kept":      "Sender the operator would see: Release bot (kept from the open ask)",
		"service":   "Sender the operator would see: Release bot (named by the Hub from the service that is asking)",
		"agent_run": "Sender the operator would see: Release bot (named by the Hub: no --from, and no session label was found)",
		"unnamed":   "Sender the operator would see: Release bot (named by the Hub: it could not tell who is asking)",
		"future":    "Sender the operator would see: Release bot\n",
	} {
		stub := &hubStub{checkResp: &v1.CheckAskResponse{PushTitle: "t", PushBody: "b", Sender: "Release bot", SenderSource: source}}
		stdout, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--dry-run")...)
		require.NoError(t, err, source)
		require.Contains(t, stdout, want, source)
	}

	stub := &hubStub{checkResp: &v1.CheckAskResponse{PushTitle: "t", PushBody: "b"}}
	stdout, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--dry-run")...)
	require.NoError(t, err)
	require.NotContains(t, stdout, "Sender the operator would see", "a Hub that names none is not guessed for")
}

func TestDryRunJSONCarriesTheHubsSenderAndItsSource(t *testing.T) {
	stub := &hubStub{checkResp: &v1.CheckAskResponse{PushTitle: "t", PushBody: "b", Sender: "Agent run", SenderSource: "agent_run"}}
	stdout, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--dry-run", "--json")...)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed))
	require.Equal(t, "Agent run", parsed["sender"])
	require.Equal(t, "agent_run", parsed["sender_source"])
}

func TestDryRunAsksTheHubAboutTheSameSenderARealSendWouldCarry(t *testing.T) {
	var calls atomic.Int32
	stub := &hubStub{}
	svc := newTestService(t, stub)
	svc.senderSource = labelled("Research Desk", &calls)
	_, _, err := runCommand(t, svc, "ask", minimalAsk("--dry-run")...)
	require.NoError(t, err)
	require.Equal(t, "Research Desk", stub.checks[0].GetAsk().GetDefaultSender(), "the check sees the default a send would carry")

	stub = &hubStub{}
	_, _, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk("--dry-run", "--from", "Release bot")...)
	require.NoError(t, err)
	require.Equal(t, "Release bot", stub.checks[0].GetAsk().GetSender())
}

func askUpdated() *v1.AskResponse { return &v1.AskResponse{AskId: "ask-1", Outcome: "updated"} }
