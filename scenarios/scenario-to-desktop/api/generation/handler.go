package generation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"
)

const (
	defaultTemplateType = "universal"
)

var validTemplateTypes = []string{"universal", "basic", "advanced", "multi_window", "kiosk"}

// Handler provides HTTP handlers for generation operations.
type Handler struct {
	service         *DefaultService
	configValidator ConfigValidator
	configPersister ConfigPersister
	configLoader    ConfigLoader
	recordDeleter   RecordDeleter
	logger          *slog.Logger
}

// ConfigValidator validates desktop configurations.
type ConfigValidator interface {
	Validate(config *DesktopConfig) error
	ValidateAndPrepareBundle(config *DesktopConfig) error
}

// ConfigLoader loads saved desktop connection configurations.
type ConfigLoader interface {
	Load(scenarioPath string) (*ConnectionConfig, error)
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithConfigValidator sets the config validator.
func WithConfigValidator(v ConfigValidator) HandlerOption {
	return func(h *Handler) {
		h.configValidator = v
	}
}

// WithConfigPersister sets the config persister.
func WithConfigPersister(p ConfigPersister) HandlerOption {
	return func(h *Handler) {
		h.configPersister = p
	}
}

// WithConfigLoader sets the config loader.
func WithConfigLoader(l ConfigLoader) HandlerOption {
	return func(h *Handler) {
		h.configLoader = l
	}
}

// WithRecordDeleter sets the record deleter.
func WithRecordDeleter(d RecordDeleter) HandlerOption {
	return func(h *Handler) {
		h.recordDeleter = d
	}
}

// WithHandlerLogger sets the logger.
func WithHandlerLogger(l *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = l
	}
}

