package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Resource represents a Vrooli resource with derived category.
type Resource struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Category    string `json:"category"`
	Installed   string `json:"installed"`
	LastUpdated string `json:"last_updated"`
}

// rawResourceFile mirrors the on-disk running-resources.json format.
type rawResourceFile struct {
	Resources []struct {
		Name        string `json:"name"`
		Status      string `json:"status"`
		Installed   string `json:"installed"`
		LastUpdated string `json:"last_updated"`
	} `json:"resources"`
	LastUpdated string `json:"last_updated"`
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

// resolveResourcesPath returns the path to running-resources.json.
func resolveResourcesPath() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return filepath.Join(root, ".vrooli", "running-resources.json")
	}
	// Walk up from the api binary directory
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, ".vrooli", "running-resources.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// loadResources reads and parses running-resources.json.
func loadResources() ([]Resource, error) {
	path := resolveResourcesPath()
	if path == "" {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw rawResourceFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	resources := make([]Resource, 0, len(raw.Resources))
	for _, r := range raw.Resources {
		// Normalize status to running/installed/stopped
		status := strings.ToLower(r.Status)
		switch status {
		case "running", "installed", "stopped":
			// keep as-is
		default:
			status = "stopped"
		}

		resources = append(resources, Resource{
			Name:        r.Name,
			Status:      status,
			Category:    categorize(r.Name),
			Installed:   r.Installed,
			LastUpdated: r.LastUpdated,
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
