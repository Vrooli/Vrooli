package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
}

// AppService handles business logic for application management
type AppService struct {
	repo  repository.AppRepository
	cache *orchestratorCache
}

// NewAppService creates a new app service
func NewAppService(repo repository.AppRepository) *AppService {
	return &AppService{
		repo: repo,
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

// GetAppsFromOrchestrator fetches app status from the vrooli orchestrator with caching
func (s *AppService) GetAppsFromOrchestrator(ctx context.Context) ([]repository.App, error) {
	// Check cache first (cache valid for 15 seconds)
	s.cache.mu.RLock()
	if time.Since(s.cache.timestamp) < 15*time.Second && len(s.cache.data) > 0 {
		cachedData := s.cache.data
		s.cache.mu.RUnlock()
		return cachedData, nil
	}
	s.cache.mu.RUnlock()

	// Prevent concurrent fetches using mutex (add a fetchMutex field to orchestratorCache)
	// For now, use the write lock as a simple mutex
	s.cache.mu.Lock()
	
	// Check cache again after acquiring lock
	if time.Since(s.cache.timestamp) < 15*time.Second && len(s.cache.data) > 0 {
		cachedData := s.cache.data
		s.cache.mu.Unlock()
		return cachedData, nil
	}
	
	// Keep the lock while fetching to prevent concurrent fetches
	defer s.cache.mu.Unlock()

	// Execute vrooli scenario status --json with timeout
	// 15 second timeout to account for scenarios with issues that need attention
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Return cached data if available on error (we already have the lock)
		if len(s.cache.data) > 0 {
			return s.cache.data, nil
		}
		return nil, fmt.Errorf("failed to execute vrooli scenario status: %w", err)
	}

	// Parse the orchestrator response
	var orchestratorResp OrchestratorResponse
	if err := json.Unmarshal(output, &orchestratorResp); err != nil {
		return nil, fmt.Errorf("failed to parse orchestrator response: %w", err)
	}

	// Convert orchestrator apps to our App format
	apps := make([]repository.App, 0, len(orchestratorResp.Scenarios))
	for _, orchApp := range orchestratorResp.Scenarios {
		// Map orchestrator status to our status format
		status := orchApp.Status
		if orchApp.Status == "running" && orchApp.HealthStatus != nil && *orchApp.HealthStatus != "" {
			// Use actual health for running apps
			if *orchApp.HealthStatus == "degraded" || *orchApp.HealthStatus == "unhealthy" {
				status = *orchApp.HealthStatus
			}
		}

		// Format ports for display
		portMappings := make(map[string]interface{})
		for name, port := range orchApp.Ports {
			portMappings[name] = port
		}

		// Use DisplayName if available, otherwise fall back to Name
		displayName := orchApp.DisplayName
		if displayName == "" {
			displayName = orchApp.Name
		}

		app := repository.App{
			ID:           orchApp.Name, // Use name as ID for now
			Name:         displayName,
			ScenarioName: orchApp.Name,
			Status:       status,
			PortMappings: portMappings,
			Environment:  make(map[string]interface{}),
			Config:       make(map[string]interface{}),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Parse started_at if available
		if orchApp.StartedAt != "" && orchApp.StartedAt != "never" {
			if t, err := time.Parse(time.RFC3339, orchApp.StartedAt); err == nil {
				app.CreatedAt = t
				app.UpdatedAt = t
			}
		}

		apps = append(apps, app)
	}

	// Update cache (we already have the lock from above)
	s.cache.data = apps
	s.cache.timestamp = time.Now()

	return apps, nil
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