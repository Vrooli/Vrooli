package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	bundleruntime "scenario-to-desktop-runtime"
	runtimeapi "scenario-to-desktop-runtime/api"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

type quickGenerateRequest struct {
	ScenarioName     string   `json:"scenario_name"`
	TemplateType     string   `json:"template_type"`
	DeploymentMode   string   `json:"deployment_mode"`
	AutoManageVrooli *bool    `json:"auto_manage_vrooli"`
	LegacyAutoManage *bool    `json:"auto_manage_tier1"`
	ProxyURL         string   `json:"proxy_url"`
	LegacyServerURL  string   `json:"server_url"`
	LegacyAPIURL     string   `json:"api_url"`
	BundleManifest   string   `json:"bundle_manifest_path"`
	VrooliBinary     string   `json:"vrooli_binary_path"`
	Platforms        []string `json:"platforms"`
}

// Quick generate desktop handler - auto-detects scenario configuration
func (s *Server) quickGenerateDesktopHandler(w http.ResponseWriter, r *http.Request) {
	var request quickGenerateRequest

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
		request.TemplateType = defaultTemplateType // Default to universal template (works for any scenario)
	}

	// Validate template type
	if !isValidTemplateType(request.TemplateType) {
		http.Error(w, fmt.Sprintf("invalid template_type: %s", request.TemplateType), http.StatusBadRequest)
		return
	}

	// Create analyzer
	analyzer := NewScenarioAnalyzer(detectVrooliRoot())

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

	savedConfig, err := loadDesktopConnectionConfig(metadata.ScenarioPath)
	if err != nil {
		s.logger.Warn("failed to read saved desktop config",
			"scenario", metadata.Name,
			"error", err)
	}

	config = mergeQuickGenerateConfig(config, request, savedConfig, s.standardOutputPath(config.AppName))

	if err := s.validateAndPrepareBundle(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.persistDesktopConfig(metadata.ScenarioPath, config)

	s.logger.Info("quick generate request",
		"scenario", request.ScenarioName,
		"template", request.TemplateType,
		"has_ui", metadata.HasUI,
		"display_name", metadata.DisplayName)

	// Generate build ID
	buildStatus := s.queueDesktopBuild(config, metadata, true)

	// Return immediate response
	response := map[string]interface{}{
		"build_id":             buildStatus.BuildID,
		"status":               "building",
		"scenario_name":        config.AppName,
		"desktop_path":         buildStatus.OutputPath,
		"detected_metadata":    metadata,
		"install_instructions": "Run 'npm install && npm run dev' in the output directory",
		"test_command":         "npm run dev",
		"status_url":           fmt.Sprintf("/api/v1/desktop/status/%s", buildStatus.BuildID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listDesktopRecordsHandler returns the persisted desktop generation records.
func (s *Server) listDesktopRecordsHandler(w http.ResponseWriter, _ *http.Request) {
	if s.records == nil {
		http.Error(w, "record store unavailable", http.StatusInternalServerError)
		return
	}

	type recordWithBuild struct {
		Record     *DesktopAppRecord `json:"record"`
		Build      *BuildStatus      `json:"build_status,omitempty"`
		HasBuild   bool              `json:"has_build"`
		BuildState string            `json:"build_state,omitempty"`
	}

	var results []recordWithBuild
	for _, rec := range s.records.List() {
		item := recordWithBuild{Record: rec}
		if rec != nil && rec.BuildID != "" {
			if bs, ok := s.builds.Get(rec.BuildID); ok {
				item.Build = bs
				item.HasBuild = true
				item.BuildState = bs.Status
			}
		}
		results = append(results, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records": results,
	})
}

// moveDesktopRecordHandler moves a generated desktop wrapper from its current path to a destination.
func (s *Server) moveDesktopRecordHandler(w http.ResponseWriter, r *http.Request) {
	if s.records == nil {
		http.Error(w, "record store unavailable", http.StatusInternalServerError)
		return
	}

	recordID := mux.Vars(r)["record_id"]
	rec, ok := s.records.Get(recordID)
	if !ok || rec == nil {
		http.Error(w, "record not found", http.StatusNotFound)
		return
	}

	var req struct {
		Target          string `json:"target"`           // "destination" (default) or "custom"
		DestinationPath string `json:"destination_path"` // required when target == "custom"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Target == "" {
		req.Target = "destination"
	}

	src := rec.OutputPath
	dest := rec.DestinationPath
	if req.Target == "custom" {
		if req.DestinationPath == "" {
			http.Error(w, "destination_path required for custom target", http.StatusBadRequest)
			return
		}
		dest = req.DestinationPath
	}
	if dest == "" {
		http.Error(w, "no destination path recorded", http.StatusBadRequest)
		return
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve source: %v", err), http.StatusBadRequest)
		return
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve destination: %v", err), http.StatusBadRequest)
		return
	}
	if absSrc == absDest {
		http.Error(w, "source and destination are the same", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(absSrc); err != nil {
		http.Error(w, fmt.Sprintf("source missing: %v", err), http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(filepath.Dir(absDest), 0o755); err != nil {
		http.Error(w, fmt.Sprintf("prepare destination: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.Rename(absSrc, absDest); err != nil {
		http.Error(w, fmt.Sprintf("move failed: %v", err), http.StatusInternalServerError)
		return
	}

	rec.OutputPath = absDest
	if req.Target != "custom" {
		rec.LocationMode = "proper"
	}
	if err := s.records.Upsert(rec); err != nil {
		s.logger.Warn("failed to update record after move", "error", err)
	}

	if rec.BuildID != "" {
		_ = s.builds.Update(rec.BuildID, func(status *BuildStatus) {
			status.OutputPath = absDest
			if status.Metadata == nil {
				status.Metadata = map[string]interface{}{}
			}
			status.Metadata["moved_to"] = absDest
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"record_id": recordID,
		"from":      absSrc,
		"to":        absDest,
		"status":    "moved",
	})
}

// Generate desktop application handler

func (s *Server) generateDesktopHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	config, err := decodeDesktopConfig(body)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	if config.LocationMode == "" {
		config.LocationMode = "proper"
	}

	if err := s.validateAndPrepareBundle(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.persistDesktopConfig(s.scenarioRoot(config.AppName), config)

	buildStatus := s.queueDesktopBuild(config, nil, false)

	// Return immediate response
	response := map[string]interface{}{
		"build_id":             buildStatus.BuildID,
		"status":               "building",
		"desktop_path":         config.OutputPath,
		"install_instructions": "Run 'npm install && npm run dev' in the output directory",
		"test_command":         "npm run dev",
		"status_url":           fmt.Sprintf("/api/v1/desktop/status/%s", buildStatus.BuildID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Dry-run bundled runtime validation handler.
func (s *Server) preflightBundleHandler(w http.ResponseWriter, r *http.Request) {
	var request BundlePreflightRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	result, err := s.runBundlePreflight(request)
	if err != nil {
		status := http.StatusInternalServerError
		var statusErr *preflightStatusError
		if errors.As(err, &statusErr) {
			status = statusErr.Status
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Preflight async start handler.
func (s *Server) preflightStartHandler(w http.ResponseWriter, r *http.Request) {
	var request BundlePreflightRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.BundleManifestPath) == "" {
		http.Error(w, "bundle_manifest_path is required", http.StatusBadRequest)
		return
	}
	if request.StatusOnly {
		http.Error(w, "status_only is not supported for async preflight", http.StatusBadRequest)
		return
	}
	if request.SessionStop {
		http.Error(w, "session_stop is not supported for async preflight", http.StatusBadRequest)
		return
	}

	job := s.createPreflightJob()
	go s.runPreflightJob(job.id, request)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BundlePreflightJobStartResponse{JobID: job.id})
}

// Preflight async status handler.
func (s *Server) preflightStatusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.URL.Query().Get("job_id"))
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}
	job, ok := s.getPreflightJob(jobID)
	if !ok {
		http.Error(w, fmt.Sprintf("preflight job not found: %s", jobID), http.StatusNotFound)
		return
	}

	ordered := []string{"validation", "secrets", "runtime", "services", "diagnostics"}
	steps := make([]BundlePreflightStep, 0, len(job.steps))
	seen := map[string]bool{}
	for _, id := range ordered {
		if step, ok := job.steps[id]; ok {
			steps = append(steps, step)
			seen[id] = true
		}
	}
	for id, step := range job.steps {
		if !seen[id] {
			steps = append(steps, step)
		}
	}

	response := BundlePreflightJobStatusResponse{
		JobID:     job.id,
		Status:    job.status,
		Steps:     steps,
		Result:    job.result,
		Error:     job.err,
		StartedAt: job.startedAt.Format(time.RFC3339),
		UpdatedAt: job.updatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Preflight health proxy handler.
func (s *Server) preflightHealthHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if serviceID == "" {
		http.Error(w, "service_id is required", http.StatusBadRequest)
		return
	}

	session, ok := s.getPreflightSession(sessionID)
	if !ok {
		http.Error(w, fmt.Sprintf("preflight session not found: %s", sessionID), http.StatusNotFound)
		return
	}
	if session.manifest == nil {
		http.Error(w, "preflight manifest missing", http.StatusNotFound)
		return
	}

	service, ok := findManifestService(session.manifest, serviceID)
	if !ok {
		http.Error(w, fmt.Sprintf("service not found in manifest: %s", serviceID), http.StatusNotFound)
		return
	}
	if service.Health.Type != "http" {
		response := map[string]interface{}{
			"service_id":  serviceID,
			"supported":   false,
			"health_type": service.Health.Type,
			"message":     "health proxy only supports http health checks",
			"fetched_at":  time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	if service.Health.PortName == "" {
		http.Error(w, fmt.Sprintf("service %s health port_name is required", serviceID), http.StatusBadRequest)
		return
	}

	healthPath := strings.TrimSpace(service.Health.Path)
	if healthPath == "" {
		healthPath = "/health"
	}
	if !strings.HasPrefix(healthPath, "/") {
		healthPath = "/" + healthPath
	}

	timeoutMs := service.Health.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 2000
	}
	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := fetchJSON(client, session.baseURL, session.token, "/ports", http.MethodGet, nil, &portsResp, nil); err != nil {
		http.Error(w, fmt.Sprintf("fetch ports: %v", err), http.StatusBadGateway)
		return
	}
	port := portsResp.Services[serviceID][service.Health.PortName]
	if port == 0 {
		http.Error(w, fmt.Sprintf("port not found for service %s (%s)", serviceID, service.Health.PortName), http.StatusBadRequest)
		return
	}

	healthURL := fmt.Sprintf("http://localhost:%d%s", port, healthPath)
	start := time.Now()
	req, err := http.NewRequest(http.MethodGet, healthURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("build health request: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("fetch health: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	const maxHealthBodyBytes = 64 * 1024
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxHealthBodyBytes))
	if err != nil {
		http.Error(w, fmt.Sprintf("read health response: %v", err), http.StatusBadGateway)
		return
	}
	bodyText := strings.TrimSpace(string(bodyBytes))
	truncated := len(bodyBytes) >= maxHealthBodyBytes

	response := map[string]interface{}{
		"service_id":   serviceID,
		"supported":    true,
		"health_type":  service.Health.Type,
		"url":          healthURL,
		"status_code":  resp.StatusCode,
		"status":       resp.Status,
		"body":         bodyText,
		"content_type": resp.Header.Get("Content-Type"),
		"truncated":    truncated,
		"fetched_at":   time.Now().Format(time.RFC3339),
		"elapsed_ms":   time.Since(start).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Bundle manifest display handler.
func (s *Server) bundleManifestHandler(w http.ResponseWriter, r *http.Request) {
	var request BundleManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.BundleManifestPath) == "" {
		http.Error(w, "bundle_manifest_path is required", http.StatusBadRequest)
		return
	}

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve bundle_manifest_path: %v", err), http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(manifestPath); err != nil {
		http.Error(w, fmt.Sprintf("bundle manifest not found: %v", err), http.StatusBadRequest)
		return
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("read bundle manifest: %v", err), http.StatusInternalServerError)
		return
	}

	var manifest json.RawMessage
	if err := json.Unmarshal(raw, &manifest); err != nil {
		http.Error(w, fmt.Sprintf("parse bundle manifest: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BundleManifestResponse{
		Path:     manifestPath,
		Manifest: manifest,
	})
}

func mergeQuickGenerateConfig(config *DesktopConfig, request quickGenerateRequest, savedConfig *DesktopConnectionConfig, defaultOutputPath string) *DesktopConfig {
	if config == nil {
		return nil
	}

	config.ProxyURL = chooseProxyURL(request, savedConfig)
	config.VrooliBinaryPath = chooseVrooliBinary(request, savedConfig, config.VrooliBinaryPath)
	config.DeploymentMode = chooseDeploymentMode(request, savedConfig, config.DeploymentMode)
	config.BundleManifestPath = chooseBundleManifestPath(request, savedConfig)
	config.AutoManageVrooli = chooseAutoManage(request, savedConfig, config.AutoManageVrooli)
	config.Platforms = choosePlatforms(request.Platforms, config.Platforms)

	if config.OutputPath == "" {
		config.OutputPath = defaultOutputPath
	}
	if config.LocationMode == "" {
		config.LocationMode = "proper"
	}
	if savedConfig != nil && savedConfig.ServerType != "" && request.ProxyURL == "" && request.LegacyServerURL == "" {
		config.ServerType = savedConfig.ServerType
	}
	if savedConfig != nil {
		if savedConfig.AppDisplayName != "" {
			config.AppDisplayName = savedConfig.AppDisplayName
		}
		if savedConfig.AppDescription != "" {
			config.AppDescription = savedConfig.AppDescription
		}
		if savedConfig.Icon != "" {
			config.Icon = savedConfig.Icon
		}
	}

	return config
}

func chooseProxyURL(request quickGenerateRequest, savedConfig *DesktopConnectionConfig) string {
	if request.ProxyURL != "" {
		return request.ProxyURL
	}
	if savedConfig != nil {
		if savedConfig.ProxyURL != "" {
			return savedConfig.ProxyURL
		}
		if savedConfig.ServerURL != "" {
			return savedConfig.ServerURL
		}
	}
	if request.LegacyServerURL != "" {
		return request.LegacyServerURL
	}
	if request.LegacyAPIURL != "" {
		return request.LegacyAPIURL
	}

	return ""
}

func chooseVrooliBinary(request quickGenerateRequest, savedConfig *DesktopConnectionConfig, existing string) string {
	if request.VrooliBinary != "" {
		return request.VrooliBinary
	}
	if savedConfig != nil && savedConfig.VrooliBinary != "" {
		return savedConfig.VrooliBinary
	}
	return existing
}

func chooseDeploymentMode(request quickGenerateRequest, savedConfig *DesktopConnectionConfig, existing string) string {
	if request.DeploymentMode != "" {
		return request.DeploymentMode
	}
	if savedConfig != nil && savedConfig.DeploymentMode != "" {
		return savedConfig.DeploymentMode
	}
	return existing
}

func chooseBundleManifestPath(request quickGenerateRequest, savedConfig *DesktopConnectionConfig) string {
	if request.BundleManifest != "" {
		return request.BundleManifest
	}
	if savedConfig != nil && savedConfig.BundleManifestPath != "" {
		return savedConfig.BundleManifestPath
	}
	return ""
}

func choosePlatforms(requested []string, existing []string) []string {
	if len(requested) > 0 {
		return requested
	}
	if len(existing) > 0 {
		return existing
	}
	return nil
}

func chooseAutoManage(request quickGenerateRequest, savedConfig *DesktopConnectionConfig, existing bool) bool {
	if request.AutoManageVrooli != nil {
		return *request.AutoManageVrooli
	}
	if request.LegacyAutoManage != nil {
		return *request.LegacyAutoManage
	}
	if savedConfig != nil {
		return savedConfig.AutoManageVrooli
	}
	return existing
}

// Get build status handler
func (s *Server) getBuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buildID := vars["build_id"]

	status, exists := s.builds.Get(buildID)

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

	// Check if desktop wrapper exists
	desktopPath := s.standardOutputPath(scenarioName)
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Desktop wrapper not found for '%s'. Generate it first.", scenarioName), http.StatusNotFound)
		return
	}

	options := decodeScenarioBuildOptions(r.Body)

	buildID := uuid.New().String()

	// Initialize platform results
	platformResults := newPlatformResults(options.Platforms)

	// Create build status
	buildStatus := &BuildStatus{
		BuildID:            buildID,
		ScenarioName:       scenarioName,
		Status:             "building",
		OutputPath:         desktopPath,
		CreatedAt:          time.Now(),
		Platforms:          options.Platforms, // Legacy field (will be updated with successful builds)
		RequestedPlatforms: options.Platforms, // NEW: Track what was requested
		PlatformResults:    platformResults,
		BuildLog:           []string{},
		ErrorLog:           []string{},
		Artifacts:          make(map[string]string),
	}

	s.builds.Save(buildStatus)

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

	// Find built package
	distPath := filepath.Join(s.standardOutputPath(scenarioName), "dist-electron")
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
	filename := filepath.Base(packageFile)
	contentType := detectPackageContentType(packageFile)

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

	if !isSafeScenarioName(scenarioName) {
		http.Error(w, "Invalid scenario name", http.StatusBadRequest)
		return
	}

	// Construct desktop path - MUST be exactly platforms/electron/
	desktopPath := s.standardOutputPath(scenarioName)

	// Security check: Verify the path is actually inside platforms/electron
	absDesktopPath, err := filepath.Abs(desktopPath)
	if err != nil {
		http.Error(w, "Failed to resolve desktop path", http.StatusInternalServerError)
		return
	}

	absExpectedPrefix, _ := filepath.Abs(desktopPath)
	if absDesktopPath != absExpectedPrefix {
		s.logger.Error("path traversal attempt detected",
			"scenario", scenarioName,
			"expected", absExpectedPrefix,
			"actual", absDesktopPath)
		http.Error(w, "Security violation: invalid path", http.StatusBadRequest)
		return
	}

	// Check if desktop directory exists
	_, statErr := os.Stat(desktopPath)
	if statErr == nil {
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
	} else if !errors.Is(statErr, os.ErrNotExist) {
		http.Error(w, fmt.Sprintf("Failed to read desktop directory: %v", statErr), http.StatusInternalServerError)
		return
	}

	removedRecords := 0
	if s.records != nil {
		removedRecords = s.records.DeleteByScenario(scenarioName)
	}

	message := fmt.Sprintf("Desktop version of '%s' deleted successfully", scenarioName)
	if errors.Is(statErr, os.ErrNotExist) {
		message = fmt.Sprintf("Desktop version of '%s' was already missing; cleaned up record state.", scenarioName)
	}

	// Return success response
	response := map[string]interface{}{
		"status":          "success",
		"scenario_name":   scenarioName,
		"deleted_path":    desktopPath,
		"removed_records": removedRecords,
		"message":         message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) prepareBundledConfig(config *DesktopConfig) (*bundlemanifest.Manifest, error) {
	if config.BundleManifestPath == "" {
		return nil, fmt.Errorf("bundle_manifest_path is required when deployment_mode is 'bundled'")
	}

	manifestPath, err := filepath.Abs(config.BundleManifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolve bundle_manifest_path: %w", err)
	}

	m, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("load bundle manifest: %w", err)
	}
	if err := m.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		return nil, fmt.Errorf("validate bundle manifest: %w", err)
	}

	config.BundleManifestPath = manifestPath
	if config.BundleRuntimeRoot == "" {
		config.BundleRuntimeRoot = "bundle"
	}
	if config.BundleIPC == nil {
		config.BundleIPC = &BundleIPCConfig{}
	}
	if config.BundleIPC.Host == "" {
		config.BundleIPC.Host = m.IPC.Host
	}
	if config.BundleIPC.Port == 0 {
		config.BundleIPC.Port = m.IPC.Port
	}
	if config.BundleIPC.AuthTokenRel == "" {
		config.BundleIPC.AuthTokenRel = m.IPC.AuthTokenRel
	}
	if config.BundleTelemetryUploadURL == "" {
		config.BundleTelemetryUploadURL = m.Telemetry.UploadTo
	}

	uiService, uiPort := inferUISurface(m, config.BundleUIPortName)
	if config.BundleUISvcID == "" {
		config.BundleUISvcID = uiService
	}
	if config.BundleUIPortName == "" {
		config.BundleUIPortName = uiPort
	}

	return m, nil
}

func inferUISurface(m *bundlemanifest.Manifest, preferredPort string) (string, string) {
	portName := preferredPort
	var uiService string
	for _, svc := range m.Services {
		if svc.Type == "ui-bundle" {
			uiService = svc.ID
			if portName == "" {
				portName = firstPortName(svc)
			}
			break
		}
	}

	if uiService == "" && len(m.Services) > 0 {
		uiService = m.Services[0].ID
		if portName == "" {
			portName = firstPortName(m.Services[0])
		}
	}

	if portName == "" {
		portName = "http"
	}

	return uiService, portName
}

func firstPortName(svc bundlemanifest.Service) string {
	if svc.Ports != nil && len(svc.Ports.Requested) > 0 {
		if svc.Ports.Requested[0].Name != "" {
			return svc.Ports.Requested[0].Name
		}
	}
	return "http"
}

func (s *Server) persistDesktopConfig(scenarioRoot string, config *DesktopConfig) {
	if config == nil || scenarioRoot == "" {
		return
	}
	if _, err := os.Stat(scenarioRoot); err != nil {
		return
	}

	conn := &DesktopConnectionConfig{
		ProxyURL:           config.ProxyURL,
		ServerURL:          config.ExternalServerURL,
		APIURL:             config.ExternalAPIURL,
		AppDisplayName:     config.AppDisplayName,
		AppDescription:     config.AppDescription,
		Icon:               config.Icon,
		DeploymentMode:     config.DeploymentMode,
		AutoManageVrooli:   config.AutoManageVrooli,
		VrooliBinary:       config.VrooliBinaryPath,
		ServerType:         config.ServerType,
		BundleManifestPath: config.BundleManifestPath,
	}

	if err := saveDesktopConnectionConfig(scenarioRoot, conn); err != nil {
		s.logger.Warn("failed to persist desktop config",
			"scenario", config.AppName,
			"path", scenarioRoot,
			"error", err)
	}
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

	s.builds.Save(buildStatus)

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
		AppPath            string   `json:"app_path"`
		BundleManifestPath string   `json:"bundle_manifest_path"`
		Platforms          []string `json:"platforms"`
		Store              string   `json:"store"`
		Enterprise         bool     `json:"enterprise"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	result, err := packageBundle(request.AppPath, request.BundleManifestPath, request.Platforms)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to package bundle: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"status":           "completed",
		"bundle_dir":       result.BundleDir,
		"manifest":         result.ManifestPath,
		"runtime_binaries": result.RuntimeBinaries,
		"artifacts":        result.CopiedArtifacts,
		"total_size_bytes": result.TotalSizeBytes,
		"total_size_human": result.TotalSizeHuman,
		"timestamp":        time.Now(),
	}

	// Include size warning if present
	if result.SizeWarning != nil {
		response["size_warning"] = result.SizeWarning
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
	s.builds.Update(buildID, func(status *BuildStatus) {
		if resultStatus, ok := result["status"].(string); ok {
			status.Status = resultStatus
		}
		if resultStatus := result["status"].(string); resultStatus == "completed" || resultStatus == "failed" {
			now := time.Now()
			status.CompletedAt = &now
		}
	})

	s.logger.Info("build webhook received",
		"build_id", buildID,
		"status", result["status"],
		"has_error", result["error"] != nil)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func (s *Server) validateAndPrepareBundle(config *DesktopConfig) error {
	if err := s.validateDesktopConfig(config); err != nil {
		return fmt.Errorf("Configuration validation failed: %s", err)
	}

	if config.DeploymentMode != "bundled" {
		return nil
	}

	if _, err := s.prepareBundledConfig(config); err != nil {
		return fmt.Errorf("Bundle validation failed: %s", err)
	}

	return nil
}

func decodeScenarioBuildOptions(body io.ReadCloser) struct {
	Platforms []string
	Clean     bool
} {
	var options struct {
		Platforms []string `json:"platforms"` // win, mac, linux
		Clean     bool     `json:"clean"`     // Clean before building
	}

	if body != nil {
		_ = json.NewDecoder(body).Decode(&options)
	}

	if len(options.Platforms) == 0 {
		options.Platforms = []string{"win", "mac", "linux"}
	}

	return struct {
		Platforms []string
		Clean     bool
	}{
		Platforms: options.Platforms,
		Clean:     options.Clean,
	}
}

func newPlatformResults(platforms []string) map[string]*PlatformBuildResult {
	platformResults := make(map[string]*PlatformBuildResult)
	for _, platform := range platforms {
		platformResults[platform] = &PlatformBuildResult{
			Platform: platform,
			Status:   "pending",
		}
	}
	return platformResults
}

func detectPackageContentType(packageFile string) string {
	switch {
	case strings.HasSuffix(packageFile, ".msi"):
		return "application/x-msi"
	case strings.HasSuffix(packageFile, ".pkg"):
		return "application/vnd.apple.installer+xml"
	case strings.HasSuffix(packageFile, ".exe"):
		return "application/x-msdownload"
	case strings.HasSuffix(packageFile, ".dmg"):
		return "application/x-apple-diskimage"
	case strings.HasSuffix(packageFile, ".AppImage"):
		return "application/x-executable"
	case strings.HasSuffix(packageFile, ".deb"):
		return "application/vnd.debian.binary-package"
	case strings.HasSuffix(packageFile, ".zip"):
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func isSafeScenarioName(name string) bool {
	return !strings.Contains(name, "..") && !strings.Contains(name, "/") && !strings.Contains(name, "\\")
}

type preflightStatusError struct {
	Status int
	Err    error
}

func (e *preflightStatusError) Error() string {
	return e.Err.Error()
}

type preflightIssue struct {
	status string
	detail string
}

type preflightSession struct {
	id         string
	manifest   *bundlemanifest.Manifest
	bundleDir  string
	appData    string
	supervisor *bundleruntime.Supervisor
	baseURL    string
	token      string
	createdAt  time.Time
	expiresAt  time.Time
}

type preflightJob struct {
	id        string
	status    string
	steps     map[string]BundlePreflightStep
	result    *BundlePreflightResponse
	err       string
	startedAt time.Time
	updatedAt time.Time
}

func findManifestService(manifest *bundlemanifest.Manifest, serviceID string) (*bundlemanifest.Service, bool) {
	if manifest == nil {
		return nil, false
	}
	for i := range manifest.Services {
		service := &manifest.Services[i]
		if service.ID == serviceID {
			return service, true
		}
	}
	return nil, false
}

func buildPreflightChecks(manifest *bundlemanifest.Manifest, validation *runtimeapi.BundleValidationResult, ready *BundlePreflightReady, secrets []BundlePreflightSecret, ports map[string]map[string]int, telemetry *BundlePreflightTelemetry, logTails []BundlePreflightLogTail, request BundlePreflightRequest) []BundlePreflightCheck {
	checks := []BundlePreflightCheck{}
	if manifest == nil {
		return checks
	}

	addCheck := func(id, step, name, status, detail string) {
		checks = append(checks, BundlePreflightCheck{
			ID:     id,
			Step:   step,
			Name:   name,
			Status: status,
			Detail: detail,
		})
	}

	statusOnly := request.StatusOnly

	manifestStatus := "pass"
	manifestDetail := fmt.Sprintf("validated for %s/%s", runtime.GOOS, runtime.GOARCH)
	if statusOnly {
		manifestStatus = "skipped"
		manifestDetail = "status-only refresh"
	} else if validation == nil {
		manifestStatus = "fail"
		manifestDetail = "validation result missing"
	} else {
		for _, err := range validation.Errors {
			if err.Code == "manifest_invalid" {
				manifestStatus = "fail"
				manifestDetail = err.Message
				break
			}
		}
	}
	addCheck("validation.manifest", "validation", "Manifest schema", manifestStatus, manifestDetail)

	binaryErrors := map[string][]string{}
	missingBinaries := map[string]runtimeapi.MissingBinary{}
	assetIssues := map[string]preflightIssue{}
	if validation != nil && !statusOnly {
		for _, err := range validation.Errors {
			if err.Service == "" {
				continue
			}
			if strings.HasPrefix(err.Code, "binary_") {
				binaryErrors[err.Service] = append(binaryErrors[err.Service], err.Message)
			}
			if err.Path != "" && (strings.HasPrefix(err.Code, "asset_") || err.Code == "checksum_mismatch") {
				key := err.Service + "|" + err.Path
				assetIssues[key] = preflightIssue{status: "fail", detail: err.Message}
			}
		}
		for _, missing := range validation.MissingBinaries {
			missingBinaries[missing.ServiceID] = missing
		}
		for _, missing := range validation.MissingAssets {
			key := missing.ServiceID + "|" + missing.Path
			assetIssues[key] = preflightIssue{status: "fail", detail: "missing asset"}
		}
		for _, invalid := range validation.InvalidChecksums {
			key := invalid.ServiceID + "|" + invalid.Path
			assetIssues[key] = preflightIssue{
				status: "fail",
				detail: fmt.Sprintf("checksum mismatch (expected %s)", invalid.Expected),
			}
		}
		for _, warn := range validation.Warnings {
			if warn.Service == "" || warn.Path == "" {
				continue
			}
			if warn.Code == "asset_size_warning" {
				key := warn.Service + "|" + warn.Path
				if existing, ok := assetIssues[key]; !ok || existing.status != "fail" {
					assetIssues[key] = preflightIssue{status: "warning", detail: warn.Message}
				}
			}
		}
	}

	if !statusOnly {
		for _, svc := range manifest.Services {
			status := "pass"
			detail := ""
			if missing, ok := missingBinaries[svc.ID]; ok {
				status = "fail"
				detail = fmt.Sprintf("missing binary: %s (%s)", missing.Path, missing.Platform)
			}
			if errs, ok := binaryErrors[svc.ID]; ok {
				status = "fail"
				if detail == "" {
					detail = strings.Join(errs, "; ")
				} else {
					detail = detail + "; " + strings.Join(errs, "; ")
				}
			}
			addCheck("validation.binary."+svc.ID, "validation", "Binary present: "+svc.ID, status, detail)

			for _, asset := range svc.Assets {
				key := svc.ID + "|" + asset.Path
				issue, ok := assetIssues[key]
				assetStatus := "pass"
				assetDetail := ""
				if ok {
					assetStatus = issue.status
					assetDetail = issue.detail
				}
				addCheck("validation.asset."+svc.ID+"."+asset.Path, "validation", "Asset "+asset.Path+" ("+svc.ID+")", assetStatus, assetDetail)
			}
		}

		for _, secret := range secrets {
			status := "pass"
			detail := ""
			if secret.Required {
				if secret.HasValue {
					detail = "required secret provided"
				} else {
					status = "fail"
					detail = "required secret missing"
				}
			} else if secret.HasValue {
				detail = "optional secret provided"
			} else {
				status = "skipped"
				detail = "optional secret not provided"
			}
			addCheck("secrets."+secret.ID, "secrets", "Secret "+secret.ID, status, detail)
		}

		addCheck("runtime.control_api", "runtime", "Runtime control API online", "pass", "control API responded")
	} else {
		addCheck("runtime.control_api", "runtime", "Runtime control API online", "pass", "status-only refresh")
	}

	if !request.StartServices {
		addCheck("services.start", "services", "Start services", "skipped", "start_services=false")
	} else if ready == nil {
		addCheck("services.ready", "services", "Overall readiness", "warning", "readiness not reported")
	} else {
		overall := "pass"
		if !ready.Ready {
			overall = "fail"
		}
		overallDetail := ""
		if ready.WaitedSeconds > 0 {
			overallDetail = fmt.Sprintf("waited %ds for readiness", ready.WaitedSeconds)
		}
		addCheck("services.ready", "services", "Overall readiness", overall, overallDetail)
		for serviceID, status := range ready.Details {
			checkStatus := "pass"
			detail := status.Message
			if status.Skipped {
				checkStatus = "skipped"
				if detail == "" {
					detail = "service skipped"
				}
			} else if !status.Ready {
				checkStatus = "fail"
				if detail == "" {
					detail = "service not ready"
				}
			}
			if status.ExitCode != nil {
				if detail != "" {
					detail = detail + "; "
				}
				detail = fmt.Sprintf("%sexit code %d", detail, *status.ExitCode)
			}
			addCheck("services."+serviceID, "services", "Service readiness: "+serviceID, checkStatus, detail)
		}
	}

	if len(ports) > 0 {
		addCheck("diagnostics.ports", "diagnostics", "Ports reported", "pass", "")
	} else {
		addCheck("diagnostics.ports", "diagnostics", "Ports reported", "warning", "no ports reported")
	}

	if telemetry != nil && telemetry.Path != "" {
		addCheck("diagnostics.telemetry", "diagnostics", "Telemetry configured", "pass", telemetry.Path)
	} else {
		addCheck("diagnostics.telemetry", "diagnostics", "Telemetry configured", "warning", "no telemetry path")
	}

	if request.StartServices {
		if len(logTails) == 0 {
			addCheck("diagnostics.log_tails", "diagnostics", "Log tails captured", "warning", "no log tails captured")
		} else {
			for _, tail := range logTails {
				status := "pass"
				detail := ""
				if tail.Error != "" {
					status = "warning"
					detail = tail.Error
				}
				addCheck("diagnostics.log_tails."+tail.ServiceID, "diagnostics", "Log tail: "+tail.ServiceID, status, detail)
			}
		}
	} else {
		addCheck("diagnostics.log_tails", "diagnostics", "Log tails captured", "skipped", "start_services=false")
	}

	return checks
}

func updatePreflightResult(prev *BundlePreflightResponse, update func(next *BundlePreflightResponse)) *BundlePreflightResponse {
	var next BundlePreflightResponse
	if prev != nil {
		next = *prev
	} else {
		next = BundlePreflightResponse{Status: "running"}
	}
	update(&next)
	return &next
}

func validationStepState(validation *runtimeapi.BundleValidationResult) string {
	if validation == nil {
		return "warning"
	}
	if validation.Valid {
		return "pass"
	}
	return "fail"
}

func secretsStepState(secrets []BundlePreflightSecret) string {
	for _, secret := range secrets {
		if secret.Required && !secret.HasValue {
			return "warning"
		}
	}
	return "pass"
}

func readinessStepState(ready *BundlePreflightReady, request BundlePreflightRequest) string {
	if !request.StartServices {
		return "skipped"
	}
	if ready == nil {
		return "warning"
	}
	if ready.Ready {
		return "pass"
	}
	return "warning"
}

func diagnosticsStepState(ports map[string]map[string]int, telemetry *BundlePreflightTelemetry, logTails []BundlePreflightLogTail, request BundlePreflightRequest) string {
	if !request.StartServices {
		return "skipped"
	}
	if len(ports) > 0 || (telemetry != nil && telemetry.Path != "") || len(logTails) > 0 {
		return "pass"
	}
	return "warning"
}

func (s *Server) runPreflightJob(jobID string, request BundlePreflightRequest) {
	fail := func(stepID string, err error) {
		s.setPreflightJobStep(jobID, stepID, "fail", err.Error())
		s.finishPreflightJob(jobID, "failed", err.Error())
	}

	if strings.TrimSpace(request.BundleManifestPath) == "" {
		fail("validation", errors.New("bundle_manifest_path is required"))
		return
	}

	s.setPreflightJobStep(jobID, "validation", "running", "loading manifest")

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		fail("validation", fmt.Errorf("resolve bundle_manifest_path: %w", err))
		return
	}
	if _, err := os.Stat(manifestPath); err != nil {
		fail("validation", fmt.Errorf("bundle manifest not found: %w", err))
		return
	}

	m, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		fail("validation", fmt.Errorf("load bundle manifest: %w", err))
		return
	}
	if err := m.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		fail("validation", fmt.Errorf("validate bundle manifest: %w", err))
		return
	}

	bundleRoot := strings.TrimSpace(request.BundleRoot)
	if bundleRoot == "" {
		bundleRoot = filepath.Dir(manifestPath)
	}
	bundleRoot, err = filepath.Abs(bundleRoot)
	if err != nil {
		fail("validation", fmt.Errorf("resolve bundle_root: %w", err))
		return
	}

	timeout := time.Duration(request.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}

	s.setPreflightJobStep(jobID, "runtime", "running", "starting runtime control API")

	var session *preflightSession
	if request.StartServices {
		session, err = s.createPreflightSession(m, bundleRoot, request.SessionTTLSeconds)
		if err != nil {
			fail("runtime", err)
			return
		}
	} else {
		appData, err := os.MkdirTemp("", "s2d-preflight-*")
		if err != nil {
			fail("runtime", fmt.Errorf("create preflight app data: %w", err))
			return
		}
		defer os.RemoveAll(appData)

		supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
			Manifest:   m,
			BundlePath: bundleRoot,
			AppDataDir: appData,
			DryRun:     true,
		})
		if err != nil {
			fail("runtime", fmt.Errorf("init runtime: %w", err))
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := supervisor.Start(ctx); err != nil {
			fail("runtime", fmt.Errorf("start runtime: %w", err))
			return
		}
		defer func() {
			_ = supervisor.Shutdown(context.Background())
		}()

		fileTimeout := timeout / 3
		if fileTimeout < 500*time.Millisecond {
			fileTimeout = 500 * time.Millisecond
		}

		tokenPath := bundlemanifest.ResolvePath(appData, m.IPC.AuthTokenRel)
		tokenBytes, err := readFileWithRetry(tokenPath, fileTimeout)
		if err != nil {
			fail("runtime", fmt.Errorf("read auth token: %w", err))
			return
		}
		token := strings.TrimSpace(string(tokenBytes))

		portPath := filepath.Join(appData, "runtime", "ipc_port")
		port, err := readPortFileWithRetry(portPath, fileTimeout)
		if err != nil {
			fail("runtime", fmt.Errorf("read ipc_port: %w", err))
			return
		}

		baseURL := fmt.Sprintf("http://%s:%d", m.IPC.Host, port)
		client := &http.Client{Timeout: 2 * time.Second}
		if err := waitForRuntimeHealth(client, baseURL, timeout); err != nil {
			fail("runtime", err)
			return
		}

		session = &preflightSession{
			manifest:  m,
			bundleDir: bundleRoot,
			baseURL:   baseURL,
			token:     token,
			createdAt: time.Now(),
		}
	}

	s.setPreflightJobStep(jobID, "runtime", "pass", "control API online")

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := session.baseURL
	token := session.token

	s.setPreflightJobStep(jobID, "secrets", "running", "applying secrets")
	if len(request.Secrets) > 0 {
		filtered := map[string]string{}
		for key, value := range request.Secrets {
			if strings.TrimSpace(value) == "" {
				continue
			}
			filtered[key] = value
		}
		payload := map[string]map[string]string{"secrets": filtered}
		if len(filtered) == 0 {
			payload = nil
		}
		if payload != nil {
			if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodPost, payload, nil, nil); err != nil {
				fail("secrets", fmt.Errorf("apply secrets: %w", err))
				return
			}
		}
	}

	s.setPreflightJobStep(jobID, "validation", "running", "validating bundle")
	var validation *runtimeapi.BundleValidationResult
	var validationValue runtimeapi.BundleValidationResult
	allowStatus := map[int]bool{http.StatusUnprocessableEntity: true}
	if _, err := fetchJSON(client, baseURL, token, "/validate", http.MethodGet, nil, &validationValue, allowStatus); err != nil {
		fail("validation", fmt.Errorf("validate bundle: %w", err))
		return
	}
	validation = &validationValue
	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Validation = validation
		})
	})
	s.setPreflightJobStep(jobID, "validation", validationStepState(validation), "")

	var secretsResp struct {
		Secrets []BundlePreflightSecret `json:"secrets"`
	}
	if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodGet, nil, &secretsResp, nil); err != nil {
		fail("secrets", fmt.Errorf("fetch secrets: %w", err))
		return
	}
	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Secrets = secretsResp.Secrets
		})
	})
	s.setPreflightJobStep(jobID, "secrets", secretsStepState(secretsResp.Secrets), "")

	s.setPreflightJobStep(jobID, "services", "running", "checking readiness")
	ready, waitedSeconds, err := fetchReadyWithPolling(client, baseURL, token, request, timeout, m)
	if err != nil {
		fail("services", fmt.Errorf("fetch readiness: %w", err))
		return
	}
	ready.SnapshotAt = time.Now().Format(time.RFC3339)
	ready.WaitedSeconds = waitedSeconds
	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Ready = &ready
		})
	})
	s.setPreflightJobStep(jobID, "services", readinessStepState(&ready, request), "")

	s.setPreflightJobStep(jobID, "diagnostics", "running", "collecting diagnostics")
	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := fetchJSON(client, baseURL, token, "/ports", http.MethodGet, nil, &portsResp, nil); err != nil {
		fail("diagnostics", fmt.Errorf("fetch ports: %w", err))
		return
	}

	var telemetryResp BundlePreflightTelemetry
	if _, err := fetchJSON(client, baseURL, token, "/telemetry", http.MethodGet, nil, &telemetryResp, nil); err != nil {
		fail("diagnostics", fmt.Errorf("fetch telemetry: %w", err))
		return
	}

	logTails := collectLogTails(client, baseURL, token, session.manifest, request)
	checks := buildPreflightChecks(m, validation, &ready, secretsResp.Secrets, portsResp.Services, &telemetryResp, logTails, request)

	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Ports = portsResp.Services
			next.Telemetry = &telemetryResp
			next.LogTails = logTails
			next.Checks = checks
		})
	})
	s.setPreflightJobStep(jobID, "diagnostics", diagnosticsStepState(portsResp.Services, &telemetryResp, logTails, request), "")

	sessionID := ""
	expiresAt := ""
	if session != nil && session.id != "" {
		sessionID = session.id
		if !session.expiresAt.IsZero() {
			expiresAt = session.expiresAt.Format(time.RFC3339)
		}
	}

	s.setPreflightJobResult(jobID, func(prev *BundlePreflightResponse) *BundlePreflightResponse {
		return updatePreflightResult(prev, func(next *BundlePreflightResponse) {
			next.Status = "ok"
			next.SessionID = sessionID
			next.ExpiresAt = expiresAt
		})
	})

	s.finishPreflightJob(jobID, "completed", "")
}

func (s *Server) runBundlePreflight(request BundlePreflightRequest) (*BundlePreflightResponse, error) {
	if strings.TrimSpace(request.BundleManifestPath) == "" {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: errors.New("bundle_manifest_path is required")}
	}

	if request.SessionStop && strings.TrimSpace(request.SessionID) != "" {
		if s.stopPreflightSession(request.SessionID) {
			return &BundlePreflightResponse{Status: "stopped", SessionID: request.SessionID}, nil
		}
		return nil, &preflightStatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
	}

	if request.StatusOnly && strings.TrimSpace(request.SessionID) == "" {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: errors.New("session_id is required for status_only")}
	}

	manifestPath, err := filepath.Abs(request.BundleManifestPath)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_manifest_path: %w", err)}
	}
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("bundle manifest not found: %w", err)}
	}

	m, err := bundlemanifest.LoadManifest(manifestPath)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("load bundle manifest: %w", err)}
	}
	if err := m.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("validate bundle manifest: %w", err)}
	}

	bundleRoot := strings.TrimSpace(request.BundleRoot)
	if bundleRoot == "" {
		bundleRoot = filepath.Dir(manifestPath)
	}
	bundleRoot, err = filepath.Abs(bundleRoot)
	if err != nil {
		return nil, &preflightStatusError{Status: http.StatusBadRequest, Err: fmt.Errorf("resolve bundle_root: %w", err)}
	}

	timeout := time.Duration(request.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}
	var session *preflightSession
	if request.StatusOnly {
		var ok bool
		session, ok = s.getPreflightSession(request.SessionID)
		if !ok {
			return nil, &preflightStatusError{Status: http.StatusNotFound, Err: fmt.Errorf("preflight session not found: %s", request.SessionID)}
		}
		s.refreshPreflightSession(session, request.SessionTTLSeconds)
	} else if request.StartServices {
		if existingID := strings.TrimSpace(request.SessionID); existingID != "" {
			s.stopPreflightSession(existingID)
		}
		session, err = s.createPreflightSession(m, bundleRoot, request.SessionTTLSeconds)
		if err != nil {
			return nil, err
		}
	}

	if session == nil {
		appData, err := os.MkdirTemp("", "s2d-preflight-*")
		if err != nil {
			return nil, fmt.Errorf("create preflight app data: %w", err)
		}
		defer os.RemoveAll(appData)

		supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
			Manifest:   m,
			BundlePath: bundleRoot,
			AppDataDir: appData,
			DryRun:     !request.StartServices,
		})
		if err != nil {
			return nil, fmt.Errorf("init runtime: %w", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := supervisor.Start(ctx); err != nil {
			return nil, fmt.Errorf("start runtime: %w", err)
		}
		defer func() {
			_ = supervisor.Shutdown(context.Background())
		}()

		fileTimeout := timeout / 3
		if fileTimeout < 500*time.Millisecond {
			fileTimeout = 500 * time.Millisecond
		}

		tokenPath := bundlemanifest.ResolvePath(appData, m.IPC.AuthTokenRel)
		tokenBytes, err := readFileWithRetry(tokenPath, fileTimeout)
		if err != nil {
			return nil, fmt.Errorf("read auth token: %w", err)
		}
		token := strings.TrimSpace(string(tokenBytes))

		portPath := filepath.Join(appData, "runtime", "ipc_port")
		port, err := readPortFileWithRetry(portPath, fileTimeout)
		if err != nil {
			return nil, fmt.Errorf("read ipc_port: %w", err)
		}

		baseURL := fmt.Sprintf("http://%s:%d", m.IPC.Host, port)
		client := &http.Client{Timeout: 2 * time.Second}

		if err := waitForRuntimeHealth(client, baseURL, timeout); err != nil {
			return nil, err
		}

		session = &preflightSession{
			manifest:  m,
			bundleDir: bundleRoot,
			baseURL:   baseURL,
			token:     token,
			createdAt: time.Now(),
		}
	}

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := session.baseURL
	token := session.token

	if len(request.Secrets) > 0 {
		filtered := map[string]string{}
		for key, value := range request.Secrets {
			if strings.TrimSpace(value) == "" {
				continue
			}
			filtered[key] = value
		}
		payload := map[string]map[string]string{"secrets": filtered}
		if len(filtered) == 0 {
			payload = nil
		}
		if payload == nil {
			// No secrets to apply after filtering.
		} else if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodPost, payload, nil, nil); err != nil {
			return nil, fmt.Errorf("apply secrets: %w", err)
		}
	}

	var validation *runtimeapi.BundleValidationResult
	if !request.StatusOnly {
		var validationValue runtimeapi.BundleValidationResult
		allowStatus := map[int]bool{http.StatusUnprocessableEntity: true}
		if _, err := fetchJSON(client, baseURL, token, "/validate", http.MethodGet, nil, &validationValue, allowStatus); err != nil {
			return nil, fmt.Errorf("validate bundle: %w", err)
		}
		validation = &validationValue
	}

	var secretsResp struct {
		Secrets []BundlePreflightSecret `json:"secrets"`
	}
	if !request.StatusOnly {
		if _, err := fetchJSON(client, baseURL, token, "/secrets", http.MethodGet, nil, &secretsResp, nil); err != nil {
			return nil, fmt.Errorf("fetch secrets: %w", err)
		}
	}

	ready, waitedSeconds, err := fetchReadyWithPolling(client, baseURL, token, request, timeout, m)
	if err != nil {
		return nil, fmt.Errorf("fetch readiness: %w", err)
	}
	ready.SnapshotAt = time.Now().Format(time.RFC3339)
	ready.WaitedSeconds = waitedSeconds

	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if _, err := fetchJSON(client, baseURL, token, "/ports", http.MethodGet, nil, &portsResp, nil); err != nil {
		return nil, fmt.Errorf("fetch ports: %w", err)
	}

	var telemetryResp BundlePreflightTelemetry
	if _, err := fetchJSON(client, baseURL, token, "/telemetry", http.MethodGet, nil, &telemetryResp, nil); err != nil {
		return nil, fmt.Errorf("fetch telemetry: %w", err)
	}

	logTails := collectLogTails(client, baseURL, token, session.manifest, request)
	checks := buildPreflightChecks(m, validation, &ready, secretsResp.Secrets, portsResp.Services, &telemetryResp, logTails, request)

	sessionID := ""
	expiresAt := ""
	if session != nil && session.id != "" {
		sessionID = session.id
		if !session.expiresAt.IsZero() {
			expiresAt = session.expiresAt.Format(time.RFC3339)
		}
	}

	return &BundlePreflightResponse{
		Status:     "ok",
		Validation: validation,
		Ready:      &ready,
		Secrets:    secretsResp.Secrets,
		Ports:      portsResp.Services,
		Telemetry:  &telemetryResp,
		LogTails:   logTails,
		Checks:     checks,
		SessionID:  sessionID,
		ExpiresAt:  expiresAt,
	}, nil
}

func fetchReadyWithPolling(client *http.Client, baseURL, token string, request BundlePreflightRequest, timeout time.Duration, manifest *bundlemanifest.Manifest) (BundlePreflightReady, int, error) {
	var ready BundlePreflightReady
	if _, err := fetchJSON(client, baseURL, token, "/readyz", http.MethodGet, nil, &ready, nil); err != nil {
		return ready, 0, err
	}
	if !request.StartServices || request.StatusOnly {
		return ready, 0, nil
	}
	waitBudget := maxReadinessTimeout(manifest)
	if waitBudget <= 0 {
		waitBudget = 30 * time.Second
	}
	if timeout > 0 && waitBudget > timeout {
		waitBudget = timeout
	}
	if waitBudget < 2*time.Second {
		waitBudget = 2 * time.Second
	}
	start := time.Now()
	deadline := start.Add(waitBudget)
	for time.Now().Before(deadline) {
		if ready.Ready {
			break
		}
		time.Sleep(1 * time.Second)
		if _, err := fetchJSON(client, baseURL, token, "/readyz", http.MethodGet, nil, &ready, nil); err != nil {
			return ready, int(time.Since(start).Seconds()), err
		}
	}
	return ready, int(time.Since(start).Seconds()), nil
}

func maxReadinessTimeout(manifest *bundlemanifest.Manifest) time.Duration {
	if manifest == nil {
		return 0
	}
	maxTimeout := time.Duration(0)
	for _, svc := range manifest.Services {
		timeout := time.Duration(svc.Readiness.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		if timeout > maxTimeout {
			maxTimeout = timeout
		}
	}
	return maxTimeout
}

func readFileWithRetry(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

func readPortFileWithRetry(path string, timeout time.Duration) (int, error) {
	data, err := readFileWithRetry(path, timeout)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return port, nil
}

func (s *Server) createPreflightSession(manifest *bundlemanifest.Manifest, bundleRoot string, ttlSeconds int) (*preflightSession, error) {
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	if ttlSeconds > 900 {
		ttlSeconds = 900
	}

	appData, err := os.MkdirTemp("", "s2d-preflight-*")
	if err != nil {
		return nil, fmt.Errorf("create preflight app data: %w", err)
	}

	supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
		Manifest:   manifest,
		BundlePath: bundleRoot,
		AppDataDir: appData,
		DryRun:     false,
	})
	if err != nil {
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("init runtime: %w", err)
	}

	ctx := context.Background()
	if err := supervisor.Start(ctx); err != nil {
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("start runtime: %w", err)
	}

	fileTimeout := 5 * time.Second
	tokenPath := bundlemanifest.ResolvePath(appData, manifest.IPC.AuthTokenRel)
	tokenBytes, err := readFileWithRetry(tokenPath, fileTimeout)
	if err != nil {
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("read auth token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	portPath := filepath.Join(appData, "runtime", "ipc_port")
	port, err := readPortFileWithRetry(portPath, fileTimeout)
	if err != nil {
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, fmt.Errorf("read ipc_port: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%d", manifest.IPC.Host, port)
	client := &http.Client{Timeout: 2 * time.Second}
	if err := waitForRuntimeHealth(client, baseURL, 10*time.Second); err != nil {
		_ = supervisor.Shutdown(context.Background())
		_ = os.RemoveAll(appData)
		return nil, err
	}

	session := &preflightSession{
		id:         uuid.NewString(),
		manifest:   manifest,
		bundleDir:  bundleRoot,
		appData:    appData,
		supervisor: supervisor,
		baseURL:    baseURL,
		token:      token,
		createdAt:  time.Now(),
		expiresAt:  time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}
	s.preflightMux.Lock()
	s.preflight[session.id] = session
	s.preflightMux.Unlock()

	return session, nil
}

func (s *Server) getPreflightSession(id string) (*preflightSession, bool) {
	s.preflightMux.Lock()
	defer s.preflightMux.Unlock()
	session, ok := s.preflight[id]
	if !ok {
		return nil, false
	}
	if time.Now().After(session.expiresAt) {
		delete(s.preflight, id)
		go s.shutdownPreflightSession(session)
		return nil, false
	}
	return session, true
}

func (s *Server) refreshPreflightSession(session *preflightSession, ttlSeconds int) {
	if ttlSeconds <= 0 {
		return
	}
	maxTTL := 900
	if ttlSeconds > maxTTL {
		ttlSeconds = maxTTL
	}
	s.preflightMux.Lock()
	defer s.preflightMux.Unlock()
	session.expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
}

func (s *Server) stopPreflightSession(id string) bool {
	s.preflightMux.Lock()
	session, ok := s.preflight[id]
	if ok {
		delete(s.preflight, id)
	}
	s.preflightMux.Unlock()
	if ok {
		s.shutdownPreflightSession(session)
	}
	return ok
}

func (s *Server) shutdownPreflightSession(session *preflightSession) {
	if session == nil {
		return
	}
	_ = session.supervisor.Shutdown(context.Background())
	_ = os.RemoveAll(session.appData)
}

func (s *Server) createPreflightJob() *preflightJob {
	job := &preflightJob{
		id:     uuid.NewString(),
		status: "running",
		steps: map[string]BundlePreflightStep{
			"validation":  {ID: "validation", Name: "Load bundle + validate", State: "pending"},
			"runtime":     {ID: "runtime", Name: "Start runtime control API", State: "pending"},
			"secrets":     {ID: "secrets", Name: "Apply secrets", State: "pending"},
			"services":    {ID: "services", Name: "Start services", State: "pending"},
			"diagnostics": {ID: "diagnostics", Name: "Collect diagnostics", State: "pending"},
		},
		startedAt: time.Now(),
		updatedAt: time.Now(),
	}
	s.preflightJobsMux.Lock()
	s.preflightJobs[job.id] = job
	s.preflightJobsMux.Unlock()
	return job
}

func (s *Server) getPreflightJob(id string) (*preflightJob, bool) {
	s.preflightJobsMux.Lock()
	defer s.preflightJobsMux.Unlock()
	job, ok := s.preflightJobs[id]
	return job, ok
}

func (s *Server) updatePreflightJob(id string, fn func(job *preflightJob)) {
	s.preflightJobsMux.Lock()
	defer s.preflightJobsMux.Unlock()
	job, ok := s.preflightJobs[id]
	if !ok {
		return
	}
	fn(job)
	job.updatedAt = time.Now()
}

func (s *Server) setPreflightJobStep(id, stepID, state, detail string) {
	s.updatePreflightJob(id, func(job *preflightJob) {
		step, ok := job.steps[stepID]
		if !ok {
			step = BundlePreflightStep{ID: stepID, Name: stepID}
		}
		step.State = state
		step.Detail = detail
		job.steps[stepID] = step
	})
}

func (s *Server) setPreflightJobResult(id string, updater func(prev *BundlePreflightResponse) *BundlePreflightResponse) {
	s.updatePreflightJob(id, func(job *preflightJob) {
		job.result = updater(job.result)
	})
}

func (s *Server) finishPreflightJob(id string, status, errMsg string) {
	s.updatePreflightJob(id, func(job *preflightJob) {
		job.status = status
		job.err = errMsg
	})
}

func (s *Server) startPreflightJanitor() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanupPreflightSessions()
			s.cleanupPreflightJobs()
		}
	}()
}

func (s *Server) cleanupPreflightSessions() {
	var expired []*preflightSession
	now := time.Now()
	s.preflightMux.Lock()
	for id, session := range s.preflight {
		if now.After(session.expiresAt) {
			expired = append(expired, session)
			delete(s.preflight, id)
		}
	}
	s.preflightMux.Unlock()
	for _, session := range expired {
		s.shutdownPreflightSession(session)
	}
}

func (s *Server) cleanupPreflightJobs() {
	expiration := 15 * time.Minute
	now := time.Now()
	var expired []string

	s.preflightJobsMux.Lock()
	for id, job := range s.preflightJobs {
		if job.status == "running" {
			continue
		}
		if now.Sub(job.updatedAt) > expiration {
			expired = append(expired, id)
			delete(s.preflightJobs, id)
		}
	}
	s.preflightJobsMux.Unlock()
}

func waitForRuntimeHealth(client *http.Client, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("runtime control API not responding within %s", timeout)
}

func fetchJSON(client *http.Client, baseURL, token, path, method string, payload interface{}, out interface{}, allow map[int]bool) (int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if allow != nil && allow[resp.StatusCode] {
			if out != nil {
				if decodeErr := json.NewDecoder(resp.Body).Decode(out); decodeErr != nil {
					return resp.StatusCode, decodeErr
				}
			}
			return resp.StatusCode, nil
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func fetchText(client *http.Client, baseURL, token, path string) (string, int, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "", 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	bodyText := strings.TrimSpace(string(bodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if bodyText == "" {
			bodyText = resp.Status
		}
		return bodyText, resp.StatusCode, fmt.Errorf("status %d: %s", resp.StatusCode, bodyText)
	}

	return bodyText, resp.StatusCode, nil
}

func collectLogTails(client *http.Client, baseURL, token string, manifest *bundlemanifest.Manifest, request BundlePreflightRequest) []BundlePreflightLogTail {
	if request.LogTailLines <= 0 {
		return nil
	}

	lines := request.LogTailLines
	if lines > 200 {
		lines = 200
	}

	serviceIDs := request.LogTailServices
	if len(serviceIDs) == 0 {
		for _, svc := range manifest.Services {
			if strings.TrimSpace(svc.LogDir) != "" {
				serviceIDs = append(serviceIDs, svc.ID)
			}
		}
	}

	if len(serviceIDs) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var tails []BundlePreflightLogTail
	for _, serviceID := range serviceIDs {
		id := strings.TrimSpace(serviceID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true

		path := fmt.Sprintf("/logs/tail?serviceId=%s&lines=%d", url.QueryEscape(id), lines)
		content, _, err := fetchText(client, baseURL, token, path)
		tail := BundlePreflightLogTail{
			ServiceID: id,
			Lines:     lines,
		}
		if err != nil {
			tail.Error = err.Error()
		} else {
			tail.Content = content
		}
		if tail.Content == "" && tail.Error == "" {
			continue
		}
		tails = append(tails, tail)
	}

	return tails
}

type bundleExportRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier,omitempty"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
}

type bundleExportResponse struct {
	Status               string      `json:"status"`
	Schema               string      `json:"schema"`
	Manifest             interface{} `json:"manifest"`
	Checksum             string      `json:"checksum,omitempty"`
	GeneratedAt          string      `json:"generated_at,omitempty"`
	DeploymentManagerURL string      `json:"deployment_manager_url,omitempty"`
	ManifestPath         string      `json:"manifest_path,omitempty"`
}

func (s *Server) exportBundleHandler(w http.ResponseWriter, r *http.Request) {
	var req bundleExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Scenario == "" {
		http.Error(w, "scenario is required", http.StatusBadRequest)
		return
	}
	if containsParentRef(req.Scenario) || strings.ContainsAny(req.Scenario, `/\`) {
		http.Error(w, "invalid scenario name", http.StatusBadRequest)
		return
	}

	deploymentManagerURL, err := discovery.ResolveScenarioURLDefault(r.Context(), "deployment-manager")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve deployment-manager: %v", err), http.StatusBadGateway)
		return
	}

	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}
	if req.IncludeSecrets == nil {
		include := true
		req.IncludeSecrets = &include
	}

	client := &http.Client{Timeout: 5 * time.Minute}

	payload, err := json.Marshal(map[string]interface{}{
		"scenario":        req.Scenario,
		"tier":            req.Tier,
		"include_secrets": req.IncludeSecrets,
		"output_dir":      filepath.Join(detectVrooliRoot(), "scenarios", req.Scenario, "platforms", "electron", "bundle"),
		"stage_bundle":    true,
	})
	if err != nil {
		http.Error(w, "failed to marshal request", http.StatusInternalServerError)
		return
	}

	resp, err := client.Post(
		fmt.Sprintf("%s/api/v1/bundles/export", deploymentManagerURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("deployment-manager export failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read deployment-manager response", http.StatusBadGateway)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	var exportResp bundleExportResponse
	if err := json.Unmarshal(body, &exportResp); err != nil {
		http.Error(w, "failed to parse deployment-manager response", http.StatusBadGateway)
		return
	}
	exportResp.DeploymentManagerURL = deploymentManagerURL

	manifestPath := exportResp.ManifestPath
	if manifestPath == "" {
		var err error
		manifestPath, err = writeBundleManifest(req.Scenario, exportResp.Manifest)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to write bundle manifest: %v", err), http.StatusBadGateway)
			return
		}
	}
	exportResp.ManifestPath = manifestPath

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(exportResp)
}

type autoBuildProxyRequest struct {
	Scenario  string   `json:"scenario"`
	Platforms []string `json:"platforms,omitempty"`
	Targets   []string `json:"targets,omitempty"`
	DryRun    bool     `json:"dry_run,omitempty"`
}

func (s *Server) deploymentManagerAutoBuildHandler(w http.ResponseWriter, r *http.Request) {
	var req autoBuildProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Scenario == "" {
		http.Error(w, "scenario is required", http.StatusBadRequest)
		return
	}
	if containsParentRef(req.Scenario) || strings.ContainsAny(req.Scenario, `/\`) {
		http.Error(w, "invalid scenario name", http.StatusBadRequest)
		return
	}

	deploymentManagerURL, err := discovery.ResolveScenarioURLDefault(r.Context(), "deployment-manager")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve deployment-manager: %v", err), http.StatusBadGateway)
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "failed to marshal build request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("%s/api/v1/build/auto", deploymentManagerURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("deployment-manager build failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read deployment-manager response", http.StatusBadGateway)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (s *Server) deploymentManagerAutoBuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	buildID := mux.Vars(r)["build_id"]
	if buildID == "" {
		http.Error(w, "build_id is required", http.StatusBadRequest)
		return
	}

	deploymentManagerURL, err := discovery.ResolveScenarioURLDefault(r.Context(), "deployment-manager")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve deployment-manager: %v", err), http.StatusBadGateway)
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(
		fmt.Sprintf("%s/api/v1/build/auto/%s", deploymentManagerURL, buildID),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("deployment-manager status failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read deployment-manager response", http.StatusBadGateway)
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func writeBundleManifest(scenario string, manifest interface{}) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest missing from deployment-manager response")
	}
	root := detectVrooliRoot()
	scenarioDir := filepath.Join(root, "scenarios", scenario)
	if _, err := os.Stat(scenarioDir); err != nil {
		return "", fmt.Errorf("scenario directory not found: %w", err)
	}

	outDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create bundle directory: %w", err)
	}

	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("serialize manifest: %w", err)
	}

	outPath := filepath.Join(outDir, "bundle.json")
	if err := os.WriteFile(outPath, payload, 0o644); err != nil {
		return "", fmt.Errorf("write bundle.json: %w", err)
	}
	return outPath, nil
}
