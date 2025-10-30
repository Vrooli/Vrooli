package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"app-monitor-api/logger"
	"app-monitor-api/repository"
)

// NewAppService creates a new service instance with default dependencies
func NewAppService(repo repository.AppRepository) *AppService {
	return NewAppServiceWithOptions(repo, nil, nil)
}

// NewAppServiceWithOptions creates a new service instance with custom dependencies for testing
func NewAppServiceWithOptions(repo repository.AppRepository, httpClient HTTPClient, timeProvider TimeProvider) *AppService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if timeProvider == nil {
		timeProvider = time.Now
	}

	repoRoot, _ := findRepoRoot()
	return &AppService{
		repo:          repo,
		httpClient:    httpClient,
		timeNow:       timeProvider,
		cache:         &orchestratorCache{},
		viewStats:     make(map[string]*viewStatsEntry),
		issueCache:    make(map[string]*issueCacheEntry),
		issueCacheTTL: issueTrackerCacheTTL,
		repoRoot:      repoRoot,
	}
}

// =============================================================================
// Core CRUD Operations
// =============================================================================

// GetAppsSummary returns a fast-loading list of scenarios while background hydration runs
func (s *AppService) GetAppsSummary(ctx context.Context) ([]repository.App, error) {
	summaries, err := s.fetchScenarioSummaries(ctx)
	if err != nil {
		return nil, err
	}

	clone := make([]repository.App, len(summaries))
	copy(clone, summaries)

	s.cache.mu.Lock()
	s.cache.data = clone
	s.cache.timestamp = s.timeNow()
	s.cache.isPartial = true
	shouldHydrate := !s.cache.loading
	if shouldHydrate {
		s.cache.loading = true
	}
	s.cache.mu.Unlock()

	if shouldHydrate {
		s.hydrateOrchestratorInBackground("background hydration failed")
	}

	return summaries, nil
}

// GetApps retrieves apps, preferring orchestrator data with database fallback
func (s *AppService) GetApps(ctx context.Context) ([]repository.App, error) {
	// Try to get fresh data from orchestrator first
	apps, err := s.GetAppsFromOrchestrator(ctx)
	if err == nil {
		return apps, nil
	}

	// Log orchestrator error but continue with database fallback
	logger.Warn("failed to get apps from orchestrator", err)

	// Fall back to database if orchestrator fails
	if s.hasRepo() {
		fallbackApps, repoErr := s.repo.GetApps(ctx)
		if repoErr == nil {
			for i := range fallbackApps {
				markFallbackApp(&fallbackApps[i])
			}
			return fallbackApps, nil
		}

		// If repository lookup fails, prefer bubbling the orchestrator error chain
		if err == nil {
			return nil, repoErr
		}
	}

	// If no database either, return empty list
	return []repository.App{}, nil
}

// GetApp retrieves a single app by ID
func (s *AppService) GetApp(ctx context.Context, id string) (*repository.App, error) {
	var lastErr error

	if cached, isPartial := s.getAppFromCache(id, true); cached != nil {
		if isPartial {
			s.hydrateOrchestratorCacheAsync()
		}
		return cached, nil
	}

	apps, err := s.GetAppsFromOrchestrator(ctx)
	if err == nil {
		for i := range apps {
			candidate := apps[i]
			if candidate.ID == id || candidate.Name == id || candidate.ScenarioName == id {
				result := candidate
				return &result, nil
			}
		}
	} else {
		lastErr = err
	}

	if s.hasRepo() {
		app, repoErr := s.repo.GetApp(ctx, id)
		if repoErr == nil {
			markFallbackApp(app)
			return app, nil
		}

		if lastErr == nil {
			lastErr = repoErr
		}
	}

	if summaries, summaryErr := s.fetchScenarioSummaries(ctx); summaryErr == nil {
		for i := range summaries {
			candidate := summaries[i]
			if candidate.ID == id || candidate.Name == id || candidate.ScenarioName == id {
				result := candidate
				return &result, nil
			}
		}
	} else if lastErr == nil {
		lastErr = summaryErr
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, fmt.Errorf("app not found: %s", id)
}

func markFallbackApp(app *repository.App) {
	if app == nil {
		return
	}

	app.IsPartial = true

	trimmedStatus := normalizeLower(app.Status)
	if trimmedStatus == "" {
		trimmedStatus = "unknown"
	}
	app.Status = trimmedStatus

	if strings.TrimSpace(app.HealthStatus) == "" {
		app.HealthStatus = trimmedStatus
	}
}

// UpdateAppStatus updates the status of an app
func (s *AppService) UpdateAppStatus(ctx context.Context, id string, status string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	return s.repo.UpdateAppStatus(ctx, id, status)
}

// =============================================================================
// Lifecycle Management
// =============================================================================

// StartApp starts an application using vrooli commands
func (s *AppService) StartApp(ctx context.Context, appName string) error {
	// Add timeout to prevent hanging (60s for start as it can take time)
	_, err := executeVrooliCommand(ctx, 60*time.Second, "scenario", "run", appName)
	if err != nil {
		return fmt.Errorf("failed to start app %s: %w", appName, err)
	}

	// Update status in database if available
	if s.hasRepo() {
		s.repo.UpdateAppStatus(ctx, appName, "running")
	}

	s.invalidateCache()

	return nil
}

// StopApp stops an application using vrooli commands
func (s *AppService) StopApp(ctx context.Context, appName string) error {
	// Add timeout to prevent hanging (20s for stop to allow graceful shutdown)
	_, err := executeVrooliCommand(ctx, 20*time.Second, "scenario", "stop", appName)
	if err != nil {
		return fmt.Errorf("failed to stop app %s: %w", appName, err)
	}

	// Update status in database if available
	if s.hasRepo() {
		s.repo.UpdateAppStatus(ctx, appName, "stopped")
	}

	s.invalidateCache()

	return nil
}

// RestartApp restarts an application using vrooli commands
func (s *AppService) RestartApp(ctx context.Context, appName string) error {
	// Restart may take longer due to stop + start sequencing
	_, err := executeVrooliCommand(ctx, 90*time.Second, "scenario", "restart", appName)
	if err != nil {
		return fmt.Errorf("failed to restart app %s: %w", appName, err)
	}

	if s.hasRepo() {
		s.repo.UpdateAppStatus(ctx, appName, "running")
	}

	s.invalidateCache()

	return nil
}

// =============================================================================
// Repository Operations
// =============================================================================

func (s *AppService) RecordAppStatus(ctx context.Context, status *repository.AppStatus) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	return s.repo.CreateAppStatus(ctx, status)
}

