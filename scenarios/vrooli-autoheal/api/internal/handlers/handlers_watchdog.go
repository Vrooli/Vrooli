package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/watchdog"
)

func (h *Handlers) Watchdog(w http.ResponseWriter, r *http.Request) {
	// Check if refresh is requested
	refresh := r.URL.Query().Get("refresh") == "true"

	var status *watchdog.Status
	if refresh {
		status = h.watchdogDetector.Detect()
	} else {
		status = h.watchdogDetector.GetCached()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		apierrors.LogError("watchdog", "encode_response", err)
	}
}

// WatchdogTemplate returns the service configuration template for the current platform
// [REQ:WATCH-LINUX-001] [REQ:WATCH-MAC-001] [REQ:WATCH-WIN-001]
func (h *Handlers) WatchdogTemplate(w http.ResponseWriter, r *http.Request) {
	template, err := h.watchdogDetector.GetServiceTemplate()
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("watchdog", "service template", string(h.platform.Platform)))
		return
	}

	// Build API base URL from request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	apiBaseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	platformStr := string(h.platform.Platform)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"platform":     h.platform.Platform,
		"template":     template,
		"instructions": getInstallInstructions(platformStr),
		"oneLiner":     getOneLinerInstall(platformStr, apiBaseURL),
	}); err != nil {
		apierrors.LogError("watchdog_template", "encode_response", err)
	}
}

// getInstallInstructions returns platform-specific installation instructions
func getInstallInstructions(platformStr string) string {
	switch platformStr {
	case "linux":
		return `1. Keep the template for audit/reference.
2. Run the control-plane setup or the watchdog install endpoint so the native service backend provisions and starts it.`
	case "macos":
		return `1. Keep the template for audit/reference.
2. Run the control-plane setup or the watchdog install endpoint so the native service backend provisions and starts it.`
	case "windows":
		return `1. Keep the template for audit/reference.
2. Run the control-plane setup or the watchdog install endpoint so the native service backend provisions and starts it.`
	default:
		return "Watchdog installation not supported on this platform"
	}
}

// getOneLinerInstall returns a one-liner command to install the watchdog service
func getOneLinerInstall(platformStr, apiBaseURL string) string {
	if platformStr != "linux" && platformStr != "macos" && platformStr != "windows" {
		return ""
	}
	return fmt.Sprintf(`curl -fsS -X POST %s/api/v1/watchdog/install -H 'Content-Type: application/json' -d '{}'`, apiBaseURL)
}

// watchdogMutationResult is the response shape of the watchdog mutation
// endpoints. They exist for older clients; the only mutation any of them
// performs is delegating to the control-plane safeguard.
type watchdogMutationResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// WatchdogInstall delegates to the control-plane safeguard that owns the unit.
// [REQ:WATCH-INSTALL-001]
func (h *Handlers) WatchdogInstall(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil && r.ContentLength > 0 {
		var opts map[string]any
		if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("watchdog", "invalid request body", err))
			return
		}
		// Options are ignored: policy is resolved by the setup-owned safeguard.
	}
	output, runErr := h.controlPlane.OutputCombined(r.Context(), "host", "safeguard", "autoheal_watchdog")
	result := watchdogMutationResult{Success: runErr == nil, Message: "autoheal watchdog installation delegated to vrooli setup"}
	if runErr != nil {
		result.Error = fmt.Sprintf("vrooli host safeguard autoheal_watchdog: %v: %s", runErr, strings.TrimSpace(string(output)))
	}
	h.watchdogDetector.Invalidate()
	w.Header().Set("Content-Type", "application/json")
	if !result.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		apierrors.LogError("watchdog_install", "encode_response", err)
	}
}

// WatchdogUninstall is retained for older clients and refuses: setup owns the
// unit's lifecycle and must preserve its declared protection contract.
func (h *Handlers) WatchdogUninstall(w http.ResponseWriter, _ *http.Request) {
	writeWatchdogRefusal(w, "watchdog_uninstall", "autoheal watchdog lifecycle is owned by project setup", "scenario-owned watchdog uninstall is unsupported; use the project setup policy")
}

// WatchdogEnableLinger is retained for older clients and refuses: lingering
// follows the operator's boot policy, which `vrooli setup` applies.
func (h *Handlers) WatchdogEnableLinger(w http.ResponseWriter, _ *http.Request) {
	writeWatchdogRefusal(w, "watchdog_linger", "systemd lingering is owned by project setup", "run `vrooli setup` to apply the selected boot policy")
}

func writeWatchdogRefusal(w http.ResponseWriter, op, message, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	if err := json.NewEncoder(w).Encode(watchdogMutationResult{Success: false, Message: message, Error: detail}); err != nil {
		apierrors.LogError(op, "encode_response", err)
	}
}

// WatchdogStatus returns detailed installation status
func (h *Handlers) WatchdogStatus(w http.ResponseWriter, r *http.Request) {
	status := h.watchdogDetector.GetInstallStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		apierrors.LogError("watchdog_status", "encode_response", err)
	}
}

// GetCheckActions returns available recovery actions for a check
// [REQ:HEAL-ACTION-001]
