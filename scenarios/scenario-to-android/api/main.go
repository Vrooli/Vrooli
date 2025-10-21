package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Readiness bool      `json:"readiness"`
}

type StatusResponse struct {
	AndroidSDK  string `json:"android_sdk"`
	Java        string `json:"java"`
	Gradle      string `json:"gradle"`
	Ready       bool   `json:"ready"`
	BuildSystem string `json:"build_system"`
}

type MetricsResponse struct {
	TotalBuilds      int64   `json:"total_builds"`
	SuccessfulBuilds int64   `json:"successful_builds"`
	FailedBuilds     int64   `json:"failed_builds"`
	ActiveBuilds     int64   `json:"active_builds"`
	SuccessRate      float64 `json:"success_rate"`
	AverageDuration  float64 `json:"average_duration_seconds"`
	Uptime           float64 `json:"uptime_seconds"`
}

type BuildRequest struct {
	ScenarioName    string            `json:"scenario_name"`
	ConfigOverrides map[string]string `json:"config_overrides,omitempty"`
}

type BuildResponse struct {
	Success  bool              `json:"success"`
	APKPath  string            `json:"apk_path,omitempty"`
	BuildID  string            `json:"build_id"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    string            `json:"error,omitempty"`
}

type BuildStatusResponse struct {
	Status   string   `json:"status"`
	Progress int      `json:"progress"`
	Logs     []string `json:"logs,omitempty"`
}

// Build state tracking
type buildState struct {
	Status      string
	Progress    int
	Logs        []string
	APKPath     string
	Metadata    map[string]string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// Build metrics tracking
type buildMetrics struct {
	TotalBuilds      int64
	SuccessfulBuilds int64
	FailedBuilds     int64
	ActiveBuilds     int64
	AverageDuration  float64
}

var (
	builds          = make(map[string]*buildState)
	buildsMutex     sync.RWMutex
	metrics         buildMetrics
	metricsMutex    sync.RWMutex
	buildDurations  []time.Duration
	serverStartTime time.Time
)

const (
	maxScenarioNameLength = 100
	maxBuildsInMemory     = 100
	buildRetentionTime    = 1 * time.Hour
	maxDurationSamples    = 100
)

func buildHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(BuildResponse{
			Success: false,
			BuildID: "",
			Error:   "Method not allowed",
		})
		return
	}

	var req BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BuildResponse{
			Success: false,
			BuildID: "",
			Error:   "Invalid request body: " + err.Error(),
		})
		slog.Warn("Invalid build request body",
			"error", err.Error(),
			"remote_addr", r.RemoteAddr)
		return
	}

	if req.ScenarioName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BuildResponse{
			Success: false,
			BuildID: "",
			Error:   "scenario_name is required",
		})
		slog.Warn("Build request missing scenario_name",
			"remote_addr", r.RemoteAddr)
		return
	}

	// Validate scenario name length
	if len(req.ScenarioName) > maxScenarioNameLength {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(BuildResponse{
			Success: false,
			BuildID: "",
			Error:   fmt.Sprintf("scenario_name too long (max %d characters)", maxScenarioNameLength),
		})
		slog.Warn("Build request scenario_name too long",
			"scenario_name_length", len(req.ScenarioName),
			"max_length", maxScenarioNameLength,
			"remote_addr", r.RemoteAddr)
		return
	}

	// Validate scenario name format (alphanumeric, hyphens, underscores only)
	for _, char := range req.ScenarioName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(BuildResponse{
				Success: false,
				BuildID: "",
				Error:   "scenario_name contains invalid characters (only alphanumeric, hyphens, and underscores allowed)",
			})
			slog.Warn("Build request scenario_name contains invalid characters",
				"scenario_name", req.ScenarioName,
				"remote_addr", r.RemoteAddr)
			return
		}
	}

	// Clean up old builds before adding new one
	cleanupOldBuilds()

	// Generate build ID
	buildID := uuid.New().String()

	// Initialize build state
	buildsMutex.Lock()
	builds[buildID] = &buildState{
		Status:    "pending",
		Progress:  0,
		Logs:      []string{"Build initiated"},
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
	}
	buildsMutex.Unlock()

	// Update metrics
	metricsMutex.Lock()
	metrics.TotalBuilds++
	metrics.ActiveBuilds++
	metricsMutex.Unlock()

	slog.Info("Build request accepted",
		"build_id", buildID,
		"scenario_name", req.ScenarioName,
		"has_config_overrides", len(req.ConfigOverrides) > 0)

	// Start build in background
	go executeBuild(buildID, req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BuildResponse{
		Success: true,
		BuildID: buildID,
	})
}

// cleanupOldBuilds removes completed builds older than retention time or enforces max builds limit
func cleanupOldBuilds() {
	buildsMutex.Lock()
	defer buildsMutex.Unlock()

	now := time.Now()
	var toDelete []string

	// Find builds to delete (completed and older than retention time)
	for id, state := range builds {
		if (state.Status == "complete" || state.Status == "failed") &&
			now.Sub(state.CreatedAt) > buildRetentionTime {
			toDelete = append(toDelete, id)
		}
	}

	// If still too many builds, delete oldest completed builds
	if len(builds)-len(toDelete) > maxBuildsInMemory {
		type buildAge struct {
			id  string
			age time.Time
		}
		var completedBuilds []buildAge

		for id, state := range builds {
			if (state.Status == "complete" || state.Status == "failed") &&
				!contains(toDelete, id) {
				completedBuilds = append(completedBuilds, buildAge{id, state.CreatedAt})
			}
		}

		// Sort by age (oldest first) using standard library for better performance
		sort.Slice(completedBuilds, func(i, j int) bool {
			return completedBuilds[i].age.Before(completedBuilds[j].age)
		})

		// Delete oldest until we're under the limit
		for i := 0; len(builds)-len(toDelete) > maxBuildsInMemory && i < len(completedBuilds); i++ {
			toDelete = append(toDelete, completedBuilds[i].id)
		}
	}

	// Perform deletions
	for _, id := range toDelete {
		delete(builds, id)
	}

	if len(toDelete) > 0 {
		slog.Info("Cleaned up old builds",
			"deleted_count", len(toDelete),
			"remaining_builds", len(builds))
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func buildStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate HTTP method
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Extract build ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 || pathParts[5] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Build ID required",
		})
		return
	}
	buildID := pathParts[5]

	buildsMutex.RLock()
	state, exists := builds[buildID]
	buildsMutex.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Build not found",
		})
		return
	}

	json.NewEncoder(w).Encode(BuildStatusResponse{
		Status:   state.Status,
		Progress: state.Progress,
		Logs:     state.Logs,
	})
}

func executeBuild(buildID string, req BuildRequest) {
	updateBuildState := func(status string, progress int, logMsg string) {
		buildsMutex.Lock()
		defer buildsMutex.Unlock()
		if state, ok := builds[buildID]; ok {
			state.Status = status
			state.Progress = progress
			if logMsg != "" {
				state.Logs = append(state.Logs, logMsg)
			}
		}
	}

	slog.Info("Build execution started",
		"build_id", buildID,
		"scenario_name", req.ScenarioName)

	updateBuildState("building", 10, "Starting build process...")

	// Get scenario root
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Find the convert script
	convertScript := filepath.Join(vrooliRoot, "scenarios", "scenario-to-android", "cli", "convert.sh")
	if _, err := os.Stat(convertScript); os.IsNotExist(err) {
		slog.Error("Convert script not found",
			"build_id", buildID,
			"script_path", convertScript,
			"error", err)
		updateBuildState("failed", 0, "Convert script not found: "+convertScript)
		return
	}

	slog.Debug("Found conversion script",
		"build_id", buildID,
		"script_path", convertScript)

	updateBuildState("building", 20, "Found conversion script")

	// Prepare output directory
	outputDir := filepath.Join(os.TempDir(), "scenario-to-android-builds", buildID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("Failed to create output directory",
			"build_id", buildID,
			"output_dir", outputDir,
			"error", err)
		updateBuildState("failed", 0, "Failed to create output directory: "+err.Error())
		return
	}

	slog.Debug("Created output directory",
		"build_id", buildID,
		"output_dir", outputDir)

	updateBuildState("building", 30, "Created output directory: "+outputDir)

	// Build command arguments
	args := []string{
		convertScript,
		"--scenario", req.ScenarioName,
		"--output", outputDir,
	}

	// Add config overrides if provided
	if appName, ok := req.ConfigOverrides["app_name"]; ok {
		args = append(args, "--app-name", appName)
	}
	if version, ok := req.ConfigOverrides["version"]; ok {
		args = append(args, "--version", version)
	}

	updateBuildState("building", 40, "Executing conversion: "+strings.Join(args, " "))

	// Execute the conversion
	cmd := exec.Command("bash", args...)
	cmd.Dir = vrooliRoot

	slog.Info("Executing conversion command",
		"build_id", buildID,
		"command", strings.Join(args, " "),
		"working_dir", vrooliRoot)

	output, err := cmd.CombinedOutput()

	if err != nil {
		slog.Error("Build failed",
			"build_id", buildID,
			"scenario_name", req.ScenarioName,
			"error", err,
			"output", string(output))
		updateBuildState("failed", 0, "Build failed: "+err.Error()+"\n"+string(output))

		// Update metrics for failure
		metricsMutex.Lock()
		metrics.ActiveBuilds--
		metrics.FailedBuilds++
		metricsMutex.Unlock()

		// Record completion time for failed build
		now := time.Now()
		buildsMutex.Lock()
		if state, ok := builds[buildID]; ok {
			state.CompletedAt = &now
		}
		buildsMutex.Unlock()

		return
	}

	slog.Info("Conversion completed successfully",
		"build_id", buildID,
		"scenario_name", req.ScenarioName)

	updateBuildState("building", 80, "Conversion completed")

	// Find the generated APK
	apkPath := ""
	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".apk") {
			apkPath = path
			return filepath.SkipAll
		}
		return nil
	})

	// Record completion time and duration
	now := time.Now()
	buildsMutex.Lock()
	if state, ok := builds[buildID]; ok {
		state.CompletedAt = &now
		duration := now.Sub(state.CreatedAt)

		// Update duration tracking
		buildDurations = append(buildDurations, duration)
		if len(buildDurations) > maxDurationSamples {
			buildDurations = buildDurations[1:]
		}
	}
	buildsMutex.Unlock()

	// Update metrics
	metricsMutex.Lock()
	metrics.ActiveBuilds--
	metrics.SuccessfulBuilds++

	// Calculate average duration
	if len(buildDurations) > 0 {
		var total time.Duration
		for _, d := range buildDurations {
			total += d
		}
		metrics.AverageDuration = total.Seconds() / float64(len(buildDurations))
	}
	metricsMutex.Unlock()

	if apkPath == "" {
		slog.Info("Build completed without APK",
			"build_id", buildID,
			"scenario_name", req.ScenarioName,
			"reason", "Android SDK not configured",
			"project_dir", outputDir)
		updateBuildState("complete", 100, "Build completed (no APK generated - requires Android SDK)")
		buildsMutex.Lock()
		if state, ok := builds[buildID]; ok {
			state.APKPath = ""
			state.Metadata["note"] = "Android SDK not configured - project created but APK not built"
			state.Metadata["project_dir"] = outputDir
		}
		buildsMutex.Unlock()
	} else {
		slog.Info("Build completed successfully with APK",
			"build_id", buildID,
			"scenario_name", req.ScenarioName,
			"apk_path", apkPath,
			"project_dir", outputDir)
		updateBuildState("complete", 100, "APK generated successfully: "+apkPath)
		buildsMutex.Lock()
		if state, ok := builds[buildID]; ok {
			state.APKPath = apkPath
			state.Metadata["apk_path"] = apkPath
			state.Metadata["project_dir"] = outputDir
		}
		buildsMutex.Unlock()
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate HTTP method
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "scenario-to-android",
		Version:   "1.0.0",
		Readiness: true,
	}

	json.NewEncoder(w).Encode(response)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate HTTP method
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	androidHome := os.Getenv("ANDROID_HOME")
	javaHome := os.Getenv("JAVA_HOME")

	response := StatusResponse{
		AndroidSDK:  androidHome,
		Java:        javaHome,
		Gradle:      "wrapper",
		Ready:       androidHome != "",
		BuildSystem: "gradle",
	}

	json.NewEncoder(w).Encode(response)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate HTTP method
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	metricsMutex.RLock()
	currentMetrics := metrics
	metricsMutex.RUnlock()

	// Calculate success rate
	successRate := 0.0
	completedBuilds := currentMetrics.SuccessfulBuilds + currentMetrics.FailedBuilds
	if completedBuilds > 0 {
		successRate = float64(currentMetrics.SuccessfulBuilds) / float64(completedBuilds) * 100
	}

	// Calculate uptime
	uptime := time.Since(serverStartTime).Seconds()

	response := MetricsResponse{
		TotalBuilds:      currentMetrics.TotalBuilds,
		SuccessfulBuilds: currentMetrics.SuccessfulBuilds,
		FailedBuilds:     currentMetrics.FailedBuilds,
		ActiveBuilds:     currentMetrics.ActiveBuilds,
		SuccessRate:      successRate,
		AverageDuration:  currentMetrics.AverageDuration,
		Uptime:           uptime,
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	// Ensure binary is run through Vrooli lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-to-android

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize server start time (after lifecycle check)
	serverStartTime = time.Now()

	// API_PORT must be set by the lifecycle system
	port := os.Getenv("API_PORT")
	if port == "" {
		slog.Error("API_PORT environment variable is required",
			"error", "missing required environment variable",
			"instruction", "use make start or vrooli scenario start scenario-to-android",
		)
		os.Exit(1)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/v1/health", healthHandler)
	http.HandleFunc("/api/v1/status", statusHandler)
	http.HandleFunc("/api/v1/android/build", buildHandler)
	http.HandleFunc("/api/v1/android/status/", buildStatusHandler)
	http.HandleFunc("/api/v1/metrics", metricsHandler)

	addr := fmt.Sprintf(":%s", port)

	// Use structured logging for better observability
	slog.Info("Scenario to Android API server starting",
		"addr", addr,
		"port", port)
	slog.Info("Health endpoint available",
		"url", fmt.Sprintf("http://localhost:%s/health", port))
	slog.Info("API status endpoint available",
		"url", fmt.Sprintf("http://localhost:%s/api/v1/status", port))

	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
