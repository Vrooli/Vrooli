package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DesktopBuildArtifact describes an installer produced for a specific platform
type DesktopBuildArtifact struct {
	Platform     string `json:"platform"`
	FileName     string `json:"file_name"`
	SizeBytes    int64  `json:"size_bytes"`
	ModifiedAt   string `json:"modified_at"`
	AbsolutePath string `json:"absolute_path"`
	RelativePath string `json:"relative_path"`
}

// ScenarioDesktopStatus represents the aggregated desktop build status for a scenario
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
	BuildArtifacts   []DesktopBuildArtifact   `json:"build_artifacts,omitempty"`
}

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

				// Calculate total size of dist directory and capture artifacts per file
				var totalSize int64
				filepath.Walk(distPath, func(currentPath string, info os.FileInfo, walkErr error) error {
					if walkErr != nil {
						return nil
					}
					if info.IsDir() {
						return nil
					}

					totalSize += info.Size()
					platform := detectPlatformFromFilename(info.Name())
					if platform != "" {
						status.Platforms = append(status.Platforms, platform)
					}

					relative := strings.TrimPrefix(currentPath, vrooliRoot)
					relative = strings.TrimPrefix(relative, string(os.PathSeparator))

					status.BuildArtifacts = append(status.BuildArtifacts, DesktopBuildArtifact{
						Platform:     platform,
						FileName:     info.Name(),
						SizeBytes:    info.Size(),
						ModifiedAt:   info.ModTime().Format("2006-01-02 15:04:05"),
						AbsolutePath: currentPath,
						RelativePath: relative,
					})
					return nil
				})
				status.PackageSize = totalSize
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

// detectPlatformFromFilename tries to infer a platform from an artifact filename
func detectPlatformFromFilename(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, ".exe") || strings.Contains(lower, "win"):
		return "win"
	case strings.Contains(lower, ".dmg") || strings.Contains(lower, "mac") || strings.Contains(lower, "darwin"):
		return "mac"
	case strings.Contains(lower, ".appimage") || strings.Contains(lower, "linux") || strings.Contains(lower, ".deb") || strings.Contains(lower, ".tar"):
		return "linux"
	default:
		return ""
	}
}
