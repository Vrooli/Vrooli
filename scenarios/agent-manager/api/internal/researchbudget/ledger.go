// Package researchbudget retains run-bound reporting records for research
// spend. Treasury remains the authority for mandates and settlement.
package researchbudget

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrConflict          = errors.New("research budget record conflicts with retained evidence")
	ErrSettlementMissing = errors.New("free-alternative settlement was not recorded for this run")
)

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
	FreeAlternative                             *FreeAlternative
}

// FreeAlternative is a bounded observation of a free option considered for
// one exact paid settlement. It is explicitly self-reported research evidence;
// it does not prove the provider or result was independently verified.
type FreeAlternative struct {
	RunID         string    `json:"run_id,omitempty"`
	SettlementID  string    `json:"settlement_id,omitempty"`
	SourceID      string    `json:"source_id"`
	Label         string    `json:"label"`
	ResultRef     string    `json:"result_ref"`
	SHA256        string    `json:"sha256"`
	ResponseBytes int64     `json:"response_bytes"`
	Limitation    string    `json:"limitation"`
	ObservedAt    time.Time `json:"observed_at"`
	RecordedAt    time.Time `json:"recorded_at,omitempty"`
}

type Summary struct {
	RunID, TreasuryBudgetID, MandateID, Currency, UnitCode string
	CapMinor, SpendMinor, RemainingMinor                   int64
	SpendCount                                             int
	Spend                                                  []Spend
}

