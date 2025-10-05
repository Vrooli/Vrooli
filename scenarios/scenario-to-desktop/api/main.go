package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

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

// BuildStatus represents the status of a desktop application build
type BuildStatus struct {
	BuildID      string                 `json:"build_id"`
	ScenarioName string                 `json:"scenario_name"`
	Status       string                 `json:"status"` // building, ready, failed
	Framework    string                 `json:"framework"`
	TemplateType string                 `json:"template_type"`
	Platforms    []string               `json:"platforms"`
	OutputPath   string                 `json:"output_path"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	ErrorLog     []string               `json:"error_log,omitempty"`
	BuildLog     []string               `json:"build_log,omitempty"`
	Artifacts    map[string]string      `json:"artifacts,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
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
	templateDir   string
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	server := &Server{
		router:        mux.NewRouter(),
		port:          port,
		buildStatuses: make(map[string]*BuildStatus),
		templateDir:   "./templates",
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/api/v1/health", s.healthHandler).Methods("GET")

	// System status
	s.router.HandleFunc("/api/v1/status", s.statusHandler).Methods("GET")

	// Template management
	s.router.HandleFunc("/api/v1/templates", s.listTemplatesHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/templates/{type}", s.getTemplateHandler).Methods("GET")

	// Desktop application operations
	s.router.HandleFunc("/api/v1/desktop/generate", s.generateDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/status/{build_id}", s.getBuildStatusHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/desktop/build", s.buildDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/test", s.testDesktopHandler).Methods("POST")
	s.router.HandleFunc("/api/v1/desktop/package", s.packageDesktopHandler).Methods("POST")

	// Webhook endpoints
	s.router.HandleFunc("/api/v1/desktop/webhook/build-complete", s.buildCompleteWebhookHandler).Methods("POST")

	// Setup CORS
	s.router.Use(corsMiddleware)
	s.router.Use(loggingMiddleware)
}

// Start starts the server
func (s *Server) Start() error {
	log.Printf("Starting scenario-to-desktop API server on port %d", s.port)

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
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Desktop API server started at http://localhost:%d", s.port)
	log.Printf("Health check: http://localhost:%d/api/v1/health", s.port)
	log.Printf("API documentation: http://localhost:%d/api/v1/status", s.port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

// Health check handler
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "scenario-to-desktop",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
		"uptime":    "unknown", // Could be tracked if needed
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

	// Read template configuration file
	templatePath := filepath.Join(s.templateDir, "advanced", templateType+".json")

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
	s.buildStatuses[buildID] = buildStatus

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

	status, exists := s.buildStatuses[buildID]
	if !exists {
		http.Error(w, "Build not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
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

	s.buildStatuses[buildID] = buildStatus

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
	if status, exists := s.buildStatuses[buildID]; exists {
		if resultStatus, ok := result["status"].(string); ok {
			status.Status = resultStatus
		}
		if resultStatus := result["status"].(string); resultStatus == "completed" || resultStatus == "failed" {
			now := time.Now()
			status.CompletedAt = &now
		}
	}

	log.Printf("Build webhook received for %s: %v", buildID, result)

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
	if config.OutputPath == "" {
		return fmt.Errorf("output_path is required")
	}

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
	status := s.buildStatuses[buildID]

	defer func() {
		if r := recover(); r != nil {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Panic: %v", r))
			now := time.Now()
			status.CompletedAt = &now
		}
	}()

	// Create configuration JSON file
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to marshal config: %v", err))
		return
	}

	// Write config to temporary file
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("desktop-config-%s.json", buildID))
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		status.Status = "failed"
		status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("Failed to write config file: %v", err))
		return
	}
	defer os.Remove(configPath)

	// Execute template generator
	cmd := exec.Command("node",
		"./templates/build-tools/dist/template-generator.js",
		configPath)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)
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
}

// Perform desktop build (async)
func (s *Server) performDesktopBuild(buildID string, request *struct {
	DesktopPath string   `json:"desktop_path"`
	Platforms   []string `json:"platforms"`
	Sign        bool     `json:"sign"`
	Publish     bool     `json:"publish"`
}) {
	status := s.buildStatuses[buildID]

	// Build steps: install dependencies, build, package
	steps := []string{"npm install", "npm run build", "npm run dist"}

	for _, step := range steps {
		cmd := exec.Command("bash", "-c", step)
		cmd.Dir = request.DesktopPath

		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		status.BuildLog = append(status.BuildLog, fmt.Sprintf("%s: %s", step, outputStr))

		if err != nil {
			status.Status = "failed"
			status.ErrorLog = append(status.ErrorLog, fmt.Sprintf("%s failed: %v", step, err))
			now := time.Now()
			status.CompletedAt = &now
			return
		}
	}

	status.Status = "ready"
	now := time.Now()
	status.CompletedAt = &now
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

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Build-ID")

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
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-to-desktop

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment or use default
	port := 3202
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// Create and start server
	server := NewServer(port)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
