package hostdesktop

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

const (
	captureTimeout         = 5 * time.Second
	maxCaptureBytes        = 32 << 20
	maxCapturePixels       = 64 << 20
	maxEncodedCaptureBytes = ((maxCaptureBytes + 2) / 3) * 4
)

type Adapter struct {
	display    string
	xauthority string
	capture    func(context.Context) (strategy.Frame, error)
	command    func(context.Context, string, ...string) ([]byte, error)
	run        func(context.Context, string, ...string) error
}

func New() *Adapter {
	a := &Adapter{display: strings.TrimSpace(os.Getenv("DISPLAY")), xauthority: strings.TrimSpace(os.Getenv("XAUTHORITY"))}
	if a.display == "" || a.xauthority == "" {
		// Managed lifecycle startup passes the helper config to the sidecar, but
		// does not need to duplicate its private display binding in the API
		// process environment. Reuse only the public display number and the
		// already-provisioned Xauthority path; malformed config remains an
		// unavailable probe result rather than an inferred desktop grant.
		var binding struct {
			Display        *uint16 `json:"display"`
			XAuthorityFile string  `json:"xauthority_file"`
		}
		path := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_DESKTOP_HELPER_CONFIG"))
		if path != "" {
			if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &binding) == nil {
				if a.display == "" && binding.Display != nil {
					a.display = fmt.Sprintf(":%d", *binding.Display)
				}
				if a.xauthority == "" {
					a.xauthority = strings.TrimSpace(binding.XAuthorityFile)
				}
			}
		}
	}
	a.capture = a.Observe
	return a
}
func (a *Adapter) ID() string { return "host-desktop" }

// Enumerate exposes the authenticated local desktop as one stable target when
// the configured display has produced a complete capture. A desktop is a
// target in its own right; it does not need a physical serial number, and a
// failed probe must never be represented as an available device.
func (a *Adapter) Enumerate(ctx context.Context) ([]strategy.Device, error) {
	declaration, err := a.Describe(ctx)
	if err != nil {
		return nil, err
	}
	if declaration.Status != strategy.StatusAvailable {
		reason := declaration.Capabilities[strategy.CapScreenshot].Reason
		if reason == "" {
			reason = "desktop capture probe failed"
		}
		return nil, errors.New(reason)
	}
	node := strings.TrimSpace(os.Getenv("VROOLI_NODE_ID"))
	if node == "" {
		node, _ = os.Hostname()
	}
	if node == "" {
		node = "local"
	}
	session := strings.TrimSpace(os.Getenv("XDG_SESSION_ID"))
	if session == "" {
		session = strings.TrimSpace(a.display)
	}
	if session == "" {
		session = "default"
	}
	return []strategy.Device{{
		ID:           "host-desktop",
		Name:         "Local desktop",
		Serial:       node + ":" + session,
		Model:        "Local desktop",
		StrategyID:   a.ID(),
		Transport:    "local",
		Health:       strategy.StatusAvailable,
		HealthReason: "desktop capture probe succeeded",
		ObservedAt:   time.Now().UTC(),
	}}, nil
}

