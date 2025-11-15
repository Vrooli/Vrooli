package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Global logger for middleware and initialization code
var globalLogger *slog.Logger

func init() {
	// Initialize global structured logger with JSON output
	globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// DesktopConfig represents the configuration for generating a desktop application
type DesktopConfig struct {
	// Application identity
	AppName        string `json:"app_name" validate:"required"`
	AppDisplayName string `json:"app_display_name" validate:"required"`
	AppDescription string `json:"app_description" validate:"required"`
	Version        string `json:"version" validate:"required"`
	Author         string `json:"author" validate:"required"`
	License        string `json:"license"`
	AppID          string `json:"app_id" validate:"required"`
	AppURL         string `json:"app_url"`

	// Server configuration
	ServerType   string `json:"server_type" validate:"required,oneof=node static external executable"`
	ServerPort   int    `json:"server_port"`
	ServerPath   string `json:"server_path" validate:"required"`
	APIEndpoint  string `json:"api_endpoint" validate:"required,url"`
	ScenarioPath string `json:"scenario_dist_path"`

	// Template configuration
	Framework    string `json:"framework" validate:"required,oneof=electron tauri neutralino"`
	TemplateType string `json:"template_type" validate:"required,oneof=basic advanced multi_window kiosk"`

	// Features
	Features map[string]interface{} `json:"features"`

	// Window configuration
	Window map[string]interface{} `json:"window"`

	// Platform targets
	Platforms []string `json:"platforms"`

	// Output configuration
	OutputPath string `json:"output_path" validate:"required"`

	// Styling
	Styling map[string]interface{} `json:"styling"`
}

// PlatformBuildResult represents the result of building for a specific platform
type PlatformBuildResult struct {
	Platform    string     `json:"platform"`
	Status      string     `json:"status"` // building, ready, failed, skipped
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ErrorLog    []string   `json:"error_log,omitempty"`
	Artifact    string     `json:"artifact,omitempty"`
	FileSize    int64      `json:"file_size,omitempty"`
	SkipReason  string     `json:"skip_reason,omitempty"` // e.g., "Wine not installed"
}

// BuildStatus represents the status of a desktop application build
type BuildStatus struct {
	BuildID         string                 `json:"build_id"`
	ScenarioName    string                 `json:"scenario_name"`
	Status          string                 `json:"status"` // building, ready, partial, failed
	Framework       string                 `json:"framework"`
	TemplateType    string                 `json:"template_type"`
	Platforms       []string               `json:"platforms"`
	PlatformResults map[string]*PlatformBuildResult `json:"platform_results,omitempty"`
	OutputPath      string                 `json:"output_path"`
	CreatedAt       time.Time              `json:"created_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	ErrorLog        []string               `json:"error_log,omitempty"`
	BuildLog        []string               `json:"build_log,omitempty"`
	Artifacts       map[string]string      `json:"artifacts,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateInfo represents information about available templates
type TemplateInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Framework   string   `json:"framework"`
	UseCases    []string `json:"use_cases"`
	Features    []string `json:"features"`
	Complexity  string   `json:"complexity"`
	Examples    []string `json:"examples"`
}

// Server represents the API server
type Server struct {
	router        *mux.Router
	port          int
	buildStatuses map[string]*BuildStatus
	buildMutex    sync.RWMutex
	templateDir   string
	logger        *slog.Logger
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	// Initialize structured logger with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	server := &Server{
		router:        mux.NewRouter(),
		port:          port,
		buildStatuses: make(map[string]*BuildStatus),
		templateDir:   "../templates", // Templates are in parent directory when running from api/
		logger:        logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check at root level (required for lifecycle system)
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Health check (also available under API prefix)
	s.router.HandleFunc("/api/v1/health", s.healthHandler).Methods("GET")

	// System status
	s.router.HandleFunc("/api/v1/status", s.statusHandler).Methods("GET")

	// Template management
	s.router.HandleFunc("/api/v1/templates", s.listTemplatesHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/templates/{type}", s.getTemplateHandler).Methods("GET")

	// Desktop application operations
	s.router.HandleFunc("/api/v1/desktop/generate", s.generateDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/generate/quick", s.quickGenerateDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/status/{build_id}", s.getBuildStatusHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/build", s.buildDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/test", s.testDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/package", s.packageDesktopHandler).Methods("POST")

	// Scenario discovery
	s.router.HandleFunc("/api/v1/scenarios/desktop-status", s.getScenarioDesktopStatusHandler).Methods("GET")

	// Build by scenario name (simplified endpoint)
	s.router.HandleFunc("/api/v1/desktop/build/{scenario_name}", s.buildScenarioDesktopHandler).Methods("POST")

	// Download built packages
	s.router.HandleFunc("/api/v1/desktop/download/{scenario_name}/{platform}", s.downloadDesktopHandler).Methods("GET")

	// Delete desktop application
	s.router.HandleFunc("/api/v1/desktop/delete/{scenario_name}", s.deleteDesktopHandler).Methods("DELETE")

	// Webhook endpoints
	s.router.HandleFunc("/api/v1/desktop/webhook/build-complete", s.buildCompleteWebhookHandler).Methods("POST")

	// Setup middleware - CORS must be registered before logging to handle OPTIONS requests correctly
	s.router.Use(corsMiddleware)
	s.router.Use(loggingMiddleware)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("starting server",
		"service", "scenario-to-desktop-api",
		"port", s.port,
		"endpoints", []string{"/api/v1/health", "/api/v1/status", "/api/v1/desktop/generate"})

	// Setup graceful shutdown
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("server startup failed", "error", err)
			log.Fatal(err)
		}
	}()

	s.logger.Info("server started successfully",
		"url", fmt.Sprintf("http://localhost:%d", s.port),
		"health_endpoint", fmt.Sprintf("http://localhost:%d/api/v1/health", s.port),
		"status_endpoint", fmt.Sprintf("http://localhost:%d/api/v1/status", s.port))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("shutdown signal received", "timeout", "10s")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("server stopped successfully")
	return nil
}

// Health check handler
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "scenario-to-desktop-api",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Required field for API health check schema
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Status handler provides system information
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	// Count build statuses
	building := 0
	completed := 0
	failed := 0

	for _, status := range s.buildStatuses {
		switch status.Status {
		case "building":
			building++
		case "ready":
			completed++
		case "failed":
			failed++
		}
	}

	response := map[string]interface{}{
		"service": map[string]interface{}{
			"name":        "scenario-to-desktop",
			"version":     "1.0.0",
			"description": "Transform Vrooli scenarios into professional desktop applications",
			"status":      "running",
		},
		"statistics": map[string]interface{}{
			"total_builds":     len(s.buildStatuses),
			"active_builds":    building,
			"completed_builds": completed,
			"failed_builds":    failed,
		},
		"capabilities": []string{
			"desktop_app_generation",
			"cross_platform_packaging",
			"template_system",
			"build_automation",
		},
		"supported_frameworks": []string{"electron", "tauri", "neutralino"},
		"supported_templates":  []string{"basic", "advanced", "multi_window", "kiosk"},
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/api/v1/health", "description": "Health check"},
			{"method": "GET", "path": "/api/v1/status", "description": "System status"},
			{"method": "GET", "path": "/api/v1/templates", "description": "List templates"},
			{"method": "POST", "path": "/api/v1/desktop/generate", "description": "Generate desktop app"},
			{"method": "GET", "path": "/api/v1/desktop/status/{build_id}", "description": "Get build status"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// List templates handler
func (s *Server) listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates := []TemplateInfo{
		{
			Name:        "Basic Desktop App",
			Description: "Simple desktop application with essential features",
			Type:        "basic",
			Framework:   "electron",
			UseCases:    []string{"Simple utilities", "Basic productivity apps", "Quick prototypes"},
			Features:    []string{"Native menus", "Auto-updater", "File operations"},
			Complexity:  "low",
			Examples:    []string{"picker-wheel", "qr-code-generator", "palette-gen"},
		},
		{
			Name:        "Advanced Desktop App",
			Description: "Full-featured desktop application with advanced OS integration",
			Type:        "advanced",
			Framework:   "electron",
			UseCases:    []string{"Professional tools", "System administration", "Background services"},
			Features:    []string{"System tray", "Global shortcuts", "Rich notifications"},
			Complexity:  "medium",
			Examples:    []string{"system-monitor", "document-manager", "research-assistant"},
		},
		{
			Name:        "Multi-Window Desktop App",
			Description: "Advanced application supporting multiple windows and complex workflows",
			Type:        "multi_window",
			Framework:   "electron",
			UseCases:    []string{"IDE-like applications", "Dashboard applications", "Professional tools"},
			Features:    []string{"Window management", "Inter-window communication", "Floating panels"},
			Complexity:  "high",
			Examples:    []string{"agent-dashboard", "mind-maps", "brand-manager"},
		},
		{
			Name:        "Kiosk Mode App",
			Description: "Full-screen application for dedicated hardware and public displays",
			Type:        "kiosk",
			Framework:   "electron",
			UseCases:    []string{"Public displays", "Point-of-sale", "Industrial controls"},
			Features:    []string{"Full-screen lock", "Remote monitoring", "Auto-restart"},
			Complexity:  "high",
			Examples:    []string{"information-display", "booking-system", "retail-kiosk"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// Get template handler
func (s *Server) getTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateType := vars["type"]

	// Map template types to actual filenames
	templateFiles := map[string]string{
		"basic":        "basic-app.json",
		"advanced":     "advanced-app.json",
		"multi_window": "multi-window.json",
		"kiosk":        "kiosk-mode.json",
	}

	filename, valid := templateFiles[templateType]
	if !valid {
		http.Error(w, fmt.Sprintf("Invalid template type: %s", templateType), http.StatusBadRequest)
		return
	}

	// Read template configuration file (now safe from path traversal)
	templatePath := filepath.Join(s.templateDir, "advanced", filename)

	data, err := os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template not found: %s", templateType), http.StatusNotFound)
		return
	}

	var templateConfig map[string]interface{}
	if err := json.Unmarshal(data, &templateConfig); err != nil {
		http.Error(w, "Failed to parse template configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templateConfig)
}

// Quick generate desktop handler - auto-detects scenario configuration
func (s *Server) quickGenerateDesktopHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ScenarioName string `json:"scenario_name"`
		TemplateType string `json:"template_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate inputs
	if request.ScenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}
	if request.TemplateType == "" {
		request.TemplateType = "basic" // Default to basic template
	}

	// Validate template type
	validTemplates := []string{"basic", "advanced", "multi_window", "kiosk"}
	if !contains(validTemplates, request.TemplateType) {
		http.Error(w, fmt.Sprintf("invalid template_type: %s", request.TemplateType), http.StatusBadRequest)
		return
	}

	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}

	// Create analyzer
	analyzer := NewScenarioAnalyzer(vrooliRoot)

	// Analyze scenario
	metadata, err := analyzer.AnalyzeScenario(request.ScenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze scenario: %s", err), http.StatusBadRequest)
		return
	}

	// Validate scenario is ready for desktop generation
	if !metadata.HasUI {
		http.Error(w, fmt.Sprintf("Scenario '%s' does not have a built UI. Build it first with: cd scenarios/%s/ui && npm run build",
			request.ScenarioName, request.ScenarioName), http.StatusBadRequest)
		return
	}

	// Create desktop config from metadata
	config, err := analyzer.CreateDesktopConfigFromMetadata(metadata, request.TemplateType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create config: %s", err), http.StatusInternalServerError)
		return
	}

	s.logger.Info("quick generate request",
		"scenario", request.ScenarioName,
		"template", request.TemplateType,
		"has_ui", metadata.HasUI,
		"display_name", metadata.DisplayName)

	// Generate build ID
	buildID := uuid.New().String()

	// Create build status
	buildStatus := &BuildStatus{
		BuildID:      buildID,
		ScenarioName: config.AppName,
		Status:       "building",
		Framework:    config.Framework,
		TemplateType: config.TemplateType,
		Platforms:    config.Platforms,
		OutputPath:   filepath.Join(metadata.ScenarioPath, "platforms", "electron"),
		CreatedAt:    time.Now(),
		BuildLog:     []string{},
		ErrorLog:     []string{},
		Artifacts:    make(map[string]string),
		Metadata: map[string]interface{}{
			"auto_detected":  true,
			"ui_dist_path":   metadata.UIDistPath,
			"has_api":        config.ServerType == "external",
			"category":       metadata.Category,
			"source_version": metadata.Version,
		},
	}

	// Store build status
	s.buildMutex.Lock()
	s.buildStatuses[buildID] = buildStatus
	s.buildMutex.Unlock()

	// Start build process asynchronously
	go s.performDesktopGeneration(buildID, config)

	// Return immediate response
	response := map[string]interface{}{
		"build_id":             buildID,
		"status":               "building",
		"scenario_name":        config.AppName,
		"desktop_path":         buildStatus.OutputPath,
		"detected_metadata":    metadata,
		"install_instructions": "Run 'npm install && npm run dev' in the output directory",
		"test_command":         "npm run dev",
		"status_url":           fmt.Sprintf("/api/v1/desktop/status/%s", buildID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Generate desktop application handler
func (s *Server) generateDesktopHandler(w http.ResponseWriter, r *http.Request) {
	var config DesktopConfig

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &config); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := s.validateDesktopConfig(&config); err != nil {
		http.Error(w, fmt.Sprintf("Configuration validation failed: %s", err), http.StatusBadRequest)
		return
	}

	// Set default output path to standard location if not provided
	if config.OutputPath == "" {
		vrooliRoot := os.Getenv("VROOLI_ROOT")
		if vrooliRoot == "" {
			// Fallback to calculating from current directory
			// This assumes API is in <vrooli-root>/scenarios/scenario-to-desktop/api/
			currentDir, _ := os.Getwd()
			vrooliRoot = filepath.Join(currentDir, "../../..")
		}
		config.OutputPath = filepath.Join(
			vrooliRoot,
			"scenarios",
			config.AppName,
			"platforms",
			"electron",
		)
		s.logger.Info("using standard output path",
			"scenario", config.AppName,
			"path", config.OutputPath)
	}

	// Generate build ID
	buildID := uuid.New().String()

	// Create build status
	buildStatus := &BuildStatus{
		BuildID:      buildID,
		ScenarioName: config.AppName,
		Status:       "building",
		Framework:    config.Framework,
		TemplateType: config.TemplateType,
		Platforms:    config.Platforms,
		OutputPath:   config.OutputPath,
		CreatedAt:    time.Now(),
		BuildLog:     []string{},
		ErrorLog:     []string{},
		Artifacts:    make(map[string]string),
		Metadata:     make(map[string]interface{}),
	}

	// Store build status
	s.buildMutex.Lock()
	s.buildStatuses[buildID] = buildStatus
	s.buildMutex.Unlock()

	// Start build process asynchronously
	go s.performDesktopGeneration(buildID, &config)

	// Return immediate response
	response := map[string]interface{}{
		"build_id":             buildID,
		"status":               "building",
		"desktop_path":         config.OutputPath,
		"install_instructions": "Run 'npm install && npm run dev' in the output directory",
		"test_command":         "npm run dev",
		"status_url":           fmt.Sprintf("/api/v1/desktop/status/%s", buildID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Get build status handler
func (s *Server) getBuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buildID := vars["build_id"]

	s.buildMutex.RLock()
	status, exists := s.buildStatuses[buildID]
	s.buildMutex.RUnlock()

	if !exists {
		http.Error(w, "Build not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Build scenario desktop application by name (simplified endpoint)
func (s *Server) buildScenarioDesktopHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario_name"]

	if scenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	// Get Vrooli root
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}

	// Check if desktop wrapper exists
	desktopPath := filepath.Join(vrooliRoot, "scenarios", scenarioName, "platforms", "electron")
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Desktop wrapper not found for '%s'. Generate it first.", scenarioName), http.StatusNotFound)
		return
	}

	// Optional: Parse build options from request body
	var options struct {
		Platforms []string `json:"platforms"` // win, mac, linux
		Clean     bool     `json:"clean"`     // Clean before building
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&options)
	}

	// Default to all platforms if not specified
	if len(options.Platforms) == 0 {
		options.Platforms = []string{"win", "mac", "linux"}
	}

	buildID := uuid.New().String()

	// Initialize platform results
	platformResults := make(map[string]*PlatformBuildResult)
	for _, platform := range options.Platforms {
		platformResults[platform] = &PlatformBuildResult{
			Platform: platform,
			Status:   "pending",
		}
	}

	// Create build status
	buildStatus := &BuildStatus{
		BuildID:         buildID,
		ScenarioName:    scenarioName,
		Status:          "building",
		OutputPath:      desktopPath,
		CreatedAt:       time.Now(),
		Platforms:       options.Platforms,
		PlatformResults: platformResults,
		BuildLog:        []string{},
		ErrorLog:        []string{},
		Artifacts:       make(map[string]string),
	}

	s.buildMutex.Lock()
	s.buildStatuses[buildID] = buildStatus
	s.buildMutex.Unlock()

	s.logger.Info("starting desktop build",
		"scenario", scenarioName,
		"build_id", buildID,
		"platforms", options.Platforms)

	// Start build process asynchronously
	go s.performScenarioDesktopBuild(buildID, scenarioName, desktopPath, options.Platforms, options.Clean)

	response := map[string]interface{}{
		"build_id":     buildID,
		"status":       "building",
		"scenario":     scenarioName,
		"desktop_path": desktopPath,
		"platforms":    options.Platforms,
		"status_url":   fmt.Sprintf("/api/v1/desktop/status/%s", buildID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Download desktop application package
func (s *Server) downloadDesktopHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario_name"]
	platform := vars["platform"]

	if scenarioName == "" || platform == "" {
		http.Error(w, "scenario_name and platform are required", http.StatusBadRequest)
		return
	}

	// Validate platform
	validPlatforms := []string{"win", "mac", "linux"}
	if !contains(validPlatforms, platform) {
		http.Error(w, fmt.Sprintf("Invalid platform '%s'. Must be one of: win, mac, linux", platform), http.StatusBadRequest)
		return
	}

	// Get Vrooli root
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}

	// Find built package
	distPath := filepath.Join(vrooliRoot, "scenarios", scenarioName, "platforms", "electron", "dist-electron")
	packageFile, err := s.findBuiltPackage(distPath, platform)
	if err != nil {
		http.Error(w, fmt.Sprintf("Built package not found: %s. Build the desktop app first.", err), http.StatusNotFound)
		return
	}

	// Get file info
	fileInfo, err := os.Stat(packageFile)
	if err != nil {
		http.Error(w, "Failed to read package file", http.StatusInternalServerError)
		return
	}

	// Set appropriate content-type and headers
	contentType := "application/octet-stream"
	filename := filepath.Base(packageFile)

	if strings.HasSuffix(packageFile, ".exe") {
		contentType = "application/x-msdownload"
	} else if strings.HasSuffix(packageFile, ".dmg") {
		contentType = "application/x-apple-diskimage"
	} else if strings.HasSuffix(packageFile, ".AppImage") {
		contentType = "application/x-executable"
	} else if strings.HasSuffix(packageFile, ".deb") {
		contentType = "application/vnd.debian.binary-package"
	}

	s.logger.Info("serving download",
		"scenario", scenarioName,
		"platform", platform,
		"file", filename,
		"size", fileInfo.Size())

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream file to client
	http.ServeFile(w, r, packageFile)
}

// Delete desktop application handler
func (s *Server) deleteDesktopHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario_name"]

	if scenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	// Validate scenario name to prevent path traversal
	if strings.Contains(scenarioName, "..") || strings.Contains(scenarioName, "/") || strings.Contains(scenarioName, "\\") {
		http.Error(w, "Invalid scenario name", http.StatusBadRequest)
		return
	}

	// Get Vrooli root
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}

	// Construct desktop path - MUST be exactly platforms/electron/
	desktopPath := filepath.Join(vrooliRoot, "scenarios", scenarioName, "platforms", "electron")

	// Security check: Verify the path is actually inside platforms/electron
	absDesktopPath, err := filepath.Abs(desktopPath)
	if err != nil {
		http.Error(w, "Failed to resolve desktop path", http.StatusInternalServerError)
		return
	}

	expectedPrefix := filepath.Join(vrooliRoot, "scenarios", scenarioName, "platforms", "electron")
	absExpectedPrefix, _ := filepath.Abs(expectedPrefix)
	if absDesktopPath != absExpectedPrefix {
		s.logger.Error("path traversal attempt detected",
			"scenario", scenarioName,
			"expected", absExpectedPrefix,
			"actual", absDesktopPath)
		http.Error(w, "Security violation: invalid path", http.StatusBadRequest)
		return
	}

	// Check if desktop directory exists
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Desktop version does not exist for scenario '%s'", scenarioName), http.StatusNotFound)
		return
	}

	// Remove the entire platforms/electron directory
	if err := os.RemoveAll(desktopPath); err != nil {
		s.logger.Error("failed to delete desktop directory",
			"scenario", scenarioName,
			"path", desktopPath,
			"error", err)
		http.Error(w, fmt.Sprintf("Failed to delete desktop directory: %v", err), http.StatusInternalServerError)
		return
	}

	s.logger.Info("deleted desktop application",
		"scenario", scenarioName,
		"path", desktopPath)

	// Return success response
	response := map[string]interface{}{
		"status":        "success",
		"scenario_name": scenarioName,
		"deleted_path":  desktopPath,
		"message":       fmt.Sprintf("Desktop version of '%s' deleted successfully", scenarioName),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Build desktop application handler
func (s *Server) buildDesktopHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		DesktopPath string   `json:"desktop_path"`
		Platforms   []string `json:"platforms"`
		Sign        bool     `json:"sign"`
		Publish     bool     `json:"publish"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	buildID := uuid.New().String()

	// Create build status
	buildStatus := &BuildStatus{
		BuildID:    buildID,
		Status:     "building",
		OutputPath: request.DesktopPath,
		CreatedAt:  time.Now(),
		Platforms:  request.Platforms,
		BuildLog:   []string{},
		ErrorLog:   []string{},
	}

	s.buildMutex.Lock()
	s.buildStatuses[buildID] = buildStatus
	s.buildMutex.Unlock()

	// Start build process
	go s.performDesktopBuild(buildID, &request)

	response := map[string]interface{}{
		"build_id":   buildID,
		"status":     "building",
		"status_url": fmt.Sprintf("/api/v1/desktop/status/%s", buildID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Test desktop application handler
func (s *Server) testDesktopHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AppPath   string   `json:"app_path"`
		Platforms []string `json:"platforms"`
		Headless  bool     `json:"headless"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Perform basic validation tests
	testResults := s.runDesktopTests(&request)

	response := map[string]interface{}{
		"test_results": testResults,
		"status":       "completed",
		"timestamp":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Package desktop application handler
func (s *Server) packageDesktopHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AppPath    string `json:"app_path"`
		Store      string `json:"store"`
		Enterprise bool   `json:"enterprise"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Placeholder for packaging logic
	response := map[string]interface{}{
		"status":    "completed",
		"packages":  []string{},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Build complete webhook handler
func (s *Server) buildCompleteWebhookHandler(w http.ResponseWriter, r *http.Request) {
	buildID := r.Header.Get("X-Build-ID")
	if buildID == "" {
		http.Error(w, "Missing X-Build-ID header", http.StatusBadRequest)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Update build status if it exists
	s.buildMutex.Lock()
	if status, exists := s.buildStatuses[buildID]; exists {
		if resultStatus, ok := result["status"].(string); ok {
			status.Status = resultStatus
		}
		if resultStatus := result["status"].(string); resultStatus == "completed" || resultStatus == "failed" {
			now := time.Now()
			status.CompletedAt = &now
		}
	}
	s.buildMutex.Unlock()

	s.logger.Info("build webhook received",
		"build_id", buildID,
		"status", result["status"],
		"has_error", result["error"] != nil)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

// Validate desktop configuration
func (s *Server) validateDesktopConfig(config *DesktopConfig) error {
	// Basic validation
	if config.AppName == "" {
		return fmt.Errorf("app_name is required")
	}
	if config.Framework == "" {
		return fmt.Errorf("framework is required")
	}
	if config.TemplateType == "" {
		return fmt.Errorf("template_type is required")
	}
	// Note: output_path is optional - defaults to standard location if empty

	// Validate framework
	validFrameworks := []string{"electron", "tauri", "neutralino"}
	if !contains(validFrameworks, config.Framework) {
		return fmt.Errorf("invalid framework: %s", config.Framework)
	}

	// Validate template type
	validTemplates := []string{"basic", "advanced", "multi_window", "kiosk"}
	if !contains(validTemplates, config.TemplateType) {
		return fmt.Errorf("invalid template_type: %s", config.TemplateType)
	}

	// Set defaults
	if config.License == "" {
		config.License = "MIT"
	}
	if len(config.Platforms) == 0 {
		config.Platforms = []string{"win", "mac", "linux"}
	}
	if config.ServerPort == 0 && (config.ServerType == "node" || config.ServerType == "executable") {
		config.ServerPort = 3000
	}

	return nil
}

// Perform desktop generation (async)
func (s *Server) performDesktopGeneration(buildID string, config *DesktopConfig) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	s.buildMutex.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			s.buildMutex.Lock()
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Panic: %v", r))
			s.buildMutex.Unlock()
			now := time.Now()
			status.CompletedAt = &now
		}
	}()

	// Create configuration JSON file
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		s.buildMutex.Lock()
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to marshal config: %v", err))
		s.buildMutex.Unlock()
		return
	}

	// Write config to temporary file
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("desktop-config-%s.json", buildID))
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		s.buildMutex.Lock()
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to write config file: %v", err))
		s.buildMutex.Unlock()
		return
	}
	defer os.Remove(configPath)

	// Execute template generator
	// Path is relative to scenario root, not api directory
	templateGeneratorPath := filepath.Join("..", "templates", "build-tools", "dist", "template-generator.js")
	cmd := exec.Command("node",
		templateGeneratorPath,
		configPath)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	s.buildMutex.Lock()
	status.BuildLog = append(status.BuildLog, outputStr)

	if err != nil {
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Generation failed: %v", err))
		status.ErrorLog = append(status.ErrorLog, outputStr)
	} else {
		status.Status = "ready"
		status.Artifacts["config_path"] = configPath
		status.Artifacts["output_path"] = config.OutputPath
	}

	now := time.Now()
	status.CompletedAt = &now
	s.buildMutex.Unlock()
}

// Perform desktop build (async)
func (s *Server) performDesktopBuild(buildID string, request *struct {
	DesktopPath string   `json:"desktop_path"`
	Platforms   []string `json:"platforms"`
	Sign        bool     `json:"sign"`
	Publish     bool     `json:"publish"`
}) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	s.buildMutex.RUnlock()

	// Build steps: install dependencies, build, package
	steps := []string{"npm install", "npm run build", "npm run dist"}

	for _, step := range steps {
		cmd := exec.Command("bash", "-c", step)
		cmd.Dir = request.DesktopPath

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		s.buildMutex.Lock()
		status.BuildLog = append(status.BuildLog, fmt.Sprintf("%s: %s", step, outputStr))

		if err != nil {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("%s failed: %v", step, err))
			now := time.Now()
			status.CompletedAt = &now
			s.buildMutex.Unlock()
			return
		}
		s.buildMutex.Unlock()
	}

	s.buildMutex.Lock()
	status.Status = "ready"
	now := time.Now()
	status.CompletedAt = &now
	s.buildMutex.Unlock()
}

// Run desktop tests
func (s *Server) runDesktopTests(request *struct {
	AppPath   string   `json:"app_path"`
	Platforms []string `json:"platforms"`
	Headless  bool     `json:"headless"`
}) map[string]interface{} {
	results := map[string]interface{}{
		"package_json_valid": s.testPackageJSON(request.AppPath),
		"build_files_exist":  s.testBuildFiles(request.AppPath),
		"dependencies_ok":    s.testDependencies(request.AppPath),
	}

	return results
}

func (s *Server) testPackageJSON(appPath string) bool {
	packagePath := filepath.Join(appPath, "package.json")
	_, err := os.Stat(packagePath)
	return err == nil
}

func (s *Server) testBuildFiles(appPath string) bool {
	requiredFiles := []string{"src/main.ts", "src/preload.ts"}
	for _, file := range requiredFiles {
		filePath := filepath.Join(appPath, file)
		if _, err := os.Stat(filePath); err != nil {
			return false
		}
	}
	return true
}

func (s *Server) testDependencies(appPath string) bool {
	cmd := exec.Command("npm", "list", "--prod")
	cmd.Dir = appPath
	err := cmd.Run()
	return err == nil
}

// Perform scenario desktop build (async) - simplified version for scenario builds
func (s *Server) performScenarioDesktopBuild(buildID, scenarioName, desktopPath string, platforms []string, clean bool) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	s.buildMutex.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			s.buildMutex.Lock()
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Panic during build: %v", r))
			now := time.Now()
			status.CompletedAt = &now
			s.buildMutex.Unlock()
		}
	}()

	s.logger.Info("build started", "scenario", scenarioName, "build_id", buildID)

	// Phase 1: Common build steps (clean, install, compile)
	var commonSteps []struct {
		name    string
		command string
	}

	if clean {
		commonSteps = append(commonSteps, struct {
			name    string
			command string
		}{"clean", "npm run clean"})
	}

	commonSteps = append(commonSteps, []struct {
		name    string
		command string
	}{
		{"install", "npm install"},
		{"compile", "npm run build"},
	}...)

	// Execute common steps
	for i, step := range commonSteps {
		s.logger.Info("executing build step",
			"scenario", scenarioName,
			"step", step.name,
			"progress", fmt.Sprintf("%d/%d", i+1, len(commonSteps)))

		cmd := exec.Command("bash", "-c", step.command)
		cmd.Dir = desktopPath

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		s.buildMutex.Lock()
		logEntry := fmt.Sprintf("[%s] %s", step.name, step.command)
		if err != nil {
			logEntry += fmt.Sprintf("\nFAILED: %v", err)
		} else {
			logEntry += "\nSUCCESS"
		}
		if err != nil || len(outputStr) < 500 {
			logEntry += fmt.Sprintf("\nOutput: %s", outputStr)
		} else {
			logEntry += fmt.Sprintf("\nOutput: %s... (%d bytes)", outputStr[:500], len(outputStr))
		}
		status.BuildLog = append(status.BuildLog, logEntry)

		if err != nil {
			// Common step failed - mark all platforms as failed
			for _, platform := range platforms {
				if result, ok := status.PlatformResults[platform]; ok {
					result.Status = "failed"
					result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Common build step '%s' failed", step.name))
					result.ErrorLog = append(result.ErrorLog, outputStr)
					now := time.Now()
					result.CompletedAt = &now
				}
			}
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("%s failed: %v", step.name, err))
			status.ErrorLog = append(status.ErrorLog, outputStr)
			now := time.Now()
			status.CompletedAt = &now
			s.buildMutex.Unlock()

			s.logger.Error("common build step failed",
				"scenario", scenarioName,
				"build_id", buildID,
				"step", step.name,
				"error", err)
			return
		}
		s.buildMutex.Unlock()
	}

	// Phase 2: Build each platform independently
	distPath := filepath.Join(desktopPath, "dist-electron")
	var wg sync.WaitGroup

	for _, platform := range platforms {
		wg.Add(1)
		go func(plt string) {
			defer wg.Done()
			s.buildPlatform(buildID, scenarioName, desktopPath, distPath, plt)
		}(platform)
	}

	// Wait for all platform builds to complete
	wg.Wait()

	// Determine overall build status
	s.buildMutex.Lock()
	successCount := 0
	failedCount := 0
	skippedCount := 0

	for _, result := range status.PlatformResults {
		switch result.Status {
		case "ready":
			successCount++
		case "failed":
			failedCount++
		case "skipped":
			skippedCount++
		}
	}

	if successCount > 0 && failedCount == 0 && skippedCount == 0 {
		status.Status = "ready"
	} else if successCount > 0 {
		status.Status = "partial"
	} else {
		status.Status = "failed"
	}

	now := time.Now()
	status.CompletedAt = &now
	s.buildMutex.Unlock()

	s.logger.Info("build completed",
		"scenario", scenarioName,
		"build_id", buildID,
		"status", status.Status,
		"success", successCount,
		"failed", failedCount,
		"skipped", skippedCount)
}

