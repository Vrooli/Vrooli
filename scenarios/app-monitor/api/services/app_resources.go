// Package services provides resource management services for app-monitor.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ResourceStatus represents the status of a Vrooli resource.
type ResourceStatus struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	Enabled      bool   `json:"enabled"`
	EnabledKnown bool   `json:"enabled_known"`
	Running      bool   `json:"running"`
	StatusDetail string `json:"status_detail"`
}

// GetResources returns the status of all Vrooli resources.
func (s *AppService) GetResources(ctx context.Context) ([]ResourceStatus, error) {
	// Execute vrooli resource status --json with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "vrooli", "resource", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute vrooli resource status: %w", err)
	}

	// Parse the JSON response
	var resources []map[string]interface{}
	if err := json.Unmarshal(output, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse resource response: %w", err)
	}

	// Transform resource data
	result := make([]ResourceStatus, 0, len(resources))
	for _, resource := range resources {
		if transformed := transformResource(resource); transformed != nil {
			result = append(result, *transformed)
		}
	}

	return result, nil
}

// GetResource returns detailed information about a specific resource.
func (s *AppService) GetResource(ctx context.Context, resourceID string) (*ResourceStatus, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, fmt.Errorf("resource_id is required")
	}

	// Execute vrooli resource status <name> --json
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "vrooli", "resource", "status", resourceID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", resourceID, err)
	}

	// Parse the JSON response
	var parsed interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse resource response: %w", err)
	}

	// Handle array or object response
	switch value := parsed.(type) {
	case []interface{}:
		for _, item := range value {
			if typed, ok := item.(map[string]interface{}); ok {
				if transformed := transformResource(typed); transformed != nil {
					return transformed, nil
				}
			}
		}
	case map[string]interface{}:
		if transformed := transformResource(value); transformed != nil {
			return transformed, nil
		}
	}

	return nil, fmt.Errorf("resource %s not found", resourceID)
}

// StartResource starts a Vrooli resource.
func (s *AppService) StartResource(ctx context.Context, resourceID string) error {
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Errorf("resource_id is required")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "vrooli", "resource", "start", resourceID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Errorf("failed to start resource %s: %s", resourceID, errMsg)
	}

	return nil
}

// StopResource stops a Vrooli resource.
func (s *AppService) StopResource(ctx context.Context, resourceID string) error {
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Errorf("resource_id is required")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "vrooli", "resource", "stop", resourceID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Errorf("failed to stop resource %s: %s", resourceID, errMsg)
	}

	return nil
}

// transformResource converts raw CLI output to ResourceStatus.
func transformResource(raw map[string]interface{}) *ResourceStatus {
	name := stringVal(lookupVal(raw, "Name"))
	if name == "" {
		name = stringVal(lookupVal(raw, "name"))
	}
	if name == "" {
		return nil
	}

	enabled, enabledKnown := parseBoolVal(lookupVal(raw, "Enabled"))
	if !enabledKnown {
		enabled, enabledKnown = parseBoolVal(lookupVal(raw, "enabled"))
	}

	running, _ := parseBoolVal(lookupVal(raw, "Running"))
	if !running {
		running, _ = parseBoolVal(lookupVal(raw, "running"))
	}

	statusDetail := stringVal(lookupVal(raw, "Status"))
	if statusDetail == "" {
		statusDetail = stringVal(lookupVal(raw, "status"))
	}
	normalizedStatus := strings.ToLower(statusDetail)

	typeValue := stringVal(lookupVal(raw, "Type"))
	if typeValue == "" {
		typeValue = stringVal(lookupVal(raw, "type"))
	}
	if typeValue == "" {
		typeValue = name
	}

	status := "offline"
	switch {
	case strings.Contains(normalizedStatus, "unregistered"):
		status = "unregistered"
	case strings.Contains(normalizedStatus, "error") || strings.Contains(normalizedStatus, "failed"):
		status = "error"
	case running:
		status = "online"
	case enabled && enabledKnown:
		status = "stopped"
	case !enabledKnown && !running:
		status = "unknown"
	}

	return &ResourceStatus{
		ID:           name,
		Name:         name,
		Type:         typeValue,
		Status:       status,
		Enabled:      enabled,
		EnabledKnown: enabledKnown,
		Running:      running,
		StatusDetail: statusDetail,
	}
}

// lookupVal looks up a value in a map case-insensitively.
func lookupVal(raw map[string]interface{}, key string) interface{} {
	if raw == nil {
		return nil
	}
	for k, v := range raw {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}

// stringVal converts a value to a string.
func stringVal(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// parseBoolVal parses a value as a boolean.
func parseBoolVal(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		normalized := strings.TrimSpace(strings.ToLower(v))
		switch normalized {
		case "true", "yes", "y", "1", "online", "running", "enabled":
			return true, true
		case "false", "no", "n", "0", "offline", "disabled", "stopped":
			return false, true
		case "", "n/a", "na", "unknown":
			return false, false
		default:
			return false, false
		}
	case float64:
		return v != 0, true
	default:
		return false, false
	}
}
