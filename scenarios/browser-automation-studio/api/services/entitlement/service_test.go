package entitlement

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
)

// createTestService creates a service instance for testing.
func createTestService(t *testing.T) *Service {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	cfg := config.EntitlementConfig{
		RequestTimeout:     5 * time.Second,
		CacheTTL:           5 * time.Minute,
		OfflineGracePeriod: 5 * time.Hour,
		DefaultTier:        "free",
		ServiceURL:         "https://vrooli.com",
		TierLimits: map[string]int{
			"free":     50,
			"solo":     200,
			"pro":      -1,
			"studio":   -1,
			"business": -1,
		},
		AICreditsLimits: map[string]int{
			"free":     10,
			"solo":     50,
			"pro":      -1,
			"studio":   -1,
			"business": -1,
		},
		WatermarkTiers: []string{"free"},
		AITiers:        []string{"solo", "pro", "studio", "business"},
		RecordingTiers: []string{"pro", "studio", "business"},
	}

	return NewService(cfg, log)
}

func TestGetEntitlementBindsAuthorityQueryAndResponseIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("user"); got != "alice@example.com" {
			t.Fatalf("authority query user = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer opaque-access" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"active","plan_tier":"pro","credits":{"customer_email":"alice@example.com","balance_credits":41},"subscription":{"user_identity":"alice@example.com"}}`))
	}))
	defer server.Close()
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	ent, err := svc.GetEntitlement(WithAccessToken(context.Background(), "opaque-access"), " Alice@Example.com ")
	if err != nil {
		t.Fatal(err)
	}
	if ent.UserIdentity != "alice@example.com" || ent.Tier != TierPro || ent.Credits != 41 {
		t.Fatalf("entitlement = %#v", ent)
	}
}

func TestGetEntitlementRejectsAuthorityIdentityMismatchWithoutCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"active","plan_tier":"business","subscription":{"user_identity":"bob@example.com"}}`))
	}))
	defer server.Close()
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	_, err := svc.GetEntitlement(WithAccessToken(context.Background(), "opaque-access"), "alice@example.com")
	if err == nil || !strings.Contains(err.Error(), "authority subscription identity does not match") {
		t.Fatalf("mismatch error = %v", err)
	}
}

