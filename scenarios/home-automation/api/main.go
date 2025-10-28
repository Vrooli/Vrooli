package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type App struct {
	DB                                *sql.DB
	DeviceController                  *DeviceController
	SafetyValidator                   *SafetyValidator
	CalendarScheduler                 *CalendarScheduler
	haConfigLock                      sync.RWMutex
	haConfig                          HomeAssistantConfig
	homeAssistantProvisionLock        sync.Mutex
	lastHomeAssistantProvisionAttempt time.Time
}

type HomeAssistantConfig struct {
	BaseURL              string
	Token                string
	TokenConfigured      bool
	TokenType            string
	RefreshToken         string
	AccessTokenExpiresAt *time.Time
	MockMode             bool
	LastStatus           string
	LastMessage          string
	LastError            string
	LastCheckedAt        *time.Time
	UpdatedAt            *time.Time
	UpdatedBy            string
}

type HomeAssistantDiagnostics struct {
	Status    string
	Message   string
	Error     string
	Action    string
	Target    string
	MockMode  bool
	CheckedAt *time.Time
}

var startTime time.Time

func main() {
	// Validate required lifecycle environment variable
	lifecycleManaged := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
	if lifecycleManaged == "" || lifecycleManaged != "true" {
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
		log.Fatal("❌ Database password environment variable is required")
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		// Default to home_automation for this scenario
		dbName = "home_automation"
		log.Println("ℹ️ POSTGRES_DB not set, using default: home_automation")
	}

	// Build connection string - NEVER log the password
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Log connection details WITHOUT password
	log.Printf("ℹ️  Connecting to database: host=%s port=%s user=%s dbname=%s", dbHost, dbPort, dbUser, dbName)

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

	// Implement exponential backoff for database connection with RANDOM jitter
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	// Seed random number generator for jitter (prevents thundering herd)
	rand.Seed(time.Now().UnixNano())

	log.Println("🔄 Attempting database connection with exponential backoff and random jitter...")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = app.DB.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add RANDOM jitter to prevent thundering herd across multiple instances
		// Use rand.Float64() for true randomness - NOT deterministic calculations
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(rand.Float64() * jitterRange)
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt (base: %v + random jitter: %v)", actualDelay, delay, jitter)

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	ctx := context.Background()
	haConfig, err := loadHomeAssistantConfig(ctx, app.DB)
	if err != nil {
		log.Printf("⚠️  Failed to load Home Assistant config: %v", err)
	}

	haConfig = applyEnvironmentOverrides(haConfig)
	app.setHomeAssistantConfig(haConfig)

	if should, reason := app.shouldAutoProvisionHomeAssistant(haConfig, nil); should {
		attemptReason := "startup"
		trimmedReason := strings.TrimSpace(reason)
		if trimmedReason != "" {
			attemptReason = "startup-" + trimmedReason
		}
		if diag, updated := app.attemptHomeAssistantAutoProvision(ctx, haConfig, attemptReason); updated {
			haConfig = app.getHomeAssistantConfig()
			if strings.EqualFold(diag.Status, "healthy") {
				log.Printf("🔌 Home Assistant auto-provisioned during startup for %s", strings.TrimSpace(haConfig.BaseURL))
			} else {
				log.Printf("⚠️  Home Assistant auto-provision during startup returned status=%s error=%s", diag.Status, strings.TrimSpace(diag.Error))
			}
		}
	}

	haConfig = app.getHomeAssistantConfig()
	client := buildHomeAssistantClient(haConfig)
	if haConfig.MockMode {
		log.Println("⚠️  Home Assistant mock mode enabled - using fallback data")
	} else if client == nil {
		log.Println("⚠️  Home Assistant client not configured - using fallback data")
	} else {
		log.Printf("🔌 Home Assistant client configured for %s", strings.TrimSpace(haConfig.BaseURL))
	}

	// Initialize components
	app.DeviceController = NewDeviceController(app.DB, client)
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

	// Home Assistant integration configuration
	api.HandleFunc("/integrations/home-assistant/config", app.handleGetHomeAssistantConfig).Methods("GET")
	api.HandleFunc("/integrations/home-assistant/config", app.handleUpdateHomeAssistantConfig).Methods("POST")
	api.HandleFunc("/integrations/home-assistant/provision", app.handleProvisionHomeAssistant).Methods("POST")

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Start server - Port must be explicitly configured
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	log.Printf("Home Automation API server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func (app *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true

	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":       overallStatus,
		"service":      "home-automation-api",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"readiness":    true,
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

	// Check Home Assistant dependency
	haDiag := app.homeAssistantDiagnostics(r.Context(), true)
	haHealth := map[string]interface{}{
		"status": haDiag.Status,
	}
	if haDiag.Message != "" {
		haHealth["message"] = haDiag.Message
	}
	if haDiag.Error != "" {
		haHealth["error"] = haDiag.Error
	}
	if haDiag.Action != "" {
		haHealth["action_required"] = haDiag.Action
	}
	if haDiag.Target != "" {
		haHealth["target"] = haDiag.Target
	}
	if haDiag.MockMode {
		haHealth["mock_mode"] = true
	}
	if haDiag.CheckedAt != nil {
		haHealth["last_checked_at"] = haDiag.CheckedAt.Format(time.RFC3339)
	}

	if haDiag.Status == "unhealthy" {
		readiness = false
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
	} else if haDiag.Status != "healthy" && overallStatus == "healthy" {
		overallStatus = "degraded"
	}

	healthResponse["dependencies"].(map[string]interface{})["home_assistant"] = haHealth

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
		"total_dependencies":   4,
		"healthy_dependencies": app.countHealthyDependencies(healthResponse["dependencies"].(map[string]interface{})),
		"uptime_seconds":       time.Now().Unix() - startTime.Unix(),
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
			"code":      "DATABASE_CONNECTION_FAILED",
			"message":   "Database connection failed: " + err.Error(),
			"category":  "resource",
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
				"code":      "DATABASE_TABLES_MISSING",
				"message":   "Database tables not initialized. Run database setup.",
				"category":  "resource",
				"retryable": true,
			}
		} else {
			health["status"] = "degraded"
			health["error"] = map[string]interface{}{
				"code":      "DATABASE_DEVICES_TABLE_ERROR",
				"message":   "Failed to query devices table: " + err.Error(),
				"category":  "resource",
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
				"code":      "DATABASE_PROFILES_TABLE_ERROR",
				"message":   "Failed to query home_profiles table: " + err.Error(),
				"category":  "resource",
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
				"code":      "DATABASE_RULES_TABLE_ERROR",
				"message":   "Failed to query automation_rules table: " + err.Error(),
				"category":  "resource",
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
			"code":      "DEVICE_CONTROLLER_NOT_INITIALIZED",
			"message":   "Device controller not initialized",
			"category":  "internal",
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
			"code":      "DEVICE_LISTING_FAILED",
			"message":   "Failed to list devices: " + err.Error(),
			"category":  "internal",
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
		health["checks"].(map[string]interface{})["device_data_source"] = app.DeviceController.LastDeviceDataSource()
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
			"code":      "SAFETY_VALIDATOR_NOT_INITIALIZED",
			"message":   "Safety validator not initialized",
			"category":  "internal",
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
				"code":      "SAFETY_RULES_CHECK_FAILED",
				"message":   "Failed to check safety rules: " + err.Error(),
				"category":  "resource",
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
			"code":      "CALENDAR_SCHEDULER_NOT_INITIALIZED",
			"message":   "Calendar scheduler not initialized",
			"category":  "internal",
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
			"code":      "CALENDAR_CONTEXT_CHECK_FAILED",
			"message":   "Failed to get current context: " + err.Error(),
			"category":  "internal",
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
				"code":      "SCHEDULED_AUTOMATION_CHECK_FAILED",
				"message":   "Failed to check scheduled automations: " + err.Error(),
				"category":  "resource",
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

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (app *App) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := app.DeviceController.ListDevices(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	dataSource := app.DeviceController.LastDeviceDataSource()
	response := map[string]interface{}{
		"devices":     devices,
		"count":       len(devices),
		"data_source": dataSource,
		"mock_data":   strings.EqualFold(dataSource, "mock"),
	}
	json.NewEncoder(w).Encode(response)
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
	// Query parameters for filtering
	profileID := r.URL.Query().Get("profile_id")
	activeOnly := r.URL.Query().Get("active") == "true"

	// Build query with optional filters
	query := `
		SELECT
			id, name, description, created_by, trigger_type,
			trigger_config, conditions, actions, active,
			generated_by_ai, source_code, execution_count,
			last_executed, created_at, updated_at
		FROM automation_rules
		WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	if profileID != "" {
		query += fmt.Sprintf(" AND created_by = $%d", argCount)
		args = append(args, profileID)
		argCount++
	}

	if activeOnly {
		query += fmt.Sprintf(" AND active = $%d", argCount)
		args = append(args, true)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		log.Printf("❌ Error querying automations: %v", err)
		http.Error(w, "Failed to fetch automations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	automations := []map[string]interface{}{}

	for rows.Next() {
		var (
			id, name, description, createdBy, triggerType string
			triggerConfig, conditions, actions            string
			active, generatedByAI                         bool
			sourceCode                                    sql.NullString
			executionCount                                int
			lastExecuted, createdAt, updatedAt            time.Time
		)

		err := rows.Scan(
			&id, &name, &description, &createdBy, &triggerType,
			&triggerConfig, &conditions, &actions, &active,
			&generatedByAI, &sourceCode, &executionCount,
			&lastExecuted, &createdAt, &updatedAt,
		)
		if err != nil {
			log.Printf("❌ Error scanning automation row: %v", err)
			continue
		}

		automation := map[string]interface{}{
			"id":              id,
			"name":            name,
			"description":     description,
			"created_by":      createdBy,
			"trigger_type":    triggerType,
			"trigger_config":  json.RawMessage(triggerConfig),
			"conditions":      json.RawMessage(conditions),
			"actions":         json.RawMessage(actions),
			"active":          active,
			"generated_by_ai": generatedByAI,
			"execution_count": executionCount,
			"last_executed":   lastExecuted,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
		}

		if sourceCode.Valid {
			automation["source_code"] = sourceCode.String
		}

		automations = append(automations, automation)
	}

	if err = rows.Err(); err != nil {
		log.Printf("❌ Error iterating automation rows: %v", err)
		http.Error(w, "Failed to process automations", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"automations": automations,
		"total":       len(automations),
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

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(context)
}

// Profile handlers
func (app *App) GetProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := app.DeviceController.GetProfiles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
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

func (app *App) setHomeAssistantConfig(cfg HomeAssistantConfig) {
	app.haConfigLock.Lock()
	defer app.haConfigLock.Unlock()
	app.haConfig = cfg
}

func (app *App) getHomeAssistantConfig() HomeAssistantConfig {
	app.haConfigLock.RLock()
	defer app.haConfigLock.RUnlock()
	return app.haConfig
}

func (app *App) homeAssistantDiagnostics(ctx context.Context, allowProbe bool) HomeAssistantDiagnostics {
	cfg := app.getHomeAssistantConfig()
	var client *HomeAssistantClient
	if app.DeviceController != nil {
		client = app.DeviceController.getHomeAssistantClient()
	}
	diag := evaluateHomeAssistant(ctx, cfg, client, allowProbe)

	if allowProbe {
		if should, reason := app.shouldAutoProvisionHomeAssistant(cfg, &diag); should {
			if newDiag, updated := app.attemptHomeAssistantAutoProvision(ctx, cfg, reason); updated {
				cfg = app.getHomeAssistantConfig()
				diag = newDiag
			}
		}
	}

	diag.Target = strings.TrimSpace(cfg.BaseURL)
	diag.MockMode = cfg.MockMode
	return diag
}

func applyEnvironmentOverrides(cfg HomeAssistantConfig) HomeAssistantConfig {
	base := strings.TrimSpace(os.Getenv("HOME_ASSISTANT_BASE_URL"))
	alternative := strings.TrimSpace(os.Getenv("HOME_ASSISTANT_URL"))
	if base == "" && alternative != "" {
		base = alternative
	}
	if base != "" {
		if normalized, err := normalizeBaseURL(base); err == nil {
			cfg.BaseURL = normalized
		} else {
			cfg.BaseURL = base
		}
	}

	if token := strings.TrimSpace(os.Getenv("HOME_ASSISTANT_TOKEN")); token != "" {
		cfg.Token = token
		cfg.TokenType = "long_lived"
	}

	if mock := strings.TrimSpace(os.Getenv("HOME_ASSISTANT_MOCK")); mock != "" {
		cfg.MockMode = parseBoolish(mock)
	}

	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "http://localhost:8123"
	}
	cfg.TokenConfigured = strings.TrimSpace(cfg.Token) != "" || strings.TrimSpace(cfg.RefreshToken) != ""
	return cfg
}

func parseBoolish(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func buildHomeAssistantClient(cfg HomeAssistantConfig) *HomeAssistantClient {
	if cfg.MockMode {
		return nil
	}

	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		return nil
	}

	token := strings.TrimSpace(cfg.Token)
	refresh := strings.TrimSpace(cfg.RefreshToken)
	tokenType := strings.TrimSpace(cfg.TokenType)

	if tokenType == "" {
		if token != "" {
			tokenType = "long_lived"
		} else if refresh != "" {
			tokenType = "refresh"
		}
	}

	if strings.EqualFold(tokenType, "refresh") {
		if refresh == "" {
			return nil
		}
		return NewHomeAssistantClient(base, token, tokenType, refresh, cfg.AccessTokenExpiresAt)
	}

	if token == "" {
		return nil
	}

	return NewHomeAssistantClient(base, token, tokenType, "", cfg.AccessTokenExpiresAt)
}

func loadHomeAssistantConfig(ctx context.Context, db *sql.DB) (HomeAssistantConfig, error) {
	cfg := HomeAssistantConfig{
		BaseURL: "http://localhost:8123",
	}
	if db == nil {
		return cfg, nil
	}

	const query = `SELECT value, updated_at, updated_by::text FROM system_config WHERE key = $1`
	row := db.QueryRowContext(ctx, query, "home_assistant_connection")

	var raw json.RawMessage
	var updatedAt sql.NullTime
	var updatedBy sql.NullString
	if err := row.Scan(&raw, &updatedAt, &updatedBy); err != nil {
		if err == sql.ErrNoRows {
			return cfg, nil
		}
		return cfg, err
	}

	if len(raw) > 0 {
		var stored struct {
			BaseURL              string `json:"base_url"`
			Token                string `json:"token"`
			TokenType            string `json:"token_type"`
			RefreshToken         string `json:"refresh_token"`
			AccessTokenExpiresAt string `json:"access_token_expires_at"`
			MockMode             bool   `json:"mock_mode"`
			LastStatus           string `json:"last_status"`
			LastMessage          string `json:"last_message"`
			LastError            string `json:"last_error"`
			LastCheckedAt        string `json:"last_checked_at"`
		}

		if err := json.Unmarshal(raw, &stored); err == nil {
			if strings.TrimSpace(stored.BaseURL) != "" {
				cfg.BaseURL = stored.BaseURL
			}
			cfg.Token = stored.Token
			cfg.TokenType = stored.TokenType
			cfg.RefreshToken = stored.RefreshToken
			cfg.AccessTokenExpiresAt = parseOptionalTime(stored.AccessTokenExpiresAt)
			cfg.MockMode = stored.MockMode
			cfg.LastStatus = stored.LastStatus
			cfg.LastMessage = stored.LastMessage
			cfg.LastError = stored.LastError
			cfg.LastCheckedAt = parseOptionalTime(stored.LastCheckedAt)
		}
	}

	if updatedAt.Valid {
		t := updatedAt.Time.UTC()
		cfg.UpdatedAt = &t
	}
	if updatedBy.Valid {
		cfg.UpdatedBy = updatedBy.String
	}

	cfg.TokenConfigured = strings.TrimSpace(cfg.Token) != "" || strings.TrimSpace(cfg.RefreshToken) != ""
	return cfg, nil
}

func saveHomeAssistantConfig(ctx context.Context, db *sql.DB, cfg HomeAssistantConfig) error {
	if db == nil {
		return nil
	}

	payload := map[string]interface{}{
		"base_url":  cfg.BaseURL,
		"token":     cfg.Token,
		"mock_mode": cfg.MockMode,
	}

	if strings.TrimSpace(cfg.TokenType) != "" {
		payload["token_type"] = cfg.TokenType
	}
	if strings.TrimSpace(cfg.RefreshToken) != "" {
		payload["refresh_token"] = cfg.RefreshToken
	}
	if cfg.AccessTokenExpiresAt != nil && !cfg.AccessTokenExpiresAt.IsZero() {
		payload["access_token_expires_at"] = cfg.AccessTokenExpiresAt.UTC().Format(time.RFC3339)
	}

	if cfg.LastStatus != "" {
		payload["last_status"] = cfg.LastStatus
	}
	if cfg.LastMessage != "" {
		payload["last_message"] = cfg.LastMessage
	}
	if cfg.LastError != "" {
		payload["last_error"] = cfg.LastError
	}
	if cfg.LastCheckedAt != nil && !cfg.LastCheckedAt.IsZero() {
		payload["last_checked_at"] = cfg.LastCheckedAt.UTC().Format(time.RFC3339)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO system_config (key, value, description, updated_at)
		VALUES ($1, $2::jsonb, $3, NOW())
		ON CONFLICT (key) DO UPDATE
			SET value = EXCLUDED.value,
			    description = EXCLUDED.description,
			    updated_at = NOW()
	`, "home_assistant_connection", string(encoded), "Home Assistant integration settings")
	return err
}

func parseOptionalTime(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		t := parsed.UTC()
		return &t
	}
	return nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func evaluateHomeAssistant(ctx context.Context, cfg HomeAssistantConfig, client *HomeAssistantClient, allowProbe bool) HomeAssistantDiagnostics {
	diag := HomeAssistantDiagnostics{
		Status:   "unknown",
		Message:  "Home Assistant status unknown",
		Target:   strings.TrimSpace(cfg.BaseURL),
		MockMode: cfg.MockMode,
	}

	baseConfigured := strings.TrimSpace(cfg.BaseURL) != ""
	if !baseConfigured {
		diag.Status = "unconfigured"
		diag.Message = "Home Assistant base URL not configured"
		diag.Action = "Provide the Home Assistant base URL in Settings"
		diag.Error = ""
		return diag
	}

	if cfg.MockMode {
		diag.Status = "degraded"
		diag.Message = "Mock mode enabled; using cached or demo devices"
		diag.Action = "Disable mock mode to connect to Home Assistant"
		diag.Error = ""
		return diag
	}

	if !cfg.TokenConfigured {
		diag.Status = "degraded"
		diag.Message = "Home Assistant token not configured"
		diag.Action = "Add a Home Assistant long-lived access token"
		diag.Error = ""
		return diag
	}

	if client == nil {
		diag.Status = "degraded"
		diag.Message = "Home Assistant client inactive"
		diag.Action = "Save configuration and restart the scenario"
		diag.Error = ""
		return diag
	}

	if !allowProbe {
		if cfg.LastStatus != "" {
			diag.Status = cfg.LastStatus
		}
		diag.Message = cfg.LastMessage
		diag.Error = cfg.LastError
		diag.CheckedAt = cfg.LastCheckedAt
		return diag
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := client.doRequest(ctx, http.MethodGet, "/api/", nil)
	if err != nil {
		diag.Status = "degraded"
		diag.Message = "Unable to reach Home Assistant"
		diag.Error = err.Error()
		diag.Action = actionForHomeAssistantError(diag.Error)
		now := time.Now().UTC()
		diag.CheckedAt = &now
		return diag
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	diag.Status = "healthy"
	diag.Message = "Home Assistant reachable"
	diag.Error = ""
	now := time.Now().UTC()
	diag.CheckedAt = &now
	return diag
}

func actionForHomeAssistantError(errMsg string) string {
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "401"), strings.Contains(lower, "unauthorized"), strings.Contains(lower, "forbidden"), strings.Contains(lower, "403"):
		return "Update the Home Assistant token in Settings"
	case strings.Contains(lower, "connect"), strings.Contains(lower, "refused"), strings.Contains(lower, "timeout"), strings.Contains(lower, "no such host"):
		return "Verify the Home Assistant URL and that the server is reachable"
	default:
		return "Review Home Assistant logs and configuration"
	}
}

func normalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	value := trimmed
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("missing host in Home Assistant URL")
	}
	parsed.Fragment = ""
	parsed.RawFragment = ""
	normalizedPath := strings.TrimRight(parsed.Path, "/")
	parsed.Path = normalizedPath
	parsed.RawPath = ""

	result := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	if normalizedPath != "" {
		result += normalizedPath
	}
	if parsed.RawQuery != "" {
		result += "?" + parsed.RawQuery
	}

	return result, nil
}

