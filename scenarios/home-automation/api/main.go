package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type App struct {
	DB                 *sql.DB
	DeviceController   *DeviceController
	SafetyValidator    *SafetyValidator
	CalendarScheduler  *CalendarScheduler
}

var startTime time.Time

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start home-automation

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	startTime = time.Now()
	app := &App{}
	
	// Database connection - REQUIRED, no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		log.Fatal("❌ POSTGRES_HOST environment variable is required")
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		log.Fatal("❌ POSTGRES_PORT environment variable is required")
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		log.Fatal("❌ POSTGRES_USER environment variable is required")
	}
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		log.Fatal("❌ POSTGRES_PASSWORD environment variable is required")
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		// Default to home_automation for this scenario
		dbName = "home_automation"
		log.Println("ℹ️ POSTGRES_DB not set, using default: home_automation")
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	var err error
	app.DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer app.DB.Close()
	
	// Configure connection pool
	app.DB.SetMaxOpenConns(25)
	app.DB.SetMaxIdleConns(5)
	app.DB.SetConnMaxLifetime(5 * time.Minute)
	
	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = app.DB.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")

	// Initialize components
	app.DeviceController = NewDeviceController(app.DB)
	app.SafetyValidator = NewSafetyValidator(app.DB)
	app.CalendarScheduler = NewCalendarScheduler(app.DB)
	app.CalendarScheduler.SetDeviceController(app.DeviceController)
	
	// Setup routes
	router := mux.NewRouter()
	
	// Root health check for lifecycle system
	router.HandleFunc("/health", app.HealthCheck).Methods("GET")
	
	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", app.HealthCheck).Methods("GET")
	
	// Device Control routes
	api.HandleFunc("/devices/control", app.ControlDevice).Methods("POST")
	api.HandleFunc("/devices/{id}/control", app.ControlDevice).Methods("POST")
	api.HandleFunc("/devices/{id}/status", app.GetDeviceStatus).Methods("GET")
	api.HandleFunc("/devices", app.ListDevices).Methods("GET")
	
	// Automation routes with rate limiting
	// Rate limit: 10 automation generations per minute per client
	automationLimiter := NewRateLimiter(10, time.Minute)

	api.Handle("/automations/generate",
		automationLimiter.RateLimitMiddleware(http.HandlerFunc(app.GenerateAutomation))).Methods("POST")
	api.HandleFunc("/automations/validate", app.ValidateAutomation).Methods("POST")
	api.HandleFunc("/automations/{id}/safety-check", app.GetSafetyStatus).Methods("GET")
	api.HandleFunc("/automations", app.ListAutomations).Methods("GET")

	// Calendar Automation routes
	api.HandleFunc("/calendar/trigger", app.HandleCalendarEvent).Methods("POST")
	api.HandleFunc("/contexts/{context}/activate", app.ActivateContext).Methods("POST")
	api.HandleFunc("/contexts/{context}/deactivate", app.DeactivateContext).Methods("POST")
	api.HandleFunc("/contexts/current", app.GetCurrentContext).Methods("GET")
	
	// Home Profiles routes
	api.HandleFunc("/profiles", app.GetProfiles).Methods("GET")
	api.HandleFunc("/profiles/{id}/permissions", app.GetProfilePermissions).Methods("GET")
	
	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(router)
	
	// Start server
	port := getEnv("API_PORT", "30500")
	log.Printf("Home Automation API server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (app *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true

	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   "home-automation-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true,
		"dependencies": map[string]interface{}{},
	}

	// Check database connectivity and home automation tables
	dbHealth := app.checkDatabaseHealth()
	healthResponse["dependencies"].(map[string]interface{})["database"] = dbHealth
	if dbHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if dbHealth["status"] == "unhealthy" {
			readiness = false
		}
		if dbHealth["error"] != nil {
			errors = append(errors, dbHealth["error"].(map[string]interface{}))
		}
	}

	// Check Device Controller functionality
	deviceHealth := app.checkDeviceController()
	healthResponse["dependencies"].(map[string]interface{})["device_controller"] = deviceHealth
	if deviceHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if deviceHealth["error"] != nil {
			errors = append(errors, deviceHealth["error"].(map[string]interface{}))
		}
	}

	// Check Safety Validator system
	safetyHealth := app.checkSafetyValidator()
	healthResponse["dependencies"].(map[string]interface{})["safety_validator"] = safetyHealth
	if safetyHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if safetyHealth["error"] != nil {
			errors = append(errors, safetyHealth["error"].(map[string]interface{}))
		}
	}

	// Check Calendar Scheduler functionality
	calendarHealth := app.checkCalendarScheduler()
	healthResponse["dependencies"].(map[string]interface{})["calendar_scheduler"] = calendarHealth
	if calendarHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if calendarHealth["error"] != nil {
			errors = append(errors, calendarHealth["error"].(map[string]interface{}))
		}
	}

	// Update final status
	healthResponse["status"] = overallStatus
	healthResponse["readiness"] = readiness

	// Add errors if any
	if len(errors) > 0 {
		healthResponse["errors"] = errors
	}

	// Add metrics
	healthResponse["metrics"] = map[string]interface{}{
		"total_dependencies": 4,
		"healthy_dependencies": app.countHealthyDependencies(healthResponse["dependencies"].(map[string]interface{})),
		"uptime_seconds": time.Now().Unix() - startTime.Unix(),
	}

	// Return appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(healthResponse)
}

