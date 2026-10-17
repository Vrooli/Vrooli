package backlog

import (
	"errors"
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"
)

type effortCandidateQualifyRequest struct {
	Runner       string `json:"runner"`
	Model        string `json:"model"`
	Metered      bool   `json:"metered,omitempty"`
	BillingKnown bool   `json:"billing_known,omitempty"`
	Funded       bool   `json:"funded,omitempty"`
	Qualified    bool   `json:"qualified,omitempty"`
}

type effortCandidateQualifyResponse struct {
	EffortID      string                          `json:"effort_id"`
	Qualification identity.CandidateQualification `json:"qualification"`
}

// QualifyEffortCandidate decides whether a last-resort fallback may select one
// proposed runner/model. A withheld, unbound, unqualified or unknown-billing
// candidate is refused before any dispatch effect.
func (h *Handler) QualifyEffortCandidate(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] qualify-effort-candidate"
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, action, apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, action, apierr.Internal("effort control service is not configured"))
		return
	}
	request := effortCandidateQualifyRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	qualification, err := h.effortControl.QualifyLastResortCandidate(effortID, identity.Candidate{
		Runner:       request.Runner,
		Model:        request.Model,
		Metered:      request.Metered,
		BillingKnown: request.BillingKnown,
		Funded:       request.Funded,
		Qualified:    request.Qualified,
	})
	if err != nil {
		mapEffortCandidateError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortCandidateQualifyResponse{EffortID: effortID, Qualification: qualification})
}

// mapEffortCandidateError translates candidate-qualification sentinels into
// transport status. A withheld or unfunded candidate is forbidden; an
// unqualified or unbound candidate is a conflict with the reviewed binding.
func mapEffortCandidateError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierr.MapError(w, action, apierr.NotFound("effort control revision not found"))
	case errors.Is(err, identity.ErrCandidateIdentityRequired):
		apierr.MapError(w, action, apierr.BadRequest("candidate runner and model are required"))
	case errors.Is(err, identity.ErrCandidateWithheld):
		apierr.MapError(w, action, apierr.Forbidden("%s", err.Error()))
	case errors.Is(err, identity.ErrCandidateBillingUnknown):
		apierr.MapError(w, action, apierr.Forbidden("%s", err.Error()))
	case errors.Is(err, identity.ErrCandidateUnqualified), errors.Is(err, identity.ErrCandidateUnbound):
		apierr.MapError(w, action, apierr.Conflict("%s", err.Error()))
	default:
		apierr.MapError(w, action, apierr.BadRequest("%s", err.Error()))
	}
}
