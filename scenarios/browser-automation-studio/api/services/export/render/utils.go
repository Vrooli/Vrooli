package render

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"
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
		execID := strings.TrimSpace(spec.GetExecution().GetExecutionId())
		if len(execID) >= 8 {
			stem = fmt.Sprintf("browser-automation-replay-%s", execID[:8])
		}
	}
	return fmt.Sprintf("%s.%s", stem, extension)
}

const (
	renderTimeoutBufferMillis  = 120000
	perFrameRenderBudgetMillis = 220
)

// EstimateReplayRenderTimeout returns a conservative timeout budget for rendering a
// replay movie spec. The calculation ensures handlers can provision a long-lived
// context without hard-coding large constants.
func EstimateReplayRenderTimeout(spec *ReplayMovieSpec) time.Duration {
	timeoutMs := renderTimeoutBufferMillis
	if spec != nil {
		captureInterval := int(spec.GetPlayback().GetFrameIntervalMs())
		if captureInterval <= 0 {
			captureInterval = defaultCaptureInterval
		}

		totalDuration := int(spec.GetSummary().GetTotalDurationMs())
		if totalDuration <= 0 {
			totalDuration = int(spec.GetPlayback().GetDurationMs())
		}
		if totalDuration <= 0 && len(spec.GetFrames()) > 0 {
			sum := 0
			for _, frame := range spec.GetFrames() {
				duration := int(frame.GetDurationMs())
				if duration <= 0 {
					duration = int(frame.GetHoldMs() + frame.GetEnter().GetDurationMs() + frame.GetExit().GetDurationMs())
				}
				if duration <= 0 {
					duration = captureInterval
				}
				sum += duration
			}
			totalDuration = sum
		}

		frameBudget := int(spec.GetPlayback().GetTotalFrames())
		if frameBudget <= 0 && captureInterval > 0 && totalDuration > 0 {
			frameBudget = int(math.Ceil(float64(totalDuration) / float64(captureInterval)))
		}
		if frameBudget <= 0 {
			frameBudget = len(spec.GetFrames())
		}
		if frameBudget < 1 {
			frameBudget = 1
		}

		timeoutMs += frameBudget * perFrameRenderBudgetMillis
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
