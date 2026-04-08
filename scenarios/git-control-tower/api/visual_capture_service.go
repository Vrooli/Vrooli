package main

import (
	"context"
	"fmt"
	"log"
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

// resolvePages determines which pages to capture and how they were discovered.
func resolvePages(fs FileIO, repoDir, scenarioSlug string, explicitPages []string) ([]LighthousePage, string) {
	if len(explicitPages) > 0 {
		pages := make([]LighthousePage, len(explicitPages))
		for i, p := range explicitPages {
			pages[i] = LighthousePage{Path: p, Label: p}
		}
		return pages, "explicit"
	}
	return discoverPagesWithMethod(fs, repoDir, scenarioSlug)
}

// capturePageLabel returns a log-friendly identifier for a page+preset combination.
func capturePageLabel(slug string, page LighthousePage, preset CapturePreset) string {
	return fmt.Sprintf("%s%s@%dx%d_%s", slug, page.Path, preset.Width, preset.Height, preset.Theme)
}

// capturePageScreenshot executes the BAS workflow for a single page+preset and returns the PNG data.
// Returns nil data and an error string on failure (non-fatal).
func capturePageScreenshot(ctx context.Context, deps VisualCaptureDeps, slug string, page LighthousePage, preset CapturePreset) ([]byte, string) {
	label := capturePageLabel(slug, page, preset)
	pageCtx, pageCancel := context.WithTimeout(ctx, 60*time.Second)
	defer pageCancel()

	wfJSON, err := buildScreenshotWorkflow(slug, page, preset)
	if err != nil {
		log.Printf("WARNING: build workflow failed for %s: %v", label, err)
		return nil, err.Error()
	}

	execResp, err := deps.BAS.ExecuteAdhocWorkflow(pageCtx, BASExecuteAdhocRequest{
		FlowDefinition: wfJSON,
	}, false)
	if err != nil {
		log.Printf("WARNING: execute workflow failed for %s: %v", label, err)
		return nil, err.Error()
	}

	detail, err := deps.BAS.PollExecutionCompletion(pageCtx, execResp.ExecutionID, 500*time.Millisecond)
	if err != nil {
		log.Printf("WARNING: poll failed for %s: %v", label, err)
		return nil, err.Error()
	}
	if detail.Status != "EXECUTION_STATUS_COMPLETED" {
		errMsg := fmt.Sprintf("execution %s for %s: %s", detail.Status, label, detail.Error)
		log.Printf("WARNING: %s", errMsg)
		return nil, errMsg
	}

	ssResp, err := deps.BAS.GetScreenshots(pageCtx, execResp.ExecutionID)
	if err != nil {
		log.Printf("WARNING: get screenshots failed for %s: %v", label, err)
		return nil, err.Error()
	}
	if len(ssResp.Screenshots) == 0 {
		log.Printf("WARNING: no screenshots returned for %s", label)
		return nil, "no screenshots returned"
	}

	lastSS := ssResp.Screenshots[len(ssResp.Screenshots)-1]
	pngData, _, err := deps.BAS.GetScreenshotData(pageCtx, lastSS.Screenshot.Url)
	if err != nil {
		log.Printf("WARNING: fetch screenshot data failed for %s: %v", label, err)
		return nil, err.Error()
	}

	return pngData, ""
}

// CaptureScenario orchestrates a full capture for one scenario.
//
// In baseline mode, all existing snapshots are cleared and the new capture
// becomes the reference "Before" image. In capture mode (default), any
// existing capture-role snapshot is replaced while the baseline is preserved.
// captureAllPages runs the BAS workflow for every page+preset combination and
// returns the collected screenshots, the list of successfully captured page paths,
// and the last error message (if any).
func captureAllPages(ctx context.Context, deps VisualCaptureDeps, slug string, pages []LighthousePage, presets []CapturePreset) (map[string][]byte, []string, string) {
	screenshots := map[string][]byte{}
	capturedPagesSet := map[string]bool{}
	var captureErr string

	for _, preset := range presets {
		for _, page := range pages {
			pngData, errMsg := capturePageScreenshot(ctx, deps, slug, page, preset)
			if errMsg != "" {
				captureErr = errMsg
				continue
			}
			filename := sanitizeFilename(page.Path) + presetSuffix(preset) + ".png"
			screenshots[filename] = pngData
			capturedPagesSet[page.Path] = true
		}
	}

	var capturedPages []string
	for _, page := range pages {
		if capturedPagesSet[page.Path] {
			capturedPages = append(capturedPages, page.Path)
		}
	}
	return screenshots, capturedPages, captureErr
}

// captureStatus returns status and error string based on screenshot count.
func captureStatus(screenshotCount int, captureErr string) (string, string) {
	if screenshotCount == 0 {
		if captureErr == "" {
			captureErr = "no screenshots captured"
		}
		return "failed", captureErr
	}
	return "complete", captureErr
}

func CaptureScenario(ctx context.Context, deps VisualCaptureDeps, req VisualCaptureRequest) (*SnapshotSetMeta, error) {
	mode := req.Mode
	if mode == "" {
		mode = CaptureModeCapture
	}

	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "manual"
	}

	pages, discoveryMethod := resolvePages(deps.FS, deps.RepoDir, req.ScenarioSlug, req.Pages)

	presets := req.Presets
	if len(presets) == 0 {
		presets = []CapturePreset{{Name: "Desktop Light", Width: 1440, Height: 900, Theme: "light"}}
	}

	screenshots, capturedPages, captureErr := captureAllPages(ctx, deps, req.ScenarioSlug, pages, presets)
	status, captureErr := captureStatus(len(screenshots), captureErr)

	role := SnapshotRoleCapture
	if mode == CaptureModeBaseline {
		role = SnapshotRoleBaseline
	}

	meta := SnapshotSetMeta{
		ID:                  fmt.Sprintf("%d", time.Now().UnixNano()),
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
		Presets:             presets,
		PageDiscoveryMethod: discoveryMethod,
	}

	if err := prepareForCapture(deps.Storage, deps.RepoID, req.ScenarioSlug, mode); err != nil {
		log.Printf("WARNING: pre-capture cleanup failed for %s: %v", req.ScenarioSlug, err)
	}

	if err := deps.Storage.SaveSnapshotSet(deps.RepoID, meta, screenshots, nil); err != nil {
		return nil, fmt.Errorf("save snapshot set: %w", err)
	}

	return &meta, nil
}
