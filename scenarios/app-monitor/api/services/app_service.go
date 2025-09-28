package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"app-monitor-api/repository"
)

// Cache for orchestrator data to prevent excessive command execution
type orchestratorCache struct {
	data      []repository.App
	timestamp time.Time
	mu        sync.RWMutex
	isPartial bool
	loading   bool
}

type viewStatsEntry struct {
	Count       int64
	FirstViewed time.Time
	HasFirst    bool
	LastViewed  time.Time
	HasLast     bool
}

// AppService handles business logic for application management
type AppService struct {
	repo        repository.AppRepository
	cache       *orchestratorCache
	viewStatsMu sync.RWMutex
	viewStats   map[string]*viewStatsEntry
}

// AppLogsResult captures lifecycle logs and background step logs for a scenario.
type AppLogsResult struct {
	Lifecycle  []string
	Background []BackgroundLog
}

// BackgroundLog describes a single background step log stream.
type BackgroundLog struct {
	Step    string
	Phase   string
	Label   string
	Command string
	Lines   []string
}

type backgroundLogCandidate struct {
	Step    string
	Phase   string
	Label   string
	Command string
}

var backgroundViewCommandRegex = regexp.MustCompile(`^View:\s+vrooli\s+scenario\s+logs\s+(\S+)\s+--step\s+([^\s]+)`)

// NewAppService creates a new app service
func NewAppService(repo repository.AppRepository) *AppService {
	return &AppService{
		repo:      repo,
		cache:     &orchestratorCache{},
		viewStats: make(map[string]*viewStatsEntry),
	}
}

func (s *AppService) invalidateCache() {
	if s == nil || s.cache == nil {
		return
	}

	s.cache.mu.Lock()
	s.cache.timestamp = time.Time{}
	s.cache.isPartial = true
	s.cache.mu.Unlock()
}

// OrchestratorResponse represents the response from vrooli scenario status --json
type OrchestratorResponse struct {
	Success bool `json:"success"`
	Summary struct {
		TotalScenarios int `json:"total_scenarios"`
		Running        int `json:"running"`
		Stopped        int `json:"stopped"`
	} `json:"summary"`
	Scenarios []OrchestratorApp `json:"scenarios"`
}

// OrchestratorApp represents an app from the orchestrator
type OrchestratorApp struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	HealthStatus *string        `json:"health_status"`
	Ports        map[string]int `json:"ports"`
	Processes    int            `json:"processes"`
	Runtime      string         `json:"runtime"`
	StartedAt    string         `json:"started_at,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
}

// scenarioListResponse represents the output from `vrooli scenario list --json`
type scenarioListResponse struct {
	Success   bool               `json:"success"`
	Scenarios []scenarioMetadata `json:"scenarios"`
}

// scenarioMetadata captures static scenario details such as description and filesystem path
type scenarioMetadata struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Path        string         `json:"path"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
	Tags        []string       `json:"tags"`
	Ports       []scenarioPort `json:"ports"`
}

type scenarioPort struct {
	Key  string      `json:"key"`
	Step string      `json:"step"`
	Port interface{} `json:"port"`
}

// GetAppsFromOrchestrator fetches app status from the vrooli orchestrator with caching
func (s *AppService) GetAppsFromOrchestrator(ctx context.Context) ([]repository.App, error) {
	// Check cache first (cache valid for 15 seconds)
	s.cache.mu.RLock()
	cacheFresh := time.Since(s.cache.timestamp) < 15*time.Second
	if cacheFresh && len(s.cache.data) > 0 && !s.cache.isPartial {
		cachedData := s.cache.data
		s.cache.mu.RUnlock()
		return cachedData, nil
	}
	s.cache.mu.RUnlock()

	// Prevent concurrent fetches using mutex (write lock as coarse mutex)
	s.cache.mu.Lock()

	// Check cache again after acquiring lock
	cacheFresh = time.Since(s.cache.timestamp) < 15*time.Second
	if cacheFresh && len(s.cache.data) > 0 && !s.cache.isPartial {
		cachedData := s.cache.data
		s.cache.mu.Unlock()
		return cachedData, nil
	}

	// Note that we're actively fetching to prevent duplicate hydrations
	s.cache.loading = true

	// Keep the lock while fetching to prevent concurrent fetches
	defer s.cache.mu.Unlock()

	// Execute `vrooli scenario status --json`
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		s.cache.loading = false
		if len(s.cache.data) > 0 {
			return s.cache.data, nil
		}
		return nil, fmt.Errorf("failed to execute vrooli scenario status: %w", err)
	}

	var orchestratorResp OrchestratorResponse
	if err := json.Unmarshal(output, &orchestratorResp); err != nil {
		s.cache.loading = false
		return nil, fmt.Errorf("failed to parse orchestrator response: %w", err)
	}

	metadataMap, metaErr := s.fetchScenarioMetadata(ctx)
	if metaErr != nil {
		fmt.Printf("Warning: Failed to fetch scenario metadata: %v\n", metaErr)
		metadataMap = map[string]scenarioMetadata{}
	}

	apps := make([]repository.App, 0, len(orchestratorResp.Scenarios))
	now := time.Now().UTC()

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

	s.cache.data = apps
	s.cache.timestamp = time.Now()
	s.cache.isPartial = false
	s.cache.loading = false

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

	now := time.Now().UTC()
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

