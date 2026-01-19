package entitlement

import (
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
		OfflineGracePeriod: 24 * time.Hour,
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
		WatermarkTiers:  []string{"free"},
		AITiers:         []string{"solo", "pro", "studio", "business"},
		RecordingTiers:  []string{"pro", "studio", "business"},
	}

	return NewService(cfg, log)
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
