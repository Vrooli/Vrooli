package entitlement

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// createTestService creates a service instance for testing.
func createTestService(t *testing.T) *Service {
	t.Helper()
	// Lease-store tests must not write synthetic accounts to the user's global
	// application data directory.
	t.Setenv("APP_DATA_DIR", t.TempDir())

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	cfg := config.EntitlementConfig{
		RequestTimeout: 5 * time.Second,
		CacheTTL:       5 * time.Minute,
		ServiceURL:     "https://vrooli.com",
	}

	return NewService(cfg, log)
}

func signedLeaseFixture(t *testing.T, identity, tier string) (string, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const keyID = "test-lease-key"
	payload := entitlementclient.Payload{UserIdentity: identity, Status: "active", PlanTier: tier, Features: []string{FeatureAI}, Limits: []entitlementclient.Limit{{Key: "workflow_executions", Value: 10}}, Credits: map[string]any{"balance_credits": float64(41)}, NotAfter: time.Now().Add(time.Hour)}
	lease, err := entitlementclient.Sign(payload, keyID, key)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: keyID, Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	return lease, jwks
}

func TestGetEntitlementBindsAuthorityQueryAndResponseIdentity(t *testing.T) {
	lease, jwks := signedLeaseFixture(t, "alice@example.com", "pro")
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Write(jwks)
			return
		}
		if got := r.URL.Query().Get("user"); got != "alice@example.com" {
			t.Fatalf("authority query user = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer opaque-access" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lease":"` + lease + `"}`))
	}))
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	ent, err := svc.GetEntitlement(monetization.WithAccessToken(context.Background(), "opaque-access"), " Alice@Example.com ")
	if err != nil {
		t.Fatal(err)
	}
	if ent.UserIdentity != "alice@example.com" || ent.Tier != "pro" || ent.Credits != 41 {
		t.Fatalf("entitlement = %#v", ent)
	}
}

func TestReserveWorkflowExecutionUsesAuthenticatedLPBSUsageAndSignedLimit(t *testing.T) {
	lease, jwks := signedLeaseFixture(t, "alice@example.com", "pro")
	used := int64(9)
	pending := int64(0)
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/jwks.json":
			w.Write(jwks)
		case "/api/v1/entitlements":
			if r.URL.Query().Get("user") != "alice@example.com" || r.Header.Get("Authorization") != "Bearer opaque-access" {
				t.Fatalf("entitlement request identity/token mismatch: %s", r.URL.String())
			}
			w.Write([]byte(`{"lease":"` + lease + `"}`))
		case "/api/v1/usage/reservations":
			if r.Header.Get("Authorization") != "Bearer opaque-access" {
				t.Fatalf("reservation request token mismatch: %s", r.URL.String())
			}
			var body struct {
				LimitKey string `json:"limit_key"`
				Amount   int64  `json:"amount"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LimitKey != "workflow_executions" || body.Amount != 1 {
				t.Fatalf("reservation request = %+v, err=%v", body, err)
			}
			if used+pending+body.Amount > 10 {
				http.Error(w, "quota limit exceeded", http.StatusPaymentRequired)
				return
			}
			pending += body.Amount
			_, _ = w.Write([]byte(`{"reservation_id":"hold-1"}`))
		case "/api/v1/usage/reservations/hold-1/finalize":
			pending--
			used++
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/usage/reservations/hold-1/release":
			pending--
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()
	ctx := monetization.WithAccessToken(context.Background(), "opaque-access")
	hold, err := svc.ReserveWorkflowExecution(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("reserve below limit: %v", err)
	}
	if err := hold.Commit(ctx, 1); err != nil {
		t.Fatalf("commit accepted execution: %v", err)
	}
	used = 10
	if _, err := svc.ReserveWorkflowExecution(ctx, "alice@example.com"); err == nil {
		t.Fatal("execution at signed limit was admitted")
	}
}

func TestGetEntitlementRejectsAuthorityIdentityMismatchWithoutCredits(t *testing.T) {
	lease, jwks := signedLeaseFixture(t, "bob@example.com", "business")
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Write(jwks)
			return
		}
		_, _ = w.Write([]byte(`{"lease":"` + lease + `"}`))
	}))
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	_, err := svc.GetEntitlement(monetization.WithAccessToken(context.Background(), "opaque-access"), "alice@example.com")
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("mismatch error = %v", err)
	}
}

func TestGetEntitlementRejectsAuthorityResponseWithoutIdentity(t *testing.T) {
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lease":""}`))
	}))
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	_, err := svc.GetEntitlement(monetization.WithAccessToken(context.Background(), "opaque-access"), "alice@example.com")
	if err == nil || !strings.Contains(err.Error(), "malformed lease response") {
		t.Fatalf("missing identity error = %v", err)
	}
}

