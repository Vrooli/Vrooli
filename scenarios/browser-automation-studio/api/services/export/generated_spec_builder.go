package export

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ErrMovieSpecUnavailable is returned when neither a generated replay spec nor
// a server baseline exists for export.
var ErrMovieSpecUnavailable = errors.New("movie spec unavailable")

// BuildReplaySpec merges an optional client contract with the server generated
// baseline and applies the stable defaults required by the replay renderer.
// The returned value remains the generated API contract; callers project it
// only when they enter the renderer.
func BuildReplaySpec(baseline, incoming *exportsv1.ReplaySpec, executionID uuid.UUID) (*exportsv1.ReplaySpec, error) {
	if baseline == nil && incoming == nil {
		return nil, ErrMovieSpecUnavailable
	}
	var spec *exportsv1.ReplaySpec
	if incoming != nil {
		spec = proto.Clone(incoming).(*exportsv1.ReplaySpec)
	} else {
		spec = proto.Clone(baseline).(*exportsv1.ReplaySpec)
	}
	if spec.Execution == nil {
		spec.Execution = &exportsv1.ReplayExecutionMetadata{}
	}
	if err := mergeReplayExecution(spec.Execution, baseline.GetExecution(), executionID); err != nil {
		return nil, err
	}
	if spec.Version == "" && baseline != nil {
		spec.Version = baseline.GetVersion()
	}
	if spec.Version == "" {
		spec.Version = replayExportSchemaVersion
	}
	if spec.GeneratedAt == nil || !spec.GeneratedAt.IsValid() || spec.GeneratedAt.AsTime().IsZero() {
		if baseline != nil && baseline.GetGeneratedAt().IsValid() && !baseline.GetGeneratedAt().AsTime().IsZero() {
			spec.GeneratedAt = proto.Clone(baseline.GeneratedAt).(*timestamppb.Timestamp)
		} else {
			spec.GeneratedAt = timestamppb.New(time.Now().UTC())
		}
	}
	if len(spec.Frames) == 0 && baseline != nil && len(baseline.Frames) > 0 {
		spec.Frames = cloneMessages(baseline.Frames)
	}
	if len(spec.Frames) == 0 {
		return nil, fmt.Errorf("movie spec missing frames")
	}
	if spec.Theme == nil {
		spec.Theme = &exportsv1.ReplayTheme{}
	}
	if spec.Decor == nil {
		spec.Decor = &exportsv1.ReplayDecor{}
	}
	if spec.Cursor == nil {
		spec.Cursor = &exportsv1.ReplayCursor{}
	}
	if spec.Cursor.Trail == nil {
		spec.Cursor.Trail = &exportsv1.ReplayCursorTrail{}
	}
	if spec.Cursor.ClickPulse == nil {
		spec.Cursor.ClickPulse = &exportsv1.ReplayClickPulse{}
	}
	if spec.CursorMotion == nil {
		spec.CursorMotion = &exportsv1.ReplayCursorMotion{}
	}
	if spec.Presentation == nil {
		spec.Presentation = &exportsv1.ReplayPresentation{}
	}
	if spec.Presentation.Canvas == nil {
		spec.Presentation.Canvas = &exportsv1.ReplayDimensions{}
	}
	if spec.Presentation.Viewport == nil {
		spec.Presentation.Viewport = &exportsv1.ReplayDimensions{}
	}
	if spec.Presentation.BrowserFrame == nil {
		spec.Presentation.BrowserFrame = &exportsv1.ReplayFrameRect{}
	}
	if spec.Playback == nil {
		spec.Playback = &exportsv1.ReplayPlayback{}
	}
	if spec.Summary == nil {
		spec.Summary = &exportsv1.ReplaySummary{}
	}
	mergeReplayNested(spec, baseline)
	if spec.Cursor.Trail == nil {
		spec.Cursor.Trail = &exportsv1.ReplayCursorTrail{}
	}
	if spec.Cursor.ClickPulse == nil {
		spec.Cursor.ClickPulse = &exportsv1.ReplayClickPulse{}
	}
	applyReplayDefaults(spec, baseline)
	if len(spec.Assets) == 0 && baseline != nil {
		spec.Assets = cloneMessages(baseline.Assets)
	}
	if spec.Execution.TotalDurationMs <= 0 {
		spec.Execution.TotalDurationMs = spec.Summary.TotalDurationMs
	}
	return spec, nil
}

