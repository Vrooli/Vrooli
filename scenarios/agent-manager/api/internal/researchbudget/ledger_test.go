package researchbudget

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/runmodel"
	_ "modernc.org/sqlite"
)

func testLedger(t *testing.T) Ledger {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "research-budget.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(runmodel.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO runs(id,status) VALUES('run-1','running')`); err != nil {
		t.Fatal(err)
	}
	return Ledger{DB: db}
}

func TestRunBudgetStoresRunBoundTreasuryMandateAndSettlementBenefit(t *testing.T) {
	ctx := context.Background()
	ledger := testLedger(t)
	budget := Budget{RunID: "run-1", TreasuryBudgetID: "budget-1", MandateID: "mandate-1", AgentSubject: "agent:research", CapMinor: 500, Currency: "usdc", UnitCode: "USDC_ATOMS", PolicyRevision: "policy-1", CreatedAt: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)}
	if err := ledger.PutBudget(ctx, budget); err != nil {
		t.Fatal(err)
	}
	if err := ledger.PutBudget(ctx, budget); err != nil {
		t.Fatalf("identical budget replay should be idempotent: %v", err)
	}
	conflicting := budget
	conflicting.CapMinor++
	if err := ledger.PutBudget(ctx, conflicting); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed cap must not rewrite the budget snapshot: %v", err)
	}
	settledAt := time.Date(2026, 11, 10, 12, 1, 0, 0, time.UTC)
	spend := Spend{ID: "spend-1", RunID: "run-1", SettlementID: "settlement-1", TreasuryReceiptID: "receipt-1", IntegrationID: "x402:research-api", Category: "x402_call", AmountMinor: 125, Currency: "USDC", UnitCode: "USDC_ATOMS", ResultRef: "result-1", BenefitState: "measured", BenefitSummary: "Returned 3 verified source records used in the checkpoint", SettledAt: settledAt}
	if err := ledger.RecordSpend(ctx, spend); err != nil {
		t.Fatal(err)
	}
	if err := ledger.RecordSpend(ctx, spend); err != nil {
		t.Fatalf("identical settlement replay should be idempotent: %v", err)
	}
	conflictingSpend := spend
	conflictingSpend.AmountMinor++
	if err := ledger.RecordSpend(ctx, conflictingSpend); !errors.Is(err, ErrConflict) {
		t.Fatalf("same settlement cannot be changed to a different amount: %v", err)
	}
	spend.ID, spend.SettlementID, spend.TreasuryReceiptID, spend.IntegrationID, spend.Category, spend.AmountMinor, spend.ResultRef, spend.BenefitState, spend.BenefitSummary = "spend-2", "settlement-2", "receipt-2", "ci4:compute", "compute", 200, "result-2", "pending", ""
	if err := ledger.RecordSpend(ctx, spend); err != nil {
		t.Fatal(err)
	}
	got, err := ledger.Read(ctx, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != "run-1" || got.TreasuryBudgetID != "budget-1" || got.MandateID != "mandate-1" || got.CapMinor != 500 || got.SpendMinor != 325 || got.RemainingMinor != 175 || got.SpendCount != 2 {
		t.Fatalf("unexpected run spend summary: %+v", got)
	}
	if got.Spend[0].BenefitState != "measured" || got.Spend[0].ResultRef != "result-1" || got.Spend[1].BenefitState != "pending" {
		t.Fatalf("summary lost benefit evidence: %+v", got.Spend)
	}
}

func TestRunSpendRequiresExistingBoundBudgetAndBenefitEvidence(t *testing.T) {
	ctx := context.Background()
	ledger := testLedger(t)
	spend := Spend{ID: "spend", RunID: "run-1", SettlementID: "settlement", TreasuryReceiptID: "receipt", IntegrationID: "x402:research-api", Category: "x402_call", AmountMinor: 5, Currency: "USD", BenefitState: "measured", SettledAt: time.Now().UTC()}
	if err := ledger.RecordSpend(ctx, spend); err == nil {
		t.Fatal("spend without retained run budget should be rejected")
	}
	if err := ledger.PutBudget(ctx, Budget{RunID: "run-1", TreasuryBudgetID: "budget-1", MandateID: "mandate-1", AgentSubject: "agent:research", CapMinor: 100, Currency: "USDC", UnitCode: "USDC_ATOMS", PolicyRevision: "policy-1"}); err != nil {
		t.Fatal(err)
	}
	if err := ledger.RecordSpend(ctx, spend); err == nil {
		t.Fatal("measured benefit without result reference and summary should be rejected")
	}
	spend.RunID = "other-run"
	spend.BenefitState, spend.BenefitSummary = "pending", ""
	if err := ledger.RecordSpend(ctx, spend); err == nil {
		t.Fatal("spend under a different run must be rejected")
	}
}
