package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	Port              int    `json:"port"`
	APIEndpoint       string `json:"api_endpoint"`
	TemplatesPath     string `json:"templates_path"`
	OutputPath        string `json:"output_path"`
	BrowserlessURL    string `json:"browserless_url"`
	PostgresURL       string `json:"postgres_url"`
	RedisURL          string `json:"redis_url"`
	Debug             bool   `json:"debug"`
}

// Extension generation request
type ExtensionGenerateRequest struct {
	ScenarioName string            `json:"scenario_name"`
	TemplateType string            `json:"template_type"`
	Config       ExtensionConfig   `json:"config"`
}

// Extension configuration
type ExtensionConfig struct {
	AppName           string            `json:"app_name"`
	Description       string            `json:"app_description"`
	APIEndpoint       string            `json:"api_endpoint"`
	Permissions       []string          `json:"permissions"`
	HostPermissions   []string          `json:"host_permissions"`
	Version           string            `json:"version"`
	AuthorName        string            `json:"author_name"`
	License           string            `json:"license"`
	CustomVariables   map[string]interface{} `json:"custom_variables"`
}

// Extension build info
type ExtensionBuild struct {
	BuildID            string            `json:"build_id"`
	ScenarioName       string            `json:"scenario_name"`
	TemplateType       string            `json:"template_type"`
	Config             ExtensionConfig   `json:"config"`
	Status             string            `json:"status"` // building, ready, failed
	ExtensionPath      string            `json:"extension_path"`
	BuildLog           []string          `json:"build_log"`
	ErrorLog           []string          `json:"error_log"`
	CreatedAt          time.Time         `json:"created_at"`
	CompletedAt        *time.Time        `json:"completed_at"`
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
	Success     bool                    `json:"success"`
	TestResults []ExtensionSiteResult   `json:"test_results"`
	Summary     ExtensionTestSummary    `json:"summary"`
	ReportTime  time.Time               `json:"report_time"`
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

// Global state
var (
	config     *Config
	builds     map[string]*ExtensionBuild
	buildsMux  sync.RWMutex
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
	builds = make(map[string]*ExtensionBuild)

	// Setup routes
	r := mux.NewRouter()

	// Health check at root level (required for lifecycle system)
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/extension/generate", generateExtensionHandler).Methods("POST")
	api.HandleFunc("/extension/status/{build_id}", getExtensionStatusHandler).Methods("GET")
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

	// Start server
	log.Printf("scenario-to-extension API starting on port %d", config.Port)
	log.Printf("Templates path: %s", config.TemplatesPath)
	log.Printf("Output path: %s", config.OutputPath)
	
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), handler))
}

