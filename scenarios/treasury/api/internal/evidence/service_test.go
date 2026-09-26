package evidence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"

	"treasury/internal/authorization"
	"treasury/internal/evidence"
)

// [REQ:TRS-P0-009] A refused proposal has one self-contained terminal record;
// replay reads only that immutable row and the database rejects rewrites.
func TestRefusedAttemptIsCompleteReplayableAndImmutable(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(evidence.Schema)))
	recorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(handle))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	require.NoError(t, recorder.RecordDecision(ctx, authorization.DecisionEvidence{
		ID: "auth-refused:decision", AuthorizationID: "auth-refused", MandateID: "mandate-1",
		AgentSubject: "agent:1", Verdict: authorization.VerdictRefused,
		ViolatedConstraint: "per_transaction_cap", Detail: "250 exceeds 100",
		IdempotencyKey: "proposal-key", AmountMinor: 250, Currency: "USD", Counterparty: "vendor.example", CreatedAt: now,
	}))

	replayed, err := recorder.Replay(ctx, "auth-refused")
	require.NoError(t, err)
	require.Equal(t, "refused", replayed.Outcome)
	require.Equal(t, "policy_evaluation", replayed.Basis)
	var request map[string]any
	require.NoError(t, json.Unmarshal([]byte(replayed.RequestJSON), &request))
	require.Equal(t, "auth-refused", request["authorization_id"])
	require.Equal(t, "proposal-key", request["idempotency_key"])
	require.Equal(t, float64(250), request["amount_minor"])

	_, err = handle.ExecContext(ctx, `UPDATE spend_attempt_evidence SET outcome='settled' WHERE authorization_id='auth-refused'`)
	require.ErrorContains(t, err, "append-only")
	_, err = handle.ExecContext(ctx, `DELETE FROM spend_attempt_evidence WHERE authorization_id='auth-refused'`)
	require.ErrorContains(t, err, "append-only")

	changed := replayed
	changed.Basis = "rewritten"
	require.ErrorContains(t, evidence.NewSQLiteRecorder(handle).AppendAttempt(ctx, changed), "different immutable evidence")
}