// humanizeDuration converts a duration into a terse "1h 4m" style string
func humanizeDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := make([]string, 0, 3)
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if hours == 0 && minutes == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	if len(parts) == 0 {
		parts = append(parts, "0s")
	}

	return strings.Join(parts, " ")
}

// uniqueStrings returns a deduplicated slice while preserving order
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}

func convertPortsToMap(entries []scenarioPort) map[string]interface{} {
	if len(entries) == 0 {
		return nil
	}

	ports := make(map[string]interface{}, len(entries))
	for _, entry := range entries {
		key := strings.TrimSpace(entry.Key)
		if key == "" || entry.Port == nil {
			continue
		}

		ports[key] = normalizePortValue(entry.Port)
	}

	if len(ports) == 0 {
		return nil
	}

	return ports
}

func normalizePortValue(value interface{}) interface{} {
	switch v := value.(type) {
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
		return v.String()
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return trimmed
		}
		if i, err := strconv.Atoi(trimmed); err == nil {
			return i
		}
		return trimmed
	default:
		return v
	}
}

func derivePrimaryPort(ports map[string]interface{}) (string, interface{}) {
	if len(ports) == 0 {
		return "", nil
	}

	if value, ok := ports["UI_PORT"]; ok {
		return "UI_PORT", value
	}
	if value, ok := ports["API_PORT"]; ok {
		return "API_PORT", value
	}

	keys := make([]string, 0, len(ports))
	for key := range ports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	first := keys[0]
	return first, ports[first]
}

func applyPortConfig(app *repository.App, ports map[string]interface{}) {
	if len(ports) == 0 {
		return
	}

	app.PortMappings = make(map[string]interface{}, len(ports))
	for key, value := range ports {
		app.PortMappings[key] = value
	}

	if app.Config == nil {
		app.Config = make(map[string]interface{})
	}
	app.Config["ports"] = app.PortMappings

	primaryLabel, primaryValue := derivePrimaryPort(app.PortMappings)
	if primaryLabel != "" {
		app.Config["primary_port_label"] = primaryLabel
		app.Config["primary_port"] = fmt.Sprintf("%v", primaryValue)
	}

	if apiPort, ok := app.PortMappings["API_PORT"]; ok {
		app.Config["api_port"] = apiPort
	}
}

