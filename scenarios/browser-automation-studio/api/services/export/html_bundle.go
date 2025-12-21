package export

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/storage"
)

const (
	defaultHTMLViewportWidth  = 1920
	defaultHTMLViewportHeight = 1080
	defaultHTMLFrameDuration  = 1600
	minHTMLFrameDuration      = 800
	defaultHTMLHoldDuration   = 650
)

type htmlPayload struct {
	GeneratedAt string               `json:"generatedAt"`
	Execution   htmlPayloadExecution `json:"execution"`
	Theme       ExportTheme          `json:"theme"`
	Frames      []htmlPayloadFrame   `json:"frames"`
}

type htmlPayloadExecution struct {
	ExecutionID  string `json:"executionId"`
	WorkflowID   string `json:"workflowId"`
	WorkflowName string `json:"workflowName"`
	Status       string `json:"status"`
	FrameCount   int    `json:"frameCount"`
}

type htmlPayloadFrame struct {
	Index            int                 `json:"index"`
	Title            string              `json:"title"`
	Subtitle         string              `json:"subtitle"`
	Status           string              `json:"status"`
	PlayDuration     int                 `json:"playDuration"`
	Screenshot       string              `json:"screenshot,omitempty"`
	ZoomFactor       float64             `json:"zoomFactor"`
	HighlightRegions []htmlHighlightRect `json:"highlightRegions,omitempty"`
	Cursor           *htmlCursor         `json:"cursor,omitempty"`
	CursorTrail      []htmlPoint         `json:"cursorTrail,omitempty"`
	AddressBar       string              `json:"addressBar"`
	FinalURL         string              `json:"finalUrl,omitempty"`
	AssertionMessage string              `json:"assertionMessage,omitempty"`
	RetryLabel       string              `json:"retryLabel,omitempty"`
	DurationLabel    string              `json:"durationLabel,omitempty"`
	ConsoleCount     int                 `json:"consoleCount,omitempty"`
	NetworkCount     int                 `json:"networkCount,omitempty"`
}

type htmlHighlightRect struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Color  string  `json:"color,omitempty"`
}

type htmlPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type htmlCursor struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Pulse bool    `json:"pulse"`
}

// WriteHTMLBundle builds a zip archive containing a standalone HTML replay package.
func WriteHTMLBundle(
	ctx context.Context,
	writer io.Writer,
	spec *ReplayMovieSpec,
	storageClient storage.StorageInterface,
	log *logrus.Logger,
	baseURL string,
) error {
	if spec == nil {
		return fmt.Errorf("html export requires replay spec")
	}
	zipWriter := zip.NewWriter(writer)
	assetPaths := make(map[string]string, len(spec.Assets))

	for index, asset := range spec.Assets {
		localPath, err := writeReplayAsset(ctx, zipWriter, asset, index, storageClient, log, baseURL)
		if err != nil {
			if log != nil {
				log.WithError(err).WithField("asset_id", asset.ID).Warn("Failed to include replay asset in HTML bundle")
			}
			continue
		}
		if localPath != "" && strings.TrimSpace(asset.ID) != "" {
			assetPaths[asset.ID] = localPath
		}
	}

	payload := buildHTMLPayload(spec, assetPaths)
	html, err := buildHTMLDocument(payload)
	if err != nil {
		_ = zipWriter.Close()
		return err
	}
	if err := writeZipFile(zipWriter, "index.html", []byte(html)); err != nil {
		_ = zipWriter.Close()
		return err
	}

	readme := fmt.Sprintf("Vrooli Ascension Replay\n\nGenerated: %s\nFrames: %d\n\nOpen index.html in a browser to view the replay.\n",
		payload.GeneratedAt, len(payload.Frames))
	if err := writeZipFile(zipWriter, "README.txt", []byte(readme)); err != nil {
		_ = zipWriter.Close()
		return err
	}

	if err := zipWriter.Close(); err != nil {
		return err
	}
	return nil
}

