package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"app-monitor-api/logger"
	"app-monitor-api/repository"
)

// =============================================================================
// Cache Management
// =============================================================================

func (s *AppService) invalidateCache() {
	if s == nil || s.cache == nil {
		return
	}

	s.cache.mu.Lock()
	s.cache.timestamp = time.Time{}
	s.cache.isPartial = true
	s.cache.mu.Unlock()
}

func (s *AppService) getAppFromCache(id string, allowPartial bool) (*repository.App, bool) {
	if s == nil || s.cache == nil {
		return nil, false
	}

	normalizedID := normalizeIdentifier(id)
	if normalizedID == "" {
		return nil, false
	}

	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	if len(s.cache.data) == 0 {
		return nil, false
	}

	age := time.Since(s.cache.timestamp)
	isPartial := s.cache.isPartial

	if !allowPartial {
		if isPartial || age > orchestratorCacheTTL {
			return nil, false
		}
	} else {
		expiry := orchestratorCacheTTL
		if isPartial {
			expiry = partialCacheTTL
		}
		if age > expiry {
			return nil, false
		}
	}

	for i := range s.cache.data {
		candidate := s.cache.data[i]
		identifiers := []string{
			normalizeIdentifier(candidate.ID),
			normalizeIdentifier(candidate.Name),
			normalizeIdentifier(candidate.ScenarioName),
		}

		for _, ident := range identifiers {
			if ident != "" && ident == normalizedID {
				copy := candidate
				return &copy, isPartial
			}
		}
	}

	return nil, false
}

func (s *AppService) hydrateOrchestratorCacheAsync() {
	if s == nil || s.cache == nil {
		return
	}

	s.cache.mu.RLock()
	isLoading := s.cache.loading
	partial := s.cache.isPartial
	age := time.Since(s.cache.timestamp)
	s.cache.mu.RUnlock()

	if isLoading {
		return
	}

	shouldHydrate := partial || age > orchestratorCacheTTL
	if !shouldHydrate {
		return
	}

	go func() {
		if _, err := s.GetAppsFromOrchestrator(context.Background()); err != nil {
			logger.Warn("background hydration during app lookup failed", err)
		}
	}()
}

// =============================================================================
// Orchestrator Data Fetching
// =============================================================================

