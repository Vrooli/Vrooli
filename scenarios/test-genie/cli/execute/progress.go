package execute

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"test-genie/cli/internal/phases"
)

// Spinner characters for animated progress indicator (Braille Unicode characters).
// These create a smooth spinning animation when cycled rapidly in a terminal.
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// isTTYFunc is the function used to check if a writer is a TTY.
// This is a variable to allow testing - tests can replace this with a mock.
//
// Design Decision: Testing Seam for TTY Detection
// ------------------------------------------------
// TTY detection is inherently environment-dependent and difficult to test directly.
// By making this a package-level variable, tests can inject their own implementation
// to verify both TTY and non-TTY code paths without requiring actual terminal access.
var isTTYFunc = defaultIsTTY

// defaultIsTTY checks if the given writer is connected to a terminal.
// Returns true only if w is an *os.File and that file descriptor is a terminal.
func defaultIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// ProgressConfig holds configuration for progress display.
// This allows callers to customize progress behavior for different environments.
type ProgressConfig struct {
	// Writer is where progress output is written (typically os.Stderr).
	Writer io.Writer

	// PhaseList is the ordered list of phases to be executed.
	PhaseList []string

	// Targets maps phase names to their expected durations for time estimation.
	Targets map[string]time.Duration

	// Timeouts maps phase names to their execution timeout budgets.
	Timeouts map[string]time.Duration

	// ForceTTY overrides TTY detection when set to a non-nil value.
	// Use this for testing or when you need to force a specific mode.
	// true = force TTY mode (animated spinner)
	// false = force non-TTY mode (phase-change logging)
	ForceTTY *bool
}

// StartProgress begins a progress display that shows execution status.
// Returns a stop function that must be called when execution completes.
//
// # Display Modes
//
// This function operates in two distinct modes based on whether the output
// is connected to an interactive terminal (TTY):
//
// ## TTY Mode (Interactive Terminal)
//
// When output goes to a real terminal, we display an animated spinner with
// a progress bar that updates every 200ms. The carriage return (\r) allows
// us to overwrite the same line, creating smooth animation:
//
//	⠧ Running tests ██░░░░░░░░░░░░░░░░░░ [1/10] structure • elapsed 7s • remaining ~1h47m23s
//
// This provides excellent UX for humans watching the terminal.
//
// ## Non-TTY Mode (Piped/Captured Output)
//
// When output is piped, redirected, or captured (e.g., by CI systems, log
// aggregators, or AI agents running commands), the carriage return trick
// fails - each "update" becomes a separate line, creating hundreds of
// redundant lines like:
//
//	⠋ Running tests ██░░░░░░░░░░░░░░░░░░ [1/10] structure • elapsed 0s
//	⠙ Running tests ██░░░░░░░░░░░░░░░░░░ [1/10] structure • elapsed 0s
//	⠹ Running tests ██░░░░░░░░░░░░░░░░░░ [1/10] structure • elapsed 0s
//	... (hundreds more identical lines)
//
// Instead, we print simple one-line status updates only when the estimated
// phase changes:
//
//	[1/10] Running structure phase... (estimated ~2m0s)
//	[2/10] Running standards phase... (estimated ~1m0s)
//
// This gives CI/agents useful progress information without output spam.
//
// # Why This Matters
//
// AI agents (like Claude Code) capture command output as text. When they run
// test suites, they see every 200ms spinner update as a new line, which:
// - Wastes context window tokens on redundant information
// - Makes logs harder to parse and understand
// - Provides no additional value (the animation is meaningless in text)
//
// The phase-change mode solves this by providing meaningful progress updates
// only when something actually changes.
func StartProgress(w io.Writer, phaseList []string, targets map[string]time.Duration, timeouts map[string]time.Duration) func() {
	return StartProgressWithConfig(ProgressConfig{
		Writer:    w,
		PhaseList: phaseList,
		Targets:   targets,
		Timeouts:  timeouts,
	})
}

// StartProgressWithConfig begins progress display with full configuration control.
// This is the primary implementation that supports all options including ForceTTY.
func StartProgressWithConfig(cfg ProgressConfig) func() {
	if len(cfg.PhaseList) == 0 {
		cfg.PhaseList = []string{"structure", "unit", "integration"}
	}

	// Determine display mode: TTY (animated) vs non-TTY (phase-change logging)
	isTTY := false
	if cfg.ForceTTY != nil {
		isTTY = *cfg.ForceTTY
	} else {
		isTTY = isTTYFunc(cfg.Writer)
	}

	// Calculate total estimated time by summing all phase targets
	var totalEstimate time.Duration
	for _, p := range cfg.PhaseList {
		if d, ok := cfg.Targets[phases.NormalizeName(p)]; ok && d > 0 {
			totalEstimate += d
		}
	}

	start := time.Now()
	done := make(chan struct{})

	if isTTY {
		// TTY mode: animated spinner with rapid updates
		return startTTYProgress(cfg.Writer, cfg.PhaseList, cfg.Targets, totalEstimate, start, done)
	}

	// Non-TTY mode: simple phase-change logging
	return startNonTTYProgress(cfg.Writer, cfg.PhaseList, cfg.Targets, cfg.Timeouts, totalEstimate, start, done)
}

