package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrMovieSpecUnavailable is returned when no movie spec is available for export.
var ErrMovieSpecUnavailable = errors.New("movie spec unavailable")

// BuildSpec constructs a validated ReplayMovieSpec for export by merging client-provided
// and server-generated specs, validating execution ID matching, and filling in defaults.
func BuildSpec(baseline, incoming *ReplayMovieSpec, executionID uuid.UUID) (*ReplayMovieSpec, error) {
	if incoming == nil && baseline == nil {
		return nil, ErrMovieSpecUnavailable
	}

	var (
		spec *ReplayMovieSpec
		err  error
	)

	// Use client-provided spec if available, otherwise use server baseline
	if incoming != nil {
		spec, err = Clone(incoming)
		if err != nil {
			return nil, fmt.Errorf("invalid movie spec: %w", err)
		}
	} else {
		spec, err = Clone(baseline)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare movie spec: %w", err)
		}
	}

	// Harmonize and validate the spec against baseline
	if err := Harmonize(spec, baseline, executionID); err != nil {
		return nil, err
	}
	return spec, nil
}

// Clone creates a deep copy of a ReplayMovieSpec via JSON marshaling.
func Clone(spec *ReplayMovieSpec) (*ReplayMovieSpec, error) {
	if spec == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var cloned ReplayMovieSpec
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}

// Harmonize validates and fills in missing fields in spec using values from baseline.
// It ensures the execution ID matches and applies sensible defaults for all required fields.
func Harmonize(spec, baseline *ReplayMovieSpec, executionID uuid.UUID) error {
	if spec == nil {
		return ErrMovieSpecUnavailable
	}

	// Validate execution ID
	if spec.Execution.ExecutionID == uuid.Nil {
		spec.Execution.ExecutionID = executionID
	} else if spec.Execution.ExecutionID != executionID {
		return fmt.Errorf("movie spec execution_id mismatch")
	}

	// Fill execution metadata from baseline
	if baseline != nil {
		if baseline.Execution.WorkflowID != uuid.Nil {
			if spec.Execution.WorkflowID == uuid.Nil {
				spec.Execution.WorkflowID = baseline.Execution.WorkflowID
			} else if spec.Execution.WorkflowID != baseline.Execution.WorkflowID {
				return fmt.Errorf("movie spec workflow_id mismatch")
			}
		}
		if spec.Execution.WorkflowName == "" {
			spec.Execution.WorkflowName = baseline.Execution.WorkflowName
		}
		if spec.Execution.Status == "" {
			spec.Execution.Status = baseline.Execution.Status
		}
		if spec.Execution.Progress == 0 {
			spec.Execution.Progress = baseline.Execution.Progress
		}
		if spec.Execution.StartedAt.IsZero() && !baseline.Execution.StartedAt.IsZero() {
			spec.Execution.StartedAt = baseline.Execution.StartedAt
		}
		if spec.Execution.CompletedAt == nil && baseline.Execution.CompletedAt != nil {
			t := *baseline.Execution.CompletedAt
			spec.Execution.CompletedAt = &t
		}
	}

	// Ensure version and generation timestamp
	if spec.Version == "" && baseline != nil {
		spec.Version = baseline.Version
	}
	if spec.Version == "" {
		spec.Version = "2025-11-07"
	}
	if spec.GeneratedAt.IsZero() {
		if baseline != nil && !baseline.GeneratedAt.IsZero() {
			spec.GeneratedAt = baseline.GeneratedAt
		} else {
			spec.GeneratedAt = time.Now().UTC()
		}
	}

	// Ensure frames are present
	if len(spec.Frames) == 0 {
		if baseline != nil && len(baseline.Frames) > 0 {
			spec.Frames = baseline.Frames
		} else {
			return fmt.Errorf("movie spec missing frames")
		}
	}

	// Fill in nested structures
	ensureTheme(spec, baseline)
	ensureDecor(spec, baseline)
	ensureCursor(spec, baseline)
	ensurePresentation(spec, baseline)
	ensureSummaryAndPlayback(spec, baseline)

	// Copy assets if missing
	if len(spec.Assets) == 0 && baseline != nil {
		spec.Assets = baseline.Assets
	}

	// Sync execution total duration with summary
	if spec.Execution.TotalDuration <= 0 {
		spec.Execution.TotalDuration = spec.Summary.TotalDurationMs
	}

	return nil
}

