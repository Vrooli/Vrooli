package desktoplink

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	link "landing-page-business-suite-api/internal/desktoplink"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

func TestRoutesIssueRedeemAndRevokeWithoutCopyingWebsiteToken(t *testing.T) {
	service := link.NewService(link.NewMemoryRepository())
	verifier := "desktop-verifier"
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "local-principal-1", Verified: true, Source: identity.SourceScenarioAuthenticator}
	router := mux.NewRouter()
	RegisterRoutes(router, Dependencies{
		Service: service, LPBSUserID: func(context.Context) string { return "lpbs-user-1" },
		ResolveBusinessAccountID: func(_ context.Context, userID, requestedID string) (string, error) {
			if userID != "lpbs-user-1" || requestedID != "account-1" {
				return "", context.Canceled
			}
			return requestedID, nil
		},
		VerifyLocalIdentity: func(_ context.Context, proof string) (identity.Principal, error) {
			if proof != "local-proof" {
				return identity.Principal{}, context.Canceled
			}
			return principal, nil
		},
		IssueLease: func(context.Context, link.Link) (link.EntitlementLease, error) {
			return link.EntitlementLease{Token: "signed-lease", ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
		},
	}, func(next http.HandlerFunc) http.HandlerFunc { return next })

	issueBody := `{"business_account_id":"account-1","installation_id":"install-1","resource":"demo","audience":"scenario:demo","scopes":["demo:read"],"code_challenge":"` + challenge(verifier) + `","code_challenge_method":"S256","redirect_uri":"http://127.0.0.1:43120/callback"}`
	issue := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/links", strings.NewReader(issueBody))
	issueResponse := httptest.NewRecorder()
	router.ServeHTTP(issueResponse, issue)
	if issueResponse.Code != http.StatusCreated {
		t.Fatalf("issue status = %d: %s", issueResponse.Code, issueResponse.Body.String())
	}
	var issued map[string]interface{}
	if err := json.Unmarshal(issueResponse.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	code, _ := issued["code"].(string)
	if issued["business_account_id"] != "account-1" {
		t.Fatalf("issued account = %#v", issued["business_account_id"])
	}
	if code == "" || strings.Contains(issueResponse.Body.String(), "access_token") {
		t.Fatalf("issue response leaked a website token: %s", issueResponse.Body.String())
	}

	redeem := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/links/redeem", strings.NewReader(`{"code":"`+code+`","code_verifier":"`+verifier+`","installation_id":"install-1","resource":"demo"}`))
	redeem.Header.Set("Authorization", "Bearer local-proof")
	redeemResponse := httptest.NewRecorder()
	router.ServeHTTP(redeemResponse, redeem)
	if redeemResponse.Code != http.StatusOK {
		t.Fatalf("redeem status = %d: %s", redeemResponse.Code, redeemResponse.Body.String())
	}
	if !strings.Contains(redeemResponse.Body.String(), "signed-lease") || !strings.Contains(redeemResponse.Body.String(), "local-principal-1") {
		t.Fatalf("redeem response = %s", redeemResponse.Body.String())
	}

	status := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/links/local?installation_id=install-1&resource=demo", nil)
	status.Header.Set("Authorization", "Bearer local-proof")
	statusResponse := httptest.NewRecorder()
	router.ServeHTTP(statusResponse, status)
	if statusResponse.Code != http.StatusOK || !strings.Contains(statusResponse.Body.String(), `"state":"active"`) {
		t.Fatalf("status response = %d: %s", statusResponse.Code, statusResponse.Body.String())
	}

	revoke := httptest.NewRequest(http.MethodDelete, "/api/v1/desktop/links/local", strings.NewReader(`{"installation_id":"install-1","resource":"demo"}`))
	revoke.Header.Set("Authorization", "Bearer local-proof")
	revokeResponse := httptest.NewRecorder()
	router.ServeHTTP(revokeResponse, revoke)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revokeResponse.Code, revokeResponse.Body.String())
	}

	revokedStatus := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/links/local?installation_id=install-1&resource=demo", nil)
	revokedStatus.Header.Set("Authorization", "Bearer local-proof")
	revokedStatusResponse := httptest.NewRecorder()
	router.ServeHTTP(revokedStatusResponse, revokedStatus)
	if revokedStatusResponse.Code != http.StatusForbidden {
		t.Fatalf("revoked status = %d: %s", revokedStatusResponse.Code, revokedStatusResponse.Body.String())
	}
}

