package book_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
)

// [REQ:TRS-P0-010] The real schema admits many books for the operator but no
// second beneficiary, even when a future caller bypasses an HTTP boundary.
func TestSQLiteRepositoryEnforcesOneOperatorBeneficiary(t *testing.T) {
	databaseHandle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), databaseHandle, database.SchemaProviderFunc(book.Schema)))
	service := book.NewService(book.NewSQLiteRepository(databaseHandle), schedule.NewFake(time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)))

	_, err := service.Create(context.Background(), book.CreateInput{ID: "personal", Name: "Personal", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = service.Create(context.Background(), book.CreateInput{ID: "business", Name: "Business", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err, "multiple books may separate contexts for the same beneficiary")
	_, err = service.Create(context.Background(), book.CreateInput{ID: "third-party", Name: "Third party", BeneficiaryIdentity: "customer:2"})
	require.ErrorIs(t, err, book.ErrBeneficiaryConflict)
}

// [REQ:TRS-P0-010] enforces invariant: onlyOperatorBeneficiaryCanBeRepresented.
// This test deliberately bypasses every service and handler. The persistence
// schema itself must make third-party custody and cross-book authority
// relationships unrepresentable.
func TestSchemaRejectsRawThirdPartyCustody(t *testing.T) {
	ctx := context.Background()
	databaseHandle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, databaseHandle,
		database.SchemaProviderFunc(book.Schema),
		database.SchemaProviderFunc(budget.Schema),
		database.SchemaProviderFunc(mandate.Schema),
		database.SchemaProviderFunc(instrument.Schema),
	))

	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	_, err := databaseHandle.ExecContext(ctx, `INSERT INTO treasury_beneficiary(singleton_key, identity) VALUES(1, 'operator:1')`)
	require.NoError(t, err)
	_, err = databaseHandle.ExecContext(ctx, `INSERT INTO treasury_beneficiary(singleton_key, identity) VALUES(2, 'customer:2')`)
	require.Error(t, err, "the singleton CHECK must reject a second beneficiary row")

	_, err = databaseHandle.ExecContext(ctx, `INSERT INTO books(id, name, beneficiary_identity, created_at) VALUES('operator-book', 'Operator', 'operator:1', ?)`, now)
	require.NoError(t, err)
	_, err = databaseHandle.ExecContext(ctx, `INSERT INTO books(id, name, beneficiary_identity, created_at) VALUES('third-party-book', 'Third party', 'customer:2', ?)`, now)
	require.Error(t, err, "the books foreign key must reject an identity outside the singleton")
	_, err = databaseHandle.ExecContext(ctx, `INSERT INTO books(id, name, beneficiary_identity, created_at) VALUES('other-operator-book', 'Other operator context', 'operator:1', ?)`, now)
	require.NoError(t, err)
	_, err = databaseHandle.ExecContext(ctx, `UPDATE treasury_beneficiary SET identity = 'customer:2' WHERE singleton_key = 1`)
	require.Error(t, err, "referenced operator identity must not be replaceable behind existing books")

	_, err = databaseHandle.ExecContext(ctx, `INSERT INTO budgets(id, book_id, currency, total_cap_minor, periodic_cap_minor, per_transaction_cap_minor, period_seconds, requires_approval, frozen, created_at) VALUES('budget-1', 'operator-book', 'USD', 1000, 1000, 1000, 3600, 1, 0, ?)`, now)
	require.NoError(t, err)
	_, err = databaseHandle.ExecContext(ctx, `INSERT INTO budgets(id, book_id, currency, total_cap_minor, periodic_cap_minor, per_transaction_cap_minor, period_seconds, requires_approval, frozen, created_at) VALUES('orphan-budget', 'missing-book', 'USD', 1000, 1000, 1000, 3600, 1, 0, ?)`, now)
	require.Error(t, err, "a budget must resolve to an operator-owned book")

	mandateInsert := `INSERT INTO mandates(id, idempotency_key, book_id, budget_id, authorizer, cap_minor, currency, allowed_counterparties_json, denied_counterparties_json, required_evidence_json, expires_at, issued_at, signature, status) VALUES(?, ?, ?, 'budget-1', 'operator:1', 1000, 'USD', '[]', '[]', '[]', ?, ?, X'01', 'live')`
	_, err = databaseHandle.ExecContext(ctx, mandateInsert, "cross-book-mandate", "cross-book-key", "other-operator-book", now, now)
	require.ErrorContains(t, err, "mandate budget must belong to mandate book")
	_, err = databaseHandle.ExecContext(ctx, mandateInsert, "mandate-1", "mandate-key", "operator-book", now, now)
	require.NoError(t, err)
	_, err = databaseHandle.ExecContext(ctx, `UPDATE mandates SET book_id = 'other-operator-book' WHERE id = 'mandate-1'`)
	require.ErrorContains(t, err, "mandate budget must belong to mandate book")

	instrumentInsert := `INSERT INTO instruments(id, book_id, mandate_id, rail, credential_reference, cap_minor, currency, counterparty, expires_at, created_at) VALUES(?, ?, 'mandate-1', 'manual', 'manual/operator-attestation', 1000, 'USD', 'vendor.example', ?, ?)`
	_, err = databaseHandle.ExecContext(ctx, instrumentInsert, "cross-book-instrument", "other-operator-book", now, now)
	require.ErrorContains(t, err, "instrument mandate must belong to instrument book")
	_, err = databaseHandle.ExecContext(ctx, instrumentInsert, "instrument-1", "operator-book", now, now)
	require.NoError(t, err)
	_, err = databaseHandle.ExecContext(ctx, `UPDATE instruments SET book_id = 'other-operator-book' WHERE id = 'instrument-1'`)
	require.ErrorContains(t, err, "instrument mandate must belong to instrument book")

	for _, table := range []string{"budgets", "mandates", "instruments"} {
		assertNoBeneficiaryColumn(t, databaseHandle, table)
	}
}

func assertNoBeneficiaryColumn(t *testing.T, databaseHandle *sql.DB, table string) {
	t.Helper()
	rows, err := databaseHandle.Query(`PRAGMA table_info(` + table + `)`)
	require.NoError(t, err)
	defer rows.Close()
	found := false
	for rows.Next() {
		found = true
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		require.NoError(t, rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey))
		require.NotContains(t, strings.ToLower(name), "beneficiary", "%s must derive custody identity exclusively through book_id", table)
	}
	require.NoError(t, rows.Err())
	require.True(t, found, "%s must exist in the real schema", table)
}
