package approval_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/evidence"
)

type evidenceController struct{ value authorization.Record }

func (c evidenceController) Approve(context.Context, string) (authorization.Record, error) {
	return c.value, nil
}

func (c evidenceController) Release(context.Context, string) (authorization.Record, error) {
	return c.value, nil
}

// [REQ:TRS-P0-009] A human decline retains the same request dimensions a
// settled attempt does, plus its resolver and approval provenance.
func TestDeclinedApprovalRecordsCompleteAttempt(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, `CREATE TABLE authorizations(id TEXT PRIMARY KEY)`)
	require.NoError(t, err)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(approval.Schema), database.SchemaProviderFunc(evidence.Schema)))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	auth := authorization.Record{ID: "auth-declined", IdempotencyKey: "proposal-key", MandateID: "mandate-1", RequestingAgent: "agent:1", AmountMinor: 375, Currency: "USD", Counterparty: "vendor.example"}
	_, err = handle.ExecContext(ctx, `INSERT INTO authorizations(id) VALUES(?)`, auth.ID)
	require.NoError(t, err)
	recorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(handle))
	service := approval.NewService(approval.NewSQLiteRepository(handle), evidenceController{value: auth}, nil, schedule.NewFake(now), recorder)
	require.NoError(t, service.Admit(ctx, authorization.ApprovalAdmission{ID: "approval-1", AuthorizationID: auth.ID, MandateID: auth.MandateID, RequestingAgent: auth.RequestingAgent, AmountMinor: auth.AmountMinor, Currency: auth.Currency, Counterparty: auth.Counterparty, ExpiresAt: now.Add(time.Hour)}))
	_, err = service.Resolve(ctx, "approval-1", approval.StatusDeclined, "operator:1")
	require.NoError(t, err)

	replayed, err := recorder.Replay(ctx, auth.ID)
	require.NoError(t, err)
	require.Equal(t, "declined", replayed.Outcome)
	require.Equal(t, "approval-1", replayed.ApprovalID)
	var request, receipt map[string]any
	require.NoError(t, json.Unmarshal([]byte(replayed.RequestJSON), &request))
	require.NoError(t, json.Unmarshal([]byte(replayed.ReceiptJSON), &receipt))
	require.Equal(t, float64(375), request["amount_minor"])
	require.Equal(t, "vendor.example", request["counterparty"])
	require.Equal(t, "operator:1", receipt["resolver"])
}

// [REQ:TRS-P0-009] Expiry is a terminal attempt, not cleanup telemetry, and
// therefore retains the same self-contained request and approval provenance.
func TestExpiredApprovalRecordsCompleteAttempt(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, `CREATE TABLE authorizations(id TEXT PRIMARY KEY)`)
	require.NoError(t, err)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(approval.Schema), database.SchemaProviderFunc(evidence.Schema)))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	auth := authorization.Record{ID: "auth-expired", IdempotencyKey: "expiry-key", MandateID: "mandate-1", RequestingAgent: "agent:1", AmountMinor: 425, Currency: "USD", Counterparty: "vendor.example"}
	_, err = handle.ExecContext(ctx, `INSERT INTO authorizations(id) VALUES(?)`, auth.ID)
	require.NoError(t, err)
	clock := schedule.NewFake(now)
	recorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(handle))
	service := approval.NewService(approval.NewSQLiteRepository(handle), evidenceController{value: auth}, nil, clock, recorder)
	require.NoError(t, service.Admit(ctx, authorization.ApprovalAdmission{ID: "approval-expired", AuthorizationID: auth.ID, MandateID: auth.MandateID, RequestingAgent: auth.RequestingAgent, AmountMinor: auth.AmountMinor, Currency: auth.Currency, Counterparty: auth.Counterparty, ExpiresAt: now.Add(time.Minute)}))
	clock.Advance(2 * time.Minute)
	_, err = service.Expire(ctx, "approval-expired")
	require.NoError(t, err)
	replayed, err := recorder.Replay(ctx, auth.ID)
	require.NoError(t, err)
	require.Equal(t, "expired", replayed.Outcome)
	require.Equal(t, "approval-expired", replayed.ApprovalID)
}
