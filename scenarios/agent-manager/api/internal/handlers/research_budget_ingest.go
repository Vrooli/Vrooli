package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/researchbudget"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

const (
	researchBudgetSnapshotPath   = "/api/v1/research-budgets/{run_id}/snapshot"
	researchBudgetSettlementPath = "/api/v1/research-budgets/{run_id}/settlements"
	researchBudgetOperation      = "research-budget.ingest"
	researchBudgetMaxBody        = 32 << 10
)

// RegisterResearchBudgetIngestion exposes Treasury's narrow, authenticated
// reporting boundary. It never accepts agent or operator writes: only a
// verified Treasury service identity with an exact run-scoped operation grant
// can record immutable evidence.
func RegisterResearchBudgetIngestion(router *mux.Router, ledger researchbudget.Ledger) {
	router.HandleFunc(researchBudgetSnapshotPath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizeResearchBudgetIngest(w, r) {
			return
		}
		runID, ok := researchBudgetRunID(w, r)
		if !ok {
			return
		}
		var body struct {
			TreasuryBudgetID string    `json:"treasury_budget_id"`
			MandateID        string    `json:"mandate_id"`
			AgentSubject     string    `json:"agent_subject"`
			CapMinor         int64     `json:"cap_minor"`
			Currency         string    `json:"currency"`
			UnitCode         string    `json:"unit_code"`
			PolicyRevision   string    `json:"policy_revision"`
			CreatedAt        time.Time `json:"created_at"`
		}
		if !decodeResearchBudgetBody(w, r, &body) {
			return
		}
		if body.CreatedAt.IsZero() {
			http.Error(w, "budget snapshot creation time is required", http.StatusBadRequest)
			return
		}
		err := ledger.PutBudget(r.Context(), researchbudget.Budget{RunID: runID.String(), TreasuryBudgetID: body.TreasuryBudgetID, MandateID: body.MandateID, AgentSubject: body.AgentSubject, CapMinor: body.CapMinor, Currency: body.Currency, UnitCode: body.UnitCode, PolicyRevision: body.PolicyRevision, CreatedAt: body.CreatedAt})
		writeResearchBudgetResult(w, err)
	}).Methods(http.MethodPost)

	router.HandleFunc(researchBudgetSettlementPath, func(w http.ResponseWriter, r *http.Request) {
		if !authorizeResearchBudgetIngest(w, r) {
			return
		}
		runID, ok := researchBudgetRunID(w, r)
		if !ok {
			return
		}
		var body struct {
			ID                string    `json:"id"`
			SettlementID      string    `json:"settlement_id"`
			TreasuryReceiptID string    `json:"treasury_receipt_id"`
			IntegrationID     string    `json:"integration_id"`
			Category          string    `json:"category"`
			AmountMinor       int64     `json:"amount_minor"`
			Currency          string    `json:"currency"`
			UnitCode          string    `json:"unit_code"`
			ResultRef         string    `json:"result_ref"`
			BenefitState      string    `json:"benefit_state"`
			BenefitSummary    string    `json:"benefit_summary"`
			SettledAt         time.Time `json:"settled_at"`
		}
		if !decodeResearchBudgetBody(w, r, &body) {
			return
		}
		err := ledger.RecordSpend(r.Context(), researchbudget.Spend{ID: body.ID, RunID: runID.String(), SettlementID: body.SettlementID, TreasuryReceiptID: body.TreasuryReceiptID, IntegrationID: body.IntegrationID, Category: body.Category, AmountMinor: body.AmountMinor, Currency: body.Currency, UnitCode: body.UnitCode, ResultRef: body.ResultRef, BenefitState: body.BenefitState, BenefitSummary: body.BenefitSummary, SettledAt: body.SettledAt})
		writeResearchBudgetResult(w, err)
	}).Methods(http.MethodPost)
}

