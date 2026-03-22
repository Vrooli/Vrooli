package livedesktop

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/screenrecording"
)

// LinuxBackend implements PlatformBackend using Xvfb, x11vnc, xclip, xrandr, and unshare.
type LinuxBackend struct {
	displayMgr screenrecording.DisplayManager
	logger     *slog.Logger
}

// NewLinuxBackend creates a Linux-specific platform backend.
func NewLinuxBackend(logger *slog.Logger) *LinuxBackend {
	return &LinuxBackend{
		displayMgr: screenrecording.NewDisplayManager(),
		logger:     logger,
	}
}

func (b *LinuxBackend) PlatformID() string { return "linux-xvfb" }

// --- Display lifecycle ---

// linuxDisplay adapts screenrecording.ManagedDisplay to PlatformDisplay.
type linuxDisplay struct {
	md *screenrecording.ManagedDisplay
}

func (d *linuxDisplay) DisplayID() string { return d.md.DisplayID }
func (d *linuxDisplay) Width() int        { return d.md.Width }
func (d *linuxDisplay) Height() int       { return d.md.Height }
func (d *linuxDisplay) IsRunning() bool   { return d.md.IsRunning() }
func (d *linuxDisplay) Stop()             { d.md.Stop() }

func (b *LinuxBackend) CreateDisplay(width, height int) (PlatformDisplay, error) {
	md, err := b.displayMgr.CreateManagedDisplay(width, height)
	if err != nil {
		return nil, err
	}
	return &linuxDisplay{md: md}, nil
}

// --- Remote access (VNC) ---

// linuxRemoteAccess holds x11vnc and websockify process handles.
type linuxRemoteAccess struct {
	x11vncCmd     *exec.Cmd
	websockifyCmd *exec.Cmd
}

// portMutex prevents concurrent port allocation races.
var portMutex sync.Mutex

func (b *LinuxBackend) StartRemoteAccess(display PlatformDisplay) (RemoteAccessInfo, RemoteAccessHandle, error) {
	if err := checkLinuxVNCDeps(); err != nil {
		return RemoteAccessInfo{}, nil, err
	}

	portMutex.Lock()
	defer portMutex.Unlock()

	vncPort, err := findAvailablePort(5900, 5999)
	if err != nil {
		return RemoteAccessInfo{}, nil, fmt.Errorf("finding VNC port: %w", err)
	}

	wsPort, err := findAvailablePort(6080, 6180)
	if err != nil {
		return RemoteAccessInfo{}, nil, fmt.Errorf("finding WebSocket port: %w", err)
	}

	x11vncCmd := exec.Command("x11vnc",
		"-display", display.DisplayID(),
		"-rfbport", fmt.Sprintf("%d", vncPort),
		"-nopw", "-shared", "-forever", "-noxdamage",
	)
	if err := x11vncCmd.Start(); err != nil {
		return RemoteAccessInfo{}, nil, fmt.Errorf("starting x11vnc: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	if x11vncCmd.ProcessState != nil {
		return RemoteAccessInfo{}, nil, fmt.Errorf("x11vnc exited immediately — is the display %s active?", display.DisplayID())
	}

	websockifyCmd := exec.Command("websockify",
		fmt.Sprintf("%d", wsPort),
		fmt.Sprintf("localhost:%d", vncPort),
	)
	if err := websockifyCmd.Start(); err != nil {
		_ = x11vncCmd.Process.Kill()
		_ = x11vncCmd.Wait()
		return RemoteAccessInfo{}, nil, fmt.Errorf("starting websockify: %w", err)
	}

	b.logger.Info("VNC session started", "display", display.DisplayID(), "vnc_port", vncPort, "ws_port", wsPort)

	info := RemoteAccessInfo{
		Protocol: "vnc",
		Port:     vncPort,
		WSPort:   wsPort,
	}
	handle := &linuxRemoteAccess{
		x11vncCmd:     x11vncCmd,
		websockifyCmd: websockifyCmd,
	}
	return info, handle, nil
}

func (b *LinuxBackend) StopRemoteAccess(handle RemoteAccessHandle) {
	ra, ok := handle.(*linuxRemoteAccess)
	if !ok || ra == nil {
		return
	}
	if ra.websockifyCmd != nil && ra.websockifyCmd.Process != nil {
		_ = ra.websockifyCmd.Process.Kill()
		_ = ra.websockifyCmd.Wait()
	}
	if ra.x11vncCmd != nil && ra.x11vncCmd.Process != nil {
		_ = ra.x11vncCmd.Process.Kill()
		_ = ra.x11vncCmd.Wait()
	}
}

// --- App lifecycle ---

// linuxProcess wraps an exec.Cmd process handle.
type linuxProcess struct {
	cmd *exec.Cmd
}

func (p *linuxProcess) PID() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

func (p *linuxProcess) IsRunning() bool {
	if p.cmd == nil || p.cmd.ProcessState != nil {
		return false
	}
	return p.cmd.Process != nil
}

func (b *LinuxBackend) LaunchApp(ctx context.Context, display PlatformDisplay, appPath string, opts LaunchOptions) (PlatformProcess, error) {
	// Build environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("DISPLAY=%s", display.DisplayID()))

	for k, v := range opts.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	if opts.DarkMode {
		env = append(env, "GTK_THEME=Adwaita:dark")
	}
	if opts.Locale != "" {
		env = append(env, fmt.Sprintf("LANG=%s", opts.Locale))
		env = append(env, fmt.Sprintf("LC_ALL=%s", opts.Locale))
	}

	// Build command based on network mode
	var cmdName string
	var cmdArgs []string

	switch opts.NetworkMode {
	case "offline":
		cmdName = "unshare"
		cmdArgs = []string{"--net", appPath}
	case "slow":
		bw := opts.BandwidthKbps
		if bw <= 0 {
			bw = 256
		}
		tcCmd := fmt.Sprintf("tc qdisc add dev lo root tbf rate %dkbit burst 32kbit latency 400ms && exec %s",
			bw, appPath)
		cmdName = "unshare"
		cmdArgs = []string{"--net", "sh", "-c", tcCmd}
	default:
		cmdName = appPath
	}

	if opts.DarkMode {
		cmdArgs = append(cmdArgs, "--force-dark-mode")
	}

	cmd := exec.CommandContext(context.Background(), cmdName, cmdArgs...)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("launching app: %w", err)
	}

	return &linuxProcess{cmd: cmd}, nil
}