func buildHTMLPayload(spec *ReplayMovieSpec, assetPaths map[string]string) htmlPayload {
	frames := make([]htmlPayloadFrame, 0, len(spec.Frames))
	for index, frame := range spec.Frames {
		viewport := resolveHTMLViewport(frame, spec)
		screenshotPath := ""
		if frame.ScreenshotAssetID != "" {
			if path, ok := assetPaths[frame.ScreenshotAssetID]; ok {
				screenshotPath = path
			}
		}

		highlightRegions := normalizeHighlightRegions(frame.HighlightRegions, viewport)
		cursorTrail := normalizeCursorTrail(frame.CursorTrail, viewport)
		cursorPoint := resolveCursorPoint(frame, cursorTrail, viewport)
		var cursor *htmlCursor
		if cursorPoint != nil {
			status := strings.ToLower(strings.TrimSpace(frame.Status))
			cursor = &htmlCursor{
				X:     cursorPoint.X,
				Y:     cursorPoint.Y,
				Pulse: status != "failure",
			}
		}

		title := sanitizeTitle(frame.Title, fmt.Sprintf("%s #%d", fallbackStepType(frame.StepType), index+1))
		subtitle := strings.ToUpper(strings.TrimSpace(frame.StepType))
		status := strings.ToLower(strings.TrimSpace(frame.Status))
		if status == "" {
			status = "success"
		}

		playDuration := resolvePlayDuration(frame.DurationMs, frame.HoldMs)
		assertionMessage := resolveAssertionMessage(frame.Assertion)
		retryLabel := resolveRetryLabel(frame.Resilience.Attempt, frame.Resilience.MaxAttempts)
		durationLabel := ""
		if frame.DurationMs > 0 {
			durationLabel = fmt.Sprintf("%d ms", frame.DurationMs)
		}

		addressBar := frame.FinalURL
		if strings.TrimSpace(addressBar) == "" {
			addressBar = spec.Execution.WorkflowName
		}
		if strings.TrimSpace(addressBar) == "" {
			addressBar = "automation replay"
		}

		zoomFactor := frame.ZoomFactor
		if zoomFactor <= 0 {
			zoomFactor = 1
		}

		frames = append(frames, htmlPayloadFrame{
			Index:            index,
			Title:            title,
			Subtitle:         subtitle,
			Status:           status,
			PlayDuration:     playDuration,
			Screenshot:       screenshotPath,
			ZoomFactor:       zoomFactor,
			HighlightRegions: highlightRegions,
			Cursor:           cursor,
			CursorTrail:      cursorTrail,
			AddressBar:       addressBar,
			FinalURL:         frame.FinalURL,
			AssertionMessage: assertionMessage,
			RetryLabel:       retryLabel,
			DurationLabel:    durationLabel,
			ConsoleCount:     frame.ConsoleLogCount,
			NetworkCount:     frame.NetworkEventCount,
		})
	}

	return htmlPayload{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Execution: htmlPayloadExecution{
			ExecutionID:  spec.Execution.ExecutionID.String(),
			WorkflowID:   spec.Execution.WorkflowID.String(),
			WorkflowName: spec.Execution.WorkflowName,
			Status:       spec.Execution.Status,
			FrameCount:   len(frames),
		},
		Theme:  spec.Theme,
		Frames: frames,
	}
}

func resolveHTMLViewport(frame ExportFrame, spec *ReplayMovieSpec) ExportDimensions {
	width := frame.Viewport.Width
	height := frame.Viewport.Height
	if width <= 0 && spec != nil && spec.Presentation.Viewport.Width > 0 {
		width = spec.Presentation.Viewport.Width
	}
	if height <= 0 && spec != nil && spec.Presentation.Viewport.Height > 0 {
		height = spec.Presentation.Viewport.Height
	}
	if width <= 0 {
		width = defaultHTMLViewportWidth
	}
	if height <= 0 {
		height = defaultHTMLViewportHeight
	}
	return ExportDimensions{Width: width, Height: height}
}