func mergeReplayExecution(dst, src *exportsv1.ReplayExecutionMetadata, executionID uuid.UUID) error {
	wantID := executionID.String()
	if got := strings.TrimSpace(dst.GetExecutionId()); got == "" {
		dst.ExecutionId = wantID
	} else if got != wantID {
		return fmt.Errorf("movie spec execution_id mismatch")
	}
	if src == nil {
		return nil
	}
	if got := strings.TrimSpace(src.GetWorkflowId()); got != "" {
		if dst.WorkflowId == "" {
			dst.WorkflowId = got
		} else if dst.WorkflowId != got {
			return fmt.Errorf("movie spec workflow_id mismatch")
		}
	}
	if dst.WorkflowName == "" {
		dst.WorkflowName = src.GetWorkflowName()
	}
	if dst.Status == "" {
		dst.Status = src.GetStatus()
	}
	if dst.Progress == 0 {
		dst.Progress = src.GetProgress()
	}
	if dst.StartedAt == nil && src.StartedAt != nil {
		dst.StartedAt = proto.Clone(src.StartedAt).(*timestamppb.Timestamp)
	}
	if dst.CompletedAt == nil && src.CompletedAt != nil {
		dst.CompletedAt = proto.Clone(src.CompletedAt).(*timestamppb.Timestamp)
	}
	return nil
}

