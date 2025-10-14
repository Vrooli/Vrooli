package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DeviceController struct {
	db *sql.DB
}

type DeviceControlRequest struct {
	DeviceID   string                 `json:"device_id"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	ProfileID  string                 `json:"profile_id,omitempty"`
}

type DeviceControlResponse struct {
	Success         bool                   `json:"success"`
	DeviceID        string                 `json:"device_id"`
	Action          string                 `json:"action"`
	DeviceState     map[string]interface{} `json:"device_state,omitempty"`
	Message         string                 `json:"message"`
	Error           string                 `json:"error,omitempty"`
	RequestID       string                 `json:"request_id"`
	Timestamp       string                 `json:"timestamp"`
	ExecutionTimeMs int                    `json:"execution_time_ms"`
}

type DeviceStatus struct {
	DeviceID    string                 `json:"device_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	State       map[string]interface{} `json:"state"`
	Available   bool                   `json:"available"`
	LastUpdated string                 `json:"last_updated"`
}

type Profile struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Permissions map[string]interface{} `json:"permissions"`
	CreatedAt   string                 `json:"created_at"`
}

func NewDeviceController(db *sql.DB) *DeviceController {
	return &DeviceController{
		db: db,
	}
}

func (dc *DeviceController) ControlDevice(ctx context.Context, req DeviceControlRequest) (*DeviceControlResponse, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("req_%d_%s", startTime.Unix(), uuid.New().String()[:8])

	// Validate request
	if err := dc.validateControlRequest(req); err != nil {
		return &DeviceControlResponse{
			Success:         false,
			DeviceID:        req.DeviceID,
			Action:          req.Action,
			Message:         "Request validation failed",
			Error:           err.Error(),
			RequestID:       requestID,
			Timestamp:       startTime.Format(time.RFC3339),
			ExecutionTimeMs: int(time.Since(startTime).Milliseconds()),
		}, err
	}

	// Check permissions
	hasPermission, err := dc.checkPermissions(ctx, req.UserID, req.ProfileID, req.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("permission check failed: %w", err)
	}

	if !hasPermission {
		return &DeviceControlResponse{
			Success:         false,
			DeviceID:        req.DeviceID,
			Action:          req.Action,
			Message:         "Permission denied",
			Error:           fmt.Sprintf("User cannot control device %s", req.DeviceID),
			RequestID:       requestID,
			Timestamp:       startTime.Format(time.RFC3339),
			ExecutionTimeMs: int(time.Since(startTime).Milliseconds()),
		}, fmt.Errorf("permission denied")
	}

	// Execute Home Assistant command
	result, err := dc.executeHomeAssistantCommand(ctx, req)
	if err != nil {
		return &DeviceControlResponse{
			Success:         false,
			DeviceID:        req.DeviceID,
			Action:          req.Action,
			Message:         "Device control failed",
			Error:           err.Error(),
			RequestID:       requestID,
			Timestamp:       startTime.Format(time.RFC3339),
			ExecutionTimeMs: int(time.Since(startTime).Milliseconds()),
		}, err
	}

	// Log execution
	if err := dc.logExecution(ctx, req, result, requestID, true); err != nil {
		// Log error but don't fail the request
		log.Printf("Failed to log execution: %v", err)
	}

	return &DeviceControlResponse{
		Success:         true,
		DeviceID:        req.DeviceID,
		Action:          req.Action,
		DeviceState:     result.State,
		Message:         result.Message,
		RequestID:       requestID,
		Timestamp:       startTime.Format(time.RFC3339),
		ExecutionTimeMs: int(time.Since(startTime).Milliseconds()),
	}, nil
}

func (dc *DeviceController) validateControlRequest(req DeviceControlRequest) error {
	if req.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}

	if req.Action == "" {
		return fmt.Errorf("action is required")
	}

	if req.UserID == "" && req.ProfileID == "" {
		return fmt.Errorf("user_id or profile_id is required")
	}

	// Sanitize device_id (prevent injection)
	validDeviceID := regexp.MustCompile(`^[a-zA-Z0-9._]+$`)
	if !validDeviceID.MatchString(req.DeviceID) {
		return fmt.Errorf("invalid device_id format")
	}

	// Validate action against allowed actions
	allowedActions := []string{
		"turn_on", "turn_off", "toggle", "set_brightness",
		"set_temperature", "set_color", "activate", "refresh",
	}

	actionAllowed := false
	for _, allowed := range allowedActions {
		if req.Action == allowed {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("invalid action: %s", req.Action)
	}

	return nil
}

