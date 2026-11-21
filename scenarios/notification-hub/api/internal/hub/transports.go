package hub

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type EmailSender interface {
	Send(context.Context, string, string, string) (string, error)
	Available() (bool, string)
}

type DesktopSender interface {
	Send(context.Context, string, string, string, string) (string, error)
	Available(string) (bool, string)
}

type unavailableEmail struct{}

func (unavailableEmail) Send(context.Context, string, string, string) (string, error) {
	return "", errors.New("smtp email sender is not configured")
}

func (unavailableEmail) Available() (bool, string) {
	return false, "SMTP credentials are not configured"
}

type unavailableDesktop struct{}

func (unavailableDesktop) Send(context.Context, string, string, string, string) (string, error) {
	return "", errors.New("desktop sender is unavailable")
}

func (unavailableDesktop) Available(string) (bool, string) {
	return false, "desktop delivery is unavailable on this host"
}

type SMTPSender struct {
	Host                     string
	Port                     int
	Username, Password, From string
	Auth                     smtp.Auth
}

func NewSMTPSenderFromEnvironment(getenv func(string) string) EmailSender {
	host := strings.TrimSpace(getenv("NOTIFICATION_HUB_SMTP_HOST"))
	if host == "" {
		return unavailableEmail{}
	}
	port, _ := strconv.Atoi(strings.TrimSpace(getenv("NOTIFICATION_HUB_SMTP_PORT")))
	if port == 0 {
		port = 587
	}
	from := strings.TrimSpace(getenv("NOTIFICATION_HUB_SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(getenv("NOTIFICATION_HUB_SMTP_USERNAME"))
	}
	if from == "" {
		return unavailableEmail{}
	}
	username, password := getenv("NOTIFICATION_HUB_SMTP_USERNAME"), getenv("NOTIFICATION_HUB_SMTP_PASSWORD")
	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	return SMTPSender{Host: host, Port: port, Username: username, Password: password, From: from, Auth: auth}
}

func (s SMTPSender) Available() (bool, string) {
	return s.Host != "" && s.From != "", func() string {
		if s.Host == "" {
			return "SMTP host is not configured"
		}
		return "SMTP sender configured"
	}()
}

func (s SMTPSender) Send(ctx context.Context, to, subject, body string) (string, error) {
	if s.Host == "" || s.From == "" {
		return "", errors.New("SMTP sender is not configured")
	}
	message := []byte("From: " + s.From + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n")
	address := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	result := make(chan error, 1)
	go func() { result <- smtp.SendMail(address, s.Auth, s.From, []string{to}, message) }()
	select {
	case err := <-result:
		if err != nil {
			return "", err
		}
		return address, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

type MacOSDesktopSender struct{ Command string }

func NewMacOSDesktopSender() DesktopSender {
	if runtime.GOOS != "darwin" {
		return unavailableDesktop{}
	}
	return MacOSDesktopSender{Command: "osascript"}
}

func (s MacOSDesktopSender) Available(channel string) (bool, string) {
	if runtime.GOOS != "darwin" {
		return false, "macOS desktop delivery requires a macOS host"
	}
	if channel != "macos_notification" && channel != "imessage" {
		return false, "unsupported macOS channel"
	}
	if _, err := exec.LookPath(s.Command); err != nil {
		return false, "osascript is not installed"
	}
	return true, "macOS adapter available; Messages permissions are checked on send"
}

func (s MacOSDesktopSender) Send(ctx context.Context, channel, address, title, body string) (string, error) {
	ready, reason := s.Available(channel)
	if !ready {
		return "", errors.New(reason)
	}
	var script string
	switch channel {
	case "macos_notification":
		script = `on run argv
display notification (item 2 of argv) with title (item 1 of argv)
return "delivered"
end run`
	case "imessage":
		script = `on run argv
tell application "Messages"
set targetService to 1st service whose service type is iMessage
set targetBuddy to buddy (item 1 of argv) of targetService
send (item 3 of argv) to targetBuddy
end tell
return "delivered"
end run`
	default:
		return "", fmt.Errorf("unsupported macOS channel %q", channel)
	}
	args := []string{"-e", script, "--", address, title, body}
	cmd := exec.CommandContext(ctx, s.Command, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return address, nil
}

// LinuxDesktopSender integrates with the host desktop notification daemon.
// notify-send is intentionally treated as an optional local capability: a
// headless host must report unavailable instead of advertising a channel that
// cannot deliver.
type LinuxDesktopSender struct {
	Command  string
	Env      func(string) string
	LookPath func(string) (string, error)
}

func NewLinuxDesktopSender() DesktopSender {
	if runtime.GOOS != "linux" {
		return unavailableDesktop{}
	}
	return LinuxDesktopSender{Command: "notify-send", Env: os.Getenv, LookPath: exec.LookPath}
}

func (s LinuxDesktopSender) Available(channel string) (bool, string) {
	if runtime.GOOS != "linux" {
		return false, "Linux desktop delivery requires a Linux host"
	}
	if channel != "linux_notification" {
		return false, "unsupported Linux channel"
	}
	env := s.Env
	if env == nil {
		env = os.Getenv
	}
	if strings.TrimSpace(env("DISPLAY")) == "" && strings.TrimSpace(env("WAYLAND_DISPLAY")) == "" {
		return false, "no DISPLAY or WAYLAND_DISPLAY is available"
	}
	bus := strings.TrimSpace(env("DBUS_SESSION_BUS_ADDRESS"))
	if bus == "" {
		runtimeDir := strings.TrimSpace(env("XDG_RUNTIME_DIR"))
		if runtimeDir != "" {
			if _, err := os.Stat(filepath.Join(runtimeDir, "bus")); err == nil {
				bus = "unix:path=" + filepath.Join(runtimeDir, "bus")
			}
		}
	}
	if bus == "" {
		return false, "no D-Bus session bus is available"
	}
	lookPath := s.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if _, err := lookPath(s.Command); err != nil {
		return false, "notify-send is not installed"
	}
	return true, "Linux desktop notification adapter available"
}

func (s LinuxDesktopSender) Send(ctx context.Context, channel, address, title, body string) (string, error) {
	ready, reason := s.Available(channel)
	if !ready {
		return "", errors.New(reason)
	}
	cmd := exec.CommandContext(ctx, s.Command, title, body)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	if strings.TrimSpace(address) == "" {
		return "local", nil
	}
	return address, nil
}

// NewDesktopSender selects only the adapter supported by the running host.
// This keeps capability discovery truthful on Linux, macOS, and headless
// environments without exposing macOS channels from a Linux process.
func NewDesktopSender() DesktopSender {
	switch runtime.GOOS {
	case "darwin":
		return NewMacOSDesktopSender()
	case "linux":
		return NewLinuxDesktopSender()
	default:
		return unavailableDesktop{}
	}
}

var (
	_ EmailSender   = unavailableEmail{}
	_ DesktopSender = unavailableDesktop{}
	_ DesktopSender = LinuxDesktopSender{}
	_ DesktopSender = MacOSDesktopSender{}
)
