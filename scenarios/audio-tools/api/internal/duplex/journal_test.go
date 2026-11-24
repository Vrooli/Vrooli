package duplex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	_ "modernc.org/sqlite"
)

func openJournalDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSQLJournalSurvivesRestartAndRetainsUnknownUsage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplex.sqlite")
	ctx := context.Background()
	db1 := openJournalDB(t, path)
	journal, err := NewSQLJournal(db1)
	require.NoError(t, err)
	op := JournalOperation{OperationID: "op-1", OwnerKey: "owner-1", ProviderID: "provider", ModelID: "model", BudgetMicros: 500}
	require.NoError(t, journal.Create(ctx, op))
	require.NoError(t, journal.Transition(ctx, op.OperationID, "reserve_pending", "reserved", ProviderUsage{}))
	require.NoError(t, journal.Transition(ctx, op.OperationID, "reserved", "dispatch_pending", ProviderUsage{}))
	require.NoError(t, journal.Transition(ctx, op.OperationID, "dispatch_pending", "dispatched", ProviderUsage{}))
	// A restart has no terminal provider receipt. The durable record stays
	// recoverable and never invents zero-cost usage.
	require.NoError(t, db1.Close())
	db2 := openJournalDB(t, path)
	restarted, err := NewSQLJournal(db2)
	require.NoError(t, err)
	rows, err := restarted.Recoverable(ctx, "owner-1")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "dispatched", rows[0].Phase)
	require.Empty(t, rows[0].ProviderReceiptID)
	require.Zero(t, rows[0].CostMicros)
	require.Error(t, restarted.Transition(ctx, "op-1", "reserved", "settled", ProviderUsage{}), "stale phase must not overwrite a dispatched hold")
}

func TestSQLJournalRecoveryFailsClosedOnCorruptTimestamps(t *testing.T) {
	db := openJournalDB(t, filepath.Join(t.TempDir(), "duplex-corrupt-time.sqlite"))
	journal, err := NewSQLJournal(db)
	require.NoError(t, err)
	ctx := context.Background()
	op := JournalOperation{OperationID: "op-corrupt-time", OwnerKey: "owner-1", ProviderID: "provider", ModelID: "model", BudgetMicros: 500}
	require.NoError(t, journal.Create(ctx, op))
	_, err = db.ExecContext(ctx, `UPDATE duplex_accounting_operations SET updated_at='not-a-timestamp' WHERE operation_id=?`, op.OperationID)
	require.NoError(t, err)

	rows, err := journal.Recoverable(ctx, op.OwnerKey)
	require.ErrorContains(t, err, "parse duplex accounting operation update time")
	require.Nil(t, rows, "corrupt durable metadata must not be returned as a valid recovery record")
}

func TestSQLJournalCreateIsIdempotentWithoutRebindingOperation(t *testing.T) {
	db := openJournalDB(t, filepath.Join(t.TempDir(), "duplex-idempotency.sqlite"))
	journal, err := NewSQLJournal(db)
	require.NoError(t, err)
	ctx := context.Background()
	op := JournalOperation{OperationID: "op-stable", OwnerKey: "owner-1", ProviderID: "provider", ModelID: "model", BudgetMicros: 500}
	require.NoError(t, journal.Create(ctx, op))
	require.NoError(t, journal.Transition(ctx, op.OperationID, "reserve_pending", "reserved", ProviderUsage{}))
	// A retry after an ambiguous database response must preserve the progressed
	// phase rather than reset the operation to its initial intent.
	require.NoError(t, journal.Create(ctx, op))
	rows, err := journal.Recoverable(ctx, op.OwnerKey)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "reserved", rows[0].Phase)

	for _, changed := range []JournalOperation{
		{OperationID: op.OperationID, OwnerKey: "owner-2", ProviderID: op.ProviderID, ModelID: op.ModelID, BudgetMicros: op.BudgetMicros},
		{OperationID: op.OperationID, OwnerKey: op.OwnerKey, ProviderID: "other-provider", ModelID: op.ModelID, BudgetMicros: op.BudgetMicros},
		{OperationID: op.OperationID, OwnerKey: op.OwnerKey, ProviderID: op.ProviderID, ModelID: "other-model", BudgetMicros: op.BudgetMicros},
		{OperationID: op.OperationID, OwnerKey: op.OwnerKey, ProviderID: op.ProviderID, ModelID: op.ModelID, BudgetMicros: op.BudgetMicros + 1},
	} {
		require.Error(t, journal.Create(ctx, changed), "operation owner/provider/model/budget must not be rebound")
	}
}

