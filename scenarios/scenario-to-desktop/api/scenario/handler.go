package scenario

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"
	pathutil "scenario-to-desktop-api/shared/path"
)

// Handler provides HTTP handlers for scenario endpoints.
type Handler struct {
	vrooliRoot string
	records    RecordStore
	logger     *slog.Logger
}

// NewHandler creates a new scenario handler.
func NewHandler(vrooliRoot string, records RecordStore, logger *slog.Logger) *Handler {
	return &Handler{
		vrooliRoot: vrooliRoot,
		records:    records,
		logger:     logger,
	}
}

// RegisterRoutes registers scenario routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/scenarios/desktop-status", h.DesktopStatusHandler).Methods("GET", "OPTIONS")
}

// DesktopStatusHandler discovers all scenarios and their desktop deployment status.
func (h *Handler) DesktopStatusHandler(w http.ResponseWriter, r *http.Request) {
	vrooliRoot := h.vrooliRoot
	if vrooliRoot == "" {
		vrooliRoot = pathutil.DetectVrooliRoot()
	}

	scenariosPath := filepath.Join(vrooliRoot, "scenarios")
	entries, err := os.ReadDir(scenariosPath)
	if err != nil {
		h.logger.Error("failed to read scenarios directory",
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
		status := h.buildScenarioStatus(entry.Name(), scenariosPath, vrooliRoot)
		scenarios = append(scenarios, status)
	}

	httputil.WriteJSON(w, http.StatusOK, ListResponse{
		Scenarios: scenarios,
		Stats:     computeStats(scenarios),
	})
}

// buildScenarioStatus assembles the desktop status for a single scenario directory.
func (h *Handler) buildScenarioStatus(scenarioName, scenariosPath, vrooliRoot string) ScenarioDesktopStatus {
	scenarioRoot := filepath.Join(scenariosPath, scenarioName)
	electronPath := filepath.Join(scenarioRoot, "platforms", "electron")

	status := ScenarioDesktopStatus{Name: scenarioName}

	if info, err := loadScenarioServiceInfo(scenarioRoot); err == nil && info != nil {
		status.ServiceDisplay = strings.TrimSpace(info.DisplayName)
		status.ServiceDesc = strings.TrimSpace(info.Description)
	}
	status.ServiceIconPath = findScenarioIcon(scenarioRoot)

	if electronInfo, err := os.Stat(electronPath); err == nil && electronInfo.IsDir() {
		h.populateDesktopInfo(&status, electronPath, scenarioName, vrooliRoot)
	}

	if cfg, err := loadDesktopConnectionConfig(scenarioRoot); err == nil {
		status.ConnectionConfig = cfg
	} else if err != nil {
		h.logger.Warn("failed to read desktop config",
			"scenario", scenarioName,
			"error", err)
	}

	return status
}

// populateDesktopInfo fills desktop-specific fields when platforms/electron exists.
func (h *Handler) populateDesktopInfo(status *ScenarioDesktopStatus, electronPath, scenarioName, vrooliRoot string) {
	status.HasDesktop = true
	status.DesktopPath = electronPath

	readPackageJSON(electronPath, status)

	status.ArtifactsExpectedPath = filepath.Join(electronPath, "dist-electron")
	h.attachRecordInfo(status, scenarioName)
	resolveArtifacts(status, vrooliRoot)
}

// readPackageJSON reads display name and version from the electron package.json.
func readPackageJSON(electronPath string, status *ScenarioDesktopStatus) {
	pkgPath := filepath.Join(electronPath, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}
	var pkg map[string]interface{}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	if name, ok := pkg["name"].(string); ok {
		status.DisplayName = name
	}
	if version, ok := pkg["version"].(string); ok {
		status.Version = version
	}
}

// attachRecordInfo populates record fields from the most recent matching record.
func (h *Handler) attachRecordInfo(status *ScenarioDesktopStatus, scenarioName string) {
	records := h.listScenarioRecords(scenarioName)
	if len(records) == 0 {
		return
	}
	record := records[0]
	status.RecordID = record.ID
	status.RecordOutputPath = recordOutputPath(record)
	status.RecordLocationMode = record.LocationMode
	status.RecordUpdatedAt = recordTimestamp(record)
}

// resolveArtifacts finds built artifacts from either the standard or record dist path.
func resolveArtifacts(status *ScenarioDesktopStatus, vrooliRoot string) {
	if result, ok := scanDistArtifacts(status.ArtifactsExpectedPath, vrooliRoot); ok {
		applyArtifactScan(status, result, status.ArtifactsExpectedPath, "standard")
		return
	}
	if status.RecordOutputPath == "" {
		return
	}
	recordDistPath := filepath.Join(status.RecordOutputPath, "dist-electron")
	if recordDistPath == status.ArtifactsExpectedPath {
		return
	}
	if result, ok := scanDistArtifacts(recordDistPath, vrooliRoot); ok {
		applyArtifactScan(status, result, recordDistPath, "record")
	}
}

// applyArtifactScan copies scan results into the status struct.
func applyArtifactScan(status *ScenarioDesktopStatus, result *distArtifactScan, distPath, source string) {
	status.Built = true
	status.DistPath = distPath
	status.LastModified = result.lastModified
	status.PackageSize = result.totalSize
	status.BuildArtifacts = result.artifacts
	status.Platforms = uniqueStrings(result.platforms)
	status.ArtifactsSource = source
	status.ArtifactsPath = distPath
}

// computeStats counts aggregate scenario statistics.
func computeStats(scenarios []ScenarioDesktopStatus) *ScenarioStats {
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
	return &ScenarioStats{
		Total:       len(scenarios),
		WithDesktop: withDesktop,
		Built:       withBuilt,
		WebOnly:     len(scenarios) - withDesktop,
	}
}

func loadScenarioServiceInfo(scenarioRoot string) (*scenarioServiceInfo, error) {
	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Service scenarioServiceInfo `json:"service"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload.Service, nil
}

func findScenarioIcon(scenarioRoot string) string {
	candidates := []string{
		filepath.Join("ui", "dist", "manifest-icon-512.maskable.png"),
		filepath.Join("ui", "dist", "manifest-icon-192.maskable.png"),
		filepath.Join("ui", "dist", "apple-icon-180.png"),
		filepath.Join("ui", "dist", "favicon-196.png"),
		filepath.Join("ui", "public", "manifest-icon-512.maskable.png"),
		filepath.Join("ui", "public", "manifest-icon-192.maskable.png"),
		filepath.Join("ui", "public", "apple-icon-180.png"),
		filepath.Join("ui", "public", "favicon-196.png"),
		filepath.Join("ui", "public", "favicon-32x32.png"),
		filepath.Join("ui", "public", "favicon-16x16.png"),
		filepath.Join("ui", "public", "icon.png"),
		filepath.Join("ui", "assets", "icon.png"),
		filepath.Join("ui", "src", "assets", "icon.png"),
		filepath.Join("ui", "electron", "assets", "icon.png"),
		filepath.Join("ui", "electron", "assets", "tray-icon.png"),
	}

	for _, candidate := range candidates {
		path := filepath.Join(scenarioRoot, candidate)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func loadDesktopConnectionConfig(scenarioRoot string) (*DesktopConnectionConfig, error) {
	configPath := filepath.Join(scenarioRoot, ".vrooli", "desktop-config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg DesktopConnectionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// detectPlatformFromFilename tries to infer a platform from an artifact filename.
func detectPlatformFromFilename(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, ".msi") || strings.Contains(lower, "setup.exe") || strings.Contains(lower, ".exe") || strings.Contains(lower, "win"):
		return "win"
	case strings.Contains(lower, ".pkg") || strings.Contains(lower, ".dmg") || strings.Contains(lower, "mac") || strings.Contains(lower, "darwin"):
		return "mac"
	case strings.Contains(lower, ".appimage") || strings.Contains(lower, "linux") || strings.Contains(lower, ".deb") || strings.Contains(lower, ".tar"):
		return "linux"
	default:
		return ""
	}
}

func scanDistArtifacts(distPath, vrooliRoot string) (*distArtifactScan, bool) {
	distInfo, err := os.Stat(distPath)
	if err != nil || !distInfo.IsDir() {
		return nil, false
	}

	result := &distArtifactScan{
		lastModified: distInfo.ModTime().Format("2006-01-02 15:04:05"),
	}

	_ = filepath.Walk(distPath, func(currentPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		result.totalSize += info.Size()
		platform := detectPlatformFromFilename(info.Name())
		if platform != "" {
			result.platforms = append(result.platforms, platform)
		}

		relative := strings.TrimPrefix(currentPath, vrooliRoot)
		relative = strings.TrimPrefix(relative, string(os.PathSeparator))

		result.artifacts = append(result.artifacts, DesktopBuildArtifact{
			Platform:     platform,
			FileName:     info.Name(),
			SizeBytes:    info.Size(),
			ModifiedAt:   info.ModTime().Format("2006-01-02 15:04:05"),
			AbsolutePath: currentPath,
			RelativePath: relative,
		})
		return nil
	})

	return result, true
}

func (h *Handler) listScenarioRecords(scenarioName string) []*DesktopAppRecord {
	if h.records == nil {
		return nil
	}
	var matches []*DesktopAppRecord
	for _, rec := range h.records.List() {
		if rec != nil && rec.ScenarioName == scenarioName {
			matches = append(matches, rec)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return latestRecordTime(matches[i]).After(latestRecordTime(matches[j]))
	})
	return matches
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, str := range s {
		if str != "" && !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}
