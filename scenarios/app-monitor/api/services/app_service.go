package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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

// AppService handles business logic for application management
type AppService struct {
	repo  repository.AppRepository
	cache *orchestratorCache
}

// NewAppService creates a new app service
func NewAppService(repo repository.AppRepository) *AppService {
	return &AppService{
		repo:  repo,
		cache: &orchestratorCache{},
	}
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
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Path        string   `json:"path"`
	Version     string   `json:"version"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
}

// scenarioStatusDetail represents detailed runtime/port information for a scenario
type scenarioStatusDetail struct {
	Name           string              `json:"name"`
	Status         string              `json:"status"`
	Runtime        string              `json:"runtime"`
	StartedAt      string              `json:"started_at"`
	AllocatedPorts map[string]int      `json:"allocated_ports"`
	Diagnostics    scenarioDiagnostics `json:"-"`
}

// scenarioStatusResponse is the envelope returned by `vrooli scenario status <name> --json`
type scenarioStatusResponse struct {
	Success      bool                 `json:"success"`
	ScenarioData scenarioStatusDetail `json:"scenario_data"`
	Diagnostics  scenarioDiagnostics  `json:"diagnostics"`
}

type scenarioDiagnostics struct {
	Responsiveness scenarioResponsiveness               `json:"responsiveness"`
	HealthChecks   map[string]scenarioHealthCheckResult `json:"health_checks"`
}

type scenarioResponsiveness struct {
	API scenarioProbe `json:"api"`
	UI  scenarioProbe `json:"ui"`
}

type scenarioProbe struct {
	Available    bool    `json:"available"`
	ResponseTime float64 `json:"response_time"`
	Timeout      bool    `json:"timeout"`
	Error        string  `json:"error"`
}

type scenarioHealthCheckResult struct {
	Status   string `json:"status"`
	Target   string `json:"target"`
	Message  string `json:"message"`
	Critical bool   `json:"critical"`
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

		portMappings := make(map[string]interface{}, len(orchApp.Ports))
		var primaryPortLabel string
		var primaryPortValue string
		for name, port := range orchApp.Ports {
			portMappings[name] = port
			if primaryPortLabel == "" {
				primaryPortLabel = name
				primaryPortValue = fmt.Sprintf("%d", port)
			}
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

		app := repository.App{
			ID:           orchApp.Name,
			Name:         displayName,
			ScenarioName: orchApp.Name,
			Status:       status,
			PortMappings: portMappings,
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

		if primaryPortLabel != "" {
			app.Config["primary_port_label"] = primaryPortLabel
			app.Config["primary_port"] = primaryPortValue
		}

		if orchApp.Processes > 0 {
			app.Config["process_count"] = orchApp.Processes
		}

		apps = append(apps, app)
	}

	// Enrich running scenarios with precise runtime and port data
	s.enrichScenarioDetails(ctx, apps)

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

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "list", "--json")
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

		app := repository.App{
			ID:           scenario.Name,
			Name:         scenario.Name,
			ScenarioName: scenario.Name,
			Path:         scenario.Path,
			CreatedAt:    now,
			UpdatedAt:    now,
			Status:       status,
			PortMappings: make(map[string]interface{}),
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

		apps = append(apps, app)
	}

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

// isActiveStatus indicates whether a scenario should have live runtime/port data
func isActiveStatus(status string) bool {
	switch strings.ToLower(status) {
	case "running", "healthy", "degraded", "unhealthy":
		return true
	default:
		return false
	}
}

// collectScenarioStatus gathers detailed status information for the provided scenarios
func (s *AppService) collectScenarioStatus(ctx context.Context, scenarios []string) map[string]scenarioStatusDetail {
	results := make(map[string]scenarioStatusDetail, len(scenarios))
	if len(scenarios) == 0 {
		return results
	}

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		sem = make(chan struct{}, 4)
	)

	for _, scenarioName := range scenarios {
		scenarioName := scenarioName
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Allow generous timeout for scenarios under load
			detailCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(detailCtx, "vrooli", "scenario", "status", scenarioName, "--json")
			output, err := cmd.Output()
			if err != nil {
				fmt.Printf("Warning: failed to fetch detailed status for %s: %v\n", scenarioName, err)
				return
			}

			var resp scenarioStatusResponse
			if err := json.Unmarshal(output, &resp); err != nil {
				fmt.Printf("Warning: failed to parse detailed status for %s: %v\n", scenarioName, err)
				return
			}
			if !resp.Success {
				return
			}

			resp.ScenarioData.Diagnostics = resp.Diagnostics

			mu.Lock()
			results[scenarioName] = resp.ScenarioData
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results
}

// enrichScenarioDetails augments running scenarios with precise runtime and port data
func (s *AppService) enrichScenarioDetails(ctx context.Context, apps []repository.App) {
	activeNames := make([]string, 0, len(apps))
	indexByScenario := make(map[string]int, len(apps))

	for i := range apps {
		if isActiveStatus(apps[i].Status) {
			activeNames = append(activeNames, apps[i].ScenarioName)
			indexByScenario[apps[i].ScenarioName] = i
		}
	}

	if len(activeNames) == 0 {
		return
	}

	details := s.collectScenarioStatus(ctx, activeNames)
	now := time.Now().UTC()

	for scenarioName, detail := range details {
		idx, ok := indexByScenario[scenarioName]
		if !ok {
			continue
		}

		app := &apps[idx]

		// Merge allocated ports, falling back to CLI lookup when orchestrator omits them
		ports := detail.AllocatedPorts
		if len(ports) == 0 {
			ports = lookupScenarioPorts(ctx, scenarioName)
		}
		if len(ports) > 0 {
			if app.PortMappings == nil {
				app.PortMappings = make(map[string]interface{}, len(ports))
			}
			for name, port := range ports {
				app.PortMappings[name] = port
			}

			if app.Config == nil {
				app.Config = make(map[string]interface{})
			}
			app.Config["ports"] = ports
			if uiPort, ok := ports["UI_PORT"]; ok {
				app.Config["primary_port_label"] = "UI_PORT"
				app.Config["primary_port"] = fmt.Sprintf("%d", uiPort)
			}
			if apiPort, ok := ports["API_PORT"]; ok {
				app.Config["api_port"] = apiPort
			}
		}

		if isActiveStatus(app.Status) {
			healthStatus, healthMetrics := evaluateHealthStatus(app.HealthStatus, detail)
			if healthStatus != "" {
				app.HealthStatus = healthStatus
				switch healthStatus {
				case "healthy":
					app.Status = "healthy"
				case "degraded":
					app.Status = "degraded"
				case "unhealthy":
					app.Status = "unhealthy"
				case "unknown":
					if strings.EqualFold(app.Status, "unhealthy") || strings.EqualFold(app.Status, "unknown") {
						app.Status = "running"
					}
				}
			}

			if len(healthMetrics) > 0 {
				if app.Config == nil {
					app.Config = make(map[string]interface{})
				}
				app.Config["health_probes"] = healthMetrics
			}
		}

		runtime := strings.TrimSpace(detail.Runtime)
		if runtime != "" {
			app.Runtime = runtime
		}

		if start := strings.TrimSpace(detail.StartedAt); start != "" && start != "never" {
			if parsed, err := time.Parse(time.RFC3339, start); err == nil {
				app.CreatedAt = parsed.UTC()
				app.UpdatedAt = now
				app.Uptime = humanizeDuration(now.Sub(parsed.UTC()))
			}
		}

		if app.Uptime == "" {
			if duration, err := time.ParseDuration(strings.ReplaceAll(strings.ToLower(runtime), " ", "")); err == nil && duration > 0 {
				app.Uptime = humanizeDuration(duration)
			}
		}

		if app.Uptime == "" || strings.EqualFold(app.Uptime, "n/a") {
			if runtime != "" && !strings.EqualFold(runtime, "n/a") {
				app.Uptime = runtime
			}
		}
	}
}

func evaluateHealthStatus(currentHealth string, detail scenarioStatusDetail) (string, map[string]interface{}) {
	metrics := make(map[string]interface{})
	resp := detail.Diagnostics.Responsiveness

	metrics["api"] = map[string]interface{}{
		"available":     resp.API.Available,
		"timeout":       resp.API.Timeout,
		"response_time": resp.API.ResponseTime,
		"error":         resp.API.Error,
	}
	metrics["ui"] = map[string]interface{}{
		"available":     resp.UI.Available,
		"timeout":       resp.UI.Timeout,
		"response_time": resp.UI.ResponseTime,
		"error":         resp.UI.Error,
	}

	health := strings.ToLower(strings.TrimSpace(currentHealth))

	probeHealth := ""
	switch {
	case resp.API.Available && resp.UI.Available:
		probeHealth = "healthy"
	case resp.API.Available || resp.UI.Available:
		probeHealth = "degraded"
	case resp.API.Timeout || resp.UI.Timeout:
		probeHealth = "unhealthy"
	}

	if probeHealth != "" {
		health = probeHealth
	}

	if len(detail.Diagnostics.HealthChecks) > 0 {
		healthChecks := make(map[string]interface{}, len(detail.Diagnostics.HealthChecks))
		for name, check := range detail.Diagnostics.HealthChecks {
			status := strings.ToLower(strings.TrimSpace(check.Status))
			healthChecks[name] = map[string]interface{}{
				"status":   status,
				"target":   check.Target,
				"message":  check.Message,
				"critical": check.Critical,
			}

			switch status {
			case "", "pass", "ok", "healthy", "success":
				if health == "" {
					health = "healthy"
				}
			case "warn", "warning", "degraded":
				if health == "" || health == "healthy" {
					health = "degraded"
				}
			default:
				health = "unhealthy"
			}

			if health == "unhealthy" {
				// No need to continue evaluating once a critical failure is found
				break
			}
		}
		metrics["checks"] = healthChecks
	}

	if health == "" {
		health = "unknown"
	}

	return health, metrics
}

// lookupScenarioPorts fetches key ports via the Vrooli CLI when detailed status omits them
func lookupScenarioPorts(ctx context.Context, scenarioName string) map[string]int {
	ports := make(map[string]int, 2)
	for _, portName := range []string{"UI_PORT", "API_PORT"} {
		lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		cmd := exec.CommandContext(lookupCtx, "vrooli", "scenario", "port", scenarioName, portName)
		output, err := cmd.Output()
		cancel()
		if err != nil {
			continue
		}

		portStr := strings.TrimSpace(string(output))
		if portStr == "" {
			continue
		}

		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 {
			continue
		}

		ports[portName] = port
	}
	return ports
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
	if s.repo == nil {
		return nil, fmt.Errorf("database not available")
	}
	return s.repo.GetApp(ctx, id)
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

	return nil
}

// GetAppLogs retrieves logs for an application
func (s *AppService) GetAppLogs(ctx context.Context, appName string, logType string) ([]string, error) {
	args := []string{"scenario", "logs", appName}
	if logType != "" && logType != "both" {
		args = append(args, "--type", logType)
	}

	// Add timeout to prevent hanging (10s for logs as they can be large)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail on "not found" errors - just return empty logs
		if strings.Contains(string(output), "not found") || strings.Contains(string(output), "No such") {
			return []string{fmt.Sprintf("No logs available for scenario '%s'", appName)}, nil
		}
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return []string{"No logs available"}, nil
	}

	return lines, nil
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
