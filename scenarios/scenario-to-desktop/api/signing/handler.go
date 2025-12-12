package signing

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"scenario-to-desktop-api/signing/generation"
	"scenario-to-desktop-api/signing/platforms"
	"scenario-to-desktop-api/signing/validation"
)

// Handler provides HTTP endpoints for signing configuration.
type Handler struct {
	repo          Repository
	validator     Validator
	prereqChecker PrerequisiteChecker
	detector      CertificateDiscoverer
	generator     ConfigGenerator
}

// NewHandler creates a new signing handler.
func NewHandler() *Handler {
	return &Handler{
		repo:          NewFileRepository(),
		validator:     validation.NewValidator(),
		prereqChecker: validation.NewPrerequisiteChecker(),
		detector:      platforms.NewMultiPlatformDetector(),
		generator:     generation.NewGenerator(nil),
	}
}

// RegisterRoutes registers all signing routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// System-level (no scenario context) — register before dynamic routes to avoid shadowing
	r.HandleFunc("/api/v1/signing/prerequisites", h.GetPrerequisites).Methods("GET")
	r.HandleFunc("/api/v1/signing/discover/{platform}", h.DiscoverCertificates).Methods("GET")

	// Signing configuration CRUD
	r.HandleFunc("/api/v1/signing/{scenario}", h.GetConfig).Methods("GET")
	r.HandleFunc("/api/v1/signing/{scenario}", h.PutConfig).Methods("PUT")
	r.HandleFunc("/api/v1/signing/{scenario}/{platform}", h.PatchPlatformConfig).Methods("PATCH")
	r.HandleFunc("/api/v1/signing/{scenario}", h.DeleteConfig).Methods("DELETE")
	r.HandleFunc("/api/v1/signing/{scenario}/{platform}", h.DeletePlatformConfig).Methods("DELETE")

	// Validation and checks
	r.HandleFunc("/api/v1/signing/{scenario}/validate", h.ValidateConfig).Methods("POST")
	r.HandleFunc("/api/v1/signing/{scenario}/ready", h.CheckReady).Methods("GET")
}

// GetConfig returns the signing configuration for a scenario.
// GET /api/v1/signing/{scenario}
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]

	config, err := h.repo.Get(r.Context(), scenario)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get config: "+err.Error())
		return
	}

	response := SigningConfigResponse{
		Scenario:   scenario,
		Config:     config,
		ConfigPath: h.repo.GetPath(scenario),
	}

	writeJSON(w, http.StatusOK, response)
}

// PutConfig sets the full signing configuration for a scenario.
// PUT /api/v1/signing/{scenario}
func (h *Handler) PutConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]

	var config SigningConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate the config structure
	result := h.validator.ValidateConfig(&config)
	if !result.Valid {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "validation failed",
			"validation": result,
		})
		return
	}

	if err := h.repo.Save(r.Context(), scenario, &config); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	response := SigningConfigResponse{
		Scenario:   scenario,
		Config:     &config,
		ConfigPath: h.repo.GetPath(scenario),
	}

	writeJSON(w, http.StatusOK, response)
}

// PatchPlatformConfig updates a specific platform's configuration.
// PATCH /api/v1/signing/{scenario}/{platform}
func (h *Handler) PatchPlatformConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]
	platform := vars["platform"]

	// Get existing config or create new
	config, err := h.repo.Get(r.Context(), scenario)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get config: "+err.Error())
		return
	}
	if config == nil {
		config = NewSigningConfig()
	}

	// Decode the platform-specific config based on platform
	switch platform {
	case PlatformWindows:
		var windowsConfig WindowsSigningConfig
		if err := json.NewDecoder(r.Body).Decode(&windowsConfig); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		config.Windows = &windowsConfig

	case PlatformMacOS:
		var macConfig MacOSSigningConfig
		if err := json.NewDecoder(r.Body).Decode(&macConfig); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		config.MacOS = &macConfig

	case PlatformLinux:
		var linuxConfig LinuxSigningConfig
		if err := json.NewDecoder(r.Body).Decode(&linuxConfig); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		config.Linux = &linuxConfig

	default:
		writeError(w, http.StatusBadRequest, "invalid platform: "+platform)
		return
	}

	// Save the updated config
	if err := h.repo.Save(r.Context(), scenario, config); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	response := SigningConfigResponse{
		Scenario:   scenario,
		Config:     config,
		ConfigPath: h.repo.GetPath(scenario),
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteConfig removes the signing configuration for a scenario.
// DELETE /api/v1/signing/{scenario}
func (h *Handler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]

	if err := h.repo.Delete(r.Context(), scenario); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "deleted",
		"scenario": scenario,
	})
}