func normalizeHighlightRegions(regions []*contracts.HighlightRegion, viewport ExportDimensions) []htmlHighlightRect {
	if len(regions) == 0 {
		return nil
	}
	result := make([]htmlHighlightRect, 0, len(regions))
	for _, region := range regions {
		if region == nil || region.BoundingBox == nil {
			continue
		}
		normalized := normalizeBox(region.BoundingBox, viewport)
		if normalized == nil {
			continue
		}
		color := strings.TrimSpace(region.GetCustomRgba())
		result = append(result, htmlHighlightRect{
			Left:   normalized.Left,
			Top:    normalized.Top,
			Width:  normalized.Width,
			Height: normalized.Height,
			Color:  color,
		})
	}
	return result
}

func normalizeCursorTrail(points []*contracts.Point, viewport ExportDimensions) []htmlPoint {
	if len(points) == 0 {
		return nil
	}
	result := make([]htmlPoint, 0, len(points))
	for _, point := range points {
		normalized := normalizePoint(point, viewport)
		if normalized == nil {
			continue
		}
		result = append(result, htmlPoint{X: normalized.X, Y: normalized.Y})
	}
	return result
}

func resolveCursorPoint(frame ExportFrame, trail []htmlPoint, viewport ExportDimensions) *htmlPoint {
	if frame.NormalizedClickPosition != nil {
		return &htmlPoint{X: frame.NormalizedClickPosition.X, Y: frame.NormalizedClickPosition.Y}
	}
	if frame.ClickPosition != nil {
		normalized := normalizePoint(frame.ClickPosition, viewport)
		if normalized != nil {
			return &htmlPoint{X: normalized.X, Y: normalized.Y}
		}
	}
	if len(trail) > 0 {
		last := trail[len(trail)-1]
		return &htmlPoint{X: last.X, Y: last.Y}
	}
	return nil
}

func normalizeBox(box *contracts.BoundingBox, viewport ExportDimensions) *htmlHighlightRect {
	if box == nil || viewport.Width <= 0 || viewport.Height <= 0 {
		return nil
	}
	width := float64(viewport.Width)
	height := float64(viewport.Height)
	safeWidth := width
	safeHeight := height
	if safeWidth <= 0 {
		safeWidth = 1
	}
	if safeHeight <= 0 {
		safeHeight = 1
	}
	return &htmlHighlightRect{
		Left:   safeDivide(box.X, safeWidth),
		Top:    safeDivide(box.Y, safeHeight),
		Width:  safeDivide(box.Width, safeWidth),
		Height: safeDivide(box.Height, safeHeight),
	}
}

func normalizePoint(point *contracts.Point, viewport ExportDimensions) *htmlPoint {
	if point == nil || viewport.Width <= 0 || viewport.Height <= 0 {
		return nil
	}
	width := float64(viewport.Width)
	height := float64(viewport.Height)
	safeWidth := width
	safeHeight := height
	if safeWidth <= 0 {
		safeWidth = 1
	}
	if safeHeight <= 0 {
		safeHeight = 1
	}
	return &htmlPoint{
		X: safeDivide(point.X, safeWidth),
		Y: safeDivide(point.Y, safeHeight),
	}
}

func safeDivide(value float64, denom float64) float64 {
	if denom == 0 {
		return 0
	}
	return value / denom
}

func resolvePlayDuration(durationMs int, holdMs int) int {
	duration := durationMs
	if duration <= 0 {
		duration = defaultHTMLFrameDuration
	}
	if duration < minHTMLFrameDuration {
		duration = minHTMLFrameDuration
	}
	hold := holdMs
	if hold <= 0 {
		hold = defaultHTMLHoldDuration
	}
	return duration + hold
}

func resolveAssertionMessage(assertion *contracts.AssertionOutcome) string {
	if assertion == nil {
		return ""
	}
	if msg := strings.TrimSpace(assertion.Message); msg != "" {
		return msg
	}
	mode := strings.TrimSpace(assertion.Mode)
	selector := strings.TrimSpace(assertion.Selector)
	if mode != "" && selector != "" {
		return fmt.Sprintf("%s on %s", mode, selector)
	}
	return ""
}

func resolveRetryLabel(attempt int, maxAttempts int) string {
	if attempt <= 0 || maxAttempts <= 0 {
		return ""
	}
	return fmt.Sprintf("%d / %d", attempt, maxAttempts)
}

func fallbackStepType(stepType string) string {
	stepType = strings.TrimSpace(stepType)
	if stepType == "" {
		return "step"
	}
	return stepType
}

