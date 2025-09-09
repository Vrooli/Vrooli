package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"app-monitor-api/repository"
)

// AppService handles business logic for application management
type AppService struct {
	repo repository.AppRepository
}

// NewAppService creates a new app service
func NewAppService(repo repository.AppRepository) *AppService {
	return &AppService{repo: repo}
}

// OrchestratorResponse represents the response from vrooli scenario status --json
type OrchestratorResponse struct {
	Total   int               `json:"total"`
	Running int               `json:"running"`
	Apps    []OrchestratorApp `json:"apps"`
}

// OrchestratorApp represents an app from the orchestrator
type OrchestratorApp struct {
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	ActualHealth   string         `json:"actual_health,omitempty"`
	AllocatedPorts map[string]int `json:"allocated_ports,omitempty"`
	PID            int            `json:"pid,omitempty"`
	StartedAt      string         `json:"started_at,omitempty"`
	StoppedAt      string         `json:"stopped_at,omitempty"`
	RestartCount   int            `json:"restart_count,omitempty"`
}

// GetAppsFromOrchestrator fetches app status from the vrooli orchestrator
func (s *AppService) GetAppsFromOrchestrator(ctx context.Context) ([]repository.App, error) {
	// Execute vrooli scenario status --json
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute vrooli scenario status: %w", err)
	}

	// Parse the orchestrator response
	var orchestratorResp OrchestratorResponse
	if err := json.Unmarshal(output, &orchestratorResp); err != nil {
		return nil, fmt.Errorf("failed to parse orchestrator response: %w", err)
	}

	// Convert orchestrator apps to our App format
	apps := make([]repository.App, 0, len(orchestratorResp.Apps))
	for _, orchApp := range orchestratorResp.Apps {
		// Map orchestrator status to our status format
		status := orchApp.Status
		if orchApp.Status == "running" && orchApp.ActualHealth != "" {
			// Use actual health for running apps
			if orchApp.ActualHealth == "degraded" || orchApp.ActualHealth == "unhealthy" {
				status = orchApp.ActualHealth
			}
		}

		// Format ports for display
		portMappings := make(map[string]interface{})
		for name, port := range orchApp.AllocatedPorts {
			portMappings[name] = port
		}

		app := repository.App{
			ID:           orchApp.Name, // Use name as ID for now
			Name:         orchApp.Name,
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
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "run", appName)
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
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "stop", appName)
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

	cmd := exec.CommandContext(ctx, "vrooli", args...)
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