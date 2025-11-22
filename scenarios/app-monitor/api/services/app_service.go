package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
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
		repo:               repo,
		httpClient:         httpClient,
		timeNow:            timeProvider,
		cache:              &orchestratorCache{},
		completenessCache:  &completenessCache{data: make(map[string]*CompletenessResponse)},
		viewStats:          make(map[string]*viewStatsEntry),
		issueCache:         make(map[string]*issueCacheEntry),
		issueCacheTTL:      issueTrackerCacheTTL,
		repoRoot:           repoRoot,
		browserlessService: NewBrowserlessService(),
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

	// Trigger Pass 3: Completeness score hydration
	s.hydrateCompletenessInBackground()

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
	var app *repository.App

	if cached, isPartial := s.getAppFromCache(id, true); cached != nil {
		if isPartial {
			s.hydrateOrchestratorCacheAsync()
		}
		app = cached
	} else {
		apps, err := s.GetAppsFromOrchestrator(ctx)
		if err == nil {
			for i := range apps {
				candidate := apps[i]
				if candidate.ID == id || candidate.Name == id || candidate.ScenarioName == id {
					result := candidate
					app = &result
					break
				}
			}
		} else {
			lastErr = err
		}

		if app == nil && s.hasRepo() {
			repoApp, repoErr := s.repo.GetApp(ctx, id)
			if repoErr == nil {
				markFallbackApp(repoApp)
				app = repoApp
			} else if lastErr == nil {
				lastErr = repoErr
			}
		}

		if app == nil {
			if summaries, summaryErr := s.fetchScenarioSummaries(ctx); summaryErr == nil {
				for i := range summaries {
					candidate := summaries[i]
					if candidate.ID == id || candidate.Name == id || candidate.ScenarioName == id {
						result := candidate
						app = &result
						break
					}
				}
			} else if lastErr == nil {
				lastErr = summaryErr
			}
		}

		if app == nil {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("app not found: %s", id)
		}
	}

	// Enrich app with detailed insights (tech stack and dependencies)
	// This is done for individual app lookups to avoid N+1 queries on list views
	if enrichErr := s.enrichAppWithDetailedInsights(ctx, app); enrichErr != nil {
		logger.Warn(fmt.Sprintf("failed to enrich app %s with detailed insights", id), enrichErr)
		// Don't fail the request if enrichment fails - just return app without detailed data
	}

	return app, nil
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

	// Only set HealthStatus if empty AND different from status
	// This avoids UI redundancy when health and status are the same
	trimmedHealth := normalizeLower(app.HealthStatus)
	if trimmedHealth == "" || trimmedHealth == trimmedStatus {
		app.HealthStatus = ""
	} else {
		app.HealthStatus = trimmedHealth
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

// =============================================================================
// Log Fetching
// =============================================================================

func (s *AppService) runScenarioLogsCommand(ctx context.Context, appName string, extraArgs ...string) (string, error) {
	args := []string{"scenario", "logs", appName}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(output)
		lower := strings.ToLower(outStr)
		if strings.Contains(lower, "not found") || strings.Contains(lower, "no such") {
			return outStr, nil
		}
		return "", fmt.Errorf("failed to execute %s: %w (output: %s)", strings.Join(cmd.Args, " "), err, strings.TrimSpace(outStr))
	}

	return string(output), nil
}