// GetAppsFromOrchestrator fetches app status from the vrooli orchestrator with caching
func (s *AppService) GetAppsFromOrchestrator(ctx context.Context) ([]repository.App, error) {
	// Check cache first (cache valid for 90 seconds)
	s.cache.mu.RLock()
	cacheFresh := time.Since(s.cache.timestamp) < orchestratorCacheTTL
	if cacheFresh && len(s.cache.data) > 0 && !s.cache.isPartial {
		cachedData := s.cache.data
		s.cache.mu.RUnlock()
		return cachedData, nil
	}
	s.cache.mu.RUnlock()

	// Prevent concurrent fetches - acquire lock only to set loading flag
	s.cache.mu.Lock()

	// Check cache again after acquiring lock
	cacheFresh = time.Since(s.cache.timestamp) < orchestratorCacheTTL
	if cacheFresh && len(s.cache.data) > 0 && !s.cache.isPartial {
		cachedData := s.cache.data
		s.cache.mu.Unlock()
		return cachedData, nil
	}

	// If already loading, return stale cache or wait briefly
	if s.cache.loading {
		// Return stale cache if available to avoid blocking
		if len(s.cache.data) > 0 {
			cachedData := s.cache.data
			s.cache.mu.Unlock()
			return cachedData, nil
		}
		s.cache.mu.Unlock()
		return nil, fmt.Errorf("orchestrator data is currently being fetched")
	}

	// Mark as loading and release lock before slow operations
	s.cache.loading = true
	s.cache.mu.Unlock()

	// Ensure loading flag is cleared on exit
	defer func() {
		s.cache.mu.Lock()
		s.cache.loading = false
		s.cache.mu.Unlock()
	}()

	// Use vrooli scenario status CLI for runtime data (includes running/stopped status, health, processes, runtime)
	// This is the canonical way to get scenario status without hard-coded ports or API dependencies
	// IMPORTANT: Execute this WITHOUT holding the cache lock to prevent blocking other requests
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		// On error, try to return stale cache if available
		s.cache.mu.RLock()
		if len(s.cache.data) > 0 {
			cachedData := s.cache.data
			s.cache.mu.RUnlock()
			return cachedData, nil
		}
		s.cache.mu.RUnlock()
		return nil, fmt.Errorf("failed to execute vrooli scenario status: %w", err)
	}

	var orchestratorResp OrchestratorResponse
	if err := json.Unmarshal(output, &orchestratorResp); err != nil {
		return nil, fmt.Errorf("failed to parse vrooli scenario status response: %w", err)
	}

	metadataMap, metaErr := s.fetchScenarioMetadata(ctx)
	if metaErr != nil {
		logger.Warn("failed to fetch scenario metadata", metaErr)
		metadataMap = map[string]scenarioMetadata{}
	}

	apps := make([]repository.App, 0, len(orchestratorResp.Scenarios))
	now := s.timeNow().UTC()

	for _, orchApp := range orchestratorResp.Scenarios {
		health := ""
		if orchApp.HealthStatus != nil {
			health = strings.ToLower(strings.TrimSpace(*orchApp.HealthStatus))
		}

		status := strings.ToLower(strings.TrimSpace(orchApp.Status))
		if status == "" {
			status = "unknown"
		}
		if status == "running" && health != "" && health != "healthy" {
			status = health
		}

		displayName := strings.TrimSpace(orchApp.DisplayName)
		if displayName == "" {
			displayName = orchApp.Name
		}

		meta := metadataMap[orchApp.Name]
		description := strings.TrimSpace(orchApp.Description)
		if description == "" {
			description = strings.TrimSpace(meta.Description)
		}

		tags := uniqueStrings(append(append([]string{}, orchApp.Tags...), meta.Tags...))
		runtime := strings.TrimSpace(orchApp.Runtime)

		portMappings := make(map[string]interface{}, len(orchApp.Ports))
		for name, port := range orchApp.Ports {
			portMappings[name] = port
		}

		app := repository.App{
			ID:           orchApp.Name,
			Name:         displayName,
			ScenarioName: orchApp.Name,
			Status:       status,
			Environment:  make(map[string]interface{}),
			Config:       make(map[string]interface{}),
			Description:  description,
			Tags:         tags,
			Runtime:      runtime,
			Type:         "scenario",
			HealthStatus: health,
		}

		if meta.Path != "" {
			app.Path = meta.Path
		}

		var startedAt time.Time
		if startValue := strings.TrimSpace(orchApp.StartedAt); startValue != "" && startValue != "never" {
			if parsed, err := time.Parse(time.RFC3339, startValue); err == nil {
				startedAt = parsed.UTC()
			}
		}

		if !startedAt.IsZero() {
			app.CreatedAt = startedAt
			app.UpdatedAt = now
			app.Uptime = humanizeDuration(now.Sub(startedAt))
		} else {
			app.CreatedAt = now
			app.UpdatedAt = now
		}

		if app.Uptime == "" {
			if dur, err := time.ParseDuration(strings.ToLower(runtime)); err == nil && dur > 0 {
				app.Uptime = humanizeDuration(dur)
			} else if runtime != "" && strings.ToLower(runtime) != "n/a" {
				app.Uptime = runtime
			} else {
				app.Uptime = "N/A"
			}
		}

		if app.HealthStatus == "" {
			app.HealthStatus = status
		}

		if orchApp.Processes > 0 {
			app.Config["process_count"] = orchApp.Processes
		}

		if len(portMappings) == 0 && len(meta.Ports) > 0 {
			portMappings = convertPortsToMap(meta.Ports)
		}

		applyPortConfig(&app, portMappings)

		apps = append(apps, app)
	}

	s.mergeViewStats(ctx, apps)

	// Acquire lock only to update cache with fresh data
	s.cache.mu.Lock()
	s.cache.data = apps
	s.cache.timestamp = s.timeNow()
	s.cache.isPartial = false
	// Note: loading flag is cleared by defer above
	s.cache.mu.Unlock()

	return apps, nil
}

// fetchScenarioList retrieves scenario metadata from the Vrooli CLI
func (s *AppService) fetchScenarioList(ctx context.Context) ([]scenarioMetadata, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "list", "--json", "--include-ports")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var resp scenarioListResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	return resp.Scenarios, nil
}

func (s *AppService) fetchScenarioMetadata(ctx context.Context) (map[string]scenarioMetadata, error) {
	scenarios, err := s.fetchScenarioList(ctx)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]scenarioMetadata, len(scenarios))
	for _, scenario := range scenarios {
		metadata[scenario.Name] = scenario
	}

	return metadata, nil
}

