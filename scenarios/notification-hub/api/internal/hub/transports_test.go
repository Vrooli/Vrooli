package hub

import (
	"runtime"
	"strings"
	"testing"
)

func TestDesktopSenderDoesNotAdvertiseForeignChannels(t *testing.T) {
	sender := NewDesktopSender()
	if runtime.GOOS == "linux" {
		if ready, _ := sender.Available("macos_notification"); ready {
			t.Fatal("Linux desktop sender must not advertise macOS notifications")
		}
		if ready, _ := sender.Available("linux_notification"); ready {
			// The adapter may be ready on a workstation, or unavailable on a
			// headless test host. Both are truthful outcomes.
			return
		}
	}
}

func TestLinuxDesktopSenderRejectsUnsupportedChannel(t *testing.T) {
	sender := LinuxDesktopSender{Command: "notify-send"}
	if ready, reason := sender.Available("imessage"); ready || reason == "" {
		t.Fatalf("expected unsupported Linux channel, ready=%v reason=%q", ready, reason)
	}
}

func TestLinuxDesktopSenderReportsHeadlessSessionBusGap(t *testing.T) {
	sender := LinuxDesktopSender{
		Command: "notify-send",
		Env: func(key string) string {
			return map[string]string{
				"DISPLAY":                  "",
				"WAYLAND_DISPLAY":          "",
				"DBUS_SESSION_BUS_ADDRESS": "",
				"XDG_RUNTIME_DIR":          "",
			}[key]
		},
	}
	ready, reason := sender.Available("linux_notification")
	if ready || !strings.Contains(reason, "DISPLAY") {
		t.Fatalf("ready=%v reason=%q, want a truthful headless reason", ready, reason)
	}
}

func TestLinuxDesktopSenderRequiresSessionBusAfterDisplayIsPresent(t *testing.T) {
	sender := LinuxDesktopSender{
		Command: "notify-send",
		Env: func(key string) string {
			if key == "DISPLAY" {
				return ":99"
			}
			return ""
		},
	}
	ready, reason := sender.Available("linux_notification")
	if ready || !strings.Contains(reason, "D-Bus") {
		t.Fatalf("ready=%v reason=%q, want a session-bus reason", ready, reason)
	}
}