// buildPlatform builds for a specific platform with dependency checking
func (s *Server) buildPlatform(buildID, scenarioName, desktopPath, distPath, platform string) {
	s.buildMutex.RLock()
	status := s.buildStatuses[buildID]
	result := status.PlatformResults[platform]
	s.buildMutex.RUnlock()

	// Check platform dependencies
	if platform == "win" {
		if !s.isWineInstalled() {
			s.buildMutex.Lock()
			result.Status = "skipped"
			result.SkipReason = "Wine not installed (required for Windows builds on Linux). Install with: sudo apt install wine"
			now := time.Now()
			result.CompletedAt = &now
			s.buildMutex.Unlock()
			s.logger.Warn("skipping Windows build - Wine not installed", "scenario", scenarioName)
			return
		}
	}

	// Mark platform build as started
	s.buildMutex.Lock()
	result.Status = "building"
	now := time.Now()
	result.StartedAt = &now
	s.buildMutex.Unlock()

	// Determine build command
	var distCommand string
	switch platform {
	case "win":
		distCommand = "npm run dist:win"
	case "mac":
		distCommand = "npm run dist:mac"
	case "linux":
		distCommand = "npm run dist:linux"
	default:
		s.buildMutex.Lock()
		result.Status = "failed"
		result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Unknown platform: %s", platform))
		now := time.Now()
		result.CompletedAt = &now
		s.buildMutex.Unlock()
		return
	}

	s.logger.Info("building platform",
		"scenario", scenarioName,
		"build_id", buildID,
		"platform", platform)

	cmd := exec.Command("bash", "-c", distCommand)
	cmd.Dir = desktopPath

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	s.buildMutex.Lock()
	logEntry := fmt.Sprintf("[package-%s] %s", platform, distCommand)
	if err != nil {
		logEntry += fmt.Sprintf("\nFAILED: %v", err)
		result.Status = "failed"
		result.ErrorLog = append(result.ErrorLog, outputStr)
	} else {
		logEntry += "\nSUCCESS"
		result.Status = "ready"
	}
	if len(outputStr) < 500 {
		logEntry += fmt.Sprintf("\nOutput: %s", outputStr)
	} else {
		logEntry += fmt.Sprintf("\nOutput: %s... (%d bytes)", outputStr[:500], len(outputStr))
	}
	status.BuildLog = append(status.BuildLog, logEntry)
	now = time.Now()
	result.CompletedAt = &now
	s.buildMutex.Unlock()

	if err != nil {
		s.logger.Error("platform build failed",
			"scenario", scenarioName,
			"build_id", buildID,
			"platform", platform,
			"error", err)
		return
	}

	// Find built package
	packageFile, err := s.findBuiltPackage(distPath, platform)
	if err == nil {
		fileInfo, _ := os.Stat(packageFile)
		s.buildMutex.Lock()
		result.Artifact = packageFile
		if fileInfo != nil {
			result.FileSize = fileInfo.Size()
		}
		status.Artifacts[platform] = packageFile
		s.buildMutex.Unlock()

		s.logger.Info("platform build succeeded",
			"scenario", scenarioName,
			"platform", platform,
			"artifact", packageFile)
	} else {
		s.buildMutex.Lock()
		result.Status = "failed"
		result.ErrorLog = append(result.ErrorLog, fmt.Sprintf("Built package not found: %v", err))
		s.buildMutex.Unlock()

		s.logger.Warn("platform package not found",
			"scenario", scenarioName,
			"platform", platform,
			"error", err)
	}
}

