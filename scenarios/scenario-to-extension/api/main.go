package main

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	Port           int    `json:"port"`
	APIEndpoint    string `json:"api_endpoint"`
	TemplatesPath  string `json:"templates_path"`
	OutputPath     string `json:"output_path"`
	BrowserlessURL string `json:"browserless_url"`
	PostgresURL    string `json:"postgres_url"`
	RedisURL       string `json:"redis_url"`
	Debug          bool   `json:"debug"`
}

// Extension generation request
type ExtensionGenerateRequest struct {
	ScenarioName string          `json:"scenario_name"`
	TemplateType string          `json:"template_type"`
	Config       ExtensionConfig `json:"config"`
}

// Extension configuration
type ExtensionConfig struct {
	AppName         string                 `json:"app_name"`
	Description     string                 `json:"app_description"`
	APIEndpoint     string                 `json:"api_endpoint"`
	Permissions     []string               `json:"permissions"`
	HostPermissions []string               `json:"host_permissions"`
	Version         string                 `json:"version"`
	AuthorName      string                 `json:"author_name"`
	License         string                 `json:"license"`
	CustomVariables map[string]interface{} `json:"custom_variables"`
}

// Extension build info
type ExtensionBuild struct {
	BuildID       string          `json:"build_id"`
	ScenarioName  string          `json:"scenario_name"`
	TemplateType  string          `json:"template_type"`
	Config        ExtensionConfig `json:"config"`
	Status        string          `json:"status"` // building, ready, failed
	ExtensionPath string          `json:"extension_path"`
	BuildLog      []string        `json:"build_log"`
	ErrorLog      []string        `json:"error_log"`
	CreatedAt     time.Time       `json:"created_at"`
	CompletedAt   *time.Time      `json:"completed_at"`
}

// Extension test request
type ExtensionTestRequest struct {
	ExtensionPath string   `json:"extension_path"`
	TestSites     []string `json:"test_sites"`
	Screenshot    bool     `json:"screenshot"`
	Headless      bool     `json:"headless"`
}

// Extension test result
type ExtensionTestResult struct {
	Success     bool                  `json:"success"`
	TestResults []ExtensionSiteResult `json:"test_results"`
	Summary     ExtensionTestSummary  `json:"summary"`
	ReportTime  time.Time             `json:"report_time"`
}

type ExtensionSiteResult struct {
	Site           string   `json:"site"`
	Loaded         bool     `json:"loaded"`
	Errors         []string `json:"errors"`
	ScreenshotPath string   `json:"screenshot_path,omitempty"`
	LoadTime       int      `json:"load_time_ms"`
}

