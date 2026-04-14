package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Resource represents a Vrooli resource with derived category.
type Resource struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Category  string `json:"category"`
	Installed bool   `json:"installed"`
}

type resourceStatusList struct {
	Resources []resourceStatusItem `json:"resources"`
}

type resourceStatusItem struct {
	Resource struct {
		Name string `json:"name"`
	} `json:"resource"`
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Health    string `json:"health"`
	Message   string `json:"message"`
}

var runResourceStatusJSON = func(ctx context.Context) ([]byte, error) {
	return exec.CommandContext(ctx, "vrooli", "resource", "status", "--json").Output()
}

// categoryMap maps resource names to human-friendly categories.
var categoryMap = map[string]string{
	// Databases & storage
	"postgres": "database", "postgis": "database", "redis": "database",
	"qdrant": "database", "neo4j": "database", "sqlite": "database",
	"minio": "storage",
	// AI / ML
	"ollama": "ai", "claude-code": "ai", "autogpt": "ai", "autogen-studio": "ai",
	"crewai": "ai", "langchain": "ai", "llamaindex": "ai", "haystack": "ai",
	"openrouter": "ai", "gemini": "ai", "cline": "ai", "codex": "ai",
	"opencode": "ai", "whisper": "ai", "kokoro": "ai",
	"segment-anything": "ai", "ultralytics-yolo": "ai", "nsfw-detector": "ai",
	"unstructured-io": "ai", "agent-s2": "ai",
	// Browser / automation
	"browserless": "browser",
	// IoT / hardware
	"zigbee2mqtt": "iot", "home-assistant": "iot", "esphome": "iot",
	"eclipse-ditto": "iot", "traccar": "iot",
	// Engineering / simulation
	"freecad": "engineering", "blender": "engineering", "kicad": "engineering",
	"gazebo": "engineering", "elmer-fem": "engineering", "su2": "engineering",
	"godot": "engineering", "sagemath": "engineering", "simpy": "engineering",
	// DevOps / infrastructure
	"earthly": "devops", "k6": "devops", "judge0": "devops",
	"n8n": "devops", "kafka": "devops",
	// Media
	"ffmpeg": "media",
	// Security
	"vault": "security", "step-ca": "security", "keycloak": "security",
	"virustotal": "security",
	// Communication
	"pushover": "communication", "twilio": "communication",
	"mail-in-a-box": "communication",
	// Data / analytics
	"airbyte": "data", "apache-superset": "data", "geonode": "data",
	// Business / enterprise
	"erpnext": "business", "btcpay": "business", "geth": "business",
	// Content / collaboration
	"nextcloud": "collaboration", "wikijs": "collaboration",
	// Search
	"searxng": "search",
	// Agriculture
	"farmos": "agriculture",
	// Networking
	"cloudflare-ai-gateway": "networking", "mcrcon": "networking",
}

func categorize(name string) string {
	if cat, ok := categoryMap[name]; ok {
		return cat
	}
	return "general"
}

func normalizeResourceStatus(item resourceStatusItem) string {
	if item.Running {
		return "running"
	}

	statusText := strings.ToLower(strings.TrimSpace(item.Health))
	if statusText == "" {
		statusText = strings.ToLower(strings.TrimSpace(item.Message))
	}

	switch {
	case strings.Contains(statusText, "stopped"):
		return "stopped"
	case strings.Contains(statusText, "not installed"):
		return "stopped"
	case item.Installed:
		return "installed"
	default:
		return "stopped"
	}
}

// loadResources reads resource status from the Vrooli CLI.
func loadResources() ([]Resource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := runResourceStatusJSON(ctx)
	if err != nil {
		return nil, fmt.Errorf("vrooli resource status --json failed: %w", err)
	}

	var raw resourceStatusList
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse resource status output: %w", err)
	}

	resources := make([]Resource, 0, len(raw.Resources))
	for _, item := range raw.Resources {
		if strings.TrimSpace(item.Resource.Name) == "" {
			continue
		}
		resources = append(resources, Resource{
			Name:      item.Resource.Name,
			Status:    normalizeResourceStatus(item),
			Category:  categorize(item.Resource.Name),
			Installed: item.Installed,
		})
	}
	return resources, nil
}

func (s *Server) handleListResources(w http.ResponseWriter, _ *http.Request) {
	resources, err := loadResources()
	if err != nil {
		writeResourceLoadError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resources": resources,
		"count":     len(resources),
		"loaded_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleGetResource(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	resources, err := loadResources()
	if err != nil {
		writeResourceLoadError(w, err)
		return
	}

	for _, res := range resources {
		if strings.EqualFold(res.Name, name) {
			writeJSON(w, http.StatusOK, res)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": "resource not found: " + name,
	})
}
