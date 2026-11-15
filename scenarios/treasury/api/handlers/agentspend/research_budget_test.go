package agentspend

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/identity"
	"treasury/internal/mandate"
	"treasury/internal/researchbudget"
	"treasury/internal/settlement"
)

type researchBudgetPublisherSpy struct {
	snapshots   []researchbudget.Snapshot
	settlements []researchbudget.Settlement
	err         error
}

func (p *researchBudgetPublisherSpy) PutSnapshot(_ context.Context, runID string, value researchbudget.Snapshot) error {
	if p.err != nil {
		return p.err
	}
	if runID == "" {
		return errors.New("missing run ID")
	}
	p.snapshots = append(p.snapshots, value)
	return nil
}

func (p *researchBudgetPublisherSpy) RecordSettlement(_ context.Context, runID string, value researchbudget.Settlement) error {
	if p.err != nil {
		return p.err
	}
	if runID == "" {
		return errors.New("missing run ID")
	}
	p.settlements = append(p.settlements, value)
	return nil
}

type researchBudgetMandateSpy struct{ value mandate.Mandate }

func (m researchBudgetMandateSpy) Get(context.Context, string) (mandate.Mandate, error) {
	return m.value, nil
}

type researchBudgetHeadroomSpy struct{ value budget.Headroom }

func (h researchBudgetHeadroomSpy) Headroom(context.Context, string) (budget.Headroom, error) {
	return h.value, nil
}

func TestPublishResearchBudgetEvidenceUsesTreasuryRunBindingAndReceipt(t *testing.T) {
	issued := time.Date(2026, 11, 14, 2, 0, 0, 0, time.UTC)
	settled := issued.Add(time.Minute)
	runID := "282159e3-8730-40ef-95af-307494b74ed4"
	publisher := &researchBudgetPublisherSpy{}
	h := &connectHandler{
		researchBudget: publisher,
		mandateRead: researchBudgetMandateSpy{value: mandate.Mandate{
			ID: "mandate-1", BudgetID: "budget-1", AgentSubject: "agent:research",
			AgentRunID: runID, IssuedAt: issued,
		}},
		headroom: researchBudgetHeadroomSpy{value: budget.Headroom{
			BudgetID: "budget-1", Currency: "USD", TotalCapMinor: 1000,
		}},
	}
	claims := identity.Claims{RunID: runID, Subject: "agent:research"}
	auth := authorization.Record{BudgetID: "budget-1", MandateID: "mandate-1", Currency: "USD"}
	charge := settlement.Record{
		ID: "settlement-1", Rail: "x402", Counterparty: "127.0.0.1:14022", Outcome: settlement.OutcomeSettled,
		AmountMinor: 1, Currency: "USD", ExternalID: "0xfeed", OccurredAt: settled,
		Detail: "x402 v2 exact payment settled on eip155:8453 (6 asset decimals); response_bytes=81; response_sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}

	require.NoError(t, h.publishResearchBudgetEvidence(context.Background(), claims, auth, charge))
	require.Len(t, publisher.snapshots, 1)
	require.Equal(t, researchbudget.Snapshot{
		TreasuryBudgetID: "budget-1", MandateID: "mandate-1", AgentSubject: "agent:research",
		CapMinor: 1000, Currency: "USD", UnitCode: "USD_MINOR", PolicyRevision: "treasury-budget-headroom-v1", CreatedAt: issued,
	}, publisher.snapshots[0])
	require.Len(t, publisher.settlements, 1)
	require.Equal(t, "settlement-1", publisher.settlements[0].SettlementID)
	require.Equal(t, "0xfeed", publisher.settlements[0].TreasuryReceiptID)
	require.Equal(t, "x402:127.0.0.1:14022", publisher.settlements[0].IntegrationID)
	require.Equal(t, "measured", publisher.settlements[0].BenefitState)
	require.Contains(t, publisher.settlements[0].BenefitSummary, "response_sha256=")
	require.Equal(t, settled, publisher.settlements[0].SettledAt)
}

func TestPublishResearchBudgetEvidenceSkipsUnboundAndRejectsDifferentRun(t *testing.T) {
	publisher := &researchBudgetPublisherSpy{}
	h := &connectHandler{researchBudget: publisher}
	charge := settlement.Record{Rail: "x402", Outcome: settlement.OutcomeSettled}
	require.NoError(t, h.publishResearchBudgetEvidence(context.Background(), identity.Claims{Subject: "agent"}, authorization.Record{}, charge))
	require.Empty(t, publisher.snapshots)

	issued := time.Date(2026, 11, 14, 2, 0, 0, 0, time.UTC)
	h.mandateRead = researchBudgetMandateSpy{value: mandate.Mandate{ID: "mandate-1", BudgetID: "budget-1", AgentSubject: "agent", AgentRunID: "different-run", IssuedAt: issued}}
	h.headroom = researchBudgetHeadroomSpy{value: budget.Headroom{BudgetID: "budget-1", Currency: "USD", TotalCapMinor: 100}}
	err := h.publishResearchBudgetEvidence(context.Background(), identity.Claims{RunID: "requested-run", Subject: "agent"}, authorization.Record{BudgetID: "budget-1", MandateID: "mandate-1", Currency: "USD"}, settlement.Record{ID: "settlement", Rail: "x402", Outcome: settlement.OutcomeSettled, AmountMinor: 1, Currency: "USD", ExternalID: "0xfeed", OccurredAt: issued})
	require.ErrorContains(t, err, "does not match the verified Agent Manager run")
	require.Empty(t, publisher.snapshots)
	require.Empty(t, publisher.settlements)
}

func TestPublishResearchBudgetEvidenceRequiresExactBudgetAndCurrency(t *testing.T) {
	issued := time.Date(2026, 11, 14, 2, 0, 0, 0, time.UTC)
	publisher := &researchBudgetPublisherSpy{}
	h := &connectHandler{
		researchBudget: publisher,
		mandateRead:    researchBudgetMandateSpy{value: mandate.Mandate{ID: "mandate-1", BudgetID: "budget-1", AgentSubject: "agent", AgentRunID: "run-1", IssuedAt: issued}},
		headroom:       researchBudgetHeadroomSpy{value: budget.Headroom{BudgetID: "budget-1", Currency: "USD", TotalCapMinor: 100}},
	}
	err := h.publishResearchBudgetEvidence(context.Background(), identity.Claims{RunID: "run-1", Subject: "agent"}, authorization.Record{BudgetID: "budget-1", MandateID: "mandate-1", Currency: "EUR"}, settlement.Record{ID: "settlement", Rail: "x402", Outcome: settlement.OutcomeSettled, AmountMinor: 1, Currency: "EUR", ExternalID: "0xfeed", OccurredAt: issued})
	require.ErrorContains(t, err, "do not match the authorized run budget")
	require.Empty(t, publisher.snapshots)
	require.Empty(t, publisher.settlements)
}
