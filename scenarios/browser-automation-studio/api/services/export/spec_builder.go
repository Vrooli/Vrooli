//go:generate go run ../../cmd/movie-spec-gen/main.go

package export

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

// BuildReplayMovieSpec compiles execution timeline data into an export-ready movie specification.
func BuildReplayMovieSpec(
	execution *database.ExecutionIndex,
	workflow *database.WorkflowIndex,
	timeline *ExecutionTimeline,
) (*ReplayMovieSpec, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is required for export")
	}
	if workflow == nil {
		return nil, fmt.Errorf("workflow context is required for export")
	}
	if timeline == nil {
		return nil, fmt.Errorf("execution timeline is required for export")
	}
	if timeline.ExecutionID != uuid.Nil && timeline.ExecutionID != execution.ID {
		return nil, fmt.Errorf("timeline execution id %s does not match execution %s", timeline.ExecutionID, execution.ID)
	}
	if timeline.WorkflowID != uuid.Nil && timeline.WorkflowID != workflow.ID {
		return nil, fmt.Errorf("timeline workflow id %s does not match workflow %s", timeline.WorkflowID, workflow.ID)
	}
	if len(timeline.Frames) == 0 {
		return nil, fmt.Errorf("execution timeline missing frames")
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

	// Progress is now stored in the result JSON file, not the index.
	// For export, we infer progress from status: completed = 100, else 0.
	progress := 0
	if execution.Status == database.ExecutionStatusCompleted {
		progress = 100
	}

	executionMeta := ExportExecutionMetadata{
		ExecutionID:   execution.ID,
		WorkflowID:    execution.WorkflowID,
		WorkflowName:  workflowName,
		Status:        execution.Status,
		StartedAt:     execution.StartedAt,
		CompletedAt:   execution.CompletedAt,
		Progress:      progress,
		TotalDuration: totalDuration,
	}

	accent := DefaultAccentColor
	theme := DefaultExportTheme(workflowName, accent)
	cursor := DefaultCursorSpec(accent)
	decor := DefaultDecor(cursor)

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

// frameTitle generates a human-readable title for a timeline frame.
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

// humanize converts snake_case or kebab-case to Title Case.
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

// dimensionsForFrame extracts viewport dimensions from a timeline frame.
func dimensionsForFrame(frame TimelineFrame) ExportDimensions {
	if frame.Screenshot != nil {
		if frame.Screenshot.Width > 0 && frame.Screenshot.Height > 0 {
			return ExportDimensions{Width: frame.Screenshot.Width, Height: frame.Screenshot.Height}
		}
	}
	return ExportDimensions{Width: fallbackViewportWidth, Height: fallbackViewportHeight}
}

// baseFrameDuration calculates the base duration for a frame.
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

// transitionDuration calculates the transition duration based on frame duration.
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

// transitionSpec builds a TransitionSpec for a frame's enter or exit animation.
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

// normalizePoint converts a point to normalized 0..1 coordinates.
func normalizePoint(point *autocontracts.Point, dims ExportDimensions) *ExportNormalizedPoint {
	if point == nil || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	return &ExportNormalizedPoint{
		X: clamp01(float64(point.X) / float64(dims.Width)),
		Y: clamp01(float64(point.Y) / float64(dims.Height)),
	}
}

// normalizeRect converts a bounding box to normalized 0..1 coordinates.
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

// normalizeTrail converts a cursor trail to normalized 0..1 coordinates.
func normalizeTrail(trail []*autocontracts.Point, dims ExportDimensions) []ExportNormalizedPoint {
	if len(trail) == 0 || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	normalized := make([]ExportNormalizedPoint, 0, len(trail))
	for _, point := range trail {
		if point == nil {
			continue
		}
		normalized = append(normalized, ExportNormalizedPoint{
			X: clamp01(float64(point.X) / float64(dims.Width)),
			Y: clamp01(float64(point.Y) / float64(dims.Height)),
		})
	}
	return normalized
}

// clamp01 clamps a value to the range [0, 1].
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
