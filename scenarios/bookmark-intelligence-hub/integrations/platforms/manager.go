package platforms

import (
	"fmt"
	"sync"
	"time"
)

// PlatformManager coordinates all platform integrations
type PlatformManager struct {
	integrations map[string]PlatformIntegration
	configs      map[string]PlatformConfig
	mutex        sync.RWMutex
	syncStatus   map[string]SyncStatus
}

// NewPlatformManager creates a new platform manager instance
func NewPlatformManager() *PlatformManager {
	return &PlatformManager{
		integrations: make(map[string]PlatformIntegration),
		configs:      make(map[string]PlatformConfig),
		syncStatus:   make(map[string]SyncStatus),
	}
}

// RegisterPlatform registers a new platform integration
func (pm *PlatformManager) RegisterPlatform(name string, integration PlatformIntegration, config PlatformConfig) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	// Validate the integration
	if err := integration.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid configuration for platform %s: %w", name, err)
	}
	
	// Test connection
	if err := integration.TestConnection(); err != nil {
		config.Status = StatusError
		config.LastError = err.Error()
		config.ErrorCount++
	} else {
		config.Status = StatusActive
		config.LastError = ""
	}
	
	pm.integrations[name] = integration
	pm.configs[name] = config
	pm.syncStatus[name] = SyncStatus{
		PlatformName:   name,
		Status:        StatusIdle,
		LastSyncAt:    config.LastSyncAt,
		ItemsProcessed: 0,
	}
	
	return nil
}

// GetAvailablePlatforms returns all registered platform names
func (pm *PlatformManager) GetAvailablePlatforms() []string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	platforms := make([]string, 0, len(pm.integrations))
	for name := range pm.integrations {
		platforms = append(platforms, name)
	}
	return platforms
}

// GetPlatformInfo returns information about a specific platform
func (pm *PlatformManager) GetPlatformInfo(platformName string) (PlatformInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	integration, exists := pm.integrations[platformName]
	if !exists {
		return PlatformInfo{}, fmt.Errorf("platform %s not found", platformName)
	}
	
	return integration.GetPlatformInfo(), nil
}

// GetAllPlatformInfo returns information about all registered platforms
func (pm *PlatformManager) GetAllPlatformInfo() map[string]PlatformInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	info := make(map[string]PlatformInfo)
	for name, integration := range pm.integrations {
		info[name] = integration.GetPlatformInfo()
	}
	return info
}

// SyncPlatform syncs bookmarks from a specific platform
func (pm *PlatformManager) SyncPlatform(platformName string, limit int) (*BookmarkProcessingResult, error) {
	pm.mutex.Lock()
	integration, exists := pm.integrations[platformName]
	config := pm.configs[platformName]
	pm.mutex.Unlock()
	
	if !exists {
		return nil, fmt.Errorf("platform %s not found", platformName)
	}
	
	if config.Status != StatusActive {
		return nil, fmt.Errorf("platform %s is not active (status: %s)", platformName, config.Status)
	}
	
	// Update sync status
	pm.updateSyncStatus(platformName, SyncStatus{
		PlatformName:   platformName,
		Status:        StatusSyncing,
		LastSyncAt:    config.LastSyncAt,
		ItemsProcessed: 0,
	})
	
	startTime := time.Now()
	
	// Get bookmarks from platform
	bookmarks, err := integration.GetBookmarks(limit)
	if err != nil {
		pm.handleSyncError(platformName, err)
		return nil, fmt.Errorf("failed to get bookmarks from %s: %w", platformName, err)
	}
	
	// Process bookmarks
	result := &BookmarkProcessingResult{
		ProcessedCount:   len(bookmarks),
		Categories:       make(map[string]int),
		BookmarkIDs:      make([]string, 0, len(bookmarks)),
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}
	
	for _, bookmark := range bookmarks {
		// Categorize bookmark
		categories := integration.EstimateCategories(bookmark)
		if len(categories) > 0 {
			result.CategorizedCount++
			bestCategory := categories[0] // Assume first is highest confidence
			result.Categories[bestCategory.Category]++
		}
		
		// Generate suggested actions
		actions := integration.GetSuggestedActions(bookmark)
		result.ActionsGenerated += len(actions)
		
		result.BookmarkIDs = append(result.BookmarkIDs, bookmark.ID)
		result.SuccessCount++
	}
	
	// Update configuration
	pm.mutex.Lock()
	config.LastSyncAt = &startTime
	config.ErrorCount = 0
	config.LastError = ""
	config.Status = StatusActive
	pm.configs[platformName] = config
	pm.mutex.Unlock()
	
	// Update sync status
	pm.updateSyncStatus(platformName, SyncStatus{
		PlatformName:    platformName,
		Status:         StatusIdle,
		LastSyncAt:     &startTime,
		NextSyncAt:     calculateNextSync(startTime, config.SyncInterval),
		ItemsProcessed: len(bookmarks),
		ProgressPercent: 100,
	})
	
	return result, nil
}