func normalizeIdentifier(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

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

	entry.Count = stats.ViewCount
	if stats.FirstViewed != nil {
		entry.FirstViewed = stats.FirstViewed.UTC()
		entry.HasFirst = true
	}
	if stats.LastViewed != nil {
		entry.LastViewed = stats.LastViewed.UTC()
		entry.HasLast = true
	}

	for _, key := range keys {
		s.viewStats[key] = entry
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

	now := time.Now().UTC()

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

func (s *AppService) mergeViewStats(ctx context.Context, apps []repository.App) {
	if len(apps) == 0 {
		s.mergeInMemoryViewStats(apps)
		return
	}

	if s.repo != nil {
		stats, err := s.repo.GetAppViewStats(ctx)
		if err != nil {
			fmt.Printf("Warning: failed to fetch app view stats: %v\n", err)
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
	s.cache.timestamp = time.Now()
	s.cache.isPartial = true
	shouldHydrate := !s.cache.loading
	if shouldHydrate {
		s.cache.loading = true
	}
	s.cache.mu.Unlock()

	if shouldHydrate {
		go func() {
			if _, err := s.GetAppsFromOrchestrator(context.Background()); err != nil {
				fmt.Printf("Warning: background hydration failed: %v\n", err)
			}

			s.cache.mu.Lock()
			s.cache.loading = false
			s.cache.mu.Unlock()
		}()
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
	fmt.Printf("Warning: Failed to get apps from orchestrator: %v\n", err)

	// Fall back to database if orchestrator fails
	if s.repo != nil {
		return s.repo.GetApps(ctx)
	}

	// If no database either, return empty list
	return []repository.App{}, nil
}

// GetApp retrieves a single app by ID
func (s *AppService) GetApp(ctx context.Context, id string) (*repository.App, error) {
	var lastErr error

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

	if s.repo != nil {
		app, repoErr := s.repo.GetApp(ctx, id)
		if repoErr == nil {
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

// UpdateAppStatus updates the status of an app
func (s *AppService) UpdateAppStatus(ctx context.Context, id string, status string) error {
	if s.repo == nil {
		return fmt.Errorf("database not available")
	}
	return s.repo.UpdateAppStatus(ctx, id, status)
}

// StartApp starts an application using vrooli commands
func (s *AppService) StartApp(ctx context.Context, appName string) error {
	// Add timeout to prevent hanging (60s for start as it can take time)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "run", appName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start app %s: %w (output: %s)", appName, err, string(output))
	}

	// Update status in database if available
	if s.repo != nil {
		s.repo.UpdateAppStatus(ctx, appName, "running")
	}

	s.invalidateCache()

	return nil
}

// StopApp stops an application using vrooli commands
func (s *AppService) StopApp(ctx context.Context, appName string) error {
	// Add timeout to prevent hanging (20s for stop to allow graceful shutdown)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "stop", appName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop app %s: %w (output: %s)", appName, err, string(output))
	}

	// Update status in database if available
	if s.repo != nil {
		s.repo.UpdateAppStatus(ctx, appName, "stopped")
	}

	s.invalidateCache()

	return nil
}

// RestartApp restarts an application using vrooli commands
func (s *AppService) RestartApp(ctx context.Context, appName string) error {
	// Restart may take longer due to stop + start sequencing
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "restart", appName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart app %s: %w (output: %s)", appName, err, string(output))
	}

	if s.repo != nil {
		s.repo.UpdateAppStatus(ctx, appName, "running")
	}

	s.invalidateCache()

	return nil
}

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
			fmt.Printf("Warning: failed to fetch background logs for %s/%s: %v\n", appName, candidate.Step, stepErr)
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

// RecordAppStatus records current status metrics for an app
func (s *AppService) RecordAppStatus(ctx context.Context, status *repository.AppStatus) error {
	if s.repo == nil {
		return fmt.Errorf("database not available")
	}
	return s.repo.CreateAppStatus(ctx, status)
}

// GetAppStatusHistory retrieves historical status for an app
func (s *AppService) GetAppStatusHistory(ctx context.Context, appID string, hours int) ([]repository.AppStatus, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("database not available")
	}
	return s.repo.GetAppStatusHistory(ctx, appID, hours)
}

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

	if s.repo != nil {
		persisted, err := s.repo.RecordAppView(ctx, scenarioName)
		if err != nil {
			fmt.Printf("Warning: failed to persist view stats for %s: %v\n", scenarioName, err)
			persisted = nil
		}
		stats = persisted
	}

	if stats == nil {
		stats = s.recordAppViewInMemory(identifier, scenarioName)
		if stats == nil {
			now := time.Now().UTC()
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

// IssueReportRequest captures a request to forward an issue to app-issue-tracker
type IssueReportRequest struct {
	AppID             string                 `json:"-"`
	Message           string                 `json:"message"`
	IncludeScreenshot *bool                  `json:"includeScreenshot"`
	PreviewURL        *string                `json:"previewUrl"`
	AppName           *string                `json:"appName"`
	ScenarioName      *string                `json:"scenarioName"`
	Source            *string                `json:"source"`
	ScreenshotData    *string                `json:"screenshotData"`
	Logs              []string               `json:"logs"`
	LogsTotal         *int                   `json:"logsTotal"`
	LogsCapturedAt    *string                `json:"logsCapturedAt"`
	ConsoleLogs       []IssueConsoleLogEntry `json:"consoleLogs"`
	ConsoleLogsTotal  *int                   `json:"consoleLogsTotal"`
	ConsoleCapturedAt *string                `json:"consoleLogsCapturedAt"`
	NetworkRequests   []IssueNetworkEntry    `json:"networkRequests"`
	NetworkTotal      *int                   `json:"networkRequestsTotal"`
	NetworkCapturedAt *string                `json:"networkCapturedAt"`
}

type IssueConsoleLogEntry struct {
	Timestamp int64  `json:"ts"`
	Level     string `json:"level"`
	Source    string `json:"source"`
	Text      string `json:"text"`
}

type IssueNetworkEntry struct {
	Timestamp  int64  `json:"ts"`
	Kind       string `json:"kind"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	Status     *int   `json:"status,omitempty"`
	OK         *bool  `json:"ok,omitempty"`
	DurationMs *int   `json:"durationMs,omitempty"`
	Error      string `json:"error,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueOrDefault(value *int, fallback int) int {
	if value != nil {
		return *value
	}
	return fallback
}

// IssueReportResult represents the outcome of forwarding an issue report
type IssueReportResult struct {
	IssueID  string
	Message  string
	IssueURL string
}

// ReportAppIssue forwards an issue report to the app-issue-tracker scenario
func (s *AppService) ReportAppIssue(ctx context.Context, req *IssueReportRequest) (*IssueReportResult, error) {
	if req == nil {
		return nil, errors.New("request payload is required")
	}

	appID := strings.TrimSpace(req.AppID)
	if appID == "" {
		return nil, errors.New("app identifier is required")
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, errors.New("issue message is required")
	}

	reportedAt := time.Now().UTC()

	appName := strings.TrimSpace(stringValue(req.AppName))
	scenarioName := strings.TrimSpace(stringValue(req.ScenarioName))

	if app, err := s.GetApp(ctx, appID); err == nil && app != nil {
		if appName == "" {
			appName = app.Name
		}
		if scenarioName == "" {
			scenarioName = app.ScenarioName
		}
	}

	if appName == "" {
		appName = appID
	}
	if scenarioName == "" {
		scenarioName = appID
	}

	previewURL := ""
	if req.PreviewURL != nil {
		previewURL = normalizePreviewURL(*req.PreviewURL)
	}

	screenshotData := strings.TrimSpace(stringValue(req.ScreenshotData))
	if screenshotData != "" {
		if _, err := base64.StdEncoding.DecodeString(screenshotData); err != nil {
			fmt.Printf("Warning: invalid screenshot data provided, ignoring: %v\n", err)
			screenshotData = ""
		}
	}

	const (
		maxReportLogs        = 300
		maxConsoleLogs       = 200
		maxNetworkEntries    = 150
		maxConsoleTextLength = 2000
		maxNetworkURLLength  = 2048
		maxNetworkErrLength  = 1500
		maxRequestIDLength   = 128
	)
	sanitizedLogs := make([]string, 0, len(req.Logs))
	for _, line := range req.Logs {
		trimmed := strings.TrimRight(line, "\r\n")
		sanitizedLogs = append(sanitizedLogs, trimmed)
	}
	if len(sanitizedLogs) > maxReportLogs {
		sanitizedLogs = sanitizedLogs[len(sanitizedLogs)-maxReportLogs:]
	}

	logsTotal := valueOrDefault(req.LogsTotal, len(req.Logs))
	if logsTotal <= 0 {
		logsTotal = len(req.Logs)
	}

	logsCapturedAt := strings.TrimSpace(stringValue(req.LogsCapturedAt))
	consoleCapturedAt := strings.TrimSpace(stringValue(req.ConsoleCapturedAt))
	networkCapturedAt := strings.TrimSpace(stringValue(req.NetworkCapturedAt))

	consoleLogs := sanitizeConsoleLogs(req.ConsoleLogs, maxConsoleLogs, maxConsoleTextLength)
	consoleTotal := valueOrDefault(req.ConsoleLogsTotal, len(req.ConsoleLogs))
	if consoleTotal <= 0 {
		consoleTotal = len(req.ConsoleLogs)
	}

	networkEntries := sanitizeNetworkRequests(req.NetworkRequests, maxNetworkEntries, maxNetworkURLLength, maxNetworkErrLength, maxRequestIDLength)
	networkTotal := valueOrDefault(req.NetworkTotal, len(req.NetworkRequests))
	if networkTotal <= 0 {
		networkTotal = len(req.NetworkRequests)
	}

	title := fmt.Sprintf("[app-monitor] %s", summarizeIssueTitle(message))
	reportSource := strings.TrimSpace(stringValue(req.Source))
	description := buildIssueDescription(
		appName,
		scenarioName,
		previewURL,
		reportSource,
		message,
		screenshotData,
		reportedAt,
		sanitizedLogs,
		logsTotal,
		logsCapturedAt,
		consoleLogs,
		consoleTotal,
		consoleCapturedAt,
		networkEntries,
		networkTotal,
		networkCapturedAt,
	)
	hasScreenshot := screenshotData != ""
	hasConsole := len(consoleLogs) > 0
	hasNetwork := len(networkEntries) > 0
	tags := buildIssueTags(hasScreenshot, hasConsole, hasNetwork)
	environment := buildIssueEnvironment(appID, appName, previewURL, reportSource, reportedAt)

	port, err := s.locateIssueTrackerAPIPort(ctx)
	if err != nil {
		return nil, err
	}

	metadataExtra := map[string]string{}
	if reportSource != "" {
		metadataExtra["report_source"] = reportSource
	}
	if previewURL != "" {
		metadataExtra["preview_url"] = previewURL
	}
	if logsTotal > 0 {
		metadataExtra["logs_total"] = strconv.Itoa(logsTotal)
	}
	if logsCapturedAt != "" {
		metadataExtra["logs_captured_at"] = logsCapturedAt
	}
	if consoleTotal > 0 {
		metadataExtra["console_total"] = strconv.Itoa(consoleTotal)
	}
	if consoleCapturedAt != "" {
		metadataExtra["console_captured_at"] = consoleCapturedAt
	}
	if networkTotal > 0 {
		metadataExtra["network_total"] = strconv.Itoa(networkTotal)
	}
	if networkCapturedAt != "" {
		metadataExtra["network_captured_at"] = networkCapturedAt
	}
	if screenshotData != "" {
		metadataExtra["screenshot_included"] = "true"
	}

	artifacts := make([]map[string]interface{}, 0, 4)
	if len(sanitizedLogs) > 0 {
		artifactContent := strings.Join(sanitizedLogs, "\n")
		artifacts = append(artifacts, map[string]interface{}{
			"name":         "app-monitor-lifecycle.txt",
			"category":     "logs",
			"content":      artifactContent,
			"encoding":     "plain",
			"content_type": "text/plain",
		})
	}
	if len(consoleLogs) > 0 {
		consolePayload, err := json.MarshalIndent(consoleLogs, "", "  ")
		consoleContent := ""
		if err == nil {
			consoleContent = string(consolePayload)
		} else {
			consoleContent = "[]"
		}
		artifacts = append(artifacts, map[string]interface{}{
			"name":         "app-monitor-console.json",
			"category":     "console",
			"content":      consoleContent,
			"encoding":     "plain",
			"content_type": "application/json",
		})
	}
	if len(networkEntries) > 0 {
		networkPayload, err := json.MarshalIndent(networkEntries, "", "  ")
		networkContent := ""
		if err == nil {
			networkContent = string(networkPayload)
		} else {
			networkContent = "[]"
		}
		artifacts = append(artifacts, map[string]interface{}{
			"name":         "app-monitor-network.json",
			"category":     "network",
			"content":      networkContent,
			"encoding":     "plain",
			"content_type": "application/json",
		})
	}
	if screenshotData != "" {
		artifacts = append(artifacts, map[string]interface{}{
			"name":         "app-monitor-screenshot.png",
			"category":     "screenshot",
			"content":      screenshotData,
			"encoding":     "base64",
			"content_type": "image/png",
		})
	}

	if len(metadataExtra) == 0 {
		metadataExtra = nil
	}

	reporterName := "App Monitor"
	if reportSource != "" {
		reporterName = fmt.Sprintf("App Monitor – %s", reportSource)
	}
	reporterEmail := "monitor@vrooli.local"

	payload := map[string]interface{}{
		"title":          title,
		"description":    description,
		"type":           "bug",
		"priority":       "medium",
		"app_id":         scenarioName,
		"tags":           tags,
		"environment":    environment,
		"metadata_extra": metadataExtra,
		"artifacts":      artifacts,
		"reporter_name":  reporterName,
		"reporter_email": reporterEmail,
	}

	result, err := submitIssueToTracker(ctx, port, payload)
	if err != nil {
		return nil, err
	}

	if result != nil && result.IssueID != "" {
		if uiPort, err := resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "UI_PORT"); err == nil && uiPort > 0 {
			u := url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("localhost:%d", uiPort),
				Path:   "/",
			}
			query := u.Query()
			query.Set("issue", result.IssueID)
			u.RawQuery = query.Encode()
			result.IssueURL = u.String()
		} else if err != nil {
			fmt.Printf("Warning: failed to resolve app-issue-tracker UI port: %v\n", err)
		}
	}

	return result, nil
}

func (s *AppService) locateIssueTrackerAPIPort(ctx context.Context) (int, error) {
	if port, err := resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "API_PORT"); err == nil && port > 0 {
		return port, nil
	} else if err != nil {
		fmt.Printf("Warning: failed to resolve app-issue-tracker port via CLI: %v\n", err)
	}

	apps, err := s.GetAppsFromOrchestrator(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect scenarios: %w", err)
	}

	for _, candidate := range apps {
		name := strings.ToLower(strings.TrimSpace(candidate.ScenarioName))
		if name == "" {
			name = strings.ToLower(strings.TrimSpace(candidate.ID))
		}
		if name != "app-issue-tracker" {
			continue
		}

		port := resolvePort(candidate.PortMappings, []string{"api", "api_port", "API", "API_PORT"})
		if port > 0 {
			return port, nil
		}
	}

	return 0, errors.New("app-issue-tracker is not running or no API port was found")
}

func resolveScenarioPortViaCLI(ctx context.Context, scenarioName, portLabel string) (int, error) {
	port, err := executeScenarioPortCommand(ctx, scenarioName, portLabel)
	if err == nil {
		return port, nil
	}

	fallbackPorts, fallbackErr := executeScenarioPortList(ctx, scenarioName)
	if fallbackErr == nil {
		candidate := strings.ToUpper(strings.TrimSpace(portLabel))
		if value, ok := fallbackPorts[candidate]; ok && value > 0 {
			return value, nil
		}
	}

	if fallbackErr != nil {
		return 0, fmt.Errorf("%v; fallback error: %w", err, fallbackErr)
	}

	return 0, err
}

func executeScenarioPortCommand(ctx context.Context, scenarioName, portLabel string) (int, error) {
	if strings.TrimSpace(scenarioName) == "" || strings.TrimSpace(portLabel) == "" {
		return 0, errors.New("scenario and port labels are required")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "port", scenarioName, portLabel)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("vrooli scenario port %s %s failed: %s", scenarioName, portLabel, strings.TrimSpace(string(output)))
	}

	return parsePortValueFromString(strings.TrimSpace(string(output)))
}

func executeScenarioPortList(ctx context.Context, scenarioName string) (map[string]int, error) {
	if strings.TrimSpace(scenarioName) == "" {
		return nil, errors.New("scenario name is required")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "port", scenarioName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario port %s failed: %s", scenarioName, strings.TrimSpace(string(output)))
	}

	ports := make(map[string]int)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		port, err := parsePortValueFromString(value)
		if err != nil {
			continue
		}

		if port > 0 {
			ports[key] = port
		}
	}

	if len(ports) == 0 {
		return nil, errors.New("no port mappings returned")
	}

	return ports, nil
}

func parsePortValueFromString(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errors.New("empty port value")
	}

	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid port value %q", value)
	}

	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("port value out of range: %d", port)
	}

	return port, nil
}

