package entitlement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

const accessTokenContextKey contextKey = "subscription_access_token"

var (
	ErrAccessTokenRequired     = errors.New("subscription access token is required")
	ErrEntitlementUnavailable  = errors.New("subscription entitlement service is unavailable")
	ErrEntitlementUnauthorized = errors.New("subscription access token was rejected")
)

// WithAccessToken attaches the short-lived consumer access token to a
// request-scoped entitlement lookup. It is never persisted or cached.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenContextKey, strings.TrimSpace(token))
}

func accessTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(accessTokenContextKey).(string)
	return strings.TrimSpace(token)
}

// Service provides entitlement checking and feature gating.
type Service struct {
	cfg        config.EntitlementConfig
	log        *logrus.Logger
	httpClient *http.Client

	// Cache
	cacheMu sync.RWMutex
	cache   map[string]*Entitlement

	// Offline tracking
	lastSuccessfulFetch time.Time
	offlineMu           sync.RWMutex

	// API source override (for dev mode)
	apiSourceMu     sync.RWMutex
	apiSource       string // "production", "local", or "disabled"
	localPort       int    // Port for local LPBS API
	sessionResolver *credentialclient.ConsumerSessionResolver
}

// NewService creates a new entitlement service.
func NewService(cfg config.EntitlementConfig, log *logrus.Logger) *Service {
	return &Service{
		cfg: cfg,
		log: log,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		cache:               make(map[string]*Entitlement),
		lastSuccessfulFetch: time.Now(), // Assume online at startup
		apiSource:           "production",
		localPort:           15000, // Default LPBS API port range start
	}
}

// SetSessionResolver wires the platform-owned shared subscription session.
// The resolver keeps only the short-lived access token in memory.
func (s *Service) SetSessionResolver(resolver *credentialclient.ConsumerSessionResolver) {
	s.sessionResolver = resolver
}

// SetApiSource sets the API source for entitlement verification.
// source can be "production", "local", or "disabled".
// localPort is the port for local LPBS API (used when source is "local").
func (s *Service) SetApiSource(source string, localPort int) {
	s.apiSourceMu.Lock()
	defer s.apiSourceMu.Unlock()

	s.apiSource = source
	if localPort > 0 {
		s.localPort = localPort
	}

	// Clear cache when switching sources to get fresh data
	s.cacheMu.Lock()
	s.cache = make(map[string]*Entitlement)
	s.cacheMu.Unlock()
	if s.sessionResolver != nil {
		s.sessionResolver.Clear()
	}
}

// GetApiSource returns the current API source configuration.
func (s *Service) GetApiSource() (source string, localPort int) {
	s.apiSourceMu.RLock()
	defer s.apiSourceMu.RUnlock()
	return s.apiSource, s.localPort
}

// GetEntitlement retrieves the entitlement for a user, using cache when available.
func (s *Service) GetEntitlement(ctx context.Context, userIdentity string) (*Entitlement, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))

	// Empty identity is never a valid paid lookup. The token's verified claims
	// are the only identity accepted by LPBS.
	if userIdentity == "" {
		return nil, ErrAccessTokenRequired
	}

	// Check cache first
	if cached := s.getCached(userIdentity); cached != nil && !cached.IsExpired() {
		return cached, nil
	}

	// Fetch from service
	ent, err := s.fetchEntitlement(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).WithField("user", userIdentity).Warn("Failed to fetch entitlement; denying until subscription is verified")
		return nil, err
	}

	// Cache the result
	s.setCached(userIdentity, ent)
	s.markOnline()

	return ent, nil
}

// CanExecuteWorkflow checks if the user can execute a workflow based on their tier limits.
// Returns true if execution is allowed, false if limit reached.
func (s *Service) CanExecuteWorkflow(ctx context.Context, userIdentity string, currentMonthCount int) bool {
	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).Warn("Failed to check entitlement; denying workflow until subscription is verified")
		return false
	}

	limit := s.getTierLimit(ent.Tier)
	if limit < 0 {
		// Unlimited
		return true
	}

	return currentMonthCount < limit
}

// GetRemainingExecutions returns how many executions the user has remaining this month.
// Returns -1 for unlimited.
func (s *Service) GetRemainingExecutions(ctx context.Context, userIdentity string, currentMonthCount int) int {
	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		return 0
	}

	limit := s.getTierLimit(ent.Tier)
	if limit < 0 {
		return -1
	}

	remaining := limit - currentMonthCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RequiresWatermark returns true if exports for this user should be watermarked.