// SyncAllPlatforms syncs bookmarks from all active platforms
func (pm *PlatformManager) SyncAllPlatforms(limit int) (map[string]*BookmarkProcessingResult, error) {
	results := make(map[string]*BookmarkProcessingResult)
	errors := make([]string, 0)
	
	platforms := pm.GetAvailablePlatforms()
	
	// Use goroutines for concurrent syncing
	resultsChan := make(chan struct {
		platform string
		result   *BookmarkProcessingResult
		err      error
	}, len(platforms))
	
	for _, platformName := range platforms {
		go func(platform string) {
			result, err := pm.SyncPlatform(platform, limit)
			resultsChan <- struct {
				platform string
				result   *BookmarkProcessingResult
				err      error
			}{platform, result, err}
		}(platformName)
	}
	
	// Collect results
	for i := 0; i < len(platforms); i++ {
		res := <-resultsChan
		if res.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", res.platform, res.err))
		} else {
			results[res.platform] = res.result
		}
	}
	
	if len(errors) > 0 {
		return results, fmt.Errorf("sync errors: %v", errors)
	}
	
	return results, nil
}

// GetPlatformHealth returns health status for a platform
func (pm *PlatformManager) GetPlatformHealth(platformName string) (PlatformHealth, error) {
	pm.mutex.RLock()
	integration, exists := pm.integrations[platformName]
	config, configExists := pm.configs[platformName]
	pm.mutex.RUnlock()
	
	if !exists || !configExists {
		return PlatformHealth{}, fmt.Errorf("platform %s not found", platformName)
	}
	
	health := PlatformHealth{
		PlatformName:    platformName,
		LastCheckAt:     time.Now(),
		Issues:          make([]string, 0),
	}
	
	// Test connection and measure response time
	startTime := time.Now()
	err := integration.TestConnection()
	health.ResponseTimeMs = time.Since(startTime).Milliseconds()
	
	if err != nil {
		health.IsHealthy = false
		health.Issues = append(health.Issues, err.Error())
	} else {
		health.IsHealthy = true
	}
	
	// Calculate error rate from config
	if config.ErrorCount > 0 {
		// Simple error rate calculation - could be more sophisticated
		health.ErrorRate = float64(config.ErrorCount) / 100.0 // Percentage
	}
	
	// Check rate limit status
	info := integration.GetPlatformInfo()
	if hourlyLimit, exists := info.RateLimits["hourly"]; exists {
		health.RemainingRequests = hourlyLimit - config.ErrorCount // Simplified
		if health.RemainingRequests < hourlyLimit/10 { // Less than 10% remaining
			health.RateLimitStatus = "warning"
			health.Issues = append(health.Issues, "Approaching rate limit")
		} else if health.RemainingRequests <= 0 {
			health.RateLimitStatus = "exceeded"
			health.Issues = append(health.Issues, "Rate limit exceeded")
		} else {
			health.RateLimitStatus = "ok"
		}
	}
	
	return health, nil
}

// GetAllPlatformHealth returns health status for all platforms
func (pm *PlatformManager) GetAllPlatformHealth() map[string]PlatformHealth {
	platforms := pm.GetAvailablePlatforms()
	health := make(map[string]PlatformHealth)
	
	for _, platformName := range platforms {
		if h, err := pm.GetPlatformHealth(platformName); err == nil {
			health[platformName] = h
		}
	}
	
	return health
}

// GetSyncStatus returns the current sync status for a platform
func (pm *PlatformManager) GetSyncStatus(platformName string) (SyncStatus, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	status, exists := pm.syncStatus[platformName]
	if !exists {
		return SyncStatus{}, fmt.Errorf("platform %s not found", platformName)
	}
	
	return status, nil
}

// GetAllSyncStatus returns sync status for all platforms
func (pm *PlatformManager) GetAllSyncStatus() map[string]SyncStatus {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	status := make(map[string]SyncStatus)
	for name, s := range pm.syncStatus {
		status[name] = s
	}
	return status
}

// UpdatePlatformConfig updates the configuration for a platform
func (pm *PlatformManager) UpdatePlatformConfig(platformName string, config PlatformConfig) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	if _, exists := pm.integrations[platformName]; !exists {
		return fmt.Errorf("platform %s not found", platformName)
	}
	
	pm.configs[platformName] = config
	return nil
}