func resolvePort(portMappings map[string]interface{}, preferredKeys []string) int {
	if len(portMappings) == 0 {
		return 0
	}

	for _, key := range preferredKeys {
		for label, value := range portMappings {
			if strings.EqualFold(label, key) {
				if port, ok := parsePortValue(value); ok {
					return port
				}
			}
		}
	}

	for _, value := range portMappings {
		if port, ok := parsePortValue(value); ok {
			return port
		}
	}

	return 0
}

func parsePortValue(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		if v > 0 {
			return v, true
		}
	case int32:
		if v > 0 {
			return int(v), true
		}
	case int64:
		if v > 0 {
			return int(v), true
		}
	case float64:
		port := int(v)
		if port > 0 {
			return port, true
		}
	case json.Number:
		if i, err := v.Int64(); err == nil && i > 0 {
			return int(i), true
		}
		if f, err := v.Float64(); err == nil {
			port := int(f)
			if port > 0 {
				return port, true
			}
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil && parsed > 0 {
			return parsed, true
		}
	}

	return 0, false
}

func normalizePreviewURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}

	return parsed.String()
}

func summarizeIssueTitle(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "Issue reported from App Monitor"
	}

	firstLine := trimmed
	if idx := strings.IndexAny(trimmed, "\n\r"); idx != -1 {
		firstLine = strings.TrimSpace(trimmed[:idx])
	}

	runes := []rune(firstLine)
	if len(runes) > 60 {
		return string(runes[:60]) + "..."
	}

	return firstLine
}

