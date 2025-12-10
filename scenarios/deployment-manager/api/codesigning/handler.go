package codesigning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// Handler handles code signing HTTP requests.
type Handler struct {
	repo      Repository
	validator Validator
	checker   PrerequisiteChecker
	log       func(string, map[string]interface{})
}

// NewHandler creates a new code signing handler.
// The validator and checker should be passed from the server initialization
// to avoid import cycles between codesigning and codesigning/validation.
func NewHandler(repo Repository, validator Validator, checker PrerequisiteChecker, log func(string, map[string]interface{})) *Handler {
	return &Handler{
		repo:      repo,
		validator: validator,
		checker:   checker,
		log:       log,
	}
}

// GetSigning handles GET /api/v1/profiles/{id}/signing
// Returns the full signing configuration for a profile.
func (h *Handler) GetSigning(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}

	config, err := h.repo.Get(r.Context(), profileID)
	if errors.Is(err, ErrProfileNotFound) {
		h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
		return
	}
	if err != nil {
		h.log("error", map[string]interface{}{"msg": "failed to get signing config", "error": err.Error()})
		h.writeError(w, http.StatusInternalServerError, "failed to retrieve signing config")
		return
	}

	// Return empty config if none exists
	if config == nil {
		config = DefaultSigningConfig()
	}

	h.writeJSON(w, http.StatusOK, config)
}

// SetSigning handles PUT /api/v1/profiles/{id}/signing
// Replaces the full signing configuration for a profile.
func (h *Handler) SetSigning(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}

	// Parse request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10)) // 64KB limit
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var config SigningConfig
	if err := json.Unmarshal(body, &config); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate the configuration structurally
	result := h.validator.ValidateConfig(&config)
	if !result.Valid {
		h.writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "validation failed",
			"validation": result,
		})
		return
	}

	// Save the config
	if err := h.repo.Save(r.Context(), profileID, &config); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.log("error", map[string]interface{}{"msg": "failed to save signing config", "error": err.Error()})
		h.writeError(w, http.StatusInternalServerError, "failed to save signing config")
		return
	}

	h.log("info", map[string]interface{}{
		"msg":        "signing config updated",
		"profile_id": profileID,
		"enabled":    config.Enabled,
	})

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "updated",
		"config":  config,
		"message": "Signing configuration saved successfully",
	})
}

// SetPlatformSigning handles PATCH /api/v1/profiles/{id}/signing/{platform}
// Updates only a specific platform's signing configuration.
func (h *Handler) SetPlatformSigning(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	platform := vars["platform"]

	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	if !IsValidPlatform(platform) {
		h.writeError(w, http.StatusBadRequest, "invalid platform: "+platform+". Valid: windows, macos, linux")
		return
	}

	// Parse request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	// Parse platform-specific config
	var platformConfig interface{}
	switch platform {
	case PlatformWindows:
		var winConfig WindowsSigningConfig
		if err := json.Unmarshal(body, &winConfig); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid Windows config: "+err.Error())
			return
		}
		platformConfig = &winConfig
	case PlatformMacOS:
		var macConfig MacOSSigningConfig
		if err := json.Unmarshal(body, &macConfig); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid macOS config: "+err.Error())
			return
		}
		platformConfig = &macConfig
	case PlatformLinux:
		var linuxConfig LinuxSigningConfig
		if err := json.Unmarshal(body, &linuxConfig); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid Linux config: "+err.Error())
			return
		}
		platformConfig = &linuxConfig
	}

	// Save the platform config
	if err := h.repo.SaveForPlatform(r.Context(), profileID, platform, platformConfig); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.log("error", map[string]interface{}{"msg": "failed to save platform signing config", "error": err.Error()})
		h.writeError(w, http.StatusInternalServerError, "failed to save signing config")
		return
	}

	h.log("info", map[string]interface{}{
		"msg":        "platform signing config updated",
		"profile_id": profileID,
		"platform":   platform,
	})

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "updated",
		"platform": platform,
		"config":   platformConfig,
		"message":  fmt.Sprintf("%s signing configuration saved successfully", platform),
	})
}

// DeleteSigning handles DELETE /api/v1/profiles/{id}/signing
// Removes all signing configuration for a profile.
func (h *Handler) DeleteSigning(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}

	if err := h.repo.Delete(r.Context(), profileID); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.log("error", map[string]interface{}{"msg": "failed to delete signing config", "error": err.Error()})
		h.writeError(w, http.StatusInternalServerError, "failed to delete signing config")
		return
	}

	h.log("info", map[string]interface{}{
		"msg":        "signing config deleted",
		"profile_id": profileID,
	})

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "deleted",
		"message": "Signing configuration removed",
	})
}

// DeletePlatformSigning handles DELETE /api/v1/profiles/{id}/signing/{platform}
// Removes signing configuration for a specific platform only.
func (h *Handler) DeletePlatformSigning(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	platform := vars["platform"]

	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}
	if !IsValidPlatform(platform) {
		h.writeError(w, http.StatusBadRequest, "invalid platform: "+platform)
		return
	}

	// Get SQL repository to use DeleteForPlatform
	sqlRepo, ok := h.repo.(*SQLRepository)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "platform deletion not supported")
		return
	}

	if err := sqlRepo.DeleteForPlatform(r.Context(), profileID, platform); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
			return
		}
		h.log("error", map[string]interface{}{"msg": "failed to delete platform signing config", "error": err.Error()})
		h.writeError(w, http.StatusInternalServerError, "failed to delete signing config")
		return
	}

	h.log("info", map[string]interface{}{
		"msg":        "platform signing config deleted",
		"profile_id": profileID,
		"platform":   platform,
	})

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "deleted",
		"platform": platform,
		"message":  fmt.Sprintf("%s signing configuration removed", platform),
	})
}

