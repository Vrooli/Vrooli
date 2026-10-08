package procmetrics

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

// XdotoolDetector checks for visible X11 windows using xdotool.
type XdotoolDetector struct {
	shell        ShellFunc
	logger       *slog.Logger
	warnOnce     sync.Once
	xdotoolAvail *bool // nil = not checked, true/false = result cached
	mu           sync.Mutex
}

// NewXdotoolDetector creates a new window detector backed by xdotool.
func NewXdotoolDetector(shell ShellFunc, logger *slog.Logger) *XdotoolDetector {
	return &XdotoolDetector{
		shell:  shell,
		logger: logger,
	}
}

// HasVisibleWindow returns true if the process has at least one visible window on the display.
// It first tries PID-specific search, then falls back to checking any visible window on the
// display. The fallback is needed because Electron apps fork renderer child processes that
// may own the actual X11 windows, making PID-based search miss them. Since each app runs on
// a dedicated virtual display (Xvfb), any visible window on that display belongs to our app.
func (d *XdotoolDetector) HasVisibleWindow(ctx context.Context, pid int, display string) (bool, error) {
	ids := d.findVisibleWindowIDs(ctx, pid, display)
	return len(ids) > 0, nil
}

// LargestVisibleWindow returns the geometry of the largest visible window on the display.
// Returns nil if no visible window exists or xdotool is not available.
func (d *XdotoolDetector) LargestVisibleWindow(ctx context.Context, pid int, display string) (*WindowGeometry, error) {
	ids := d.findVisibleWindowIDs(ctx, pid, display)
	if len(ids) == 0 {
		return nil, nil
	}

	env := []string{fmt.Sprintf("DISPLAY=%s", display)}
	var largest *WindowGeometry

	for _, id := range ids {
		stdout, err := d.shell(ctx, env, "xdotool", "getwindowgeometry", "--shell", id)
		if err != nil {
			continue
		}
		w, h := parseGeometryShell(string(stdout))
		if w <= 0 || h <= 0 {
			continue
		}
		area := w * h
		if largest == nil || area > largest.Width*largest.Height {
			largest = &WindowGeometry{Width: w, Height: h}
		}
	}

	return largest, nil
}

// findVisibleWindowIDs returns window IDs visible on the display.
// Tries PID-specific search first, falls back to any visible window on the display.
func (d *XdotoolDetector) findVisibleWindowIDs(ctx context.Context, pid int, display string) []string {
	if !d.isXdotoolAvailable(ctx) {
		return nil
	}

	env := []string{fmt.Sprintf("DISPLAY=%s", display)}

	// Try PID-specific search first (most precise).
	stdout, err := d.shell(ctx, env, "xdotool", "search", "--onlyvisible", "--pid", fmt.Sprintf("%d", pid))
	if err == nil {
		if ids := splitWindowIDs(string(stdout)); len(ids) > 0 {
			return ids
		}
	}

	// Fallback: any visible window on the display (handles Electron child processes).
	stdout, err = d.shell(ctx, env, "xdotool", "search", "--onlyvisible", "--name", "")
	if err != nil {
		return nil
	}
	return splitWindowIDs(string(stdout))
}

// splitWindowIDs splits xdotool search output into non-empty window ID strings.
func splitWindowIDs(output string) []string {
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if id := strings.TrimSpace(line); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// parseGeometryShell parses xdotool getwindowgeometry --shell output.
// Example output:
//
//	WINDOW=12345678
//	X=0
//	Y=0
//	WIDTH=1280
//	HEIGHT=720
func parseGeometryShell(output string) (width, height int) {
	for _, line := range strings.Split(output, "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			switch strings.TrimSpace(k) {
			case "WIDTH":
				width, _ = strconv.Atoi(strings.TrimSpace(v))
			case "HEIGHT":
				height, _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
	}
	return width, height
}

// isXdotoolAvailable checks (and caches) whether xdotool is installed.
func (d *XdotoolDetector) isXdotoolAvailable(ctx context.Context) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.xdotoolAvail != nil {
		return *d.xdotoolAvail
	}

	_, err := d.shell(ctx, nil, "which", "xdotool")
	avail := err == nil
	d.xdotoolAvail = &avail

	if !avail {
		d.warnOnce.Do(func() {
			d.logger.Warn("xdotool not installed — window detection disabled; startup timing will not be measured")
		})
	}

	return avail
}
