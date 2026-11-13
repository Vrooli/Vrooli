package notifications_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"

	notificationsH "notification-hub/handlers/notifications"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

type producerOwnerVerifier struct{}

func (producerOwnerVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{}, errors.New("unverified owner")
}

// agentRun describes the claims the fixture identity authority returns for the
// token "valid-<run>". Any other token is invalid.
type agentRun struct {
	scopes []string
	grant  *apiidentity.OperationGrant
}

type producerFixture struct {
	client  connectv1.NotificationsServiceClient
	service *hub.Service
	db      *database.RoutedDB
}

func newProducerFixture(t *testing.T, clock hub.Clock, runs map[string]agentRun) producerFixture {
	t.Helper()
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, clock, nil)
	service.SetDefaultRecipient("configured-owner")
	router := mux.NewRouter()
	notificationsH.ModuleWithVerifier(service, producerOwnerVerifier{}).Mount(router)
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		runID, ok := strings.CutPrefix(token, "valid-")
		run, known := runs[runID]
		if !ok || !known {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		meta := map[string]string{}
		if run.grant != nil {
			data, err := json.Marshal(run.grant)
			require.NoError(t, err)
			meta["operation_grant"] = string(data)
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: runID, Subject: "acting-agent", Scopes: run.scopes, Meta: meta}}, nil
	})
	server := httptest.NewServer(provenance.Middleware(verifier)(router))
	t.Cleanup(server.Close)
	return producerFixture{client: connectv1.NewNotificationsServiceClient(server.Client(), server.URL), service: service, db: db}
}

func (f producerFixture) send(ctx context.Context, token, key string) (*connect.Response[v1.SendResponse], error) {
	req := connect.NewRequest(&v1.SendRequest{Body: "fixture " + key, SensitivityLabel: "private", IdempotencyKey: key})
	if token != "" {
		req.Header().Set(cliutil.HeaderAgentIdentityToken, token)
	}
	return f.client.Send(ctx, req)
}

func (f producerFixture) count(t *testing.T) int {
	t.Helper()
	var count int
	require.NoError(t, f.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM notifications").Scan(&count))
	return count
}

// settle lets the asynchronous dispatcher finish before the fixture database
// closes. No delivery providers are wired.
func (f producerFixture) settle(t *testing.T, ids ...string) {
	t.Helper()
	for _, id := range ids {
		require.Eventually(t, func() bool {
			stored, _, err := f.service.GetForRecipient(context.Background(), id, "configured-owner")
			return err == nil && stored.State != "pending"
		}, 2*time.Second, 10*time.Millisecond)
	}
}

// [REQ:NOTIFICA-P0-002] Direct send and relay accept identityless host callers
// as well as verified agents and owners, always using the configured recipient.
func TestNotificationProducerIngress(t *testing.T) {
	runs := map[string]agentRun{
		"managed":     {scopes: []string{"notification-hub:write"}},
		"interactive": {},
		"read-only":   {scopes: []string{"notification-hub:read"}},
		"other-grant": {grant: &apiidentity.OperationGrant{ID: "team-grant", Approver: "acting-agent", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:team/research"}, ExpiresAt: time.Now().Add(time.Hour)}},
	}
	for _, method := range []string{"send", "relay"} {
		for _, tc := range []struct {
			name, token string
			code        connect.Code
		}{
			{name: "managed run with sender scope", token: "valid-managed"},
			{name: "interactive session without scope or grant", token: "valid-interactive"},
			{name: "run with only a read scope", token: "valid-read-only"},
			{name: "session holding an unrelated grant", token: "valid-other-grant"},
			{name: "identityless host caller"},
			{name: "unverified agent token", token: "forged-run", code: connect.CodeUnauthenticated},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				ctx := context.Background()
				f := newProducerFixture(t, nil, runs)
				var id, recipient string
				var err error
				if method == "send" {
					resp, sendErr := f.send(ctx, tc.token, "fixture")
					err = sendErr
					if resp != nil {
						id, recipient = resp.Msg.GetNotification().GetId(), resp.Msg.GetNotification().GetRequestedBy()
					}
				} else {
					req := connect.NewRequest(&v1.RelayRequest{PayloadBase64: base64.RawStdEncoding.EncodeToString([]byte(`{"body":"fixture","sensitivity_label":"private","idempotency_key":"fixture"}`))})
					if tc.token != "" {
						req.Header().Set(cliutil.HeaderAgentIdentityToken, tc.token)
					}
					req.Header().Set("X-Vrooli-Identity-Subject", "forged-owner")
					resp, relayErr := f.client.Relay(ctx, req)
					err = relayErr
					if resp != nil {
						id, recipient = resp.Msg.GetNotification().GetId(), resp.Msg.GetNotification().GetRequestedBy()
					}
				}
				if tc.code != 0 {
					require.Equal(t, tc.code, connect.CodeOf(err))
					require.Zero(t, f.count(t))
					return
				}
				require.NoError(t, err)
				require.NotEmpty(t, id)
				require.Equal(t, "configured-owner", recipient)
				f.settle(t, id)
			})
		}
	}
}

type steppingClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *steppingClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *steppingClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type pushCounter struct {
	mu sync.Mutex
	n  int
}

func (p *pushCounter) Send(context.Context, hub.PushSubscription, hub.PushMessage) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.n++
	return fmt.Sprintf("push-%d", p.n), nil
}