func (a *Adapter) Describe(ctx context.Context) (strategy.Declaration, error) {
	const description = "The local host desktop through the configured display"
	supported := []string{"linux", "darwin", "windows"}
	if unsupported, ok := strategy.ResolveHostSupport(a.ID(), description, supported); ok {
		return unsupported, nil
	}
	// Discovery never injects input to test permission. Executable discovery is
	// not evidence of permission or a binding to an authenticated user session.
	inputNext := "Complete user-session input admission before controlling this desktop."
	captureNext := "Check the desktop session and screen capture permission, then retry capture."
	semanticNext := "Provision the platform accessibility adapter before using semantic targets."
	d := strategy.UnavailableDeclaration(a.ID(), description, []strategy.Capability{
		{Name: strategy.CapInput, Reason: "user-session input admission is unverified", NextAction: inputNext},
		{Name: strategy.CapScreenshot, Reason: "capture has not succeeded", NextAction: captureNext},
		{Name: strategy.CapSemanticTree, Reason: "native accessibility adapter is not provisioned", NextAction: semanticNext},
	}, captureNext, inputNext)
	d = strategy.WithSupportedHostOS(d, supported...)
	if strategy.HostOS == "linux" && activeWaylandSession() {
		// A Wayland login may also export DISPLAY for XWayland.  Treating that
		// compatibility display as an authorized desktop would bypass the
		// compositor's portal and can target the wrong session.
		const reason = "active Wayland session requires a portal-backed adapter"
		d.Capabilities[strategy.CapScreenshot] = strategy.Capability{Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: reason, NextAction: "Select the named GNOME or KDE Wayland backend."}
		d.Capabilities[strategy.CapInput] = strategy.Capability{Name: strategy.CapInput, Status: strategy.StatusUnavailable, Reason: reason, NextAction: "Select the named GNOME or KDE Wayland backend."}
		d.Capabilities[strategy.CapSemanticTree] = strategy.Capability{Name: strategy.CapSemanticTree, Status: strategy.StatusUnavailable, Reason: reason, NextAction: "Select the named GNOME or KDE Wayland backend."}
		d.NextActions = []string{"Select the named GNOME or KDE Wayland backend."}
		d.Tiers = strategy.Tiers(d)
		return d, nil
	}
	probeCtx, cancel := context.WithTimeout(ctx, captureTimeout)
	defer cancel()
	capture := a.capture
	if capture == nil {
		capture = a.Observe
	}
	frame, err := capture(probeCtx)
	if err != nil || probeCtx.Err() != nil || frame.Width <= 0 || frame.Height <= 0 || len(frame.Bytes) == 0 {
		// Do not expose command stderr, desktop pixels, or environment details in
		// inventory. Failure is not sufficient evidence to infer permission denial.
		d.Capabilities[strategy.CapScreenshot] = strategy.Capability{Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "desktop capture probe failed", NextAction: captureNext}
		return d, nil
	}
	d.Status = strategy.StatusAvailable
	d.NextActions = []string{inputNext}
	d.Capabilities[strategy.CapScreenshot] = strategy.ProbeCapability(strategy.CapScreenshot, true, "", "", fmt.Sprintf("decoded PNG %dx%d at %s", frame.Width, frame.Height, frame.Timestamp.UTC().Format(time.RFC3339Nano)))
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func (a *Adapter) Observe(ctx context.Context) (strategy.Frame, error) {
	if strategy.HostOS == "linux" && activeWaylandSession() {
		return strategy.Frame{}, &strategy.AvailabilityError{Reason: "active Wayland session requires a portal-backed adapter", NextAction: "Select the named GNOME or KDE Wayland backend."}
	}
	if strategy.HostOS == "darwin" {
		data, err := a.captureCommand(ctx, "screencapture", "-x", "-t", "png", "-")
		if err != nil {
			return strategy.Frame{}, fmt.Errorf("capture macOS desktop: %w", err)
		}
		return decodeFrame(data)
	}
	if strategy.HostOS == "windows" {
		// PowerShell is used only as the native Windows user-session boundary.
		// It emits base64 so a console host cannot rewrite binary PNG bytes.
		encoded, err := a.captureCommandLimit(ctx, windowsCaptureCommand(), maxEncodedCaptureBytes)
		if err != nil {
			return strategy.Frame{}, fmt.Errorf("capture Windows desktop: %w", err)
		}
		encoded = []byte(strings.TrimSpace(string(encoded)))
		if len(encoded) == 0 || len(encoded) > maxEncodedCaptureBytes {
			return strategy.Frame{}, errors.New("invalid Windows capture encoding")
		}
		data, err := base64.StdEncoding.DecodeString(string(encoded))
		if err != nil {
			return strategy.Frame{}, fmt.Errorf("decode Windows desktop capture: %w", err)
		}
		return decodeFrame(data)
	}
	if a.display == "" {
		return strategy.Frame{}, &strategy.AvailabilityError{Reason: "no display session", NextAction: "Start a usable DISPLAY session for host-desktop."}
	}
	// Import is present on the supported desktop images. If it is not, report a
	// precise unavailable result instead of fabricating a successful capture.
	data, err := a.captureCommand(ctx, "import", "-display", a.display, "-window", "root", "png:-")
	if err != nil {
		// ImageMagick is common on supported desktop images but is not a
		// requirement of the X11 protocol. Use the standard xwd utility and
		// ffmpeg's xwd_pipe demuxer when import is absent, preserving the same
		// bounded PNG contract and avoiding a false unavailable result.
		fallback, fallbackErr := a.captureX11(ctx)
		if fallbackErr == nil {
			return decodeFrame(fallback)
		}
		return strategy.Frame{}, fmt.Errorf("capture host display: %w (x11 fallback: %v)", err, fallbackErr)
	}
	return decodeFrame(data)
}

func (a *Adapter) captureX11(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, captureTimeout)
	defer cancel()
	xwd := a.nativeCommand(ctx, "xwd", "-display", a.display, "-root", "-silent")
	xwd.WaitDelay = time.Second
	ffmpeg := a.nativeCommand(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "xwd_pipe", "-i", "-", "-frames:v", "1", "-f", "image2pipe", "-vcodec", "png", "-")
	ffmpeg.WaitDelay = time.Second
	pipe, err := xwd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	ffmpeg.Stdin = pipe
	var output boundedCapture
	output.limit = maxCaptureBytes
	ffmpeg.Stdout = &output
	if err := ffmpeg.Start(); err != nil {
		return nil, err
	}
	if err := xwd.Start(); err != nil {
		_ = pipe.Close()
		_ = ffmpeg.Wait()
		return nil, err
	}
	xwdErr := xwd.Wait()
	if closeErr := pipe.Close(); xwdErr == nil && closeErr != nil && !errors.Is(closeErr, io.EOF) && !errors.Is(closeErr, os.ErrClosed) {
		xwdErr = closeErr
	}
	ffmpegErr := ffmpeg.Wait()
	if xwdErr != nil {
		return nil, xwdErr
	}
	if ffmpegErr != nil {
		return nil, ffmpegErr
	}
	return output.Bytes(), nil
}

