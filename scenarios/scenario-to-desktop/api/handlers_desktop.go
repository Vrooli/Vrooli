package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

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
		request.TemplateType = "universal" // Default to universal template (works for any scenario)
	}

	// Validate template type
	validTemplates := []string{"universal", "basic", "advanced", "multi_window", "kiosk"} // basic is alias for universal
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
		BuildID:            buildID,
		ScenarioName:       scenarioName,
		Status:             "building",
		OutputPath:         desktopPath,
		CreatedAt:          time.Now(),
		Platforms:          options.Platforms,           // Legacy field (will be updated with successful builds)
		RequestedPlatforms: options.Platforms,           // NEW: Track what was requested
		PlatformResults:    platformResults,
		BuildLog:           []string{},
		ErrorLog:           []string{},
		Artifacts:          make(map[string]string),
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
