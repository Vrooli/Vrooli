package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
