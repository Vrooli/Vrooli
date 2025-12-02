package execute

import (
	"fmt"
	"io"
	"strings"
	"time"

	"test-genie/cli/internal/phases"
)

// Spinner characters for animated progress indicator
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// StartProgress begins a progress ticker that displays execution status.
// Returns a stop function to call when execution completes.
func StartProgress(w io.Writer, phaseList []string, targets map[string]time.Duration) func() {
	if len(phaseList) == 0 {
		phaseList = []string{"structure", "unit", "integration"}
	}
	start := time.Now()
	ticker := time.NewTicker(200 * time.Millisecond) // Faster updates for smoother animation
	done := make(chan struct{})

	// Calculate total estimated time
	var totalEstimate time.Duration
	for _, p := range phaseList {
		if d, ok := targets[phases.NormalizeName(p)]; ok && d > 0 {
			totalEstimate += d
		}
	}

	go func() {
		tick := 0
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				tick++
				spinner := spinnerChars[tick%len(spinnerChars)]
				elapsed := time.Since(start).Truncate(time.Second)

				// Calculate which phase we're likely in based on elapsed time
				currentPhase := estimateCurrentPhase(phaseList, targets, elapsed)
				phaseIdx := findPhaseIndex(phaseList, currentPhase) + 1

				// Build progress bar
				progressBar := buildProgressBar(phaseIdx, len(phaseList), 20)

				// Format timing info
				var timing string
				if totalEstimate > 0 {
					remaining := totalEstimate - elapsed
					if remaining < 0 {
						remaining = 0
					}
					timing = fmt.Sprintf("elapsed %s • remaining ~%s",
						elapsed.Truncate(time.Second),
						remaining.Truncate(time.Second))
				} else {
					timing = fmt.Sprintf("elapsed %s", elapsed.Truncate(time.Second))
				}

				// Build status line
				statusLine := fmt.Sprintf("\r%s Running tests %s [%d/%d] %-12s • %s",
					spinner,
					progressBar,
					phaseIdx,
					len(phaseList),
					currentPhase,
					timing,
				)

				// Clear line and print status
				fmt.Fprintf(w, "\r%s\r%s", strings.Repeat(" ", 100), statusLine)
			}
		}
	}()
	return func() {
		close(done)
		// Clear the progress line
		fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", 100))
	}
}

// estimateCurrentPhase guesses which phase is running based on elapsed time
func estimateCurrentPhase(phaseList []string, targets map[string]time.Duration, elapsed time.Duration) string {
	var accumulated time.Duration
	for _, phase := range phaseList {
		target, ok := targets[phases.NormalizeName(phase)]
		if !ok {
			target = 60 * time.Second // Default estimate
		}
		accumulated += target
		if elapsed < accumulated {
			return phase
		}
	}
	// If we've exceeded all estimates, show the last phase
	if len(phaseList) > 0 {
		return phaseList[len(phaseList)-1]
	}
	return "running"
}

// findPhaseIndex returns the index of a phase in the list
func findPhaseIndex(phaseList []string, phase string) int {
	for i, p := range phaseList {
		if p == phase {
			return i
		}
	}
	return 0
}

// buildProgressBar creates a visual progress bar
func buildProgressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := (current * width) / total
	if filled > width {
		filled = width
	}
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}
