// Package desktoplink exposes the explicit LPBS-to-desktop account-link flow.
// It never returns or accepts an LPBS website session as the local identity.
package desktoplink

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	link "landing-page-business-suite-api/internal/desktoplink"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

type LocalIdentityVerifier func(context.Context, string) (identity.Principal, error)

type Dependencies struct {
	Service                  *link.Service
	LPBSUserID               func(context.Context) string
	ResolveBusinessAccountID func(context.Context, string, string) (string, error)
	VerifyLocalIdentity      LocalIdentityVerifier
	IssueLease               func(context.Context, link.Link) (link.EntitlementLease, error)
	WriteAudit               func(string, map[string]interface{})
}

type authorizationRequest struct {
	BusinessAccountID string   `json:"business_account_id"`
	InstallationID    string   `json:"installation_id"`
	Resource          string   `json:"resource"`
	Audience          string   `json:"audience"`
	Scopes            []string `json:"scopes"`
	CodeChallenge     string   `json:"code_challenge"`
	CodeChallengeMode string   `json:"code_challenge_method"`
	RedirectURI       string   `json:"redirect_uri"`
}

type redeemRequest struct {
	Code           string `json:"code"`
	CodeVerifier   string `json:"code_verifier"`
	InstallationID string `json:"installation_id"`
	Resource       string `json:"resource"`
}

type revokeRequest struct {
	InstallationID string `json:"installation_id"`
	Resource       string `json:"resource"`
}

// RegisterRoutes mounts explicit account-link and unlink operations. The LPBS
// issue/revoke route is wrapped by the caller's user-auth middleware; redeem
// and local revoke verify a scenario-authenticator proof independently.
func RegisterRoutes(router *mux.Router, deps Dependencies, requireLPBSAuth func(http.HandlerFunc) http.HandlerFunc) {
	router.HandleFunc("/api/v1/desktop/links", requireLPBSAuth(issue(deps))).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/desktop/links/redeem", redeem(deps)).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/desktop/links/local", statusLocal(deps)).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/desktop/links", requireLPBSAuth(revokeLPBS(deps))).Methods(http.MethodDelete)
	router.HandleFunc("/api/v1/desktop/links/local", revokeLocal(deps)).Methods(http.MethodDelete)
}

func issue(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request authorizationRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil || request.CodeChallengeMode != "S256" {
			writeError(w, http.StatusBadRequest, "invalid desktop link request")
			return
		}
		userID := strings.TrimSpace(deps.LPBSUserID(r.Context()))
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "authenticated LPBS user required")
			return
		}
		businessAccountID := strings.TrimSpace(request.BusinessAccountID)
		if deps.ResolveBusinessAccountID != nil {
			var err error
			businessAccountID, err = deps.ResolveBusinessAccountID(r.Context(), userID, businessAccountID)
			if err != nil || strings.TrimSpace(businessAccountID) == "" {
				writeError(w, http.StatusForbidden, "business account selection rejected")
				return
			}
		}
		if businessAccountID == "" {
			// Compatibility for protocol-level tests and older callers. Production
			// composition always supplies ResolveBusinessAccountID.
			businessAccountID = userID
		}
		code, authorization, err := deps.Service.Issue(r.Context(), link.AuthorizationRequest{
			LPBSUserID: userID, BusinessAccountID: businessAccountID, InstallationID: request.InstallationID,
			Resource: request.Resource, Audience: request.Audience, Scopes: request.Scopes,
			CodeChallenge: request.CodeChallenge, RedirectURI: request.RedirectURI,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, "desktop link request rejected")
			return
		}
		if deps.WriteAudit != nil {
			deps.WriteAudit("desktop_link_authorization_issued", map[string]interface{}{"user_id": userID, "business_account_id": authorization.BusinessAccountID, "installation_id": authorization.InstallationID, "resource": authorization.Resource})
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{"code": code, "expires_at": authorization.ExpiresAt.UTC().Format(time.RFC3339), "business_account_id": authorization.BusinessAccountID, "installation_id": authorization.InstallationID, "resource": authorization.Resource, "audience": authorization.Audience, "scopes": authorization.Scopes})
	}
}