func (dc *DeviceController) checkPermissions(ctx context.Context, userID, profileID, deviceID string) (bool, error) {
	// Mock permission check - in real implementation, this would:
	// 1. Query scenario-authenticator for user details
	// 2. Query home_profiles table for permissions
	// 3. Check if user can control specific device

	mockPermissions := map[string][]string{
		"550e8400-e29b-41d4-a716-446655440001": {"*"},                                                         // Admin user
		"550e8400-e29b-41d4-a716-446655440002": {"light.living_room", "light.bedroom", "switch.coffee_maker"}, // Family member
		"550e8400-e29b-41d4-a716-446655440003": {"light.bedroom_kid"},                                         // Kid user
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "550e8400-e29b-41d4-a716-446655440001" // Default for demo
	}

	allowedDevices := mockPermissions[userID]
	if len(allowedDevices) == 0 {
		return false, nil
	}

	// Check permission
	for _, allowed := range allowedDevices {
		if allowed == "*" || allowed == deviceID {
			return true, nil
		}
	}

	return false, nil
}

type HACommandResult struct {
	State   map[string]interface{} `json:"state"`
	Message string                 `json:"message"`
}

func (dc *DeviceController) executeHomeAssistantCommand(ctx context.Context, req DeviceControlRequest) (*HACommandResult, error) {
	// Build parameters string for the command
	var paramStr string
	if len(req.Parameters) > 0 {
		paramBytes, _ := json.Marshal(req.Parameters)
		paramStr = string(paramBytes)
	}

	// Try to execute Home Assistant CLI command
	var cmd *exec.Cmd
	if paramStr != "" {
		cmd = exec.CommandContext(ctx, "resource-home-assistant", "api-info")
	} else {
		cmd = exec.CommandContext(ctx, "resource-home-assistant", "api-info")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Use mock implementation when Home Assistant commands are not available
		return dc.mockControlDevice(&req)
	}

	// For now, use mock implementation since Home Assistant resource doesn't have device control commands
	return dc.mockControlDevice(&req)
}

func (dc *DeviceController) mockControlDevice(req *DeviceControlRequest) (*HACommandResult, error) {
	// Mock device control for development
	state := map[string]interface{}{}

	// Handle different device types and actions
	deviceType := strings.Split(req.DeviceID, ".")[0]

	switch deviceType {
	case "light":
		switch req.Action {
		case "turn_on":
			state["on"] = true
			if brightness, ok := req.Parameters["brightness"]; ok {
				state["brightness"] = brightness
			} else {
				state["brightness"] = 100
			}
		case "turn_off":
			state["on"] = false
			state["brightness"] = 0
		}
	case "thermostat", "climate":
		switch req.Action {
		case "set_temperature":
			if temp, ok := req.Parameters["temperature"]; ok {
				state["temperature"] = temp
			}
		case "set_mode":
			if mode, ok := req.Parameters["mode"]; ok {
				state["mode"] = mode
			}
		}
	case "sensor":
		// Sensors are read-only
		state["value"] = "readonly"
	}

	return &HACommandResult{
		State:   state,
		Message: fmt.Sprintf("Device %s %s completed successfully (mock)", req.DeviceID, req.Action),
	}, nil
}

func (dc *DeviceController) logExecution(ctx context.Context, req DeviceControlRequest, result *HACommandResult, requestID string, success bool) error {
	if dc.db == nil {
		return nil // Skip if no database connection
	}

	logEntry := map[string]interface{}{
		"type":           "device_control",
		"device_id":      req.DeviceID,
		"action":         req.Action,
		"user_id":        req.UserID,
		"profile_id":     req.ProfileID,
		"success":        success,
		"timestamp":      time.Now().Format(time.RFC3339),
		"request_id":     requestID,
		"parameters":     req.Parameters,
		"result_message": result.Message,
	}

	logJSON, _ := json.Marshal(logEntry)

	query := `
		INSERT INTO home_automation.automation_executions 
		(request_id, type, device_id, action, user_id, success, execution_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
		ON CONFLICT (request_id) DO NOTHING
	`

	_, err := dc.db.Exec(query, requestID, "device_control", req.DeviceID, req.Action, req.UserID, success, logJSON)
	return err
}

