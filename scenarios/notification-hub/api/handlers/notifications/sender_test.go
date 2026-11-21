package notifications_test

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/hubsender"
	apiidentity "github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"

	notificationsH "notification-hub/handlers/notifications"
	"notification-hub/internal/identity"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

// A scenario that sends as itself holds a "<source>-notification-hub" service
// principal; the Hub reads the source off its name. No request field names a
// source, so a notice is always attributed to the sender's own scenario.

type senderTokens map[string]apiidentity.Principal

func (t senderTokens) Verify(_ context.Context, token string) (apiidentity.Principal, error) {
	principal, ok := t[token]
	if !ok {
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureInvalid, apiidentity.SourceScenarioAuthenticator)
	}
	return principal, nil
}

func senderPrincipal(name string) apiidentity.Principal {
	return apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: name, Verified: true, Source: apiidentity.SourceScenarioAuthenticator,
		Audience: hubsender.Audience, Scopes: []string{hubsender.Scope},
	}
}

func newSenderFixture(t *testing.T) producerFixture {
	t.Helper()
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hubtest.New(t, db, nil, nil)
	service.SetDefaultRecipient("configured-owner")
	tokens := senderTokens{
		"snd-treasury":   senderPrincipal("treasury-notification-hub"),
		"snd-desk":       senderPrincipal("research-desk-notification-hub"),
		"snd-wrong-aud":  {Kind: apiidentity.ActorService, Subject: "treasury-notification-hub", Verified: true, Audience: "scenario-authenticator:notification-hub-ask-reader", Scopes: []string{hubsender.Scope}},
		"snd-no-scope":   {Kind: apiidentity.ActorService, Subject: "treasury-notification-hub", Verified: true, Audience: hubsender.Audience},
		"snd-other-name": {Kind: apiidentity.ActorService, Subject: "treasury", Verified: true, Audience: hubsender.Audience, Scopes: []string{hubsender.Scope}},
		"snd-human":      {Kind: apiidentity.ActorHuman, Subject: "treasury-notification-hub", Verified: true, Audience: hubsender.Audience, Scopes: []string{hubsender.Scope}},
	}
	router := mux.NewRouter()
	notificationsH.ModuleWithIdentities(service, producerOwnerVerifier{}, identity.ServiceVerifiers{Senders: tokens}).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return producerFixture{client: connectv1.NewNotificationsServiceClient(server.Client(), server.URL), service: service, db: db}
}

func (f producerFixture) source(t *testing.T, id string) string {
	t.Helper()
	var source string
	require.NoError(t, f.db.QueryRowContext(context.Background(), "SELECT source_scenario FROM notifications WHERE id = ?", id).Scan(&source))
	return source
}

func bearerSend(f producerFixture, token, key string) (*connect.Response[v1.SendResponse], error) {
	req := connect.NewRequest(&v1.SendRequest{Body: "fixture " + key, SensitivityLabel: "private", IdempotencyKey: key})
	if token != "" {
		req.Header().Set("Authorization", "Bearer "+token)
	}
	return f.client.Send(context.Background(), req)
}

func bearerRelay(f producerFixture, token, key string) (*connect.Response[v1.RelayResponse], error) {
	payload := `{"body":"fixture ` + key + `","sensitivity_label":"private","idempotency_key":"` + key + `"}`
	req := connect.NewRequest(&v1.RelayRequest{PayloadBase64: base64.RawStdEncoding.EncodeToString([]byte(payload))})
	if token != "" {
		req.Header().Set("Authorization", "Bearer "+token)
	}
	return f.client.Relay(context.Background(), req)
}

func TestAServiceSenderSendsAndRelaysAsItsOwnScenario(t *testing.T) {
	f := newSenderFixture(t)
	for _, tc := range []struct{ token, key, source string }{
		{"snd-treasury", "treasury-send", "treasury"},
		{"snd-desk", "desk-send", "research-desk"},
	} {
		sent, err := bearerSend(f, tc.token, tc.key)
		require.NoError(t, err)
		require.Equal(t, "configured-owner", sent.Msg.GetNotification().GetRequestedBy(), "a sender reaches the configured recipient only")
		require.Equal(t, tc.source, f.source(t, sent.Msg.GetNotification().GetId()))

		relayed, err := bearerRelay(f, tc.token, tc.key+"-relay")
		require.NoError(t, err)
		require.Equal(t, tc.source, f.source(t, relayed.Msg.GetNotification().GetId()))
		f.settle(t, sent.Msg.GetNotification().GetId(), relayed.Msg.GetNotification().GetId())
	}
}

// A presented credential that is not a valid sender is refused, never downgraded
// to the identityless bucket that accepts a caller with no credential at all.
func TestAnInvalidSenderCredentialIsRefusedNotDowngradedToAnonymous(t *testing.T) {
	f := newSenderFixture(t)
	for name, tc := range map[string]struct {
		token string
		code  connect.Code
	}{
		"unknown bearer":             {"nothing", connect.CodeUnauthenticated},
		"token of another audience":  {"snd-wrong-aud", connect.CodePermissionDenied},
		"sender without its scope":   {"snd-no-scope", connect.CodePermissionDenied},
		"principal not named sender": {"snd-other-name", connect.CodePermissionDenied},
		"human token":                {"snd-human", connect.CodePermissionDenied},
	} {
		_, err := bearerSend(f, tc.token, "send-"+name)
		require.Equal(t, tc.code, connect.CodeOf(err), "send/"+name)
		_, err = bearerRelay(f, tc.token, "relay-"+name)
		require.Equal(t, tc.code, connect.CodeOf(err), "relay/"+name)
	}
	require.Zero(t, f.count(t), "a refused sender creates nothing, not even an anonymous notification")
}

func TestARefusedSenderIsToldWhatIsMissing(t *testing.T) {
	f := newSenderFixture(t)
	_, err := bearerSend(f, "snd-other-name", "k1")
	require.ErrorContains(t, err, `"treasury" is not named <source>-notification-hub`)
	_, err = bearerSend(f, "snd-no-scope", "k2")
	require.ErrorContains(t, err, `lacks scope "notification-hub:send"`)
}

func TestSenderSupportLeavesAnonymousSendsUnchanged(t *testing.T) {
	f := newSenderFixture(t)
	sent, err := bearerSend(f, "", "from-a-shell")
	require.NoError(t, err)
	require.Equal(t, "configured-owner", sent.Msg.GetNotification().GetRequestedBy())
	require.Empty(t, f.source(t, sent.Msg.GetNotification().GetId()), "an identityless send claims no source")
	f.settle(t, sent.Msg.GetNotification().GetId())
}

// The Hub without a sender verifier (isolated tests, an older wiring) keeps
// refusing any presented bearer that is not the owner's.
func TestWithoutASenderVerifierAServiceBearerStaysRefused(t *testing.T) {
	f := newProducerFixture(t, nil, nil)
	_, err := bearerSend(f, "snd-treasury", "k")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, f.count(t))
}