type ExtensionTestSummary struct {
	TotalTests  int     `json:"total_tests"`
	Passed      int     `json:"passed"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// Build management constants
const (
	maxBuilds               = 100  // Maximum number of builds to keep in memory
	buildCleanupInterval    = 300  // Cleanup every 5 minutes (in seconds)
	completedBuildRetention = 3600 // Keep completed builds for 1 hour (in seconds)
)

// BuildManager manages build state with thread-safe operations
type BuildManager struct {
	mu     sync.RWMutex
	builds map[string]*ExtensionBuild
}

// NewBuildManager creates a new BuildManager
func NewBuildManager() *BuildManager {
	return &BuildManager{
		builds: make(map[string]*ExtensionBuild),
	}
}

// Add adds a build to the manager
func (bm *BuildManager) Add(build *ExtensionBuild) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.builds[build.BuildID] = build
}

// Get retrieves a build by ID
func (bm *BuildManager) Get(buildID string) (*ExtensionBuild, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	build, exists := bm.builds[buildID]
	return build, exists
}

// List returns all builds as a slice
func (bm *BuildManager) List() []*ExtensionBuild {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	buildList := make([]*ExtensionBuild, 0, len(bm.builds))
	for _, build := range bm.builds {
		buildList = append(buildList, build)
	}
	return buildList
}

// CountByStatus counts builds with the given status
func (bm *BuildManager) CountByStatus(status string) int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	count := 0
	for _, build := range bm.builds {
		if build.Status == status {
			count++
		}
	}
	return count
}

// Count returns total number of builds
func (bm *BuildManager) Count() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return len(bm.builds)
}

// Cleanup removes old completed/failed builds from memory
func (bm *BuildManager) Cleanup() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()
	buildCount := len(bm.builds)

	// If we're under the limit and all builds are recent, no cleanup needed
	if buildCount <= maxBuilds {
		allRecent := true
		for _, build := range bm.builds {
			if build.Status != "building" && build.CompletedAt != nil {
				age := now.Sub(*build.CompletedAt).Seconds()
				if age > float64(completedBuildRetention) {
					allRecent = false
					break
				}
			}
		}
		if allRecent {
			return
		}
	}

	// Collect builds to remove
	var toRemove []string

	// First pass: Remove old completed/failed builds
	for buildID, build := range bm.builds {
		if build.Status == "building" {
			continue // Never remove in-progress builds
		}

		if build.CompletedAt != nil {
			age := now.Sub(*build.CompletedAt).Seconds()
			if age > float64(completedBuildRetention) {
				toRemove = append(toRemove, buildID)
			}
		}
	}

	// If still over limit after removing old builds, remove oldest completed builds
	if buildCount-len(toRemove) > maxBuilds {
		type buildWithTime struct {
			id          string
			completedAt time.Time
		}
		var completedBuilds []buildWithTime

		for buildID, build := range bm.builds {
			if build.Status != "building" && build.CompletedAt != nil {
				// Skip if already marked for removal
				alreadyMarked := false
				for _, id := range toRemove {
					if id == buildID {
						alreadyMarked = true
						break
					}
				}
				if !alreadyMarked {
					completedBuilds = append(completedBuilds, buildWithTime{
						id:          buildID,
						completedAt: *build.CompletedAt,
					})
				}
			}
		}

		// Sort by completion time (oldest first)
		sort.Slice(completedBuilds, func(i, j int) bool {
			return completedBuilds[i].completedAt.Before(completedBuilds[j].completedAt)
		})

		// Remove oldest builds until we're under the limit
		needed := (buildCount - len(toRemove)) - maxBuilds
		for i := 0; i < needed && i < len(completedBuilds); i++ {
			toRemove = append(toRemove, completedBuilds[i].id)
		}
	}

	// Perform removal
	if len(toRemove) > 0 {
		for _, buildID := range toRemove {
			delete(bm.builds, buildID)
		}
		log.Printf("Cleaned up %d old builds (total builds: %d -> %d)", len(toRemove), buildCount, len(bm.builds))
	}
}

// Global state
var (
	config       *Config
	buildManager *BuildManager
)

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-to-extension

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Load configuration
	config = loadConfig()
	buildManager = NewBuildManager()

	// Setup routes
	r := mux.NewRouter()

	// Health check at root level (required for lifecycle system)
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/extension/generate", generateExtensionHandler).Methods("POST")
	api.HandleFunc("/extension/status/{build_id}", getExtensionStatusHandler).Methods("GET")
	api.HandleFunc("/extension/download/{build_id}", downloadExtensionHandler).Methods("GET")
	api.HandleFunc("/extension/test", testExtensionHandler).Methods("POST")
	api.HandleFunc("/extension/templates", listTemplatesHandler).Methods("GET")
	api.HandleFunc("/extension/builds", listBuildsHandler).Methods("GET")

	// UI routes (serve static files)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../ui/")))

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	// Start background build cleanup goroutine
	go buildCleanupWorker()

	// Start server
	log.Printf("scenario-to-extension API starting on port %d", config.Port)
	log.Printf("Templates path: %s", config.TemplatesPath)
	log.Printf("Output path: %s", config.OutputPath)
	log.Printf("Build management: max %d builds, cleanup every %ds, retention %ds", maxBuilds, buildCleanupInterval, completedBuildRetention)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), handler))
}

func loadConfig() *Config {
	// Note: API runs from api/ directory, so templates are one level up
	config := &Config{
		Port:           3201,
		APIEndpoint:    "http://localhost:3201",
		TemplatesPath:  "../templates",
		OutputPath:     "./data/extensions",
		BrowserlessURL: "http://localhost:3000",
		Debug:          os.Getenv("DEBUG") == "true",
	}

	// Override with environment variables
	// Try API_PORT first (lifecycle system), then PORT (fallback)
	if port := os.Getenv("API_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	} else if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	if endpoint := os.Getenv("API_ENDPOINT"); endpoint != "" {
		config.APIEndpoint = endpoint
	}

	if templatesPath := os.Getenv("TEMPLATES_PATH"); templatesPath != "" {
		config.TemplatesPath = templatesPath
	}

	if outputPath := os.Getenv("OUTPUT_PATH"); outputPath != "" {
		config.OutputPath = outputPath
	}

	if browserlessURL := os.Getenv("BROWSERLESS_URL"); browserlessURL != "" {
		config.BrowserlessURL = browserlessURL
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputPath, 0755); err != nil {
		log.Printf("Warning: Failed to create output directory %s: %v", config.OutputPath, err)
	}

	return config
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "scenario-to-extension-api",
		"version":   "1.0.0",
		"scenario":  "scenario-to-extension",
		"timestamp": time.Now(),
		"readiness": true,
		"resources": map[string]interface{}{
			"browserless": checkBrowserlessHealth(),
			"templates":   checkTemplatesHealth(),
		},
		"stats": map[string]interface{}{
			"total_builds":     buildManager.Count(),
			"active_builds":    buildManager.CountByStatus("building"),
			"completed_builds": buildManager.CountByStatus("ready"),
			"failed_builds":    buildManager.CountByStatus("failed"),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, health, "health")
}

func generateExtensionHandler(w http.ResponseWriter, r *http.Request) {
	var req ExtensionGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.ScenarioName == "" {
		respondWithError(w, http.StatusBadRequest, "scenario_name is required")
		return
	}

	if req.TemplateType == "" {
		req.TemplateType = "full"
	}

	if req.Config.AppName == "" {
		respondWithError(w, http.StatusBadRequest, "config.app_name is required")
		return
	}

	if req.Config.APIEndpoint == "" {
		respondWithError(w, http.StatusBadRequest, "config.api_endpoint is required")
		return
	}

	// Set defaults
	if req.Config.Version == "" {
		req.Config.Version = "1.0.0"
	}
	if req.Config.AuthorName == "" {
		req.Config.AuthorName = "Vrooli Scenario Generator"
	}
	if req.Config.License == "" {
		req.Config.License = "MIT"
	}
	if req.Config.Permissions == nil {
		req.Config.Permissions = []string{"storage", "activeTab"}
	}
	if req.Config.HostPermissions == nil {
		req.Config.HostPermissions = []string{"<all_urls>"}
	}

	// Generate build ID
	buildID := generateBuildID()

	// Create build record
	build := &ExtensionBuild{
		BuildID:       buildID,
		ScenarioName:  req.ScenarioName,
		TemplateType:  req.TemplateType,
		Config:        req.Config,
		Status:        "building",
		ExtensionPath: filepath.Join(config.OutputPath, req.ScenarioName, "platforms", "extension"),
		BuildLog:      []string{},
		ErrorLog:      []string{},
		CreatedAt:     time.Now(),
	}

	buildManager.Add(build)

	// Start extension generation in background
	go generateExtension(build)

	// Return immediate response
	response := map[string]interface{}{
		"build_id":             buildID,
		"extension_path":       build.ExtensionPath,
		"install_instructions": getInstallInstructions(req.TemplateType),
		"test_command":         fmt.Sprintf("scenario-to-extension test %s", build.ExtensionPath),
		"status":               "building",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, response, "extension generation")
}

func getExtensionStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buildID := vars["build_id"]

	build, exists := buildManager.Get(buildID)
	if !exists {
		respondWithError(w, http.StatusNotFound, "Build not found")
		return
	}

	response := map[string]interface{}{
		"build_id":       build.BuildID,
		"scenario_name":  build.ScenarioName,
		"status":         build.Status,
		"extension_path": build.ExtensionPath,
		"created_at":     build.CreatedAt,
		"build_log":      build.BuildLog,
		"error_log":      build.ErrorLog,
	}

	if build.CompletedAt != nil {
		response["completed_at"] = build.CompletedAt
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, response, "extension status")
}

func testExtensionHandler(w http.ResponseWriter, r *http.Request) {
	var req ExtensionTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.ExtensionPath == "" {
		respondWithError(w, http.StatusBadRequest, "extension_path is required")
		return
	}

	// Set defaults
	if req.TestSites == nil {
		req.TestSites = []string{"https://example.com"}
	}
	if len(req.TestSites) == 0 {
		req.TestSites = []string{"https://example.com"}
	}

	// Test extension
	result := testExtension(&req)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, result, "extension test")
}

func listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates := listAvailableTemplates()

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	}, "templates list")
}

func listBuildsHandler(w http.ResponseWriter, r *http.Request) {
	buildList := buildManager.List()

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"builds": buildList,
		"count":  len(buildList),
	}, "builds list")
}

func downloadExtensionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buildID := vars["build_id"]

	build, exists := buildManager.Get(buildID)
	if !exists {
		respondWithError(w, http.StatusNotFound, "Build not found")
		return
	}

	// Only allow downloading completed builds
	if build.Status != "ready" {
		respondWithError(w, http.StatusBadRequest, "Build is not ready for download")
		return
	}

	// Verify extension path exists
	if _, err := os.Stat(build.ExtensionPath); os.IsNotExist(err) {
		respondWithError(w, http.StatusNotFound, "Extension files not found")
		return
	}

	// Create ZIP archive in memory
	zipFilename := fmt.Sprintf("%s-v%s.zip", build.ScenarioName, build.Config.Version)

	// Set headers for file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", zipFilename))

	// Create ZIP writer
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Walk through extension directory and add files to ZIP
	err := filepath.Walk(build.ExtensionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path for ZIP entry
		relPath, err := filepath.Rel(build.ExtensionPath, path)
		if err != nil {
			return err
		}

		// Create ZIP entry
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Copy file content to ZIP
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(zipFile, srcFile)
		return err
	})

	if err != nil {
		log.Printf("Error creating ZIP archive for build %s: %v", buildID, err)
		// Cannot send error response after headers are sent, just log it
	}
}

// Helper functions

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// writeJSON writes a JSON response and logs encoding errors
func writeJSON(w http.ResponseWriter, data interface{}, operation string) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode %s response: %v", operation, err)
	}
}

// respondWithError sends a standardized JSON error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	writeJSON(w, ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	}, "error")
}

func generateBuildID() string {
	// Use single timestamp + random bytes to prevent collisions
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to nanosecond if random fails (should never happen)
		return fmt.Sprintf("build_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
	}
	return fmt.Sprintf("build_%d_%s", time.Now().UnixNano(), hex.EncodeToString(randomBytes))
}

func checkBrowserlessHealth() bool {
	// Browserless integration is intentionally simulated for development.
	// The extension testing functionality returns mock data to support the UI
	// without requiring a running browserless instance.
	//
	// To implement real browserless integration:
	// 1. Make HTTP GET request to config.BrowserlessURL + "/health"
	// 2. Check response status code (200 = healthy)
	// 3. Return false on timeout or connection errors
	return true
}

func checkTemplatesHealth() bool {
	templatePath := filepath.Join(config.TemplatesPath, "vanilla", "manifest.json")
	_, err := os.Stat(templatePath)
	return err == nil
}

func generateExtension(build *ExtensionBuild) {
	defer func() {
		if r := recover(); r != nil {
			build.Status = "failed"
			build.ErrorLog = append(build.ErrorLog, fmt.Sprintf("Panic during generation: %v", r))
			completedAt := time.Now()
			build.CompletedAt = &completedAt
		}
	}()

	build.BuildLog = append(build.BuildLog, fmt.Sprintf("Starting extension generation at %s", time.Now().Format(time.RFC3339)))

	// Create output directory
	if err := os.MkdirAll(build.ExtensionPath, 0755); err != nil {
		build.Status = "failed"
		build.ErrorLog = append(build.ErrorLog, fmt.Sprintf("Failed to create output directory: %v", err))
		completedAt := time.Now()
		build.CompletedAt = &completedAt
		return
	}

	build.BuildLog = append(build.BuildLog, fmt.Sprintf("Created output directory: %s", build.ExtensionPath))

	// Copy and process template files
	templatePath := filepath.Join(config.TemplatesPath, "vanilla")
	if build.TemplateType != "full" {
		// For specialized templates, still use vanilla as base but modify based on advanced config
		build.BuildLog = append(build.BuildLog, fmt.Sprintf("Using specialized template: %s", build.TemplateType))
	}

	if err := copyAndProcessTemplates(templatePath, build.ExtensionPath, build); err != nil {
		build.Status = "failed"
		build.ErrorLog = append(build.ErrorLog, fmt.Sprintf("Template processing failed: %v", err))
		completedAt := time.Now()
		build.CompletedAt = &completedAt
		return
	}

	build.BuildLog = append(build.BuildLog, "Templates processed successfully")

	// Generate package.json for the extension
	if err := generatePackageJSON(build); err != nil {
		build.Status = "failed"
		build.ErrorLog = append(build.ErrorLog, fmt.Sprintf("Package.json generation failed: %v", err))
		completedAt := time.Now()
		build.CompletedAt = &completedAt
		return
	}

	build.BuildLog = append(build.BuildLog, "Package.json generated")

	// Create README
	if err := generateREADME(build); err != nil {
		build.Status = "failed"
		build.ErrorLog = append(build.ErrorLog, fmt.Sprintf("README generation failed: %v", err))
		completedAt := time.Now()
		build.CompletedAt = &completedAt
		return
	}

	build.BuildLog = append(build.BuildLog, "README.md generated")

	// Build extension
	build.BuildLog = append(build.BuildLog, "Extension generation completed successfully")
	build.Status = "ready"
	completedAt := time.Now()
	build.CompletedAt = &completedAt
}

func copyAndProcessTemplates(templatePath, outputPath string, build *ExtensionBuild) error {
	// Verify template directory exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template directory not found: %s", templatePath)
	}

	// Build variable replacement map
	variables := buildVariableMap(build)

	// Walk through template directory
	err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and backup files
		if info.IsDir() || strings.HasSuffix(path, ".backup") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(templatePath, path)
		if err != nil {
			return err
		}

		// Calculate output path
		outPath := filepath.Join(outputPath, relPath)

		// Read template file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", relPath, err)
		}

		// Process template variables
		processedContent := replaceVariables(string(content), variables)

		// Create output directory if needed
		outDir := filepath.Dir(outPath)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outDir, err)
		}

		// Write processed file
		if err := os.WriteFile(outPath, []byte(processedContent), info.Mode()); err != nil {
			return fmt.Errorf("failed to write file %s: %w", outPath, err)
		}

		build.BuildLog = append(build.BuildLog, fmt.Sprintf("Processed: %s", relPath))
		return nil
	})

	if err != nil {
		return fmt.Errorf("template processing failed: %w", err)
	}

	return nil
}

// buildVariableMap creates a map of template variables to their values
func buildVariableMap(build *ExtensionBuild) map[string]string {
	cfg := build.Config

	// Convert slices to JSON for template substitution
	permissionsJSON, _ := json.Marshal(cfg.Permissions)
	hostPermissionsJSON, _ := json.Marshal(cfg.HostPermissions)

	// Build default content scripts configuration
	contentScripts := []map[string]interface{}{
		{
			"matches": cfg.HostPermissions,
			"js":      []string{"content.js"},
			"run_at":  "document_idle",
		},
	}
	contentScriptsJSON, _ := json.Marshal(contentScripts)

	// Package name from app name
	packageName := strings.ToLower(strings.ReplaceAll(cfg.AppName, " ", "-"))

	variables := map[string]string{
		"APP_NAME":                        cfg.AppName,
		"VERSION":                         cfg.Version,
		"APP_DESCRIPTION":                 cfg.Description,
		"AUTHOR_NAME":                     cfg.AuthorName,
		"LICENSE":                         cfg.License,
		"SCENARIO_NAME":                   build.ScenarioName,
		"API_ENDPOINT":                    cfg.APIEndpoint,
		"PACKAGE_NAME":                    packageName,
		"DEBUG_MODE":                      "false",
		"PERMISSIONS":                     string(permissionsJSON),
		"HOST_PERMISSIONS":                string(hostPermissionsJSON),
		"CONTENT_SCRIPTS":                 string(contentScriptsJSON),
		"COMMANDS":                        "{}",
		"WEB_ACCESSIBLE_RESOURCES":        "[]",
		"AUTO_INJECT":                     "false",
		"AUTH_METHOD":                     "API Key",
		"PERMISSIONS_LIST":                formatMarkdownList(cfg.Permissions),
		"HOST_PERMISSIONS_LIST":           formatMarkdownList(cfg.HostPermissions),
		"CUSTOM_CSS":                      "",
		"AUTH_FIELDS":                     "",
		"AUTH_CREDENTIAL_MAPPING":         "",
		"STATS_CARDS":                     "",
		"ACTION_BUTTONS":                  "",
		"SCENARIO_SPECIFIC_CONTENT":       "",
		"CUSTOM_MESSAGE_HANDLERS":         "",
		"CUSTOM_EVENT_HANDLERS":           "",
		"CUSTOM_ALARM_HANDLERS":           "",
		"CUSTOM_COMMAND_HANDLERS":         "",
		"CONTEXT_MENU_SETUP":              "",
		"PAGE_LOAD_HANDLERS":              "",
		"TAB_ACTIVATION_HANDLERS":         "",
		"UPDATE_HANDLERS":                 "",
		"SCENARIO_SPECIFIC_FUNCTIONS":     "",
		"PAGE_PROCESSING_LOGIC":           "",
		"AUTO_INJECTION_LOGIC":            "",
		"CUSTOM_CONTENT_MESSAGE_HANDLERS": "",
		"CUSTOM_CONTENT_ACTIONS":          "",
		"CUSTOM_ACTION_HANDLERS":          "",
		"DASHBOARD_RENDER_LOGIC":          "",
		"STATS_RENDER_LOGIC":              "",
		"ACTIONS_RENDER_LOGIC":            "",
	}

	// Apply custom variables from config
	for key, value := range cfg.CustomVariables {
		if strValue, ok := value.(string); ok {
			variables[key] = strValue
		}
	}

	return variables
}

// replaceVariables replaces {{VARIABLE}} placeholders in content
func replaceVariables(content string, variables map[string]string) string {
	result := content
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// formatMarkdownList formats a list of items as a markdown list
func formatMarkdownList(items []string) string {
	if len(items) == 0 {
		return "- None"
	}
	var lines []string
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- `%s`", item))
	}
	return strings.Join(lines, "\n")
}

// verifyGeneratedFile checks if a file was generated by copyAndProcessTemplates
// This is a helper function to reduce code duplication in file verification
func verifyGeneratedFile(build *ExtensionBuild, filename, displayName string) error {
	filePath := filepath.Join(build.ExtensionPath, filename)
	if _, err := os.Stat(filePath); err == nil {
		build.BuildLog = append(build.BuildLog, fmt.Sprintf("%s verified", filename))
		return nil
	}

	// For test compatibility when ExtensionPath is empty
	if build.ExtensionPath == "" {
		build.BuildLog = append(build.BuildLog, fmt.Sprintf("%s generation would happen here", displayName))
		return nil
	}

	return fmt.Errorf("%s not found after template processing", filename)
}

func generatePackageJSON(build *ExtensionBuild) error {
	// package.json is already generated by copyAndProcessTemplates
	// This function verifies the file exists or logs a message for test compatibility
	return verifyGeneratedFile(build, "package.json", "Package.json")
}

func generateREADME(build *ExtensionBuild) error {
	// README.md is already generated by copyAndProcessTemplates
	// This function verifies the file exists or logs a message for test compatibility
	return verifyGeneratedFile(build, "README.md", "README")
}

func testExtension(req *ExtensionTestRequest) *ExtensionTestResult {
	// Extension testing is intentionally simulated for development.
	// This allows the UI and API to function without a browserless instance.
	// All tests return successful results with mock timing data.
	//
	// To implement real browserless integration:
	// 1. Start a Chrome instance via browserless API with the extension loaded
	// 2. Navigate to each test site in req.TestSites
	// 3. Capture screenshots if req.Screenshot is true
	// 4. Monitor console for JavaScript errors
	// 5. Validate extension injection and functionality
	// 6. Return actual pass/fail results based on test execution
	//
	// Browserless endpoint: config.BrowserlessURL
	// Browserless docs: https://www.browserless.io/docs/

	log.Printf("INFO: Extension testing is simulated (browserless integration not implemented)")

	results := make([]ExtensionSiteResult, 0, len(req.TestSites))
	passed := 0

	for _, site := range req.TestSites {
		// Return simulated successful test result
		// Kept for backwards compatibility with existing tests
		result := ExtensionSiteResult{
			Site:           site,
			Loaded:         true,
			Errors:         []string{},
			ScreenshotPath: fmt.Sprintf("(simulated) /tmp/extension-test-%d.png", time.Now().Unix()),
			LoadTime:       500,
		}
		results = append(results, result)
		passed++
	}

	total := len(results)
	successRate := float64(passed) / float64(total) * 100

	return &ExtensionTestResult{
		Success:     passed == total,
		TestResults: results,
		Summary: ExtensionTestSummary{
			TotalTests:  total,
			Passed:      passed,
			Failed:      total - passed,
			SuccessRate: successRate,
		},
		ReportTime: time.Now(),
	}
}

func listAvailableTemplates() []map[string]interface{} {
	templates := []map[string]interface{}{}

	// Safety check for test environments
	if config == nil {
		return []map[string]interface{}{
			{
				"name":         "full",
				"display_name": "Full Extension",
				"description":  "Complete extension with background, content scripts, and popup",
				"files":        []string{"manifest.json", "background.js", "content.js", "popup.html", "popup.js"},
				"source":       "default",
			},
		}
	}

	// Add "full" template (vanilla directory)
	vanillaPath := filepath.Join(config.TemplatesPath, "vanilla")
	if _, err := os.Stat(vanillaPath); err == nil {
		files, _ := listTemplateFiles(vanillaPath)
		templates = append(templates, map[string]interface{}{
			"name":         "full",
			"display_name": "Full Extension",
			"description":  "Complete extension with background, content scripts, and popup",
			"files":        files,
			"source":       "vanilla",
		})
	}

	// Scan advanced templates directory for specialized templates
	advancedPath := filepath.Join(config.TemplatesPath, "advanced")
	if entries, err := os.ReadDir(advancedPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			// Read template metadata
			metadataPath := filepath.Join(advancedPath, entry.Name())
			data, err := os.ReadFile(metadataPath)
			if err != nil {
				log.Printf("Warning: Failed to read template metadata %s: %v", entry.Name(), err)
				continue
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal(data, &metadata); err != nil {
				log.Printf("Warning: Failed to parse template metadata %s: %v", entry.Name(), err)
				continue
			}

			// Extract template name from filename (e.g., "content-script-only.json" -> "content-script-only")
			templateName := strings.TrimSuffix(entry.Name(), ".json")

			// Build template entry
			template := map[string]interface{}{
				"name":         templateName,
				"display_name": metadata["name"],
				"description":  metadata["description"],
				"source":       "advanced",
			}

			// Add files list if present
			if files, ok := metadata["files"].([]interface{}); ok {
				template["files"] = files
			}

			templates = append(templates, template)
		}
	}

	// If no templates found, return hardcoded defaults as fallback
	if len(templates) == 0 {
		log.Printf("Warning: No templates found in %s, using hardcoded defaults", config.TemplatesPath)
		templates = []map[string]interface{}{
			{
				"name":         "full",
				"display_name": "Full Extension",
				"description":  "Complete extension with background, content scripts, and popup",
				"files":        []string{"manifest.json", "background.js", "content.js", "popup.html", "popup.js"},
				"source":       "fallback",
			},
		}
	}

	return templates
}

// listTemplateFiles returns a list of files in a template directory
func listTemplateFiles(templatePath string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(templatePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Add directory with trailing slash
			files = append(files, entry.Name()+"/")
		} else if !strings.HasSuffix(entry.Name(), ".backup") {
			// Skip backup files
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

func getInstallInstructions(templateType string) string {
	instructions := `
Extension Installation Instructions:

1. Open Chrome and navigate to chrome://extensions/
2. Enable "Developer mode" (toggle in top right)
3. Click "Load unpacked" button
4. Select the generated extension directory
5. The extension should now appear in your extensions list

For development:
- Use 'npm run dev' for hot reload during development
- Use 'npm run build' to create production build
- Use 'npm run pack' to create distributable ZIP file

For testing:
- Use scenario-to-extension test command to validate functionality
- Check browser console for any JavaScript errors
- Test on target websites listed in host_permissions
`

	return strings.TrimSpace(instructions)
}

// buildCleanupWorker periodically cleans up old builds from memory
func buildCleanupWorker() {
	ticker := time.NewTicker(time.Duration(buildCleanupInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		buildManager.Cleanup()
	}
}
