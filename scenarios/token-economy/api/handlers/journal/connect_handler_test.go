package journal_test

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/cli-core/cliutil"

	accessH "token-economy/handlers/access"
	journalH "token-economy/handlers/journal"
	internalaccess "token-economy/internal/access"
	"token-economy/internal/journal"
	"token-economy/internal/mints"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

type minterValidator struct{}

func (minterValidator) Validate(context.Context, string) (internalaccess.Identity, error) {
	return internalaccess.Identity{Subject: "parent:alex", Scopes: []string{internalaccess.ScopeMinter}}, nil
}

type journalHandlerStore interface {
	journal.Repository
	journal.Projector
}

func newJournalHandlerStore(t *testing.T) (*sql.DB, journalHandlerStore) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db,
		database.SchemaProviderFunc(mints.Schema), database.SchemaProviderFunc(journal.Schema),
	))
	tokenTypes := mints.NewSQLiteRepository(db)
	_, err := tokenTypes.Create(context.Background(), mints.TokenType{
		ID: "chores", Name: "Chores", Symbol: "C", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority:    mints.MinterAuthority{TokenTypeID: "chores", Subject: "parent:alex"},
		CreatedAt:    time.Date(2026, 8, 19, 20, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	return db, journal.NewSQLiteRepository(db)
}

// [REQ:TKE-P0-010] [REQ:TKE-P0-011] The authenticated minter RPC appends a
// correction and persists claims verified through cli-core's identity shape.
func TestReverseEventPersistsVerifiedAgentProvenance(t *testing.T) {
	_, repository := newJournalHandlerStore(t)
	base := time.Date(2026, 8, 19, 20, 0, 0, 0, time.UTC)
	original, err := repository.Append(context.Background(), journal.Event{
		ID: "grant-credit", TokenTypeID: "chores", HolderID: "child:sam", Amount: 5,
		Kind: journal.EventKindCredit, CauseReference: "grant:mistake", ActorIdentity: "parent:alex", CreatedAt: base,
	})
	require.NoError(t, err)

	service := journal.NewService(repository, schedule.NewFake(base.Add(time.Second)))
	module := accessH.Module(nil, nil, nil, nil, nil, nil, journalH.NewConnectHandler(service, nil), minterValidator{})
	router := mux.NewRouter()
	module.Mount(router)
	handler := provenance.Middleware(provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		require.Equal(t, "signed-agent-token", token)
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
			RunID: "run-agent-reversal", Subject: "agent:household-helper", ProfileKey: "household/helper",
		}}, nil
	}))(router)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := accessconnect.NewMinterServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.ReverseEventRequest{
		OriginalEventId: original.ID, Reason: "mistaken grant", IdempotencyKey: "reverse-grant-credit",
	})
	request.Header().Set("Authorization", "Bearer minter-token")
	request.Header().Set(cliutil.HeaderAgentIdentityToken, "signed-agent-token")
	response, err := client.ReverseEvent(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, accessv1.EventKind_EVENT_KIND_REVERSAL, response.Msg.Reversal.Kind)
	require.Equal(t, accessv1.ActorKind_ACTOR_KIND_AGENT, response.Msg.Reversal.ActorKind)
	require.Equal(t, accessv1.VerificationStatus_VERIFICATION_STATUS_VERIFIED, response.Msg.Reversal.ActorVerificationStatus)
	require.Equal(t, "agent:household-helper", response.Msg.Reversal.ActorIdentity)
	require.Equal(t, "run-agent-reversal", response.Msg.Reversal.ActorRunId)
	require.Equal(t, "mistaken grant", response.Msg.Reversal.Reason)

	events, err := repository.Read(context.Background(), "child:sam", "chores")
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, original, events[0], "the original event must remain unchanged")
}
