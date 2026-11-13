package conversations_test

import (
	"context"
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

// Identityless callers and verified runs can raise a decision ask for the
// configured recipient. Creating or withdrawing an ask never lets a run
// answer or dismiss it: those actions stay owner-only.
func TestAgentAsksWithoutGrantButCannotDecide(t *testing.T) {
	ctx := context.Background()
	client, service, db := newAskFixture(t)

	identityless, err := client.Ask(ctx, agentAsk("", "anonymous"))
	require.NoError(t, err, "a host shell needs no identity or permission to create a decision request")
	require.NotEmpty(t, identityless.Msg.GetAskId())
	_, err = client.Ask(ctx, agentAsk("forged-run", "forged"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Equal(t, 1, countAsks(t, db))

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

type askPushCounter struct {
	mu sync.Mutex
	n  int
}

func (p *askPushCounter) Send(context.Context, hub.PushSubscription, hub.PushMessage) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.n++
	return "push", nil
}

func (p *askPushCounter) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.n
}

// Decision requests stay durable after the shared source bucket empties;
// only their device push is suppressed.
func TestAgentAskPushBudgetStoresWithoutDevicePush(t *testing.T) {
	ctx := context.Background()
	client, service, db := newAskFixture(t)
	service.SetPushBudget(hub.PushBudgetConfig{Capacity: 1, RefillInterval: time.Minute})
	require.NoError(t, service.RegisterPushSubscription(ctx, "configured-owner", hub.PushSubscription{Endpoint: "https://push.example/ask", P256DH: "key", Auth: "auth", Origin: "https://app.example"}))
	sender := &askPushCounter{}
	service.SetPushSender(sender)
	first, err := client.Ask(ctx, agentAsk("valid-run-1", "first"))
	require.NoError(t, err)
	second, err := client.Ask(ctx, agentAsk("valid-run-1", "second"))
	require.NoError(t, err, "over-budget decision requests still succeed")
	require.Equal(t, 2, countAsks(t, db), "both asks remain stored")
	require.Eventually(t, func() bool { return sender.count() == 1 }, time.Second, 10*time.Millisecond)
	_, receipts, err := service.GetForRecipient(ctx, first.Msg.GetNotification().GetId(), "configured-owner")
	require.NoError(t, err)
	require.Len(t, receipts, 1, "the first ask's device push is delivered")
	require.Eventually(t, func() bool {
		stored, _, getErr := service.GetForRecipient(ctx, second.Msg.GetNotification().GetId(), "configured-owner")
		return getErr == nil && stored.State == "suppressed"
	}, time.Second, 10*time.Millisecond)
	require.NotEmpty(t, second.Msg.GetAskId())
}

func subjectAsk(key, question string, options ...string) *connect.Request[v1.AskRequest] {
	if len(options) == 0 {
		options = []string{"clean", "wait"}
	}
	askOptions := make([]*v1.AskOption, 0, len(options))
	for _, option := range options {
		askOptions = append(askOptions, &v1.AskOption{Key: option, Label: option})
	}
	req := connect.NewRequest(&v1.AskRequest{
		Question: question, Options: askOptions, Recommended: options[0], Deadline: askDeadline,
		SensitivityLabel: "private", IdempotencyKey: key, SubjectKey: "disk-pressure",
	})
	req.Header().Set(cliutil.HeaderAgentIdentityToken, "valid-run-1")
	return req
}

// A direct ask with a subject key is one decision however often it is sent:
// the CLI's --subject-key rides on these fields.
func TestDirectAskSubjectKeyUpsertsAndWithdrawsBySubject(t *testing.T) {
	ctx := context.Background()
	client, _, db := newAskFixture(t)

	created, err := client.Ask(ctx, subjectAsk("k1", "Disk is 91% full."))
	require.NoError(t, err)
	require.Equal(t, hub.AskCreated, created.Msg.GetOutcome())
	askID := created.Msg.GetAskId()

	repeat, err := client.Ask(ctx, subjectAsk("k2", "Disk is 91% full."))
	require.NoError(t, err)
	require.Equal(t, hub.AskReasserted, repeat.Msg.GetOutcome())
	require.Equal(t, askID, repeat.Msg.GetAskId())

	updated, err := client.Ask(ctx, subjectAsk("k3", "Disk is 94% full."))
	require.NoError(t, err)
	require.Equal(t, hub.AskUpdated, updated.Msg.GetOutcome())
	require.Equal(t, askID, updated.Msg.GetAskId())
	require.Equal(t, 1, countAsks(t, db))

	read, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	ask := read.Msg.GetAsk()
	require.Equal(t, "disk-pressure", ask.GetSubjectKey())
	require.Equal(t, int32(1), ask.GetRevisionCount())
	require.Equal(t, "Disk is 94% full.", ask.GetQuestion())
	require.Len(t, ask.GetRevisions(), 1)
	require.Equal(t, "Disk is 91% full.", ask.GetRevisions()[0].GetQuestion())

	replaced, err := client.Ask(ctx, subjectAsk("k4", "Disk is 94% full.", "clean", "wait", "ignore"))
	require.NoError(t, err)
	require.Equal(t, hub.AskReplaced, replaced.Msg.GetOutcome())
	require.NotEqual(t, askID, replaced.Msg.GetAskId())
	old, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, old.Msg.GetAsk().GetState())
	require.Equal(t, replaced.Msg.GetAskId(), old.Msg.GetAsk().GetSupersededByAskId(), "the page can link to the replacing decision")

	both := connect.NewRequest(&v1.WithdrawRequest{AskId: askID, SubjectKey: "disk-pressure", Reason: "x"})
	both.Header().Set(cliutil.HeaderAgentIdentityToken, "valid-run-1")
	_, err = client.Withdraw(ctx, both)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "name the ask by id or by subject key, not both")

	withdraw := connect.NewRequest(&v1.WithdrawRequest{SubjectKey: "disk-pressure", Reason: "Disk is at 40% again."})
	withdraw.Header().Set(cliutil.HeaderAgentIdentityToken, "valid-run-1")
	closed, err := client.Withdraw(ctx, withdraw)
	require.NoError(t, err)
	require.Equal(t, replaced.Msg.GetAskId(), closed.Msg.GetAskId())
	require.Equal(t, hub.AskWithdrawn, closed.Msg.GetState())
	current, err := client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: replaced.Msg.GetAskId()}))
	require.NoError(t, err)
	require.Equal(t, "Disk is at 40% again.", current.Msg.GetAsk().GetReason())

	invalid := subjectAsk("k5", "q")
	invalid.Msg.SubjectKey = "no good"
	_, err = client.Ask(ctx, invalid)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