func (s *Service) RequiresWatermark(ctx context.Context, userIdentity string) bool {
	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		// Fail safe - require watermark if we can't check
		return true
	}

	return s.tierRequiresWatermark(ent.Tier)
}

// CanUseAI returns true if the user has access to AI-powered features.
func (s *Service) CanUseAI(ctx context.Context, userIdentity string) bool {
	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		// Fail closed for premium features - don't allow AI access if we can't verify entitlement.
		// This is a local pre-check; the LPBS AI gateway provides the authoritative credit check
		// with atomic reservation when processing actual AI requests.
		return false
	}

	return s.tierCanUseAI(ent.Tier)
}

// CanUseRecording returns true if the user has access to live recording features.
func (s *Service) CanUseRecording(ctx context.Context, userIdentity string) bool {
	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		// Fail closed for premium features
		return false
	}

	return s.tierCanUseRecording(ent.Tier)
}

// InvalidateCache removes a user's cached entitlement, forcing a refresh on next check.
func (s *Service) InvalidateCache(userIdentity string) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	s.cacheMu.Lock()
	delete(s.cache, userIdentity)
	s.cacheMu.Unlock()
}

// BuildOverrideEntitlement creates a local entitlement for tier overrides.
func (s *Service) BuildOverrideEntitlement(userIdentity string, tier Tier) *Entitlement {
	now := time.Now()
	return &Entitlement{
		UserIdentity: userIdentity,
		Status:       StatusActive,
		Tier:         tier,
		FetchedAt:    now,
		ExpiresAt:    now.Add(s.cfg.CacheTTL),
	}
}

// fetchEntitlement calls the remote entitlement service.
func (s *Service) fetchEntitlement(ctx context.Context, userIdentity string) (*Entitlement, error) {
	// Get current API source
	s.apiSourceMu.RLock()
	apiSource := s.apiSource
	localPort := s.localPort
	s.apiSourceMu.RUnlock()

	// If disabled, return nil to use default tier
	if apiSource == "disabled" {
		return nil, fmt.Errorf("%w: API source is disabled", ErrEntitlementUnavailable)
	}

	// Determine the service URL based on api source
	var serviceURL string
	if apiSource == "local" {
		serviceURL = fmt.Sprintf("http://localhost:%d", localPort)
	} else {
		// Production: use configured service URL
		serviceURL = s.cfg.ServiceURL
		if serviceURL == "" {
			return nil, fmt.Errorf("%w: service URL is not configured", ErrEntitlementUnavailable)
		}
	}
	if accessTokenFromContext(ctx) == "" {
		if s.sessionResolver == nil {
			return nil, ErrAccessTokenRequired
		}
		access, err := s.sessionResolver.ResolveAt(ctx, serviceURL)
		if err != nil {
			return nil, err
		}
		ctx = WithAccessToken(ctx, access.AccessToken)
	}

	// Build request URL
	reqURL, err := url.Parse(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid service URL: %w", err)
	}
	reqURL.Path = strings.TrimSuffix(reqURL.Path, "/") + "/api/v1/entitlements"
	query := reqURL.Query()
	// LPBS binds this query identity to the verified bearer-token identity.
	// Sending it prevents BAS from turning a caller-controlled requested user
	// into an entitlement lookup for the cached shared session.
	query.Set("user", userIdentity)
	reqURL.RawQuery = query.Encode()
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessTokenFromContext(ctx))

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("%w: status %d", ErrEntitlementUnauthorized, resp.StatusCode)
		}
		if resp.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("%w: status %d", ErrEntitlementUnavailable, resp.StatusCode)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var entResp entitlementResponse
	if err := json.NewDecoder(resp.Body).Decode(&entResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if entResp.Credits != nil && strings.TrimSpace(entResp.Credits.CustomerEmail) != "" &&
		!strings.EqualFold(strings.TrimSpace(entResp.Credits.CustomerEmail), userIdentity) {
		return nil, fmt.Errorf("%w: authority identity does not match requested identity", ErrEntitlementUnauthorized)
	}
	if entResp.Subscription != nil && strings.TrimSpace(entResp.Subscription.UserIdentity) != "" &&
		!strings.EqualFold(strings.TrimSpace(entResp.Subscription.UserIdentity), userIdentity) {
		return nil, fmt.Errorf("%w: authority subscription identity does not match requested identity", ErrEntitlementUnauthorized)
	}
	if (entResp.Credits == nil || strings.TrimSpace(entResp.Credits.CustomerEmail) == "") &&
		(entResp.Subscription == nil || strings.TrimSpace(entResp.Subscription.UserIdentity) == "") {
		return nil, fmt.Errorf("%w: authority response did not establish identity", ErrEntitlementUnauthorized)
	}

	// Convert to our entitlement type
	now := time.Now()
	ent := &Entitlement{
		UserIdentity:      userIdentity,
		Status:            Status(entResp.Status),
		Tier:              Tier(strings.ToLower(entResp.PlanTier)),
		PriceID:           entResp.PriceID,
		Features:          entResp.Features,
		BillingCycleStart: entResp.BillingCycleStart,
		FetchedAt:         now,
		ExpiresAt:         now.Add(s.cfg.CacheTTL),
	}

	// Handle credits if present
	if entResp.Credits != nil {
		ent.Credits = entResp.Credits.BalanceCredits
	}

	// Normalize tier
	if ent.Tier == "" {
		ent.Tier = TierFree
	}

	// Normalize status
	if ent.Status == "" {
		ent.Status = StatusInactive
	}

	return ent, nil
}