// isWineInstalled checks if Wine is installed on the system
func (s *Server) isWineInstalled() bool {
	cmd := exec.Command("which", "wine")
	err := cmd.Run()
	return err == nil
}

// findBuiltPackage finds the built package file for a specific platform
func (s *Server) findBuiltPackage(distPath, platform string) (string, error) {
	// Check if dist-electron directory exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dist-electron directory not found at %s", distPath)
	}

	// Platform-specific file patterns
	var patterns []string
	switch platform {
	case "win":
		patterns = []string{"*.exe"}
	case "mac":
		patterns = []string{"*.dmg"}
	case "linux":
		patterns = []string{"*.AppImage", "*.deb"}
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}

	// Search for matching files
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(distPath, pattern))
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			// Return the first match (usually there's only one)
			// Prefer Setup.exe over portable.exe for Windows
			if platform == "win" && len(matches) > 1 {
				for _, match := range matches {
					if strings.Contains(strings.ToLower(match), "setup") {
						return match, nil
					}
				}
			}
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no built package found for platform %s in %s", platform, distPath)
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Get scenario desktop status handler - discovers all scenarios and their desktop deployment status
func (s *Server) getScenarioDesktopStatusHandler(w http.ResponseWriter, r *http.Request) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		// Fallback to calculating from current directory
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}

	scenariosPath := filepath.Join(vrooliRoot, "scenarios")
	entries, err := os.ReadDir(scenariosPath)
	if err != nil {
		s.logger.Error("failed to read scenarios directory",
			"path", scenariosPath,
			"error", err)
		http.Error(w, "Failed to read scenarios directory", http.StatusInternalServerError)
		return
	}

	type ScenarioDesktopStatus struct {
		Name            string   `json:"name"`
		DisplayName     string   `json:"display_name,omitempty"`
		HasDesktop      bool     `json:"has_desktop"`
		DesktopPath     string   `json:"desktop_path,omitempty"`
		Version         string   `json:"version,omitempty"`
		Platforms       []string `json:"platforms,omitempty"`
		Built           bool     `json:"built,omitempty"`
		DistPath        string   `json:"dist_path,omitempty"`
		LastModified    string   `json:"last_modified,omitempty"`
		PackageSize     int64    `json:"package_size,omitempty"`
	}

	var scenarios []ScenarioDesktopStatus

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		electronPath := filepath.Join(scenariosPath, scenarioName, "platforms", "electron")

		status := ScenarioDesktopStatus{
			Name: scenarioName,
		}

		// Check if platforms/electron exists
		if electronInfo, err := os.Stat(electronPath); err == nil && electronInfo.IsDir() {
			status.HasDesktop = true
			status.DesktopPath = electronPath

			// Read package.json for details
			pkgPath := filepath.Join(electronPath, "package.json")
			if data, err := os.ReadFile(pkgPath); err == nil {
				var pkg map[string]interface{}
				if json.Unmarshal(data, &pkg) == nil {
					if name, ok := pkg["name"].(string); ok {
						status.DisplayName = name
					}
					if version, ok := pkg["version"].(string); ok {
						status.Version = version
					}
				}
			}

			// Check if dist-electron exists (built packages)
			distPath := filepath.Join(electronPath, "dist-electron")
			if distInfo, err := os.Stat(distPath); err == nil && distInfo.IsDir() {
				status.Built = true
				status.DistPath = distPath
				status.LastModified = distInfo.ModTime().Format("2006-01-02 15:04:05")

				// Calculate total size of dist directory
				var totalSize int64
				filepath.Walk(distPath, func(_ string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					if !info.IsDir() {
						totalSize += info.Size()
					}
					return nil
				})
				status.PackageSize = totalSize

				// Try to detect which platforms were built by looking at dist files
				distEntries, _ := os.ReadDir(distPath)
				for _, de := range distEntries {
					name := de.Name()
					if strings.Contains(name, ".exe") || strings.Contains(name, "win") {
						status.Platforms = append(status.Platforms, "win")
					} else if strings.Contains(name, ".dmg") || strings.Contains(name, "mac") {
						status.Platforms = append(status.Platforms, "mac")
					} else if strings.Contains(name, ".AppImage") || strings.Contains(name, "linux") {
						status.Platforms = append(status.Platforms, "linux")
					}
				}
				// Remove duplicates
				status.Platforms = uniqueStrings(status.Platforms)
			}
		}

		scenarios = append(scenarios, status)
	}

	// Count statistics
	withDesktop := 0
	withBuilt := 0
	for _, s := range scenarios {
		if s.HasDesktop {
			withDesktop++
		}
		if s.Built {
			withBuilt++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"stats": map[string]int{
			"total":        len(scenarios),
			"with_desktop": withDesktop,
			"built":        withBuilt,
			"web_only":     len(scenarios) - withDesktop,
		},
	})
}