func sanitizeTitle(title string, fallback string) string {
	if trimmed := strings.TrimSpace(title); trimmed != "" {
		return trimmed
	}
	return fallback
}

func buildHTMLDocument(payload htmlPayload) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	escaped := escapeForScript(string(jsonPayload))

	gradient := "radial-gradient(circle at 20% 20%, #1e3a8a, #020617)"
	if len(payload.Theme.BackgroundGradient) > 0 {
		gradient = fmt.Sprintf("linear-gradient(135deg, %s)", strings.Join(payload.Theme.BackgroundGradient, ", "))
	}
	accent := "#38bdf8"
	if strings.TrimSpace(payload.Theme.AccentColor) != "" {
		accent = payload.Theme.AccentColor
	}
	surface := "rgba(15, 23, 42, 0.86)"
	if strings.TrimSpace(payload.Theme.SurfaceColor) != "" {
		surface = payload.Theme.SurfaceColor
	}

	title := sanitizeTitle(payload.Execution.WorkflowName, "Automation Workflow")
	executionID := payload.Execution.ExecutionID

	html := htmlTemplate
	replacer := strings.NewReplacer(
		"{{GRADIENT}}", gradient,
		"{{ACCENT}}", accent,
		"{{SURFACE}}", surface,
		"{{WORKFLOW_NAME}}", title,
		"{{EXECUTION_ID}}", executionID,
		"{{PAYLOAD}}", escaped,
	)
	return replacer.Replace(html), nil
}

func escapeForScript(input string) string {
	replacer := strings.NewReplacer(
		string(rune(0x2028)), "",
		string(rune(0x2029)), "",
		"<", "\\u003c",
		">", "\\u003e",
	)
	return replacer.Replace(input)
}

func writeReplayAsset(
	ctx context.Context,
	zipWriter *zip.Writer,
	asset ExportAsset,
	index int,
	storageClient storage.StorageInterface,
	log *logrus.Logger,
	baseURL string,
) (string, error) {
	source := strings.TrimSpace(asset.Source)
	if source == "" || strings.HasPrefix(source, "inline:") {
		return "", nil
	}

	filename := buildAssetFilename(asset, index)
	if filename == "" {
		return "", nil
	}
	targetPath := path.Join("assets", filename)

	if storageClient != nil {
		if objectName, ok := resolveScreenshotObjectName(source); ok {
			reader, _, err := storageClient.GetScreenshot(ctx, objectName)
			if err != nil {
				return "", err
			}
			defer reader.Close()
			if err := writeZipFileFromReader(zipWriter, targetPath, reader); err != nil {
				return "", err
			}
			return targetPath, nil
		}
	}

	assetURL, err := resolveAssetURL(source, baseURL)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("asset download failed (%d)", resp.StatusCode)
	}
	if err := writeZipFileFromReader(zipWriter, targetPath, resp.Body); err != nil {
		return "", err
	}
	if log != nil {
		log.WithField("asset_url", assetURL).Debug("Downloaded replay asset for HTML export")
	}
	return targetPath, nil
}

func buildAssetFilename(asset ExportAsset, index int) string {
	rawID := strings.TrimSpace(asset.ID)
	if rawID == "" {
		rawID = fmt.Sprintf("asset-%d", index+1)
	}
	safeID := sanitizeAssetID(rawID)
	ext := resolveAssetExtension(asset.Source)
	if ext == "" {
		ext = ".bin"
	}
	return safeID + ext
}

func sanitizeAssetID(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "asset"
	}
	return result
}

func resolveAssetExtension(source string) string {
	if source == "" {
		return ""
	}
	parsed, err := url.Parse(source)
	if err == nil {
		if ext := path.Ext(parsed.Path); ext != "" {
			return ext
		}
	}
	if ext := path.Ext(source); ext != "" {
		return ext
	}
	return ""
}