// boundedCapture avoids buffering unbounded native output. Returning an error
// terminates the pipe copy; CommandContext also imposes a fixed deadline.
type boundedCapture struct {
	bytes.Buffer
	limit int
}

func (b *boundedCapture) Write(p []byte) (int, error) {
	limit := maxCaptureBytes
	if b.limit > 0 {
		limit = b.limit
	}
	if len(p) > limit-b.Len() {
		return 0, errors.New("capture exceeds byte limit")
	}
	return b.Buffer.Write(p)
}

func (a *Adapter) captureCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return a.captureCommandLimit(ctx, append([]string{name}, args...), maxCaptureBytes)
}

func (a *Adapter) captureCommandLimit(ctx context.Context, args []string, limit int) ([]byte, error) {
	if len(args) == 0 || args[0] == "" {
		return nil, errors.New("capture command is empty")
	}
	ctx, cancel := context.WithTimeout(ctx, captureTimeout)
	defer cancel()
	if a.command != nil {
		return a.command(ctx, args[0], args[1:]...)
	}
	cmd := a.nativeCommand(ctx, args[0], args[1:]...)
	cmd.WaitDelay = time.Second
	output := boundedCapture{limit: limit}
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func decodeFrame(data []byte) (strategy.Frame, error) {
	if len(data) == 0 || len(data) > maxCaptureBytes {
		return strategy.Frame{}, errors.New("invalid capture byte size")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return strategy.Frame{}, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxCapturePixels {
		return strategy.Frame{}, errors.New("capture dimensions exceed limit")
	}
	// A valid header alone is not a successful capture: reject truncated pixels
	// and invalid checksums before reporting readiness.
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return strategy.Frame{}, err
	}
	return strategy.Frame{Width: cfg.Width, Height: cfg.Height, Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/png", Bytes: data}, nil
}

func (a *Adapter) Actuate(ctx context.Context, event strategy.Actuation) error {
	if strategy.HostOS == "linux" && activeWaylandSession() {
		return &strategy.AvailabilityError{Reason: "active Wayland session requires a portal-backed adapter", NextAction: "Select the named GNOME or KDE Wayland backend."}
	}
	if strategy.HostOS == "darwin" {
		return a.actuateMacOS(ctx, event)
	}
	if strategy.HostOS == "windows" {
		return a.actuateWindows(ctx, event)
	}
	if a.display == "" {
		return &strategy.AvailabilityError{Reason: "no display session", NextAction: "Start a usable DISPLAY session for host-desktop."}
	}
	if event.Pointer != nil {
		p := event.Pointer
		args := []string{"mousemove", strconv.Itoa(int(p.X)), strconv.Itoa(int(p.Y))}
		switch strings.ToLower(strings.TrimSpace(p.Kind)) {
		case "move", "":
		case "down":
			args = append(args, "mousedown", buttonNumber(p.Button))
		case "up":
			args = append(args, "mouseup", buttonNumber(p.Button))
		default:
			args = append(args, "click", buttonNumber(p.Button))
		}
		return a.runCommand(ctx, "xdotool", args...)
	}
	if event.Key != nil {
		if event.Key.Key == "" {
			return errors.New("host-desktop key is empty")
		}
		return a.runCommand(ctx, "xdotool", "key", event.Key.Key)
	}
	if event.Text != "" {
		return a.runCommand(ctx, "xdotool", "type", "--delay", "0", "--", event.Text)
	}
	return errors.New("host-desktop action is empty")
}

func activeWaylandSession() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")), "wayland") || strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) != ""
}

func (a *Adapter) runCommand(ctx context.Context, name string, args ...string) error {
	if a.run != nil {
		return a.run(ctx, name, args...)
	}
	return a.nativeCommand(ctx, name, args...).Run()
}

func (a *Adapter) nativeCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	env := os.Environ()
	if a.display != "" && strings.TrimSpace(os.Getenv("DISPLAY")) == "" {
		env = append(env, "DISPLAY="+a.display)
	}
	if a.xauthority != "" && strings.TrimSpace(os.Getenv("XAUTHORITY")) == "" {
		env = append(env, "XAUTHORITY="+a.xauthority)
	}
	cmd.Env = env
	return cmd
}