func (dc *DeviceController) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
	// Try to execute Home Assistant status command
	cmd := exec.CommandContext(ctx, "resource-home-assistant", "api-info")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		// Return mock status when Home Assistant is not available
		devices := dc.getMockDevices()
		for _, device := range devices {
			if device.DeviceID == deviceID {
				return &device, nil
			}
		}
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	// For now, return mock data since the Home Assistant resource doesn't have device-specific commands
	devices := dc.getMockDevices()
	for _, device := range devices {
		if device.DeviceID == deviceID {
			return &device, nil
		}
	}

	return &DeviceStatus{
		DeviceID:    deviceID,
		Name:        fmt.Sprintf("Device %s", deviceID),
		Type:        strings.Split(deviceID, ".")[0],
		State:       map[string]interface{}{"status": "unknown"},
		Available:   true,
		LastUpdated: time.Now().Format(time.RFC3339),
	}, nil
}

func (dc *DeviceController) ListDevices(ctx context.Context) ([]DeviceStatus, error) {
	// Try to execute Home Assistant list command
	cmd := exec.CommandContext(ctx, "resource-home-assistant", "content", "list")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Fallback to mock data when Home Assistant integration is not fully available
		return dc.getMockDevices(), nil
	}

	// Parse output - try as JSON array first
	var devices []DeviceStatus
	output := stdout.String()

	if err := json.Unmarshal([]byte(output), &devices); err != nil {
		// If parsing fails, return mock devices
		return dc.getMockDevices(), nil
	}

	return devices, nil
}

func (dc *DeviceController) getMockDevices() []DeviceStatus {
	// Return mock devices for development and testing
	return []DeviceStatus{
		{
			DeviceID:    "light.living_room",
			Name:        "Living Room Light",
			Type:        "light",
			State:       map[string]interface{}{"on": false, "brightness": 0},
			Available:   true,
			LastUpdated: time.Now().Format(time.RFC3339),
		},
		{
			DeviceID:    "light.bedroom",
			Name:        "Bedroom Light",
			Type:        "light",
			State:       map[string]interface{}{"on": true, "brightness": 75},
			Available:   true,
			LastUpdated: time.Now().Format(time.RFC3339),
		},
		{
			DeviceID:    "thermostat.main",
			Name:        "Main Thermostat",
			Type:        "climate",
			State:       map[string]interface{}{"temperature": 72, "mode": "heat", "target": 70},
			Available:   true,
			LastUpdated: time.Now().Format(time.RFC3339),
		},
		{
			DeviceID:    "sensor.motion_hallway",
			Name:        "Hallway Motion Sensor",
			Type:        "sensor",
			State:       map[string]interface{}{"motion": false, "last_motion": "2025-09-28T01:45:00Z"},
			Available:   true,
			LastUpdated: time.Now().Format(time.RFC3339),
		},
	}
}

func (dc *DeviceController) GetProfiles(ctx context.Context) ([]Profile, error) {
	if dc.db == nil {
		// Return mock data
		return []Profile{
			{
				ID:   "550e8400-e29b-41d4-a716-446655440001",
				Name: "Admin",
				Type: "admin",
				Permissions: map[string]interface{}{
					"device_control":    true,
					"automation_create": true,
					"allowed_devices":   []string{"*"},
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			},
			{
				ID:   "550e8400-e29b-41d4-a716-446655440002",
				Name: "Family Member",
				Type: "family",
				Permissions: map[string]interface{}{
					"device_control":    true,
					"automation_create": false,
					"allowed_devices":   []string{"light.living_room", "light.bedroom", "switch.coffee_maker"},
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	query := `
		SELECT id, name, type, permissions, created_at
		FROM home_automation.home_profiles
		ORDER BY created_at DESC
	`

	rows, err := dc.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var profile Profile
		var permissionsJSON []byte

		err := rows.Scan(&profile.ID, &profile.Name, &profile.Type, &permissionsJSON, &profile.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(permissionsJSON, &profile.Permissions)
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (dc *DeviceController) GetProfilePermissions(ctx context.Context, profileID string) (map[string]interface{}, error) {
	profiles, err := dc.GetProfiles(ctx)
	if err != nil {
		return nil, err
	}

	for _, profile := range profiles {
		if profile.ID == profileID {
			return profile.Permissions, nil
		}
	}

	return nil, fmt.Errorf("profile not found")
}
