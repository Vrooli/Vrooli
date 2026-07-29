package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type hostRequirement struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
	Notes    string `json:"notes,omitempty"`
}
type hostItem struct {
	hostRequirement
	Description string   `json:"description,omitempty"`
	Risk        string   `json:"risk,omitempty"`
	Privilege   string   `json:"privilege,omitempty"`
	Bundling    string   `json:"bundling,omitempty"`
	Platforms   []string `json:"platforms,omitempty"`
	Commands    []string `json:"commands,omitempty"`
	Status      string   `json:"status"`
}
type hostRequirementsResponse struct {
	Tools      []hostItem `json:"tools"`
	Safeguards []hostItem `json:"safeguards"`
}

func loadHostRequirements(path string) ([]hostRequirement, []hostRequirement, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var manifest struct {
		HostTools      []hostRequirement `json:"hostTools"`
		HostSafeguards []hostRequirement `json:"hostSafeguards"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, err
	}
	return manifest.HostTools, manifest.HostSafeguards, nil
}

func mergeHostRequirements(target map[string]hostRequirement, source []hostRequirement) {
	for _, item := range source {
		if item.Name == "" {
			continue
		}
		old, exists := target[item.Name]
		if !exists || item.Required {
			target[item.Name] = item
		} else if old.Reason == "" {
			target[item.Name] = item
		}
	}
}
func hostManifest(root, kind, name string) ([]byte, error) {
	entries, err := os.ReadDir(filepath.Join(root, "internal", kind))
	if err != nil {
		return nil, err
	}
	fileName := strings.TrimSuffix(kind, "s") + ".json"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "internal", kind, entry.Name(), fileName)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var identity struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &identity) == nil && identity.Name == name {
			return data, nil
		}
	}
	return nil, fmt.Errorf("%s manifest for %q not found", strings.TrimSuffix(kind, "s"), name)
}

func hostItems(root, kind string, requirements map[string]hostRequirement, choices map[string]OptInChoice) ([]hostItem, error) {
	items := make([]hostItem, 0, len(requirements))
	for name, requirement := range requirements {
		data, err := hostManifest(root, kind, name)
		if err != nil {
			return nil, err
		}
		var meta struct {
			Description string   `json:"description"`
			Risk        string   `json:"risk"`
			Privilege   string   `json:"privilege"`
			Bundling    string   `json:"bundling"`
			Platforms   []string `json:"platforms"`
			Commands    []string `json:"commands"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, err
		}
		status := "optional"
		if requirement.Required {
			status = "required"
		} else if choice, ok := choices[name]; ok && choice.OptedIn != nil && *choice.OptedIn {
			status = "opted_in"
		}
		items = append(items, hostItem{hostRequirement: requirement, Description: meta.Description, Risk: meta.Risk, Privilege: meta.Privilege, Bundling: meta.Bundling, Platforms: meta.Platforms, Commands: meta.Commands, Status: status})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func deriveV2HostRequirements(root string, state OperatorState, models []ScenarioReadModel) (hostRequirementsResponse, error) {
	tools, safeguards := map[string]hostRequirement{}, map[string]hostRequirement{}
	resources := map[string]struct{}{}
	for _, model := range models {
		a, b, err := loadHostRequirements(filepath.Join(root, "scenarios", model.Name, ".vrooli", "service.json"))
		if err != nil {
			return hostRequirementsResponse{}, err
		}
		mergeHostRequirements(tools, a)
		mergeHostRequirements(safeguards, b)
		for _, resource := range model.Resources {
			resources[resource] = struct{}{}
		}
	}
	for resource := range resources {
		a, b, err := loadHostRequirements(filepath.Join(root, "resources", resource, "resource.json"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return hostRequirementsResponse{}, err
		}
		mergeHostRequirements(tools, a)
		mergeHostRequirements(safeguards, b)
	}
	toolItems, err := hostItems(root, "tools", tools, state.HostTools)
	if err != nil {
		return hostRequirementsResponse{}, err
	}
	safeguardItems, err := hostItems(root, "safeguards", safeguards, state.HostSafeguards)
	if err != nil {
		return hostRequirementsResponse{}, err
	}
	return hostRequirementsResponse{Tools: toolItems, Safeguards: safeguardItems}, nil
}

func (s *Server) handleV2HostRequirements(w http.ResponseWriter, _ *http.Request) {
	root, err := manifestRoot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	state, err := loadOperatorState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	models, err := selectedScenarioModels()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	response, err := deriveV2HostRequirements(root, state, models)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func hostPlatformSupported(platforms []string) bool {
	if len(platforms) == 0 {
		return true
	}
	for _, platform := range platforms {
		if platform == runtime.GOOS || (platform == "macos" && runtime.GOOS == "darwin") {
			return true
		}
	}
	return false
}