func TestSQLJournalCountCreatedBetweenReturnsAggregateOnlyWindowCount(t *testing.T) {
	db := openJournalDB(t, filepath.Join(t.TempDir(), "duplex-count.sqlite"))
	journal, err := NewSQLJournal(db)
	require.NoError(t, err)
	ctx := context.Background()
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	for i, created := range []time.Time{from.Add(-time.Second), from.Add(-time.Nanosecond), from, from.Add(time.Nanosecond), to.Add(-time.Nanosecond), to.Add(time.Nanosecond), to} {
		op := JournalOperation{OperationID: fmt.Sprintf("op-%d", i), OwnerKey: "owner", ProviderID: "provider", ModelID: "model", BudgetMicros: 10, CreatedAt: created}
		require.NoError(t, journal.Create(ctx, op))
	}
	count, err := journal.CountCreatedBetween(ctx, from, to)
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
}

func TestSQLJournalPersistsConfirmedSettlementAndSerializesTransitions(t *testing.T) {
	db := openJournalDB(t, filepath.Join(t.TempDir(), "duplex.sqlite"))
	journal, err := NewSQLJournal(db)
	require.NoError(t, err)
	ctx := context.Background()
	op := JournalOperation{OperationID: "op-2", OwnerKey: "owner-1", ProviderID: "provider", ModelID: "model", BudgetMicros: 500}
	require.NoError(t, journal.Create(ctx, op))
	for _, pair := range [][2]string{{"reserve_pending", "reserved"}, {"reserved", "dispatch_pending"}, {"dispatch_pending", "dispatched"}, {"dispatched", "settlement_pending"}} {
		require.NoError(t, journal.Transition(ctx, op.OperationID, pair[0], pair[1], ProviderUsage{}))
	}
	usage := ProviderUsage{ProviderReceiptID: "receipt-1", Model: "model", Provider: "provider", ActiveSeconds: 2, CostMicros: 35}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- journal.Transition(ctx, op.OperationID, "settlement_pending", "settled", usage)
		}()
	}
	wg.Wait()
	close(results)
	successes, failures := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, failures)
	recoverable, err := journal.Recoverable(ctx, "owner-1")
	require.NoError(t, err)
	require.Empty(t, recoverable)
	var receipt string
	var cost int64
	require.NoError(t, db.QueryRow(`SELECT provider_receipt_id,cost_micros FROM duplex_accounting_operations WHERE operation_id=?`, op.OperationID).Scan(&receipt, &cost))
	require.Equal(t, "receipt-1", receipt)
	require.EqualValues(t, 35, cost)
}

func TestReconcileJournalUsesLPBSRecoveryAndKeepsUnknownPending(t *testing.T) {
	db := openJournalDB(t, filepath.Join(t.TempDir(), "duplex.sqlite"))
	journal, err := NewSQLJournal(db)
	require.NoError(t, err)
	ctx := verifiedUserContext(t)
	ownerA, err := verifiedHumanOwnerKey(ctx)
	require.NoError(t, err)
	ctxB := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-2", Source: identity.SourceConsumerAccessToken, Verified: true})
	ctxB, err = WithVerifiedHumanBearer(ctxB, "Bearer human-2-access")
	require.NoError(t, err)
	ownerB, err := verifiedHumanOwnerKey(ctxB)
	require.NoError(t, err)
	for _, entry := range []struct{ id, owner string }{{"op-settled", ownerA}, {"op-unknown", ownerA}, {"op-other-user", ownerB}} {
		id := entry.id
		op := JournalOperation{OperationID: id, OwnerKey: entry.owner, ProviderID: "provider", ModelID: "model", BudgetMicros: 500}
		require.NoError(t, journal.Create(ctx, op))
		require.NoError(t, journal.Transition(ctx, id, "reserve_pending", "reserved", ProviderUsage{}))
		require.NoError(t, journal.Transition(ctx, id, "reserved", "dispatch_pending", ProviderUsage{}))
		require.NoError(t, journal.Transition(ctx, id, "dispatch_pending", "dispatched", ProviderUsage{}))
	}
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer human-access", r.Header.Get("Authorization"))
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		id := pathParts[len(pathParts)-2]
		requested = append(requested, id)
		phase := "uncertain"
		if id == "op-settled" {
			phase = "settled"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"reservation_id": id, "phase": phase, "provider_receipt_id": "receipt"})
	}))
	defer server.Close()
	client, err := NewAccountCreditClient(server.URL, testAccountToken("service-token"), server.Client())
	require.NoError(t, err)
	require.NoError(t, client.ReconcileJournal(ctx, journal))
	rows, err := journal.Recoverable(context.Background(), ownerA)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "op-unknown", rows[0].OperationID)
	require.Equal(t, "dispatched", rows[0].Phase, "LPBS uncertainty must remain pending locally")
	require.Empty(t, rows[0].ProviderReceiptID, "unknown usage receipt must not be manufactured")
	require.ElementsMatch(t, []string{"op-settled", "op-unknown"}, requested, "reconciliation must only call LPBS for the authenticated owner")
	otherRows, err := journal.Recoverable(context.Background(), ownerB)
	require.NoError(t, err)
	require.Len(t, otherRows, 1)
	require.Equal(t, "op-other-user", otherRows[0].OperationID)
}
