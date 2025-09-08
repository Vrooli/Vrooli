package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OrchestratorManager handles profile activation and resource orchestration
type OrchestratorManager struct {
	profileManager *ProfileManager
	logger         *Logger
}

// NewOrchestratorManager creates a new orchestrator manager
func NewOrchestratorManager(profileManager *ProfileManager, logger *Logger) *OrchestratorManager {
	return &OrchestratorManager{
		profileManager: profileManager,
		logger:         logger,
	}
}

// ActivationResult represents the result of profile activation
type ActivationResult struct {
	ProfileName     string                 `json:"profile_name"`
	Success         bool                   `json:"success"`
	ResourcesStatus map[string]interface{} `json:"resources_status"`
	ScenariosStatus map[string]interface{} `json:"scenarios_status"`
	BrowserActions  []string               `json:"browser_actions"`
	Message         string                 `json:"message"`
	Error           string                 `json:"error,omitempty"`
}

// DeactivationResult represents the result of profile deactivation
type DeactivationResult struct {
	Success         bool                   `json:"success"`
	ResourcesStatus map[string]interface{} `json:"resources_status"`
	ScenariosStatus map[string]interface{} `json:"scenarios_status"`
	Message         string                 `json:"message"`
	Error           string                 `json:"error,omitempty"`
}

// ActivateProfile activates a profile by starting resources and scenarios
func (om *OrchestratorManager) ActivateProfile(profileName string, options map[string]interface{}) (*ActivationResult, error) {
	// Get the profile
	profile, err := om.profileManager.GetProfile(profileName)
	if err != nil {
		return &ActivationResult{
			ProfileName: profileName,
			Success:     false,
			Error:       err.Error(),
		}, err
	}
	
	om.logger.Info(fmt.Sprintf("Starting activation of profile: %s", profileName))
	
	// Deactivate any existing active profile first
	if err := om.DeactivateCurrentProfile(); err != nil {
		om.logger.Error("Failed to deactivate current profile", err)
		// Continue anyway - non-critical error
	}
	
	result := &ActivationResult{
		ProfileName:     profileName,
		Success:         true,
		ResourcesStatus: make(map[string]interface{}),
		ScenariosStatus: make(map[string]interface{}),
		BrowserActions:  []string{},
	}
	
	// Start resources
	if len(profile.Resources) > 0 {
		om.logger.Info(fmt.Sprintf("Starting %d resources", len(profile.Resources)))
		for _, resource := range profile.Resources {
			status := om.startResource(resource)
			result.ResourcesStatus[resource] = status
			if !status["success"].(bool) {
				result.Success = false
			}
		}
	}
	
	// Wait for resources to be ready
	if result.Success {
		om.logger.Info("Waiting for resources to be ready...")
		time.Sleep(3 * time.Second) // Give resources time to start
	}
	
	// Start scenarios
	if len(profile.Scenarios) > 0 && result.Success {
		om.logger.Info(fmt.Sprintf("Starting %d scenarios", len(profile.Scenarios)))
		for _, scenario := range profile.Scenarios {
			status := om.startScenario(scenario)
			result.ScenariosStatus[scenario] = status
			if !status["success"].(bool) {
				result.Success = false
			}
		}
	}
	
	// Handle browser automation
	if len(profile.AutoBrowser) > 0 && result.Success {
		om.logger.Info(fmt.Sprintf("Opening %d browser tabs", len(profile.AutoBrowser)))
		for _, url := range profile.AutoBrowser {
			action := om.openBrowserTab(url)
			result.BrowserActions = append(result.BrowserActions, action)
		}
	}
	
	// Set environment variables
	if profile.EnvironmentVars != nil {
		om.logger.Info("Setting environment variables")
		for key, value := range profile.EnvironmentVars {
			os.Setenv(key, value)
		}
	}
	
	// Mark profile as active if everything succeeded
	if result.Success {
		if err := om.profileManager.SetActiveProfile(profile.ID); err != nil {
			om.logger.Error("Failed to set profile as active", err)
			result.Success = false
			result.Error = "Failed to update profile status"
		} else {
			result.Message = fmt.Sprintf("Profile '%s' activated successfully", profileName)
			om.logger.Info(result.Message)
		}
	} else {
		result.Error = "Some resources or scenarios failed to start"
		result.Message = fmt.Sprintf("Profile '%s' activation completed with errors", profileName)
	}
	
	return result, nil
}