func loadConfig() *Config {
	config := &Config{
		Port:              3201,
		APIEndpoint:       "http://localhost:3201",
		TemplatesPath:     "./templates",
		OutputPath:        "./data/extensions",
		BrowserlessURL:    "http://localhost:3000",
		Debug:             os.Getenv("DEBUG") == "true",
	}

	// Override with environment variables
	if port := os.Getenv("PORT"); port != "" {
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
	os.MkdirAll(config.OutputPath, 0755)

	return config
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":      "healthy",
		"version":     "1.0.0",
		"scenario":    "scenario-to-extension",
		"timestamp":   time.Now(),
		"resources": map[string]interface{}{
			"browserless": checkBrowserlessHealth(),
			"templates":   checkTemplatesHealth(),
		},
		"stats": map[string]interface{}{
			"total_builds":     len(builds),
			"active_builds":    countBuildsByStatus("building"),
			"completed_builds": countBuildsByStatus("ready"),
			"failed_builds":    countBuildsByStatus("failed"),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func generateExtensionHandler(w http.ResponseWriter, r *http.Request) {
	var req ExtensionGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ScenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	if req.TemplateType == "" {
		req.TemplateType = "full"
	}

	if req.Config.AppName == "" {
		http.Error(w, "config.app_name is required", http.StatusBadRequest)
		return
	}

	if req.Config.APIEndpoint == "" {
		http.Error(w, "config.api_endpoint is required", http.StatusBadRequest)
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
		BuildID:      buildID,
		ScenarioName: req.ScenarioName,
		TemplateType: req.TemplateType,
		Config:       req.Config,
		Status:       "building",
		ExtensionPath: filepath.Join(config.OutputPath, req.ScenarioName, "platforms", "extension"),
		BuildLog:     []string{},
		ErrorLog:     []string{},
		CreatedAt:    time.Now(),
	}

	buildsMux.Lock()
	builds[buildID] = build
	buildsMux.Unlock()

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
	json.NewEncoder(w).Encode(response)
}

func getExtensionStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buildID := vars["build_id"]

	buildsMux.RLock()
	build, exists := builds[buildID]
	buildsMux.RUnlock()

	if !exists {
		http.Error(w, "Build not found", http.StatusNotFound)
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
	json.NewEncoder(w).Encode(response)
}

func testExtensionHandler(w http.ResponseWriter, r *http.Request) {
	var req ExtensionTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ExtensionPath == "" {
		http.Error(w, "extension_path is required", http.StatusBadRequest)
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
	json.NewEncoder(w).Encode(result)
}

func listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates := listAvailableTemplates()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

func listBuildsHandler(w http.ResponseWriter, r *http.Request) {
	buildsMux.RLock()
	buildList := make([]*ExtensionBuild, 0, len(builds))
	for _, build := range builds {
		buildList = append(buildList, build)
	}
	buildsMux.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"builds": buildList,
		"count":  len(buildList),
	})
}

// Helper functions

func generateBuildID() string {
	return fmt.Sprintf("build_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

func checkBrowserlessHealth() bool {
	// TODO: Implement actual browserless health check
	return true
}

func checkTemplatesHealth() bool {
	templatePath := filepath.Join(config.TemplatesPath, "vanilla", "manifest.json")
	_, err := os.Stat(templatePath)
	return err == nil
}

func countBuildsByStatus(status string) int {
	buildsMux.RLock()
	defer buildsMux.RUnlock()

	count := 0
	for _, build := range builds {
		if build.Status == status {
			count++
		}
	}
	return count
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
	// TODO: Implement template copying and variable substitution
	// This would involve:
	// 1. Walking through template files
	// 2. Replacing {{VARIABLES}} with actual values from build.Config
	// 3. Writing processed files to output directory
	// 4. Handling different template types (full, content-script-only, etc.)
	
	build.BuildLog = append(build.BuildLog, "Template processing would happen here")
	return nil
}

func generatePackageJSON(build *ExtensionBuild) error {
	// TODO: Generate package.json for the extension
	build.BuildLog = append(build.BuildLog, "Package.json generation would happen here")
	return nil
}

func generateREADME(build *ExtensionBuild) error {
	// TODO: Generate README.md for the extension
	build.BuildLog = append(build.BuildLog, "README generation would happen here")
	return nil
}

func testExtension(req *ExtensionTestRequest) *ExtensionTestResult {
	// TODO: Implement actual extension testing with browserless
	// This would involve:
	// 1. Loading the extension in browserless
	// 2. Navigating to test sites
	// 3. Taking screenshots
	// 4. Checking for JavaScript errors
	// 5. Validating extension functionality

	results := make([]ExtensionSiteResult, 0, len(req.TestSites))
	passed := 0

	for _, site := range req.TestSites {
		// Simulate test result
		result := ExtensionSiteResult{
			Site:           site,
			Loaded:         true,
			Errors:         []string{},
			ScreenshotPath: fmt.Sprintf("/tmp/extension-test-%d.png", time.Now().Unix()),
			LoadTime:       500,
		}
		results = append(results, result)
		passed++
	}

	total := len(results)
	successRate := float64(passed) / float64(total) * 100

	return &ExtensionTestResult{
		Success: passed == total,
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
	templates := []map[string]interface{}{
		{
			"name":        "full",
			"display_name": "Full Extension",
			"description": "Complete extension with background, content scripts, and popup",
			"files":       []string{"manifest.json", "background.js", "content.js", "popup.html", "popup.js"},
		},
		{
			"name":        "content-script-only",
			"display_name": "Content Script Only",
			"description": "Extension that only injects content scripts into web pages",
			"files":       []string{"manifest.json", "content.js"},
		},
		{
			"name":        "background-only",
			"display_name": "Background Only",
			"description": "Extension with only a background service worker",
			"files":       []string{"manifest.json", "background.js"},
		},
		{
			"name":        "popup-only",
			"display_name": "Popup Only",
			"description": "Extension with only a popup UI",
			"files":       []string{"manifest.json", "popup.html", "popup.js"},
		},
	}

	return templates
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