package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/handlers/intelligence"
	"landing-page-business-suite-api/internal/commerce"
	intelligencecore "landing-page-business-suite-api/internal/intelligence"
	_ "modernc.org/sqlite"
)

type paidCodingFixtureLimits struct{}

func (paidCodingFixtureLimits) GetLimit(_ context.Context, tier, key string, _ *string) (*commerce.TierLimit, error) {
	return &commerce.TierLimit{TierID: tier, LimitKey: key, LimitType: "cost_based", LimitValue: 100_000_000}, nil
}

func (paidCodingFixtureLimits) GetTierLimits(_ context.Context, tier string) ([]commerce.TierLimit, error) {
	return []commerce.TierLimit{{TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: 100_000_000, CostMultiplier: 1_000_000}}, nil
}

type paidCodingFixtureAccount struct{}

func (paidCodingFixtureAccount) GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error) {
	tier := "pro"
	return &shared.SubscriptionStatus{PlanTier: &tier}, nil
}

func (paidCodingFixtureAccount) GetSubscriptionForBusinessAccountContext(context.Context, string, string) (*shared.SubscriptionStatus, error) {
	return paidCodingFixtureAccount{}.GetSubscriptionContext(context.Background(), "")
}

type paidCodingFixtureProvider struct {
	usage              *intelligencecore.OpenRouterUsage
	text               string
	err                error
	toolRequestCounter *atomic.Int32
}

func (p paidCodingFixtureProvider) Chat(context.Context, intelligencecore.OpenRouterChatRequest) (*intelligencecore.OpenRouterChatResponse, error) {
	return nil, errors.New("non-streaming request was not expected")
}

func (p paidCodingFixtureProvider) ChatStream(_ context.Context, _ intelligencecore.OpenRouterChatRequest, emit func(string)) (*intelligencecore.OpenRouterUsage, error) {
	if emit != nil {
		content := p.text
		if content == "" {
			content = "partial coding output"
		}
		emit(content)
	}
	return p.usage, p.err
}

func (p paidCodingFixtureProvider) ChatStreamEvents(_ context.Context, req intelligencecore.OpenRouterChatRequest, emit func(intelligencecore.OpenRouterStreamEvent)) (*intelligencecore.OpenRouterUsage, error) {
	if len(req.Tools) > 0 && p.toolRequestCounter != nil {
		p.toolRequestCounter.Add(1)
	}
	if emit != nil {
		content := p.text
		if content == "" {
			content = "partial coding output"
		}
		emit(intelligencecore.OpenRouterStreamEvent{Content: content})
	}
	return p.usage, p.err
}

func (paidCodingFixtureProvider) VerifyAPIKey(context.Context) error { return nil }

type paidCodingFixtureLimiter struct{}

func (paidCodingFixtureLimiter) Allow(string) bool { return true }

