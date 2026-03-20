package livedesktop

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"sync"
	"time"
)

// portMutex prevents concurrent port allocation races.
var portMutex sync.Mutex

// findAvailablePort probes for a free TCP port in [start, end].
func findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", start, end)
}

// checkVNCDependencies verifies that x11vnc and websockify are installed.
// Returns a user-friendly error with install instructions if either is missing.
func checkVNCDependencies() error {
	var missing []string
	if _, err := exec.LookPath("x11vnc"); err != nil {
		missing = append(missing, "x11vnc")
	}
	if _, err := exec.LookPath("websockify"); err != nil {
		missing = append(missing, "websockify")
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"required tools not installed: %v. "+
				"Run 'sudo apt-get install -y x11vnc websockify' or re-run "+
				"'./scripts/manage.sh setup' to install all dependencies",
			missing,
		)
	}
	return nil
}

// startVNCSession starts x11vnc and websockify as background processes.
// Returns the VNC port, WebSocket port, process handles, and any error.
func startVNCSession(display string) (vncPort, wsPort int, x11vncCmd, websockifyCmd *exec.Cmd, err error) {
	if err := checkVNCDependencies(); err != nil {
		return 0, 0, nil, nil, err
	}

	portMutex.Lock()
	defer portMutex.Unlock()

	vncPort, err = findAvailablePort(5900, 5999)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("finding VNC port: %w", err)
	}

	wsPort, err = findAvailablePort(6080, 6180)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("finding WebSocket port: %w", err)
	}

	// Start x11vnc as a background process
	x11vncCmd = exec.Command("x11vnc",
		"-display", display,
		"-rfbport", fmt.Sprintf("%d", vncPort),
		"-nopw",
		"-shared",
		"-forever",
		"-noxdamage",
	)
	if err := x11vncCmd.Start(); err != nil {
		return 0, 0, nil, nil, fmt.Errorf("starting x11vnc: %w", err)
	}

	// Give x11vnc a moment to bind its port
	time.Sleep(500 * time.Millisecond)

	// Verify x11vnc is still running
	if x11vncCmd.ProcessState != nil {
		return 0, 0, nil, nil, fmt.Errorf("x11vnc exited immediately — is the display %s active?", display)
	}

	// Start websockify as a background process
	websockifyCmd = exec.Command("websockify",
		fmt.Sprintf("%d", wsPort),
		fmt.Sprintf("localhost:%d", vncPort),
	)
	if err := websockifyCmd.Start(); err != nil {
		// Clean up x11vnc
		_ = x11vncCmd.Process.Kill()
		_ = x11vncCmd.Wait()
		return 0, 0, nil, nil, fmt.Errorf("starting websockify: %w", err)
	}

	slog.Info("VNC session started", "display", display, "vnc_port", vncPort, "ws_port", wsPort)
	return vncPort, wsPort, x11vncCmd, websockifyCmd, nil
}

// stopVNCProcesses kills websockify and x11vnc for a session.
func stopVNCProcesses(session *Session) {
	if session.WebsockifyCmd != nil && session.WebsockifyCmd.Process != nil {
		_ = session.WebsockifyCmd.Process.Kill()
		_ = session.WebsockifyCmd.Wait()
	}
	if session.X11VNCCmd != nil && session.X11VNCCmd.Process != nil {
		_ = session.X11VNCCmd.Process.Kill()
		_ = session.X11VNCCmd.Wait()
	}
}
