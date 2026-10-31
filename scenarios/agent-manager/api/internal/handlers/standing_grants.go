package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// StandingGrantAdmin is the operator-facing surface for standing grants:
// create, revoke, and list the durable records. Evaluating grants for a caller
// is the authority's job, not this handler's.
type StandingGrantAdmin interface {
	CreateStandingGrant(context.Context, orchestration.StandingGrantRequest) (*domain.StandingGrant, error)
	RevokeStandingGrant(context.Context, string) (bool, error)
	ListStandingGrants(context.Context, int) ([]*domain.StandingGrant, error)
	ListStandingGrantsForApprover(context.Context, string, int) ([]*domain.StandingGrant, error)
}

// createStandingGrantBody is the operator's request. The approver is NOT taken
// from the body — it is the verified human's own subject — so an operator can
// only ever make themselves the accountable owner, never name another.
type createStandingGrantBody struct {
	Subject           string  `json:"subject"`
	SessionID         string  `json:"session_id,omitempty"`
	MaxConcurrentRuns int     `json:"max_concurrent_runs,omitempty"`
	MaxRunBudgetUSD   float64 `json:"max_run_budget_usd,omitempty"`
	TTLSeconds        int64   `json:"ttl_seconds"`
}

const standingGrantReadLimit = 64
const standingGrantReadFetchLimit = standingGrantReadLimit + 1

const (
	promptManagerAdmissionDiagnosticSubject = "prompt-manager"
	promptManagerAdmissionDiagnosticTarget  = "agent-manager:service/prompt-manager"
	promptManagerAdmissionDiagnosticPath    = "/api/v1/standing-grants/diagnose/prompt-manager-run-create"
)

// RegisterStandingGrants mounts the human-only create/revoke endpoints.
//
// SECURITY: writing a standing grant is the human-in-the-loop lynchpin of the
// whole model — it mints durable authority. Every write here requires a verified
// INDEPENDENT human operator with agent-manager:write, re-checked in the handler
// as defense in depth on top of AgentOperationMiddleware (which already refuses
// agents and services on these paths). An agent or service can never create or
// revoke a grant, so a caller can never grant itself authority.
func RegisterStandingGrants(r *mux.Router, admin StandingGrantAdmin) {
	if admin == nil {
		return
	}
	r.HandleFunc("/api/v1/standing-grants", func(w http.ResponseWriter, r *http.Request) {
		approver, err := requireOperatorForGrantWrite(r)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		var body createStandingGrantBody
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid standing grant request"})
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must be a single JSON object"})
			return
		}
		grant, err := admin.CreateStandingGrant(r.Context(), orchestration.StandingGrantRequest{
			Subject:   body.Subject,
			Approver:  approver,
			SessionID: body.SessionID,
			Limits:    authn.StandingGrantLimits{MaxConcurrentRuns: body.MaxConcurrentRuns, MaxRunBudgetUSD: body.MaxRunBudgetUSD},
			TTL:       time.Duration(body.TTLSeconds) * time.Second,
		})
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id": grant.ID, "subject": grant.Subject, "approver": grant.Grant.Approver,
			"operations": grant.Grant.Operations, "targets": grant.Grant.Targets,
			"max_concurrent_runs": grant.Limits.MaxConcurrentRuns, "max_run_budget_usd": grant.Limits.MaxRunBudgetUSD,
			"expires_at": grant.ExpiresAt,
		})
	}).Methods(http.MethodPost)

	r.HandleFunc("/api/v1/standing-grants/{id}/revoke", func(w http.ResponseWriter, r *http.Request) {
		if _, err := requireOperatorForGrantWrite(r); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		transitioned, err := admin.RevokeStandingGrant(r.Context(), mux.Vars(r)["id"])
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"revoked": transitioned})
	}).Methods(http.MethodPost)

	// Listing is read-only. Humans retain global read access; a verified agent may
	// inspect only grants attributed to its verified owner.
	r.HandleFunc("/api/v1/standing-grants", func(w http.ResponseWriter, r *http.Request) {
		principal, owner, err := requireStandingGrantRead(r)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		var grants []*domain.StandingGrant
		if principal.Kind == identity.ActorAgent {
			grants, err = admin.ListStandingGrantsForApprover(r.Context(), owner, standingGrantReadFetchLimit)
		} else {
			grants, err = admin.ListStandingGrants(r.Context(), standingGrantReadFetchLimit)
		}
		if err != nil {
			writeError(w, r, err)
			return
		}
		hasMore := len(grants) > standingGrantReadLimit
		if hasMore {
			grants = grants[:standingGrantReadLimit]
		}
		out := make([]map[string]any, 0, len(grants))
		for _, g := range grants {
			if g == nil || (owner != "" && (g.ApprovedBy != owner || g.Grant == nil || g.Grant.Approver != owner)) {
				continue
			}
			row := map[string]any{
				"id": g.ID, "subject": g.Subject, "state": g.State,
				"max_concurrent_runs": g.Limits.MaxConcurrentRuns, "max_run_budget_usd": g.Limits.MaxRunBudgetUSD,
				"approved_by": g.ApprovedBy, "approved_at": g.ApprovedAt, "expires_at": g.ExpiresAt,
			}
			if g.Grant != nil {
				row["approver"] = g.Grant.Approver
				row["operations"] = g.Grant.Operations
				row["targets"] = g.Grant.Targets
			}
			if g.RevokedAt != nil {
				row["revoked_at"] = g.RevokedAt
			}
			out = append(out, row)
		}
		writeJSON(w, http.StatusOK, map[string]any{"grants": out, "has_more": hasMore})
	}).Methods(http.MethodGet)
}