// GetAppLogs retrieves lifecycle and background logs for an application.
func (s *AppService) GetAppLogs(ctx context.Context, appName string, logType string) (*AppLogsResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(logType))
	if normalized == "" || normalized == "all" {
		normalized = "both"
	}

	primaryOutput, err := s.runScenarioLogsCommand(ctx, appName)
	if err != nil {
		return nil, err
	}

	trimmedOutput := strings.TrimSpace(primaryOutput)
	lifecycle := make([]string, 0)
	if trimmedOutput != "" {
		lifecycle = strings.Split(trimmedOutput, "\n")
	}
	if len(lifecycle) == 0 {
		lifecycle = []string{fmt.Sprintf("No logs available for scenario '%s'", appName)}
	}

	result := &AppLogsResult{}
	if normalized != "background" {
		result.Lifecycle = lifecycle
	}

	if normalized == "lifecycle" {
		return result, nil
	}

	backgroundCandidates := parseBackgroundLogCandidates(primaryOutput)
	if len(backgroundCandidates) == 0 {
		return result, nil
	}

	backgroundLogs := make([]BackgroundLog, 0, len(backgroundCandidates))
	for _, candidate := range backgroundCandidates {
		stepOutput, stepErr := s.runScenarioLogsCommand(ctx, appName, "--step", candidate.Step)
		if stepErr != nil {
			logger.Warn(fmt.Sprintf("failed to fetch background logs for %s/%s", appName, candidate.Step), stepErr)
			continue
		}

		trimmed := strings.TrimSpace(stepOutput)
		lines := make([]string, 0)
		if trimmed != "" {
			lines = strings.Split(trimmed, "\n")
		}
		if len(lines) == 0 {
			lines = []string{fmt.Sprintf("No logs captured for background step '%s'", candidate.Step)}
		}

		backgroundLogs = append(backgroundLogs, BackgroundLog{
			Step:    candidate.Step,
			Phase:   candidate.Phase,
			Label:   candidate.Label,
			Command: candidate.Command,
			Lines:   lines,
		})
	}

	if len(backgroundLogs) > 0 {
		result.Background = backgroundLogs
	}

	return result, nil
}

// parseBackgroundLogCandidates extracts background log information from lifecycle logs
func parseBackgroundLogCandidates(raw string) []backgroundLogCandidate {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	results := make([]backgroundLogCandidate, 0)

	inSection := false
	lastLabel := ""
	lastAvailable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.Contains(trimmed, "BACKGROUND STEP LOGS AVAILABLE") {
			inSection = true
			lastLabel = ""
			lastAvailable = false
			continue
		}

		if !inSection {
			continue
		}

		if strings.HasPrefix(trimmed, "💡") {
			break
		}

		switch {
		case strings.HasPrefix(trimmed, "✅"):
			lastLabel = trimmed
			lastAvailable = true
			continue
		case strings.HasPrefix(trimmed, "⚠️"), strings.HasPrefix(trimmed, "⚠"):
			lastLabel = trimmed
			lastAvailable = false
			continue
		case strings.HasPrefix(trimmed, "❌"):
			lastLabel = trimmed
			lastAvailable = false
			continue
		case strings.HasPrefix(trimmed, "View:"):
			matches := backgroundViewCommandRegex.FindStringSubmatch(trimmed)
			if len(matches) != 3 {
				continue
			}
			if !lastAvailable {
				continue
			}

			step := strings.TrimSpace(matches[2])
			if step == "" {
				continue
			}

			label, phase := normalizeBackgroundLabel(lastLabel)
			key := step + "|" + phase
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			candidate := backgroundLogCandidate{
				Step:    step,
				Phase:   phase,
				Label:   label,
				Command: fmt.Sprintf("vrooli scenario logs %s --step %s", strings.TrimSpace(matches[1]), step),
			}
			results = append(results, candidate)
		}
	}

	return results
}

func normalizeBackgroundLabel(raw string) (string, string) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "", ""
	}

	iconPrefixes := []string{"✅", "⚠️", "⚠", "❌", "•"}
	for _, prefix := range iconPrefixes {
		clean = strings.TrimPrefix(clean, prefix)
		clean = strings.TrimPrefix(clean, strings.TrimSpace(prefix))
	}
	clean = strings.TrimSpace(clean)

	if idx := strings.Index(clean, " - "); idx > -1 {
		clean = strings.TrimSpace(clean[:idx])
	}

	label := clean
	phase := ""
	if open := strings.LastIndex(clean, "("); open > -1 && strings.HasSuffix(clean, ")") {
		phase = strings.TrimSpace(clean[open+1 : len(clean)-1])
		name := strings.TrimSpace(clean[:open])
		if name != "" {
			label = fmt.Sprintf("%s (%s)", name, phase)
		}
	}

	return label, phase
}