func buildIssueDescription(
	appName, scenarioName, previewURL, source, message, screenshotData string,
	reportedAt time.Time,
	logs []string,
	logsTotal int,
	logsCapturedAt string,
	consoleLogs []IssueConsoleLogEntry,
	consoleTotal int,
	consoleCapturedAt string,
	network []IssueNetworkEntry,
	networkTotal int,
	networkCapturedAt string,
) string {
	var builder strings.Builder
	builder.WriteString("## App Monitor Issue Report\n\n")
	builder.WriteString(fmt.Sprintf("- App Name: %s\n", appName))
	builder.WriteString(fmt.Sprintf("- Scenario Identifier: %s\n", scenarioName))
	if previewURL != "" {
		builder.WriteString(fmt.Sprintf("- Preview URL: %s\n", previewURL))
	}
	if source != "" {
		builder.WriteString(fmt.Sprintf("- Reported By: %s\n", source))
	}
	builder.WriteString(fmt.Sprintf("- Reported At: %s\n", reportedAt.Format(time.RFC3339)))

	builder.WriteString("\n### Reporter Notes\n\n")
	builder.WriteString(message)
	builder.WriteString("\n")

	if len(logs) > 0 || logsTotal > 0 {
		builder.WriteString("\n### Recent Logs\n\n")
		if logsCapturedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, logsCapturedAt); err == nil {
				builder.WriteString(fmt.Sprintf("_Captured at: %s_\n\n", parsed.Format(time.RFC3339)))
			} else {
				builder.WriteString(fmt.Sprintf("_Captured at: %s_\n\n", logsCapturedAt))
			}
		}

		if len(logs) > 0 {
			builder.WriteString("```text\n")
			for _, line := range logs {
				builder.WriteString(line)
				builder.WriteByte('\n')
			}
			builder.WriteString("```\n")

			if logsTotal > len(logs) && logsTotal > 0 {
				builder.WriteString(fmt.Sprintf("\n_Note: showing last %d of %d lines._\n", len(logs), logsTotal))
			}
		} else {
			builder.WriteString("_No log entries were captured for this report._\n")
		}
	}

	if len(consoleLogs) > 0 || consoleTotal > 0 {
		builder.WriteString("\n### Console Logs\n\n")
		if consoleCapturedAt != "" {
			builder.WriteString(fmt.Sprintf("_Captured at: %s_\n\n", parseOrEchoTimestamp(consoleCapturedAt)))
		}

		if len(consoleLogs) > 0 {
			builder.WriteString("```text\n")
			for _, entry := range consoleLogs {
				builder.WriteString(formatConsoleLogEntry(entry))
				builder.WriteByte('\n')
			}
			builder.WriteString("```\n")
			if consoleTotal > len(consoleLogs) && consoleTotal > 0 {
				builder.WriteString(fmt.Sprintf("\n_Note: showing last %d of %d console events._\n", len(consoleLogs), consoleTotal))
			}
		} else {
			builder.WriteString("_No console events were captured for this report._\n")
		}
	}

	if len(network) > 0 || networkTotal > 0 {
		builder.WriteString("\n### Network Requests\n\n")
		if networkCapturedAt != "" {
			builder.WriteString(fmt.Sprintf("_Captured at: %s_\n\n", parseOrEchoTimestamp(networkCapturedAt)))
		}

		if len(network) > 0 {
			builder.WriteString("```text\n")
			for _, entry := range network {
				builder.WriteString(formatNetworkEntry(entry))
				builder.WriteByte('\n')
			}
			builder.WriteString("```\n")
			if networkTotal > len(network) && networkTotal > 0 {
				builder.WriteString(fmt.Sprintf("\n_Note: showing last %d of %d requests._\n", len(network), networkTotal))
			}
		} else {
			builder.WriteString("_No network activity was captured for this report._\n")
		}
	}

	if screenshotData != "" {
		builder.WriteString("\n### Screenshot\n\n")
		builder.WriteString("![Preview Screenshot](data:image/png;base64,")
		builder.WriteString(screenshotData)
		builder.WriteString(")\n")
	}

	return builder.String()
}