func redeem(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request redeemRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil || deps.VerifyLocalIdentity == nil {
			writeError(w, http.StatusBadRequest, "invalid desktop redemption request")
			return
		}
		proof := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		principal, err := deps.VerifyLocalIdentity(r.Context(), proof)
		if err != nil || !principal.IsHuman() || principal.Source != identity.SourceScenarioAuthenticator {
			writeError(w, http.StatusUnauthorized, "verified local identity required")
			return
		}
		connected, err := deps.Service.Redeem(r.Context(), request.Code, request.CodeVerifier, principal.Subject, request.InstallationID, request.Resource)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "desktop link authorization rejected")
			return
		}
		if deps.WriteAudit != nil {
			deps.WriteAudit("desktop_link_redeemed", map[string]interface{}{"link_id": connected.ID, "local_principal": connected.LocalPrincipal, "business_account_id": connected.BusinessAccountID, "installation_id": connected.InstallationID, "resource": connected.Resource})
		}
		response := map[string]interface{}{"link": connected, "local_principal": connected.LocalPrincipal, "business_account_id": connected.BusinessAccountID}
		if deps.IssueLease != nil {
			lease, leaseErr := deps.IssueLease(r.Context(), connected)
			if leaseErr != nil {
				writeError(w, http.StatusServiceUnavailable, "entitlement lease unavailable")
				return
			}
			response["entitlement_lease"] = lease
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func revokeLPBS(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request revokeRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			writeError(w, http.StatusBadRequest, "invalid desktop unlink request")
			return
		}
		userID := strings.TrimSpace(deps.LPBSUserID(r.Context()))
		if err := deps.Service.RevokeForLPBS(r.Context(), userID, request.InstallationID, request.Resource, userID); err != nil {
			writeError(w, http.StatusBadRequest, "desktop link could not be revoked")
			return
		}
		if deps.WriteAudit != nil {
			deps.WriteAudit("desktop_link_revoked", map[string]interface{}{"actor": userID, "installation_id": request.InstallationID, "resource": request.Resource, "side": "lpbs"})
		}
		writeJSON(w, http.StatusNoContent, nil)
	}
}

func revokeLocal(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request revokeRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil || deps.VerifyLocalIdentity == nil {
			writeError(w, http.StatusBadRequest, "invalid local unlink request")
			return
		}
		proof := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		principal, err := deps.VerifyLocalIdentity(r.Context(), proof)
		if err != nil || !principal.IsHuman() || principal.Source != identity.SourceScenarioAuthenticator {
			writeError(w, http.StatusUnauthorized, "verified local identity required")
			return
		}
		if err := deps.Service.RevokeForLocal(r.Context(), principal.Subject, request.InstallationID, request.Resource, principal.Subject); err != nil {
			writeError(w, http.StatusBadRequest, "desktop link could not be revoked")
			return
		}
		if deps.WriteAudit != nil {
			deps.WriteAudit("desktop_link_revoked", map[string]interface{}{"actor": principal.Subject, "installation_id": request.InstallationID, "resource": request.Resource, "side": "local"})
		}
		writeJSON(w, http.StatusNoContent, nil)
	}
}

func statusLocal(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.VerifyLocalIdentity == nil {
			writeError(w, http.StatusServiceUnavailable, "local identity provider unavailable")
			return
		}
		proof := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		principal, err := deps.VerifyLocalIdentity(r.Context(), proof)
		if err != nil || !principal.IsHuman() || principal.Source != identity.SourceScenarioAuthenticator {
			writeError(w, http.StatusUnauthorized, "verified local identity required")
			return
		}
		connected, err := deps.Service.StatusForLocal(r.Context(), principal.Subject, r.URL.Query().Get("installation_id"), r.URL.Query().Get("resource"))
		if err != nil {
			writeError(w, http.StatusNotFound, "desktop link not found")
			return
		}
		if connected.RevokedAt != nil {
			writeError(w, http.StatusForbidden, "desktop link revoked")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"link": connected, "state": "active"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	if status != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
