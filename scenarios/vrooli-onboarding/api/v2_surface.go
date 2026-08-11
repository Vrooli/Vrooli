package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

type operatorSurfaceItem struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Schema    any    `json:"schema,omitempty"`
	Risk      string `json:"risk,omitempty"`
	Privilege string `json:"privilege,omitempty"`
	Default   any    `json:"default,omitempty"`
}

func loadSurfaceCatalog(root, kind, manifestName string) ([]operatorSurfaceItem, error) {
	entries, err := os.ReadDir(filepath.Join(root, "internal", kind))
	if os.IsNotExist(err) {
		return nil, &catalogUnavailableError{Missing: filepath.ToSlash(filepath.Join("catalog", "internal", kind)), Remediation: "rebuild the bundle with the declared host catalog"}
	}
	if err != nil {
		return nil, err
	}
	items := make([]operatorSurfaceItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(root, "internal", kind, entry.Name(), manifestName))
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		var manifest struct {
			Name      string `json:"name"`
			Risk      string `json:"risk"`
			Privilege string `json:"privilege"`
			Config    any    `json:"config_schema"`
			Default   any    `json:"default"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		name := manifest.Name
		if name == "" {
			name = entry.Name()
		}
		items = append(items, operatorSurfaceItem{Name: name, Type: kind, Schema: manifest.Config, Risk: manifest.Risk, Privilege: manifest.Privilege, Default: manifest.Default})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s *Server) handleV2Surface(w http.ResponseWriter, _ *http.Request) {
	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	tools, err := loadSurfaceCatalog(root, "tools", "tool.json")
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	safeguards, err := loadSurfaceCatalog(root, "safeguards", "safeguard.json")
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": append(tools, safeguards...), "tools": tools, "safeguards": safeguards})
}
