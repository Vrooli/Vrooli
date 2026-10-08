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
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuildReplaySpecFromTimeline builds the canonical generated contract directly
// from timeline facts. Render-specific projection types are not involved.
func BuildReplaySpecFromTimeline(execution *database.ExecutionIndex, workflow *database.WorkflowIndex, timeline *ExecutionTimeline) (*exportsv1.ReplaySpec, error) {
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

	frames := make([]*exportsv1.ReplayFrame, 0, len(timeline.Frames))
	assetsByID := make(map[string]*exportsv1.ReplayAsset)
	var totalDuration, maxDuration int32
	var screenshotCount int32
	var startOffset int32
	canvas := &exportsv1.ReplayDimensions{Width: fallbackViewportWidth, Height: fallbackViewportHeight}
	viewport := &exportsv1.ReplayDimensions{Width: fallbackViewportWidth, Height: fallbackViewportHeight}
	deviceScaleFactor := timeline.DeviceScaleFactor

	for index, source := range timeline.Frames {
		dims := dimensionsForFrame(source)
		if timeline.ViewportWidth > 0 && timeline.ViewportHeight > 0 {
			dims = frameDimensions{Width: timeline.ViewportWidth, Height: timeline.ViewportHeight}
		}
		if index == 0 && dims.Width > 0 && dims.Height > 0 {
			viewport = &exportsv1.ReplayDimensions{Width: int32(dims.Width), Height: int32(dims.Height)}
		}
		baseDuration := baseFrameDuration(source)
		enterDuration, exitDuration := transitionDuration(baseDuration), transitionDuration(baseDuration)
		holdDuration := baseDuration - enterDuration - exitDuration
		if holdDuration < minHoldDurationMs {
			deficit := minHoldDurationMs - holdDuration
			holdDuration = minHoldDurationMs
			baseDuration += deficit
		}
		if baseDuration > int(maxDuration) {
			maxDuration = int32(baseDuration)
		}
		totalDuration += int32(baseDuration)
		screenshotID := ""
		if shot := source.Screenshot; shot != nil {
			if index == 0 && deviceScaleFactor <= 0 && timeline.ViewportWidth > 0 && shot.Width > 0 {
				deviceScaleFactor = float64(shot.Width) / float64(timeline.ViewportWidth)
			}
			screenshotID = shot.ArtifactID
			if screenshotID == "" && shot.URL != "" {
				screenshotID = fmt.Sprintf("%s-frame-%d", execution.ID.String(), index)
			}
			if screenshotID != "" {
				if _, exists := assetsByID[screenshotID]; !exists {
					assetsByID[screenshotID] = &exportsv1.ReplayAsset{Id: screenshotID, Type: "screenshot", Source: shot.URL, Thumbnail: shot.ThumbnailURL, Width: int32(shot.Width), Height: int32(shot.Height), SizeBytes: shot.SizeBytes}
					screenshotCount++
				}
			}
		}
		if index == 0 && source.Screenshot != nil && source.Screenshot.Width > 0 && source.Screenshot.Height > 0 {
			canvas = &exportsv1.ReplayDimensions{Width: int32(source.Screenshot.Width), Height: int32(source.Screenshot.Height)}
		}
		transitionIn := timelineTransition(source, true, enterDuration)
		transitionOut := timelineTransition(source, false, exitDuration)
		cursorTrail := source.CursorTrail
		if len(cursorTrail) == 0 && source.CursorPosition != nil {
			cursorTrail = []*autocontracts.Point{source.CursorPosition}
		}
		frameViewport := (*exportsv1.ReplayDimensions)(nil)
		if timeline.ViewportWidth > 0 && timeline.ViewportHeight > 0 {
			frameViewport = &exportsv1.ReplayDimensions{Width: int32(timeline.ViewportWidth), Height: int32(timeline.ViewportHeight)}
		}
		frame := &exportsv1.ReplayFrame{
			Index: int32(index), StepIndex: int32(source.StepIndex), NodeId: source.NodeID, StepType: source.StepType,
			Title: frameTitle(source), Status: source.Status, StartOffsetMs: startOffset, DurationMs: int32(baseDuration), HoldMs: int32(holdDuration),
			Enter: transitionIn, Exit: transitionOut, ScreenshotAssetId: screenshotID,
			Viewport: frameViewport, ZoomFactor: source.ZoomFactor,
			HighlightRegions: source.HighlightRegions, MaskRegions: source.MaskRegions, FocusedElement: source.FocusedElement,
			ElementBoundingBox: source.ElementBoundingBox, NormalizedFocusBounds: normalizeProtoRect(focusedBounds(source), dims),
			NormalizedElementBounds: normalizeProtoRect(source.ElementBoundingBox, dims), ClickPosition: source.ClickPosition,
			NormalizedClickPosition: normalizeProtoPoint(source.ClickPosition, dims), CursorTrail: cursorTrail,
			NormalizedCursorTrail: normalizeProtoTrail(cursorTrail, dims), ConsoleLogCount: int32(source.ConsoleLogCount),
			NetworkEventCount: int32(source.NetworkEventCount), FinalUrl: source.FinalURL, Error: strings.TrimSpace(source.Error),
			Assertion: timelineAssertion(source.Assertion), Resilience: timelineResilience(source),
		}
		frames = append(frames, frame)
		startOffset += int32(baseDuration)
	}
	assets := make([]*exportsv1.ReplayAsset, 0, len(assetsByID))
	for _, asset := range assetsByID {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].GetId() < assets[j].GetId() })

	workflowName := strings.TrimSpace(workflow.Name)
	accent := DefaultAccentColor
	progress := int32(0)
	if execution.Status == database.ExecutionStatusCompleted {
		progress = 100
	}
	frameInterval := int32(defaultPlaybackFrameIntervalMs)
	fps := int32(math.Round(1000 / float64(frameInterval)))
	if fps <= 0 {
		fps = 25
	}
	totalFrames := int32(0)
	if totalDuration > 0 {
		totalFrames = int32(math.Ceil(float64(totalDuration) / float64(frameInterval)))
	}
	return &exportsv1.ReplaySpec{
		Version: replayExportSchemaVersion, GeneratedAt: timestamppb.New(time.Now().UTC()),
		Execution:    &exportsv1.ReplayExecutionMetadata{ExecutionId: execution.ID.String(), WorkflowId: execution.WorkflowID.String(), WorkflowName: workflowName, Status: execution.Status, StartedAt: timestamppb.New(execution.StartedAt), CompletedAt: timestampOrNil(execution.CompletedAt), Progress: progress, TotalDurationMs: totalDuration},
		Theme:        &exportsv1.ReplayTheme{BackgroundGradient: []string{"#0f172a", "#020617", "#111827"}, BackgroundPattern: "orbits", AccentColor: accent, SurfaceColor: "rgba(15,23,42,0.72)", AmbientGlow: "rgba(56,189,248,0.22)", BrowserChrome: &exportsv1.ReplayBrowserChrome{Visible: true, Variant: "dark", Title: workflowName, ShowAddress: true, AccentColor: accent}},
		Cursor:       &exportsv1.ReplayCursor{Style: "halo", AccentColor: accent, Trail: &exportsv1.ReplayCursorTrail{Enabled: true, FadeMs: 650, Weight: 0.16, Opacity: 0.55}, ClickPulse: &exportsv1.ReplayClickPulse{Enabled: true, Radius: 42, DurationMs: 420, Opacity: 0.65}, Scale: 1, InitialPosition: "center", ClickAnimation: "pulse"},
		Decor:        &exportsv1.ReplayDecor{ChromeTheme: "aurora", BackgroundTheme: "aurora", Background: replayBackground("aurora"), CursorTheme: "white", CursorInitialPosition: "center", CursorClickAnimation: "pulse", CursorScale: 1},
		Playback:     &exportsv1.ReplayPlayback{Fps: fps, DurationMs: totalDuration, FrameIntervalMs: frameInterval, TotalFrames: totalFrames},
		Presentation: &exportsv1.ReplayPresentation{Canvas: canvas, Viewport: viewport, BrowserFrame: &exportsv1.ReplayFrameRect{Width: canvas.Width, Height: canvas.Height, Radius: 24}, DeviceScaleFactor: deviceScaleFactor},
		CursorMotion: &exportsv1.ReplayCursorMotion{SpeedProfile: defaultCursorSpeedProfile, PathStyle: defaultCursorPathStyle, InitialPosition: "center", ClickAnimation: "pulse", CursorScale: 1},
		Frames:       frames, Assets: assets, Summary: &exportsv1.ReplaySummary{FrameCount: int32(len(frames)), ScreenshotCount: screenshotCount, TotalDurationMs: totalDuration, MaxFrameDurationMs: maxDuration},
	}, nil
}

