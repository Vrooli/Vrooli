// Package researchbudget retains run-bound reporting records for research
// spend. Treasury remains the authority for mandates and settlement.
package researchbudget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrConflict = errors.New("research budget record conflicts with retained evidence")

type Budget struct {
	RunID, TreasuryBudgetID, MandateID, AgentSubject string
	CapMinor                                         int64
	Currency, UnitCode, PolicyRevision               string
	CreatedAt                                        time.Time
}

type Spend struct {
	ID, RunID, SettlementID, TreasuryReceiptID  string
	IntegrationID, Category, Currency, UnitCode string
	AmountMinor                                 int64
	ResultRef, BenefitState, BenefitSummary     string
	SettledAt, RecordedAt                       time.Time
}

type Summary struct {
	RunID, TreasuryBudgetID, MandateID, Currency, UnitCode string
	CapMinor, SpendMinor, RemainingMinor                   int64
	SpendCount                                             int
	Spend                                                  []Spend
}

type Ledger struct{ DB *sql.DB }

func (l Ledger) PutBudget(ctx context.Context, b Budget) error {
	if l.DB == nil || b.RunID == "" || b.TreasuryBudgetID == "" || b.MandateID == "" || b.AgentSubject == "" || b.Currency == "" || b.UnitCode == "" || b.PolicyRevision == "" || b.CapMinor < 0 {
		return errors.New("run, Treasury budget, mandate, agent, nonnegative cap, currency, unit code and policy revision are required")
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	_, err := l.DB.ExecContext(ctx, `INSERT INTO research_run_budgets(run_id,treasury_budget_id,mandate_id,agent_subject,cap_minor,currency,unit_code,policy_revision,created_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(run_id) DO NOTHING`, b.RunID, b.TreasuryBudgetID, b.MandateID, b.AgentSubject, b.CapMinor, strings.ToUpper(b.Currency), b.UnitCode, b.PolicyRevision, b.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	var retained Budget
	var created string
	err = l.DB.QueryRowContext(ctx, `SELECT run_id,treasury_budget_id,mandate_id,agent_subject,cap_minor,currency,unit_code,policy_revision,created_at FROM research_run_budgets WHERE run_id=?`, b.RunID).Scan(&retained.RunID, &retained.TreasuryBudgetID, &retained.MandateID, &retained.AgentSubject, &retained.CapMinor, &retained.Currency, &retained.UnitCode, &retained.PolicyRevision, &created)
	if err != nil {
		return err
	}
	if retained.TreasuryBudgetID != b.TreasuryBudgetID || retained.MandateID != b.MandateID || retained.AgentSubject != b.AgentSubject || retained.CapMinor != b.CapMinor || retained.Currency != strings.ToUpper(b.Currency) || retained.UnitCode != b.UnitCode || retained.PolicyRevision != b.PolicyRevision {
		return ErrConflict
	}
	retained.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return nil
}

// RecordSpend accepts only a Treasury settlement receipt identity. Replaying
// identical evidence is idempotent; conflicting reuse is rejected. This is a
// checkpoint/report ledger, not a spending authorization gate.
func (l Ledger) RecordSpend(ctx context.Context, s Spend) error {
	if l.DB == nil || s.ID == "" || s.RunID == "" || s.SettlementID == "" || s.TreasuryReceiptID == "" || s.IntegrationID == "" || s.Currency == "" || s.UnitCode == "" || s.AmountMinor < 0 {
		return errors.New("run, settlement and Treasury receipt identities, integration, currency, unit code and nonnegative amount are required")
	}
	if s.Category != "x402_call" && s.Category != "compute" && s.Category != "dataset" {
		return fmt.Errorf("unsupported research spend category %q", s.Category)
	}
	if s.BenefitState != "measured" && s.BenefitState != "pending" && s.BenefitState != "none" {
		return fmt.Errorf("unsupported benefit state %q", s.BenefitState)
	}
	if s.SettledAt.IsZero() {
		return errors.New("settlement time is required")
	}
	if s.RecordedAt.IsZero() {
		s.RecordedAt = time.Now().UTC()
	}
	if s.BenefitState == "measured" && (s.ResultRef == "" || s.BenefitSummary == "") {
		return errors.New("measured benefit requires a result reference and summary")
	}
	_, err := l.DB.ExecContext(ctx, `INSERT INTO research_run_spend(id,run_id,settlement_id,treasury_receipt_id,integration_id,category,amount_minor,currency,unit_code,result_ref,benefit_state,benefit_summary,settled_at,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(settlement_id) DO NOTHING`, s.ID, s.RunID, s.SettlementID, s.TreasuryReceiptID, s.IntegrationID, s.Category, s.AmountMinor, strings.ToUpper(s.Currency), s.UnitCode, s.ResultRef, s.BenefitState, s.BenefitSummary, s.SettledAt.UTC().Format(time.RFC3339Nano), s.RecordedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	var got Spend
	var settled, recorded string
	err = l.DB.QueryRowContext(ctx, `SELECT id,run_id,settlement_id,treasury_receipt_id,integration_id,category,amount_minor,currency,unit_code,result_ref,benefit_state,benefit_summary,settled_at,recorded_at FROM research_run_spend WHERE settlement_id=?`, s.SettlementID).Scan(&got.ID, &got.RunID, &got.SettlementID, &got.TreasuryReceiptID, &got.IntegrationID, &got.Category, &got.AmountMinor, &got.Currency, &got.UnitCode, &got.ResultRef, &got.BenefitState, &got.BenefitSummary, &settled, &recorded)
	if err != nil {
		return err
	}
	if got.ID != s.ID || got.RunID != s.RunID || got.TreasuryReceiptID != s.TreasuryReceiptID || got.IntegrationID != s.IntegrationID || got.Category != s.Category || got.AmountMinor != s.AmountMinor || got.Currency != strings.ToUpper(s.Currency) || got.UnitCode != s.UnitCode || got.ResultRef != s.ResultRef || got.BenefitState != s.BenefitState || got.BenefitSummary != s.BenefitSummary {
		return ErrConflict
	}
	return nil
}

func (l Ledger) Read(ctx context.Context, runID string) (Summary, error) {
	var out Summary
	if l.DB == nil || runID == "" {
		return out, errors.New("database and run identity are required")
	}
	err := l.DB.QueryRowContext(ctx, `SELECT run_id,treasury_budget_id,mandate_id,currency,unit_code,cap_minor FROM research_run_budgets WHERE run_id=?`, runID).Scan(&out.RunID, &out.TreasuryBudgetID, &out.MandateID, &out.Currency, &out.UnitCode, &out.CapMinor)
	if err != nil {
		return out, err
	}
	rows, err := l.DB.QueryContext(ctx, `SELECT id,run_id,settlement_id,treasury_receipt_id,integration_id,category,amount_minor,currency,unit_code,result_ref,benefit_state,benefit_summary,settled_at,recorded_at FROM research_run_spend WHERE run_id=? ORDER BY settled_at,id`, runID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var s Spend
		var settled, recorded string
		if err := rows.Scan(&s.ID, &s.RunID, &s.SettlementID, &s.TreasuryReceiptID, &s.IntegrationID, &s.Category, &s.AmountMinor, &s.Currency, &s.UnitCode, &s.ResultRef, &s.BenefitState, &s.BenefitSummary, &settled, &recorded); err != nil {
			return out, err
		}
		s.SettledAt, _ = time.Parse(time.RFC3339Nano, settled)
		s.RecordedAt, _ = time.Parse(time.RFC3339Nano, recorded)
		if s.Currency != out.Currency || s.UnitCode != out.UnitCode {
			return out, fmt.Errorf("spend unit %s/%s does not match run budget unit %s/%s", s.Currency, s.UnitCode, out.Currency, out.UnitCode)
		}
		out.SpendMinor += s.AmountMinor
		out.SpendCount++
		out.Spend = append(out.Spend, s)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.RemainingMinor = out.CapMinor - out.SpendMinor
	return out, nil
}