func mergeReplayNested(spec, baseline *exportsv1.ReplaySpec) {
	if baseline == nil {
		return
	}
	if spec.Theme.BackgroundGradient == nil {
		spec.Theme.BackgroundGradient = append([]string(nil), baseline.GetTheme().GetBackgroundGradient()...)
	}
	if spec.Theme.BackgroundPattern == "" {
		spec.Theme.BackgroundPattern = baseline.GetTheme().GetBackgroundPattern()
	}
	if spec.Theme.AccentColor == "" {
		spec.Theme.AccentColor = baseline.GetTheme().GetAccentColor()
	}
	if spec.Theme.SurfaceColor == "" {
		spec.Theme.SurfaceColor = baseline.GetTheme().GetSurfaceColor()
	}
	if spec.Theme.AmbientGlow == "" {
		spec.Theme.AmbientGlow = baseline.GetTheme().GetAmbientGlow()
	}
	if (spec.Theme.BrowserChrome == nil || (spec.Theme.BrowserChrome.Variant == "" && baseline.GetTheme().GetBrowserChrome().GetVariant() != "")) && baseline.GetTheme().GetBrowserChrome() != nil {
		spec.Theme.BrowserChrome = proto.Clone(baseline.Theme.BrowserChrome).(*exportsv1.ReplayBrowserChrome)
	} else if spec.Theme.BrowserChrome != nil && spec.Theme.BrowserChrome.AccentColor == "" {
		spec.Theme.BrowserChrome.AccentColor = spec.Theme.AccentColor
	}
	if spec.Decor.Background == nil && baseline.GetDecor().GetBackground() != nil {
		spec.Decor.Background = proto.Clone(baseline.Decor.Background).(*structpb.Struct)
	}
	if spec.Decor.ChromeTheme == "" {
		spec.Decor.ChromeTheme = baseline.GetDecor().GetChromeTheme()
	}
	if spec.Decor.BackgroundTheme == "" {
		spec.Decor.BackgroundTheme = baseline.GetDecor().GetBackgroundTheme()
	}
	if spec.Decor.CursorTheme == "" {
		spec.Decor.CursorTheme = baseline.GetDecor().GetCursorTheme()
	}
	if spec.Decor.CursorInitialPosition == "" {
		spec.Decor.CursorInitialPosition = baseline.GetDecor().GetCursorInitialPosition()
	}
	if spec.Decor.CursorClickAnimation == "" {
		spec.Decor.CursorClickAnimation = baseline.GetDecor().GetCursorClickAnimation()
	}
	if spec.Decor.CursorScale == 0 {
		spec.Decor.CursorScale = baseline.GetDecor().GetCursorScale()
	}
	if spec.Cursor.Style == "" && baseline.GetCursor() != nil {
		spec.Cursor = proto.Clone(baseline.Cursor).(*exportsv1.ReplayCursor)
	} else if baseline.GetCursor() != nil {
		baseCursor := baseline.Cursor
		if spec.Cursor.AccentColor == "" {
			spec.Cursor.AccentColor = baseCursor.GetAccentColor()
		}
		if spec.Cursor.InitialPosition == "" {
			spec.Cursor.InitialPosition = baseCursor.GetInitialPosition()
		}
		if spec.Cursor.ClickAnimation == "" {
			spec.Cursor.ClickAnimation = baseCursor.GetClickAnimation()
		}
		if spec.Cursor.Trail.FadeMs == 0 {
			spec.Cursor.Trail.FadeMs = baseCursor.GetTrail().GetFadeMs()
		}
		if spec.Cursor.Trail.Weight == 0 {
			spec.Cursor.Trail.Weight = baseCursor.GetTrail().GetWeight()
		}
		if spec.Cursor.Trail.Opacity == 0 {
			spec.Cursor.Trail.Opacity = baseCursor.GetTrail().GetOpacity()
		}
		if !spec.Cursor.Trail.Enabled && baseCursor.GetTrail().GetEnabled() {
			spec.Cursor.Trail.Enabled = true
		}
		if spec.Cursor.ClickPulse.Radius == 0 {
			spec.Cursor.ClickPulse.Radius = baseCursor.GetClickPulse().GetRadius()
		}
		if spec.Cursor.ClickPulse.DurationMs == 0 {
			spec.Cursor.ClickPulse.DurationMs = baseCursor.GetClickPulse().GetDurationMs()
		}
		if spec.Cursor.ClickPulse.Opacity == 0 {
			spec.Cursor.ClickPulse.Opacity = baseCursor.GetClickPulse().GetOpacity()
		}
		if !spec.Cursor.ClickPulse.Enabled && baseCursor.GetClickPulse().GetEnabled() {
			spec.Cursor.ClickPulse.Enabled = true
		}
	}
	if spec.Presentation.Canvas.Width == 0 && baseline.GetPresentation().GetCanvas() != nil {
		spec.Presentation.Canvas = proto.Clone(baseline.Presentation.Canvas).(*exportsv1.ReplayDimensions)
	}
	if spec.Presentation.Viewport.Width == 0 && baseline.GetPresentation().GetViewport() != nil {
		spec.Presentation.Viewport = proto.Clone(baseline.Presentation.Viewport).(*exportsv1.ReplayDimensions)
	}
	if spec.Presentation.BrowserFrame.Width == 0 && baseline.GetPresentation().GetBrowserFrame() != nil {
		spec.Presentation.BrowserFrame = proto.Clone(baseline.Presentation.BrowserFrame).(*exportsv1.ReplayFrameRect)
	}
	if spec.Presentation.DeviceScaleFactor == 0 {
		spec.Presentation.DeviceScaleFactor = baseline.GetPresentation().GetDeviceScaleFactor()
	}
	if spec.Cursor.Style == "" && baseline.GetCursor() != nil {
		spec.Cursor = proto.Clone(baseline.Cursor).(*exportsv1.ReplayCursor)
	}
	if spec.CursorMotion.SpeedProfile == "" {
		spec.CursorMotion.SpeedProfile = baseline.GetCursorMotion().GetSpeedProfile()
	}
	if spec.CursorMotion.PathStyle == "" {
		spec.CursorMotion.PathStyle = baseline.GetCursorMotion().GetPathStyle()
	}
	if spec.Playback.FrameIntervalMs == 0 {
		spec.Playback.FrameIntervalMs = baseline.GetPlayback().GetFrameIntervalMs()
	}
}