func buildIssueTags(hasScreenshot, hasConsole, hasNetwork bool) []string {
	tags := []string{"app-monitor", "preview"}
	if hasScreenshot {
		tags = append(tags, "screenshot")
	}
	if hasConsole {
		tags = append(tags, "console-capture")
	}
	if hasNetwork {
		tags = append(tags, "network-capture")
	}
	return tags
}

func sanitizeConsoleLogs(entries []IssueConsoleLogEntry, maxEntries, maxText int) []IssueConsoleLogEntry {
	if len(entries) == 0 {
		return nil
	}
	sanitized := make([]IssueConsoleLogEntry, 0, len(entries))
	for _, entry := range entries {
		level := strings.ToLower(strings.TrimSpace(entry.Level))
		switch level {
		case "log", "info", "warn", "error", "debug":
			// keep as-is
		default:
			level = "log"
		}

		source := strings.ToLower(strings.TrimSpace(entry.Source))
		if source != "console" && source != "runtime" {
			source = "console"
		}

		text := strings.TrimSpace(entry.Text)
		if text == "" {
			text = "(no message supplied)"
		}

		sanitized = append(sanitized, IssueConsoleLogEntry{
			Timestamp: entry.Timestamp,
			Level:     level,
			Source:    source,
			Text:      truncateString(text, maxText),
		})
	}

	if len(sanitized) > maxEntries {
		sanitized = sanitized[len(sanitized)-maxEntries:]
	}

	return sanitized
}