// getCached retrieves a cached entitlement.
func (s *Service) getCached(userIdentity string) *Entitlement {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.cache[userIdentity]
}

// setCached stores an entitlement in the cache.
func (s *Service) setCached(userIdentity string, ent *Entitlement) {
	s.cacheMu.Lock()
	s.cache[userIdentity] = ent
	s.cacheMu.Unlock()
}

// markOnline records a successful fetch.
func (s *Service) markOnline() {
	s.offlineMu.Lock()
	s.lastSuccessfulFetch = time.Now()
	s.offlineMu.Unlock()
}

// withinOfflineGrace returns true if we're within the offline grace period.
func (s *Service) withinOfflineGrace() bool {
	s.offlineMu.RLock()
	last := s.lastSuccessfulFetch
	s.offlineMu.RUnlock()
	return time.Since(last) < s.cfg.OfflineGracePeriod
}

// defaultEntitlement returns the default (free tier) entitlement.
func (s *Service) defaultEntitlement(userIdentity string) *Entitlement {
	now := time.Now()
	return &Entitlement{
		UserIdentity: userIdentity,
		Status:       StatusInactive,
		Tier:         Tier(s.cfg.DefaultTier),
		FetchedAt:    now,
		ExpiresAt:    now.Add(s.cfg.CacheTTL),
	}
}

// TierLimit returns the execution limit for a tier.
// Returns -1 for unlimited.
func (s *Service) TierLimit(tier Tier) int {
	return s.getTierLimit(tier)
}

// TierRequiresWatermark checks if a tier requires watermarked exports.
func (s *Service) TierRequiresWatermark(tier Tier) bool {
	return s.tierRequiresWatermark(tier)
}

// TierCanUseAI checks if a tier has access to AI features.
func (s *Service) TierCanUseAI(tier Tier) bool {
	return s.tierCanUseAI(tier)
}

// TierCanUseRecording checks if a tier has access to recording features.
func (s *Service) TierCanUseRecording(tier Tier) bool {
	return s.tierCanUseRecording(tier)
}

// MinTierForAI returns the lowest tier that grants AI access.
func (s *Service) MinTierForAI() Tier {
	return minTierFromList(s.cfg.AITiers)
}

// MinTierForRecording returns the lowest tier that grants recording access.
func (s *Service) MinTierForRecording() Tier {
	return minTierFromList(s.cfg.RecordingTiers)
}

// MinTierWithoutWatermark returns the lowest tier that removes watermarks.
func (s *Service) MinTierWithoutWatermark() Tier {
	watermarkTiers := make(map[string]struct{}, len(s.cfg.WatermarkTiers))
	for _, tier := range s.cfg.WatermarkTiers {
		normalized := strings.TrimSpace(strings.ToLower(tier))
		if normalized != "" {
			watermarkTiers[normalized] = struct{}{}
		}
	}

	for _, tier := range []Tier{TierFree, TierSolo, TierPro, TierStudio, TierBusiness} {
		if _, exists := watermarkTiers[string(tier)]; !exists {
			return tier
		}
	}
	return ""
}