type testIssuerKeyStore struct{ jwks []byte }

func (s *testIssuerKeyStore) Load(context.Context, string) ([]byte, error) {
	return append([]byte(nil), s.jwks...), nil
}

func (s *testIssuerKeyStore) Save(_ context.Context, _ string, jwks []byte) error {
	s.jwks = append([]byte(nil), jwks...)
	return nil
}

func TestSignedLeaseSurvivesBASServiceReloadAndOfflineRestart(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APP_DATA_DIR", appData)
	lease, jwks := signedLeaseFixture(t, "alice@example.com", "pro")
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			_, _ = w.Write(jwks)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lease":"` + lease + `"}`))
	}))
	configCfg := config.EntitlementConfig{RequestTimeout: time.Second, ServiceURL: server.URL}
	trust := &testIssuerKeyStore{}
	first := NewService(configCfg, logrus.New())
	first.httpClient = server.Client()
	first.gate.Entitlements.HTTPClient = first.httpClient
	first.gate.Entitlements.SetIssuerKeyStore(trust)
	if _, err := first.GetEntitlement(monetization.WithAccessToken(context.Background(), "session"), "alice@example.com"); err != nil {
		t.Fatal(err)
	}
	if !first.CanUseFeatureOffline("alice@example.com", FeatureAI, 0) {
		t.Fatal("fresh signed lease did not grant its feature")
	}

	server.Close()
	restarted := NewService(configCfg, logrus.New())
	restarted.httpClient = server.Client()
	restarted.gate.Entitlements.SetIssuerKeyStore(trust)
	if !restarted.CanUseFeatureOffline("alice@example.com", FeatureAI, 0) {
		t.Fatal("persisted signed lease was not reloaded for offline use after service restart")
	}
	if restarted.CanUseFeatureOffline("alice@example.com", "unpublished-feature", 0) {
		t.Fatal("offline restart granted a feature absent from the signed lease")
	}
}

// ============================================================================
// Entitlement Type Tests
// ============================================================================

func TestEntitlement_IsActive(t *testing.T) {
	testCases := []struct {
		status   Status
		expected bool
	}{
		{Status("active"), true},
		{Status("trialing"), true},
		{Status("past_due"), false},
		{Status("canceled"), false},
		{Status("inactive"), false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.status), func(t *testing.T) {
			ent := &Entitlement{Status: tc.status}
			if ent.IsActive() != tc.expected {
				t.Errorf("expected IsActive() = %v for status %q", tc.expected, tc.status)
			}
		})
	}
}