func sanitizeNetworkRequests(entries []IssueNetworkEntry, maxEntries, maxURLLength, maxErrLength, maxIDLength int) []IssueNetworkEntry {
	if len(entries) == 0 {
		return nil
	}

	sanitized := make([]IssueNetworkEntry, 0, len(entries))
	for _, entry := range entries {
		kind := strings.ToLower(strings.TrimSpace(entry.Kind))
		if kind != "fetch" && kind != "xhr" {
			kind = "fetch"
		}

		method := strings.ToUpper(strings.TrimSpace(entry.Method))
		if method == "" {
			method = "GET"
		}

		urlValue := strings.TrimSpace(entry.URL)
		if urlValue == "" {
			urlValue = "(unknown URL)"
		}
		urlValue = truncateString(urlValue, maxURLLength)

		errorText := truncateString(strings.TrimSpace(entry.Error), maxErrLength)

		var statusPtr *int
		if entry.Status != nil {
			val := *entry.Status
			if val >= 0 {
				statusPtr = intPtr(val)
			}
		}

		var okPtr *bool
		if entry.OK != nil {
			okPtr = boolPtr(*entry.OK)
		}

		var durationPtr *int
		if entry.DurationMs != nil {
			val := *entry.DurationMs
			if val < 0 {
				val = 0
			}
			durationPtr = intPtr(val)
		}

		requestID := truncateString(strings.TrimSpace(entry.RequestID), maxIDLength)

		sanitized = append(sanitized, IssueNetworkEntry{
			Timestamp:  entry.Timestamp,
			Kind:       kind,
			Method:     method,
			URL:        urlValue,
			Status:     statusPtr,
			OK:         okPtr,
			DurationMs: durationPtr,
			Error:      errorText,
			RequestID:  requestID,
		})
	}

	if len(sanitized) > maxEntries {
		sanitized = sanitized[len(sanitized)-maxEntries:]
	}

	return sanitized
}

