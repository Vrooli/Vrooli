package conversations_test

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
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
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"

	conversationsH "notification-hub/handlers/conversations"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
)

type refusingOwnerVerifier struct{}

func (refusingOwnerVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{}, errors.New("not an owner credential")
}

func newAskFixture(t *testing.T) (conversationsconnect.ConversationsServiceClient, *hub.Service, *database.RoutedDB) {
	t.Helper()
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetDefaultRecipient("configured-owner")
	router := mux.NewRouter()
	conversationsH.ModuleWithVerifier(service, refusingOwnerVerifier{}).Mount(router)
	// Every "valid-<run>" token is a verified run with no scopes and no grant.
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		runID, ok := strings.CutPrefix(token, "valid-")
		if !ok {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: runID, Subject: "acting-agent"}}, nil
	})
	server := httptest.NewServer(provenance.Middleware(verifier)(router))
	t.Cleanup(server.Close)
	return conversationsconnect.NewConversationsServiceClient(server.Client(), server.URL), service, db
}

// askDeadline is fixed so an idempotent replay repeats identical terms.
var askDeadline = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)

func agentAsk(token, key string) *connect.Request[v1.AskRequest] {
	req := connect.NewRequest(&v1.AskRequest{
		Question: "Continue with plan " + key + "?", AllowedAnswers: []string{"yes", "no"},
		Deadline: askDeadline, SensitivityLabel: "private", IdempotencyKey: key,
	})
	if token != "" {
		req.Header().Set(cliutil.HeaderAgentIdentityToken, token)
	}
	return req
}

func countAsks(t *testing.T, db *database.RoutedDB) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM asks").Scan(&count))
	return count
}

// A verified run without a grant raises a decision ask for the configured
// recipient and may withdraw it. It cannot answer or dismiss the ask: those
// stay owner-only. An anonymous caller cannot create an ask.
func TestAgentAsksWithoutGrantButCannotDecide(t *testing.T) {
	ctx := context.Background()
	client, service, db := newAskFixture(t)

	_, err := client.Ask(ctx, agentAsk("", "anonymous"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = client.Ask(ctx, agentAsk("forged-run", "forged"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, countAsks(t, db))

	created, err := client.Ask(ctx, agentAsk("valid-run-1", "plan-a"))
	require.NoError(t, err)
	askID := created.Msg.GetAskId()
	require.Equal(t, "configured-owner", created.Msg.GetNotification().GetRequestedBy())

	answer := connect.NewRequest(&v1.AnswerRequest{AskId: askID, Answer: "yes"})
	answer.Header().Set(cliutil.HeaderAgentIdentityToken, "valid-run-1")
	_, err = client.Answer(ctx, answer)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "an agent cannot answer its own ask")
	dismiss := connect.NewRequest(&v1.DismissRequest{AskId: askID, Note: "not needed"})
	dismiss.Header().Set(cliutil.HeaderAgentIdentityToken, "valid-run-1")
	_, err = client.Dismiss(ctx, dismiss)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "an agent cannot dismiss for the operator")
	record, err := service.GetAsk(ctx, askID)
	require.NoError(t, err)
	require.True(t, record.Open())

	withdraw := connect.NewRequest(&v1.WithdrawRequest{AskId: askID, Reason: "the plan changed"})
	withdraw.Header().Set(cliutil.HeaderAgentIdentityToken, "valid-run-1")
	withdrawn, err := client.Withdraw(ctx, withdraw)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, withdrawn.Msg.GetState())
}

// A producer over its ask cooldown is refused loudly with the retry time; the
// ask is not stored, so it is never dropped without the producer knowing.
func TestAgentAskCooldownRefusesWithRetryTime(t *testing.T) {
	ctx := context.Background()
	client, _, db := newAskFixture(t)
	_, err := client.Ask(ctx, agentAsk("valid-run-1", "first"))
	require.NoError(t, err)
	limit := hub.DefaultAskCooldown
	for i := 1; i < limit.Count; i++ {
		_, err = client.Ask(ctx, agentAsk("valid-run-1", fmt.Sprintf("next-%d", i)))
		require.NoError(t, err)
	}
	_, err = client.Ask(ctx, agentAsk("valid-run-1", "over"))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	var refusal *connect.Error
	require.ErrorAs(t, err, &refusal)
	require.NotEmpty(t, refusal.Meta().Get("Retry-After"))
	require.Contains(t, refusal.Message(), "The ask was not created")
	require.Equal(t, limit.Count, countAsks(t, db))

	_, err = client.Ask(ctx, agentAsk("valid-run-1", "first"))
	require.NoError(t, err, "an idempotent replay of a stored ask uses no budget")
	_, err = client.Ask(ctx, agentAsk("valid-run-2", "other-run"))
	require.NoError(t, err, "each run has its own ask budget")
}
