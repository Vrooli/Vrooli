package conversations_test

import (
	"connectrpc.com/connect"
	"context"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	cv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	"net/http/httptest"
	conversations "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
	"testing"
	"time"
)

type pendingNativeConsent struct{}

func (pendingNativeConsent) ReadConsent(_ context.Context, id string) (hub.NativeConsentStatus, error) {
	return hub.NativeConsentStatus{AuthorizationID: id, State: "pending", ReviewExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}, nil
}
func TestNativeConsentRPC_GivenExistingOwnerNotification_WhenRead_ThenReturnsNativeReviewWithoutAnswerAuthority(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, modules.AllSchemas()...))
	s := hub.New(database.NewFromPrimary(db), nil, nil)
	s.SetDefaultRecipient("alice")
	s.SetNativeConsentReader(pendingNativeConsent{})
	id := uuid.NewString()
	_, err := s.Send(ctx, hub.SendInput{RequestedBy: "alice", Title: "Native permission", Body: "Exact scope", SensitivityLabel: "private", IdempotencyKey: id, DedupeKey: id, SourceScenario: "agent-manager", Category: "authorization-requested", ContextURL: "https://agent-manager.example.test/api/v1/authorizations/" + id + "/approve"})
	require.NoError(t, err)
	router := mux.NewRouter()
	conversations.Module(s).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := cv1.NewConversationsServiceClient(server.Client(), server.URL)
	resp, err := client.ListNativeConsents(ctx, connect.NewRequest(&v1.ListNativeConsentsRequest{}))
	require.NoError(t, err)
	require.Equal(t, "no-store", resp.Header().Get("Cache-Control"))
	require.Len(t, resp.Msg.Consents, 1)
	require.Equal(t, id, resp.Msg.Consents[0].AuthorizationId)
	_, err = client.Answer(ctx, connect.NewRequest(&v1.AnswerRequest{AskId: id, Answer: "approve"}))
	require.Error(t, err)
	s.SetDefaultRecipient("bob")
	resp, err = client.ListNativeConsents(ctx, connect.NewRequest(&v1.ListNativeConsentsRequest{}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.Consents)
}
