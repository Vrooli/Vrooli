package conversations_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/identity"
)

// In VROOLI_AUTH_MODE=personal_local a loopback caller with no credential is the
// machine's owner. A caller that presents a bearer is what the bearer says, and
// nothing else: a service token reads and sends as its own scenario, and a bad
// token is refused. The API mounts the owner middleware ahead of the handlers,
// so these tests mount it too; a fixture without it cannot show the owner winning.
func personalLocal(next http.Handler) http.Handler {
	return identity.OwnerAuthenticator{
		Providers:    []authn.Provider{identity.LoopbackOwnerProvider{}},
		OwnerSubject: func(context.Context) string { return "alice" },
	}.Middleware(next)
}

func TestInPersonalLocalABearerIsItsOwnIdentityAndNeverTheLoopbackOwner(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixtureBehind(t, personalLocal)
	mine := f.ask(t, "bearer:snd-treasury", "mine")
	theirs := f.ask(t, "bearer:snd-desk", "theirs")
	event := f.eventAsk(t, "treasury", "event-1")

	// The sender is named by what the Hub verified, not "Unnamed sender".
	sent, err := f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: mine}))
	require.NoError(t, err)
	require.Equal(t, "Treasury", sent.Msg.GetAsk().GetSender())

	// A sender token reads back only what it sent.
	list, err := f.client.ListAsks(ctx, reading("bearer:snd-treasury", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{mine}, ids(list.Msg.GetAsks()))
	for _, other := range []string{theirs, event} {
		_, err = f.client.GetAsk(ctx, reading("bearer:snd-treasury", &v1.GetAskRequest{AskId: other}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	}

	// An ask-reader token reads only the event asks of its own source.
	readerList, err := f.client.ListAsks(ctx, reading("bearer:svc-autoheal", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.Empty(t, ids(readerList.Msg.GetAsks()), "an ask-reader token is not the owner either")

	// A bad bearer is refused, not downgraded to the loopback owner.
	_, err = f.client.ListAsks(ctx, reading("bearer:not-a-token", &v1.ListAsksRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = f.client.Ask(ctx, reading("bearer:not-a-token", &v1.AskRequest{
		Question: "Q?", AllowedAnswers: []string{"yes"}, SensitivityLabel: "private", IdempotencyKey: "bad",
	}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestInPersonalLocalACallerWithNoCredentialIsTheLoopbackOwner(t *testing.T) {
	ctx := context.Background()
	f := newReaderFixtureBehind(t, personalLocal)
	first := f.ask(t, "bearer:snd-treasury", "one")
	second := f.ask(t, "bearer:snd-desk", "two")
	list, err := f.client.ListAsks(ctx, reading("", &v1.ListAsksRequest{}))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{first, second}, ids(list.Msg.GetAsks()), "the contract of the explicit local mode: the machine's own user reads everything")
}
