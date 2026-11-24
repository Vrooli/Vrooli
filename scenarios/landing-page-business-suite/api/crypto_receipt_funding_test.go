package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/principalrole"
)

type staticServicePrincipalVerifier struct{ principal apiidentity.Principal }

func (v staticServicePrincipalVerifier) Verify(context.Context, string) (apiidentity.Principal, error) {
	return v.principal, nil
}

var _ authn.TokenVerifier = staticServicePrincipalVerifier{}

func TestAcceptedReceiptGateAcceptsOnlyItsScopedServicePrincipal(t *testing.T) {
	call := func(principal apiidentity.Principal) *httptest.ResponseRecorder {
		server := &Server{servicePrincipalVerifier: staticServicePrincipalVerifier{principal: principal}}
		handler := server.requireTreasuryReceiptService(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/crypto-funding/accepted-receipts", nil)
		request.Header.Set("Authorization", "Bearer synthetic-token")
		response := httptest.NewRecorder()
		handler(response, request)
		return response
	}
	base := apiidentity.Principal{Kind: apiidentity.ActorService, Subject: lpbsidentity.TreasuryAcceptedReceiptCredit.Principal("treasury"), Verified: true, Audience: lpbsidentity.TreasuryAcceptedReceiptCredit.Audience, Scopes: []string{lpbsidentity.TreasuryAcceptedReceiptCredit.Scope}}
	require.Equal(t, http.StatusNoContent, call(base).Code)

	wrongScope := base
	wrongScope.Scopes = []string{"lpbs:admin.write"}
	require.Equal(t, http.StatusUnauthorized, call(wrongScope).Code)

	wrongPrincipal := base
	wrongPrincipal.Subject = "deployment-manager-lpbs-credit-grant"
	require.Equal(t, http.StatusUnauthorized, call(wrongPrincipal).Code)

	invalid := base
	invalid.Verified = false
	require.Equal(t, http.StatusUnauthorized, call(invalid).Code)
}

func TestBrowserSessionSettlementRouteRequiresItsExactAgentManagerRole(t *testing.T) {
	call := func(principal apiidentity.Principal) *httptest.ResponseRecorder {
		server := &Server{servicePrincipalVerifier: staticServicePrincipalVerifier{principal: principal}}
		handler := server.requireAgentManagerBrowserSessionSettlement(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/browser-sessions/settle-and-receipt", nil)
		request.Header.Set("Authorization", "Bearer synthetic-token")
		response := httptest.NewRecorder()
		handler(response, request)
		return response
	}
	role := lpbsidentity.AgentManagerBrowserSessionSettlement
	valid := apiidentity.Principal{Kind: apiidentity.ActorService, Subject: role.Principal("agent-manager"), Verified: true, Audience: role.Audience, Scopes: []string{role.Scope}}
	require.Equal(t, http.StatusNoContent, call(valid).Code)

	wrongScope := valid
	wrongScope.Scopes = []string{"lpbs:usage.read"}
	require.Equal(t, http.StatusUnauthorized, call(wrongScope).Code)
	wrongAudience := valid
	wrongAudience.Audience = "scenario-authenticator:other-resource"
	require.Equal(t, http.StatusUnauthorized, call(wrongAudience).Code)
	wrongPrincipal := valid
	wrongPrincipal.Subject = "agent-manager"
	require.Equal(t, http.StatusUnauthorized, call(wrongPrincipal).Code)
	notVerified := valid
	notVerified.Verified = false
	require.Equal(t, http.StatusUnauthorized, call(notVerified).Code)
}

func TestDeploymentAndAnalyticsRoutesRequireTheirOwnServiceRole(t *testing.T) {
	request := func(verifier authn.TokenVerifier, wrap func(*Server, http.HandlerFunc) http.HandlerFunc) int {
		server := &Server{servicePrincipalVerifier: verifier}
		handler := wrap(server, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer synthetic-token")
		resp := httptest.NewRecorder()
		handler(resp, req)
		return resp.Code
	}
	principalFor := func(role principalrole.Role, source string) apiidentity.Principal {
		return apiidentity.Principal{Kind: apiidentity.ActorService, Subject: role.Principal(source), Verified: true, Audience: role.Audience, Scopes: []string{role.Scope}}
	}
	deployment := func(s *Server, next http.HandlerFunc) http.HandlerFunc { return s.requireDeploymentPrincipal(next) }
	metrics := func(s *Server, next http.HandlerFunc) http.HandlerFunc { return s.requireMetricsReader(next) }
	recovery := func(s *Server, next http.HandlerFunc) http.HandlerFunc { return s.requireAdminRecovery(next) }
	profileReconcile := func(s *Server, next http.HandlerFunc) http.HandlerFunc { return s.requireRemoteProfileReconcile(next) }
	profileRead := func(s *Server, next http.HandlerFunc) http.HandlerFunc { return s.requireRemoteProfileReader(next) }

	for _, entry := range []struct {
		source string
		role   principalrole.Role
	}{
		{source: "scenario-to-desktop", role: lpbsidentity.ScenarioToDesktopDeployment},
		{source: "deployment-manager", role: lpbsidentity.DeploymentManagerRelease},
	} {
		principal := principalFor(entry.role, entry.source)
		require.Equal(t, http.StatusNoContent, request(staticServicePrincipalVerifier{principal: principal}, deployment))
	}
	wrongDeployment := principalFor(lpbsidentity.CommandCenterAnalytics, "command-center")
	require.Equal(t, http.StatusUnauthorized, request(staticServicePrincipalVerifier{principal: wrongDeployment}, deployment))

	analytics := principalFor(lpbsidentity.CommandCenterAnalytics, "command-center")
	require.Equal(t, http.StatusNoContent, request(staticServicePrincipalVerifier{principal: analytics}, metrics))
	wrongAnalytics := principalFor(lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop")
	require.Equal(t, http.StatusUnauthorized, request(staticServicePrincipalVerifier{principal: wrongAnalytics}, metrics))

	localRecovery := principalFor(lpbsidentity.LocalAdminRecovery, "landing-page-business-suite")
	require.Equal(t, http.StatusNoContent, request(staticServicePrincipalVerifier{principal: localRecovery}, recovery))
	wrongRecovery := principalFor(lpbsidentity.TreasuryAcceptedReceiptCredit, "treasury")
	require.Equal(t, http.StatusUnauthorized, request(staticServicePrincipalVerifier{principal: wrongRecovery}, recovery))

	profileManager := principalFor(lpbsidentity.ScenarioToCloudRemoteProfiles, "scenario-to-cloud")
	require.Equal(t, http.StatusNoContent, request(staticServicePrincipalVerifier{principal: profileManager}, profileReconcile))
	wrongProfileManager := principalFor(lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop")
	require.Equal(t, http.StatusUnauthorized, request(staticServicePrincipalVerifier{principal: wrongProfileManager}, profileReconcile))
	profileProxy := principalFor(lpbsidentity.CommandCenterRemoteProfiles, "command-center")
	require.Equal(t, http.StatusNoContent, request(staticServicePrincipalVerifier{principal: profileProxy}, profileRead))
	wrongProfileProxy := principalFor(lpbsidentity.CommandCenterAnalytics, "command-center")
	require.Equal(t, http.StatusUnauthorized, request(staticServicePrincipalVerifier{principal: wrongProfileProxy}, profileRead))
	desktopCatalog := func(s *Server, next http.HandlerFunc) http.HandlerFunc {
		return s.requireScenarioToDesktopDeployment(next)
	}
	require.Equal(t, http.StatusNoContent, request(staticServicePrincipalVerifier{principal: principalFor(lpbsidentity.ScenarioToDesktopDeployment, "scenario-to-desktop")}, desktopCatalog))
	require.Equal(t, http.StatusUnauthorized, request(staticServicePrincipalVerifier{principal: principalFor(lpbsidentity.DeploymentManagerRelease, "deployment-manager")}, desktopCatalog))
}

func TestAcceptedReceiptDatabaseContextRoutesTestModeAndRejectsOtherValues(t *testing.T) {
	plain := httptest.NewRequest("POST", "/api/v1/internal/crypto-funding/accepted-receipts", nil)
	ctx, err := acceptedReceiptDatabaseContext(plain)
	require.NoError(t, err)
	require.False(t, database.IsTestMode(ctx))

	testMode := httptest.NewRequest("POST", "/api/v1/internal/crypto-funding/accepted-receipts", nil)
	testMode.Header.Set("X-Vrooli-Test-Mode", "1")
	ctx, err = acceptedReceiptDatabaseContext(testMode)
	require.NoError(t, err)
	require.True(t, database.IsTestMode(ctx))

	invalid := httptest.NewRequest("POST", "/api/v1/internal/crypto-funding/accepted-receipts", nil)
	invalid.Header.Set("X-Vrooli-Test-Mode", "true")
	_, err = acceptedReceiptDatabaseContext(invalid)
	require.Error(t, err)
}