// NewHandler creates a new generation handler.
func NewHandler(service *DefaultService, opts ...HandlerOption) *Handler {
	h := &Handler{
		service: service,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RegisterRoutes registers generation routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/desktop/generate", h.GenerateHandler).Methods("POST")
	r.HandleFunc("/api/v1/desktop/generate/quick", h.QuickGenerateHandler).Methods("POST")
	r.HandleFunc("/api/v1/desktop/delete/{scenario_name}", h.DeleteHandler).Methods("DELETE")
}

// QuickGenerateHandler handles POST requests for quick desktop generation.
// This endpoint auto-detects scenario configuration and generates a desktop app.
func (h *Handler) QuickGenerateHandler(w http.ResponseWriter, r *http.Request) {
	var request QuickGenerateRequest
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
		request.TemplateType = defaultTemplateType
	}

	// Validate template type
	if !isValidTemplateType(request.TemplateType) {
		http.Error(w, fmt.Sprintf("invalid template_type: %s", request.TemplateType), http.StatusBadRequest)
		return
	}

	// Get analyzer from service
	analyzer := h.service.GetAnalyzer()
	if analyzer == nil {
		http.Error(w, "Analyzer not configured", http.StatusInternalServerError)
		return
	}

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

	// Load saved config
	var savedConfig *ConnectionConfig
	if h.configLoader != nil {
		savedConfig, _ = h.configLoader.Load(metadata.ScenarioPath)
	}

	// Merge request values into config
	config = mergeQuickGenerateConfig(config, request, savedConfig, h.service.StandardOutputPath(config.AppName))

	// Validate and prepare bundle
	if h.configValidator != nil {
		if err := h.configValidator.ValidateAndPrepareBundle(config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Persist config
	if h.configPersister != nil {
		_ = h.configPersister.Save(metadata.ScenarioPath, config)
	}

	// Queue build
	buildStatus := h.service.QueueBuild(config, metadata, true)

	// Return response
	response := GenerateResponse{
		BuildID:             buildStatus.BuildID,
		Status:              "building",
		ScenarioName:        config.AppName,
		DesktopPath:         buildStatus.OutputPath,
		DetectedMetadata:    metadata,
		InstallInstructions: "Run 'npm install && npm run dev' in the output directory",
		TestCommand:         "npm run dev",
		StatusURL:           fmt.Sprintf("/api/v1/desktop/status/%s", buildStatus.BuildID),
	}

	httputil.WriteJSON(w, http.StatusCreated, response)
}

// GenerateHandler handles POST requests for desktop generation with full config.
func (h *Handler) GenerateHandler(w http.ResponseWriter, r *http.Request) {
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

	// Validate and prepare bundle
	if h.configValidator != nil {
		if err := h.configValidator.ValidateAndPrepareBundle(config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Persist config
	if h.configPersister != nil && config.AppName != "" {
		_ = h.configPersister.Save(h.service.ScenarioRoot(config.AppName), config)
	}

	// Queue build
	buildStatus := h.service.QueueBuild(config, nil, false)

	// Return response
	response := GenerateResponse{
		BuildID:             buildStatus.BuildID,
		Status:              "building",
		DesktopPath:         config.OutputPath,
		InstallInstructions: "Run 'npm install && npm run dev' in the output directory",
		TestCommand:         "npm run dev",
		StatusURL:           fmt.Sprintf("/api/v1/desktop/status/%s", buildStatus.BuildID),
	}

	httputil.WriteJSON(w, http.StatusCreated, response)
}

// isValidTemplateType checks if the template type is valid.
func isValidTemplateType(templateType string) bool {
	for _, t := range validTemplateTypes {
		if t == templateType {
			return true
		}
	}
	return false
}

// decodeDesktopConfig decodes a DesktopConfig from JSON bytes.
func decodeDesktopConfig(body []byte) (*DesktopConfig, error) {
	var config DesktopConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// mergeQuickGenerateConfig merges request values and saved config into the generated config.
func mergeQuickGenerateConfig(config *DesktopConfig, request QuickGenerateRequest, savedConfig *ConnectionConfig, defaultOutputPath string) *DesktopConfig {
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

func chooseProxyURL(request QuickGenerateRequest, savedConfig *ConnectionConfig) string {
	if request.ProxyURL != "" {
		return request.ProxyURL
	}
	if savedConfig != nil && savedConfig.ProxyURL != "" {
		return savedConfig.ProxyURL
	}
	if request.LegacyServerURL != "" {
		return request.LegacyServerURL
	}
	if request.LegacyAPIURL != "" {
		return request.LegacyAPIURL
	}
	return ""
}

func chooseVrooliBinary(request QuickGenerateRequest, savedConfig *ConnectionConfig, existing string) string {
	if request.VrooliBinary != "" {
		return request.VrooliBinary
	}
	if savedConfig != nil && savedConfig.VrooliBinaryPath != "" {
		return savedConfig.VrooliBinaryPath
	}
	return existing
}

func chooseDeploymentMode(request QuickGenerateRequest, savedConfig *ConnectionConfig, existing string) string {
	if request.DeploymentMode != "" {
		return request.DeploymentMode
	}
	if savedConfig != nil && savedConfig.DeploymentMode != "" {
		return savedConfig.DeploymentMode
	}
	return existing
}

func chooseBundleManifestPath(request QuickGenerateRequest, savedConfig *ConnectionConfig) string {
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

func chooseAutoManage(request QuickGenerateRequest, savedConfig *ConnectionConfig, existing bool) bool {
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

// DeleteHandler handles DELETE requests to delete a generated desktop application.
func (h *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario_name"]

	if scenarioName == "" {
		httputil.WriteBadRequest(w, "scenario_name is required")
		return
	}

	if !isSafeScenarioName(scenarioName) {
		httputil.WriteBadRequest(w, "Invalid scenario name")
		return
	}

	// Construct desktop path - MUST be exactly platforms/electron/
	desktopPath := h.service.StandardOutputPath(scenarioName)

	// Security check: Verify the path is actually inside platforms/electron
	absDesktopPath, err := filepath.Abs(desktopPath)
	if err != nil {
		httputil.WriteInternalError(w, "Failed to resolve desktop path")
		return
	}

	absExpectedPrefix, _ := filepath.Abs(desktopPath)
	if absDesktopPath != absExpectedPrefix {
		h.logger.Error("path traversal attempt detected",
			"scenario", scenarioName,
			"expected", absExpectedPrefix,
			"actual", absDesktopPath)
		httputil.WriteBadRequest(w, "Security violation: invalid path")
		return
	}

	// Check if desktop directory exists
	_, statErr := os.Stat(desktopPath)
	if statErr == nil {
		// Remove the entire platforms/electron directory
		if err := os.RemoveAll(desktopPath); err != nil {
			h.logger.Error("failed to delete desktop directory",
				"scenario", scenarioName,
				"path", desktopPath,
				"error", err)
			httputil.WriteInternalError(w, fmt.Sprintf("Failed to delete desktop directory: %v", err))
			return
		}
		h.logger.Info("deleted desktop application",
			"scenario", scenarioName,
			"path", desktopPath)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		httputil.WriteInternalError(w, fmt.Sprintf("Failed to read desktop directory: %v", statErr))
		return
	}

	removedRecords := 0
	if h.recordDeleter != nil {
		removedRecords = h.recordDeleter.DeleteByScenario(scenarioName)
	}

	message := fmt.Sprintf("Desktop version of '%s' deleted successfully", scenarioName)
	if errors.Is(statErr, os.ErrNotExist) {
		message = fmt.Sprintf("Desktop version of '%s' was already missing; cleaned up record state.", scenarioName)
	}

	// Return success response
	httputil.WriteJSONOK(w, map[string]interface{}{
		"status":          "success",
		"scenario_name":   scenarioName,
		"deleted_path":    desktopPath,
		"removed_records": removedRecords,
		"message":         message,
	})
}

// isSafeScenarioName checks if the scenario name is safe (no path traversal).
func isSafeScenarioName(name string) bool {
	return !strings.Contains(name, "..") && !strings.Contains(name, "/") && !strings.Contains(name, "\\")
}