// GetAICreditsLimit returns the AI credits limit for a tier.
// Returns -1 for unlimited, 0 for no access.
func (s *Service) GetAICreditsLimit(tier Tier) int {
	if limit, ok := s.cfg.AICreditsLimits[string(tier)]; ok {
		return limit
	}
	// Default to 0 (no access) if tier not found
	return 0
}

// MinTierForAICredits returns the lowest tier that grants AI credits access.
func (s *Service) MinTierForAICredits() Tier {
	for _, tier := range []Tier{TierFree, TierSolo, TierPro, TierStudio, TierBusiness} {
		limit := s.GetAICreditsLimit(tier)
		if limit != 0 { // Either has credits or unlimited
			return tier
		}
	}
	return ""
}

func minTierFromList(tiers []string) Tier {
	var selected Tier
	for _, entry := range tiers {
		tier, ok := ParseTier(entry)
		if !ok {
			continue
		}
		if selected == "" || tier.Order() < selected.Order() {
			selected = tier
		}
	}
	return selected
}

// getTierLimit returns the execution limit for a tier.
// Returns -1 for unlimited.
func (s *Service) getTierLimit(tier Tier) int {
	if limit, ok := s.cfg.TierLimits[string(tier)]; ok {
		return limit
	}
	// Default to free tier limit if tier not found
	if limit, ok := s.cfg.TierLimits["free"]; ok {
		return limit
	}
	return 50 // Hardcoded fallback
}

// tierRequiresWatermark checks if a tier requires watermarked exports.
func (s *Service) tierRequiresWatermark(tier Tier) bool {
	tierStr := string(tier)
	for _, t := range s.cfg.WatermarkTiers {
		if strings.EqualFold(t, tierStr) {
			return true
		}
	}
	return false
}

// tierCanUseAI checks if a tier has access to AI features.
func (s *Service) tierCanUseAI(tier Tier) bool {
	tierStr := string(tier)
	for _, t := range s.cfg.AITiers {
		if strings.EqualFold(t, tierStr) {
			return true
		}
	}
	return false
}

// tierCanUseRecording checks if a tier has access to recording features.
func (s *Service) tierCanUseRecording(tier Tier) bool {
	tierStr := string(tier)
	for _, t := range s.cfg.RecordingTiers {
		if strings.EqualFold(t, tierStr) {
			return true
		}
	}
	return false
}

// CanUseAIWithEntitlement checks AI access using features array first, then tier fallback.
// If the Features array is present (non-nil and non-empty), it is authoritative.
// If Features is nil or empty, falls back to tier-based config for backwards compatibility.
func (s *Service) CanUseAIWithEntitlement(ent *Entitlement) bool {
	if ent == nil {
		return false
	}
	// Features array is authoritative when present
	if len(ent.Features) > 0 {
		return ent.HasFeature(FeatureAI)
	}
	// Fall back to tier-based config for backwards compatibility
	return s.tierCanUseAI(ent.Tier)
}

// CanUseRecordingWithEntitlement checks recording access using features array first.
// If the Features array is present (non-nil and non-empty), it is authoritative.
// If Features is nil or empty, falls back to tier-based config for backwards compatibility.
func (s *Service) CanUseRecordingWithEntitlement(ent *Entitlement) bool {
	if ent == nil {
		return false
	}
	if len(ent.Features) > 0 {
		return ent.HasFeature(FeatureRecording)
	}
	return s.tierCanUseRecording(ent.Tier)
}

// RequiresWatermarkWithEntitlement checks if watermark is required using features array first.
// If the Features array is present (non-nil and non-empty), it is authoritative.
// If Features is nil or empty, falls back to tier-based config for backwards compatibility.
func (s *Service) RequiresWatermarkWithEntitlement(ent *Entitlement) bool {
	if ent == nil {
		return true // Fail safe: require watermark
	}
	if len(ent.Features) > 0 {
		return !ent.HasFeature(FeatureWatermarkFree)
	}
	return s.tierRequiresWatermark(ent.Tier)
}
