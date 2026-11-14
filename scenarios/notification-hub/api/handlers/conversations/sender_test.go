package conversations_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/hubsender"
	apiidentity "github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// A scenario that sends as itself holds a "<source>-notification-hub" service
// principal. The Hub reads the source off the principal's name: no request field
// names one, so what a sender raises is always attributed to its own scenario.

func senderPrincipal(name string) apiidentity.Principal {
	return apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: name, Verified: true, Source: apiidentity.SourceScenarioAuthenticator,
		Audience: hubsender.Audience, Scopes: []string{hubsender.Scope},
	}
}

func askSource(t *testing.T, f *readerFixture, askID string) string {
	t.Helper()
	got, err := f.client.GetAsk(context.Background(), reading("owner", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	return got.Msg.GetAsk().GetSource()
}

func TestAServiceSenderRaisesAsksAsItsOwnScenario(t *testing.T) {
	f := newReaderFixture(t)
	treasury := f.ask(t, "bearer:snd-treasury", "treasury-approve-budget")
	desk := f.ask(t, "bearer:snd-desk", "desk-approve-budget")

	require.NotEqual(t, treasury, desk)
	require.Equal(t, "treasury", askSource(t, f, treasury))
	require.Equal(t, "research-desk", askSource(t, f, desk))
}

func TestAServiceSenderReadsBackOnlyTheAsksItSent(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	mine := f.ask(t, "bearer:snd-treasury", "mine")
	theirs := f.ask(t, "bearer:snd-desk", "theirs")
	anonymous := f.ask(t, "", "shell")

	got, err := f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: mine}))
	require.NoError(t, err)
	require.Equal(t, mine, got.Msg.GetAsk().GetId())
	list, err := f.client.ListAsks(ctx, reading("bearer:snd-treasury", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{mine}, ids(list.Msg.GetAsks()))
	for _, other := range []string{theirs, anonymous} {
		_, err = f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: other}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a sender never reads another source's ask")
	}
}

// A sender token is not an ask-reader token and the reverse: each role has its own
// exclusive audience, so neither can be replayed as the other.
func TestSenderAndAskReaderTokensAreNotInterchangeable(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	event := f.eventAsk(t, "treasury", "e1")

	// A sender does not read the event asks of its source; that is the ask-reader's.
	_, err := f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: event}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	// An ask-reader token cannot send.
	_, err = f.client.Ask(ctx, reading("bearer:svc-autoheal", &v1.AskRequest{
		Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: "k",
	}))
	require.Error(t, err)
	require.NotEqual(t, connect.CodeOf(err), connect.CodeUnknown)
}

// A presented credential that is not a valid sender is refused, never downgraded
// to the identityless bucket that accepts a caller with no credential at all.
func TestAnInvalidSenderCredentialIsRefusedNotDowngradedToAnonymous(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixture(t)
	cases := map[string]connect.Code{
		"unknown bearer":             connect.CodeUnauthenticated,
		"token of another audience":  connect.CodePermissionDenied,
		"sender without its scope":   connect.CodePermissionDenied,
		"principal not named sender": connect.CodePermissionDenied,
		"human token":                connect.CodePermissionDenied,
		"unverified principal":       connect.CodePermissionDenied,
	}
	who := map[string]string{
		"unknown bearer": "bearer:nothing", "token of another audience": "bearer:snd-wrong-aud",
		"sender without its scope": "bearer:snd-no-scope", "principal not named sender": "bearer:snd-other-name",
		"human token": "bearer:snd-human", "unverified principal": "bearer:snd-unverified",
	}
	for name, code := range cases {
		_, err := f.client.Ask(ctx, reading(who[name], &v1.AskRequest{
			Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: name,
		}))
		require.Equal(t, code, connect.CodeOf(err), name)
	}
	list, err := f.client.ListAsks(ctx, reading("owner", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, list.Msg.GetAsks(), "a refused sender creates nothing, not even an anonymous ask")
}

func TestARefusedSenderIsToldWhatIsMissing(t *testing.T) {
	f := newReaderFixture(t)
	_, err := f.client.Ask(context.Background(), reading("bearer:snd-other-name", &v1.AskRequest{
		Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: "k",
	}))
	require.Error(t, err)
	require.Contains(t, err.Error(), `"treasury" is not named <source>-notification-hub`)
	_, err = f.client.Ask(context.Background(), reading("bearer:snd-no-scope", &v1.AskRequest{
		Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: "k2",
	}))
	require.Contains(t, err.Error(), `lacks scope "notification-hub:send"`)
}

func TestAnAnonymousAskIsUnchangedBySenderSupport(t *testing.T) {
	f := newReaderFixture(t)
	anonymous := f.ask(t, "", "from-a-shell")
	require.Equal(t, "alice", askSource(t, f, anonymous), "an identityless ask keeps the recipient as its source")
	run := f.ask(t, "run:valid-run-1", "from-a-run")
	require.Equal(t, "alice", askSource(t, f, run), "a run's ask keeps the recipient as its source")
}