// ensureTheme fills in missing theme fields from baseline or applies defaults.
func ensureTheme(spec, baseline *ReplayMovieSpec) {
	if baseline != nil {
		if len(spec.Theme.BackgroundGradient) == 0 {
			spec.Theme.BackgroundGradient = baseline.Theme.BackgroundGradient
		}
		if spec.Theme.BackgroundPattern == "" {
			spec.Theme.BackgroundPattern = baseline.Theme.BackgroundPattern
		}
		if spec.Theme.AccentColor == "" {
			spec.Theme.AccentColor = baseline.Theme.AccentColor
		}
		if spec.Theme.SurfaceColor == "" {
			spec.Theme.SurfaceColor = baseline.Theme.SurfaceColor
		}
		if spec.Theme.AmbientGlow == "" {
			spec.Theme.AmbientGlow = baseline.Theme.AmbientGlow
		}
		if spec.Theme.BrowserChrome.Variant == "" && baseline.Theme.BrowserChrome.Variant != "" {
			spec.Theme.BrowserChrome = baseline.Theme.BrowserChrome
		} else if spec.Theme.BrowserChrome.AccentColor == "" {
			spec.Theme.BrowserChrome.AccentColor = spec.Theme.AccentColor
		}
		return
	}

	// Apply defaults when no baseline available
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
	if spec.Theme.BrowserChrome.Variant == "" {
		spec.Theme.BrowserChrome = ExportBrowserChrome{
			Visible:     true,
			Variant:     "dark",
			Title:       spec.Execution.WorkflowName,
			ShowAddress: true,
			AccentColor: spec.Theme.AccentColor,
		}
	} else if spec.Theme.BrowserChrome.AccentColor == "" {
		spec.Theme.BrowserChrome.AccentColor = spec.Theme.AccentColor
	}
}

// ensureDecor fills in missing decor preset names from baseline or applies defaults.
func ensureDecor(spec, baseline *ReplayMovieSpec) {
	if baseline != nil {
		if spec.Decor.ChromeTheme == "" {
			spec.Decor.ChromeTheme = baseline.Decor.ChromeTheme
		}
		if spec.Decor.BackgroundTheme == "" {
			spec.Decor.BackgroundTheme = baseline.Decor.BackgroundTheme
		}
		if spec.Decor.CursorTheme == "" {
			spec.Decor.CursorTheme = baseline.Decor.CursorTheme
		}
		if spec.Decor.CursorInitial == "" {
			spec.Decor.CursorInitial = baseline.Decor.CursorInitial
		}
		if spec.Decor.CursorClickAnimation == "" {
			spec.Decor.CursorClickAnimation = baseline.Decor.CursorClickAnimation
		}
		if spec.Decor.CursorScale == 0 {
			spec.Decor.CursorScale = baseline.Decor.CursorScale
		}
		return
	}

	// Apply defaults when no baseline available
	if spec.Decor.ChromeTheme == "" {
		spec.Decor.ChromeTheme = "aurora"
	}
	if spec.Decor.BackgroundTheme == "" {
		spec.Decor.BackgroundTheme = "aurora"
	}
	if spec.Decor.CursorTheme == "" {
		spec.Decor.CursorTheme = "white"
	}
	if spec.Decor.CursorInitial == "" {
		spec.Decor.CursorInitial = "center"
	}
	if spec.Decor.CursorClickAnimation == "" {
		spec.Decor.CursorClickAnimation = "pulse"
	}
	if spec.Decor.CursorScale == 0 {
		spec.Decor.CursorScale = 1
	}
}

