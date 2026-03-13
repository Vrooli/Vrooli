package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
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
		pageCtx, pageCancel := context.WithTimeout(ctx, 60*time.Second)

		wfJSON, err := buildScreenshotWorkflow(req.ScenarioSlug, page, viewport)
		if err != nil {
			pageCancel()
			log.Printf("WARNING: build workflow failed for %s%s: %v", req.ScenarioSlug, page.Path, err)
			captureErr = err.Error()
			continue
		}

		execResp, err := deps.BAS.ExecuteAdhocWorkflow(pageCtx, BASExecuteAdhocRequest{
			FlowDefinition: wfJSON,
		}, false)
		if err != nil {
			pageCancel()
			log.Printf("WARNING: execute workflow failed for %s%s: %v", req.ScenarioSlug, page.Path, err)
			captureErr = err.Error()
			continue
		}

		detail, err := deps.BAS.PollExecutionCompletion(pageCtx, execResp.ExecutionID, 500*time.Millisecond)
		if err != nil {
			pageCancel()
			log.Printf("WARNING: poll failed for %s%s: %v", req.ScenarioSlug, page.Path, err)
			captureErr = err.Error()
			continue
		}
		if detail.Status != "EXECUTION_STATUS_COMPLETED" {
			pageCancel()
			errMsg := fmt.Sprintf("execution %s for %s%s: %s", detail.Status, req.ScenarioSlug, page.Path, detail.Error)
			log.Printf("WARNING: %s", errMsg)
			captureErr = errMsg
			continue
		}

		ssResp, err := deps.BAS.GetScreenshots(pageCtx, execResp.ExecutionID)
		if err != nil {
			pageCancel()
			log.Printf("WARNING: get screenshots failed for %s%s: %v", req.ScenarioSlug, page.Path, err)
			captureErr = err.Error()
			continue
		}
		if len(ssResp.Screenshots) == 0 {
			pageCancel()
			log.Printf("WARNING: no screenshots returned for %s%s", req.ScenarioSlug, page.Path)
			captureErr = "no screenshots returned"
			continue
		}

		pngData, _, err := deps.BAS.GetScreenshotData(pageCtx, ssResp.Screenshots[0].Screenshot.Url)
		pageCancel()
		if err != nil {
			log.Printf("WARNING: fetch screenshot data failed for %s%s: %v", req.ScenarioSlug, page.Path, err)
			captureErr = err.Error()
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

// buildScreenshotWorkflow creates an inline BAS workflow JSON that navigates to
// a scenario page and takes a full-page screenshot.
func buildScreenshotWorkflow(scenarioSlug string, page LighthousePage, viewport BASViewport) (json.RawMessage, error) {
	nodes := []map[string]interface{}{
		{
			"id": "navigate",
			"action": map[string]interface{}{
				"type":     "ACTION_TYPE_NAVIGATE",
				"navigate": buildNavigateParams(scenarioSlug, page),
				"metadata": map[string]interface{}{
					"label": "Navigate to " + page.Path,
				},
			},
		},
	}

	// After navigation, poll until all loading spinners (.animate-spin) have
	// cleared. SPAs like React fire networkidle after the initial bundle but
	// before data-fetching API calls complete, so an explicit settle check is
	// needed to avoid capturing half-loaded pages.
	nodes = append(nodes, map[string]interface{}{
		"id": "wait-settled",
		"action": map[string]interface{}{
			"type": "ACTION_TYPE_EVALUATE",
			"evaluate": map[string]interface{}{
				"expression": settleExpression(page.WaitForSelector),
			},
			"metadata": map[string]interface{}{
				"label": "Wait for page to settle",
			},
		},
	})

	edges := []map[string]interface{}{
		{
			"id":     "e-nav-settle",
			"source": "navigate",
			"target": "wait-settled",
			"type":   "WORKFLOW_EDGE_TYPE_SMOOTHSTEP",
		},
	}

	nodes = append(nodes, map[string]interface{}{
		"id": "screenshot",
		"action": map[string]interface{}{
			"type": "ACTION_TYPE_SCREENSHOT",
			"screenshot": map[string]interface{}{
				"full_page": true,
			},
			"metadata": map[string]interface{}{
				"label": "Capture full-page screenshot",
			},
		},
	})
	edges = append(edges, map[string]interface{}{
		"id":     "e-screenshot",
		"source": "wait-settled",
		"target": "screenshot",
		"type":   "WORKFLOW_EDGE_TYPE_SMOOTHSTEP",
	})

	workflow := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":        fmt.Sprintf("screenshot-%s-%s", scenarioSlug, sanitizeFilename(page.Path)),
			"description": fmt.Sprintf("Capture screenshot of %s %s", scenarioSlug, page.Path),
		},
		"settings": map[string]interface{}{
			"viewport_width":  viewport.Width,
			"viewport_height": viewport.Height,
		},
		"nodes": nodes,
		"edges": edges,
	}

	return json.Marshal(workflow)
}

// settleExpression returns a JavaScript expression that polls until the page
// has no loading spinners (.animate-spin elements). If waitForSelector is
// provided, it additionally waits for that element to be visible.
func settleExpression(waitForSelector string) string {
	selectorCheck := ""
	if waitForSelector != "" {
		selectorCheck = fmt.Sprintf(
			` const el = document.querySelector(%q); if (!el || el.offsetParent === null) continue;`,
			waitForSelector,
		)
	}
	return fmt.Sprintf(
		`(async () => { const deadline = Date.now() + 15000; while (Date.now() < deadline) { const spinners = document.querySelectorAll(".animate-spin"); if (spinners.length === 0) {%s return "settled"; } await new Promise(r => setTimeout(r, 250)); } throw new Error("Page did not settle within 15s"); })()`,
		selectorCheck,
	)
}

// buildNavigateParams creates the navigate action parameters.
func buildNavigateParams(scenarioSlug string, page LighthousePage) map[string]interface{} {
	return map[string]interface{}{
		"destination_type": "NAVIGATE_DESTINATION_TYPE_SCENARIO",
		"scenario":         scenarioSlug,
		"scenario_path":    page.Path,
		"wait_until":       "NAVIGATE_WAIT_EVENT_NETWORKIDLE",
		"timeout_ms":       30000,
	}
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