// Health check helper methods
func (app *App) checkDatabaseHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.DB.PingContext(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_CONNECTION_FAILED",
			"message": "Database connection failed: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"

	// Test devices table query
	var deviceCount int
	deviceQuery := "SELECT COUNT(*) FROM devices"
	if err := app.DB.QueryRowContext(ctx, deviceQuery).Scan(&deviceCount); err != nil {
		// Check if table doesn't exist vs other errors
		if strings.Contains(err.Error(), "does not exist") {
			health["status"] = "degraded"
			health["error"] = map[string]interface{}{
				"code": "DATABASE_TABLES_MISSING",
				"message": "Database tables not initialized. Run database setup.",
				"category": "resource",
				"retryable": true,
			}
		} else {
			health["status"] = "degraded"
			health["error"] = map[string]interface{}{
				"code": "DATABASE_DEVICES_TABLE_ERROR",
				"message": "Failed to query devices table: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else {
		health["checks"].(map[string]interface{})["devices_table"] = "ok"
		health["checks"].(map[string]interface{})["device_count"] = deviceCount
	}

	// Test home_profiles table query
	var profileCount int
	profileQuery := "SELECT COUNT(*) FROM home_profiles"
	if err := app.DB.QueryRowContext(ctx, profileQuery).Scan(&profileCount); err != nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "DATABASE_PROFILES_TABLE_ERROR",
				"message": "Failed to query home_profiles table: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else {
		health["checks"].(map[string]interface{})["profiles_table"] = "ok"
		health["checks"].(map[string]interface{})["profile_count"] = profileCount
	}

	// Test automation_rules table query
	var ruleCount int
	ruleQuery := "SELECT COUNT(*) FROM automation_rules WHERE active = true"
	if err := app.DB.QueryRowContext(ctx, ruleQuery).Scan(&ruleCount); err != nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "DATABASE_RULES_TABLE_ERROR",
				"message": "Failed to query automation_rules table: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else {
		health["checks"].(map[string]interface{})["automation_rules_table"] = "ok"
		health["checks"].(map[string]interface{})["active_rules"] = ruleCount
	}

	return health
}

func (app *App) checkDeviceController() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	if app.DeviceController == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DEVICE_CONTROLLER_NOT_INITIALIZED",
			"message": "Device controller not initialized",
			"category": "internal",
			"retryable": false,
		}
		return health
	}

	health["checks"].(map[string]interface{})["initialized"] = "ok"

	// Test device listing functionality
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	devices, err := app.DeviceController.ListDevices(ctx)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "DEVICE_LISTING_FAILED",
			"message": "Failed to list devices: " + err.Error(),
			"category": "internal",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["device_listing"] = "ok"
		health["checks"].(map[string]interface{})["total_devices"] = len(devices)
		
		// Count online devices by checking the Available field
		onlineCount := 0
		for _, device := range devices {
			if device.Available {
				onlineCount++
			}
		}
		health["checks"].(map[string]interface{})["online_devices"] = onlineCount
	}

	return health
}

