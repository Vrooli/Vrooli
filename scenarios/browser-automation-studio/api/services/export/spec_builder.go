package export

import (
	"fmt"
	"math"
	"strings"

	"github.com/vrooli/browser-automation-studio/database"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
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

// BuildReplaySpecFromExecution creates the generated public contract for an
// execution preview directly from timeline facts.
func BuildReplaySpecFromExecution(execution *database.ExecutionIndex, workflow *database.WorkflowIndex, timeline *ExecutionTimeline) (*exportsv1.ReplaySpec, error) {
	return BuildReplaySpecFromTimeline(execution, workflow, timeline)
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
type frameDimensions struct{ Width, Height int }

func dimensionsForFrame(frame TimelineFrame) frameDimensions {
	if frame.Screenshot != nil {
		if frame.Screenshot.Width > 0 && frame.Screenshot.Height > 0 {
			return frameDimensions{Width: frame.Screenshot.Width, Height: frame.Screenshot.Height}
		}
	}
	return frameDimensions{Width: fallbackViewportWidth, Height: fallbackViewportHeight}
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
