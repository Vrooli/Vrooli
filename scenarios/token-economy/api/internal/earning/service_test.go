package earning_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"token-economy/internal/earning"
	"token-economy/internal/grants"
	"token-economy/internal/journal"
	"token-economy/internal/mints"
)

type recordingIssuer struct {
	requests []earning.GrantRequest
}

func (i *recordingIssuer) Issue(_ context.Context, request earning.GrantRequest) (earning.GrantOutcome, error) {
	i.requests = append(i.requests, request)
	return earning.GrantOutcome{ID: "grant-" + request.IdempotencyKey}, nil
}

func newService(t *testing.T, issuer earning.GrantIssuer) (earning.Service, *sql.DB) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(earning.Schema)))
	clock := schedule.NewFake(time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC))
	return earning.NewService(earning.NewSQLiteRepository(db), issuer, clock), db
}

func submissionInput(dedupKey string) earning.Input {
	return earning.Input{
		HolderID: "child:sam", TokenTypeID: "chores", AmountMinor: 8,
		Reason: "Completed the kitchen cleanup", DedupKey: dedupKey,
	}
}

// [REQ:TKE-P0-007] Operator entry and a programmatic scenario satisfy the
// same single inbound method and produce the same grant-request shape.
func TestOperatorAndProgrammaticAdaptersShareInboundMethod(t *testing.T) {
	issuer := &recordingIssuer{}
	service, _ := newService(t, issuer)

	operator, err := service.Submit(context.Background(), "operator:alex", submissionInput("shared-source-key"))
	require.NoError(t, err)
	programmatic, err := service.Submit(context.Background(), "scenario:chore-tracker", submissionInput("shared-source-key"))
	require.NoError(t, err)

	require.Len(t, issuer.requests, 2)
	require.Equal(t, operator.HolderID, programmatic.HolderID)
	require.Equal(t, operator.TokenTypeID, programmatic.TokenTypeID)
	require.Equal(t, operator.AmountMinor, programmatic.AmountMinor)
	require.Equal(t, "operator:alex", issuer.requests[0].Authorizer)
	require.Equal(t, "scenario:chore-tracker", issuer.requests[1].Authorizer)
	require.Equal(t, issuer.requests[0].Authorizer, issuer.requests[0].ActorIdentity)
	require.Equal(t, issuer.requests[1].Authorizer, issuer.requests[1].ActorIdentity)
	require.NotEqual(t, operator.ID, programmatic.ID, "dedup keys are scoped to authenticated adapter identity")
	require.NotEqual(t, issuer.requests[0].IdempotencyKey, issuer.requests[1].IdempotencyKey)
}

// [REQ:TKE-P0-007] A retry returns the first successful outcome and does not
// issue a second grant, even when the caller resubmits different payload text.
func TestReplayReturnsFirstOutcomeWithoutSecondGrant(t *testing.T) {
	issuer := &recordingIssuer{}
	service, db := newService(t, issuer)
	first, err := service.Submit(context.Background(), "scenario:chore-tracker", submissionInput("event-42"))
	require.NoError(t, err)

	retryInput := submissionInput("event-42")
	retryInput.Reason = "retry with changed text"
	retry, err := service.Submit(context.Background(), "scenario:chore-tracker", retryInput)
	require.NoError(t, err)
	require.True(t, retry.Replayed)
	require.Equal(t, first.ID, retry.ID)
	require.Equal(t, first.GrantID, retry.GrantID)
	require.Equal(t, first.SubmittedAt, retry.SubmittedAt)
	require.Len(t, issuer.requests, 1)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM earning_submissions").Scan(&count))
	require.Equal(t, 1, count)
	var summary string
	require.NoError(t, db.QueryRow("SELECT payload_summary FROM earning_submissions").Scan(&summary))
	require.NotContains(t, summary, submissionInput("unused").Reason)
	require.True(t, strings.Contains(summary, "reason_sha256="))
}

func TestListReturnsPrivacyMinimalSubmissionReceipts(t *testing.T) {
	issuer := &recordingIssuer{}
	service, _ := newService(t, issuer)
	created, err := service.Submit(context.Background(), "scenario:chore-tracker", submissionInput("event-list"))
	require.NoError(t, err)

	listed, err := service.List(context.Background())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.ID, listed[0].ID)
	require.Equal(t, created.AdapterIdentity, listed[0].AdapterIdentity)
	require.Equal(t, created.GrantID, listed[0].GrantID)
	require.Equal(t, created.PayloadSummary, listed[0].PayloadSummary)
	require.Empty(t, listed[0].Reason, "raw reason text is intentionally not retained")
	require.Empty(t, listed[0].HolderID, "holder identity is represented only in the non-reversible payload summary")
}

type realGrantIssuer struct{ service grants.Service }

func (i realGrantIssuer) Issue(ctx context.Context, request earning.GrantRequest) (earning.GrantOutcome, error) {
	grant, err := i.service.Create(ctx, grants.CreateInput{
		TokenTypeID: request.TokenTypeID, GrantSourceID: request.GrantSourceID,
		Authorizer: request.Authorizer, HolderID: request.HolderID, AmountMinor: request.AmountMinor,
		ExpiresAt: request.ExpiresAt, IdempotencyKey: request.IdempotencyKey,
		ActorIdentity: request.ActorIdentity,
	})
	return earning.GrantOutcome{ID: grant.ID}, err
}

// [REQ:TKE-P0-007] Authenticated adapter provenance survives the earning-to-
// grant translation and is stored on the resulting journal event.
func TestAdapterProvenanceReachesJournalEvent(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(mints.Schema),
		database.SchemaProviderFunc(earning.Schema),
		database.SchemaProviderFunc(grants.Schema),
		database.SchemaProviderFunc(journal.Schema),
	))
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	mintRepository := mints.NewSQLiteRepository(db)
	_, err := mintRepository.Create(ctx, mints.TokenType{
		ID: "chores", Name: "Chore tokens", Symbol: "CT", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority:    mints.MinterAuthority{TokenTypeID: "chores", Subject: "operator:alex"},
		CreatedAt:    now,
	})
	require.NoError(t, err)

	grantRepository := grants.NewSQLiteRepository(db, func(ctx context.Context, tx *sql.Tx, credit grants.Credit) error {
		_, appendErr := journal.NewTransactionalAppender(tx).Append(ctx, journal.Event{
			ID: credit.ID, TokenTypeID: credit.TokenTypeID, HolderID: credit.HolderID,
			Amount: credit.Amount, Kind: journal.EventKindCredit,
			CauseReference: credit.CauseReference, ActorIdentity: credit.ActorIdentity,
			CreatedAt: credit.CreatedAt,
		})
		return appendErr
	})
	grantService := grants.NewService(grantRepository, grants.TokenTypeReaderFunc(func(context.Context, string) (grants.TokenTypeState, error) {
		return grants.TokenTypeState{ID: "chores"}, nil
	}), nil, schedule.NewFake(now))
	service := earning.NewService(earning.NewSQLiteRepository(db), realGrantIssuer{service: grantService}, schedule.NewFake(now))

	submission, err := service.Submit(ctx, "scenario:chore-tracker", submissionInput("provenance-1"))
	require.NoError(t, err)
	events, err := journal.NewSQLiteRepository(db).Read(ctx, "child:sam", "chores")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "scenario:chore-tracker", events[0].ActorIdentity)
	require.Equal(t, "grant:"+submission.GrantID, events[0].CauseReference)
}
