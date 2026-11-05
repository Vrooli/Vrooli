package notifications_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
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

// [REQ:NOTIFICA-P0-002] Direct producers use the same verified run/scope seam
// as the standard server. The configured recipient remains the routing key.
func TestNotificationProducerIngress(t *testing.T) {
	for _, method := range []string{"send", "relay"} {
		for _, tc := range []struct {
			name, token string
			scopes      []string
			grant       *apiidentity.OperationGrant
			valid       bool
			code        connect.Code
		}{
			{name: "scoped", token: "valid", scopes: []string{"notification-hub:write"}, valid: true},
			{name: "wildcard", token: "valid", scopes: []string{"notification-hub:*"}, valid: true},
			{name: "approved recipient", token: "valid", scopes: []string{"notification-hub:write"}, valid: true, grant: senderGrant("notification.send", "notification-hub:recipient/configured-owner")},
			{name: "approved wildcard", token: "valid", scopes: []string{"notification-hub:write"}, valid: true, grant: senderGrant("notification.send", "notification-hub:recipient/*")},
			{name: "different recipient", token: "valid", scopes: []string{"notification-hub:write"}, valid: true, grant: senderGrant("notification.send", "notification-hub:recipient/other"), code: connect.CodePermissionDenied},
			{name: "different operation", token: "valid", scopes: []string{"notification-hub:write"}, valid: true, grant: senderGrant("conversation.answer", "notification-hub:recipient/configured-owner"), code: connect.CodePermissionDenied},
			{name: "missing", code: connect.CodeUnauthenticated},
			{name: "invalid", token: "invalid", code: connect.CodeUnauthenticated},
			{name: "expired", token: "expired", code: connect.CodeUnauthenticated},
			{name: "read only", token: "valid", scopes: []string{"notification-hub:read"}, valid: true, code: connect.CodePermissionDenied},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				ctx := context.Background()
				primary := databasetest.NewSQLite(t)
				require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
				db := database.NewFromPrimary(primary)
				service := hub.New(db, nil, nil)
				service.SetDefaultRecipient("configured-owner")
				router := mux.NewRouter()
				notificationsH.ModuleWithVerifier(service, producerOwnerVerifier{}).Mount(router)
				verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
					meta := map[string]string{}
					if tc.grant != nil {
						data, err := json.Marshal(tc.grant)
						require.NoError(t, err)
						meta["operation_grant"] = string(data)
					}
					return &cliutil.VerifyResult{Valid: token == "valid" && tc.valid, Claims: &cliutil.VerifiedClaims{RunID: "verified-run", Subject: "acting-agent", Scopes: tc.scopes, Meta: meta}}, nil
				})
				server := httptest.NewServer(provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(router)))
				defer server.Close()
				client := connectv1.NewNotificationsServiceClient(server.Client(), server.URL)
				var id, recipient string
				var err error
				if method == "send" {
					req := connect.NewRequest(&v1.SendRequest{Body: "fixture", SensitivityLabel: "private", IdempotencyKey: "fixture"})
					req.Header().Set(cliutil.HeaderAgentIdentityToken, tc.token)
					req.Header().Set("X-Vrooli-Identity-Subject", "forged-owner")
					resp, sendErr := client.Send(ctx, req)
					err = sendErr
					if resp != nil {
						id, recipient = resp.Msg.GetNotification().GetId(), resp.Msg.GetNotification().GetRequestedBy()
					}
				} else {
					req := connect.NewRequest(&v1.RelayRequest{PayloadBase64: base64.RawStdEncoding.EncodeToString([]byte(`{"body":"fixture","sensitivity_label":"private","idempotency_key":"fixture"}`))})
					req.Header().Set(cliutil.HeaderAgentIdentityToken, tc.token)
					req.Header().Set("X-Vrooli-Identity-Subject", "forged-owner")
					resp, relayErr := client.Relay(ctx, req)
					err = relayErr
					if resp != nil {
						id, recipient = resp.Msg.GetNotification().GetId(), resp.Msg.GetNotification().GetRequestedBy()
					}
				}
				if tc.code != 0 {
					require.Equal(t, tc.code, connect.CodeOf(err))
					var count int
					require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications").Scan(&count))
					require.Zero(t, count)
					return
				}
				require.NoError(t, err)
				require.NotEmpty(t, id)
				require.Equal(t, "configured-owner", recipient)
				stored, _, err := service.GetForRecipient(ctx, id, "configured-owner")
				require.NoError(t, err)
				require.Equal(t, id, stored.ID)
				// Let the existing asynchronous dispatcher finish before closing
				// this private fixture database. No delivery providers are wired.
				require.Eventually(t, func() bool {
					stored, _, err := service.GetForRecipient(ctx, id, "configured-owner")
					return err == nil && stored.State != "pending"
				}, time.Second, 10*time.Millisecond)
			})
		}
	}
}