func resolveScreenshotObjectName(source string) (string, bool) {
	parsed, err := url.Parse(source)
	if err != nil {
		return "", false
	}
	assetPath := parsed.Path
	if assetPath == "" {
		return "", false
	}
	assetPath = strings.TrimPrefix(assetPath, "/")
	for _, prefix := range []string{"api/v1/screenshots/thumbnail/", "api/v1/screenshots/"} {
		if strings.HasPrefix(assetPath, prefix) {
			return strings.TrimPrefix(assetPath, prefix), true
		}
	}
	return "", false
}

func resolveAssetURL(source string, baseURL string) (string, error) {
	parsed, err := url.Parse(source)
	if err == nil && parsed.IsAbs() {
		return parsed.String(), nil
	}
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("base URL required to resolve asset %q", source)
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimPrefix(source, "/")
	rel, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(rel).String(), nil
}

func writeZipFile(zipWriter *zip.Writer, name string, data []byte) error {
	if zipWriter == nil {
		return fmt.Errorf("zip writer is nil")
	}
	entry, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = entry.Write(data)
	return err
}

func writeZipFileFromReader(zipWriter *zip.Writer, name string, reader io.Reader) error {
	if zipWriter == nil {
		return fmt.Errorf("zip writer is nil")
	}
	entry, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, reader)
	return err
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Vrooli Ascension Replay</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    :root {
      color-scheme: dark;
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    }
    body {
      margin: 0;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background: {{GRADIENT}};
      overflow: hidden;
      color: #e2e8f0;
    }
    .backdrop {
      position: absolute;
      inset: 0;
      overflow: hidden;
      z-index: 0;
    }
    .backdrop::before {
      content: '';
      position: absolute;
      width: 120vmax;
      height: 120vmax;
      background: radial-gradient(circle, rgba(56,189,248,0.18), transparent 70%);
      top: -40vmax;
      right: -20vmax;
      filter: blur(120px);
      opacity: 0.7;
    }
    .backdrop::after {
      content: '';
      position: absolute;
      width: 100vmax;
      height: 100vmax;
      background: radial-gradient(circle, rgba(59,130,246,0.18), transparent 70%);
      bottom: -30vmax;
      left: -25vmax;
      filter: blur(140px);
      opacity: 0.6;
    }
    .shell {
      position: relative;
      z-index: 1;
      width: min(1200px, 92vw);
      display: flex;
      flex-direction: column;
      gap: 20px;
      padding: 24px;
    }
    .header {
      display: flex;
      justify-content: space-between;
      flex-wrap: wrap;
      row-gap: 8px;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;
      padding: 4px 12px;
      border-radius: 999px;
      background: rgba(148, 163, 184, 0.16);
      backdrop-filter: blur(6px);
    }
    .stage {
      position: relative;
      border-radius: 28px;
      background: {{SURFACE}};
      box-shadow: 0 40px 120px rgba(15, 23, 42, 0.55);
      padding: 36px;
      backdrop-filter: blur(18px);
    }
    .browser {
      position: relative;
      border-radius: 18px;
      background: #0f172a;
      overflow: hidden;
      border: 1px solid rgba(148, 163, 184, 0.14);
      box-shadow: inset 0 0 0 1px rgba(255,255,255,0.04);
    }
    .browser-bar {
      height: 44px;
      display: flex;
      align-items: center;
      padding: 0 18px;
      border-bottom: 1px solid rgba(148, 163, 184, 0.12);
      background: rgba(15, 23, 42, 0.95);
      backdrop-filter: blur(8px);
    }
    .browser-dots {
      display: flex;
      gap: 6px;
      margin-right: 12px;
    }
    .dot {
      width: 12px;
      height: 12px;
      border-radius: 999px;
    }
    .dot.red { background: #f87171; }
    .dot.amber { background: #fbbf24; }
    .dot.green { background: #34d399; }
    .address-bar {
      flex: 1;
      min-width: 0;
      background: rgba(15, 23, 42, 0.7);
      border-radius: 999px;
      padding: 7px 16px;
      font-size: 14px;
      color: rgba(226, 232, 240, 0.82);
      border: 1px solid rgba(148, 163, 184, 0.22);
    }
    .viewport {
      position: relative;
      width: 100%;
      aspect-ratio: 16 / 9;
      overflow: hidden;
      background: #020617;
    }
    #frame-image {
      width: 100%;
      height: 100%;
      object-fit: contain;
      transition: transform 1.4s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.5s ease;
      opacity: 0;
    }
    #frame-image.ready {
      opacity: 1;
    }
    .overlay-layer {
      position: absolute;
      inset: 0;
      pointer-events: none;
    }
    .highlight {
      position: absolute;
      border: 2px solid {{ACCENT}};
      border-radius: 12px;
      box-shadow: 0 0 0 6px rgba(56, 189, 248, 0.35);
      mix-blend-mode: screen;
      transition: opacity 0.3s ease;
    }
    .cursor {
      position: absolute;
      width: 36px;
      height: 36px;
      border-radius: 999px;
      border: 3px solid {{ACCENT}};
      transform: translate(-50%, -50%) scale(0.86);
      box-shadow: 0 0 30px rgba(56, 189, 248, 0.55);
      opacity: 0;
      transition: opacity 0.4s ease, transform 0.45s ease;
    }
    .cursor.visible {
      opacity: 1;
    }
    .info-panel {
      margin-top: 18px;
      display: flex;
      justify-content: space-between;
      flex-wrap: wrap;
      gap: 12px;
      font-size: 14px;
      color: rgba(226, 232, 240, 0.82);
    }
    .step-title {
      font-size: 22px;
      font-weight: 600;
      color: #f8fafc;
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 12px;
      border-radius: 999px;
      background: rgba(59, 130, 246, 0.15);
      color: #93c5fd;
      font-size: 13px;
    }
    .status-pill.success {
      background: rgba(34, 197, 94, 0.16);
      color: #bbf7d0;
    }
    .status-pill.failure {
      background: rgba(248, 113, 113, 0.18);
      color: #fecaca;
    }
    .timeline {
      position: relative;
      width: 100%;
      height: 4px;
      background: rgba(148, 163, 184, 0.25);
      border-radius: 999px;
      overflow: hidden;
      margin-top: 16px;
    }
    .timeline-progress {
      position: absolute;
      inset: 0;
      background: linear-gradient(90deg, {{ACCENT}}, rgba(148, 163, 184, 0.45));
      transform-origin: left;
      transform: scaleX(0);
      transition: transform linear;
    }
    .meta-grid {
      display: flex;
      flex-wrap: wrap;
      gap: 14px 24px;
      margin-top: 18px;
      font-size: 13px;
    }
    .meta-label {
      text-transform: uppercase;
      letter-spacing: 0.12em;
      font-size: 11px;
      color: rgba(226, 232, 240, 0.56);
    }
    .meta-value {
      margin-top: 4px;
      color: rgba(226, 232, 240, 0.86);
    }
    @media (max-width: 900px) {
      .stage { padding: 18px; border-radius: 20px; }
      .browser { border-radius: 16px; }
      .viewport { aspect-ratio: 4 / 3; }
    }
  </style>
</head>
<body>
  <div class="backdrop"></div>
  <div class="shell">
    <div class="header">
      <div>
        <div class="badge">Automation Replay</div>
        <div style="margin-top:6px;font-size:28px;font-weight:600;">{{WORKFLOW_NAME}}</div>
      </div>
      <div class="badge" id="execution-summary">{{EXECUTION_ID}}</div>
    </div>
    <div class="stage">
      <div class="browser">
        <div class="browser-bar">
          <div class="browser-dots">
            <span class="dot red"></span>
            <span class="dot amber"></span>
            <span class="dot green"></span>
          </div>
          <div class="address-bar" id="address-bar">Loading replay...</div>
        </div>
        <div class="viewport" id="viewport">
          <img id="frame-image" alt="Replay frame" />
          <div class="overlay-layer" id="highlight-layer"></div>
          <div class="cursor" id="cursor"></div>
          <div class="timeline"><div class="timeline-progress" id="timeline-progress"></div></div>
        </div>
      </div>
      <div class="info-panel">
        <div>
          <div class="step-title" id="step-title"></div>
          <div style="margin-top:6px; color: rgba(226,232,240,0.7);" id="step-description"></div>
        </div>
        <div class="status-pill" id="step-status">PREPARING</div>
      </div>
      <div class="meta-grid" id="meta-grid"></div>
    </div>
  </div>
  <script>
    window.__BAS_REPLAY__ = {{PAYLOAD}};
    (function initReplay() {
      const data = window.__BAS_REPLAY__;
      const frames = Array.isArray(data.frames) ? data.frames : [];
      if (!frames.length) {
        document.getElementById('step-title').textContent = 'No frames available';
        return;
      }

      const img = document.getElementById('frame-image');
      const highlightLayer = document.getElementById('highlight-layer');
      const cursor = document.getElementById('cursor');
      let cursorAnimationHandle = null;
      let cursorPulseAnimation = null;
      const titleEl = document.getElementById('step-title');
      const descEl = document.getElementById('step-description');
      const statusEl = document.getElementById('step-status');
      const addressBar = document.getElementById('address-bar');
      const metaGrid = document.getElementById('meta-grid');
      const timelineProgress = document.getElementById('timeline-progress');
      const executionSummary = document.getElementById('execution-summary');

      if (data.execution && data.execution.executionId) {
        executionSummary.textContent = data.execution.executionId;
      }

      const defaultHold = 900;
      let playhead = 0;
      let timer = null;

      function clamp(value) {
        if (typeof value !== 'number' || Number.isNaN(value)) {
          return 0;
        }
        if (value < 0) return 0;
        if (value > 1) return 1;
        return value;
      }

      function stopCursorAnimations() {
        if (cursorAnimationHandle !== null) {
          cancelAnimationFrame(cursorAnimationHandle);
          cursorAnimationHandle = null;
        }
        if (cursorPulseAnimation && typeof cursorPulseAnimation.cancel === 'function') {
          cursorPulseAnimation.cancel();
        }
        cursorPulseAnimation = null;
      }

      function setCursorPosition(point) {
        if (!point || typeof point.x !== 'number' || typeof point.y !== 'number') {
          return;
        }
        const clampedX = clamp(point.x);
        const clampedY = clamp(point.y);
        cursor.style.left = (clampedX * 100) + '%';
        cursor.style.top = (clampedY * 100) + '%';
      }

      function animateCursorTrail(trail, duration) {
        if (!Array.isArray(trail) || trail.length === 0) {
          return;
        }

        const sanitizedTrail = trail.map((point) => ({
          x: clamp(point.x),
          y: clamp(point.y),
        }));

        if (sanitizedTrail.length === 1) {
          setCursorPosition(sanitizedTrail[0]);
          return;
        }

        const totalSegments = sanitizedTrail.length - 1;
        const effectiveDuration = Math.max(duration - 250, 400);
        const start = performance.now();

        function step(now) {
          const elapsed = now - start;
          const ratio = totalSegments > 0 ? Math.min(elapsed / effectiveDuration, 1) : 1;
          const scaled = ratio * totalSegments;
          const index = Math.min(Math.floor(scaled), totalSegments - 1);
          const segmentT = scaled - index;
          const from = sanitizedTrail[index];
          const to = sanitizedTrail[index + 1];
          setCursorPosition({
            x: from.x + ((to.x - from.x) * segmentT),
            y: from.y + ((to.y - from.y) * segmentT),
          });

          if (ratio < 1) {
            cursorAnimationHandle = requestAnimationFrame(step);
          } else {
            cursorAnimationHandle = null;
            setCursorPosition(sanitizedTrail[sanitizedTrail.length - 1]);
          }
        }

        stopCursorAnimations();
        setCursorPosition(sanitizedTrail[0]);
        cursorAnimationHandle = requestAnimationFrame(step);
      }

      function renderMeta(frame) {
        metaGrid.innerHTML = '';
        const entries = [];
        if (frame.finalUrl) {
          entries.push(['URL', frame.finalUrl]);
        }
        if (frame.durationLabel) {
          entries.push(['Duration', frame.durationLabel]);
        }
        if (frame.retryLabel) {
          entries.push(['Retries', frame.retryLabel]);
        }
        if (frame.consoleCount) {
          entries.push(['Console Logs', frame.consoleCount]);
        }
        if (frame.networkCount) {
          entries.push(['Network Events', frame.networkCount]);
        }
        if (frame.assertionMessage) {
          entries.push(['Assertion', frame.assertionMessage]);
        }

        entries.forEach(([label, value]) => {
          const wrapper = document.createElement('div');
          const labelEl = document.createElement('div');
          labelEl.className = 'meta-label';
          labelEl.textContent = label;
          const valueEl = document.createElement('div');
          valueEl.className = 'meta-value';
          valueEl.textContent = value;
          wrapper.appendChild(labelEl);
          wrapper.appendChild(valueEl);
          metaGrid.appendChild(wrapper);
        });
      }

      function applyFrame(frame, instant) {
        if (!frame) return;
        const transitionDelay = instant ? 0 : 180;
        img.classList.remove('ready');
        if (frame.screenshot) {
          setTimeout(() => {
            img.src = frame.screenshot;
          }, transitionDelay);
          img.onload = () => {
            img.classList.add('ready');
            const scale = frame.zoomFactor && frame.zoomFactor > 1 ? frame.zoomFactor : 1;
            img.style.transform = 'scale(' + scale + ')';
          };
        }

        highlightLayer.innerHTML = '';
        if (Array.isArray(frame.highlightRegions)) {
          frame.highlightRegions.forEach((region) => {
            if (!region) return;
            const node = document.createElement('div');
            node.className = 'highlight';
            node.style.left = (region.left * 100) + '%';
            node.style.top = (region.top * 100) + '%';
            node.style.width = (region.width * 100) + '%';
            node.style.height = (region.height * 100) + '%';
            if (region.color) {
              node.style.boxShadow = '0 0 0 6px ' + region.color + '44';
              node.style.borderColor = region.color;
            }
            highlightLayer.appendChild(node);
          });
        }

        stopCursorAnimations();
        const trail = Array.isArray(frame.cursorTrail) ? frame.cursorTrail : [];
        const hasTrail = trail.length > 0;
        const hasCursor = frame.cursor && typeof frame.cursor.x === 'number' && typeof frame.cursor.y === 'number';

        if (hasTrail || hasCursor) {
          cursor.classList.add('visible');
          cursor.style.transform = 'translate(-50%, -50%) scale(1)';

          if (hasTrail) {
            animateCursorTrail(trail, frame.playDuration || defaultHold);
          } else if (hasCursor) {
            setCursorPosition(frame.cursor);
          }

          const shouldPulse = frame.cursor && frame.cursor.pulse;
          if (shouldPulse) {
            cursorPulseAnimation = cursor.animate([
              { transform: 'translate(-50%, -50%) scale(0.85)', opacity: 0.9 },
              { transform: 'translate(-50%, -50%) scale(1.1)', opacity: 1 },
              { transform: 'translate(-50%, -50%) scale(0.9)', opacity: 0.9 }
            ], {
              duration: 900,
              easing: 'ease-in-out'
            });
          }
        } else {
          cursor.classList.remove('visible');
        }

        titleEl.textContent = frame.title || 'Untitled step';
        descEl.textContent = frame.subtitle || '';
        addressBar.textContent = frame.addressBar || 'automation replay';

        statusEl.classList.remove('success', 'failure');
        if (frame.status === 'success') {
          statusEl.textContent = 'SUCCESS';
          statusEl.classList.add('success');
        } else if (frame.status === 'failure') {
          statusEl.textContent = 'FAILURE';
          statusEl.classList.add('failure');
        } else {
          statusEl.textContent = String(frame.status || '').toUpperCase() || 'STEP';
        }

        renderMeta(frame);

        const duration = frame.playDuration;
        timelineProgress.style.transitionDuration = duration + 'ms';
        timelineProgress.style.transform = 'scaleX(0)';
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            timelineProgress.style.transform = 'scaleX(1)';
          });
        });
      }

      function scheduleNext() {
        if (timer) {
          clearTimeout(timer);
        }
        const frame = frames[playhead];
        applyFrame(frame, playhead === 0);
        const delay = frame.playDuration || defaultHold;
        timer = setTimeout(() => {
          playhead = (playhead + 1) % frames.length;
          scheduleNext();
        }, delay);
      }

      scheduleNext();
    })();
  </script>
</body>
</html>`
