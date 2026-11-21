package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"notification-hub/internal/hub"
)

const (
	ProviderDecisionAudience = "scenario-authenticator:notification-hub-provider-decision"
	ProviderDecisionSubject  = "ai-gateway"
	ProviderDecisionScope    = "notification-hub:provider-decision"
	ProviderDecisionRoute    = "/api/v1/internal/provider-decisions/lookup"
	providerDecisionSource   = "ai-gateway"
	providerDecisionEvent    = "ai_gateway.provider_decision.decision_requested.v1"
)

type ProviderDecisionAskReader interface {
	GetAskByRequester(context.Context, string, string, string) (hub.AskRecord, error)
}

// ProviderDecisionLookupRequest binds a read to the stable requester/event
// reference and canonical digest delivered by the provider-decision proposal.
type ProviderDecisionLookupRequest struct {
	RequesterRef string `json:"requester_ref"`
	Digest       string `json:"canonical_terms_sha256"`
	Scope        string `json:"scope"`
}

type ProviderDecisionLookupResult struct {
	AskID                    string `json:"ask_id"`
	Source                   string `json:"source"`
	EventType                string `json:"event_type"`
	Requester                string `json:"requester"`
	EventID                  string `json:"event_id"`
	Digest                   string `json:"canonical_terms_sha256"`
	Scope                    string `json:"scope"`
	State                    string `json:"state"`
	Answer                   string `json:"answer,omitempty"`
	AnsweredByIdentityKind   string `json:"answered_by_identity_kind"`
	AnsweredByIdentitySource string `json:"answered_by_identity_source,omitempty"`
	CreatedAt                string `json:"created_at"`
	AnsweredAt               string `json:"answered_at"`
	Deadline                 string `json:"deadline"`
}

// ProviderDecisionLookupHandler is private to one Gateway service principal.
// It fails closed when the verifier or reader is absent and only returns an
// exact immutable request match; it has no write operations.
func ProviderDecisionLookupHandler(reader ProviderDecisionAskReader, verifier authn.TokenVerifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if reader == nil || verifier == nil {
			http.Error(w, "provider decision lookup unavailable", http.StatusServiceUnavailable)
			return
		}
		principal, err := bearerPrincipal(r, verifier)
		if err != nil || principal.Kind != apiidentity.ActorService || principal.Subject != ProviderDecisionSubject || !principal.Verified || !scopecatalog.MatchCapability(principal.Scopes, ProviderDecisionScope) || principal.Audience != ProviderDecisionAudience {
			http.Error(w, "service identity required", http.StatusUnauthorized)
			return
		}
		var input ProviderDecisionLookupRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "invalid provider decision lookup", http.StatusBadRequest)
			return
		}
		requesterRef := strings.TrimSpace(input.RequesterRef)
		if requesterRef == "" || !validDigest(input.Digest) || (input.Scope != hub.ProviderDecisionScopeRule && input.Scope != hub.ProviderDecisionScopeAgreement) {
			http.Error(w, "invalid provider decision lookup", http.StatusBadRequest)
			return
		}
		ask, err := reader.GetAskByRequester(r.Context(), providerDecisionSource, providerDecisionEvent, requesterRef)
		if err != nil {
			http.Error(w, "provider decision not found", http.StatusNotFound)
			return
		}
		if ask.Source != providerDecisionSource || ask.SourceEventType != providerDecisionEvent || ask.RequesterRef != requesterRef || ask.ProviderDecisionContext == nil || ask.ProviderDecisionContext.SchemaVersion != 1 || ask.ProviderDecisionContext.CanonicalTermsSHA256 != input.Digest || ask.ProviderDecisionContext.Scope != input.Scope || !hasApproveRejectOptions(ask.Options) {
			http.Error(w, "provider decision identity mismatch", http.StatusNotFound)
			return
		}
		writeJSON(w, ProviderDecisionLookupResult{
			AskID: ask.ID, Source: ask.Source, EventType: ask.SourceEventType,
			Requester: ask.RequesterRef, EventID: ask.RequesterRef,
			Digest: ask.ProviderDecisionContext.CanonicalTermsSHA256, Scope: ask.ProviderDecisionContext.Scope,
			State: ask.State, Answer: ask.Answer, AnsweredByIdentityKind: ask.AnsweredByIdentityKind,
			AnsweredByIdentitySource: ask.AnsweredByIdentitySource, CreatedAt: ask.CreatedAt, Deadline: ask.Deadline,
			AnsweredAt: ask.AnsweredAt,
		})
	})
}

func bearerPrincipal(r *http.Request, verifier authn.TokenVerifier) (apiidentity.Principal, error) {
	if r == nil || verifier == nil {
		return apiidentity.Principal{}, errors.New("service identity verifier unavailable")
	}
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	token, found := strings.CutPrefix(value, "Bearer ")
	if !found || strings.TrimSpace(token) == "" || strings.ContainsAny(strings.TrimSpace(token), " \t\r\n") {
		return apiidentity.Principal{}, errors.New("bearer token required")
	}
	return verifier.Verify(r.Context(), strings.TrimSpace(token))
}

func hasApproveRejectOptions(options []hub.AskOption) bool {
	approve, reject := false, false
	for _, option := range options {
		switch option.Key {
		case "approve":
			approve = true
		case "reject":
			reject = true
		default:
			return false
		}
	}
	return approve && reject && len(options) == 2
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