func senderGrant(operation, target string) *apiidentity.OperationGrant {
	return &apiidentity.OperationGrant{ID: "sender-grant", Approver: "acting-agent", Operations: []string{operation}, Targets: []string{target}, ExpiresAt: time.Now().Add(time.Hour)}
}

type senderPairVerifier struct {
	verifyPair func() (*cliutil.VerifyPairResult, error)
}

func (v senderPairVerifier) Verify(token string) (*cliutil.VerifyResult, error) {
	return &cliutil.VerifyResult{Valid: token == "baseline", Claims: &cliutil.VerifiedClaims{RunID: "verified-run", Subject: "acting-agent"}}, nil
}

func (v senderPairVerifier) VerifyPair(context.Context, string, string) (*cliutil.VerifyPairResult, error) {
	return v.verifyPair()
}

func TestNotificationInteractiveProducerPair(t *testing.T) {
	for _, tc := range []struct {
		name      string
		target    string
		revoked   bool
		outage    bool
		expired   bool
		multi     bool
		crossRule bool
		denied    bool
	}{
		{name: "zero-scope baseline", target: "notification-hub:recipient/configured-owner"},
		{name: "multi-policy wildcard", target: "notification-hub:*", multi: true},
		{name: "wrong recipient", target: "notification-hub:recipient/other", denied: true},
		{name: "cross-rule target", target: "notification-hub:recipient/other", multi: true, crossRule: true, denied: true},
		{name: "revoked before effect", target: "notification-hub:*", revoked: true, denied: true},
		{name: "authority unavailable before effect", target: "notification-hub:*", outage: true, denied: true},
		{name: "expired proof", target: "notification-hub:*", expired: true, denied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			primary := databasetest.NewSQLite(t)
			require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
			service := hub.New(database.NewFromPrimary(primary), nil, nil)
			service.SetDefaultRecipient("configured-owner")
			router := mux.NewRouter()
			notificationsH.ModuleWithVerifier(service, producerOwnerVerifier{}).Mount(router)
			grant := senderGrant("notification.send", tc.target)
			if tc.multi {
				otherTarget := "prompt-manager:team/research"
				if tc.crossRule {
					otherTarget = "notification-hub:recipient/configured-owner"
				}
				grant.Rules = []apiidentity.OperationRule{
					{Policy: "notification-sending", Operations: []string{"notification.send"}, Targets: []string{tc.target}},
					{Policy: "team-management", Operations: []string{"team.manage"}, Targets: []string{otherTarget}},
				}
			}
			data, err := json.Marshal(grant)
			require.NoError(t, err)
			expires := time.Now().Add(time.Hour)
			if tc.expired {
				expires = time.Now().Add(-time.Minute)
			}
			calls := 0
			verifier := senderPairVerifier{verifyPair: func() (*cliutil.VerifyPairResult, error) {
				calls++
				if calls > 1 && tc.outage {
					return nil, errors.New("authority unavailable")
				}
				return &cliutil.VerifyPairResult{Valid: !(calls > 1 && tc.revoked), Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
					RunID: "verified-run", Subject: "acting-agent", Scopes: []string{"notification-hub:write"}, ExpiresAt: expires.Unix(), Meta: map[string]string{"operation_grant": string(data)},
				}}}, nil
			}}
			server := httptest.NewServer(provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(router)))
			defer server.Close()
			client := connectv1.NewNotificationsServiceClient(server.Client(), server.URL)
			req := connect.NewRequest(&v1.SendRequest{Body: "fixture", SensitivityLabel: "private", IdempotencyKey: "paired-fixture"})
			req.Header().Set(cliutil.HeaderAgentIdentityToken, "baseline")
			req.Header().Set(cliutil.HeaderAgentOperationToken, "operation")
			resp, err := client.Send(ctx, req)
			require.Equal(t, 2, calls, "reverify the live pair at the send boundary")
			if tc.denied {
				require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
				var count int
				require.NoError(t, primary.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications").Scan(&count))
				require.Zero(t, count)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "configured-owner", resp.Msg.GetNotification().GetRequestedBy())
			require.Eventually(t, func() bool {
				stored, _, err := service.GetForRecipient(ctx, resp.Msg.GetNotification().GetId(), "configured-owner")
				return err == nil && stored.State != "pending"
			}, time.Second, 10*time.Millisecond)
		})
	}
}