func (b *LinuxBackend) KillApp(proc PlatformProcess) {
	lp, ok := proc.(*linuxProcess)
	if !ok || lp == nil || lp.cmd == nil {
		return
	}
	if lp.cmd.Process != nil {
		_ = lp.cmd.Process.Kill()
		_ = lp.cmd.Wait()
	}
}

// --- Capture ---

func (b *LinuxBackend) CaptureScreenshot(ctx context.Context, display PlatformDisplay, outputPath string) error {
	pipeline := fmt.Sprintf("xwd -display %s -root -silent | ffmpeg -y -f xwd_pipe -i - -frames:v 1 -update 1 %s",
		display.DisplayID(), outputPath)
	_, err := shellExec(ctx, nil, "sh", "-c", pipeline)
	return err
}

// --- Clipboard ---

func (b *LinuxBackend) ReadClipboard(ctx context.Context, display PlatformDisplay) (string, error) {
	if _, err := exec.LookPath("xclip"); err != nil {
		return "", fmt.Errorf("xclip is not installed (sudo apt-get install -y xclip)")
	}
	out, err := shellExec(ctx, nil, "xclip", "-display", display.DisplayID(), "-selection", "clipboard", "-o")
	if err != nil {
		return "", fmt.Errorf("clipboard read failed: %w", err)
	}
	return string(out), nil
}

func (b *LinuxBackend) WriteClipboard(ctx context.Context, display PlatformDisplay, content string) error {
	if _, err := exec.LookPath("xclip"); err != nil {
		return fmt.Errorf("xclip is not installed (sudo apt-get install -y xclip)")
	}
	cmd := fmt.Sprintf("echo -n %s | xclip -display %s -selection clipboard -i",
		shellQuote(content), display.DisplayID())
	_, err := shellExec(ctx, nil, "sh", "-c", cmd)
	return err
}

// --- Display manipulation ---

func (b *LinuxBackend) ResizeDisplay(ctx context.Context, display PlatformDisplay, width, height int) error {
	sizeArg := fmt.Sprintf("%dx%d", width, height)
	_, err := shellExec(ctx, nil, "xrandr", "--display", display.DisplayID(), "-s", sizeArg)
	if err != nil {
		return fmt.Errorf("xrandr resize failed: %w", err)
	}
	return nil
}

// --- Process monitoring ---

func (b *LinuxBackend) NewMonitorFactory() procmetrics.MonitorFactory {
	procReader := &procmetrics.LinuxProcReader{}
	shellFn := procmetrics.ShellFunc(func(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
		return shellExec(ctx, env, name, args...)
	})
	windowDetector := procmetrics.NewXdotoolDetector(shellFn, b.logger)
	return procmetrics.NewDefaultMonitorFactory(procReader, windowDetector, b.logger)
}

// --- Helpers ---

// checkLinuxVNCDeps verifies that x11vnc and websockify are installed.
func checkLinuxVNCDeps() error {
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

// shellExec runs a command and returns stdout.
func shellExec(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// shellQuote wraps a string in single quotes, escaping existing single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