func TestEntitlement_IsExpired(t *testing.T) {
	// Not expired
	notExpired := &Entitlement{
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if notExpired.IsExpired() {
		t.Error("expected entitlement to not be expired")
	}

	// Expired
	expired := &Entitlement{
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	if !expired.IsExpired() {
		t.Error("expected entitlement to be expired")
	}
}

func TestEntitlement_HasFeature(t *testing.T) {
	ent := &Entitlement{
		Features:        []string{"ai", "recording", "export"},
		AppEntitlements: []entitlementclient.AppEntitlement{{AppKey: "browser-automation-studio", Status: "active", Features: []string{"ai", "recording", "export"}}},
	}

	if !ent.HasFeatureForApp("browser-automation-studio", "ai") {
		t.Error("expected HasFeature('ai') to be true")
	}
	if !ent.HasFeatureForApp("browser-automation-studio", "recording") {
		t.Error("expected HasFeature('recording') to be true")
	}
	if ent.HasFeatureForApp("browser-automation-studio", "premium") {
		t.Error("expected HasFeature('premium') to be false")
	}
}

func TestFeatureConstants(t *testing.T) {
	// Verify feature constants have expected values
	if FeatureAI != "ai" {
		t.Errorf("expected FeatureAI to be 'ai', got %q", FeatureAI)
	}
	if FeatureRecording != "recording" {
		t.Errorf("expected FeatureRecording to be 'recording', got %q", FeatureRecording)
	}
	if FeatureWatermarkFree != "watermark_free" {
		t.Errorf("expected FeatureWatermarkFree to be 'watermark_free', got %q", FeatureWatermarkFree)
	}
}

// ============================================================================
// Billing Period Tests (GAP-005)
// ============================================================================

func TestEntitlement_GetBillingPeriod_CustomDay(t *testing.T) {
	ent := &Entitlement{BillingCycleStart: 15}

	tests := []struct {
		name  string
		date  time.Time
		start string
		end   string
	}{
		{"mid-period", time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC), "2026-01-15", "2026-02-14"},
		{"start of period", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "2026-01-15", "2026-02-14"},
		{"before billing day", time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC), "2025-12-15", "2026-01-14"},
		{"year boundary", time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC), "2025-12-15", "2026-01-14"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := ent.GetBillingPeriod(tc.date)
			if start.Format("2006-01-02") != tc.start {
				t.Errorf("start: expected %s, got %s", tc.start, start.Format("2006-01-02"))
			}
			if end.Format("2006-01-02") != tc.end {
				t.Errorf("end: expected %s, got %s", tc.end, end.Format("2006-01-02"))
			}
		})
	}
}

func TestEntitlement_GetBillingPeriod_CalendarMonth(t *testing.T) {
	ent := &Entitlement{BillingCycleStart: 0}
	date := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)

	start, end := ent.GetBillingPeriod(date)

	if start.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("expected start 2026-01-01, got %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("expected end 2026-01-31, got %s", end.Format("2006-01-02"))
	}
}

func TestEntitlement_GetBillingPeriod_InvalidDay(t *testing.T) {
	// Day > 28 should fall back to calendar month
	ent := &Entitlement{BillingCycleStart: 31}
	date := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)

	start, end := ent.GetBillingPeriod(date)

	if start.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("expected calendar month start 2026-01-01, got %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("expected calendar month end 2026-01-31, got %s", end.Format("2006-01-02"))
	}
}

func TestEntitlement_GetBillingMonth(t *testing.T) {
	ent := &Entitlement{BillingCycleStart: 15}
	date := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)

	if got := ent.GetBillingMonth(date); got != "2026-01-15" {
		t.Errorf("expected 2026-01-15, got %s", got)
	}
}

func TestEntitlement_GetBillingMonth_CalendarMonthFallback(t *testing.T) {
	ent := &Entitlement{BillingCycleStart: 0}
	date := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)

	// When BillingCycleStart is 0, it returns the first of the month
	if got := ent.GetBillingMonth(date); got != "2026-01-01" {
		t.Errorf("expected 2026-01-01, got %s", got)
	}
}

func TestEntitlement_GetBillingPeriod_Day1(t *testing.T) {
	// Day 1 means first of month, same as calendar month
	ent := &Entitlement{BillingCycleStart: 1}
	date := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)

	start, end := ent.GetBillingPeriod(date)

	if start.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("expected start 2026-01-01, got %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("expected end 2026-01-31, got %s", end.Format("2006-01-02"))
	}
}

func TestEntitlement_GetBillingPeriod_Day28(t *testing.T) {
	ent := &Entitlement{BillingCycleStart: 28}
	date := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	start, end := ent.GetBillingPeriod(date)

	if start.Format("2006-01-02") != "2026-01-28" {
		t.Errorf("expected start 2026-01-28, got %s", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-02-27" {
		t.Errorf("expected end 2026-02-27, got %s", end.Format("2006-01-02"))
	}
}
