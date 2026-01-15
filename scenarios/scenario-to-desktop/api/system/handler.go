package system

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for system endpoints.
type Handler struct {
	wineService *DefaultWineService
	builds      BuildStore
	templateDir string
}

// NewHandler creates a new system handler.
func NewHandler(wineService *DefaultWineService, builds BuildStore, templateDir string) *Handler {
	return &Handler{
		wineService: wineService,
		builds:      builds,
		templateDir: templateDir,
	}
}

// RegisterRoutes registers system routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/status", h.StatusHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/templates", h.ListTemplatesHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/templates/{type}", h.GetTemplateHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/system/wine/check", h.CheckWineHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/system/wine/install", h.InstallWineHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/system/wine/install/status/{install_id}", h.GetWineInstallStatusHandler).Methods("GET", "OPTIONS")
}

// StatusHandler provides system status information.
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	building := 0
	completed := 0
	failed := 0
	var totalBuilds int

	if h.builds != nil {
		buildStatuses := h.builds.Snapshot()
		totalBuilds = len(buildStatuses)
		for _, status := range buildStatuses {
			switch status.Status {
			case "building":
				building++
			case "ready":
				completed++
			case "failed":
				failed++
			}
		}
	}

	response := StatusResponse{
		Service: map[string]interface{}{
			"name":        "scenario-to-desktop",
			"version":     "1.0.0",
			"description": "Transform Vrooli scenarios into professional desktop applications",
			"status":      "running",
		},
		Statistics: map[string]interface{}{
			"total_builds":     totalBuilds,
			"active_builds":    building,
			"completed_builds": completed,
			"failed_builds":    failed,
		},
		Capabilities: []string{
			"desktop_app_generation",
			"cross_platform_packaging",
			"template_system",
			"build_automation",
		},
		SupportedFrameworks: []string{"electron", "tauri", "neutralino"},
		SupportedTemplates:  []string{"universal", "advanced", "multi_window", "kiosk"},
		Endpoints: []map[string]string{
			{"method": "GET", "path": "/api/v1/health", "description": "Health check"},
			{"method": "GET", "path": "/api/v1/status", "description": "System status"},
			{"method": "GET", "path": "/api/v1/templates", "description": "List templates"},
			{"method": "POST", "path": "/api/v1/pipeline/run", "description": "Run pipeline (bundle, preflight, generate, build, smoketest, distribution)"},
			{"method": "GET", "path": "/api/v1/pipeline/{id}", "description": "Get pipeline status"},
			{"method": "POST", "path": "/api/v1/pipeline/{id}/resume", "description": "Resume stopped pipeline"},
			{"method": "POST", "path": "/api/v1/pipeline/{id}/cancel", "description": "Cancel running pipeline"},
			{"method": "GET", "path": "/api/v1/pipelines", "description": "List all pipelines"},
			{"method": "GET", "path": "/api/v1/desktop/status/{build_id}", "description": "Get build status"},
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// ListTemplatesHandler returns all available templates.
func (h *Handler) ListTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates := []TemplateInfo{
		{
			Name:        "Universal Desktop App",
			Description: "Universal wrapper that works for any scenario. Default choice - clean, professional desktop application with essential features.",
			Type:        "universal",
			Framework:   "electron",
			UseCases:    []string{"Any scenario needing desktop deployment", "Default choice", "Simple utilities", "Productivity apps", "Quick prototypes"},
			Features:    []string{"Native menus", "Auto-updater", "File operations", "Clean splash screen"},
			Complexity:  "low",
			Examples:    []string{"picker-wheel", "qr-code-generator", "palette-gen", "nutrition-tracker"},
		},
		{
			Name:        "Advanced Desktop App",
			Description: "Full-featured desktop application with advanced OS integration",
			Type:        "advanced",
			Framework:   "electron",
			UseCases:    []string{"Professional tools", "System administration", "Background services"},
			Features:    []string{"System tray", "Global shortcuts", "Rich notifications"},
			Complexity:  "medium",
			Examples:    []string{"system-monitor", "document-manager", "research-assistant"},
		},
		{
			Name:        "Multi-Window Desktop App",
			Description: "Advanced application supporting multiple windows and complex workflows",
			Type:        "multi_window",
			Framework:   "electron",
			UseCases:    []string{"IDE-like applications", "Dashboard applications", "Professional tools"},
			Features:    []string{"Window management", "Inter-window communication", "Floating panels"},
			Complexity:  "high",
			Examples:    []string{"agent-dashboard", "mind-maps", "brand-manager"},
		},
		{
			Name:        "Kiosk Mode App",
			Description: "Full-screen application for dedicated hardware and public displays",
			Type:        "kiosk",
			Framework:   "electron",
			UseCases:    []string{"Public displays", "Point-of-sale", "Industrial controls"},
			Features:    []string{"Full-screen lock", "Remote monitoring", "Auto-restart"},
			Complexity:  "high",
			Examples:    []string{"information-display", "booking-system", "retail-kiosk"},
		},
	}

	writeJSON(w, http.StatusOK, TemplatesResponse{
		Templates: templates,
		Count:     len(templates),
	})
}

// GetTemplateHandler retrieves a specific template configuration.
func (h *Handler) GetTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateType := vars["type"]

	templateFiles := map[string]string{
		"universal":    "universal-app.json",
		"basic":        "universal-app.json", // Alias for backward compatibility
		"advanced":     "advanced-app.json",
		"multi_window": "multi-window.json",
		"kiosk":        "kiosk-mode.json",
	}

	filename, valid := templateFiles[templateType]
	if !valid {
		http.Error(w, fmt.Sprintf("Invalid template type: %s", templateType), http.StatusBadRequest)
		return
	}

	templatePath := filepath.Join(h.templateDir, "advanced", filename)
	data, err := os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template not found: %s", templateType), http.StatusNotFound)
		return
	}

	var templateConfig map[string]interface{}
	if err := json.Unmarshal(data, &templateConfig); err != nil {
		http.Error(w, "Failed to parse template configuration", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, templateConfig)
}

// CheckWineHandler checks if Wine is installed and returns installation options.
func (h *Handler) CheckWineHandler(w http.ResponseWriter, r *http.Request) {
	response := h.wineService.CheckStatus()
	writeJSON(w, http.StatusOK, response)
}

// InstallWineHandler initiates Wine installation process.
func (h *Handler) InstallWineHandler(w http.ResponseWriter, r *http.Request) {
	var request WineInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validMethods := []string{"flatpak", "flatpak-auto", "appimage", "skip"}
	isValid := false
	for _, valid := range validMethods {
		if request.Method == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		http.Error(w, fmt.Sprintf("Invalid installation method. Supported: %s", strings.Join(validMethods, ", ")), http.StatusBadRequest)
		return
	}

	installID, err := h.wineService.StartInstallation(request.Method)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, WineInstallResponse{
		InstallID: installID,
		Status:    "pending",
		Method:    request.Method,
		StatusURL: fmt.Sprintf("/api/v1/system/wine/install/status/%s", installID),
	})
}

// GetWineInstallStatusHandler returns Wine installation status.
func (h *Handler) GetWineInstallStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installID := vars["install_id"]

	status, exists := h.wineService.GetInstallStatus(installID)
	if !exists {
		http.Error(w, "Installation ID not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