func (s *AppService) fetchScenarioSummaries(ctx context.Context) ([]repository.App, error) {
	scenarios, err := s.fetchScenarioList(ctx)
	if err != nil {
		return nil, err
	}

	now := s.timeNow().UTC()
	apps := make([]repository.App, 0, len(scenarios))

	for _, scenario := range scenarios {
		status := strings.ToLower(strings.TrimSpace(scenario.Status))
		if status == "" {
			status = "unknown"
		}

		description := strings.TrimSpace(scenario.Description)
		tags := uniqueStrings(scenario.Tags)
		portMappings := convertPortsToMap(scenario.Ports)

		app := repository.App{
			ID:           scenario.Name,
			Name:         scenario.Name,
			ScenarioName: scenario.Name,
			Path:         scenario.Path,
			CreatedAt:    now,
			UpdatedAt:    now,
			Status:       status,
			Environment:  make(map[string]interface{}),
			Config:       make(map[string]interface{}),
			Description:  description,
			Tags:         tags,
			Uptime:       "N/A",
			Type:         "scenario",
			HealthStatus: status,
			IsPartial:    true,
		}

		if scenario.Version != "" {
			app.Config["version"] = scenario.Version
		}

		applyPortConfig(&app, portMappings)

		apps = append(apps, app)
	}

	s.mergeViewStats(ctx, apps)

	return apps, nil
}

// =============================================================================
// View Statistics Management
// =============================================================================

func (s *AppService) storeViewStats(stats *repository.AppViewStats, aliases ...string) {
	if stats == nil {
		return
	}

	keys := []string{}
	if primary := normalizeIdentifier(stats.ScenarioName); primary != "" {
		keys = append(keys, primary)
	}
	for _, alias := range aliases {
		if normalized := normalizeIdentifier(alias); normalized != "" && !containsString(keys, normalized) {
			keys = append(keys, normalized)
		}
	}

	if len(keys) == 0 {
		return
	}

	s.viewStatsMu.Lock()
	defer s.viewStatsMu.Unlock()

	entry := &viewStatsEntry{
		Count: stats.ViewCount,
	}

	if stats.FirstViewed != nil {
		entry.FirstViewed = *stats.FirstViewed
		entry.HasFirst = true
	}
	if stats.LastViewed != nil {
		entry.LastViewed = *stats.LastViewed
		entry.HasLast = true
	}

	if s.viewStats == nil {
		s.viewStats = make(map[string]*viewStatsEntry)
	}

	for _, key := range keys {
		s.viewStats[key] = entry
	}
}

func (s *AppService) mergeInMemoryViewStats(apps []repository.App) {
	s.viewStatsMu.RLock()
	defer s.viewStatsMu.RUnlock()

	if len(s.viewStats) == 0 {
		return
	}

	for i := range apps {
		candidates := []string{
			normalizeIdentifier(apps[i].ScenarioName),
			normalizeIdentifier(apps[i].ID),
		}

		var entry *viewStatsEntry
		for _, key := range candidates {
			if key == "" {
				continue
			}
			if candidate, ok := s.viewStats[key]; ok && candidate != nil {
				entry = candidate
				break
			}
		}

		if entry == nil {
			continue
		}

		apps[i].ViewCount = entry.Count
		if entry.HasFirst {
			first := entry.FirstViewed
			apps[i].FirstViewed = &first
		}
		if entry.HasLast {
			last := entry.LastViewed
			apps[i].LastViewed = &last
		}
	}
}

func (s *AppService) cacheRepoViewStats(stats map[string]repository.AppViewStats) {
	if len(stats) == 0 {
		return
	}

	for _, record := range stats {
		rec := record
		s.storeViewStats(&rec)
	}
}

func (s *AppService) mergeViewStats(ctx context.Context, apps []repository.App) {
	if len(apps) == 0 {
		s.mergeInMemoryViewStats(apps)
		return
	}

	if s.repo != nil {
		stats, err := s.repo.GetAppViewStats(ctx)
		if err != nil {
			logger.Warn("failed to fetch app view stats", err)
		} else if len(stats) > 0 {
			s.cacheRepoViewStats(stats)

			for i := range apps {
				candidates := []string{
					strings.TrimSpace(apps[i].ScenarioName),
					strings.ToLower(strings.TrimSpace(apps[i].ScenarioName)),
					strings.TrimSpace(apps[i].ID),
					strings.ToLower(strings.TrimSpace(apps[i].ID)),
				}

				var (
					candidate repository.AppViewStats
					found     bool
				)

				for _, key := range candidates {
					if key == "" {
						continue
					}
					if value, ok := stats[key]; ok {
						candidate = value
						found = true
						break
					}
				}

				if !found || candidate.ScenarioName == "" {
					continue
				}

				apps[i].ViewCount = candidate.ViewCount
				apps[i].FirstViewed = candidate.FirstViewed
				apps[i].LastViewed = candidate.LastViewed
			}
		}
	}

	s.mergeInMemoryViewStats(apps)
}
