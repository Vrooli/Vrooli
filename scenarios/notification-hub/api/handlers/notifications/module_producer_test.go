package notifications_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	// A forged owner header never authenticates once a verifier is wired.
	req.Header().Set("X-Vrooli-Identity-Subject", "forged-owner")
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

// [REQ:NOTIFICA-P0-002] Any verified agent run sends and relays to the
// configured recipient without a scope or an operation grant. A caller without
// a verified agent run or owner credential is refused and nothing is stored.
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
			{name: "anonymous caller", code: connect.CodeUnauthenticated},
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

// A producer over its cooldown is refused with ResourceExhausted, the retry
// time and Retry-After; nothing is stored. Replays and other producers are not
// affected, and the producer may send again when the window passes.
func TestNotificationProducerCooldown(t *testing.T) {
	ctx := context.Background()
	clock := &steppingClock{now: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)}
	f := newProducerFixture(t, clock, map[string]agentRun{"looping": {}, "other": {}})
	f.service.SetProducerCooldown(hub.ProducerNotification, hub.CooldownLimit{Count: 2, Window: 10 * time.Minute})

	var ids []string
	for _, key := range []string{"one", "two"} {
		resp, err := f.send(ctx, "valid-looping", key)
		require.NoError(t, err)
		ids = append(ids, resp.Msg.GetNotification().GetId())
	}
	clock.advance(time.Minute)
	_, err := f.send(ctx, "valid-looping", "three")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	var refusal *connect.Error
	require.ErrorAs(t, err, &refusal)
	require.Equal(t, "540", refusal.Meta().Get("Retry-After"))
	require.Contains(t, refusal.Message(), "retry after 2026-11-11T12:10:00Z")
	require.Equal(t, 2, f.count(t), "a refused send stores nothing")

	replay, err := f.send(ctx, "valid-looping", "one")
	require.NoError(t, err, "an idempotent replay uses no budget")
	require.Equal(t, ids[0], replay.Msg.GetNotification().GetId())

	other, err := f.send(ctx, "valid-other", "other-one")
	require.NoError(t, err, "each run has its own budget")
	ids = append(ids, other.Msg.GetNotification().GetId())

	clock.advance(9 * time.Minute)
	later, err := f.send(ctx, "valid-looping", "three")
	require.NoError(t, err, "the producer may send again after the window")
	ids = append(ids, later.Msg.GetNotification().GetId())
	f.settle(t, ids...)
}