func TestGetEntitlementRejectsAuthorityResponseWithoutIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"active","plan_tier":"business"}`))
	}))
	defer server.Close()
	svc := createTestService(t)
	svc.cfg.ServiceURL = server.URL
	svc.httpClient = server.Client()

	_, err := svc.GetEntitlement(WithAccessToken(context.Background(), "opaque-access"), "alice@example.com")
	if err == nil || !strings.Contains(err.Error(), "authority response did not establish identity") {
		t.Fatalf("missing identity error = %v", err)
	}
}

// ============================================================================
// SetApiSource Tests
// ============================================================================

func TestSetApiSource_Production(t *testing.T) {
	svc := createTestService(t)

	svc.SetApiSource("production", 0)

	source, localPort := svc.GetApiSource()

	if source != "production" {
		t.Errorf("expected source 'production', got %q", source)
	}
	// Local port should remain default
	if localPort != 15000 {
		t.Errorf("expected localPort 15000, got %d", localPort)
	}
}

func TestSetApiSource_Local(t *testing.T) {
	svc := createTestService(t)

	svc.SetApiSource("local", 17000)

	source, localPort := svc.GetApiSource()

	if source != "local" {
		t.Errorf("expected source 'local', got %q", source)
	}
	if localPort != 17000 {
		t.Errorf("expected localPort 17000, got %d", localPort)
	}
}

func TestSetApiSource_Disabled(t *testing.T) {
	svc := createTestService(t)

	svc.SetApiSource("disabled", 0)

	source, _ := svc.GetApiSource()

	if source != "disabled" {
		t.Errorf("expected source 'disabled', got %q", source)
	}
}

func TestSetApiSource_ClearsCache(t *testing.T) {
	svc := createTestService(t)

	// Manually add something to the cache
	svc.cacheMu.Lock()
	svc.cache["test@example.com"] = &Entitlement{
		UserIdentity: "test@example.com",
		Tier:         TierPro,
	}
	svc.cacheMu.Unlock()

	// Verify cache has entry
	svc.cacheMu.RLock()
	if len(svc.cache) != 1 {
		t.Fatalf("expected cache to have 1 entry, got %d", len(svc.cache))
	}
	svc.cacheMu.RUnlock()

	// Change API source
	svc.SetApiSource("local", 16000)

	// Cache should be cleared
	svc.cacheMu.RLock()
	if len(svc.cache) != 0 {
		t.Errorf("expected cache to be cleared, got %d entries", len(svc.cache))
	}
	svc.cacheMu.RUnlock()
}

func TestSetApiSource_LocalPortNotUpdatedWhenZero(t *testing.T) {
	svc := createTestService(t)

	// First set a specific local port
	svc.SetApiSource("local", 18000)

	// Then set source without specifying port
	svc.SetApiSource("local", 0)

	_, localPort := svc.GetApiSource()

	// Port should remain unchanged from the previous setting
	if localPort != 18000 {
		t.Errorf("expected localPort to remain 18000, got %d", localPort)
	}
}

// ============================================================================
// GetApiSource Tests
// ============================================================================

func TestGetApiSource_DefaultValues(t *testing.T) {
	svc := createTestService(t)

	source, localPort := svc.GetApiSource()

	if source != "production" {
		t.Errorf("expected default source 'production', got %q", source)
	}
	if localPort != 15000 {
		t.Errorf("expected default localPort 15000, got %d", localPort)
	}
}

func TestGetApiSource_AfterSetLocal(t *testing.T) {
	svc := createTestService(t)

	svc.SetApiSource("local", 19000)

	source, localPort := svc.GetApiSource()

	if source != "local" {
		t.Errorf("expected source 'local', got %q", source)
	}
	if localPort != 19000 {
		t.Errorf("expected localPort 19000, got %d", localPort)
	}
}

// ============================================================================
// BuildOverrideEntitlement Tests
// ============================================================================

func TestBuildOverrideEntitlement_Success(t *testing.T) {
	svc := createTestService(t)

	ent := svc.BuildOverrideEntitlement("test@example.com", TierPro)

	if ent.UserIdentity != "test@example.com" {
		t.Errorf("expected user identity 'test@example.com', got %q", ent.UserIdentity)
	}
	if ent.Tier != TierPro {
		t.Errorf("expected tier 'pro', got %q", ent.Tier)
	}
	if ent.Status != StatusActive {
		t.Errorf("expected status 'active', got %q", ent.Status)
	}
	if ent.IsExpired() {
		t.Error("expected entitlement to not be expired")
	}
}

func TestBuildOverrideEntitlement_AllTiers(t *testing.T) {
	svc := createTestService(t)

	tiers := []Tier{TierFree, TierSolo, TierPro, TierStudio, TierBusiness}

	for _, tier := range tiers {
		t.Run(string(tier), func(t *testing.T) {
			ent := svc.BuildOverrideEntitlement("user@test.com", tier)

			if ent.Tier != tier {
				t.Errorf("expected tier %q, got %q", tier, ent.Tier)
			}
			if ent.Status != StatusActive {
				t.Errorf("expected status 'active', got %q", ent.Status)
			}
		})
	}
}

// ============================================================================
// InvalidateCache Tests
// ============================================================================

func TestInvalidateCache_RemovesEntry(t *testing.T) {
	svc := createTestService(t)

	// Add cache entry
	svc.cacheMu.Lock()
	svc.cache["user@example.com"] = &Entitlement{
		UserIdentity: "user@example.com",
		Tier:         TierPro,
	}
	svc.cacheMu.Unlock()

	// Invalidate
	svc.InvalidateCache("user@example.com")

	// Check it's gone
	svc.cacheMu.RLock()
	if _, exists := svc.cache["user@example.com"]; exists {
		t.Error("expected cache entry to be removed")
	}
	svc.cacheMu.RUnlock()
}

func TestInvalidateCache_NormalizesIdentity(t *testing.T) {
	svc := createTestService(t)

	// Add cache entry with lowercase key
	svc.cacheMu.Lock()
	svc.cache["user@example.com"] = &Entitlement{
		UserIdentity: "user@example.com",
		Tier:         TierPro,
	}
	svc.cacheMu.Unlock()

	// Invalidate with mixed case and whitespace
	svc.InvalidateCache("  User@Example.COM  ")

	// Check it's gone
	svc.cacheMu.RLock()
	if _, exists := svc.cache["user@example.com"]; exists {
		t.Error("expected cache entry to be removed after normalization")
	}
	svc.cacheMu.RUnlock()
}

func TestInvalidateCache_NoopForNonexistent(t *testing.T) {
	svc := createTestService(t)

	// Should not panic or error
	svc.InvalidateCache("nonexistent@example.com")

	svc.cacheMu.RLock()
	if len(svc.cache) != 0 {
		t.Error("expected cache to remain empty")
	}
	svc.cacheMu.RUnlock()
}

// ============================================================================
// GetTierLimit Tests
// ============================================================================

func TestTierLimit_Free(t *testing.T) {
	svc := createTestService(t)

	limit := svc.TierLimit(TierFree)

	if limit != 50 {
		t.Errorf("expected free tier limit 50, got %d", limit)
	}
}

func TestTierLimit_Solo(t *testing.T) {
	svc := createTestService(t)

	limit := svc.TierLimit(TierSolo)

	if limit != 200 {
		t.Errorf("expected solo tier limit 200, got %d", limit)
	}
}

func TestTierLimit_Pro(t *testing.T) {
	svc := createTestService(t)

	limit := svc.TierLimit(TierPro)

	// Pro should be unlimited (-1)
	if limit != -1 {
		t.Errorf("expected pro tier limit -1 (unlimited), got %d", limit)
	}
}

// ============================================================================
// TierRequiresWatermark Tests
// ============================================================================

func TestTierRequiresWatermark_Free(t *testing.T) {
	svc := createTestService(t)

	if !svc.TierRequiresWatermark(TierFree) {
		t.Error("expected free tier to require watermark")
	}
}

func TestTierRequiresWatermark_Pro(t *testing.T) {
	svc := createTestService(t)

	if svc.TierRequiresWatermark(TierPro) {
		t.Error("expected pro tier to NOT require watermark")
	}
}

// ============================================================================
// TierCanUseAI Tests
// ============================================================================

func TestTierCanUseAI_Free(t *testing.T) {
	svc := createTestService(t)

	if svc.TierCanUseAI(TierFree) {
		t.Error("expected free tier to NOT have AI access")
	}
}

func TestTierCanUseAI_Solo(t *testing.T) {
	svc := createTestService(t)

	if !svc.TierCanUseAI(TierSolo) {
		t.Error("expected solo tier to have AI access")
	}
}

func TestTierCanUseAI_Pro(t *testing.T) {
	svc := createTestService(t)

	if !svc.TierCanUseAI(TierPro) {
		t.Error("expected pro tier to have AI access")
	}
}

// ============================================================================
// TierCanUseRecording Tests
// ============================================================================

func TestTierCanUseRecording_Free(t *testing.T) {
	svc := createTestService(t)

	if svc.TierCanUseRecording(TierFree) {
		t.Error("expected free tier to NOT have recording access")
	}
}

func TestTierCanUseRecording_Solo(t *testing.T) {
	svc := createTestService(t)

	if svc.TierCanUseRecording(TierSolo) {
		t.Error("expected solo tier to NOT have recording access")
	}
}

func TestTierCanUseRecording_Pro(t *testing.T) {
	svc := createTestService(t)

	if !svc.TierCanUseRecording(TierPro) {
		t.Error("expected pro tier to have recording access")
	}
}

// ============================================================================
// GetAICreditsLimit Tests
// ============================================================================

func TestGetAICreditsLimit_Free(t *testing.T) {
	svc := createTestService(t)

	limit := svc.GetAICreditsLimit(TierFree)

	if limit != 10 {
		t.Errorf("expected free tier AI credits limit 10, got %d", limit)
	}
}

func TestGetAICreditsLimit_Solo(t *testing.T) {
	svc := createTestService(t)

	limit := svc.GetAICreditsLimit(TierSolo)

	if limit != 50 {
		t.Errorf("expected solo tier AI credits limit 50, got %d", limit)
	}
}

func TestGetAICreditsLimit_Pro(t *testing.T) {
	svc := createTestService(t)

	limit := svc.GetAICreditsLimit(TierPro)

	// Pro should be unlimited (-1)
	if limit != -1 {
		t.Errorf("expected pro tier AI credits limit -1 (unlimited), got %d", limit)
	}
}

func TestGetAICreditsLimit_UnknownTier(t *testing.T) {
	svc := createTestService(t)

	limit := svc.GetAICreditsLimit(Tier("unknown"))

	// Unknown tier should return 0 (no access)
	if limit != 0 {
		t.Errorf("expected unknown tier AI credits limit 0, got %d", limit)
	}
}

// ============================================================================
// MinTier Tests
// ============================================================================

func TestMinTierForAI(t *testing.T) {
	svc := createTestService(t)

	minTier := svc.MinTierForAI()

	// Based on AITiers config: solo, pro, studio, business
	// Solo should be the minimum
	if minTier != TierSolo {
		t.Errorf("expected min tier for AI to be 'solo', got %q", minTier)
	}
}

func TestMinTierForRecording(t *testing.T) {
	svc := createTestService(t)

	minTier := svc.MinTierForRecording()

	// Based on RecordingTiers config: pro, studio, business
	// Pro should be the minimum
	if minTier != TierPro {
		t.Errorf("expected min tier for recording to be 'pro', got %q", minTier)
	}
}

func TestMinTierWithoutWatermark(t *testing.T) {
	svc := createTestService(t)

	minTier := svc.MinTierWithoutWatermark()

	// Based on WatermarkTiers config: free requires watermark
	// Solo should be the first tier without watermark
	if minTier != TierSolo {
		t.Errorf("expected min tier without watermark to be 'solo', got %q", minTier)
	}
}

func TestMinTierForAICredits(t *testing.T) {
	svc := createTestService(t)

	minTier := svc.MinTierForAICredits()

	// Free tier has 10 credits (not 0), so free should be the minimum
	if minTier != TierFree {
		t.Errorf("expected min tier for AI credits to be 'free', got %q", minTier)
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
		{StatusActive, true},
		{StatusTrialing, true},
		{StatusPastDue, false},
		{StatusCanceled, false},
		{StatusInactive, false},
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
// Tier Type Tests
// ============================================================================

func TestTier_Order(t *testing.T) {
	testCases := []struct {
		tier     Tier
		expected int
	}{
		{TierFree, 1},
		{TierSolo, 2},
		{TierPro, 3},
		{TierStudio, 4},
		{TierBusiness, 5},
		{Tier("unknown"), 0},
	}

	for _, tc := range testCases {
		t.Run(string(tc.tier), func(t *testing.T) {
			if tc.tier.Order() != tc.expected {
				t.Errorf("expected order %d for tier %q, got %d", tc.expected, tc.tier, tc.tier.Order())
			}
		})
	}
}

func TestTier_AtLeast(t *testing.T) {
	// Pro should be at least Solo
	if !TierPro.AtLeast(TierSolo) {
		t.Error("expected Pro.AtLeast(Solo) to be true")
	}

	// Free should not be at least Pro
	if TierFree.AtLeast(TierPro) {
		t.Error("expected Free.AtLeast(Pro) to be false")
	}

	// Same tier should be at least itself
	if !TierPro.AtLeast(TierPro) {
		t.Error("expected Pro.AtLeast(Pro) to be true")
	}
}

func TestParseTier(t *testing.T) {
	testCases := []struct {
		input    string
		expected Tier
		ok       bool
	}{
		{"free", TierFree, true},
		{"FREE", TierFree, true},
		{"  Free  ", TierFree, true},
		{"solo", TierSolo, true},
		{"pro", TierPro, true},
		{"studio", TierStudio, true},
		{"business", TierBusiness, true},
		{"invalid", "", false},
		{"", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			tier, ok := ParseTier(tc.input)
			if ok != tc.ok {
				t.Errorf("expected ok=%v for input %q, got %v", tc.ok, tc.input, ok)
			}
			if ok && tier != tc.expected {
				t.Errorf("expected tier %q for input %q, got %q", tc.expected, tc.input, tier)
			}
		})
	}
}

// ============================================================================
// Feature-Aware Method Tests (GAP-004)
// ============================================================================

func TestCanUseAIWithEntitlement_FeaturesArray_HasAI(t *testing.T) {
	svc := createTestService(t)

	// Entitlement with AI feature in features array (free tier normally can't use AI)
	ent := &Entitlement{
		Tier:     TierFree,
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
		Tier:     TierPro,
		Features: []string{FeatureRecording, FeatureWatermarkFree}, // No AI
	}

	// Features array is authoritative - should deny AI despite pro tier
	if svc.CanUseAIWithEntitlement(ent) {
		t.Error("expected CanUseAIWithEntitlement to return false when features array doesn't contain AI")
	}
}

func TestCanUseAIWithEntitlement_EmptyFeatures_FallsBackToTier(t *testing.T) {
	svc := createTestService(t)

	// Pro tier with empty features array - should fall back to tier check
	ent := &Entitlement{
		Tier:     TierPro,
		Features: []string{}, // Empty
	}

	// Empty features = fallback to tier (pro has AI)
	if !svc.CanUseAIWithEntitlement(ent) {
		t.Error("expected CanUseAIWithEntitlement to fall back to tier check when features array is empty")
	}

	// Free tier with empty features array - should fall back to tier check
	entFree := &Entitlement{
		Tier:     TierFree,
		Features: []string{}, // Empty
	}

	// Empty features = fallback to tier (free doesn't have AI)
	if svc.CanUseAIWithEntitlement(entFree) {
		t.Error("expected CanUseAIWithEntitlement to deny free tier when features array is empty")
	}
}

func TestCanUseAIWithEntitlement_NilFeatures_FallsBackToTier(t *testing.T) {
	svc := createTestService(t)

	// Pro tier with nil features - should fall back to tier check
	ent := &Entitlement{
		Tier:     TierPro,
		Features: nil,
	}

	if !svc.CanUseAIWithEntitlement(ent) {
		t.Error("expected CanUseAIWithEntitlement to fall back to tier check when features is nil")
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
		Tier:     TierSolo,
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
		Tier:     TierPro,
		Features: []string{FeatureAI}, // No recording
	}

	if svc.CanUseRecordingWithEntitlement(ent) {
		t.Error("expected CanUseRecordingWithEntitlement to return false when features array doesn't contain recording")
	}
}

func TestCanUseRecordingWithEntitlement_NilFeatures_FallsBackToTier(t *testing.T) {
	svc := createTestService(t)

	// Pro tier with nil features - should fall back to tier check
	ent := &Entitlement{
		Tier:     TierPro,
		Features: nil,
	}

	if !svc.CanUseRecordingWithEntitlement(ent) {
		t.Error("expected CanUseRecordingWithEntitlement to fall back to tier check when features is nil")
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
		Tier:     TierFree,
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
		Tier:     TierPro,
		Features: []string{FeatureAI, FeatureRecording}, // No watermark_free
	}

	if !svc.RequiresWatermarkWithEntitlement(ent) {
		t.Error("expected RequiresWatermarkWithEntitlement to return true when features array doesn't contain watermark_free")
	}
}

func TestRequiresWatermarkWithEntitlement_NilFeatures_FallsBackToTier(t *testing.T) {
	svc := createTestService(t)

	// Pro tier with nil features - should fall back to tier check (pro doesn't require watermark)
	ent := &Entitlement{
		Tier:     TierPro,
		Features: nil,
	}

	if svc.RequiresWatermarkWithEntitlement(ent) {
		t.Error("expected RequiresWatermarkWithEntitlement to fall back to tier check when features is nil")
	}

	// Free tier with nil features - should fall back to tier check (free requires watermark)
	entFree := &Entitlement{
		Tier:     TierFree,
		Features: nil,
	}

	if !svc.RequiresWatermarkWithEntitlement(entFree) {
		t.Error("expected RequiresWatermarkWithEntitlement to require watermark for free tier when features is nil")
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