func (app *App) checkSafetyValidator() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	if app.SafetyValidator == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "SAFETY_VALIDATOR_NOT_INITIALIZED",
			"message": "Safety validator not initialized",
			"category": "internal",
			"retryable": false,
		}
		return health
	}

	health["checks"].(map[string]interface{})["initialized"] = "ok"

	// Test safety rule validation by checking the rules table
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var safetyRuleCount int
	safetyQuery := "SELECT COUNT(*) FROM safety_rules WHERE is_enabled = true"
	if err := app.DB.QueryRowContext(ctx, safetyQuery).Scan(&safetyRuleCount); err != nil {
		// safety_rules table is optional - mark as not configured rather than degraded
		if strings.Contains(err.Error(), "does not exist") {
			health["checks"].(map[string]interface{})["safety_rules"] = "not_configured"
			// This is acceptable - safety validation works without the table
		} else {
			health["status"] = "degraded"
			health["error"] = map[string]interface{}{
				"code": "SAFETY_RULES_CHECK_FAILED",
				"message": "Failed to check safety rules: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else {
		health["checks"].(map[string]interface{})["safety_rules_check"] = "ok"
		health["checks"].(map[string]interface{})["enabled_safety_rules"] = safetyRuleCount

		// Warning if no safety rules are enabled
		if safetyRuleCount == 0 {
			health["checks"].(map[string]interface{})["safety_rules"] = "none_enabled"
			// Don't mark as degraded - just informational
		}
	}

	return health
}

func (app *App) checkCalendarScheduler() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	if app.CalendarScheduler == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "CALENDAR_SCHEDULER_NOT_INITIALIZED",
			"message": "Calendar scheduler not initialized",
			"category": "internal",
			"retryable": false,
		}
		return health
	}

	health["checks"].(map[string]interface{})["initialized"] = "ok"

	// Test calendar events/contexts functionality
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check current context
	currentContext, err := app.CalendarScheduler.GetCurrentContext(ctx)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "CALENDAR_CONTEXT_CHECK_FAILED",
			"message": "Failed to get current context: " + err.Error(),
			"category": "internal",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["context_check"] = "ok"
		if currentContext != nil {
			health["checks"].(map[string]interface{})["current_context"] = currentContext.ContextName
		}
	}

	// Check scheduled automation count
	var scheduledCount int
	scheduledQuery := "SELECT COUNT(*) FROM automation_rules WHERE trigger_type = 'schedule' AND active = true"
	if err := app.DB.QueryRowContext(ctx, scheduledQuery).Scan(&scheduledCount); err != nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "SCHEDULED_AUTOMATION_CHECK_FAILED",
				"message": "Failed to check scheduled automations: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else {
		health["checks"].(map[string]interface{})["scheduled_automations"] = scheduledCount
	}

	return health
}

func (app *App) countHealthyDependencies(deps map[string]interface{}) int {
	count := 0
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && status == "healthy" {
				count++
			}
		}
	}
	return count
}