// Helper function to remove duplicate strings
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, val := range slice {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: Get allowed origin from environment - required for production
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			// SECURITY: In development, UI_PORT must be provided by lifecycle system
			// In production, ALLOWED_ORIGIN should be explicitly configured
			uiPort := os.Getenv("UI_PORT")
			if uiPort != "" {
				// UI_PORT is set - use it to build allowed origins
				origin := r.Header.Get("Origin")
				allowedOrigins := []string{
					"http://localhost:" + uiPort,
					"http://127.0.0.1:" + uiPort,
				}

				// Check if origin matches any allowed origin
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						allowedOrigin = origin
						break
					}
				}

				// If no match found, use primary UI port
				if allowedOrigin == "" {
					allowedOrigin = "http://localhost:" + uiPort
				}
			} else {
				// SECURITY: Neither ALLOWED_ORIGIN nor UI_PORT is set
				// This is a configuration error - log and use restrictive CORS
				globalLogger.Warn("CORS configuration missing",
					"message", "Neither ALLOWED_ORIGIN nor UI_PORT is set",
					"action", "using restrictive localhost-only CORS for security")
				// Set to request origin only if it's localhost
				origin := r.Header.Get("Origin")
				if origin != "" && (origin == "http://localhost" || origin == "http://127.0.0.1") {
					allowedOrigin = origin
				} else {
					allowedOrigin = "http://localhost" // Minimal fallback
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Build-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
}

// Main function
func main() {
	// SECURITY: Validate required environment variable - fail fast if not set
	lifecycleManaged := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
	if lifecycleManaged != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-to-desktop

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// SECURITY: Validate port environment variables - prefer API_PORT, fallback to PORT
	port := 15200
	apiPortStr := os.Getenv("API_PORT")
	portStr := os.Getenv("PORT")

	if apiPortStr != "" {
		p, err := strconv.Atoi(apiPortStr)
		if err != nil {
			log.Fatalf("❌ Invalid API_PORT value '%s': must be a valid integer", apiPortStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid API_PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	} else if portStr != "" {
		// Fallback to PORT for compatibility
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("❌ Invalid PORT value '%s': must be a valid integer", portStr)
		}
		if p < 1024 || p > 65535 {
			log.Fatalf("❌ Invalid PORT value %d: must be between 1024 and 65535", p)
		}
		port = p
	} else {
		globalLogger.Warn("no port configuration found",
			"message", "No API_PORT or PORT environment variable set",
			"action", "using default port",
			"default_port", port)
	}

	// Create and start server
	server := NewServer(port)
	if err := server.Start(); err != nil {
		globalLogger.Error("server failed", "error", err)
		log.Fatal(err)
	}
}
