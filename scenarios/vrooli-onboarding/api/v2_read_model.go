package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type catalogUnavailableError struct {
	Missing     string
	Remediation string
}

func (e *catalogUnavailableError) Error() string {
	return fmt.Sprintf("catalog %q is unavailable: %s", e.Missing, e.Remediation)
}

func writeCatalogDegraded(w http.ResponseWriter, err error) bool {
	var unavailable *catalogUnavailableError
	if !errors.As(err, &unavailable) {
		return false
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "degraded",
		"error": map[string]string{
			"code": "catalog_unavailable", "missing_catalog": unavailable.Missing,
			"remediation": unavailable.Remediation,
		},
	})
	return true
}

// ScenarioReadModel is derived from scenario manifests plus operator state.
// It is intentionally a read model: onboarding never authors service.json.
type ScenarioReadModel struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	SystemRequired bool     `json:"system_required"`
	Enabled        bool     `json:"enabled"`
	AutoRestart    bool     `json:"auto_restart"`
	Resources      []string `json:"resources"`
}

// manifestRoot resolves the immutable manifest catalog. Normal development
// uses the repository; a desktop bundle uses its explicitly staged catalog.
func manifestRoot() (string, error) {
	roots, err := resolveRoots()
	if err != nil || roots.CatalogRoot == "" {
		return "", &catalogUnavailableError{Missing: "manifest root", Remediation: "set VROOLI_ROOT for a repository install or stage BUNDLE_ROOT/catalog for a bundle"}
	}
	catalog := roots.CatalogRoot
	if info, err := os.Stat(catalog); err != nil || !info.IsDir() {
		if err != nil {
			return "", &catalogUnavailableError{Missing: "catalog", Remediation: "rebuild the bundle with its declared catalog requirements"}
		}
		return "", &catalogUnavailableError{Missing: "catalog", Remediation: "rebuild the bundle with its declared catalog requirements"}
	}
	return catalog, nil
}

func loadScenarioReadModels() ([]ScenarioReadModel, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	state, err := loadOperatorState()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(root, "scenarios"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &catalogUnavailableError{Missing: "catalog/scenarios", Remediation: "rebuild the bundle with the scenario catalog, then restart onboarding"}
		}
		return nil, err
	}
	models := make([]ScenarioReadModel, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "scenarios", entry.Name(), ".vrooli", "service.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var manifest struct {
			Service struct {
				Name           string `json:"name"`
				Description    string `json:"description"`
				SystemRequired bool   `json:"system_required"`
			} `json:"service"`
			Runtime struct {
				AutoRestartDefault bool `json:"auto_restart_default"`
			} `json:"runtime"`
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		name := strings.TrimSpace(manifest.Service.Name)
		if name == "" {
			name = entry.Name()
		}
		choice := state.Scenarios[name]
		enabled := manifest.Service.SystemRequired
		if choice.Enabled != nil && !manifest.Service.SystemRequired {
			enabled = *choice.Enabled
		}
		autoRestart := manifest.Runtime.AutoRestartDefault
		if choice.AutoRestart != nil {
			autoRestart = *choice.AutoRestart
		}
		resources := make([]string, 0, len(manifest.Dependencies.Resources))
		for resource := range manifest.Dependencies.Resources {
			resources = append(resources, resource)
		}
		sort.Strings(resources)
		models = append(models, ScenarioReadModel{Name: name, Description: manifest.Service.Description, SystemRequired: manifest.Service.SystemRequired, Enabled: enabled, AutoRestart: autoRestart, Resources: resources})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Name < models[j].Name })
	return models, nil
}

func (s *Server) handleV2Scenarios(w http.ResponseWriter, _ *http.Request) {
	models, err := loadScenarioReadModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scenarios": models, "count": len(models)})
}
