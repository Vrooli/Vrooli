package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	businesshttp "landing-page-business-suite-api/handlers/businessaccount"
	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	pricinghttp "landing-page-business-suite-api/handlers/pricing"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/analytics"
	"landing-page-business-suite-api/internal/businessaccount"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/experimentation"
)

// Only the Stripe card screen/provider and upstream router are doubles. Browser
// identity, checkout construction, signed webhook handling, wallet and AI routes
// use their production owners against the parent's isolated PostgreSQL database.
func TestJourneyBrowserServer(t *testing.T) {
	dsn := os.Getenv("VROOLI_J15_DB")
	if dsn == "" {
		t.Skip("browser journey helper")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, schema := range []string{administration.Schema(), analytics.Schema(), analytics.IndexesSchema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	source, err := os.ReadFile("../.vrooli/plans.json")
	if err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(t.TempDir(), "plans.json")
	if err := os.WriteFile(catalogPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	store := commerce.NewPlanStoreWithOptions(commerce.PlanStoreOptions{PlansPath: catalogPath, BundleKey: "business_suite", DisplayEnv: "test"})
	if err := store.LoadAll(); err != nil {
		t.Fatal(err)
	}
	plans := commerce.NewPlanServiceWithOptions(commerce.PlanServiceOptions{PlanStore: store, DefaultBundle: "business_suite", DisplayEnv: "test"})
	var topup *commerce.PlanOption
	for _, plan := range store.GetPlans() {
		if plan.GetKind().String() == "PLAN_KIND_CREDITS_TOPUP" && plan.GetDisplayEnabled() {
			topup = plan
			break
		}
	}
	if topup == nil {
		t.Fatal("configured test top-up is required")
	}
	s := &Server{router: mux.NewRouter(), db: db, businessAccounts: businessaccount.NewSQLRepository(db), planService: plans}
	server := httptest.NewServer(s.router)
	defer server.Close()
	auth := administration.NewUserAuthService(administration.UserAuthServiceOptions{Store: db, JWTIssuer: "j15-isolated", BaseURL: server.URL + "/auth/verify", AppName: "Vrooli", OutboxEnabled: true, SenderIdentity: "signin@example.invalid"})
	s.userAuthService = auth
	s.magicLinkLimiter = NewRateLimiter(5, 15*time.Minute)
	s.authThrottle = administration.NewAuthThrottle(db)
	s.accountService = commerce.NewService(db, plans, commerce.Runtime{})
	live := false
	limits := commerce.NewLimitsService(db, "postgres", nil)
	s.usageService = commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{DB: db, LimitsService: limits, Dialect: "postgres", LiveMode: &live})
	done := make(chan struct{})
	var once sync.Once
	s.router.HandleFunc("/__j15/stop", func(w http.ResponseWriter, r *http.Request) { once.Do(func() { close(done) }); w.WriteHeader(204) })
	s.router.HandleFunc("/__j15/mail", func(w http.ResponseWriter, r *http.Request) {
		var body string
		if err := db.QueryRowContext(r.Context(), `SELECT text_body FROM email_outbox WHERE recipient='buyer@example.invalid' ORDER BY requested_at DESC LIMIT 1`).Scan(&body); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		code := regexp.MustCompile(`\b[0-9]{6}\b`).FindString(body)
		json.NewEncoder(w).Encode(map[string]string{"code": code})
	})
	var stripe *httptest.Server
	var checkoutMu sync.Mutex
	var checkout url.Values
	var checkoutAmount int64
	stripe = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/prices":
			json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "price_j15_mock"}}})
		case r.URL.Path == "/v1/checkout/sessions":
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if r.Form.Get("metadata[business_account_id]") == "" {
				http.Error(w, "missing account attribution", 400)
				return
			}
			selected, err := plans.GetPlanByPriceID(r.Form.Get("line_items[0][price]"))
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			checkoutMu.Lock()
			checkout = r.Form
			checkoutAmount = selected.GetAmountCents()
			checkoutMu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"id": "cs_j15_mock", "url": stripe.URL + "/card", "amount_total": selected.GetAmountCents(), "customer_email": r.Form.Get("customer_email")})
		case r.URL.Path == "/card" && r.Method == "GET":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<!doctype html><title>Mock Stripe · test only</title><h1>Mock Stripe · test only</h1><form method="post" action="/pay"><label>Test card number<input name="card" required pattern="4242 4242 4242 4242"></label><button>Pay with test card</button></form>`)
		case r.URL.Path == "/pay" && r.Method == "POST":
			_ = r.ParseForm()
			if r.Form.Get("card") != "4242 4242 4242 4242" {
				http.Error(w, "test card required", 400)
				return
			}
			checkoutMu.Lock()
			values := checkout
			amount := checkoutAmount
			checkoutMu.Unlock()
			body, _ := json.Marshal(map[string]any{"id": "evt_j15_paid", "type": "checkout.session.completed", "livemode": false, "data": map[string]any{"object": map[string]any{"id": "cs_j15_mock", "customer_email": values.Get("customer_email"), "payment_intent": "pi_j15_mock", "payment_status": "paid", "amount_total": amount}}})
			timestamp := fmt.Sprint(time.Now().Unix())
			mac := hmac.New(sha256.New, []byte("whsec_j15_mock"))
			mac.Write([]byte(timestamp + "."))
			mac.Write(body)
			req, _ := http.NewRequest("POST", server.URL+"/api/v1/webhooks/stripe", strings.NewReader(string(body)))
			req.Header.Set("Stripe-Signature", "t="+timestamp+",v1="+hex.EncodeToString(mac.Sum(nil)))
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			defer response.Body.Close()
			if response.StatusCode != 200 {
				data, _ := io.ReadAll(response.Body)
				http.Error(w, string(data), 500)
				return
			}
			http.Redirect(w, r, values.Get("success_url"), http.StatusSeeOther)
		case strings.HasPrefix(r.URL.Path, "/v1/payment_intents/"):
			json.NewEncoder(w).Encode(map[string]any{"id": "pi_j15_mock", "latest_charge": "ch_j15_mock"})
		case strings.HasPrefix(r.URL.Path, "/v1/charges/"):
			json.NewEncoder(w).Encode(map[string]any{"id": "ch_j15_mock", "payment_intent": "pi_j15_mock", "amount": topup.GetAmountCents(), "currency": "usd", "livemode": false})
		default:
			http.NotFound(w, r)
		}
	}))
	defer stripe.Close()
	payment := commerce.NewPaymentSettingsServiceWithCredentials(db, func(context.Context, string) (string, error) { return "", nil }, nil)
	s.stripeService = NewStripeServiceWithSettings(db, plans, payment)
	s.stripeService.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_j15_mock", publishableKey: "pk_test_j15_mock", webhookSecret: "whsec_j15_mock", hasSecret: true, hasPublishable: true, hasWebhook: true, apiBase: stripe.URL}, nil
	})
	if err := s.stripeService.RefreshConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	registerAuthRoutes(s)
	businesshttp.RegisterRoutes(s.router, businesshttp.Dependencies{Repository: s.businessAccounts, UserID: getUserID, UserEmail: getUserEmail}, s.requireUserAuth)
	registerBillingRoutes(s)
	registerCreditsRoutes(s)
	brandingRoot := t.TempDir()
	variantsDir := filepath.Join(brandingRoot, "variants")
	if err := os.MkdirAll(variantsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	branding, err := os.ReadFile("../config/branding.json")
	if err != nil {
		t.Fatal(err)
	}
	var publicConfig map[string]any
	if err := json.Unmarshal(branding, &publicConfig); err != nil {
		t.Fatal(err)
	}
	delete(publicConfig, "smtp_password")
	branding, _ = json.Marshal(publicConfig)
	brandingPath := filepath.Join(brandingRoot, "branding.json")
	if err := os.WriteFile(brandingPath, branding, 0o600); err != nil {
		t.Fatal(err)
	}
	s.configStore = experimentation.NewConfigStoreWithOptions(experimentation.ConfigStoreOptions{VariantsDir: variantsDir, BrandingPath: brandingPath})
	if err := s.configStore.LoadAll(); err != nil {
		t.Fatal(err)
	}
	varianthttp.RegisterBrandingConnectRoutes(s.router, s.configStore, s.requireAdminStepUp)
	pricinghttp.RegisterRoutes(s.router, plans.GetPricingOverview)
	registerAIDoorRoutes(s, &aidoor.Door{Keys: aidoor.KeyStore{DB: db}, Wallet: commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: db, Limits: limits, Dialect: "postgres", LiveMode: &live}), GatewayURL: os.Getenv("VROOLI_J15_GATEWAY")})
	dist, err := filepath.Abs("../ui/dist")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
		t.Fatal("build LPBS UI before browser journey: ", err)
	}
	files := http.FileServer(http.Dir(dist))
	s.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/landing_page_business_suite.") {
			http.NotFound(w, r)
			return
		}
		if _, err := os.Stat(filepath.Join(dist, filepath.Clean(r.URL.Path))); err != nil {
			// Match the production UI server's root base for Vite's relative assets.
			data, err := os.ReadFile(filepath.Join(dist, "index.html"))
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, strings.Replace(string(data), "<head>", `<head><base href="/">`, 1))
			return
		}
		files.ServeHTTP(w, r)
	})
	fmt.Println("BROWSER_URL=" + server.URL)
	<-done
}
