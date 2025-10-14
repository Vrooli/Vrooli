package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

type SafetyValidator struct {
	db *sql.DB
}

type AutomationValidationRequest struct {
	AutomationID   string                 `json:"automation_id,omitempty"`
	AutomationCode string                 `json:"automation_code"`
	Description    string                 `json:"description"`
	UserID         string                 `json:"user_id,omitempty"`
	ProfileID      string                 `json:"profile_id,omitempty"`
	TargetDevices  []string               `json:"target_devices,omitempty"`
	ScheduleConfig map[string]interface{} `json:"schedule_config,omitempty"`
	GeneratedBy    string                 `json:"generated_by,omitempty"`
}

type AutomationValidationResponse struct {
	AutomationID         string               `json:"automation_id"`
	ValidationPassed     bool                 `json:"validation_passed"`
	SecurityValidation   SecurityValidation   `json:"security_validation"`
	PermissionValidation PermissionValidation `json:"permission_validation"`
	LogicValidation      LogicValidation      `json:"logic_validation"`
	OverallRiskLevel     string               `json:"overall_risk_level"`
	Recommendations      []string             `json:"recommendations,omitempty"`
	ValidationTimestamp  string               `json:"validation_timestamp"`
}

type SecurityValidation struct {
	Passed         bool            `json:"passed"`
	SecurityIssues []SecurityIssue `json:"security_issues"`
	Warnings       []SecurityIssue `json:"warnings"`
	RiskLevel      string          `json:"risk_level"`
}