func applyReplayDefaults(spec, baseline *exportsv1.ReplaySpec) {
	if baseline == nil {
		if len(spec.Theme.BackgroundGradient) == 0 {
			spec.Theme.BackgroundGradient = []string{"#0f172a", "#020617", "#111827"}
		}
		if spec.Theme.BackgroundPattern == "" {
			spec.Theme.BackgroundPattern = "orbits"
		}
		if spec.Theme.AccentColor == "" {
			spec.Theme.AccentColor = DefaultAccentColor
		}
		if spec.Theme.SurfaceColor == "" {
			spec.Theme.SurfaceColor = "rgba(15,23,42,0.72)"
		}
		if spec.Theme.AmbientGlow == "" {
			spec.Theme.AmbientGlow = "rgba(56,189,248,0.22)"
		}
		if spec.Theme.BrowserChrome == nil {
			spec.Theme.BrowserChrome = &exportsv1.ReplayBrowserChrome{}
		}
		if spec.Theme.BrowserChrome.Variant == "" {
			spec.Theme.BrowserChrome = &exportsv1.ReplayBrowserChrome{Visible: true, Variant: "dark", Title: spec.Execution.GetWorkflowName(), ShowAddress: true, AccentColor: spec.Theme.AccentColor}
		}
		if spec.Theme.BrowserChrome.AccentColor == "" {
			spec.Theme.BrowserChrome.AccentColor = spec.Theme.AccentColor
		}
		if spec.Decor.ChromeTheme == "" {
			spec.Decor.ChromeTheme = "aurora"
		}
		if spec.Decor.BackgroundTheme == "" {
			spec.Decor.BackgroundTheme = "aurora"
		}
		if spec.Decor.Background == nil {
			spec.Decor.Background, _ = structpb.NewStruct(map[string]any{"type": "theme", "id": spec.Decor.BackgroundTheme})
		}
		if spec.Decor.CursorTheme == "" {
			spec.Decor.CursorTheme = "white"
		}
		if spec.Decor.CursorInitialPosition == "" {
			spec.Decor.CursorInitialPosition = "center"
		}
		if spec.Decor.CursorClickAnimation == "" {
			spec.Decor.CursorClickAnimation = "pulse"
		}
		if spec.Decor.CursorScale == 0 {
			spec.Decor.CursorScale = 1
		}
		if spec.Cursor.Style == "" {
			spec.Cursor.Style = "halo"
		}
		if spec.Cursor.AccentColor == "" {
			spec.Cursor.AccentColor = DefaultAccentColor
		}
		if spec.Cursor.InitialPosition == "" {
			spec.Cursor.InitialPosition = "center"
		}
		if spec.Cursor.ClickAnimation == "" {
			spec.Cursor.ClickAnimation = "pulse"
		}
		if spec.Cursor.Trail.FadeMs == 0 {
			spec.Cursor.Trail.FadeMs = 650
		}
		if spec.Cursor.Trail.Weight == 0 {
			spec.Cursor.Trail.Weight = 0.16
		}
		if spec.Cursor.Trail.Opacity == 0 {
			spec.Cursor.Trail.Opacity = 0.55
		}
		if spec.Cursor.ClickPulse.Radius == 0 {
			spec.Cursor.ClickPulse.Radius = 42
		}
		if spec.Cursor.ClickPulse.DurationMs == 0 {
			spec.Cursor.ClickPulse.DurationMs = 420
		}
		if spec.Cursor.ClickPulse.Opacity == 0 {
			spec.Cursor.ClickPulse.Opacity = 0.65
		}
		if !spec.Cursor.ClickPulse.Enabled && !strings.EqualFold(spec.Cursor.Style, "hidden") {
			spec.Cursor.ClickPulse.Enabled = true
		}
		if !spec.Cursor.Trail.Enabled && !strings.EqualFold(spec.Cursor.Style, "hidden") {
			spec.Cursor.Trail.Enabled = true
		}
	}
	spec.Cursor.Scale = ClampCursorScale(spec.Cursor.Scale)
	if spec.CursorMotion.SpeedProfile == "" {
		spec.CursorMotion.SpeedProfile = "easeInOut"
	}
	if spec.CursorMotion.PathStyle == "" {
		spec.CursorMotion.PathStyle = "linear"
	}
	if spec.CursorMotion.InitialPosition == "" {
		spec.CursorMotion.InitialPosition = spec.Cursor.InitialPosition
	}
	if spec.CursorMotion.ClickAnimation == "" {
		spec.CursorMotion.ClickAnimation = spec.Cursor.ClickAnimation
	}
	if spec.CursorMotion.CursorScale <= 0 {
		spec.CursorMotion.CursorScale = spec.Cursor.Scale
	}
	if baseline == nil {
		if spec.Presentation.Canvas.Width == 0 {
			spec.Presentation.Canvas.Width = 1920
		}
		if spec.Presentation.Canvas.Height == 0 {
			spec.Presentation.Canvas.Height = 1080
		}
		if spec.Presentation.Viewport.Width == 0 {
			spec.Presentation.Viewport.Width = spec.Presentation.Canvas.Width
		}
		if spec.Presentation.Viewport.Height == 0 {
			spec.Presentation.Viewport.Height = spec.Presentation.Canvas.Height
		}
		if spec.Presentation.BrowserFrame.Width == 0 {
			spec.Presentation.BrowserFrame = &exportsv1.ReplayFrameRect{Width: spec.Presentation.Canvas.Width, Height: spec.Presentation.Canvas.Height, Radius: 24}
		}
	}
	if spec.Presentation.DeviceScaleFactor == 0 {
		spec.Presentation.DeviceScaleFactor = 1
	}
	if spec.Presentation.BrowserFrame.Radius == 0 {
		spec.Presentation.BrowserFrame.Radius = 24
	}
	fallback := spec.Playback.FrameIntervalMs
	if fallback <= 0 {
		fallback = 40
	}
	var total, max, screenshots int32
	for _, frame := range spec.Frames {
		if frame == nil {
			continue
		}
		duration := frame.DurationMs
		if duration <= 0 {
			duration = frame.HoldMs + frame.GetEnter().GetDurationMs() + frame.GetExit().GetDurationMs()
		}
		if duration <= 0 {
			duration = fallback
		}
		total += duration
		if duration > max {
			max = duration
		}
		if strings.TrimSpace(frame.GetScreenshotAssetId()) != "" {
			screenshots++
		}
	}
	if spec.Summary.FrameCount <= 0 {
		spec.Summary.FrameCount = int32(len(spec.Frames))
	}
	if spec.Summary.TotalDurationMs <= 0 {
		spec.Summary.TotalDurationMs = total
	}
	if spec.Summary.MaxFrameDurationMs <= 0 {
		spec.Summary.MaxFrameDurationMs = max
	}
	if spec.Summary.ScreenshotCount <= 0 {
		if screenshots > 0 {
			spec.Summary.ScreenshotCount = screenshots
		} else if baseline != nil {
			spec.Summary.ScreenshotCount = baseline.GetSummary().GetScreenshotCount()
		}
	}
	if spec.Playback.FrameIntervalMs <= 0 {
		spec.Playback.FrameIntervalMs = fallback
	}
	if spec.Playback.DurationMs <= 0 {
		spec.Playback.DurationMs = spec.Summary.TotalDurationMs
	}
	if spec.Playback.TotalFrames <= 0 && spec.Summary.TotalDurationMs > 0 {
		spec.Playback.TotalFrames = int32(math.Ceil(float64(spec.Summary.TotalDurationMs) / float64(spec.Playback.FrameIntervalMs)))
	}
	if spec.Playback.TotalFrames <= 0 {
		spec.Playback.TotalFrames = spec.Summary.FrameCount
	}
	if spec.Playback.Fps <= 0 {
		spec.Playback.Fps = int32(math.Round(1000 / float64(spec.Playback.FrameIntervalMs)))
	}
	if spec.Playback.Fps <= 0 {
		spec.Playback.Fps = 25
	}
}

func cloneMessages[T proto.Message](messages []T) []T {
	if messages == nil {
		return nil
	}
	clones := make([]T, 0, len(messages))
	for _, message := range messages {
		if !message.ProtoReflect().IsValid() {
			var zero T
			clones = append(clones, zero)
			continue
		}
		clones = append(clones, proto.Clone(message).(T))
	}
	return clones
}
