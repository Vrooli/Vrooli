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

const minimumUsableWindowDimension = 100

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
	for _, id := range ids {
		if _, ok := d.usableApplicationWindow(ctx, display, id); ok {
			return true, nil
		}
	}
	return false, nil
}

// IsAvailable reports whether the shared xdotool seam is usable on a display.
// Availability is cached so a journey and the process monitor make one
// consistent decision about the host prerequisite.
func (d *XdotoolDetector) IsAvailable(ctx context.Context) bool {
	return d.isXdotoolAvailable(ctx)
}

// VisibleWindowIDs returns the visible window IDs discovered through the
// shared xdotool seam. A zero PID intentionally uses the display-wide fallback
// for applications whose UI is owned by a child process.
func (d *XdotoolDetector) VisibleWindowIDs(ctx context.Context, pid int, display string) ([]string, error) {
	if !d.isXdotoolAvailable(ctx) {
		return nil, fmt.Errorf("xdotool is not available")
	}
	ids := d.findVisibleWindowIDs(ctx, pid, display)
	if len(ids) == 0 {
		return nil, fmt.Errorf("no visible window found")
	}
	return ids, nil
}

// ActivateWindow activates the largest visible application window.
func (d *XdotoolDetector) ActivateWindow(ctx context.Context, pid int, display string) error {
	id, err := d.primaryWindowID(ctx, pid, display)
	if err != nil {
		return err
	}
	return d.run(ctx, display, "windowactivate", id)
}

// MaximizeWindow makes the primary window fill the supplied display bounds.
func (d *XdotoolDetector) MaximizeWindow(ctx context.Context, pid int, display string, width, height int) error {
	id, err := d.primaryWindowID(ctx, pid, display)
	if err != nil {
		return err
	}
	if err := d.run(ctx, display, "windowactivate", id); err != nil {
		return err
	}
	if err := d.run(ctx, display, "windowmove", id, "0", "0"); err != nil {
		return err
	}
	return d.run(ctx, display, "windowsize", id, strconv.Itoa(width), strconv.Itoa(height))
}

// ResizeWindow resizes the primary visible window explicitly.
func (d *XdotoolDetector) ResizeWindow(ctx context.Context, pid int, display string, width, height int) error {
	id, err := d.primaryWindowID(ctx, pid, display)
	if err != nil {
		return err
	}
	return d.run(ctx, display, "windowsize", id, strconv.Itoa(width), strconv.Itoa(height))
}

// MoveWindow moves the primary visible window explicitly.
func (d *XdotoolDetector) MoveWindow(ctx context.Context, pid int, display string, x, y int) error {
	id, err := d.primaryWindowID(ctx, pid, display)
	if err != nil {
		return err
	}
	return d.run(ctx, display, "windowmove", id, strconv.Itoa(x), strconv.Itoa(y))
}

// Click moves the pointer and clicks the requested button at the coordinates.
func (d *XdotoolDetector) Click(ctx context.Context, display string, x, y, button int) error {
	if !d.isXdotoolAvailable(ctx) {
		return fmt.Errorf("xdotool is not available")
	}
	if button <= 0 {
		button = 1
	}
	if err := d.run(ctx, display, "mousemove", strconv.Itoa(x), strconv.Itoa(y)); err != nil {
		return err
	}
	return d.run(ctx, display, "click", strconv.Itoa(button))
}

// KeyPress sends one explicit X11 key name.
func (d *XdotoolDetector) KeyPress(ctx context.Context, display, key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("key is required")
	}
	return d.run(ctx, display, "key", key)
}

// Type enters literal text through the active X11 window. It is used by
// fixture journeys for deterministic semantic input, not as a substitute for
// application-level assertions.
func (d *XdotoolDetector) Type(ctx context.Context, display, value string) error {
	if value == "" {
		return fmt.Errorf("text is required")
	}
	if !d.isXdotoolAvailable(ctx) {
		return fmt.Errorf("xdotool is not available")
	}
	return d.run(ctx, display, "type", "--delay", "20", value)
}

// WindowGeometry returns the geometry of the largest visible window.
func (d *XdotoolDetector) WindowGeometry(ctx context.Context, pid int, display string) (*WindowGeometry, error) {
	id, err := d.primaryWindowID(ctx, pid, display)
	if err != nil {
		return nil, err
	}
	env := []string{fmt.Sprintf("DISPLAY=%s", display)}
	stdout, err := d.shell(ctx, env, "xdotool", "getwindowgeometry", "--shell", id)
	if err != nil {
		return nil, fmt.Errorf("get window geometry: %w", err)
	}
	x, y, w, h := parseGeometryShellFull(string(stdout))
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("window geometry is invalid")
	}
	return &WindowGeometry{X: x, Y: y, Width: w, Height: h}, nil
}