// DisablePlatform disables a platform integration
func (pm *PlatformManager) DisablePlatform(platformName string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	config, exists := pm.configs[platformName]
	if !exists {
		return fmt.Errorf("platform %s not found", platformName)
	}
	
	config.Enabled = false
	config.Status = StatusInactive
	pm.configs[platformName] = config
	
	return nil
}

// EnablePlatform enables a platform integration
func (pm *PlatformManager) EnablePlatform(platformName string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	integration, integrationExists := pm.integrations[platformName]
	config, configExists := pm.configs[platformName]
	
	if !integrationExists || !configExists {
		return fmt.Errorf("platform %s not found", platformName)
	}
	
	// Test connection before enabling
	if err := integration.TestConnection(); err != nil {
		return fmt.Errorf("cannot enable platform %s: connection test failed: %w", platformName, err)
	}
	
	config.Enabled = true
	config.Status = StatusActive
	config.ErrorCount = 0
	config.LastError = ""
	pm.configs[platformName] = config
	
	return nil
}

// GetIntegrationMetrics returns metrics for a platform over a time period
func (pm *PlatformManager) GetIntegrationMetrics(platformName string, periodStart, periodEnd time.Time) (IntegrationMetrics, error) {
	pm.mutex.RLock()
	config, exists := pm.configs[platformName]
	pm.mutex.RUnlock()
	
	if !exists {
		return IntegrationMetrics{}, fmt.Errorf("platform %s not found", platformName)
	}
	
	// This would typically query the database for actual metrics
	// For now, return mock metrics based on config
	metrics := IntegrationMetrics{
		PlatformName:       platformName,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		TotalBookmarks:     100,  // Would be queried from database
		NewBookmarks:       25,   // Would be calculated from period
		ProcessedBookmarks: 95,   // Would be actual processing count
		FailedBookmarks:    config.ErrorCount,
		CategoryBreakdown:  make(map[string]int),
		AverageProcessingMs: 150, // Would be calculated from processing logs
		ErrorRate:          float64(config.ErrorCount) / 100.0,
	}
	
	// Mock category breakdown
	metrics.CategoryBreakdown["Programming"] = 30
	metrics.CategoryBreakdown["News"] = 25
	metrics.CategoryBreakdown["Entertainment"] = 20
	metrics.CategoryBreakdown["Recipes"] = 15
	metrics.CategoryBreakdown["Other"] = 10
	
	return metrics, nil
}

// Helper functions

func (pm *PlatformManager) updateSyncStatus(platformName string, status SyncStatus) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.syncStatus[platformName] = status
}

func (pm *PlatformManager) handleSyncError(platformName string, err error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	config := pm.configs[platformName]
	config.ErrorCount++
	config.LastError = err.Error()
	
	// Update status based on error count
	if config.ErrorCount >= 3 {
		config.Status = StatusError
	}
	
	pm.configs[platformName] = config
	
	// Update sync status
	pm.syncStatus[platformName] = SyncStatus{
		PlatformName: platformName,
		Status:       StatusIdle,
		ErrorMessage: err.Error(),
	}
}

func calculateNextSync(lastSync time.Time, interval time.Duration) *time.Time {
	nextSync := lastSync.Add(interval)
	return &nextSync
}

// Factory function to create platform manager with common integrations
func NewPlatformManagerWithDefaults() *PlatformManager {
	manager := NewPlatformManager()
	
	// Register Reddit integration
	redditConfig := RedditConfig{
		UserAgent: "BookmarkIntelligenceHub/1.0",
		RateLimit: 1000,
	}
	redditIntegration := NewRedditIntegration(redditConfig)
	manager.RegisterPlatform("reddit", redditIntegration, PlatformConfig{
		PlatformName:  "reddit",
		Enabled:       false, // Disabled by default until configured
		SyncInterval:  30 * time.Minute,
		Status:        StatusInactive,
	})
	
	// Register Twitter integration
	twitterConfig := TwitterConfig{
		UserAgent:       "Mozilla/5.0 (compatible; BookmarkIntelligenceBot/1.0)",
		RateLimit:       300,
		FallbackEnabled: true,
	}
	twitterIntegration := NewTwitterIntegration(twitterConfig)
	manager.RegisterPlatform("twitter", twitterIntegration, PlatformConfig{
		PlatformName:  "twitter",
		Enabled:       false, // Disabled by default until configured
		SyncInterval:  60 * time.Minute,
		Status:        StatusInactive,
	})
	
	return manager
}