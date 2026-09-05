package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

func categorize(name string) string {
	root, err := manifestRoot()
	if err == nil {
		data, readErr := os.ReadFile(filepath.Join(root, "resources", name, "resource.json"))
		if readErr == nil {
			var manifest struct {
				Category string `json:"category"`
			}
			if json.Unmarshal(data, &manifest) == nil && strings.TrimSpace(manifest.Category) != "" {
				return strings.TrimSpace(manifest.Category)
			}
		}
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
		"loaded_at": operatorStateNow().UTC().Format(time.RFC3339),
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
