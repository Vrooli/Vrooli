package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// VisualCaptureDeps holds dependencies for visual capture orchestration.
type VisualCaptureDeps struct {
	BAS     *BrowserAutomationClient
	Storage *VisualCaptureStorage
	FS      FileIO
	RepoDir string
	RepoID  int64
}

// CaptureScenario orchestrates a full capture for one scenario.
//
// In baseline mode, all existing snapshots are cleared and the new capture
// becomes the reference "Before" image. In capture mode (default), any
// existing capture-role snapshot is replaced while the baseline is preserved.
func CaptureScenario(ctx context.Context, deps VisualCaptureDeps, req VisualCaptureRequest) (*SnapshotSetMeta, error) {
	mode := req.Mode
	if mode == "" {
		mode = CaptureModeCapture
	}

	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "manual"
	}

	// Resolve scenario UI URL
	uiURL, err := discovery.ResolveScenarioURL(ctx, req.ScenarioSlug, "UI_PORT")
	if err != nil {
		return nil, fmt.Errorf("resolve UI URL for %s: %w", req.ScenarioSlug, err)
	}

	// Discover pages
	var pages []LighthousePage
	var discoveryMethod string
	if len(req.Pages) > 0 {
		for _, p := range req.Pages {
			pages = append(pages, LighthousePage{Path: p, Label: p})
		}
		discoveryMethod = "explicit"
	} else {
		pages, discoveryMethod = discoverPagesWithMethod(deps.FS, deps.RepoDir, req.ScenarioSlug)
	}

	// Default viewport
	viewport := req.Viewport
	if viewport.Width == 0 || viewport.Height == 0 {
		viewport = BASViewport{Width: 1280, Height: 720}
	}

	snapshotID := fmt.Sprintf("%d", time.Now().UnixNano())
	screenshots := map[string][]byte{}
	var capturedPages []string
	var captureErr string

	for _, page := range pages {
		pageURL := strings.TrimRight(uiURL, "/") + page.Path
		resp, err := deps.BAS.CaptureScreenshot(ctx, pageURL, viewport)
		if err != nil {
			log.Printf("WARNING: screenshot failed for %s%s: %v", req.ScenarioSlug, page.Path, err)
			captureErr = err.Error()
			continue
		}

		pngData, err := decodeBase64DataURI(resp.Screenshot)
		if err != nil {
			log.Printf("WARNING: decode failed for %s%s: %v", req.ScenarioSlug, page.Path, err)
			continue
		}

		filename := sanitizeFilename(page.Path) + ".png"
		screenshots[filename] = pngData
		capturedPages = append(capturedPages, page.Path)
	}

	status := "complete"
	if len(screenshots) == 0 {
		status = "failed"
		if captureErr == "" {
			captureErr = "no screenshots captured"
		}
	}

	// Determine role from mode
	role := SnapshotRoleCapture
	if mode == CaptureModeBaseline {
		role = SnapshotRoleBaseline
	}

	meta := SnapshotSetMeta{
		ID:                  snapshotID,
		ScenarioSlug:        req.ScenarioSlug,
		Role:                role,
		TriggerType:         triggerType,
		Pages:               capturedPages,
		ScreenshotCount:     len(screenshots),
		VideoCount:          0,
		VideoStatus:         "not_implemented",
		CreatedAt:           time.Now().UTC(),
		Status:              status,
		Error:               captureErr,
		PageDiscoveryMethod: discoveryMethod,
	}

	// Pre-capture cleanup based on mode
	if err := prepareForCapture(deps.Storage, deps.RepoID, req.ScenarioSlug, mode); err != nil {
		log.Printf("WARNING: pre-capture cleanup failed for %s: %v", req.ScenarioSlug, err)
	}

	if err := deps.Storage.SaveSnapshotSet(deps.RepoID, meta, screenshots, nil); err != nil {
		return nil, fmt.Errorf("save snapshot set: %w", err)
	}

	return &meta, nil
}

// prepareForCapture handles snapshot cleanup before saving a new capture.
//
// Baseline mode: clears ALL existing snapshots (fresh start).
// Capture mode: replaces existing capture-role snapshots (preserves baseline).
func prepareForCapture(storage *VisualCaptureStorage, repoID int64, scenarioSlug, mode string) error {
	switch mode {
	case CaptureModeBaseline:
		return storage.ClearScenarioSnapshots(repoID, scenarioSlug)
	case CaptureModeCapture:
		return storage.DeleteSnapshotsByRole(repoID, scenarioSlug, SnapshotRoleCapture)
	default:
		return nil
	}
}

// CheckCaptureStaleness determines if the most recent capture for a scenario
// is outdated by comparing its creation time against file modification times
// in the scenario's source directory.
func CheckCaptureStaleness(repoDir, scenarioSlug string, captureTime time.Time) *SnapshotStalenessInfo {
	scenarioDir := filepath.Join(repoDir, "scenarios", scenarioSlug)

	// Directories to skip during the walk
	skipDirs := map[string]bool{
		"node_modules": true, ".git": true, "dist": true, "build": true,
		".next": true, "__pycache__": true, ".cache": true,
	}

	var latestMod time.Time
	_ = filepath.Walk(scenarioDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
		}
		return nil
	})

	if latestMod.IsZero() {
		return &SnapshotStalenessInfo{IsStale: false}
	}

	isStale := latestMod.After(captureTime)
	return &SnapshotStalenessInfo{
		IsStale:          isStale,
		LastFileChange:   &latestMod,
		CaptureCreatedAt: &captureTime,
	}
}

// discoverPages reads lighthouse.json from the scenario directory.
func discoverPages(fs FileIO, repoDir, scenarioSlug string) []LighthousePage {
	pages, _ := discoverPagesWithMethod(fs, repoDir, scenarioSlug)
	return pages
}

// discoverPagesWithMethod reads lighthouse.json and returns pages + discovery method.
func discoverPagesWithMethod(fs FileIO, repoDir, scenarioSlug string) ([]LighthousePage, string) {
	searchPaths := []string{
		filepath.Join(repoDir, "scenarios", scenarioSlug, ".vrooli", "lighthouse.json"),
		filepath.Join(repoDir, "apps", scenarioSlug, ".vrooli", "lighthouse.json"),
		filepath.Join(repoDir, "services", scenarioSlug, ".vrooli", "lighthouse.json"),
	}

	for _, path := range searchPaths {
		data, err := fs.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg LighthouseConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		if cfg.Enabled && len(cfg.Pages) > 0 {
			return cfg.Pages, "lighthouse"
		}
	}

	return []LighthousePage{{Path: "/", Label: "Home"}}, "fallback"
}

// sanitizeFilename converts page path "/" → "_root_", "/workflow/new" → "_workflow_new_"
func sanitizeFilename(pagePath string) string {
	if pagePath == "/" || pagePath == "" {
		return "_root_"
	}
	s := strings.TrimPrefix(pagePath, "/")
	s = strings.TrimSuffix(s, "/")
	s = strings.ReplaceAll(s, "/", "_")
	return "_" + s + "_"
}

// decodeBase64DataURI strips the data URI prefix and decodes base64.
func decodeBase64DataURI(dataURI string) ([]byte, error) {
	// Strip "data:image/png;base64," or similar prefix
	idx := strings.Index(dataURI, ",")
	if idx >= 0 {
		dataURI = dataURI[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(dataURI)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	return data, nil
}
