package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/businessaccount"
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
	plansPath := filepath.Join(t.TempDir(), "plans.json")
	fixturePlans := `{"bundle":{"bundle_key":"business_suite","name":"Journey","stripe_product_id":"prod_journey_fixture","credits_per_usd":1000000,"display_credits_multiplier":0.001,"display_credits_label":"credits","environment":"test"},"plans":[{"stripe_price_id":"price_journey_app","plan_name":"Journey App","plan_tier":"free","billing_interval":"month","amount_cents":0,"currency":"usd","plan_rank":1,"kind":"subscription","metadata":{"app_entitlements":{"audio-tools":{"features":["audio:summarize"],"limits":{"ai_credits":10000}},"web-console":{"features":["coding-backup:paid-inference"],"limits":{"ai_credits":10000}}}}}]}`
	if err := os.WriteFile(plansPath, []byte(fixturePlans), 0o600); err != nil {
		t.Fatal(err)
	}
	planStore := commerce.NewPlanStoreWithOptions(commerce.PlanStoreOptions{PlansPath: plansPath, BundleKey: "business_suite", DisplayEnv: "test"})
	if err := planStore.LoadAll(); err != nil {
		t.Fatal(err)
	}
	planService := commerce.NewPlanServiceWithOptions(commerce.PlanServiceOptions{PlanStore: planStore, DefaultBundle: "business_suite", DisplayEnv: "test"})
	if _, err := db.Exec(`INSERT INTO subscriptions(subscription_id,customer_id,customer_email,business_account_id,status,plan_tier,price_id,bundle_key) VALUES('journey-app','consumer@example.invalid','consumer@example.invalid','account','active','free','price_journey_app','business_suite')`); err != nil {
		t.Fatal(err)
	}
	pair, err := auth.CreateSession(context.Background(), &administration.User{ID: actor, Email: "consumer@example.invalid"}, "127.0.0.1", "isolated-journey")
	if err != nil {
		t.Fatal(err)
	}
	live := false
	door := &aidoor.Door{Keys: aidoor.KeyStore{DB: db}, Wallet: commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: db, Limits: commerce.NewLimitsService(db, "postgres", nil), Dialect: "postgres", LiveMode: &live}), GatewayURL: os.Getenv("VROOLI_JOURNEY_CONSUMER_GATEWAY"), SignCreditAssertion: func(aidoor.Key, string, string) (string, error) { return "fixture-credit-assertion", nil }}
	s := &Server{router: mux.NewRouter(), db: db, userAuthService: auth, businessAccounts: businessaccount.NewSQLRepository(db), planService: planService, accountService: commerce.NewService(db, planService, commerce.Runtime{})}
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