// DeactivateCurrentProfile deactivates the currently active profile
func (om *OrchestratorManager) DeactivateCurrentProfile() error {
	// Get current active profile
	activeProfile, err := om.profileManager.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}
	
	if activeProfile == nil {
		om.logger.Info("No active profile to deactivate")
		return nil
	}
	
	om.logger.Info(fmt.Sprintf("Deactivating profile: %s", activeProfile.Name))
	
	// Stop scenarios
	if len(activeProfile.Scenarios) > 0 {
		om.logger.Info(fmt.Sprintf("Stopping %d scenarios", len(activeProfile.Scenarios)))
		for _, scenario := range activeProfile.Scenarios {
			om.stopScenario(scenario)
		}
	}
	
	// Stop resources
	if len(activeProfile.Resources) > 0 {
		om.logger.Info(fmt.Sprintf("Stopping %d resources", len(activeProfile.Resources)))
		for _, resource := range activeProfile.Resources {
			om.stopResource(resource)
		}
	}
	
	// Clear active profile
	if err := om.profileManager.ClearActiveProfile(); err != nil {
		return fmt.Errorf("failed to clear active profile: %w", err)
	}
	
	om.logger.Info(fmt.Sprintf("Profile '%s' deactivated successfully", activeProfile.Name))
	return nil
}

// GetDeactivationResult returns a detailed deactivation result
func (om *OrchestratorManager) GetDeactivationResult() (*DeactivationResult, error) {
	result := &DeactivationResult{
		Success:         true,
		ResourcesStatus: make(map[string]interface{}),
		ScenariosStatus: make(map[string]interface{}),
	}
	
	if err := om.DeactivateCurrentProfile(); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to deactivate profile"
	} else {
		result.Message = "Profile deactivated successfully"
	}
	
	return result, nil
}

// startResource starts a specific resource
func (om *OrchestratorManager) startResource(resourceName string) map[string]interface{} {
	om.logger.Info(fmt.Sprintf("Starting resource: %s", resourceName))
	
	cmd := exec.Command("vrooli", "resource", resourceName, "start")
	cmd.Env = append(os.Environ(), "VROOLI_ORCHESTRATOR_MODE=true")
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	
	output, err := cmd.CombinedOutput()
	
	status := map[string]interface{}{
		"resource": resourceName,
		"output":   string(output),
		"success":  err == nil,
	}
	
	if err != nil {
		status["error"] = err.Error()
		om.logger.Error(fmt.Sprintf("Failed to start resource %s", resourceName), err)
	} else {
		om.logger.Info(fmt.Sprintf("Successfully started resource: %s", resourceName))
	}
	
	return status
}

// stopResource stops a specific resource
func (om *OrchestratorManager) stopResource(resourceName string) map[string]interface{} {
	om.logger.Info(fmt.Sprintf("Stopping resource: %s", resourceName))
	
	cmd := exec.Command("vrooli", "resource", resourceName, "stop")
	cmd.Env = append(os.Environ(), "VROOLI_ORCHESTRATOR_MODE=true")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	
	output, err := cmd.CombinedOutput()
	
	status := map[string]interface{}{
		"resource": resourceName,
		"output":   string(output),
		"success":  err == nil,
	}
	
	if err != nil {
		status["error"] = err.Error()
		om.logger.Error(fmt.Sprintf("Failed to stop resource %s", resourceName), err)
	} else {
		om.logger.Info(fmt.Sprintf("Successfully stopped resource: %s", resourceName))
	}
	
	return status
}