// DeletePlatformConfig removes a specific platform's configuration.
// DELETE /api/v1/signing/{scenario}/{platform}
func (h *Handler) DeletePlatformConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]
	platform := vars["platform"]

	if err := h.repo.DeleteForPlatform(r.Context(), scenario, platform); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete platform config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "deleted",
		"scenario": scenario,
		"platform": platform,
	})
}

// ValidateConfig performs full validation (structural + prerequisites) on a signing config.
// POST /api/v1/signing/{scenario}/validate
func (h *Handler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]

	config, err := h.repo.Get(r.Context(), scenario)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get config: "+err.Error())
		return
	}

	if config == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   true,
			"message": "no signing config exists for this scenario",
		})
		return
	}

	// Structural validation
	result := h.validator.ValidateConfig(config)

	// Prerequisite validation
	prereqResult := h.prereqChecker.CheckPrerequisites(r.Context(), config)
	result.Merge(prereqResult)

	writeJSON(w, http.StatusOK, result)
}

// CheckReady performs a quick readiness check for deployment-manager integration.
// GET /api/v1/signing/{scenario}/ready
func (h *Handler) CheckReady(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]

	config, err := h.repo.Get(r.Context(), scenario)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get config: "+err.Error())
		return
	}

	response := ReadinessResponse{
		Scenario:  scenario,
		Platforms: make(map[string]PlatformStatus),
	}

	// No config means no signing configured
	if config == nil || !config.Enabled {
		response.Ready = false
		response.Issues = append(response.Issues, "Signing is not enabled for this scenario")
		writeJSON(w, http.StatusOK, response)
		return
	}

	// Check each configured platform
	var issues []string

	// Windows
	if config.Windows != nil {
		winResult := h.validator.ValidateForPlatform(config, PlatformWindows)
		winReady := winResult.Valid
		winReason := ""
		if !winReady && len(winResult.Errors) > 0 {
			winReason = winResult.Errors[0].Message
			issues = append(issues, "Windows: "+winReason)
		}
		response.Platforms[PlatformWindows] = PlatformStatus{
			Ready:  winReady,
			Reason: winReason,
		}
	} else {
		response.Platforms[PlatformWindows] = PlatformStatus{
			Ready:  false,
			Reason: "Not configured",
		}
	}

	// macOS
	if config.MacOS != nil {
		macResult := h.validator.ValidateForPlatform(config, PlatformMacOS)
		macReady := macResult.Valid
		macReason := ""
		if !macReady && len(macResult.Errors) > 0 {
			macReason = macResult.Errors[0].Message
			issues = append(issues, "macOS: "+macReason)
		}
		response.Platforms[PlatformMacOS] = PlatformStatus{
			Ready:  macReady,
			Reason: macReason,
		}
	} else {
		response.Platforms[PlatformMacOS] = PlatformStatus{
			Ready:  false,
			Reason: "Not configured",
		}
	}

	// Linux
	if config.Linux != nil {
		linuxResult := h.validator.ValidateForPlatform(config, PlatformLinux)
		linuxReady := linuxResult.Valid
		linuxReason := ""
		if !linuxReady && len(linuxResult.Errors) > 0 {
			linuxReason = linuxResult.Errors[0].Message
			issues = append(issues, "Linux: "+linuxReason)
		}
		response.Platforms[PlatformLinux] = PlatformStatus{
			Ready:  linuxReady,
			Reason: linuxReason,
		}
	} else {
		response.Platforms[PlatformLinux] = PlatformStatus{
			Ready:  false,
			Reason: "Not configured",
		}
	}

	// Overall readiness - at least one platform must be ready
	response.Ready = response.Platforms[PlatformWindows].Ready ||
		response.Platforms[PlatformMacOS].Ready ||
		response.Platforms[PlatformLinux].Ready
	response.Issues = issues

	writeJSON(w, http.StatusOK, response)
}

// GetPrerequisites returns available signing tools on the system.
// GET /api/v1/signing/prerequisites
func (h *Handler) GetPrerequisites(w http.ResponseWriter, r *http.Request) {
	tools, err := h.prereqChecker.DetectTools(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to detect tools: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tools": tools,
	})
}

// DiscoverCertificates finds available signing certificates for a platform.
// GET /api/v1/signing/discover/{platform}
func (h *Handler) DiscoverCertificates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	platform := vars["platform"]

	certs, err := h.detector.DiscoverCertificates(context.Background(), platform)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to discover certificates: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"platform":     platform,
		"certificates": certs,
	})
}

// --- Helper Functions ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
