package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Get scenario desktop status handler - discovers all scenarios and their desktop deployment status
func (s *Server) getScenarioDesktopStatusHandler(w http.ResponseWriter, r *http.Request) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		// Fallback to calculating from current directory
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}

	scenariosPath := filepath.Join(vrooliRoot, "scenarios")
	entries, err := os.ReadDir(scenariosPath)
	if err != nil {
		s.logger.Error("failed to read scenarios directory",
			"path", scenariosPath,
			"error", err)
		http.Error(w, "Failed to read scenarios directory", http.StatusInternalServerError)
		return
	}

	type ScenarioDesktopStatus struct {
		Name             string                   `json:"name"`
		DisplayName      string                   `json:"display_name,omitempty"`
		HasDesktop       bool                     `json:"has_desktop"`
		DesktopPath      string                   `json:"desktop_path,omitempty"`
		Version          string                   `json:"version,omitempty"`
		Platforms        []string                 `json:"platforms,omitempty"`
		Built            bool                     `json:"built,omitempty"`
		DistPath         string                   `json:"dist_path,omitempty"`
		LastModified     string                   `json:"last_modified,omitempty"`
		PackageSize      int64                    `json:"package_size,omitempty"`
		ConnectionConfig *DesktopConnectionConfig `json:"connection_config,omitempty"`
	}

	var scenarios []ScenarioDesktopStatus

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		scenarioRoot := filepath.Join(scenariosPath, scenarioName)
		electronPath := filepath.Join(scenarioRoot, "platforms", "electron")

		status := ScenarioDesktopStatus{
			Name: scenarioName,
		}

		// Check if platforms/electron exists
		if electronInfo, err := os.Stat(electronPath); err == nil && electronInfo.IsDir() {
			status.HasDesktop = true
			status.DesktopPath = electronPath

			// Read package.json for details
			pkgPath := filepath.Join(electronPath, "package.json")
			if data, err := os.ReadFile(pkgPath); err == nil {
				var pkg map[string]interface{}
				if json.Unmarshal(data, &pkg) == nil {
					if name, ok := pkg["name"].(string); ok {
						status.DisplayName = name
					}
					if version, ok := pkg["version"].(string); ok {
						status.Version = version
					}
				}
			}

			// Check if dist-electron exists (built packages)
			distPath := filepath.Join(electronPath, "dist-electron")
			if distInfo, err := os.Stat(distPath); err == nil && distInfo.IsDir() {
				status.Built = true
				status.DistPath = distPath
				status.LastModified = distInfo.ModTime().Format("2006-01-02 15:04:05")

				// Calculate total size of dist directory
				var totalSize int64
				filepath.Walk(distPath, func(_ string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					if !info.IsDir() {
						totalSize += info.Size()
					}
					return nil
				})
				status.PackageSize = totalSize

				// Try to detect which platforms were built by looking at dist files
				distEntries, _ := os.ReadDir(distPath)
				for _, de := range distEntries {
					name := de.Name()
					if strings.Contains(name, ".exe") || strings.Contains(name, "win") {
						status.Platforms = append(status.Platforms, "win")
					} else if strings.Contains(name, ".dmg") || strings.Contains(name, "mac") {
						status.Platforms = append(status.Platforms, "mac")
					} else if strings.Contains(name, ".AppImage") || strings.Contains(name, "linux") {
						status.Platforms = append(status.Platforms, "linux")
					}
				}
				// Remove duplicates
				status.Platforms = uniqueStrings(status.Platforms)
			}
		}

		if cfg, err := loadDesktopConnectionConfig(scenarioRoot); err == nil {
			status.ConnectionConfig = cfg
		} else if err != nil {
			s.logger.Warn("failed to read desktop config",
				"scenario", scenarioName,
				"error", err)
		}

		scenarios = append(scenarios, status)
	}

	// Count statistics
	withDesktop := 0
	withBuilt := 0
	for _, s := range scenarios {
		if s.HasDesktop {
			withDesktop++
		}
		if s.Built {
			withBuilt++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"stats": map[string]int{
			"total":        len(scenarios),
			"with_desktop": withDesktop,
			"built":        withBuilt,
			"web_only":     len(scenarios) - withDesktop,
		},
	})
}