type SecurityIssue struct {
	Type     string `json:"type"`
	Pattern  string `json:"pattern,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	DeviceID string `json:"device_id,omitempty"`
	Action   string `json:"action,omitempty"`
}

type PermissionValidation struct {
	Passed           bool              `json:"passed"`
	PermissionIssues []PermissionIssue `json:"permission_issues"`
	UserPermissions  UserPermissions   `json:"user_permissions"`
}

type PermissionIssue struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type UserPermissions struct {
	CanCreate       bool `json:"can_create"`
	AllowedDevices  int  `json:"allowed_devices"`
	AutomationCount int  `json:"automation_count"`
	AutomationLimit int  `json:"automation_limit"`
}

type LogicValidation struct {
	Passed              bool              `json:"passed"`
	LogicIssues         []LogicIssue      `json:"logic_issues"`
	Suggestions         []LogicSuggestion `json:"suggestions"`
	DeviceCompatibility string            `json:"device_compatibility"`
	ScheduleSafety      string            `json:"schedule_safety"`
}

type LogicIssue struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id,omitempty"`
	Action   string `json:"action,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type LogicSuggestion struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type SafetyStatus struct {
	AutomationID   string                       `json:"automation_id"`
	CurrentStatus  string                       `json:"current_status"`
	RiskLevel      string                       `json:"risk_level"`
	LastValidated  string                       `json:"last_validated"`
	ValidationInfo AutomationValidationResponse `json:"validation_info"`
}

func NewSafetyValidator(db *sql.DB) *SafetyValidator {
	return &SafetyValidator{
		db: db,
	}
}

func (sv *SafetyValidator) ValidateAutomation(ctx context.Context, req AutomationValidationRequest) (*AutomationValidationResponse, error) {
	// Generate automation ID if not provided
	if req.AutomationID == "" {
		req.AutomationID = fmt.Sprintf("auto_%d", time.Now().Unix())
	}

	// Perform security validation
	securityValidation := sv.validateSecurity(req.AutomationCode, req.Description)

	// Perform permission validation
	permissionValidation, err := sv.validatePermissions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("permission validation failed: %w", err)
	}

	// Perform logic validation
	logicValidation := sv.validateLogic(req)

	// Determine overall validation result
	validationPassed := securityValidation.Passed && permissionValidation.Passed && logicValidation.Passed

	// Determine overall risk level
	riskLevel := sv.determineOverallRiskLevel(securityValidation, permissionValidation, logicValidation)

	// Generate recommendations
	recommendations := sv.generateRecommendations(securityValidation, logicValidation)

	response := &AutomationValidationResponse{
		AutomationID:         req.AutomationID,
		ValidationPassed:     validationPassed,
		SecurityValidation:   securityValidation,
		PermissionValidation: permissionValidation,
		LogicValidation:      logicValidation,
		OverallRiskLevel:     riskLevel,
		Recommendations:      recommendations,
		ValidationTimestamp:  time.Now().Format(time.RFC3339),
	}

	// Store validation result
	if err := sv.storeValidationResult(ctx, response); err != nil {
		// Log error but don't fail the validation
		log.Printf("Failed to store validation result: %v", err)
	}

	return response, nil
}

func (sv *SafetyValidator) validateSecurity(code, description string) SecurityValidation {
	var securityIssues []SecurityIssue
	var warnings []SecurityIssue

	// Dangerous command patterns
	dangerousPatterns := map[string]*regexp.Regexp{
		"file_deletion":       regexp.MustCompile(`rm\s+-rf`),
		"elevated_privileges": regexp.MustCompile(`sudo\s+`),
		"code_execution":      regexp.MustCompile(`exec\s*\(`),
		"system_calls":        regexp.MustCompile(`system\s*\(`),
		"dynamic_evaluation":  regexp.MustCompile(`eval\s*\(`),
		"variable_injection":  regexp.MustCompile(`\$\{.*\}`),
		"file_operations":     regexp.MustCompile(`\bopen\s*\(`),
		"process_spawning":    regexp.MustCompile(`import\s+subprocess`),
		"os_commands":         regexp.MustCompile(`os\.system`),
		"shell_execution":     regexp.MustCompile(`shell=True`),
		"network_requests":    regexp.MustCompile(`curl\s+|wget\s+`),
		"remote_access":       regexp.MustCompile(`ssh\s+`),
		"credential_handling": regexp.MustCompile(`password|secret|api[_\-]?key`),
	}

	for patternType, pattern := range dangerousPatterns {
		if pattern.MatchString(code) {
			securityIssues = append(securityIssues, SecurityIssue{
				Type:     "dangerous_pattern",
				Pattern:  pattern.String(),
				Severity: "high",
				Message:  fmt.Sprintf("Potentially dangerous pattern detected: %s", patternType),
			})
		}
	}

	// Resource usage limits
	if len(code) > 10000 {
		warnings = append(warnings, SecurityIssue{
			Type:     "code_size",
			Severity: "medium",
			Message:  "Automation code is very large (>10KB) - may impact performance",
		})
	}

	// Network access detection
	networkPatterns := []*regexp.Regexp{
		regexp.MustCompile(`http[s]?://`),
		regexp.MustCompile(`fetch\s*\(`),
		regexp.MustCompile(`requests\.`),
		regexp.MustCompile(`urllib`),
	}

	for _, pattern := range networkPatterns {
		if pattern.MatchString(code) {
			warnings = append(warnings, SecurityIssue{
				Type:     "network_access",
				Severity: "medium",
				Message:  "Automation contains network requests - ensure they are to trusted endpoints",
			})
			break
		}
	}

	// Device safety checks
	unsafeDevicePatterns := map[string]*regexp.Regexp{
		"unlock_doors":     regexp.MustCompile(`lock.*unlock`),
		"disable_security": regexp.MustCompile(`security.*disable`),
		"disable_alarms":   regexp.MustCompile(`alarm.*off`),
		"dangerous_temp":   regexp.MustCompile(`temp.*[89]\d|temp.*100`),
		"max_brightness":   regexp.MustCompile(`bright.*100`),
	}

	combinedText := strings.ToLower(description + " " + code)
	for patternType, pattern := range unsafeDevicePatterns {
		if pattern.MatchString(combinedText) {
			securityIssues = append(securityIssues, SecurityIssue{
				Type:     "unsafe_device_operation",
				Pattern:  pattern.String(),
				Severity: "high",
				Message:  fmt.Sprintf("Automation may perform unsafe device operations: %s", patternType),
			})
		}
	}

	// Time-based safety
	timePatterns := map[string]*regexp.Regexp{
		"every_second": regexp.MustCompile(`every\s+(second|1\s*sec)`),
		"too_frequent": regexp.MustCompile(`every\s+[1-9]\s*(second|sec)`),
	}

	for _, pattern := range timePatterns {
		if pattern.MatchString(combinedText) {
			warnings = append(warnings, SecurityIssue{
				Type:     "execution_frequency",
				Severity: "medium",
				Message:  "Automation may execute too frequently - consider rate limiting",
			})
			break
		}
	}

	hasCriticalIssues := false
	for _, issue := range securityIssues {
		if issue.Severity == "high" {
			hasCriticalIssues = true
			break
		}
	}

	riskLevel := "low"
	if hasCriticalIssues {
		riskLevel = "high"
	} else if len(warnings) > 0 {
		riskLevel = "medium"
	}

	return SecurityValidation{
		Passed:         !hasCriticalIssues,
		SecurityIssues: securityIssues,
		Warnings:       warnings,
		RiskLevel:      riskLevel,
	}
}

func (sv *SafetyValidator) validatePermissions(ctx context.Context, req AutomationValidationRequest) (PermissionValidation, error) {
	// Mock user permissions - in real implementation, query database
	mockUserPermissions := map[string]UserPermissions{
		"550e8400-e29b-41d4-a716-446655440001": {
			CanCreate:       true,
			AllowedDevices:  999, // Admin can control all devices
			AutomationCount: 5,
			AutomationLimit: 50,
		},
		"550e8400-e29b-41d4-a716-446655440002": {
			CanCreate:       false, // Family member cannot create automations
			AllowedDevices:  3,
			AutomationCount: 0,
			AutomationLimit: 0,
		},
		"550e8400-e29b-41d4-a716-446655440003": {
			CanCreate:       false, // Kid cannot create automations
			AllowedDevices:  1,
			AutomationCount: 0,
			AutomationLimit: 0,
		},
	}

	userID := req.UserID
	if userID == "" {
		userID = "550e8400-e29b-41d4-a716-446655440001" // Default admin
	}

	userPerms := mockUserPermissions[userID]
	var permissionIssues []PermissionIssue

	// Check if user can create automations
	if !userPerms.CanCreate {
		permissionIssues = append(permissionIssues, PermissionIssue{
			Type:     "insufficient_permissions",
			Severity: "high",
			Message:  "User does not have permission to create automations",
		})
	}

	// Check device permissions
	allowedDevices := []string{"light.living_room", "light.bedroom", "switch.coffee_maker"}
	if userID == "550e8400-e29b-41d4-a716-446655440001" {
		allowedDevices = []string{"*"} // Admin can control all
	} else if userID == "550e8400-e29b-41d4-a716-446655440003" {
		allowedDevices = []string{"light.bedroom_kid"} // Kid limited access
	}

	for _, deviceID := range req.TargetDevices {
		canControl := false
		for _, allowed := range allowedDevices {
			if allowed == "*" || allowed == deviceID {
				canControl = true
				break
			}
		}

		if !canControl {
			permissionIssues = append(permissionIssues, PermissionIssue{
				Type:     "device_permission_denied",
				DeviceID: deviceID,
				Severity: "high",
				Message:  fmt.Sprintf("User does not have permission to control device: %s", deviceID),
			})
		}
	}

	// Check automation count limits
	if userPerms.AutomationCount >= userPerms.AutomationLimit {
		permissionIssues = append(permissionIssues, PermissionIssue{
			Type:     "automation_limit_exceeded",
			Severity: "high",
			Message:  fmt.Sprintf("User has reached automation limit (%d)", userPerms.AutomationLimit),
		})
	}

	hasPermissionIssues := false
	for _, issue := range permissionIssues {
		if issue.Severity == "high" {
			hasPermissionIssues = true
			break
		}
	}

	return PermissionValidation{
		Passed:           !hasPermissionIssues,
		PermissionIssues: permissionIssues,
		UserPermissions:  userPerms,
	}, nil
}

func (sv *SafetyValidator) validateLogic(req AutomationValidationRequest) LogicValidation {
	var logicIssues []LogicIssue
	var suggestions []LogicSuggestion

	code := req.AutomationCode
	targetDevices := req.TargetDevices
	scheduleConfig := req.ScheduleConfig

	// Device compatibility check
	deviceTypes := map[string][]string{
		"light.":   {"turn_on", "turn_off", "set_brightness", "set_color"},
		"switch.":  {"turn_on", "turn_off", "toggle"},
		"climate.": {"set_temperature", "set_mode", "set_fan_speed"},
		"lock.":    {"lock", "unlock"},
		"sensor.":  {"read_value"},
	}

	for _, deviceID := range targetDevices {
		deviceType := ""
		for prefix := range deviceTypes {
			if strings.HasPrefix(deviceID, prefix) {
				deviceType = prefix
				break
			}
		}

		if deviceType == "" {
			logicIssues = append(logicIssues, LogicIssue{
				Type:     "unknown_device_type",
				DeviceID: deviceID,
				Severity: "medium",
				Message:  fmt.Sprintf("Unknown device type for %s - validation limited", deviceID),
			})
			continue
		}

		supportedActions := deviceTypes[deviceType]
		for _, action := range []string{"turn_on", "turn_off", "set_brightness", "set_temperature"} {
			if strings.Contains(code, action) {
				actionSupported := false
				for _, supported := range supportedActions {
					if supported == action {
						actionSupported = true
						break
					}
				}
				if !actionSupported {
					logicIssues = append(logicIssues, LogicIssue{
						Type:     "unsupported_action",
						DeviceID: deviceID,
						Action:   action,
						Severity: "high",
						Message:  fmt.Sprintf("Action '%s' not supported by device %s", action, deviceID),
					})
				}
			}
		}
	}

	// Schedule validation
	if frequency, ok := scheduleConfig["frequency"].(string); ok {
		freq := strings.ToLower(frequency)
		if strings.Contains(freq, "every second") || strings.Contains(freq, "every 1 second") {
			logicIssues = append(logicIssues, LogicIssue{
				Type:     "excessive_frequency",
				Severity: "high",
				Message:  "Automation runs too frequently - may overwhelm devices",
			})
		}
	}

	// Logic conflict detection
	if strings.Contains(code, "turn_on") && strings.Contains(code, "turn_off") && !strings.Contains(code, "delay") {
		logicIssues = append(logicIssues, LogicIssue{
			Type:     "conflicting_actions",
			Severity: "medium",
			Message:  "Automation turns devices on and off without delays - may cause flickering",
		})
	}

	// Generate suggestions
	hasClimateDevice := false
	hasLightDevice := false
	for _, deviceID := range targetDevices {
		if strings.HasPrefix(deviceID, "climate.") {
			hasClimateDevice = true
		}
		if strings.HasPrefix(deviceID, "light.") {
			hasLightDevice = true
		}
	}

	if hasClimateDevice && strings.Contains(code, "temperature") {
		suggestions = append(suggestions, LogicSuggestion{
			Type:    "energy_efficiency",
			Message: "Consider adding presence detection to avoid heating/cooling empty spaces",
		})
	}

	if hasLightDevice && strings.Contains(code, "brightness.*100") {
		suggestions = append(suggestions, LogicSuggestion{
			Type:    "safety",
			Message: "Maximum brightness may be uncomfortable - consider gradual transitions",
		})
	}

	hasLogicIssues := false
	for _, issue := range logicIssues {
		if issue.Severity == "high" {
			hasLogicIssues = true
			break
		}
	}

	scheduleSafety := "not_applicable"
	if len(scheduleConfig) > 0 {
		scheduleSafety = "checked"
	}

	return LogicValidation{
		Passed:              !hasLogicIssues,
		LogicIssues:         logicIssues,
		Suggestions:         suggestions,
		DeviceCompatibility: "validated",
		ScheduleSafety:      scheduleSafety,
	}
}

func (sv *SafetyValidator) determineOverallRiskLevel(security SecurityValidation, permission PermissionValidation, logic LogicValidation) string {
	if security.RiskLevel == "high" || !permission.Passed || !logic.Passed {
		return "high"
	}
	if security.RiskLevel == "medium" || len(logic.LogicIssues) > 0 {
		return "medium"
	}
	return "low"
}

func (sv *SafetyValidator) generateRecommendations(security SecurityValidation, logic LogicValidation) []string {
	var recommendations []string

	if len(security.SecurityIssues) > 0 {
		recommendations = append(recommendations, "Review and address security issues before deploying automation")
	}

	if len(security.Warnings) > 0 {
		recommendations = append(recommendations, "Consider security warnings to improve automation safety")
	}

	for _, suggestion := range logic.Suggestions {
		recommendations = append(recommendations, suggestion.Message)
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Automation passed all safety checks - safe to deploy")
	}

	return recommendations
}

func (sv *SafetyValidator) storeValidationResult(ctx context.Context, response *AutomationValidationResponse) error {
	if sv.db == nil {
		return nil // Skip if no database connection
	}

	resultJSON, _ := json.Marshal(response)

	query := `
		INSERT INTO home_automation.automation_validations 
		(automation_id, validation_result, validation_passed, risk_level, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (automation_id) DO UPDATE SET
		validation_result = EXCLUDED.validation_result,
		validation_passed = EXCLUDED.validation_passed,
		risk_level = EXCLUDED.risk_level,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := sv.db.Exec(query, response.AutomationID, resultJSON, response.ValidationPassed, response.OverallRiskLevel)
	return err
}

func (sv *SafetyValidator) GetSafetyStatus(ctx context.Context, automationID string) (*SafetyStatus, error) {
	if sv.db == nil {
		// Return mock data
		return &SafetyStatus{
			AutomationID:  automationID,
			CurrentStatus: "validated",
			RiskLevel:     "low",
			LastValidated: time.Now().Format(time.RFC3339),
			ValidationInfo: AutomationValidationResponse{
				AutomationID:     automationID,
				ValidationPassed: true,
				OverallRiskLevel: "low",
			},
		}, nil
	}

	query := `
		SELECT validation_result, validation_passed, risk_level, created_at
		FROM home_automation.automation_validations
		WHERE automation_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var resultJSON []byte
	var validationPassed bool
	var riskLevel string
	var createdAt time.Time

	err := sv.db.QueryRow(query, automationID).Scan(&resultJSON, &validationPassed, &riskLevel, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("safety status not found: %w", err)
	}

	var validationInfo AutomationValidationResponse
	json.Unmarshal(resultJSON, &validationInfo)

	currentStatus := "validated"
	if !validationPassed {
		currentStatus = "validation_failed"
	}

	return &SafetyStatus{
		AutomationID:   automationID,
		CurrentStatus:  currentStatus,
		RiskLevel:      riskLevel,
		LastValidated:  createdAt.Format(time.RFC3339),
		ValidationInfo: validationInfo,
	}, nil
}