// RegisterPromptManagerAdmissionDiagnostic exposes the fixed Prompt Manager
// run.create decision and bounded counted-run identities for concurrency
// refusals. It is a read-only agent.investigate operation and never calls the
// approval raiser.
func RegisterPromptManagerAdmissionDiagnostic(r *mux.Router, authority *StandingGrantAuthority) {
	r.HandleFunc(promptManagerAdmissionDiagnosticPath, func(w http.ResponseWriter, r *http.Request) {
		if _, err := authn.RequireOperation(r.Context(), "agent-manager:write", "agent.investigate", promptManagerAdmissionDiagnosticTarget); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "authorized investigation identity required"})
			return
		}
		diagnostic, err := authority.diagnoseOperationWithRunDetails(r.Context(), promptManagerAdmissionDiagnosticSubject, "run.create", RunCreationTarget, true)
		if err != nil {
			if diagnostic.ObservedAt.IsZero() {
				diagnostic.ObservedAt = time.Now().UTC()
			}
			writeJSON(w, http.StatusServiceUnavailable, diagnostic)
			return
		}
		writeJSON(w, http.StatusOK, diagnostic)
	}).Methods(http.MethodGet)
}

func requireStandingGrantRead(r *http.Request) (identity.Principal, string, error) {
	p, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !p.IsVerified() {
		return identity.Principal{}, "", errors.New("verified identity required")
	}
	switch p.Kind {
	case identity.ActorHuman:
		// Preserve the independent-human boundary when a request carries agent
		// provenance alongside a human identity.
		human, err := authn.RequireIndependentHuman(r.Context())
		if err != nil {
			return identity.Principal{}, "", err
		}
		if _, err := authn.RequireCapability(r.Context(), "agent-manager:read"); err != nil {
			return identity.Principal{}, "", err
		}
		return human, "", err
	case identity.ActorAgent:
		owner := strings.TrimSpace(p.OwnerSubject())
		if owner == "" {
			return identity.Principal{}, "", errors.New("verified agent owner is required")
		}
		return p, owner, nil
	default:
		return identity.Principal{}, "", errors.New("verified human or owned agent identity required")
	}
}

// requireOperatorForGrantWrite enforces the verified-independent-human +
// capability gate for a standing-grant write and returns the approver subject.
// It refuses a forwarded agent credential even when a human is also present, so
// the approver is always the independent human in the loop.
func requireOperatorForGrantWrite(r *http.Request) (string, error) {
	p, err := authn.RequireIndependentHuman(r.Context())
	if err != nil {
		return "", err
	}
	if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
		return "", err
	}
	return p.Subject, nil
}
