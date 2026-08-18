package entitlement

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/browser-automation-studio/config"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

// createTestService creates a service instance for testing.
func createTestService(t *testing.T) *Service {
	t.Helper()

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
	payload := entitlementclient.Payload{UserIdentity: identity, Status: "active", PlanTier: tier, Credits: map[string]any{"balance_credits": float64(41)}, NotAfter: time.Now().Add(time.Hour)}
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer server.Close()
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	ent, err := svc.GetEntitlement(WithAccessToken(context.Background(), "opaque-access"), " Alice@Example.com ")
	if err != nil {
		t.Fatal(err)
	}
	if ent.UserIdentity != "alice@example.com" || ent.Tier != "pro" || ent.Credits != 41 {
		t.Fatalf("entitlement = %#v", ent)
	}
}

func TestGetEntitlementRejectsAuthorityIdentityMismatchWithoutCredits(t *testing.T) {
	lease, jwks := signedLeaseFixture(t, "bob@example.com", "business")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Write(jwks)
			return
		}
		_, _ = w.Write([]byte(`{"lease":"` + lease + `"}`))
	}))
	defer server.Close()
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	_, err := svc.GetEntitlement(WithAccessToken(context.Background(), "opaque-access"), "alice@example.com")
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("mismatch error = %v", err)
	}
}

func TestGetEntitlementRejectsAuthorityResponseWithoutIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lease":""}`))
	}))
	defer server.Close()
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	_, err := svc.GetEntitlement(WithAccessToken(context.Background(), "opaque-access"), "alice@example.com")
	if err == nil || !strings.Contains(err.Error(), "malformed lease response") {
		t.Fatalf("missing identity error = %v", err)
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
		Features: []string{"ai", "recording", "export"},
	}

	if !ent.HasFeature("ai") {
		t.Error("expected HasFeature('ai') to be true")
	}
	if !ent.HasFeature("recording") {
		t.Error("expected HasFeature('recording') to be true")
	}
	if ent.HasFeature("premium") {
		t.Error("expected HasFeature('premium') to be false")
	}
}

// ============================================================================
// Feature-Aware Method Tests (GAP-004)
// ============================================================================

func TestCanUseAIWithEntitlement_FeaturesArray_HasAI(t *testing.T) {
	svc := createTestService(t)

	// Entitlement with AI feature in features array (free tier normally can't use AI)
	ent := &Entitlement{
		Tier:     "free",
		Features: []string{FeatureAI},
	}

	// Features array is authoritative - should allow AI despite free tier
	if !svc.CanUseAIWithEntitlement(ent) {
		t.Error("expected CanUseAIWithEntitlement to return true when features array contains AI")
	}
}

func TestCanUseAIWithEntitlement_FeaturesArray_NoAI(t *testing.T) {
	svc := createTestService(t)

	// Entitlement without AI feature (pro tier normally can use AI)
	ent := &Entitlement{
		Tier:     "pro",
		Features: []string{FeatureRecording, FeatureWatermarkFree}, // No AI
	}

	// Features array is authoritative - should deny AI despite pro tier
	if svc.CanUseAIWithEntitlement(ent) {
		t.Error("expected CanUseAIWithEntitlement to return false when features array doesn't contain AI")
	}
}

func TestCanUseAIWithEntitlement_NilEntitlement(t *testing.T) {
	svc := createTestService(t)

	if svc.CanUseAIWithEntitlement(nil) {
		t.Error("expected CanUseAIWithEntitlement to return false for nil entitlement")
	}
}

func TestCanUseRecordingWithEntitlement_FeaturesArray_HasRecording(t *testing.T) {
	svc := createTestService(t)

	// Solo tier normally can't use recording, but features array grants it
	ent := &Entitlement{
		Tier:     "solo",
		Features: []string{FeatureRecording},
	}

	if !svc.CanUseRecordingWithEntitlement(ent) {
		t.Error("expected CanUseRecordingWithEntitlement to return true when features array contains recording")
	}
}

func TestCanUseRecordingWithEntitlement_FeaturesArray_NoRecording(t *testing.T) {
	svc := createTestService(t)

	// Pro tier normally can use recording, but features array denies it
	ent := &Entitlement{
		Tier:     "pro",
		Features: []string{FeatureAI}, // No recording
	}

	if svc.CanUseRecordingWithEntitlement(ent) {
		t.Error("expected CanUseRecordingWithEntitlement to return false when features array doesn't contain recording")
	}
}

func TestCanUseRecordingWithEntitlement_NilEntitlement(t *testing.T) {
	svc := createTestService(t)

	if svc.CanUseRecordingWithEntitlement(nil) {
		t.Error("expected CanUseRecordingWithEntitlement to return false for nil entitlement")
	}
}

func TestRequiresWatermarkWithEntitlement_FeaturesArray_WatermarkFree(t *testing.T) {
	svc := createTestService(t)

	// Free tier normally requires watermark, but features array grants watermark-free
	ent := &Entitlement{
		Tier:     "free",
		Features: []string{FeatureWatermarkFree},
	}

	if svc.RequiresWatermarkWithEntitlement(ent) {
		t.Error("expected RequiresWatermarkWithEntitlement to return false when features array contains watermark_free")
	}
}

func TestRequiresWatermarkWithEntitlement_FeaturesArray_NoWatermarkFree(t *testing.T) {
	svc := createTestService(t)

	// Pro tier normally doesn't require watermark, but features array doesn't grant watermark-free
	ent := &Entitlement{
		Tier:     "pro",
		Features: []string{FeatureAI, FeatureRecording}, // No watermark_free
	}

	if !svc.RequiresWatermarkWithEntitlement(ent) {
		t.Error("expected RequiresWatermarkWithEntitlement to return true when features array doesn't contain watermark_free")
	}
}

func TestRequiresWatermarkWithEntitlement_NilEntitlement(t *testing.T) {
	svc := createTestService(t)

	// Nil entitlement should fail safe - require watermark
	if !svc.RequiresWatermarkWithEntitlement(nil) {
		t.Error("expected RequiresWatermarkWithEntitlement to return true (require watermark) for nil entitlement")
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