func (a *Adapter) actuateMacOS(ctx context.Context, event strategy.Actuation) error {
	var script string
	switch {
	case event.Pointer != nil:
		p := event.Pointer
		script = fmt.Sprintf("tell application \"System Events\" to click at {%d, %d}", int(p.X), int(p.Y))
	case event.Key != nil:
		if event.Key.Key == "" {
			return errors.New("host-desktop key is empty")
		}
		script = fmt.Sprintf("tell application \"System Events\" to keystroke %s", appleScriptString(event.Key.Key))
	case event.Text != "":
		script = fmt.Sprintf("tell application \"System Events\" to keystroke %s", appleScriptString(event.Text))
	default:
		return errors.New("host-desktop action is empty")
	}
	return a.runCommand(ctx, "osascript", "-e", script)
}

func (a *Adapter) actuateWindows(ctx context.Context, event strategy.Actuation) error {
	var expression string
	switch {
	case event.Pointer != nil:
		p := event.Pointer
		expression = windowsPointerExpression(strconv.Itoa(int(p.X)), strconv.Itoa(int(p.Y)), strings.ToLower(strings.TrimSpace(p.Kind)), strings.ToLower(strings.TrimSpace(p.Button)))
	case event.Key != nil:
		if event.Key.Key == "" {
			return errors.New("host-desktop key is empty")
		}
		expression = fmt.Sprintf("[System.Windows.Forms.SendKeys]::SendWait(%s)", powershellString(event.Key.Key))
	case event.Text != "":
		expression = fmt.Sprintf("[System.Windows.Forms.SendKeys]::SendWait(%s)", powershellString(escapeSendKeys(event.Text)))
	default:
		return errors.New("host-desktop action is empty")
	}
	script := "Add-Type -AssemblyName System.Windows.Forms; Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class VrooliInput { [DllImport(\"user32.dll\")] public static extern bool SetCursorPos(int X, int Y); [DllImport(\"user32.dll\")] public static extern void mouse_event(uint flags, uint dx, uint dy, uint data, UIntPtr extra); }'; " + expression
	return a.runCommand(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
}

func windowsCaptureCommand() []string {
	script := "$v=[System.Windows.Forms.SystemInformation]::VirtualScreen; $b=New-Object Drawing.Bitmap $v.Width,$v.Height; $g=[Drawing.Graphics]::FromImage($b); $g.CopyFromScreen($v.Left,$v.Top,0,0,$b.Size); $m=New-Object IO.MemoryStream; $b.Save($m,[Drawing.Imaging.ImageFormat]::Png); [Convert]::ToBase64String($m.ToArray()); $g.Dispose(); $b.Dispose(); $m.Dispose()"
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; " + script}
}

func windowsPointerExpression(x, y, kind, button string) string {
	flags := map[string]string{"left": "0x0002", "right": "0x0008", "middle": "0x0020"}
	flag := flags[button]
	if flag == "" {
		flag = flags["left"]
	}
	up := map[string]string{"left": "0x0004", "right": "0x0010", "middle": "0x0040"}[button]
	if up == "" {
		up = "0x0004"
	}
	move := fmt.Sprintf("[VrooliInput]::SetCursorPos(%s,%s)", x, y)
	switch kind {
	case "down":
		return move + fmt.Sprintf("; [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero)", flag)
	case "up":
		return move + fmt.Sprintf("; [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero)", up)
	default:
		return move + fmt.Sprintf("; [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero); [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero)", flag, up)
	}
}

func buttonNumber(button string) string {
	switch strings.ToLower(strings.TrimSpace(button)) {
	case "right":
		return "3"
	case "middle":
		return "2"
	default:
		return "1"
	}
}

func powershellString(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func escapeSendKeys(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '+', '^', '%', '~', '(', ')', '[', ']', '{', '}':
			b.WriteByte('{')
			b.WriteRune(r)
			b.WriteByte('}')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