func (p *pushCounter) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.n
}

// Sends stay accepted and durable after the per-run bucket empties. Only the
// device push stops, then resumes after a token drains back into the bucket.
func TestNotificationProducerPushBudgetStoresWithoutDevicePush(t *testing.T) {
	ctx := context.Background()
	clock := &steppingClock{now: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)}
	f := newProducerFixture(t, clock, map[string]agentRun{"looping": {}, "other": {}})
	f.service.SetPushBudget(hub.PushBudgetConfig{Capacity: 2, RefillInterval: time.Minute})
	require.NoError(t, f.service.RegisterPushSubscription(ctx, "configured-owner", hub.PushSubscription{Endpoint: "https://push.example/device", P256DH: "key", Auth: "auth", Origin: "https://app.example"}))
	sender := &pushCounter{}
	f.service.SetPushSender(sender)

	var ids []string
	for _, key := range []string{"one", "two", "three"} {
		resp, err := f.send(ctx, "valid-looping", key)
		require.NoError(t, err)
		ids = append(ids, resp.Msg.GetNotification().GetId())
	}
	require.Equal(t, 3, f.count(t), "all accepted sends remain in the hub")
	f.settle(t, ids...)
	require.Eventually(t, func() bool { return sender.count() == 2 }, time.Second, 10*time.Millisecond)
	stored, _, err := f.service.GetForRecipient(ctx, ids[2], "configured-owner")
	require.NoError(t, err)
	require.Equal(t, "suppressed", stored.State, "the over-limit item remains visible in the hub")

	replay, err := f.send(ctx, "valid-looping", "one")
	require.NoError(t, err, "an idempotent replay uses no budget")
	require.Equal(t, ids[0], replay.Msg.GetNotification().GetId())

	other, err := f.send(ctx, "valid-other", "other-one")
	require.NoError(t, err, "each run has its own budget")
	f.settle(t, other.Msg.GetNotification().GetId())
	require.Eventually(t, func() bool { return sender.count() == 3 }, time.Second, 10*time.Millisecond)

	clock.advance(time.Minute)
	later, err := f.send(ctx, "valid-looping", "four")
	require.NoError(t, err, "requests remain accepted after bucket refill")
	f.settle(t, later.Msg.GetNotification().GetId())
	require.Eventually(t, func() bool { return sender.count() == 4 }, time.Second, 10*time.Millisecond)
}