// ValidateSigning handles POST /api/v1/profiles/{id}/signing/validate
// Validates signing prerequisites (tools, certificates) for a profile.
func (h *Handler) ValidateSigning(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	if profileID == "" {
		h.writeError(w, http.StatusBadRequest, "profile_id is required")
		return
	}

	// Get the current config
	config, err := h.repo.Get(r.Context(), profileID)
	if errors.Is(err, ErrProfileNotFound) {
		h.writeError(w, http.StatusNotFound, "profile not found: "+profileID)
		return
	}
	if err != nil {
		h.log("error", map[string]interface{}{"msg": "failed to get signing config for validation", "error": err.Error()})
		h.writeError(w, http.StatusInternalServerError, "failed to retrieve signing config")
		return
	}

	if config == nil || !config.Enabled {
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   true,
			"message": "Signing is disabled - no validation needed",
		})
		return
	}

	// Run structural validation first
	structuralResult := h.validator.ValidateConfig(config)

	// If structural validation fails, return early
	if !structuralResult.Valid {
		h.writeJSON(w, http.StatusOK, structuralResult)
		return
	}

	// Run prerequisite checks
	ctx, cancel := context.WithTimeout(r.Context(), 30000000000) // 30 seconds
	defer cancel()

	prereqResult := h.checker.CheckPrerequisites(ctx, config)

	// Merge results
	finalResult := mergeValidationResults(structuralResult, prereqResult)

	h.writeJSON(w, http.StatusOK, finalResult)
}

// CheckPrerequisites handles GET /api/v1/signing/prerequisites
// Checks available signing tools on the current system (no profile needed).
func (h *Handler) CheckPrerequisites(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30000000000) // 30 seconds
	defer cancel()

	tools, err := h.checker.DetectTools(ctx)
	if err != nil {
		h.log("error", map[string]interface{}{"msg": "failed to detect signing tools", "error": err.Error()})
		h.writeError(w, http.StatusInternalServerError, "failed to detect signing tools")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"tools": tools,
	})
}

// CertificateDiscoverer is implemented by platform detectors that can discover certificates.
type CertificateDiscoverer interface {
	// DiscoverCertificates discovers available signing certificates/identities for the platform.
	DiscoverCertificates(ctx context.Context, platform string) ([]DiscoveredCertificate, error)
}

// DiscoverCertificates handles GET /api/v1/signing/discover/{platform}
// Discovers available certificates/identities for a specific platform.
func (h *Handler) DiscoverCertificates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	platform := vars["platform"]

	if !IsValidPlatform(platform) {
		h.writeError(w, http.StatusBadRequest, "invalid platform: "+platform+". Valid: windows, macos, linux")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30000000000) // 30 seconds
	defer cancel()

	// Try to get certificate discoverer
	discoverer, ok := h.checker.(CertificateDiscoverer)
	if !ok {
		// Fall back to returning tools info only
		tools, err := h.checker.DetectTools(ctx)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to detect signing tools")
			return
		}

		// Filter tools for the requested platform
		var platformTools []ToolDetectionResult
		for _, t := range tools {
			if t.Platform == platform {
				platformTools = append(platformTools, t)
			}
		}

		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"platform":     platform,
			"tools":        platformTools,
			"certificates": []DiscoveredCertificate{},
			"errors":       []string{"certificate discovery not available - checker does not implement CertificateDiscoverer"},
		})
		return
	}

	certs, err := discoverer.DiscoverCertificates(ctx, platform)
	var errors []string
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Also get tools info
	tools, _ := h.checker.DetectTools(ctx)
	var platformTools []ToolDetectionResult
	for _, t := range tools {
		if t.Platform == platform {
			platformTools = append(platformTools, t)
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"platform":     platform,
		"tools":        platformTools,
		"certificates": certs,
		"errors":       errors,
	})
}

// mergeValidationResults combines structural and prerequisite validation results.
func mergeValidationResults(structural, prereq *ValidationResult) *ValidationResult {
	result := NewValidationResult()
	result.Valid = structural.Valid && prereq.Valid

	// Merge errors
	result.Errors = append(result.Errors, structural.Errors...)
	result.Errors = append(result.Errors, prereq.Errors...)

	// Merge warnings
	result.Warnings = append(result.Warnings, structural.Warnings...)
	result.Warnings = append(result.Warnings, prereq.Warnings...)

	// Merge platforms - prereq takes precedence for tool info
	for platform, pv := range structural.Platforms {
		result.Platforms[platform] = pv
	}
	for platform, pv := range prereq.Platforms {
		existing, ok := result.Platforms[platform]
		if ok {
			// Merge: keep configured from structural, add tool info from prereq
			pv.Configured = existing.Configured
			pv.Errors = append(existing.Errors, pv.Errors...)
			pv.Warnings = append(existing.Warnings, pv.Warnings...)
		}
		result.Platforms[platform] = pv
	}

	return result
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError writes a JSON error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
