package smoke

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/browsercapture"
	"test-genie/internal/evidence"
	"test-genie/internal/pagediscovery"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// PageCapture is the analyzed outcome of one all-pages visual capture.
type PageCapture struct {
	Page    string           `json:"page"`
	Label   string           `json:"label,omitempty"`
	Status  Status           `json:"status"`
	Message string           `json:"message"`
	Verdict evidence.Verdict `json:"-"`
}

// allPagesResult summarizes an all-pages capture pass.
type allPagesResult struct {
	Captures []PageCapture
	// Failed reports whether any page yielded a failing verdict.
	Failed bool
	// FailureMessage is the first failing page's message.
	FailureMessage string
}

// runAllPages enumerates the scenario's discovered pages and issues one
// CaptureService.Capture per page (screenshot + console + network, plus video
// under the full profile). Each capture is analyzed via the shared evidence
// analyzer and persisted under coverage/runs/<runID>/ui-smoke/pages/<page>/ so it
// is enumerable/serveable as a run artifact. This is the visual all-pages mode —
// it runs IN ADDITION to the single-page handshake smoke (which stays on the
// workflow engine); it never embeds the iframe host shell.
func (r *Runner) runAllPages(ctx context.Context, scenarioDir, scenarioName string) allPagesResult {
	pages, method := r.discoverer().Discover(scenarioDir, nil)
	r.log("All-pages visual capture: %d page(s) discovered via %s", len(pages), method)

	pagesDir := sharedartifacts.RunUISmokePagesDir(scenarioDir, r.runID)
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		r.log("Failed to create all-pages directory: %v", err)
		return allPagesResult{}
	}

	var result allPagesResult
	for _, page := range pages {
		capture, err := r.multiCapturer.CapturePage(ctx, browsercapture.PageRequest{
			ScenarioSlug: scenarioName,
			Path:         page.Path,
			Label:        page.Label,
			IncludeVideo: r.captureVideo,
		})
		if err != nil {
			r.log("Page capture failed for %s: %v", page.Path, err)
		}

		verdict := evidence.Analyze(capture.Evidence)
		pc := PageCapture{
			Page:    page.Path,
			Label:   page.Label,
			Status:  statusFromVerdict(verdict),
			Message: verdict.Message,
			Verdict: verdict,
		}
		result.Captures = append(result.Captures, pc)
		if !verdict.Passed() && !result.Failed {
			result.Failed = true
			result.FailureMessage = fmt.Sprintf("page %s: %s", page.Path, verdict.Message)
		}

		r.writePageArtifacts(pagesDir, page, capture)
	}
	return result
}

// writePageArtifacts persists one page's screenshot, optional video, and
// metadata under the page's directory.
func (r *Runner) writePageArtifacts(pagesDir string, page pagediscovery.Page, capture browsercapture.PageResult) {
	pageDir := filepath.Join(pagesDir, sanitizePagePath(page.Path))
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		r.log("Failed to create page directory for %s: %v", page.Path, err)
		return
	}

	meta, _ := json.MarshalIndent(map[string]string{"page": page.Path, "label": page.Label}, "", "  ")
	if err := os.WriteFile(filepath.Join(pageDir, "page.json"), meta, 0o644); err != nil {
		r.log("Failed to write page metadata for %s: %v", page.Path, err)
	}

	// On the Tier-1 single host, BAS artifact paths are directly readable; copy
	// the screenshot/video into the run tree so they are addressable by run-id.
	// (Multi-host artifact transport is an explicit non-goal — plan §5.)
	r.copyArtifact(page.Path, capture.ScreenshotPath, filepath.Join(pageDir, "screenshot.png"))
	r.copyArtifact(page.Path, capture.VideoPath, filepath.Join(pageDir, "video.webm"))
}

// copyArtifact copies a BAS server-side artifact into the run tree, logging (not
// failing) on a read/write miss so one unreadable artifact does not abort the
// pass.
func (r *Runner) copyArtifact(pagePath, srcPath, destPath string) {
	if srcPath == "" {
		return
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		r.log("Failed to read artifact %s for %s: %v", srcPath, pagePath, err)
		return
	}
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		r.log("Failed to copy artifact for %s: %v", pagePath, err)
	}
}

// statusFromVerdict maps an evidence verdict to a smoke status.
func statusFromVerdict(v evidence.Verdict) Status {
	if v.Passed() {
		return StatusPassed
	}
	return StatusFailed
}

// sanitizePagePath converts a page route into a filesystem-safe directory name.
// "/" → "_root_"; "/workflow/new" → "workflow_new".
func sanitizePagePath(pagePath string) string {
	if pagePath == "/" || strings.TrimSpace(pagePath) == "" {
		return "_root_"
	}
	s := strings.Trim(pagePath, "/")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}