func parseOrEchoTimestamp(value string) string {
	if value == "" {
		return value
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format(time.RFC3339)
	}
	return value
}

func formatConsoleLogEntry(entry IssueConsoleLogEntry) string {
	ts := formatMillisTimestamp(entry.Timestamp)
	level := strings.ToUpper(entry.Level)
	source := entry.Source
	text := entry.Text
	return fmt.Sprintf("%s [%s/%s] %s", ts, source, level, text)
}

func formatNetworkEntry(entry IssueNetworkEntry) string {
	ts := formatMillisTimestamp(entry.Timestamp)
	status := "pending"
	if entry.Status != nil {
		status = fmt.Sprintf("%d", *entry.Status)
	} else if entry.OK != nil {
		if *entry.OK {
			status = "ok"
		} else {
			status = "error"
		}
	}

	extras := make([]string, 0, 3)
	if entry.DurationMs != nil {
		extras = append(extras, fmt.Sprintf("%dms", *entry.DurationMs))
	}
	if entry.Error != "" {
		extras = append(extras, fmt.Sprintf("error: %s", entry.Error))
	}
	if entry.RequestID != "" {
		extras = append(extras, fmt.Sprintf("id=%s", entry.RequestID))
	}

	tail := ""
	if len(extras) > 0 {
		tail = " (" + strings.Join(extras, ", ") + ")"
	}

	return fmt.Sprintf("%s %s %s -> %s%s", ts, entry.Method, entry.URL, status, tail)
}

func formatMillisTimestamp(value int64) string {
	if value <= 0 {
		return "unknown"
	}
	var t time.Time
	if value > 9e11 {
		t = time.Unix(0, value*int64(time.Millisecond)).UTC()
	} else {
		t = time.Unix(value, 0).UTC()
	}
	return t.Format(time.RFC3339)
}

func truncateString(value string, limit int) string {
	if limit <= 0 || value == "" {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

func intPtr(v int) *int {
	value := v
	return &value
}

func boolPtr(v bool) *bool {
	value := v
	return &value
}

func buildIssueEnvironment(appID, appName, previewURL, source string, reportedAt time.Time) map[string]string {
	environment := map[string]string{
		"app_id":      appID,
		"app_name":    appName,
		"reported_at": reportedAt.Format(time.RFC3339),
	}

	if previewURL != "" {
		environment["preview_url"] = previewURL
	}

	if source != "" {
		environment["source"] = source
	}

	return environment
}

type issueTrackerAPIResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Error   string                 `json:"error"`
	Data    map[string]interface{} `json:"data"`
}

func submitIssueToTracker(ctx context.Context, port int, payload map[string]interface{}) (*IssueReportResult, error) {
	if port <= 0 {
		return nil, errors.New("invalid app-issue-tracker port")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call app-issue-tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("app-issue-tracker returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var trackerResp issueTrackerAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackerResp); err != nil {
		return nil, fmt.Errorf("failed to decode app-issue-tracker response: %w", err)
	}

	if !trackerResp.Success {
		message := strings.TrimSpace(trackerResp.Error)
		if message == "" {
			message = strings.TrimSpace(trackerResp.Message)
		}
		if message == "" {
			message = "app-issue-tracker rejected the issue report"
		}
		return nil, errors.New(message)
	}

	issueID := ""
	if trackerResp.Data != nil {
		if value, ok := trackerResp.Data["issue_id"].(string); ok {
			issueID = value
		} else if value, ok := trackerResp.Data["issueId"].(string); ok {
			issueID = value
		} else if rawIssue, ok := trackerResp.Data["issue"]; ok {
			switch v := rawIssue.(type) {
			case map[string]interface{}:
				if nested, ok := v["id"].(string); ok {
					issueID = nested
				}
			case struct{ ID string }:
				issueID = strings.TrimSpace(v.ID)
			}
		}
	}

	resultMessage := strings.TrimSpace(trackerResp.Message)
	if resultMessage == "" {
		resultMessage = "Issue reported successfully"
	}

	return &IssueReportResult{
		IssueID: issueID,
		Message: resultMessage,
	}, nil
}
