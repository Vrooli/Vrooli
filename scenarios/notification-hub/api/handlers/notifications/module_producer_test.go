package notifications_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
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
			valid       bool
			code        connect.Code
		}{
			{name: "scoped", token: "valid", scopes: []string{"notification-hub:write"}, valid: true},
			{name: "wildcard", token: "valid", scopes: []string{"notification-hub:*"}, valid: true},
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
					return &cliutil.VerifyResult{Valid: token == "valid" && tc.valid, Claims: &cliutil.VerifiedClaims{RunID: "verified-run", Subject: "acting-agent", Scopes: tc.scopes}}, nil
				})
				server := httptest.NewServer(provenance.Middleware(verifier)(router))
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