// startScenario starts a specific scenario
func (om *OrchestratorManager) startScenario(scenarioName string) map[string]interface{} {
	om.logger.Info(fmt.Sprintf("Starting scenario: %s", scenarioName))
	
	cmd := exec.Command("vrooli", "scenario", "run", scenarioName)
	cmd.Env = append(os.Environ(), "VROOLI_ORCHESTRATOR_MODE=true")
	
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	
	output, err := cmd.CombinedOutput()
	
	status := map[string]interface{}{
		"scenario": scenarioName,
		"output":   string(output),
		"success":  err == nil,
	}
	
	if err != nil {
		status["error"] = err.Error()
		om.logger.Error(fmt.Sprintf("Failed to start scenario %s", scenarioName), err)
	} else {
		om.logger.Info(fmt.Sprintf("Successfully started scenario: %s", scenarioName))
	}
	
	return status
}

// stopScenario stops a specific scenario
func (om *OrchestratorManager) stopScenario(scenarioName string) map[string]interface{} {
	om.logger.Info(fmt.Sprintf("Stopping scenario: %s", scenarioName))
	
	cmd := exec.Command("vrooli", "scenario", "stop", scenarioName)
	cmd.Env = append(os.Environ(), "VROOLI_ORCHESTRATOR_MODE=true")
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	
	output, err := cmd.CombinedOutput()
	
	status := map[string]interface{}{
		"scenario": scenarioName,
		"output":   string(output),
		"success":  err == nil,
	}
	
	if err != nil {
		status["error"] = err.Error()
		om.logger.Error(fmt.Sprintf("Failed to stop scenario %s", scenarioName), err)
	} else {
		om.logger.Info(fmt.Sprintf("Successfully stopped scenario: %s", scenarioName))
	}
	
	return status
}

// openBrowserTab opens a URL in browser using browserless if available
func (om *OrchestratorManager) openBrowserTab(url string) string {
	om.logger.Info(fmt.Sprintf("Opening browser tab: %s", url))
	
	// Try browserless first
	browserlessPort := getResourcePort("browserless")
	if browserlessPort != "" {
		browserlessURL := fmt.Sprintf("http://localhost:%s", browserlessPort)
		
		// Simple approach - just log the URL and instruction
		// A full implementation would use browserless API to open the tab
		action := fmt.Sprintf("Browser tab requested: %s (via browserless at %s)", url, browserlessURL)
		om.logger.Info(action)
		return action
	}
	
	// Fallback - just provide the URL to user
	action := fmt.Sprintf("Please open: %s", url)
	om.logger.Info(action)
	return action
}

// ValidateProfile validates that a profile's resources and scenarios exist
func (om *OrchestratorManager) ValidateProfile(profile *Profile) []string {
	var issues []string
	
	// Validate resources
	for _, resource := range profile.Resources {
		if !om.isResourceAvailable(resource) {
			issues = append(issues, fmt.Sprintf("Resource '%s' is not available", resource))
		}
	}
	
	// Validate scenarios
	for _, scenario := range profile.Scenarios {
		if !om.isScenarioAvailable(scenario) {
			issues = append(issues, fmt.Sprintf("Scenario '%s' is not available", scenario))
		}
	}
	
	return issues
}

// isResourceAvailable checks if a resource is available
func (om *OrchestratorManager) isResourceAvailable(resourceName string) bool {
	cmd := exec.Command("vrooli", "resource", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	return strings.Contains(string(output), resourceName)
}

// isScenarioAvailable checks if a scenario is available
func (om *OrchestratorManager) isScenarioAvailable(scenarioName string) bool {
	cmd := exec.Command("vrooli", "scenario", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	return strings.Contains(string(output), scenarioName)
}