// LargestVisibleWindow returns the geometry of the largest visible window on the display.
// Returns nil if no visible window exists or xdotool is not available.
func (d *XdotoolDetector) LargestVisibleWindow(ctx context.Context, pid int, display string) (*WindowGeometry, error) {
	ids := d.findVisibleWindowIDs(ctx, pid, display)
	if len(ids) == 0 {
		return nil, nil
	}

	var largest *WindowGeometry

	for _, id := range ids {
		geometry, ok := d.usableApplicationWindow(ctx, display, id)
		if !ok {
			continue
		}
		area := geometry.Width * geometry.Height
		if largest == nil || area > largest.Width*largest.Height {
			largest = geometry
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
	// xdotool treats an empty --name pattern inconsistently across versions;
	// use an explicit match-all expression. Geometry and identity checks below
	// still exclude desktop surfaces.
	stdout, err = d.shell(ctx, env, "xdotool", "search", "--onlyvisible", "--name", ".*")
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
	_, _, width, height = parseGeometryShellFull(output)
	return width, height
}

func parseGeometryShellFull(output string) (x, y, width, height int) {
	for _, line := range strings.Split(output, "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			switch strings.TrimSpace(k) {
			case "X":
				x, _ = strconv.Atoi(strings.TrimSpace(v))
			case "Y":
				y, _ = strconv.Atoi(strings.TrimSpace(v))
			case "WIDTH":
				width, _ = strconv.Atoi(strings.TrimSpace(v))
			case "HEIGHT":
				height, _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
	}
	return x, y, width, height
}

func (d *XdotoolDetector) primaryWindowID(ctx context.Context, pid int, display string) (string, error) {
	ids, err := d.VisibleWindowIDs(ctx, pid, display)
	if err != nil {
		return "", err
	}

	primaryID := ""
	primaryArea := 0
	for _, id := range ids {
		geometry, ok := d.usableApplicationWindow(ctx, display, id)
		if !ok {
			continue
		}
		if area := geometry.Width * geometry.Height; primaryID == "" || area > primaryArea {
			primaryID = id
			primaryArea = area
		}
	}
	if primaryID == "" {
		return "", fmt.Errorf("no usable visible window found")
	}
	return primaryID, nil
}

func (d *XdotoolDetector) usableApplicationWindow(ctx context.Context, display, id string) (*WindowGeometry, bool) {
	env := []string{fmt.Sprintf("DISPLAY=%s", display)}
	stdout, err := d.shell(ctx, env, "xdotool", "getwindowgeometry", "--shell", id)
	if err != nil {
		return nil, false
	}
	x, y, width, height := parseGeometryShellFull(string(stdout))
	if !usableWindowDimensions(width, height) || !d.windowHasUsableApplicationIdentity(ctx, display, id) {
		return nil, false
	}
	return &WindowGeometry{X: x, Y: y, Width: width, Height: height}, true
}

func (d *XdotoolDetector) windowHasUsableApplicationIdentity(ctx context.Context, display, id string) bool {
	env := []string{fmt.Sprintf("DISPLAY=%s", display)}
	nameOutput, nameErr := d.shell(ctx, env, "xdotool", "getwindowname", id)
	if nameErr != nil {
		return false
	}

	name := strings.TrimSpace(string(nameOutput))
	if name == "" {
		return false
	}
	// Openbox/Xvfb desktop surfaces can be reported as visible windows. They
	// are not evidence of an application launch, even when they cover the
	// entire display.
	if strings.EqualFold(name, "desktop") || strings.EqualFold(name, "openbox") {
		return false
	}
	return true
}

func usableWindowDimensions(width, height int) bool {
	return width >= minimumUsableWindowDimension && height >= minimumUsableWindowDimension
}

func (d *XdotoolDetector) run(ctx context.Context, display, verb string, args ...string) error {
	env := []string{fmt.Sprintf("DISPLAY=%s", display)}
	if _, err := d.shell(ctx, env, "xdotool", append([]string{verb}, args...)...); err != nil {
		return fmt.Errorf("xdotool %s: %w", verb, err)
	}
	return nil
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