// GetAppStatusHistory retrieves historical status for an app
func (s *AppService) GetAppStatusHistory(ctx context.Context, appID string, hours int) ([]repository.AppStatus, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	return s.repo.GetAppStatusHistory(ctx, appID, hours)
}

// =============================================================================
// View Tracking
// =============================================================================

// RecordAppView tracks preview activity for an app and refreshes cached stats
func (s *AppService) RecordAppView(ctx context.Context, identifier string) (*repository.AppViewStats, error) {
	normalized := strings.TrimSpace(identifier)
	if normalized == "" {
		return nil, errors.New("app identifier is required")
	}

	scenarioName := normalized

	if app, err := s.GetApp(ctx, normalized); err == nil && app != nil {
		if candidate := strings.TrimSpace(app.ScenarioName); candidate != "" {
			scenarioName = candidate
		} else if candidate := strings.TrimSpace(app.ID); candidate != "" {
			scenarioName = candidate
		}
	}

	var stats *repository.AppViewStats

	if s.hasRepo() {
		persisted, err := s.repo.RecordAppView(ctx, scenarioName)
		if err != nil {
			logger.Warn(fmt.Sprintf("failed to persist view stats for %s", scenarioName), err)
			persisted = nil
		}
		stats = persisted
	}

	if stats == nil {
		stats = s.recordAppViewInMemory(identifier, scenarioName)
		if stats == nil {
			now := s.timeNow().UTC()
			stats = &repository.AppViewStats{
				ScenarioName: scenarioName,
				ViewCount:    1,
				FirstViewed:  &now,
				LastViewed:   &now,
			}
			s.storeViewStats(stats, identifier)
		}
	}

	s.updateCacheWithViewStats(stats, identifier)

	return stats, nil
}

func (s *AppService) recordAppViewInMemory(identifier, scenarioName string) *repository.AppViewStats {
	keys := []string{}
	if normalizedScenario := normalizeIdentifier(scenarioName); normalizedScenario != "" {
		keys = append(keys, normalizedScenario)
	}
	if alias := normalizeIdentifier(identifier); alias != "" && !containsString(keys, alias) {
		keys = append(keys, alias)
	}

	if len(keys) == 0 {
		return nil
	}

	now := s.timeNow().UTC()

	s.viewStatsMu.Lock()
	var entry *viewStatsEntry
	for _, key := range keys {
		if existing, ok := s.viewStats[key]; ok && existing != nil {
			entry = existing
			break
		}
	}
	if entry == nil {
		entry = &viewStatsEntry{}
	}

	entry.Count++
	if !entry.HasFirst {
		entry.FirstViewed = now
		entry.HasFirst = true
	}
	entry.LastViewed = now
	entry.HasLast = true

	for _, key := range keys {
		s.viewStats[key] = entry
	}

	count := entry.Count
	firstTime := entry.FirstViewed
	hasFirst := entry.HasFirst
	lastTime := entry.LastViewed
	hasLast := entry.HasLast
	s.viewStatsMu.Unlock()

	var firstPtr *time.Time
	if hasFirst {
		copyTime := firstTime
		firstPtr = &copyTime
	}

	var lastPtr *time.Time
	if hasLast {
		copyTime := lastTime
		lastPtr = &copyTime
	}

	return &repository.AppViewStats{
		ScenarioName: scenarioName,
		ViewCount:    count,
		FirstViewed:  firstPtr,
		LastViewed:   lastPtr,
	}
}

func (s *AppService) updateCacheWithViewStats(stats *repository.AppViewStats, aliases ...string) {
	if stats == nil {
		return
	}

	s.storeViewStats(stats, aliases...)

	normalizedKeys := make(map[string]struct{})
	if primary := normalizeIdentifier(stats.ScenarioName); primary != "" {
		normalizedKeys[primary] = struct{}{}
	}
	for _, alias := range aliases {
		if normalized := normalizeIdentifier(alias); normalized != "" {
			normalizedKeys[normalized] = struct{}{}
		}
	}

	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	if len(s.cache.data) == 0 {
		return
	}

	for i := range s.cache.data {
		candidate := &s.cache.data[i]
		keys := []string{
			strings.ToLower(strings.TrimSpace(candidate.ScenarioName)),
			strings.ToLower(strings.TrimSpace(candidate.ID)),
		}
		for _, key := range keys {
			if key == "" {
				continue
			}
			if _, ok := normalizedKeys[key]; ok {
				candidate.ViewCount = stats.ViewCount
				candidate.FirstViewed = stats.FirstViewed
				candidate.LastViewed = stats.LastViewed
				break
			}
		}
	}
}
