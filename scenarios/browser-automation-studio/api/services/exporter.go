//go:generate go run ../cmd/movie-spec-gen/main.go

package services

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

const (
	replayExportSchemaVersion      = "2025-11-07"
	defaultFrameDurationMs         = 1600
	minFrameDurationMs             = 900
	minHoldDurationMs              = 600
	transitionFraction             = 0.22
	minTransitionDurationMs        = 180
	maxTransitionDurationMs        = 600
	fallbackViewportWidth          = 1920
	fallbackViewportHeight         = 1080
	defaultPlaybackFrameIntervalMs = 40
	defaultCursorSpeedProfile      = "easeInOut"
	defaultCursorPathStyle         = "bezier"
)

// ReplayMovieSpec encapsulates replay metadata ready for renderer/export tooling.
type ReplayMovieSpec struct {
	Version      string                  `json:"version"`
	GeneratedAt  time.Time               `json:"generated_at"`
	Execution    ExportExecutionMetadata `json:"execution"`
	Theme        ExportTheme             `json:"theme"`
	Cursor       ExportCursorSpec        `json:"cursor"`
	Decor        ExportDecor             `json:"decor"`
	Playback     ExportPlayback          `json:"playback"`
	Presentation ExportPresentation      `json:"presentation"`
	CursorMotion ExportCursorMotion      `json:"cursor_motion"`
	Frames       []ExportFrame           `json:"frames"`
	Assets       []ExportAsset           `json:"assets"`
	Summary      ExportSummary           `json:"summary"`
}