// ensureCursor fills in missing cursor configuration from baseline or applies defaults.
func ensureCursor(spec, baseline *ReplayMovieSpec) {
	if baseline != nil {
		if spec.Cursor.Style == "" {
			spec.Cursor = baseline.Cursor
		} else {
			// Merge individual fields when style is set
			if spec.Cursor.AccentColor == "" {
				spec.Cursor.AccentColor = baseline.Cursor.AccentColor
			}
			if spec.Cursor.InitialPos == "" {
				spec.Cursor.InitialPos = baseline.Cursor.InitialPos
			}
			if spec.Cursor.ClickAnim == "" {
				spec.Cursor.ClickAnim = baseline.Cursor.ClickAnim
			}
			if spec.Cursor.Trail.FadeMs == 0 {
				spec.Cursor.Trail.FadeMs = baseline.Cursor.Trail.FadeMs
			}
			if spec.Cursor.Trail.Weight == 0 {
				spec.Cursor.Trail.Weight = baseline.Cursor.Trail.Weight
			}
			if spec.Cursor.Trail.Opacity == 0 {
				spec.Cursor.Trail.Opacity = baseline.Cursor.Trail.Opacity
			}
			if spec.Cursor.ClickPulse.Radius == 0 {
				spec.Cursor.ClickPulse.Radius = baseline.Cursor.ClickPulse.Radius
			}
			if spec.Cursor.ClickPulse.DurationMs == 0 {
				spec.Cursor.ClickPulse.DurationMs = baseline.Cursor.ClickPulse.DurationMs
			}
			if spec.Cursor.ClickPulse.Opacity == 0 {
				spec.Cursor.ClickPulse.Opacity = baseline.Cursor.ClickPulse.Opacity
			}
			if !spec.Cursor.ClickPulse.Enabled && baseline.Cursor.ClickPulse.Enabled {
				spec.Cursor.ClickPulse.Enabled = true
			}
			if !spec.Cursor.Trail.Enabled && baseline.Cursor.Trail.Enabled {
				spec.Cursor.Trail.Enabled = true
			}
		}
	} else {
		// Apply defaults when no baseline available
		if spec.Cursor.Style == "" {
			spec.Cursor.Style = "halo"
		}
		if spec.Cursor.AccentColor == "" {
			spec.Cursor.AccentColor = DefaultAccentColor
		}
		if spec.Cursor.InitialPos == "" {
			spec.Cursor.InitialPos = "center"
		}
		if spec.Cursor.ClickAnim == "" {
			spec.Cursor.ClickAnim = "pulse"
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

	// Clamp cursor scale
	spec.Cursor.Scale = ClampCursorScale(spec.Cursor.Scale)

	// Fill cursor motion fields
	if spec.CursorMotion.SpeedProfile == "" && baseline != nil {
		spec.CursorMotion.SpeedProfile = baseline.CursorMotion.SpeedProfile
	}
	if spec.CursorMotion.SpeedProfile == "" {
		spec.CursorMotion.SpeedProfile = "easeInOut"
	}
	if spec.CursorMotion.PathStyle == "" && baseline != nil {
		spec.CursorMotion.PathStyle = baseline.CursorMotion.PathStyle
	}
	if spec.CursorMotion.PathStyle == "" {
		spec.CursorMotion.PathStyle = "linear"
	}
	if spec.CursorMotion.InitialPosition == "" {
		spec.CursorMotion.InitialPosition = spec.Cursor.InitialPos
	}
	if spec.CursorMotion.ClickAnimation == "" {
		spec.CursorMotion.ClickAnimation = spec.Cursor.ClickAnim
	}
	if spec.CursorMotion.CursorScale <= 0 {
		spec.CursorMotion.CursorScale = spec.Cursor.Scale
	}
}

// ensurePresentation fills in canvas and viewport dimensions from baseline or applies defaults.
func ensurePresentation(spec, baseline *ReplayMovieSpec) {
	if baseline != nil {
		if spec.Presentation.Canvas.Width == 0 {
			spec.Presentation.Canvas = baseline.Presentation.Canvas
		}
		if spec.Presentation.Viewport.Width == 0 {
			spec.Presentation.Viewport = baseline.Presentation.Viewport
		}
		if spec.Presentation.BrowserFrame.Width == 0 {
			spec.Presentation.BrowserFrame = baseline.Presentation.BrowserFrame
		}
		if spec.Presentation.DeviceScaleFactor == 0 {
			spec.Presentation.DeviceScaleFactor = baseline.Presentation.DeviceScaleFactor
		}
	} else {
		// Apply defaults when no baseline available
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
			spec.Presentation.BrowserFrame = ExportFrameRect{
				X:      0,
				Y:      0,
				Width:  spec.Presentation.Canvas.Width,
				Height: spec.Presentation.Canvas.Height,
				Radius: 24,
			}
		}
	}

	// Ensure minimum defaults
	if spec.Presentation.DeviceScaleFactor == 0 {
		spec.Presentation.DeviceScaleFactor = 1
	}
	if spec.Presentation.BrowserFrame.Radius == 0 {
		spec.Presentation.BrowserFrame.Radius = 24
	}
}

// ensureSummaryAndPlayback computes or fills in summary statistics and playback settings.
func ensureSummaryAndPlayback(spec, baseline *ReplayMovieSpec) {
	// Determine frame interval fallback
	fallbackInterval := spec.Playback.FrameIntervalMs
	if fallbackInterval <= 0 && baseline != nil && baseline.Playback.FrameIntervalMs > 0 {
		fallbackInterval = baseline.Playback.FrameIntervalMs
	}
	if fallbackInterval <= 0 {
		fallbackInterval = 40
	}

	// Compute frame durations
	totalDuration, maxDuration := computeFrameDurations(spec.Frames, fallbackInterval)

	// Fill summary fields
	if spec.Summary.FrameCount <= 0 {
		spec.Summary.FrameCount = len(spec.Frames)
	}
	if spec.Summary.TotalDurationMs <= 0 {
		spec.Summary.TotalDurationMs = totalDuration
	}
	if spec.Summary.MaxFrameDurationMs <= 0 {
		spec.Summary.MaxFrameDurationMs = maxDuration
	}
	if spec.Summary.ScreenshotCount <= 0 {
		count := countFrameScreenshots(spec.Frames)
		if count > 0 {
			spec.Summary.ScreenshotCount = count
		} else if baseline != nil {
			spec.Summary.ScreenshotCount = baseline.Summary.ScreenshotCount
		}
	}

	// Fill playback fields
	if spec.Playback.FrameIntervalMs <= 0 {
		spec.Playback.FrameIntervalMs = fallbackInterval
	}
	if spec.Playback.DurationMs <= 0 {
		spec.Playback.DurationMs = spec.Summary.TotalDurationMs
	}
	if spec.Playback.TotalFrames <= 0 && spec.Playback.FrameIntervalMs > 0 && spec.Summary.TotalDurationMs > 0 {
		spec.Playback.TotalFrames = int(math.Ceil(float64(spec.Summary.TotalDurationMs) / float64(spec.Playback.FrameIntervalMs)))
	}
	if spec.Playback.TotalFrames <= 0 {
		spec.Playback.TotalFrames = spec.Summary.FrameCount
	}
	if spec.Playback.FPS <= 0 && spec.Playback.FrameIntervalMs > 0 {
		spec.Playback.FPS = int(math.Round(1000.0 / float64(spec.Playback.FrameIntervalMs)))
	}
	if spec.Playback.FPS <= 0 && baseline != nil && baseline.Playback.FPS > 0 {
		spec.Playback.FPS = baseline.Playback.FPS
	}
	if spec.Playback.FPS <= 0 {
		spec.Playback.FPS = 25
	}
}

// computeFrameDurations sums frame durations and finds the maximum duration.
func computeFrameDurations(frames []ExportFrame, fallback int) (total int, max int) {
	if fallback <= 0 {
		fallback = 40
	}
	for _, frame := range frames {
		duration := frame.DurationMs
		if duration <= 0 {
			duration = frame.HoldMs + frame.Enter.DurationMs + frame.Exit.DurationMs
		}
		if duration <= 0 {
			duration = fallback
		}
		total += duration
		if duration > max {
			max = duration
		}
	}
	return total, max
}

// countFrameScreenshots counts the number of frames with non-empty screenshot asset IDs.
func countFrameScreenshots(frames []ExportFrame) int {
	count := 0
	for _, frame := range frames {
		if strings.TrimSpace(frame.ScreenshotAssetID) != "" {
			count++
		}
	}
	return count
}
