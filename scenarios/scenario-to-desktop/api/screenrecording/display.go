package screenrecording

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DisplayManager creates and manages virtual displays for headless rendering.
type DisplayManager interface {
	// CreateDisplay starts a virtual display and returns the display ID (e.g. ":99"),
	// a cleanup function that stops the display, and any error.
	CreateDisplay(width, height int) (displayID string, cleanup func(), err error)
}

// XvfbDisplayManager implements DisplayManager using Xvfb on Linux.
type XvfbDisplayManager struct {
	colorDepth int
}

// NewDisplayManager creates a display manager appropriate for the current platform.
// On Linux it returns an Xvfb-based manager; on other platforms it returns one
// that simply reports the system display.
func NewDisplayManager() DisplayManager {
	if runtime.GOOS == "linux" {
		return &XvfbDisplayManager{colorDepth: 24}
	}
	return &systemDisplayManager{}
}

// CreateDisplay starts Xvfb on an available display number and waits for it to be ready.
func (m *XvfbDisplayManager) CreateDisplay(width, height int) (string, func(), error) {
	display := ":99"
	resolution := fmt.Sprintf("%dx%dx%d", width, height, m.colorDepth)

	cmd := exec.Command("Xvfb", display, "-screen", "0", resolution,
		"+extension", "GLX", "+render", "-noreset")
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("failed to start Xvfb: %w", err)
	}

	cleanup := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}

	// Wait for display to become available (up to 5s).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			cleanup()
			return "", nil, fmt.Errorf("Xvfb did not become ready on %s within 5s", display)
		default:
			check := exec.Command("xdpyinfo", "-display", display)
			if err := check.Run(); err == nil {
				return display, cleanup, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// systemDisplayManager returns the current DISPLAY on non-Linux platforms.
type systemDisplayManager struct{}

func (m *systemDisplayManager) CreateDisplay(width, height int) (string, func(), error) {
	// On macOS / Windows, there's no Xvfb — return a stub display.
	return ":0", func() {}, nil
}

// ParseDisplayNumber extracts the numeric display ID from a string like ":99".
func ParseDisplayNumber(display string) (int, error) {
	s := strings.TrimPrefix(display, ":")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid display number %q: %w", display, err)
	}
	return n, nil
}
