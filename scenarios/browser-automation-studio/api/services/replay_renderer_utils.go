package services

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// sanitizeFilename removes path separators and null bytes from a filename.
func sanitizeFilename(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "browser-automation-replay"
	}
	trimmed = strings.ReplaceAll(trimmed, string(os.PathSeparator), "-")
	trimmed = strings.ReplaceAll(trimmed, "\x00", "")
	return trimmed
}

// defaultFilename generates a default filename for a replay export based on the execution ID.
func defaultFilename(spec *ReplayMovieSpec, extension string) string {
	stem := "browser-automation-replay"
	if spec != nil {
		execID := spec.Execution.ExecutionID
		if execID != uuid.Nil {
			stem = fmt.Sprintf("browser-automation-replay-%s", execID.String()[:8])
		}
	}
	return fmt.Sprintf("%s.%s", stem, extension)
}

// EstimateReplayRenderTimeout returns a conservative timeout budget for rendering a
// replay movie spec. The calculation is aligned with the Browserless capture
// budgeting inside renderWithBrowserless, ensuring handlers can provision a
// long-lived context without hard-coding large constants.
func EstimateReplayRenderTimeout(spec *ReplayMovieSpec) time.Duration {
	timeoutMs := browserlessTimeoutBufferMillis
	if spec != nil {
		captureInterval := spec.Playback.FrameIntervalMs
		if captureInterval <= 0 {
			captureInterval = defaultCaptureInterval
		}

		totalDuration := spec.Summary.TotalDurationMs
		if totalDuration <= 0 {
			totalDuration = spec.Playback.DurationMs
		}
		if totalDuration <= 0 && len(spec.Frames) > 0 {
			sum := 0
			for _, frame := range spec.Frames {
				duration := frame.DurationMs
				if duration <= 0 {
					duration = frame.HoldMs + frame.Enter.DurationMs + frame.Exit.DurationMs
				}
				if duration <= 0 {
					duration = captureInterval
				}
				sum += duration
			}
			totalDuration = sum
		}

		frameBudget := spec.Playback.TotalFrames
		if frameBudget <= 0 && captureInterval > 0 && totalDuration > 0 {
			frameBudget = int(math.Ceil(float64(totalDuration) / float64(captureInterval)))
		}
		if frameBudget <= 0 {
			frameBudget = len(spec.Frames)
		}
		if frameBudget < 1 {
			frameBudget = 1
		}

		timeoutMs += frameBudget * perFrameBrowserlessBudgetMillis
		// Provide additional breathing room for ffmpeg assembly and other overheads.
		timeoutMs += 60000
	}

	duration := time.Duration(timeoutMs) * time.Millisecond
	minimum := 3 * time.Minute
	maximum := 15 * time.Minute
	if duration < minimum {
		duration = minimum
	}
	if duration > maximum {
		duration = maximum
	}
	return duration
}
