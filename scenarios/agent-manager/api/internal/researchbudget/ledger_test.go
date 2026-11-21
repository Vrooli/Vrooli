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
	conflicting = budget
	conflicting.CreatedAt = conflicting.CreatedAt.Add(time.Second)
	if err := ledger.PutBudget(ctx, conflicting); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed creation time must not rewrite the budget snapshot: %v", err)
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
	alternative := FreeAlternative{RunID: "run-1", SettlementID: "settlement-1", SourceID: "offer-desk:free-sample", Label: "no-signup sample endpoint", ResultRef: "sha256:free-result", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ResponseBytes: 81, Limitation: "shape sample only; not current catalog data", ObservedAt: settledAt.Add(time.Second), RecordedAt: settledAt.Add(2 * time.Second)}
	if err := ledger.RecordFreeAlternative(ctx, alternative); err != nil {
		t.Fatal(err)
	}
	if err := ledger.RecordFreeAlternative(ctx, alternative); err != nil {
		t.Fatalf("identical comparator retry should be idempotent: %v", err)
	}
	got, err = ledger.Read(ctx, "run-1")
	if err != nil || got.Spend[0].FreeAlternative == nil {
		t.Fatalf("free alternative not joined to exact settlement: summary=%+v err=%v", got, err)
	}
	if *got.Spend[0].FreeAlternative != alternative {
		t.Fatalf("free comparator changed on read: got=%+v want=%+v", *got.Spend[0].FreeAlternative, alternative)
	}
	conflictingAlternative := alternative
	conflictingAlternative.SHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := ledger.RecordFreeAlternative(ctx, conflictingAlternative); !errors.Is(err, ErrConflict) {
		t.Fatalf("same settlement cannot be rebound to different free evidence: %v", err)
	}
}

func TestFreeAlternativeEvidenceSurvivesDatabaseReopenAndRejectsForeignSettlement(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "research-budget-reopen.db")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
			t.Fatal(err)
		}
		return db
	}
	db := open()
	if _, err := db.Exec(runmodel.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO runs(id,status) VALUES('run-1','running')`); err != nil {
		t.Fatal(err)
	}
	ledger := Ledger{DB: db}
	if err := ledger.PutBudget(ctx, Budget{RunID: "run-1", TreasuryBudgetID: "budget-1", MandateID: "mandate-1", AgentSubject: "agent", CapMinor: 100, Currency: "USD", UnitCode: "USD_MINOR", PolicyRevision: "policy-1"}); err != nil {
		t.Fatal(err)
	}
	settled := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	if err := ledger.RecordSpend(ctx, Spend{ID: "spend-1", RunID: "run-1", SettlementID: "settlement-1", TreasuryReceiptID: "receipt-1", IntegrationID: "x402:offer-desk", Category: "x402_call", AmountMinor: 1, Currency: "USD", UnitCode: "USD_MINOR", BenefitState: "measured", ResultRef: "result-1", BenefitSummary: "paid response observed", SettledAt: settled}); err != nil {
		t.Fatal(err)
	}
	alt := FreeAlternative{RunID: "run-1", SettlementID: "settlement-1", SourceID: "offer-desk:free-sample", Label: "free sample", ResultRef: "free-result", SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ResponseBytes: 64, Limitation: "format sample only", ObservedAt: settled.Add(time.Second), RecordedAt: settled.Add(2 * time.Second)}
	if err := ledger.RecordFreeAlternative(ctx, alt); err != nil {
		t.Fatal(err)
	}
	foreign := alt
	foreign.RunID = "missing-run"
	if err := ledger.RecordFreeAlternative(ctx, foreign); err == nil {
		t.Fatal("a comparator for a foreign run must be refused")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = open()
	t.Cleanup(func() { _ = db.Close() })
	got, err := (Ledger{DB: db}).Read(ctx, "run-1")
	if err != nil || len(got.Spend) != 1 || got.Spend[0].FreeAlternative == nil {
		t.Fatalf("comparison evidence did not survive database reopen: summary=%+v err=%v", got, err)
	}
	if got.Spend[0].FreeAlternative.SHA256 != alt.SHA256 || got.Spend[0].FreeAlternative.Limitation != alt.Limitation {
		t.Fatalf("reopened comparator changed: %+v", got.Spend[0].FreeAlternative)
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

func TestReadRejectsSpendTotalOverflow(t *testing.T) {
	ctx := context.Background()
	ledger := testLedger(t)
	if err := ledger.PutBudget(ctx, Budget{RunID: "run-1", TreasuryBudgetID: "budget-1", MandateID: "mandate-1", AgentSubject: "agent:research", CapMinor: 100, Currency: "USD", UnitCode: "USD_MINOR", PolicyRevision: "policy-1"}); err != nil {
		t.Fatal(err)
	}
	settledAt := time.Date(2026, 11, 14, 1, 0, 0, 0, time.UTC)
	for _, spend := range []Spend{
		{ID: "spend-max", RunID: "run-1", SettlementID: "settlement-max", TreasuryReceiptID: "receipt-max", IntegrationID: "x402:first", Category: "x402_call", AmountMinor: int64(1<<63 - 1), Currency: "USD", UnitCode: "USD_MINOR", BenefitState: "none", SettledAt: settledAt},
		{ID: "spend-overflow", RunID: "run-1", SettlementID: "settlement-overflow", TreasuryReceiptID: "receipt-overflow", IntegrationID: "x402:second", Category: "x402_call", AmountMinor: 1, Currency: "USD", UnitCode: "USD_MINOR", BenefitState: "none", SettledAt: settledAt.Add(time.Second)},
	} {
		if err := ledger.RecordSpend(ctx, spend); err != nil {
			t.Fatalf("RecordSpend(%s): %v", spend.ID, err)
		}
	}
	if _, err := ledger.Read(ctx, "run-1"); err == nil {
		t.Fatal("run spend total overflow must fail closed rather than wrap")
	}
}
