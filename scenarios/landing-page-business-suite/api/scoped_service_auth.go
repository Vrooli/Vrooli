package main

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/principalrole"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
)

func (s *Server) verifiedLPBSRole(r *http.Request, roleSource string) bool {
	if s.servicePrincipalVerifier == nil {
		return false
	}
	for _, role := range []struct {
		definition principalrole.Role
		source     string
	}{
		{lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop"},
		{lpbsidentity.DeploymentManagerRelease, "deployment-manager"},
		{lpbsidentity.CommandCenterAnalytics, "command-center"},
		{lpbsidentity.CommandCenterRemoteProfiles, "command-center"},
		{lpbsidentity.ScenarioToCloudRemoteProfiles, "scenario-to-cloud"},
		{lpbsidentity.LocalAdminRecovery, "landing-page-business-suite"},
		{lpbsidentity.TreasuryAcceptedReceiptCredit, "treasury"},
	} {
		if role.source != roleSource {
			continue
		}
		verified, err := role.definition.VerifiedSource(r.Context(), s.servicePrincipalVerifier, r.Header)
		if err == nil && verified == role.source {
			return true
		}
	}
	return false
}

func (s *Server) verifiedLPBSRoleExact(r *http.Request, role principalrole.Role, source string) bool {
	if s.servicePrincipalVerifier == nil {
		return false
	}
	verified, err := role.VerifiedSource(r.Context(), s.servicePrincipalVerifier, r.Header)
	return err == nil && verified == source
}

func (s *Server) legacyServiceBearerMatches(r *http.Request) bool {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	provided, found := strings.CutPrefix(header, "Bearer ")
	provided = strings.TrimSpace(provided)
	if !found || provided == "" {
		return false
	}
	binding, err := administration.ResolveCredentialBinding("LPBS_" + "SERVICE_SECRET")
	return err == nil && binding.Scope == "service:lpbs-bootstrap" && len(provided) == len(binding.Value) && subtle.ConstantTimeCompare([]byte(provided), []byte(binding.Value)) == 1
}

func (s *Server) legacyServiceBearerMatchesForUnissuedRole(r *http.Request, role principalrole.Role, source string) bool {
	authority, err := credentialauthority.Default()
	if err != nil {
		return false
	}
	identity, err := credentialauthority.ParseIdentity(role.CredentialIdentity(source))
	if err != nil {
		return false
	}
	status := authority.Status(identity, "credential")
	return string(status.ProviderState) == "available" && !status.Configured && s.legacyServiceBearerMatches(r)
}

// requireDeploymentPrincipal is limited to LPBS's cross-scenario deployment
// APIs. The old broad bearer remains a migration bridge; an admin session is
// still accepted, while each machine caller's new token is source and scope
// checked before this handler can run.
func (s *Server) requireDeploymentPrincipal(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
			if s.verifiedLPBSRole(r, "scenario-to-desktop") || s.verifiedLPBSRole(r, "deployment-manager") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.DeploymentManagerRelease, "deployment-manager") {
				next(w, r)
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "Invalid service authorization.", ApiErrorTypeUnauthorized)
			return
		}
		s.requireAdmin(next)(w, r)
	}
}

// requireScenarioToDesktopDeployment narrows LPBS catalog publication to the
// S2D owner. Deployment Manager's independent role cannot publish desktop data.
func (s *Server) requireScenarioToDesktopDeployment(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
			if s.verifiedLPBSRoleExact(r, lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop") {
				next(w, r)
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "Invalid service authorization.", ApiErrorTypeUnauthorized)
			return
		}
		s.requireAdminStepUp(next)(w, r)
	}
}

// requireAgentManagerBrowserSessionSettlement accepts only the narrow AM role
// that closes an existing browser-session reservation after proof verification.
func (s *Server) requireAgentManagerBrowserSessionSettlement(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.verifiedLPBSRoleExact(r, lpbsidentity.AgentManagerBrowserSessionSettlement, "agent-manager") {
			next(w, r)
			return
		}
		writeJSONError(w, http.StatusUnauthorized, "Invalid browser session settlement authorization.", ApiErrorTypeUnauthorized)
	}
}

func (s *Server) requireRemoteProfileReader(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
			if s.verifiedLPBSRole(r, "scenario-to-desktop") || s.verifiedLPBSRole(r, "deployment-manager") || s.verifiedLPBSRole(r, "scenario-to-cloud") || s.verifiedLPBSRoleExact(r, lpbsidentity.CommandCenterRemoteProfiles, "command-center") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.DeploymentManagerRelease, "deployment-manager") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.ScenarioToCloudRemoteProfiles, "scenario-to-cloud") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.CommandCenterRemoteProfiles, "command-center") {
				next(w, r)
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "Invalid service authorization.", ApiErrorTypeUnauthorized)
			return
		}
		s.requireAdmin(next)(w, r)
	}
}

func (s *Server) requireRemoteProfileReconcile(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
			if s.verifiedLPBSRole(r, "scenario-to-cloud") || s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.ScenarioToCloudRemoteProfiles, "scenario-to-cloud") {
				next(w, r)
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "Invalid service authorization.", ApiErrorTypeUnauthorized)
			return
		}
		s.requireAdmin(next)(w, r)
	}
}

func (s *Server) requireAdminRecovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
			if s.verifiedLPBSRoleExact(r, lpbsidentity.LocalAdminRecovery, "landing-page-business-suite") {
				next(w, r.WithContext(context.WithValue(r.Context(), servicePrincipalContextKey, "lpbs-admin-recovery")))
				return
			}
			if s.legacyServiceBearerMatchesForUnissuedRole(r, lpbsidentity.LocalAdminRecovery, "landing-page-business-suite") {
				next(w, r.WithContext(context.WithValue(r.Context(), servicePrincipalContextKey, "lpbs-legacy-bootstrap")))
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "Invalid service authorization.", ApiErrorTypeUnauthorized)
			return
		}
		s.requireAdmin(next)(w, r)
	}
}