func TestPaidCodingHTTPHandlerSettlesMeasuredPartialUsageToBusinessAccount(t *testing.T) {
	db := newPaidCodingFixtureDB(t)
	usage := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		DB: db, LimitsService: paidCodingFixtureLimits{}, Dialect: "sqlite",
		Log: func(string, map[string]interface{}) {},
	})
	providerErr := errors.New("upstream stream interrupted")
	provider := paidCodingFixtureProvider{
		usage: &intelligencecore.OpenRouterUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, CostMicros: 1234, UsageReported: true},
		err:   providerErr,
	}
	service := intelligencecore.NewMeteredInferenceService(intelligencecore.MeteredInferenceServiceOptions{
		UsageService: newCommerceUsageServicer(usage), AccountService: paidCodingFixtureAccount{},
		OpenRouterClient: provider, Logger: func(string, map[string]interface{}) {},
	})
	deps := intelligence.Dependencies{
		Service:         service,
		UserRateLimiter: paidCodingFixtureLimiter{},
		IPRateLimiter:   paidCodingFixtureLimiter{},
		IPKeyFunc:       func(*http.Request) string { return "127.0.0.1" },
		UserID:          func(context.Context) string { return "user-fixture" },
		UserIdentity:    func(context.Context) string { return "coding-user@example.test" },
		ResolveBusinessAccount: func(_ context.Context, userID, identity, requestedID string) (string, error) {
			if userID != "user-fixture" || identity != "coding-user@example.test" || requestedID != "acct-selected" {
				return "", errors.New("business account membership check failed")
			}
			return requestedID, nil
		},
		WriteJSONError: writeJSONError,
		Log:            func(string, map[string]interface{}) {},
		LogError:       func(string, map[string]interface{}) {},
	}
	handler := intelligence.New(deps)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/ai/inference-stream", handler.InferenceStream()).Methods(http.MethodPost)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	body, err := json.Marshal(map[string]any{
		"role": "code.default", "business_account_id": "acct-selected",
		"messages": []map[string]string{{"role": "user", "content": "inspect this repository"}},
		"metadata": map[string]string{"app_bundle_key": "web-console", "operation": "interactive_coding"},
	})
	if err != nil {
		t.Fatalf("encode inference request: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/ai/inference-stream", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("build inference request: %v", err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("call LPBS inference handler: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("LPBS status=%d", response.StatusCode)
	}

	var accountID, actor, appBundle, model, providerName string
	var amount, costMicros int64
	var promptTokens, completionTokens int
	if err := db.QueryRow(`SELECT business_account_id,actor_identity,usage_amount FROM business_account_usage_records WHERE billing_period=?`, time.Now().UTC().Format("2006-01")).Scan(&accountID, &actor, &amount); err != nil {
		t.Fatalf("read settled account usage: %v", err)
	}
	if accountID != "acct-selected" || actor != "coding-user@example.test" {
		t.Fatalf("settlement attribution = account %q actor %q", accountID, actor)
	}
	if err := db.QueryRow(`SELECT app_bundle_key,model,cost_micros,prompt_tokens,completion_tokens,provider FROM business_account_usage_events`).Scan(&appBundle, &model, &costMicros, &promptTokens, &completionTokens, &providerName); err != nil {
		t.Fatalf("read settled usage event: %v", err)
	}
	wantAmount := intelligencecore.CalculateCost(intelligencecore.DefaultModelPricing(), "openai/gpt-4o-mini", 100, 20)
	if amount != wantAmount || appBundle != "web-console" || costMicros != 1234 || promptTokens != 100 || completionTokens != 20 || providerName != "openrouter" {
		t.Fatalf("measured settlement = amount %d app %q cost %d tokens %d/%d provider %q", amount, appBundle, costMicros, promptTokens, completionTokens, providerName)
	}
	if model == "" {
		t.Fatal("usage event omitted the policy-resolved model")
	}
	var pending int
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE status='pending'`).Scan(&pending); err != nil {
		t.Fatalf("read pending reservations: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending reservations=%d, want measured partial usage to settle", pending)
	}
}

// TestPaidCodingHTTPHandlerJoinsWebConsoleProxyAndSettlesToBusinessAccount
// hosts the real LPBS intelligence handler and SQLite usage ledger, then runs
// the Web Console owner test as an HTTP client against that live fixture. It is
// opt-in because the nested OpenCode run is an integration check, not a unit
// dependency of LPBS's ordinary suite.
func TestPaidCodingHTTPHandlerJoinsWebConsoleProxyAndSettlesToBusinessAccount(t *testing.T) {
	if os.Getenv("RUN_PAID_CODING_CROSS_OWNER_TEST") != "1" {
		t.Skip("set RUN_PAID_CODING_CROSS_OWNER_TEST=1 to run the isolated cross-owner paid-coding journey")
	}
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("OpenCode CLI is not installed")
	}

	db := newPaidCodingFixtureDB(t)
	usage := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		DB: db, LimitsService: paidCodingFixtureLimits{}, Dialect: "sqlite",
		Log: func(string, map[string]interface{}) {},
	})
	toolRequestCounter := &atomic.Int32{}
	provider := paidCodingFixtureProvider{
		usage:              &intelligencecore.OpenRouterUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, CostMicros: 1234, UsageReported: true},
		text:               "external-lpbs-fixture-ok",
		toolRequestCounter: toolRequestCounter,
	}
	service := intelligencecore.NewMeteredInferenceService(intelligencecore.MeteredInferenceServiceOptions{
		UsageService: newCommerceUsageServicer(usage), AccountService: paidCodingFixtureAccount{},
		OpenRouterClient: provider, Logger: func(string, map[string]interface{}) {},
	})
	deps := intelligence.Dependencies{
		Service:         service,
		UserRateLimiter: paidCodingFixtureLimiter{},
		IPRateLimiter:   paidCodingFixtureLimiter{},
		IPKeyFunc:       func(*http.Request) string { return "127.0.0.1" },
		UserID:          func(context.Context) string { return "user-fixture" },
		UserIdentity:    func(context.Context) string { return "coding-user@example.test" },
		ResolveBusinessAccount: func(_ context.Context, userID, identity, requestedID string) (string, error) {
			if userID != "user-fixture" || identity != "coding-user@example.test" || requestedID != "acct-selected" {
				return "", errors.New("business account membership check failed")
			}
			return requestedID, nil
		},
		WriteJSONError: writeJSONError,
		Log:            func(string, map[string]interface{}) {},
		LogError:       func(string, map[string]interface{}) {},
	}
	handler := intelligence.New(deps)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/auth/refresh", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fixture-lpbs-access","refresh_token":"fixture-lpbs-refresh","expires_at":"2030-01-01T00:00:00Z"}`))
	}).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/ai/inference-stream", handler.InferenceStream()).Methods(http.MethodPost)
	lpbs := httptest.NewServer(router)
	defer lpbs.Close()

	webConsoleAPIDir, err := filepath.Abs(filepath.Join("..", "..", "..", "scenarios", "web-console", "api"))
	if err != nil {
		t.Fatalf("resolve Web Console API module: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "./", "-run", "^TestOpenCodeCLIUsesAccountBoundProxyAgainstExternalLPBSFixture$", "-count=1", "-v")
	command.Dir = webConsoleAPIDir
	command.Env = envWithOverrides(os.Environ(), map[string]string{"WEB_CONSOLE_EXTERNAL_LPBS_URL": lpbs.URL})
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Web Console OpenCode journey against LPBS fixture failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "TestOpenCodeCLIUsesAccountBoundProxyAgainstExternalLPBSFixture") || !strings.Contains(string(output), "PASS") {
		t.Fatalf("nested Web Console test did not report a passing test:\n%s", output)
	}
	if toolRequestCounter.Load() == 0 {
		t.Fatal("OpenCode journey never exercised LPBS native tool-capable stream protocol")
	}

	var accountID, actor, appBundle, model, providerName string
	var amount, costMicros int64
	var promptTokens, completionTokens, usageEventCount int
	if err := db.QueryRow(`SELECT business_account_id,actor_identity,usage_amount FROM business_account_usage_records WHERE billing_period=?`, time.Now().UTC().Format("2006-01")).Scan(&accountID, &actor, &amount); err != nil {
		t.Fatalf("read joined journey account usage: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*),MIN(app_bundle_key),MIN(model),SUM(cost_micros),SUM(prompt_tokens),SUM(completion_tokens),MIN(provider) FROM business_account_usage_events`).Scan(&usageEventCount, &appBundle, &model, &costMicros, &promptTokens, &completionTokens, &providerName); err != nil {
		t.Fatalf("read joined journey usage event: %v", err)
	}
	wantAmount := intelligencecore.CalculateCost(intelligencecore.DefaultModelPricing(), "openai/gpt-4o-mini", 100, 20)
	if accountID != "acct-selected" || actor != "coding-user@example.test" || usageEventCount == 0 || amount != wantAmount*int64(usageEventCount) || appBundle != "web-console" || costMicros != 1234*int64(usageEventCount) || promptTokens != 100*usageEventCount || completionTokens != 20*usageEventCount || providerName != "openrouter" || model == "" {
		t.Fatalf("joined account settlement = account %q actor %q amount %d across %d events app %q model %q cost %d tokens %d/%d provider %q", accountID, actor, amount, usageEventCount, appBundle, model, costMicros, promptTokens, completionTokens, providerName)
	}
	var pending int
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE status='pending'`).Scan(&pending); err != nil {
		t.Fatalf("read joined journey pending reservations: %v", err)
	}
	if pending != 0 {
		t.Fatalf("joined journey left %d pending account reservations", pending)
	}
}

func envWithOverrides(environment []string, overrides map[string]string) []string {
	result := make([]string, 0, len(environment)+len(overrides))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if _, replaced := overrides[name]; !replaced {
			result = append(result, entry)
		}
	}
	for name, value := range overrides {
		result = append(result, name+"="+value)
	}
	return result
}

func newPaidCodingFixtureDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open fixture database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE business_account_usage_records (business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, usage_amount INTEGER NOT NULL DEFAULT 0, app_bundle_key TEXT NOT NULL, last_operation_at DATETIME, created_at DATETIME, updated_at DATETIME, UNIQUE(business_account_id,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, reserved_amount INTEGER NOT NULL, actual_amount INTEGER, status TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME, settled_at DATETIME, app_bundle_key TEXT NOT NULL DEFAULT 'unattributed', model TEXT NOT NULL DEFAULT 'unknown', cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, provider TEXT NOT NULL DEFAULT 'openrouter')`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, app_bundle_key TEXT, model TEXT, credits INTEGER NOT NULL, cost_micros INTEGER, prompt_tokens INTEGER, completion_tokens INTEGER, provider TEXT, created_at DATETIME)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create fixture schema: %v", err)
		}
	}
	return db
}
