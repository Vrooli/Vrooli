package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	adminhandler "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/nativegrants"
)

func TestNativeGrantNoSessionBeforeExchangeAndReplayRevokesSession(t *testing.T) {
	db := setupTestDB(t)
	email := "native-grant-flow@example.com"
	defer cleanupUserTestData(t, db, email)
	auth := newUserAuthServiceForTest(db, NewEmailService())
	var signInToken string
	auth.UseTokenCallback(func(_, token, _ string) { signInToken = token })
	if _, err := auth.RequestSignIn(context.Background(), administration.SignInRequest{Email: email, BrowserBinding: "native-browser-binding"}); err != nil {
		// Development test delivery may be intentionally unavailable; the
		// pending credential is still created before delivery is attempted.
		t.Logf("delivery unavailable in test composition: %v", err)
	}
	if signInToken == "" {
		t.Fatal("sign-in token was not captured")
	}

	verifier := "native-grant-verifier-with-sufficient-entropy"
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	redirectURI := "http://127.0.0.1:43111/callback"
	deps := userAuthHandlerDependencies(auth, nil)
	repo := nativegrants.NewRepository(db)
	requestBody, _ := json.Marshal(map[string]string{"token": signInToken, "code_challenge": challenge, "code_challenge_method": "S256", "redirect_uri": redirectURI})
	authorize := httptest.NewRecorder()
	adminhandler.AuthorizeWithPKCE(deps, repo).ServeHTTP(authorize, httptest.NewRequest(http.MethodPost, "/api/v1/auth/authorize", bytes.NewReader(requestBody)))
	if authorize.Code != http.StatusOK {
		t.Fatalf("authorize status=%d body=%s", authorize.Code, authorize.Body.String())
	}
	var redirect struct {
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.Unmarshal(authorize.Body.Bytes(), &redirect); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(redirect.RedirectURL)
	if err != nil {
		t.Fatal(err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatal("authorization response omitted code")
	}
	var sessionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_sessions WHERE user_id = (SELECT id FROM users WHERE email = $1)`, email).Scan(&sessionCount); err != nil {
		t.Fatal(err)
	}
	if sessionCount != 0 {
		t.Fatalf("authorization created %d session(s) before exchange", sessionCount)
	}

	exchangeHandler := adminhandler.ExchangeAuthorizationCode(deps, repo)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/token" {
			exchangeHandler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	defer httpServer.Close()
	access, err := credentialclient.ExchangeAuthorizationCode(context.Background(), httpServer.Client(), httpServer.URL, "desktop", redirectURI, code, credentialclient.PKCEChallenge{Verifier: verifier, Challenge: challenge})
	if err != nil || access.AccessToken == "" || access.ExpiresAt.IsZero() {
		t.Fatalf("credentialclient exchange access=%+v err=%v", access, err)
	}
	var sessionID string
	if err := db.QueryRow(`SELECT id FROM user_sessions WHERE user_id = (SELECT id FROM users WHERE email = $1) ORDER BY created_at DESC LIMIT 1`, email).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}

	replay := httptest.NewRecorder()
	exchangeBody, _ := json.Marshal(map[string]string{"code": code, "code_verifier": verifier, "redirect_uri": redirectURI})
	exchangeHandler.ServeHTTP(replay, httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(exchangeBody)))
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("replay status=%d body=%s", replay.Code, replay.Body.String())
	}
	var revoked bool
	if err := db.QueryRow(`SELECT revoked FROM user_sessions WHERE id = $1`, sessionID).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("replayed native grant did not revoke its session")
	}
}
