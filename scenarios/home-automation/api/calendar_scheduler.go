package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CalendarScheduler struct {
	db               *sql.DB
	deviceController *DeviceController
}

type CalendarEventRequest struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"` // starting, ending, updated
	EventData EventData `json:"event_data"`
}

type EventData struct {
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
	AllDay       bool     `json:"all_day,omitempty"`
	Participants []string `json:"participants,omitempty"`
	Location     string   `json:"location,omitempty"`
	Priority     string   `json:"priority,omitempty"`
}

type CalendarEventResponse struct {
	Success             bool                   `json:"success"`
	EventID             string                 `json:"event_id"`
	DetectedContext     string                 `json:"detected_context"`
	ContextActivated    bool                   `json:"context_activated"`
	DeviceChanges       []DeviceChange         `json:"device_changes,omitempty"`
	Message             string                 `json:"message"`
	ProcessingTimestamp string                 `json:"processing_timestamp"`
}

type DeviceChange struct {
	DeviceID    string                 `json:"device_id"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Success     bool                   `json:"success"`
	Message     string                 `json:"message,omitempty"`
}

type ContextConfig struct {
	SceneID             string                            `json:"scene_id,omitempty"`
	AutomationOverrides map[string]map[string]interface{} `json:"automation_overrides"`
	Description         string                            `json:"description"`
}

type CurrentContext struct {
	ContextName   string                 `json:"context_name"`
	ActiveSince   string                 `json:"active_since"`
	TriggeredBy   string                 `json:"triggered_by"`
	Configuration ContextConfig          `json:"configuration"`
	ActiveDevices map[string]interface{} `json:"active_devices"`
}

func NewCalendarScheduler(db *sql.DB) *CalendarScheduler {
	return &CalendarScheduler{
		db: db,
	}
}

func (cs *CalendarScheduler) SetDeviceController(dc *DeviceController) {
	cs.deviceController = dc
}

func (cs *CalendarScheduler) HandleCalendarEvent(ctx context.Context, req CalendarEventRequest) (*CalendarEventResponse, error) {
	// Validate event request
	if req.EventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	if req.EventType == "" {
		return nil, fmt.Errorf("event_type is required (starting, ending, updated)")
	}
	if req.EventData.Title == "" {
		return nil, fmt.Errorf("event_data.title is required")
	}

	// Detect context from event
	detectedContext := cs.detectContextFromEvent(req.EventData)
	
	var deviceChanges []DeviceChange
	var contextActivated bool
	var message string

	// Process based on event type
	switch req.EventType {
	case "starting", "updated":
		// Activate context
		err := cs.ActivateContext(ctx, detectedContext)
		if err != nil {
			return nil, fmt.Errorf("failed to activate context: %w", err)
		}
		
		// Apply context changes
		changes, err := cs.applyContextChanges(ctx, detectedContext)
		if err != nil {
			return nil, fmt.Errorf("failed to apply context changes: %w", err)
		}
		
		deviceChanges = changes
		contextActivated = true
		message = fmt.Sprintf("Context '%s' activated for event '%s'", detectedContext, req.EventData.Title)

	case "ending":
		// Deactivate context
		err := cs.DeactivateContext(ctx, detectedContext)
		if err != nil {
			return nil, fmt.Errorf("failed to deactivate context: %w", err)
		}
		
		contextActivated = false
		message = fmt.Sprintf("Context '%s' deactivated for ended event '%s'", detectedContext, req.EventData.Title)

	default:
		return nil, fmt.Errorf("invalid event_type: %s", req.EventType)
	}

	// Log the event processing
	if err := cs.logCalendarEvent(ctx, req, detectedContext, contextActivated); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to log calendar event: %v\n", err)
	}

	return &CalendarEventResponse{
		Success:             true,
		EventID:             req.EventID,
		DetectedContext:     detectedContext,
		ContextActivated:    contextActivated,
		DeviceChanges:       deviceChanges,
		Message:             message,
		ProcessingTimestamp: time.Now().Format(time.RFC3339),
	}, nil
}

func (cs *CalendarScheduler) detectContextFromEvent(eventData EventData) string {
	// Map calendar events to home automation contexts
	contextMapping := map[string]string{
		// Work contexts
		"meeting":        "focus_mode",
		"conference":     "focus_mode",
		"call":           "focus_mode",
		"work from home": "work_mode",
		"home office":    "work_mode",
		"zoom":           "focus_mode",
		
		// Entertainment contexts
		"movie":       "entertainment_mode",
		"tv":          "entertainment_mode",
		"game":        "entertainment_mode",
		"party":       "party_mode",
		"netflix":     "entertainment_mode",
		"streaming":   "entertainment_mode",
		
		// Sleep contexts
		"bedtime":     "sleep_mode",
		"nap":         "quiet_mode",
		"sleep":       "sleep_mode",
		"rest":        "quiet_mode",
		
		// Away contexts
		"vacation":    "away_mode",
		"trip":        "away_mode",
		"out of town": "away_mode",
		"travel":      "away_mode",
		"away":        "away_mode",
		
		// Activity contexts
		"cooking":   "kitchen_active_mode",
		"dinner":    "dinner_mode",
		"breakfast": "morning_mode",
		"lunch":     "kitchen_active_mode",
		"workout":   "exercise_mode",
		"exercise":  "exercise_mode",
		"gym":       "exercise_mode",
	}

	// Combine title and description for keyword matching
	combinedText := strings.ToLower(eventData.Title + " " + eventData.Description)

	// Find matching context
	for keyword, context := range contextMapping {
		if strings.Contains(combinedText, keyword) {
			return context
		}
	}

	// Default context
	return "default_mode"
}

func (cs *CalendarScheduler) getContextConfig(context string) ContextConfig {
	// Mock context configurations - in real implementation, query database
	contextConfigs := map[string]ContextConfig{
		"focus_mode": {
			SceneID: "",
			AutomationOverrides: map[string]map[string]interface{}{
				"light.living_room": {"brightness": 90, "color": "white"},
				"climate.thermostat": {"target_temp": 70},
				"notification_settings": {"do_not_disturb": true},
			},
			Description: "Optimized for focused work with bright lighting",
		},
		"entertainment_mode": {
			SceneID: "scene_movie_night",
			AutomationOverrides: map[string]map[string]interface{}{
				"light.living_room": {"brightness": 10, "color": "orange"},
				"climate.thermostat": {"target_temp": 72},
			},
			Description: "Dim lights and comfortable temperature for entertainment",
		},
		"sleep_mode": {
			SceneID: "scene_bedtime",
			AutomationOverrides: map[string]map[string]interface{}{
				"light.living_room": {"on": false},
				"light.bedroom": {"brightness": 5, "color_temp": 450},
				"lock.front_door": {"locked": true},
			},
			Description: "Secure home and gentle lighting for sleep",
		},
		"away_mode": {
			SceneID: "scene_away_mode",
			AutomationOverrides: map[string]map[string]interface{}{
				"security_mode": {"enabled": true},
				"climate.thermostat": {"target_temp": 65},
				"light.living_room": {"on": false},
				"light.bedroom": {"on": false},
			},
			Description: "Security and energy optimization when away",
		},
		"work_mode": {
			SceneID: "",
			AutomationOverrides: map[string]map[string]interface{}{
				"light.living_room": {"brightness": 85},
				"climate.thermostat": {"target_temp": 70},
				"switch.coffee_maker": {"on": true},
			},
			Description: "Work from home optimizations",
		},
		"kitchen_active_mode": {
			SceneID: "",
			AutomationOverrides: map[string]map[string]interface{}{
				"light.kitchen": {"brightness": 95},
				"switch.exhaust_fan": {"on": true},
			},
			Description: "Optimal lighting and ventilation for cooking",
		},
		"morning_mode": {
			SceneID: "",
			AutomationOverrides: map[string]map[string]interface{}{
				"light.bedroom": {"brightness": 60},
				"switch.coffee_maker": {"on": true},
				"climate.thermostat": {"target_temp": 72},
			},
			Description: "Gentle wake-up lighting and coffee preparation",
		},
		"party_mode": {
			SceneID: "scene_party",
			AutomationOverrides: map[string]map[string]interface{}{
				"light.living_room": {"brightness": 80, "color": "rainbow"},
				"climate.thermostat": {"target_temp": 68},
			},
			Description: "Festive lighting for entertaining",
		},
	}

	config, exists := contextConfigs[context]
	if !exists {
		return ContextConfig{
			AutomationOverrides: map[string]map[string]interface{}{},
			Description:         "Default home state",
		}
	}

	return config
}

func (cs *CalendarScheduler) applyContextChanges(ctx context.Context, contextName string) ([]DeviceChange, error) {
	config := cs.getContextConfig(contextName)
	var deviceChanges []DeviceChange

	// Apply automation overrides
	for deviceID, settings := range config.AutomationOverrides {
		// Skip special settings that aren't device commands
		if strings.Contains(deviceID, "_settings") || strings.Contains(deviceID, "_mode") {
			continue
		}

		// Determine action based on settings
		action := cs.determineActionFromSettings(settings)
		if action == "" {
			continue
		}

		// Apply device control if device controller is available
		if cs.deviceController != nil {
			controlReq := DeviceControlRequest{
				DeviceID:   deviceID,
				Action:     action,
				Parameters: settings,
				UserID:     "550e8400-e29b-41d4-a716-446655440001", // System user
			}

			controlResp, err := cs.deviceController.ControlDevice(ctx, controlReq)
			
			deviceChange := DeviceChange{
				DeviceID:   deviceID,
				Action:     action,
				Parameters: settings,
				Success:    err == nil && controlResp != nil && controlResp.Success,
			}

			if err != nil {
				deviceChange.Message = err.Error()
			} else if controlResp != nil {
				deviceChange.Message = controlResp.Message
			}

			deviceChanges = append(deviceChanges, deviceChange)
		} else {
			// Mock the device change if no controller available
			deviceChanges = append(deviceChanges, DeviceChange{
				DeviceID:   deviceID,
				Action:     action,
				Parameters: settings,
				Success:    true,
				Message:    fmt.Sprintf("Applied %s to %s", action, deviceID),
			})
		}
	}

	return deviceChanges, nil
}

func (cs *CalendarScheduler) determineActionFromSettings(settings map[string]interface{}) string {
	// Determine the primary action based on the settings provided
	if on, exists := settings["on"]; exists {
		if on == false {
			return "turn_off"
		}
		return "turn_on"
	}
	
	if locked, exists := settings["locked"]; exists {
		if locked == true {
			return "lock"
		}
		return "unlock"
	}
	
	if _, exists := settings["brightness"]; exists {
		return "set_brightness"
	}
	
	if _, exists := settings["target_temp"]; exists {
		return "set_temperature"
	}
	
	if _, exists := settings["color"]; exists {
		return "set_color"
	}

	// Default to turn_on if settings exist but no specific action determined
	return "turn_on"
}

func (cs *CalendarScheduler) ActivateContext(ctx context.Context, contextName string) error {
	if cs.db == nil {
		// Just log for mock mode
		fmt.Printf("Activating context: %s\n", contextName)
		return nil
	}

	// Store active context in database
	query := `
		INSERT INTO home_automation.active_contexts (context_name, activated_at, triggered_by)
		VALUES ($1, CURRENT_TIMESTAMP, 'calendar')
		ON CONFLICT (context_name) DO UPDATE SET
		activated_at = CURRENT_TIMESTAMP,
		triggered_by = EXCLUDED.triggered_by
	`
	
	_, err := cs.db.Exec(query, contextName)
	return err
}

func (cs *CalendarScheduler) DeactivateContext(ctx context.Context, contextName string) error {
	if cs.db == nil {
		// Just log for mock mode
		fmt.Printf("Deactivating context: %s\n", contextName)
		return nil
	}

	// Remove active context from database
	query := `DELETE FROM home_automation.active_contexts WHERE context_name = $1`
	
	_, err := cs.db.Exec(query, contextName)
	return err
}

func (cs *CalendarScheduler) GetCurrentContext(ctx context.Context) (*CurrentContext, error) {
	if cs.db == nil {
		// Return mock data
		return &CurrentContext{
			ContextName:   "default_mode",
			ActiveSince:   time.Now().Format(time.RFC3339),
			TriggeredBy:   "system",
			Configuration: cs.getContextConfig("default_mode"),
			ActiveDevices: map[string]interface{}{},
		}, nil
	}

	query := `
		SELECT context_name, activated_at, triggered_by
		FROM home_automation.active_contexts
		ORDER BY activated_at DESC
		LIMIT 1
	`
	
	var contextName, triggeredBy string
	var activatedAt time.Time
	
	err := cs.db.QueryRow(query).Scan(&contextName, &activatedAt, &triggeredBy)
	if err == sql.ErrNoRows {
		// No active context, return default
		return &CurrentContext{
			ContextName:   "default_mode",
			ActiveSince:   time.Now().Format(time.RFC3339),
			TriggeredBy:   "system",
			Configuration: cs.getContextConfig("default_mode"),
			ActiveDevices: map[string]interface{}{},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &CurrentContext{
		ContextName:   contextName,
		ActiveSince:   activatedAt.Format(time.RFC3339),
		TriggeredBy:   triggeredBy,
		Configuration: cs.getContextConfig(contextName),
		ActiveDevices: map[string]interface{}{}, // Would query device states
	}, nil
}

func (cs *CalendarScheduler) logCalendarEvent(ctx context.Context, req CalendarEventRequest, detectedContext string, contextActivated bool) error {
	if cs.db == nil {
		return nil // Skip if no database connection
	}

	eventJSON, _ := json.Marshal(req)
	
	query := `
		INSERT INTO home_automation.calendar_events 
		(event_id, event_type, event_data, detected_context, context_activated, processed_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (event_id) DO UPDATE SET
		event_type = EXCLUDED.event_type,
		event_data = EXCLUDED.event_data,
		detected_context = EXCLUDED.detected_context,
		context_activated = EXCLUDED.context_activated,
		processed_at = EXCLUDED.processed_at
	`
	
	_, err := cs.db.Exec(query, req.EventID, req.EventType, eventJSON, detectedContext, contextActivated)
	return err
}