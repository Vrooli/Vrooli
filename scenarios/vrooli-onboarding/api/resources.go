package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// cliClient is the shared typed Vrooli CLI client. It decodes the
// vrooli.cli.v1 contracts instead of hand-parsing CLI JSON, so a CLI output
// change is a compile error here rather than a silently empty or wrong result.
// Tests swap it via a Runner seam (see resources_test.go).
var cliClient = vroolicli.New()

// Resource represents a Vrooli resource with derived category, as surfaced to
// the onboarding UI. This is the onboarding API's own view-model — the UI
// depends on this shape, not on the vrooli CLI contract.
type Resource struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Category  string `json:"category"`
	Installed bool   `json:"installed"`
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

// normalizeResourceStatus derives the onboarding UI's coarse status label from
// the typed CLI resource status (running / installed / stopped).
func normalizeResourceStatus(item *cliv1.ResourceStatus) string {
	if item.GetRunning() {
		return "running"
	}

	statusText := strings.ToLower(strings.TrimSpace(item.GetHealth()))
	if statusText == "" {
		statusText = strings.ToLower(strings.TrimSpace(item.GetMessage()))
	}

	switch {
	case strings.Contains(statusText, "stopped"):
		return "stopped"
	case strings.Contains(statusText, "not installed"):
		return "stopped"
	case item.GetInstalled():
		return "installed"
	default:
		return "stopped"
	}
}

// loadResources reads resource status from the Vrooli CLI via the typed client.
// A CLI failure is propagated, never degraded to an empty list — the onboarding
// wizard must not silently present zero resources on a transient hiccup.
func loadResources() ([]Resource, error) {
	resp, err := cliClient.ResourceStatuses(context.Background())
	if err != nil {
		return nil, fmt.Errorf("vrooli resource status failed: %w", err)
	}

	resources := make([]Resource, 0, len(resp.GetResources()))
	for _, item := range resp.GetResources() {
		name := item.GetResource().GetName()
		if strings.TrimSpace(name) == "" {
			continue
		}
		resources = append(resources, Resource{
			Name:      name,
			Status:    normalizeResourceStatus(item),
			Category:  categorize(name),
			Installed: item.GetInstalled(),
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