func TestRoutesKeepBusinessAccountsAndInstallationsIsolated(t *testing.T) {
	service := link.NewService(link.NewMemoryRepository())
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "local-principal-1", Verified: true, Source: identity.SourceScenarioAuthenticator}
	router := mux.NewRouter()
	RegisterRoutes(router, Dependencies{
		Service: service, LPBSUserID: func(context.Context) string { return "lpbs-user-1" },
		ResolveBusinessAccountID: func(_ context.Context, userID, requestedID string) (string, error) {
			if userID != "lpbs-user-1" || (requestedID != "account-1" && requestedID != "account-2") {
				return "", context.Canceled
			}
			return requestedID, nil
		},
		VerifyLocalIdentity: func(_ context.Context, proof string) (identity.Principal, error) {
			if proof != "local-proof" {
				return identity.Principal{}, context.Canceled
			}
			return principal, nil
		},
		IssueLease: func(_ context.Context, connected link.Link) (link.EntitlementLease, error) {
			return link.EntitlementLease{Token: connected.BusinessAccountID + "-lease", ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
		},
	}, func(next http.HandlerFunc) http.HandlerFunc { return next })

	issue := func(accountID, installationID, verifier string) string {
		t.Helper()
		body := `{"business_account_id":"` + accountID + `","installation_id":"` + installationID + `","resource":"demo","audience":"scenario:demo","scopes":["demo:read"],"code_challenge":"` + challenge(verifier) + `","code_challenge_method":"S256","redirect_uri":"http://127.0.0.1:43120/callback"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/links", strings.NewReader(body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("issue %s/%s status = %d: %s", accountID, installationID, response.Code, response.Body.String())
		}
		var issued struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &issued); err != nil {
			t.Fatalf("decode issue %s/%s: %v", accountID, installationID, err)
		}
		return issued.Code
	}

	redeem := func(code, verifier, installationID string) *httptest.ResponseRecorder {
		t.Helper()
		body := `{"code":"` + code + `","code_verifier":"` + verifier + `","installation_id":"` + installationID + `","resource":"demo"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/links/redeem", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer local-proof")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}

	codeOne := issue("account-1", "install-1", "verifier-one")
	codeTwo := issue("account-2", "install-2", "verifier-two")
	if response := redeem(codeOne, "verifier-one", "install-2"); response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-installation redemption status = %d, want %d: %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	if response := redeem(codeOne, "verifier-one", "install-1"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "account-1-lease") {
		t.Fatalf("account-1 redemption = %d: %s", response.Code, response.Body.String())
	}
	if response := redeem(codeTwo, "verifier-two", "install-2"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "account-2-lease") {
		t.Fatalf("account-2 redemption = %d: %s", response.Code, response.Body.String())
	}

	status := func(installationID string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/links/local?installation_id="+installationID+"&resource=demo", nil)
		request.Header.Set("Authorization", "Bearer local-proof")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	if response := status("install-1"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"business_account_id":"account-1"`) {
		t.Fatalf("account-1 status = %d: %s", response.Code, response.Body.String())
	}
	if response := status("install-2"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"business_account_id":"account-2"`) {
		t.Fatalf("account-2 status = %d: %s", response.Code, response.Body.String())
	}

	revoke := httptest.NewRequest(http.MethodDelete, "/api/v1/desktop/links/local", strings.NewReader(`{"installation_id":"install-1","resource":"demo"}`))
	revoke.Header.Set("Authorization", "Bearer local-proof")
	revokeResponse := httptest.NewRecorder()
	router.ServeHTTP(revokeResponse, revoke)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("account-1 revoke status = %d: %s", revokeResponse.Code, revokeResponse.Body.String())
	}
	if response := status("install-1"); response.Code != http.StatusForbidden {
		t.Fatalf("revoked account-1 status = %d: %s", response.Code, response.Body.String())
	}
	if response := status("install-2"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"business_account_id":"account-2"`) {
		t.Fatalf("account-2 status after account-1 revoke = %d: %s", response.Code, response.Body.String())
	}
}

func challenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