func (cfg HomeAssistantConfig) sanitized() HomeAssistantConfig {
	cfg.Token = ""
	cfg.RefreshToken = ""
	return cfg
}

func buildHomeAssistantResponse(cfg HomeAssistantConfig, diag HomeAssistantDiagnostics) map[string]interface{} {
	sanitized := cfg.sanitized()
	response := map[string]interface{}{
		"base_url":         sanitized.BaseURL,
		"token_configured": sanitized.TokenConfigured,
		"mock_mode":        sanitized.MockMode,
		"status":           diag.Status,
	}

	if strings.TrimSpace(sanitized.TokenType) != "" {
		response["token_type"] = sanitized.TokenType
	}
	if sanitized.AccessTokenExpiresAt != nil && !sanitized.AccessTokenExpiresAt.IsZero() {
		response["access_token_expires_at"] = formatTimePtr(sanitized.AccessTokenExpiresAt)
	}

	if diag.Message != "" {
		response["message"] = diag.Message
	}
	if diag.Error != "" {
		response["error"] = diag.Error
	}
	if diag.Action != "" {
		response["action_required"] = diag.Action
	}
	if diag.Target != "" {
		response["target"] = diag.Target
	}
	if diag.CheckedAt != nil {
		response["status_checked_at"] = formatTimePtr(diag.CheckedAt)
	}
	if sanitized.LastCheckedAt != nil {
		response["last_checked_at"] = formatTimePtr(sanitized.LastCheckedAt)
	}
	if sanitized.UpdatedAt != nil {
		response["updated_at"] = formatTimePtr(sanitized.UpdatedAt)
	}
	if strings.TrimSpace(sanitized.UpdatedBy) != "" {
		response["updated_by"] = sanitized.UpdatedBy
	}
	return response
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

type homeAssistantConfigRequest struct {
	BaseURL    string  `json:"base_url"`
	Token      *string `json:"token"`
	MockMode   *bool   `json:"mock_mode"`
	ClearToken bool    `json:"clear_token"`
	TestOnly   bool    `json:"test_only"`
}

func applyRequestToConfig(cfg *HomeAssistantConfig, req homeAssistantConfigRequest) error {
	if cfg == nil {
		return fmt.Errorf("invalid configuration target")
	}
	if strings.TrimSpace(req.BaseURL) != "" {
		normalized, err := normalizeBaseURL(req.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid base_url: %w", err)
		}
		cfg.BaseURL = normalized
	}
	if req.MockMode != nil {
		cfg.MockMode = *req.MockMode
	}
	if req.ClearToken {
		cfg.Token = ""
		cfg.RefreshToken = ""
		cfg.TokenType = ""
		cfg.AccessTokenExpiresAt = nil
	}
	if req.Token != nil {
		cfg.Token = strings.TrimSpace(*req.Token)
		if cfg.Token != "" {
			cfg.TokenType = "long_lived"
			cfg.RefreshToken = ""
			cfg.AccessTokenExpiresAt = nil
		}
	}
	cfg.TokenConfigured = strings.TrimSpace(cfg.Token) != "" || strings.TrimSpace(cfg.RefreshToken) != ""
	return nil
}

func (app *App) handleGetHomeAssistantConfig(w http.ResponseWriter, r *http.Request) {
	diag := app.homeAssistantDiagnostics(r.Context(), true)
	cfg := app.getHomeAssistantConfig()
	writeJSON(w, http.StatusOK, buildHomeAssistantResponse(cfg, diag))
}

func (app *App) handleUpdateHomeAssistantConfig(w http.ResponseWriter, r *http.Request) {
	var req homeAssistantConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	current := app.getHomeAssistantConfig()
	working := current
	if err := applyRequestToConfig(&working, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(working.BaseURL) == "" {
		writeError(w, http.StatusBadRequest, "base_url is required")
		return
	}

	client := buildHomeAssistantClient(working)
	diag := evaluateHomeAssistant(r.Context(), working, client, true)

	if req.TestOnly {
		response := buildHomeAssistantResponse(working, diag)
		response["saved"] = false
		writeJSON(w, http.StatusOK, response)
		return
	}

	now := time.Now().UTC()
	checkedAt := now
	if diag.CheckedAt != nil {
		checkedAt = diag.CheckedAt.UTC()
	}
	working.LastCheckedAt = &checkedAt
	working.LastStatus = diag.Status
	working.LastMessage = diag.Message
	working.LastError = diag.Error
	updatedAt := now
	working.UpdatedAt = &updatedAt
	working.TokenConfigured = strings.TrimSpace(working.Token) != ""

	if err := saveHomeAssistantConfig(r.Context(), app.DB, working); err != nil {
		log.Printf("Failed to persist Home Assistant config: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to persist configuration")
		return
	}

	app.setHomeAssistantConfig(working)
	if app.DeviceController != nil {
		app.DeviceController.SetHomeAssistantClient(client)
	}

	log.Printf("🔧 Home Assistant configuration updated: status=%s target=%s", diag.Status, working.BaseURL)

	response := buildHomeAssistantResponse(working, diag)
	response["saved"] = true
	writeJSON(w, http.StatusOK, response)
}