func authorizeResearchBudgetIngest(w http.ResponseWriter, r *http.Request) bool {
	p, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !p.IsVerified() || p.Kind != identity.ActorService || p.Subject != orchestration.TreasuryServiceSubject {
		http.Error(w, "verified Treasury service identity required", http.StatusForbidden)
		return false
	}
	if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
		http.Error(w, "Treasury write capability required", http.StatusForbidden)
		return false
	}
	runID, err := uuid.Parse(mux.Vars(r)["run_id"])
	if err != nil {
		http.Error(w, "valid run UUID required", http.StatusBadRequest)
		return false
	}
	if _, err := authn.RequireOperation(r.Context(), "agent-manager:write", researchBudgetOperation, "agent-manager:run/"+runID.String()); err != nil {
		if errors.Is(err, authn.ErrNeedsApproval) {
			return requestResearchBudgetIngestApproval(w, r, runID)
		}
		http.Error(w, "Treasury operation grant does not cover this run", http.StatusForbidden)
		return false
	}
	return true
}

// requestResearchBudgetIngestApproval asks the existing human decision path to
// review one exact run. The request itself writes no grant; only the separate
// Notification Hub human resolution can issue the one-hour, exact-target grant.
func requestResearchBudgetIngestApproval(w http.ResponseWriter, r *http.Request, runID uuid.UUID) bool {
	p, ok := identity.PrincipalFromContext(r.Context())
	if !ok || p.Kind != identity.ActorService || !p.IsVerified() || p.Subject != orchestration.TreasuryServiceSubject {
		http.Error(w, "verified Treasury service identity required", http.StatusForbidden)
		return false
	}
	raiser, ok := ApprovalRaiserFromContext(r.Context())
	if !ok {
		http.Error(w, "Treasury operation grant does not cover this run", http.StatusForbidden)
		return false
	}
	verifier, ok := raiser.(interface {
		IsCF7ResearchBudgetTestRun(context.Context, uuid.UUID) bool
	})
	if !ok || !verifier.IsCF7ResearchBudgetTestRun(r.Context(), runID) {
		http.Error(w, "CF7 test-only run required for research-budget approval", http.StatusForbidden)
		return false
	}
	target := "agent-manager:run/" + runID.String()
	raised, err := raiser.RaiseAuthorizationApproval(r.Context(), orchestration.AuthorizationApprovalRequest{
		Subject: p.Subject, Operation: researchBudgetOperation, Target: target,
		Kind: domain.PendingAuthorizationKindOneTime,
	})
	if errors.Is(err, orchestration.ErrAuthorizationFlood) {
		writeServiceAdmissionRefusal(w, http.StatusTooManyRequests, "approval_raise_limited")
		return false
	}
	if err != nil || !raised {
		writeServiceAdmissionRefusal(w, http.StatusForbidden, "approval_raise_failed")
		return false
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "approval_pending", "subject": p.Subject, "operation": researchBudgetOperation,
		"target": target, "detail": "human approval requested for this exact test-run report; no grant has been issued",
	})
	return false
}

func researchBudgetRunID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(mux.Vars(r)["run_id"])
	if err != nil {
		http.Error(w, "valid run UUID required", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return id, true
}

func decodeResearchBudgetBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		http.Error(w, "request body required", http.StatusBadRequest)
		return false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, researchBudgetMaxBody+1))
	if err != nil || len(data) == 0 || len(data) > researchBudgetMaxBody {
		http.Error(w, "invalid or oversized request body", http.StatusBadRequest)
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		http.Error(w, "invalid research budget evidence", http.StatusBadRequest)
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		http.Error(w, "request must contain one JSON value", http.StatusBadRequest)
		return false
	}
	return true
}

func writeResearchBudgetResult(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if errors.Is(err, researchbudget.ErrConflict) || strings.Contains(strings.ToLower(err.Error()), "unique constraint") || strings.Contains(strings.ToLower(err.Error()), "does not match run budget unit") {
		http.Error(w, "research budget evidence conflicts with retained run evidence", http.StatusConflict)
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), "no rows") || strings.Contains(strings.ToLower(err.Error()), "foreign key constraint") {
		http.Error(w, "run-bound research budget evidence requires an existing run snapshot", http.StatusConflict)
		return
	}
	http.Error(w, "invalid research budget evidence", http.StatusBadRequest)
}
