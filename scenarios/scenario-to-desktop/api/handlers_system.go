package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

// Health check handler
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "scenario-to-desktop-api",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Required field for API health check schema
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Status handler provides system information
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	// Count build statuses
	building := 0
	completed := 0
	failed := 0

	for _, status := range s.buildStatuses {
		switch status.Status {
		case "building":
			building++
		case "ready":
			completed++
		case "failed":
			failed++
		}
	}

	response := map[string]interface{}{
		"service": map[string]interface{}{
			"name":        "scenario-to-desktop",
			"version":     "1.0.0",
			"description": "Transform Vrooli scenarios into professional desktop applications",
			"status":      "running",
		},
		"statistics": map[string]interface{}{
			"total_builds":     len(s.buildStatuses),
			"active_builds":    building,
			"completed_builds": completed,
			"failed_builds":    failed,
		},
		"capabilities": []string{
			"desktop_app_generation",
			"cross_platform_packaging",
			"template_system",
			"build_automation",
		},
		"supported_frameworks": []string{"electron", "tauri", "neutralino"},
		"supported_templates":  []string{"universal", "advanced", "multi_window", "kiosk"}, // universal is the default (basic is alias)
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/api/v1/health", "description": "Health check"},
			{"method": "GET", "path": "/api/v1/status", "description": "System status"},
			{"method": "GET", "path": "/api/v1/templates", "description": "List templates"},
			{"method": "POST", "path": "/api/v1/desktop/generate", "description": "Generate desktop app"},
			{"method": "GET", "path": "/api/v1/desktop/status/{build_id}", "description": "Get build status"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// List templates handler
func (s *Server) listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// Get template handler
func (s *Server) getTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateType := vars["type"]

	// Map template types to actual filenames
	templateFiles := map[string]string{
		"universal":    "universal-app.json", // Default template for any scenario
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

	// Read template configuration file (now safe from path traversal)
	templatePath := filepath.Join(s.templateDir, "advanced", filename)

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templateConfig)
}