// startTTYProgress runs the animated spinner for interactive terminals.
// Updates every 200ms with spinner animation, progress bar, and timing info.
func startTTYProgress(w io.Writer, phaseList []string, targets map[string]time.Duration, totalEstimate time.Duration, start time.Time, done chan struct{}) func() {
	ticker := time.NewTicker(200 * time.Millisecond)

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

				// Build status line with carriage return to overwrite previous
				statusLine := fmt.Sprintf("\r%s Running tests %s [%d/%d] %-12s • %s",
					spinner,
					progressBar,
					phaseIdx,
					len(phaseList),
					currentPhase,
					timing,
				)

				// Clear line and print status (carriage return overwrites in-place)
				fmt.Fprintf(w, "\r%s\r%s", strings.Repeat(" ", 100), statusLine)
			}
		}
	}()

	return func() {
		close(done)
		// Clear the progress line completely
		fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", 100))
	}
}

// startNonTTYProgress runs simple phase-change logging for non-interactive output.
// Only prints a new line when the estimated current phase changes, avoiding
// the output spam that occurs when carriage returns don't work.
func startNonTTYProgress(w io.Writer, phaseList []string, targets, timeouts map[string]time.Duration, totalEstimate time.Duration, start time.Time, done chan struct{}) func() {
	// Check less frequently since we only care about phase changes
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		lastPhase := ""

		// Print initial phase immediately
		if len(phaseList) > 0 {
			firstPhase := phaseList[0]
			fmt.Fprintf(w, "[1/%d] Running %s phase... (%s)\n",
				len(phaseList), firstPhase, phaseTimingSummary(firstPhase, targets, timeouts))
			lastPhase = firstPhase
		}

		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				elapsed := time.Since(start).Truncate(time.Second)
				currentPhase := estimateCurrentPhase(phaseList, targets, elapsed)

				// Only print when the phase changes
				if currentPhase != lastPhase {
					phaseIdx := findPhaseIndex(phaseList, currentPhase) + 1
					fmt.Fprintf(w, "[%d/%d] Running %s phase... (%s)\n",
						phaseIdx, len(phaseList), currentPhase, phaseTimingSummary(currentPhase, targets, timeouts))
					lastPhase = currentPhase
				}
			}
		}
	}()

	return func() {
		close(done)
		elapsed := time.Since(start).Truncate(time.Second)
		fmt.Fprintf(w, "Progress tracking stopped after %s\n", elapsed)
	}
}

// estimateCurrentPhase guesses which phase is running based on elapsed time.
// This is an estimate because actual phase completion depends on test execution,
// not wall clock time. The estimate helps users understand progress even when
// we don't have real-time phase completion events.
func estimateCurrentPhase(phaseList []string, targets map[string]time.Duration, elapsed time.Duration) string {
	var accumulated time.Duration
	for _, phase := range phaseList {
		target, ok := targets[phases.NormalizeName(phase)]
		if !ok {
			target = 60 * time.Second // Default estimate if not specified
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

// findPhaseIndex returns the index of a phase in the list.
// Returns 0 if the phase is not found.
func findPhaseIndex(phaseList []string, phase string) int {
	for i, p := range phaseList {
		if p == phase {
			return i
		}
	}
	return 0
}

// buildProgressBar creates a visual progress bar using Unicode block characters.
// Uses █ (full block) for completed portions and ░ (light shade) for remaining.
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

func phaseTimingSummary(phase string, targets, timeouts map[string]time.Duration) string {
	key := phases.NormalizeName(phase)
	target := targets[key]
	timeout := timeouts[key]

	switch {
	case target > 0 && timeout > 0:
		return fmt.Sprintf("estimate ~%s, timeout %s", target.Truncate(time.Second), timeout.Truncate(time.Second))
	case target > 0:
		return fmt.Sprintf("estimate ~%s", target.Truncate(time.Second))
	case timeout > 0:
		return fmt.Sprintf("timeout %s", timeout.Truncate(time.Second))
	default:
		return "timing unavailable"
	}
}
