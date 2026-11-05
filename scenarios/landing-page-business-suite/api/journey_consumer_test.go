package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/commerce"
)

// This helper runs production consumer authentication, session validation and
// device issuance against the same isolated PostgreSQL wallet as the door.
func TestJourneyConsumerServer(t *testing.T) {
	dsn := os.Getenv("VROOLI_JOURNEY_CONSUMER_DB")
	if dsn == "" {
		t.Skip("consumer integration helper")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth := administration.NewUserAuthService(administration.UserAuthServiceOptions{Store: db, JWTIssuer: "journey", AccessTTL: time.Hour})
	const actor = "b3ec75ee-3e58-4d8d-a6df-b83889ee7f42"
	if _, err = db.Exec(`INSERT INTO users(id,email,email_verified) VALUES($1,'consumer@example.invalid',TRUE) ON CONFLICT DO NOTHING`, actor); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO business_account_members(business_account_id,user_id,role) VALUES('account',$1,'owner') ON CONFLICT DO NOTHING`, actor); err != nil {
		t.Fatal(err)
	}
	pair, err := auth.CreateSession(context.Background(), &administration.User{ID: actor, Email: "consumer@example.invalid"}, "127.0.0.1", "isolated-journey")
	if err != nil {
		t.Fatal(err)
	}
	live := false
	door := &aidoor.Door{Keys: aidoor.KeyStore{DB: db}, Wallet: commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: db, Limits: commerce.NewLimitsService(db, "postgres", nil), Dialect: "postgres", LiveMode: &live}), GatewayURL: os.Getenv("VROOLI_JOURNEY_CONSUMER_GATEWAY")}
	s := &Server{router: mux.NewRouter(), userAuthService: auth}
	registerAIDoorRoutes(s, door)
	s.router.HandleFunc("/api/v1/auth/refresh", adminhttp.RefreshTokens(adminhttp.UserAuthDependencies{Service: auth, SecureCookies: func() bool { return false }, Now: time.Now, WriteError: writeJSONError, LogError: func(string, map[string]any) {}})).Methods("POST")
	done := make(chan struct{})
	var once sync.Once
	s.router.HandleFunc("/__consumer/stop", func(w http.ResponseWriter, r *http.Request) { once.Do(func() { close(done) }); w.WriteHeader(204) })
	server := httptest.NewServer(s.router)
	defer server.Close()
	fmt.Println("CONSUMER_URL=" + server.URL)
	fmt.Println("CONSUMER_REFRESH=" + pair.RefreshToken)
	<-done
}