// ExportExecutionMetadata summarises execution context for the replay movie spec.
type ExportExecutionMetadata struct {
	ExecutionID   uuid.UUID  `json:"execution_id"`
	WorkflowID    uuid.UUID  `json:"workflow_id"`
	WorkflowName  string     `json:"workflow_name,omitempty"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Progress      int        `json:"progress"`
	TotalDuration int        `json:"total_duration_ms"`
}

// ExportTheme encodes chrome/backdrop styling hints for marketing renders.
type ExportTheme struct {
	BackgroundGradient []string            `json:"background_gradient"`
	BackgroundPattern  string              `json:"background_pattern"`
	AccentColor        string              `json:"accent_color"`
	SurfaceColor       string              `json:"surface_color"`
	AmbientGlow        string              `json:"ambient_glow"`
	BrowserChrome      ExportBrowserChrome `json:"browser_chrome"`
}

// ExportBrowserChrome defines the stylised browser frame settings.
type ExportBrowserChrome struct {
	Visible     bool   `json:"visible"`
	Variant     string `json:"variant"`
	Title       string `json:"title,omitempty"`
	ShowAddress bool   `json:"show_address"`
	AccentColor string `json:"accent_color"`
}

// ExportCursorSpec configures synthetic cursor overlays for the replay.
type ExportCursorSpec struct {
	Style       string            `json:"style"`
	AccentColor string            `json:"accent_color"`
	Trail       ExportCursorTrail `json:"trail"`
	ClickPulse  ExportClickPulse  `json:"click_pulse"`
	Scale       float64           `json:"scale,omitempty"`
	InitialPos  string            `json:"initial_position,omitempty"`
	ClickAnim   string            `json:"click_animation,omitempty"`
}

// ExportDecor captures the high-level presentation presets selected in the UI.
type ExportDecor struct {
	ChromeTheme          string  `json:"chrome_theme,omitempty"`
	BackgroundTheme      string  `json:"background_theme,omitempty"`
	CursorTheme          string  `json:"cursor_theme,omitempty"`
	CursorInitial        string  `json:"cursor_initial_position,omitempty"`
	CursorClickAnimation string  `json:"cursor_click_animation,omitempty"`
	CursorScale          float64 `json:"cursor_scale,omitempty"`
}

// ExportPlayback captures canonical playback configuration for renderers.
type ExportPlayback struct {
	FPS             int `json:"fps"`
	DurationMs      int `json:"duration_ms"`
	FrameIntervalMs int `json:"frame_interval_ms"`
	TotalFrames     int `json:"total_frames"`
}

// ExportFrameRect describes the chrome bounding box for presentation framing.
type ExportFrameRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
	Radius int `json:"radius,omitempty"`
}

// ExportPresentation captures the canvas and viewport dimensions for replay.
type ExportPresentation struct {
	Canvas            ExportDimensions `json:"canvas"`
	Viewport          ExportDimensions `json:"viewport"`
	BrowserFrame      ExportFrameRect  `json:"browser_frame"`
	DeviceScaleFactor float64          `json:"device_scale_factor"`
}

// ExportCursorMotion summarises default cursor movement behaviours.
type ExportCursorMotion struct {
	SpeedProfile    string  `json:"speed_profile"`
	PathStyle       string  `json:"path_style"`
	InitialPosition string  `json:"initial_position"`
	ClickAnimation  string  `json:"click_animation"`
	CursorScale     float64 `json:"cursor_scale"`
}

// ExportCursorTrail describes cursor trail animation.
type ExportCursorTrail struct {
	Enabled bool    `json:"enabled"`
	FadeMs  int     `json:"fade_ms"`
	Weight  float64 `json:"weight"`
	Opacity float64 `json:"opacity"`
}

// ExportClickPulse configures click pulse animation parameters.
type ExportClickPulse struct {
	Enabled    bool    `json:"enabled"`
	Radius     float64 `json:"radius"`
	DurationMs int     `json:"duration_ms"`
	Opacity    float64 `json:"opacity"`
}

// ExportFrame captures per-step replay metadata plus animation hints.
type ExportFrame struct {
	Index                   int                             `json:"index"`
	StepIndex               int                             `json:"step_index"`
	NodeID                  string                          `json:"node_id"`
	StepType                string                          `json:"step_type"`
	Title                   string                          `json:"title"`
	Status                  string                          `json:"status"`
	StartOffsetMs           int                             `json:"start_offset_ms"`
	DurationMs              int                             `json:"duration_ms"`
	HoldMs                  int                             `json:"hold_ms"`
	Enter                   TransitionSpec                  `json:"enter"`
	Exit                    TransitionSpec                  `json:"exit"`
	ScreenshotAssetID       string                          `json:"screenshot_asset_id,omitempty"`
	Viewport                ExportDimensions                `json:"viewport"`
	ZoomFactor              float64                         `json:"zoom_factor,omitempty"`
	HighlightRegions        []autocontracts.HighlightRegion `json:"highlight_regions,omitempty"`
	MaskRegions             []autocontracts.MaskRegion      `json:"mask_regions,omitempty"`
	FocusedElement          *autocontracts.ElementFocus     `json:"focused_element,omitempty"`
	ElementBoundingBox      *autocontracts.BoundingBox      `json:"element_bounding_box,omitempty"`
	NormalizedFocusBounds   *ExportNormalizedRect           `json:"normalized_focus_bounds,omitempty"`
	NormalizedElementBounds *ExportNormalizedRect           `json:"normalized_element_bounds,omitempty"`
	ClickPosition           *autocontracts.Point            `json:"click_position,omitempty"`
	NormalizedClickPosition *ExportNormalizedPoint          `json:"normalized_click_position,omitempty"`
	CursorTrail             []autocontracts.Point           `json:"cursor_trail,omitempty"`
	NormalizedCursorTrail   []ExportNormalizedPoint         `json:"normalized_cursor_trail,omitempty"`
	ConsoleLogCount         int                             `json:"console_log_count,omitempty"`
	NetworkEventCount       int                             `json:"network_event_count,omitempty"`
	FinalURL                string                          `json:"final_url,omitempty"`
	Error                   string                          `json:"error,omitempty"`
	Assertion               *autocontracts.AssertionOutcome `json:"assertion,omitempty"`
	DomSnapshotPreview      string                          `json:"dom_snapshot_preview,omitempty"`
	DomSnapshotHTML         string                          `json:"dom_snapshot_html,omitempty"`
	Resilience              ExportResilience                `json:"resilience"`
}

// ExportResilience captures retry/backoff metadata for a step.
type ExportResilience struct {
	Attempt           int                 `json:"attempt"`
	MaxAttempts       int                 `json:"max_attempts"`
	ConfiguredRetries int                 `json:"configured_retries"`
	DelayMs           int                 `json:"delay_ms"`
	BackoffFactor     float64             `json:"backoff_factor"`
	History           []RetryHistoryEntry `json:"history,omitempty"`
}

// TransitionSpec defines enter/exit animation parameters.
type TransitionSpec struct {
	Type       string `json:"type"`
	DurationMs int    `json:"duration_ms"`
	Easing     string `json:"easing"`
}

// ExportAsset lists external assets (screenshots, audio) required for the replay.
type ExportAsset struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes *int64 `json:"size_bytes,omitempty"`
}

// ExportSummary aggregates package-level metrics.
type ExportSummary struct {
	FrameCount         int `json:"frame_count"`
	ScreenshotCount    int `json:"screenshot_count"`
	TotalDurationMs    int `json:"total_duration_ms"`
	MaxFrameDurationMs int `json:"max_frame_duration_ms"`
}

// ExportDimensions captures viewport dimensions for a frame.
type ExportDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ExportNormalizedPoint stores a point normalised to 0..1 range.
type ExportNormalizedPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ExportNormalizedRect stores a rectangle normalised to 0..1 range.
type ExportNormalizedRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// BuildReplayMovieSpec compiles execution timeline data into an export-ready movie specification.
func BuildReplayMovieSpec(
	execution *database.Execution,
	workflow *database.Workflow,
	timeline *ExecutionTimeline,
) (*ReplayMovieSpec, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is required for export")
	}
	if timeline == nil {
		return nil, fmt.Errorf("execution timeline is required for export")
	}

	frames := make([]ExportFrame, 0, len(timeline.Frames))
	assetMap := make(map[string]ExportAsset)

	totalDuration := 0
	maxFrameDuration := 0
	screenshotCount := 0
	startOffset := 0
	canvasDims := ExportDimensions{Width: fallbackViewportWidth, Height: fallbackViewportHeight}
	viewportDims := canvasDims

	for index, frame := range timeline.Frames {
		dims := dimensionsForFrame(frame)
		if index == 0 {
			if dims.Width > 0 && dims.Height > 0 {
				canvasDims = dims
				viewportDims = dims
			}
		}
		baseDuration := baseFrameDuration(frame)
		enterDuration := transitionDuration(baseDuration)
		exitDuration := transitionDuration(baseDuration)

		holdDuration := baseDuration - (enterDuration + exitDuration)
		if holdDuration < minHoldDurationMs {
			deficit := minHoldDurationMs - holdDuration
			holdDuration = minHoldDurationMs
			baseDuration += deficit
		}

		if baseDuration > maxFrameDuration {
			maxFrameDuration = baseDuration
		}

		totalDuration += baseDuration

		screenshotID := ""
		if frame.Screenshot != nil {
			screenshotID = frame.Screenshot.ArtifactID
			if screenshotID == "" && frame.Screenshot.URL != "" {
				screenshotID = fmt.Sprintf("%s-frame-%d", execution.ID.String(), index)
			}
			if screenshotID != "" {
				if _, exists := assetMap[screenshotID]; !exists {
					assetMap[screenshotID] = ExportAsset{
						ID:        screenshotID,
						Type:      "screenshot",
						Source:    frame.Screenshot.URL,
						Thumbnail: frame.Screenshot.ThumbnailURL,
						Width:     frame.Screenshot.Width,
						Height:    frame.Screenshot.Height,
						SizeBytes: frame.Screenshot.SizeBytes,
					}
					screenshotCount++
				}
			}
		}

		normalizedCursor := normalizeTrail(frame.CursorTrail, dims)
		normalizedClick := normalizePoint(frame.ClickPosition, dims)
		normalizedFocus := normalizeRect(nil, dims)
		if frame.FocusedElement != nil && frame.FocusedElement.BoundingBox != nil {
			normalizedFocus = normalizeRect(frame.FocusedElement.BoundingBox, dims)
		}
		normalizedElement := normalizeRect(frame.ElementBoundingBox, dims)

		domSnapshotHTML := ""
		if frame.DomSnapshot != nil && frame.DomSnapshot.Payload != nil {
			if raw := frame.DomSnapshot.Payload["html"]; raw != nil {
				switch v := raw.(type) {
				case string:
					domSnapshotHTML = v
				case fmt.Stringer:
					domSnapshotHTML = v.String()
				}
			}
		}

		frames = append(frames, ExportFrame{
			Index:                   index,
			StepIndex:               frame.StepIndex,
			NodeID:                  frame.NodeID,
			StepType:                frame.StepType,
			Title:                   frameTitle(frame),
			Status:                  frame.Status,
			StartOffsetMs:           startOffset,
			DurationMs:              baseDuration,
			HoldMs:                  holdDuration,
			Enter:                   transitionSpec(frame, true, enterDuration),
			Exit:                    transitionSpec(frame, false, exitDuration),
			ScreenshotAssetID:       screenshotID,
			Viewport:                dims,
			ZoomFactor:              frame.ZoomFactor,
			HighlightRegions:        frame.HighlightRegions,
			MaskRegions:             frame.MaskRegions,
			FocusedElement:          frame.FocusedElement,
			ElementBoundingBox:      frame.ElementBoundingBox,
			NormalizedFocusBounds:   normalizedFocus,
			NormalizedElementBounds: normalizedElement,
			ClickPosition:           frame.ClickPosition,
			NormalizedClickPosition: normalizedClick,
			CursorTrail:             frame.CursorTrail,
			NormalizedCursorTrail:   normalizedCursor,
			ConsoleLogCount:         frame.ConsoleLogCount,
			NetworkEventCount:       frame.NetworkEventCount,
			FinalURL:                frame.FinalURL,
			Error:                   strings.TrimSpace(frame.Error),
			Assertion:               frame.Assertion,
			DomSnapshotPreview:      frame.DomSnapshotPreview,
			DomSnapshotHTML:         domSnapshotHTML,
			Resilience: ExportResilience{
				Attempt:           frame.RetryAttempt,
				MaxAttempts:       frame.RetryMaxAttempts,
				ConfiguredRetries: frame.RetryConfigured,
				DelayMs:           frame.RetryDelayMs,
				BackoffFactor:     frame.RetryBackoffFactor,
				History:           frame.RetryHistory,
			},
		})

		startOffset += baseDuration
	}

	assets := make([]ExportAsset, 0, len(assetMap))
	for _, asset := range assetMap {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].ID < assets[j].ID
	})

	workflowName := ""
	if workflow != nil {
		workflowName = strings.TrimSpace(workflow.Name)
	}

	packageSummary := ExportSummary{
		FrameCount:         len(frames),
		ScreenshotCount:    screenshotCount,
		TotalDurationMs:    totalDuration,
		MaxFrameDurationMs: maxFrameDuration,
	}

	executionMeta := ExportExecutionMetadata{
		ExecutionID:   execution.ID,
		WorkflowID:    execution.WorkflowID,
		WorkflowName:  workflowName,
		Status:        execution.Status,
		StartedAt:     execution.StartedAt,
		CompletedAt:   execution.CompletedAt,
		Progress:      execution.Progress,
		TotalDuration: totalDuration,
	}

	accent := "#38bdf8"
	theme := defaultExportTheme(workflowName, accent)
	cursor := defaultCursorSpec(accent)
	decor := ExportDecor{
		ChromeTheme:          "aurora",
		BackgroundTheme:      "aurora",
		CursorTheme:          "white",
		CursorInitial:        cursor.InitialPos,
		CursorClickAnimation: cursor.ClickAnim,
		CursorScale:          cursor.Scale,
	}

	frameInterval := defaultPlaybackFrameIntervalMs
	if frameInterval <= 0 {
		frameInterval = 40
	}
	fps := 0
	if frameInterval > 0 {
		fps = int(math.Round(1000.0 / float64(frameInterval)))
		if fps <= 0 {
			fps = 25
		}
	}
	totalFrames := 0
	if frameInterval > 0 && packageSummary.TotalDurationMs > 0 {
		totalFrames = int(math.Ceil(float64(packageSummary.TotalDurationMs) / float64(frameInterval)))
	}

	presentation := ExportPresentation{
		Canvas:            canvasDims,
		Viewport:          viewportDims,
		BrowserFrame:      ExportFrameRect{X: 0, Y: 0, Width: canvasDims.Width, Height: canvasDims.Height, Radius: 24},
		DeviceScaleFactor: 1,
	}

	cursorMotion := ExportCursorMotion{
		SpeedProfile:    defaultCursorSpeedProfile,
		PathStyle:       defaultCursorPathStyle,
		InitialPosition: strings.TrimSpace(decor.CursorInitial),
		ClickAnimation:  strings.TrimSpace(decor.CursorClickAnimation),
		CursorScale:     decor.CursorScale,
	}
	if cursorMotion.InitialPosition == "" {
		cursorMotion.InitialPosition = cursor.InitialPos
	}
	if cursorMotion.ClickAnimation == "" {
		cursorMotion.ClickAnimation = cursor.ClickAnim
	}
	if cursorMotion.CursorScale <= 0 {
		cursorMotion.CursorScale = cursor.Scale
	}

	return &ReplayMovieSpec{
		Version:     replayExportSchemaVersion,
		GeneratedAt: time.Now().UTC(),
		Execution:   executionMeta,
		Theme:       theme,
		Cursor:      cursor,
		Decor:       decor,
		Playback: ExportPlayback{
			FPS:             fps,
			DurationMs:      packageSummary.TotalDurationMs,
			FrameIntervalMs: frameInterval,
			TotalFrames:     totalFrames,
		},
		Presentation: presentation,
		CursorMotion: cursorMotion,
		Frames:       frames,
		Assets:       assets,
		Summary:      packageSummary,
	}, nil
}

func defaultExportTheme(workflowName, accent string) ExportTheme {
	title := workflowName
	if strings.TrimSpace(title) == "" {
		title = "Browser Automation Studio"
	}

	return ExportTheme{
		BackgroundGradient: []string{"#0f172a", "#020617", "#111827"},
		BackgroundPattern:  "orbits",
		AccentColor:        accent,
		SurfaceColor:       "rgba(15,23,42,0.72)",
		AmbientGlow:        "rgba(56,189,248,0.22)",
		BrowserChrome: ExportBrowserChrome{
			Visible:     true,
			Variant:     "dark",
			Title:       title,
			ShowAddress: true,
			AccentColor: accent,
		},
	}
}

func defaultCursorSpec(accent string) ExportCursorSpec {
	return ExportCursorSpec{
		Style:       "halo",
		AccentColor: accent,
		Trail: ExportCursorTrail{
			Enabled: true,
			FadeMs:  650,
			Weight:  0.16,
			Opacity: 0.55,
		},
		ClickPulse: ExportClickPulse{
			Enabled:    true,
			Radius:     42,
			DurationMs: 420,
			Opacity:    0.65,
		},
		Scale:      1.0,
		InitialPos: "center",
		ClickAnim:  "pulse",
	}
}

func frameTitle(frame TimelineFrame) string {
	label := strings.TrimSpace(frame.StepType)
	if label == "" {
		label = "Step"
	}
	label = humanize(label)
	if frame.StepIndex >= 0 {
		return fmt.Sprintf("%s · Step %d", label, frame.StepIndex+1)
	}
	return label
}

func humanize(value string) string {
	cleaned := strings.ReplaceAll(value, "_", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}

	words := strings.Fields(cleaned)
	for i, word := range words {
		lower := strings.ToLower(word)
		if len(lower) == 0 {
			continue
		}
		words[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(words, " ")
}

func dimensionsForFrame(frame TimelineFrame) ExportDimensions {
	if frame.Screenshot != nil {
		if frame.Screenshot.Width > 0 && frame.Screenshot.Height > 0 {
			return ExportDimensions{Width: frame.Screenshot.Width, Height: frame.Screenshot.Height}
		}
	}
	return ExportDimensions{Width: fallbackViewportWidth, Height: fallbackViewportHeight}
}

func baseFrameDuration(frame TimelineFrame) int {
	duration := frame.TotalDurationMs
	if duration <= 0 {
		duration = frame.DurationMs
	}
	if duration <= 0 {
		duration = defaultFrameDurationMs
	}
	if duration < minFrameDurationMs {
		duration = minFrameDurationMs
	}
	return duration
}

func transitionDuration(base int) int {
	candidate := int(math.Round(float64(base) * transitionFraction))
	if candidate < minTransitionDurationMs {
		candidate = minTransitionDurationMs
	}
	if candidate > maxTransitionDurationMs {
		candidate = maxTransitionDurationMs
	}
	if candidate*2 > base {
		candidate = base / 2
	}
	if candidate <= 0 {
		candidate = minTransitionDurationMs
	}
	return candidate
}

func transitionSpec(frame TimelineFrame, enter bool, duration int) TransitionSpec {
	transitionType := "fade"
	if frame.ZoomFactor > 1.05 {
		if enter {
			transitionType = "zoom_in"
		} else {
			transitionType = "zoom_out"
		}
	} else if enter && len(frame.HighlightRegions) > 0 {
		transitionType = "spotlight"
	}

	easing := "easeOutCubic"
	if !enter {
		easing = "easeInCubic"
	}

	return TransitionSpec{
		Type:       transitionType,
		DurationMs: duration,
		Easing:     easing,
	}
}

func normalizePoint(point *autocontracts.Point, dims ExportDimensions) *ExportNormalizedPoint {
	if point == nil || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	return &ExportNormalizedPoint{
		X: clamp01(float64(point.X) / float64(dims.Width)),
		Y: clamp01(float64(point.Y) / float64(dims.Height)),
	}
}

func normalizeRect(box *autocontracts.BoundingBox, dims ExportDimensions) *ExportNormalizedRect {
	if box == nil || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	return &ExportNormalizedRect{
		X:      clamp01(float64(box.X) / float64(dims.Width)),
		Y:      clamp01(float64(box.Y) / float64(dims.Height)),
		Width:  clamp01(float64(box.Width) / float64(dims.Width)),
		Height: clamp01(float64(box.Height) / float64(dims.Height)),
	}
}

func normalizeTrail(trail []autocontracts.Point, dims ExportDimensions) []ExportNormalizedPoint {
	if len(trail) == 0 || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	normalized := make([]ExportNormalizedPoint, 0, len(trail))
	for _, point := range trail {
		normalized = append(normalized, ExportNormalizedPoint{
			X: clamp01(float64(point.X) / float64(dims.Width)),
			Y: clamp01(float64(point.Y) / float64(dims.Height)),
		})
	}
	return normalized
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
