package entitlement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
)

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
	}
}

// GetEntitlement retrieves the entitlement for a user, using cache when available.
func (s *Service) GetEntitlement(ctx context.Context, userIdentity string) (*Entitlement, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))

	// If entitlements are disabled, return unlimited entitlement
	if !s.cfg.Enabled {
		return s.unlimitedEntitlement(userIdentity), nil
	}

	// Empty user identity gets default tier
	if userIdentity == "" {
		return s.defaultEntitlement(""), nil
	}

	// Check cache first
	if cached := s.getCached(userIdentity); cached != nil && !cached.IsExpired() {
		return cached, nil
	}

	// Fetch from service
	ent, err := s.fetchEntitlement(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).WithField("user", userIdentity).Warn("Failed to fetch entitlement, using cached or default")

		// Try to use stale cache if within offline grace period
		if cached := s.getCached(userIdentity); cached != nil {
			if s.withinOfflineGrace() {
				s.log.WithField("user", userIdentity).Debug("Using stale cache within offline grace period")
				return cached, nil
			}
		}

		// Fall back to default tier
		return s.defaultEntitlement(userIdentity), nil
	}

	// Cache the result
	s.setCached(userIdentity, ent)
	s.markOnline()

	return ent, nil
}

// CanExecuteWorkflow checks if the user can execute a workflow based on their tier limits.
// Returns true if execution is allowed, false if limit reached.
func (s *Service) CanExecuteWorkflow(ctx context.Context, userIdentity string, currentMonthCount int) bool {
	if !s.cfg.Enabled {
		return true
	}

	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		// Fail open - allow execution if we can't check
		s.log.WithError(err).Warn("Failed to check entitlement, allowing execution")
		return true
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
	if !s.cfg.Enabled {
		return -1
	}

	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		return -1
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
	if !s.cfg.Enabled {
		return false
	}

	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		// Fail safe - require watermark if we can't check
		return true
	}

	return s.tierRequiresWatermark(ent.Tier)
}

// CanUseAI returns true if the user has access to AI-powered features.
func (s *Service) CanUseAI(ctx context.Context, userIdentity string) bool {
	if !s.cfg.Enabled {
		return true
	}

	ent, err := s.GetEntitlement(ctx, userIdentity)
	if err != nil {
		// Fail closed for premium features
		return false
	}

	return s.tierCanUseAI(ent.Tier)
}

// CanUseRecording returns true if the user has access to live recording features.
func (s *Service) CanUseRecording(ctx context.Context, userIdentity string) bool {
	if !s.cfg.Enabled {
		return true
	}

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

// fetchEntitlement calls the remote entitlement service.
func (s *Service) fetchEntitlement(ctx context.Context, userIdentity string) (*Entitlement, error) {
	if s.cfg.ServiceURL == "" {
		return nil, fmt.Errorf("entitlement service URL not configured")
	}

	// Build request URL
	reqURL, err := url.Parse(s.cfg.ServiceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid service URL: %w", err)
	}
	reqURL.Path = strings.TrimSuffix(reqURL.Path, "/") + "/api/v1/entitlements"
	q := reqURL.Query()
	q.Set("user", userIdentity)
	reqURL.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-User-Email", userIdentity)

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var entResp entitlementResponse
	if err := json.NewDecoder(resp.Body).Decode(&entResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to our entitlement type
	now := time.Now()
	ent := &Entitlement{
		UserIdentity: userIdentity,
		Status:       Status(entResp.Status),
		Tier:         Tier(strings.ToLower(entResp.PlanTier)),
		PriceID:      entResp.PriceID,
		Features:     entResp.Features,
		FetchedAt:    now,
		ExpiresAt:    now.Add(s.cfg.CacheTTL),
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

// unlimitedEntitlement returns an entitlement with no restrictions (for dev mode).
func (s *Service) unlimitedEntitlement(userIdentity string) *Entitlement {
	now := time.Now()
	return &Entitlement{
		UserIdentity: userIdentity,
		Status:       StatusActive,
		Tier:         TierBusiness,
		FetchedAt:    now,
		ExpiresAt:    now.Add(24 * time.Hour),
	}
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
