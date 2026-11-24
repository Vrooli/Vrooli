package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/health"
	"landing-page-business-suite-api/internal/administration"
)

// A provider's initial acceptance is not proof that sign-in mail reached the
// recipient. A terminal bounce must flow from the persisted delivery record
// through the real /health composition as an optional degradation.
func TestHealthEndpointDegradesAfterSignInProviderBounce(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	t.Setenv("STRIPE_MODE", "")

	db := setupTestDB(t)
	defer db.Close()
	service, _ := signInServiceWithCapture(t, db, recordingSignInSender{})
	const email = "health-bounced@example.invalid"
	if _, err := service.RequestSignIn(context.Background(), administration.SignInRequest{Email: email}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE auth_tokens SET delivery_status='rejected', provider_status='bounce', provider_reason_class='invalid_recipient', provider_status_at=NOW() WHERE email=$1`, email); err != nil {
		t.Fatal(err)
	}

	srv := &Server{db: db, userAuthService: service}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	productionHealthHandler(srv).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("optional sign-in delivery failure returned HTTP %d, want 200: %s", w.Code, w.Body.String())
	}
	var response health.Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if response.Status != "degraded" || !response.Readiness {
		t.Fatalf("health after bounce = status %q readiness %v; want degraded but ready", response.Status, response.Readiness)
	}
	database := response.Dependencies["database"]
	if !database.Connected {
		t.Fatalf("database must remain healthy after optional provider failure: %+v", database)
	}
	signIn := response.Dependencies["sign_in_email"]
	if signIn.Connected || signIn.Error == nil {
		t.Fatalf("bounced sign-in provider must be visible in health: %+v", signIn)
	}
}