func timestampOrNil(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

func replayBackground(theme string) *structpb.Struct {
	background, _ := structpb.NewStruct(map[string]any{"type": "theme", "id": theme})
	return background
}

func timelineTransition(frame TimelineFrame, enter bool, duration int) *exportsv1.ReplayTransition {
	typeName := "fade"
	if frame.ZoomFactor > 1.05 {
		if enter {
			typeName = "zoom_in"
		} else {
			typeName = "zoom_out"
		}
	} else if enter && len(frame.HighlightRegions) > 0 {
		typeName = "spotlight"
	}
	easing := "easeOutCubic"
	if !enter {
		easing = "easeInCubic"
	}
	return &exportsv1.ReplayTransition{Type: typeName, DurationMs: int32(duration), Easing: easing}
}

func timelineAssertion(source *autocontracts.AssertionOutcome) *exportsv1.ReplayAssertionOutcome {
	if source == nil {
		return nil
	}
	return &exportsv1.ReplayAssertionOutcome{Mode: &source.Mode, Selector: &source.Selector, Expected: autocontracts.AnyToJsonValue(source.Expected), Actual: autocontracts.AnyToJsonValue(source.Actual), Success: source.Success, Negated: source.Negated, CaseSensitive: source.CaseSensitive, Message: &source.Message}
}

func timelineResilience(frame TimelineFrame) *exportsv1.ReplayResilience {
	history := make([]*exportsv1.RetryHistoryEntry, 0, len(frame.RetryHistory))
	for _, entry := range frame.RetryHistory {
		history = append(history, &exportsv1.RetryHistoryEntry{Attempt: int32(entry.Attempt), Success: entry.Success, DurationMs: int32(entry.DurationMs), CallDurationMs: int32(entry.CallDurationMs), Error: entry.Error})
	}
	return &exportsv1.ReplayResilience{Attempt: int32(frame.RetryAttempt), MaxAttempts: int32(frame.RetryMaxAttempts), ConfiguredRetries: int32(frame.RetryConfigured), DelayMs: int32(frame.RetryDelayMs), BackoffFactor: frame.RetryBackoffFactor, History: history}
}

func normalizeProtoPoint(point *autocontracts.Point, dims frameDimensions) *exportsv1.ReplayNormalizedPoint {
	if point == nil || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	return &exportsv1.ReplayNormalizedPoint{X: clamp01(float64(point.X) / float64(dims.Width)), Y: clamp01(float64(point.Y) / float64(dims.Height))}
}

func normalizeProtoRect(box *autocontracts.BoundingBox, dims frameDimensions) *exportsv1.ReplayNormalizedRect {
	if box == nil || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	return &exportsv1.ReplayNormalizedRect{X: clamp01(float64(box.X) / float64(dims.Width)), Y: clamp01(float64(box.Y) / float64(dims.Height)), Width: clamp01(float64(box.Width) / float64(dims.Width)), Height: clamp01(float64(box.Height) / float64(dims.Height))}
}

func focusedBounds(frame TimelineFrame) *autocontracts.BoundingBox {
	if frame.FocusedElement == nil {
		return nil
	}
	return frame.FocusedElement.BoundingBox
}

func normalizeProtoTrail(points []*autocontracts.Point, dims frameDimensions) []*exportsv1.ReplayNormalizedPoint {
	if len(points) == 0 || dims.Width <= 0 || dims.Height <= 0 {
		return nil
	}
	result := make([]*exportsv1.ReplayNormalizedPoint, 0, len(points))
	for _, point := range points {
		if normalized := normalizeProtoPoint(point, dims); normalized != nil {
			result = append(result, normalized)
		}
	}
	return result
}
