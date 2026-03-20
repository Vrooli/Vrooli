package screenrecording

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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

// findAvailableDisplay probes display numbers starting at 99 to find one
// that is not already in use. It uses xdpyinfo to check each display,
// which works regardless of whether the X server listens on TCP, Unix
// sockets, or abstract sockets.
func findAvailableDisplay() (string, error) {
	for n := 99; n < 200; n++ {
		display := fmt.Sprintf(":%d", n)
		cmd := exec.Command("xdpyinfo", "-display", display)
		if err := cmd.Run(); err != nil {
			// xdpyinfo failed — display is not in use.
			return display, nil
		}
	}
	return "", fmt.Errorf("no available X display number found in range :99–:199")
}

// CreateDisplay starts Xvfb on an available display number, launches a
// lightweight window manager (best-effort), and waits for the display to
// be ready. The returned cleanup function stops the WM, then Xvfb.
func (m *XvfbDisplayManager) CreateDisplay(width, height int) (string, func(), error) {
	display, err := findAvailableDisplay()
	if err != nil {
		return "", nil, err
	}
	resolution := fmt.Sprintf("%dx%dx%d", width, height, m.colorDepth)

	cmd := exec.Command("Xvfb", display, "-screen", "0", resolution,
		"+extension", "GLX", "+render", "-noreset")
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("failed to start Xvfb on %s: %w", display, err)
	}

	// Wait for display to become available (up to 5s).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
			return "", nil, fmt.Errorf("Xvfb did not become ready on %s within 5s", display)
		default:
			check := exec.Command("xdpyinfo", "-display", display)
			if err := check.Run(); err == nil {
				goto displayReady
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

displayReady:
	// Launch a lightweight window manager so Electron windows render
	// correctly on the virtual display. This is best-effort — if no WM
	// is available, Electron may still render (with possible artifacts).
	wmProcess := startWindowManager(display)

	// Set a solid desktop background so recordings don't show a bare
	// black X11 root window. Best-effort — if it fails, recording
	// simply has a black background.
	setDesktopBackground(display)

	cleanup := func() {
		if wmProcess != nil {
			_ = wmProcess.Kill()
			_, _ = wmProcess.Wait()
		}
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}

	return display, cleanup, nil
}

// wmCandidates lists window managers to try, in preference order.
// Each entry is a command with arguments suitable for headless use.
var wmCandidates = []struct {
	name string
	args []string
}{
	{"openbox", []string{"--sm-disable"}},
	{"matchbox-window-manager", []string{"-use_titlebar", "no"}},
}

// startWindowManager attempts to launch a lightweight window manager on
// the given display. Returns the process handle (for cleanup) or nil if
// no suitable WM was found.
func startWindowManager(display string) *os.Process {
	displayEnv := fmt.Sprintf("DISPLAY=%s", display)

	for _, wm := range wmCandidates {
		wmPath, err := exec.LookPath(wm.name)
		if err != nil {
			continue
		}

		wmCmd := exec.Command(wmPath, wm.args...)
		wmCmd.Env = append(os.Environ(), displayEnv)
		if err := wmCmd.Start(); err != nil {
			slog.Warn("window manager failed to start",
				"wm", wm.name, "display", display, "error", err.Error())
			continue
		}

		// Give the WM a moment to initialize.
		time.Sleep(200 * time.Millisecond)

		slog.Info("window manager started",
			"wm", wm.name, "display", display, "pid", wmCmd.Process.Pid)
		return wmCmd.Process
	}

	slog.Warn("no window manager found; Electron may render with artifacts",
		"display", display,
		"tried", func() []string {
			names := make([]string, len(wmCandidates))
			for i, c := range wmCandidates {
				names[i] = c.name
			}
			return names
		}())
	return nil
}

// setDesktopBackground replaces the default black X11 root window with a
// solid color so screen recordings have a pleasant background. Best-effort:
// if xsetroot is missing or fails, the display simply stays black.
func setDesktopBackground(display string) {
	xsetPath, err := exec.LookPath("xsetroot")
	if err != nil {
		return
	}

	cmd := exec.Command(xsetPath, "-solid", "#1e293b", "-display", display)
	if err := cmd.Run(); err != nil {
		slog.Warn("failed to set desktop background",
			"display", display, "error", err.Error())
	} else {
		slog.Info("desktop background set",
			"display", display, "color", "#1e293b")
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
