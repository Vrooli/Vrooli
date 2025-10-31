package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/handlers"
)

type Component struct {
	Type        string `json:"type"`         // "scenario" | "resource"
	ID          string `json:"id"`           // e.g., "notes"
	Name        string `json:"name"`         // e.g., "notes"
	DisplayName string `json:"display_name"` // e.g., "Notes"
	Status      string `json:"status"`       // "running" | "stopped" | "unknown"
	Description string `json:"description,omitempty"`
}

// getComponentsHandler returns all available scenarios and resources
func (s *Server) getComponentsHandler(w http.ResponseWriter, r *http.Request) {
	components, err := s.discoverComponents()
	if err != nil {
		logging.LogErrorErr("Failed to discover components", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load components")
		return
	}

	// Separate by type for easier UI consumption
	scenarios := make([]Component, 0)
	resources := make([]Component, 0)

	for _, comp := range components {
		if comp.Type == "scenario" {
			scenarios = append(scenarios, comp)
		} else if comp.Type == "resource" {
			resources = append(resources, comp)
		}
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"components": map[string]interface{}{
				"scenarios": scenarios,
				"resources": resources,
			},
			"total": len(components),
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write components response", err)
	}
}

// discoverComponents scans filesystem for scenarios and resources
func (s *Server) discoverComponents() ([]Component, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory and VROOLI_ROOT not set: %w", err)
		}
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}

	components := make([]Component, 0)

	// Discover scenarios
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")
	scenarios, err := s.discoverScenarios(scenariosDir)
	if err != nil {
		logging.LogWarn("Failed to discover scenarios", "error", err, "path", scenariosDir)
	} else {
		components = append(components, scenarios...)
	}

	// Discover resources
	resources, err := s.discoverResources()
	if err != nil {
		logging.LogWarn("Failed to discover resources", "error", err)
	} else {
		components = append(components, resources...)
	}

	return components, nil
}

func (s *Server) discoverScenarios(scenariosDir string) ([]Component, error) {
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	scenarios := make([]Component, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioID := entry.Name()

		// Skip hidden directories and special directories
		if strings.HasPrefix(scenarioID, ".") || scenarioID == "node_modules" {
			continue
		}

		displayName := toDisplayName(scenarioID)

		scenarios = append(scenarios, Component{
			Type:        "scenario",
			ID:          scenarioID,
			Name:        scenarioID,
			DisplayName: displayName,
			Status:      "unknown", // Could enhance with actual status check
		})
	}

	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].ID < scenarios[j].ID
	})

	return scenarios, nil
}

func (s *Server) discoverResources() ([]Component, error) {
	// Call vrooli CLI to get resource list
	// For now, return hardcoded common resources
	// TODO: Parse `vrooli resource list` output

	commonResources := []string{
		"postgres", "redis", "qdrant", "n8n", "ollama",
		"browserless", "judge0", "vault", "claude-code",
		"airbyte", "apache-airflow", "apache-superset", "audiocraft",
		"autogen-studio", "autogpt", "blender", "btcpay",
		"cline", "cloudflare-ai-gateway", "cncjs", "codex", "comfyui",
		"crewai", "dify", "docker-mailserver", "dozzle", "duckdns",
		"duplicati", "excalidraw", "ファイルブラウザ", "filebrowser", "flowise",
		"gitea", "grafana", "grocy", "home-assistant", "homepage",
		"immich", "influxdb", "jellyfin", "kafka", "langflow",
		"litellm", "lobe-chat", "localai", "mealie", "minio",
		"mattermost", "mongodb", "mysql", "nginx", "nextcloud",
		"open-webui", "penpot", "photoprism", "pihole", "portainer",
		"prometheus", "rabbitmq", "semaphore", "strapi", "supabase",
		"syncthing", "tabby", "taskweaver", "traggo", "uptime-kuma",
		"vaultwarden", "vikunja", "watchtower", "wikijs", "windmill",
	}

	resources := make([]Component, 0, len(commonResources))

	for _, resID := range commonResources {
		resources = append(resources, Component{
			Type:        "resource",
			ID:          resID,
			Name:        resID,
			DisplayName: toDisplayName(resID),
			Status:      "unknown",
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ID < resources[j].ID
	})

	return resources, nil
}

func toDisplayName(id string) string {
	// Handle special cases
	if id == "n8n" {
		return "n8n"
	}
	if id == "ファイルブラウザ" {
		return "File Browser (ファイルブラウザ)"
	}

	parts := strings.Split(id, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}