// Device Control handlers
func (app *App) ControlDevice(w http.ResponseWriter, r *http.Request) {
	var req DeviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := app.DeviceController.ControlDevice(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (app *App) GetDeviceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	status, err := app.DeviceController.GetDeviceStatus(r.Context(), deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(status)
}

func (app *App) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := app.DeviceController.ListDevices(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

// Safety Validation handlers
func (app *App) GenerateAutomation(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req struct {
		Description string   `json:"description"`
		ProfileID   string   `json:"profile_id"`
		Context     string   `json:"schedule_context,omitempty"`
		Devices     []string `json:"target_devices,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Generate automation using Claude Code integration or fallback
	automationCode, explanation := app.generateAutomationCode(req.Description, req.Devices, req.Context)
	
	// Validate the generated automation
	validationReq := AutomationValidationRequest{
		AutomationCode: automationCode,
		Description:    req.Description,
		ProfileID:      req.ProfileID,
		TargetDevices:  req.Devices,
		GeneratedBy:    "claude-code",
	}
	
	validationResp, err := app.SafetyValidator.ValidateAutomation(r.Context(), validationReq)
	if err != nil {
		log.Printf("Validation error: %v", err)
	}
	
	// Determine energy impact based on devices
	energyImpact := "minimal"
	for _, device := range req.Devices {
		if strings.Contains(device, "climate") || strings.Contains(device, "heater") {
			energyImpact = "high"
			break
		} else if strings.Contains(device, "light") {
			energyImpact = "moderate"
		}
	}
	
	// Check for conflicts with existing automations
	conflicts := app.checkAutomationConflicts(req.Devices, req.Context)
	
	response := map[string]interface{}{
		"automation_id":           validationResp.AutomationID,
		"generated_code":          automationCode,
		"explanation":             explanation,
		"estimated_energy_impact": energyImpact,
		"conflicts":               conflicts,
		"validation":              validationResp,
		"ready_to_deploy":         validationResp.ValidationPassed,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *App) ListAutomations(w http.ResponseWriter, r *http.Request) {
	// TODO: Query database for automations
	// For now, return empty list
	response := map[string]interface{}{
		"automations": []map[string]interface{}{},
		"total":       0,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *App) ValidateAutomation(w http.ResponseWriter, r *http.Request) {
	var req AutomationValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := app.SafetyValidator.ValidateAutomation(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (app *App) GetSafetyStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	automationID := vars["id"]

	status, err := app.SafetyValidator.GetSafetyStatus(r.Context(), automationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(status)
}

// Calendar Automation handlers
func (app *App) HandleCalendarEvent(w http.ResponseWriter, r *http.Request) {
	var req CalendarEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := app.CalendarScheduler.HandleCalendarEvent(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (app *App) ActivateContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	context := vars["context"]

	err := app.CalendarScheduler.ActivateContext(r.Context(), context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"context": context,
		"status":  "activated",
	})
}

func (app *App) DeactivateContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	context := vars["context"]

	err := app.CalendarScheduler.DeactivateContext(r.Context(), context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"context": context,
		"status":  "deactivated",
	})
}

func (app *App) GetCurrentContext(w http.ResponseWriter, r *http.Request) {
	context, err := app.CalendarScheduler.GetCurrentContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(context)
}

// Profile handlers
func (app *App) GetProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := app.DeviceController.GetProfiles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
		"count":    len(profiles),
	})
}

func (app *App) GetProfilePermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	permissions, err := app.DeviceController.GetProfilePermissions(r.Context(), profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(permissions)
}

// Helper function to generate automation code
func (app *App) generateAutomationCode(description string, devices []string, context string) (string, string) {
	// Attempt to use Claude Code resource if available
	// For now, generate based on templates
	
	var code strings.Builder
	code.WriteString("# Home Automation Rule\n")
	code.WriteString(fmt.Sprintf("# Description: %s\n", description))
	code.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	
	// Parse description for common patterns
	descLower := strings.ToLower(description)
	
	// Time-based triggers
	if strings.Contains(descLower, "sunset") {
		code.WriteString("trigger:\n")
		code.WriteString("  - platform: sun\n")
		code.WriteString("    event: sunset\n")
		code.WriteString("    offset: '00:00:00'\n\n")
	} else if strings.Contains(descLower, "sunrise") {
		code.WriteString("trigger:\n")
		code.WriteString("  - platform: sun\n")
		code.WriteString("    event: sunrise\n")
		code.WriteString("    offset: '00:00:00'\n\n")
	} else if strings.Contains(descLower, "at") {
		// Extract time if mentioned
		code.WriteString("trigger:\n")
		code.WriteString("  - platform: time\n")
		code.WriteString("    at: '20:00:00'  # Default time, adjust as needed\n\n")
	}
	
	// Conditions
	if context != "" {
		code.WriteString("condition:\n")
		code.WriteString(fmt.Sprintf("  - condition: state\n"))
		code.WriteString(fmt.Sprintf("    entity_id: input_select.home_context\n"))
		code.WriteString(fmt.Sprintf("    state: '%s'\n\n", context))
	}
	
	// Actions based on description and devices
	code.WriteString("action:\n")
	for _, device := range devices {
		if strings.Contains(descLower, "turn on") || strings.Contains(descLower, "switch on") {
			code.WriteString(fmt.Sprintf("  - service: homeassistant.turn_on\n"))
			code.WriteString(fmt.Sprintf("    target:\n"))
			code.WriteString(fmt.Sprintf("      entity_id: %s\n", device))
			if strings.HasPrefix(device, "light.") && strings.Contains(descLower, "dim") {
				code.WriteString("    data:\n")
				code.WriteString("      brightness_pct: 30\n")
			}
		} else if strings.Contains(descLower, "turn off") || strings.Contains(descLower, "switch off") {
			code.WriteString(fmt.Sprintf("  - service: homeassistant.turn_off\n"))
			code.WriteString(fmt.Sprintf("    target:\n"))
			code.WriteString(fmt.Sprintf("      entity_id: %s\n", device))
		}
	}
	
	// Add delay if mentioned
	if strings.Contains(descLower, "after") || strings.Contains(descLower, "delay") {
		code.WriteString("  - delay:\n")
		code.WriteString("      minutes: 5  # Adjust delay as needed\n")
	}
	
	explanation := fmt.Sprintf("Generated automation for: %s", description)
	if len(devices) > 0 {
		explanation += fmt.Sprintf(" - Controls %d device(s)", len(devices))
	}
	
	return code.String(), explanation
}

// Helper function to check for automation conflicts
func (app *App) checkAutomationConflicts(devices []string, context string) []string {
	conflicts := []string{}
	
	// Query database for existing automations that might conflict
	// For now, return mock conflicts for demonstration
	
	// Check if multiple automations control the same device at the same time
	for _, device := range devices {
		if device == "light.living_room" {
			// Mock: existing automation already controls this device at sunset
			if context == "evening" || strings.Contains(context, "sunset") {
				conflicts = append(conflicts, 
					"Existing automation 'Evening Lights' already controls light.living_room at sunset")
			}
		}
	}
	
	// Check for opposing actions (one turns on, another turns off)
	if len(devices) > 3 {
		conflicts = append(conflicts, 
			"Warning: Controlling many devices simultaneously may cause power spikes")
	}
	
	return conflicts
}