// DB is deliberately the small common subset implemented by sql.DB and the
// API-core RoutedDB. Production callers must use RoutedDB so Test Genie and
// other isolated requests retain their database routing.
type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Ledger struct{ DB DB }

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
	if !retained.CreatedAt.Equal(b.CreatedAt.UTC()) {
		return ErrConflict
	}
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
	var budgetCurrency, budgetUnit string
	if err := l.DB.QueryRowContext(ctx, `SELECT currency,unit_code FROM research_run_budgets WHERE run_id=?`, s.RunID).Scan(&budgetCurrency, &budgetUnit); err != nil {
		return err
	}
	if strings.ToUpper(s.Currency) != budgetCurrency || s.UnitCode != budgetUnit {
		return fmt.Errorf("spend unit %s/%s does not match run budget unit %s/%s", strings.ToUpper(s.Currency), s.UnitCode, budgetCurrency, budgetUnit)
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
	if got.ID != s.ID || got.RunID != s.RunID || got.SettlementID != s.SettlementID || got.TreasuryReceiptID != s.TreasuryReceiptID || got.IntegrationID != s.IntegrationID || got.Category != s.Category || got.AmountMinor != s.AmountMinor || got.Currency != strings.ToUpper(s.Currency) || got.UnitCode != s.UnitCode || got.ResultRef != s.ResultRef || got.BenefitState != s.BenefitState || got.BenefitSummary != s.BenefitSummary || settled != s.SettledAt.UTC().Format(time.RFC3339Nano) {
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
	rows, err := l.DB.QueryContext(ctx, `SELECT s.id,s.run_id,s.settlement_id,s.treasury_receipt_id,s.integration_id,s.category,s.amount_minor,s.currency,s.unit_code,s.result_ref,s.benefit_state,s.benefit_summary,s.settled_at,s.recorded_at, f.source_id,f.label,f.result_ref,f.sha256,f.response_bytes,f.limitation,f.observed_at,f.recorded_at FROM research_run_spend s LEFT JOIN research_run_spend_free_alternatives f ON f.run_id=s.run_id AND f.settlement_id=s.settlement_id WHERE s.run_id=? ORDER BY s.settled_at,s.id`, runID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var s Spend
		var settled, recorded string
		var sourceID, label, resultRef, digest, limitation, observed, alternativeRecorded sql.NullString
		var responseBytes sql.NullInt64
		if err := rows.Scan(&s.ID, &s.RunID, &s.SettlementID, &s.TreasuryReceiptID, &s.IntegrationID, &s.Category, &s.AmountMinor, &s.Currency, &s.UnitCode, &s.ResultRef, &s.BenefitState, &s.BenefitSummary, &settled, &recorded, &sourceID, &label, &resultRef, &digest, &responseBytes, &limitation, &observed, &alternativeRecorded); err != nil {
			return out, err
		}
		s.SettledAt, _ = time.Parse(time.RFC3339Nano, settled)
		s.RecordedAt, _ = time.Parse(time.RFC3339Nano, recorded)
		if sourceID.Valid {
			alternative := &FreeAlternative{RunID: s.RunID, SettlementID: s.SettlementID, SourceID: sourceID.String, Label: label.String, ResultRef: resultRef.String, SHA256: digest.String, ResponseBytes: responseBytes.Int64, Limitation: limitation.String}
			alternative.ObservedAt, _ = time.Parse(time.RFC3339Nano, observed.String)
			alternative.RecordedAt, _ = time.Parse(time.RFC3339Nano, alternativeRecorded.String)
			s.FreeAlternative = alternative
		}
		if s.Currency != out.Currency || s.UnitCode != out.UnitCode {
			return out, fmt.Errorf("spend unit %s/%s does not match run budget unit %s/%s", s.Currency, s.UnitCode, out.Currency, out.UnitCode)
		}
		if s.AmountMinor < 0 {
			return out, fmt.Errorf("research spend %s has a negative amount", s.ID)
		}
		const maxInt64 = int64(1<<63 - 1)
		if out.SpendMinor > maxInt64-s.AmountMinor {
			return out, fmt.Errorf("run %s research spend total overflows int64", runID)
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

// RecordFreeAlternative stores one immutable comparison observation for an
// already-recorded settlement in the same run. A retry with identical
// evidence succeeds; conflicting reuse fails closed.
func (l Ledger) RecordFreeAlternative(ctx context.Context, value FreeAlternative) error {
	if l.DB == nil || value.RunID == "" || value.SettlementID == "" || value.SourceID == "" || value.Label == "" || value.ResultRef == "" || value.Limitation == "" || value.ResponseBytes < 0 || value.ResponseBytes > 2<<20 || value.ObservedAt.IsZero() {
		return errors.New("run, settlement, source, label, result reference, bounded response bytes, limitation and observation time are required")
	}
	digest, err := hex.DecodeString(value.SHA256)
	if err != nil || len(digest) != 32 || strings.ToLower(value.SHA256) != value.SHA256 {
		return errors.New("free-alternative SHA-256 must be 64 lowercase hexadecimal characters")
	}
	var retainedRun string
	if err := l.DB.QueryRowContext(ctx, `SELECT run_id FROM research_run_spend WHERE settlement_id=?`, value.SettlementID).Scan(&retainedRun); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSettlementMissing
		}
		return err
	}
	if retainedRun != value.RunID {
		return errors.New("free-alternative run does not match its Treasury settlement")
	}
	if value.RecordedAt.IsZero() {
		value.RecordedAt = time.Now().UTC()
	}
	_, err = l.DB.ExecContext(ctx, `INSERT INTO research_run_spend_free_alternatives(run_id,settlement_id,source_id,label,result_ref,sha256,response_bytes,limitation,observed_at,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(run_id,settlement_id) DO NOTHING`, value.RunID, value.SettlementID, value.SourceID, value.Label, value.ResultRef, value.SHA256, value.ResponseBytes, value.Limitation, value.ObservedAt.UTC().Format(time.RFC3339Nano), value.RecordedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	var got FreeAlternative
	var observed, recorded string
	err = l.DB.QueryRowContext(ctx, `SELECT run_id,settlement_id,source_id,label,result_ref,sha256,response_bytes,limitation,observed_at,recorded_at FROM research_run_spend_free_alternatives WHERE run_id=? AND settlement_id=?`, value.RunID, value.SettlementID).Scan(&got.RunID, &got.SettlementID, &got.SourceID, &got.Label, &got.ResultRef, &got.SHA256, &got.ResponseBytes, &got.Limitation, &observed, &recorded)
	if err != nil {
		return err
	}
	if got.RunID != value.RunID || got.SettlementID != value.SettlementID || got.SourceID != value.SourceID || got.Label != value.Label || got.ResultRef != value.ResultRef || got.SHA256 != value.SHA256 || got.ResponseBytes != value.ResponseBytes || got.Limitation != value.Limitation || observed != value.ObservedAt.UTC().Format(time.RFC3339Nano) {
		return ErrConflict
	}
	return nil
}
