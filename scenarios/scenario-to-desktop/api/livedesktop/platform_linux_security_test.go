package livedesktop

import (
	"context"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"scenario-to-desktop-api/screenrecording"
)

func TestRemoteAccessCommandsBindToLoopback(t *testing.T) {
	vnc := x11vncArgs(":99", 5901)
	if !slices.Contains(vnc, "-localhost") {
		t.Fatalf("x11vnc args must include -localhost: %v", vnc)
	}
	websockify := websockifyArgs(6081, 5901)
	if websockify[0] != "127.0.0.1:6081" || websockify[1] != "127.0.0.1:5901" {
		t.Fatalf("websockify must use loopback endpoints: %v", websockify)
	}
}

func TestWaitForLoopbackListenerRequiresReachableEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := waitForLoopbackListener(port, time.Second); err != nil {
		t.Fatalf("waitForLoopbackListener() error = %v", err)
	}
}

func TestLinuxBackendOwnsPlatformProcessAndSafeCommandHelpers(t *testing.T) {
	backend := NewLinuxBackend(slog.Default())
	if backend.PlatformID() != "linux-xvfb" || backend.NewMonitorFactory() == nil {
		t.Fatal("Linux backend did not expose its platform capabilities")
	}
	if _, err := findAvailablePort(2, 1); err == nil {
		t.Fatal("invalid port range unexpectedly resolved")
	}
	if output, err := shellExec(context.Background(), nil, "sh", "-c", "printf ready"); err != nil || string(output) != "ready" {
		t.Fatalf("shellExec = %q, %v", output, err)
	}
	if got := shellQuote("it's safe"); got != "'it'\\''s safe'" {
		t.Fatalf("shellQuote = %q", got)
	}

	script := filepath.Join(t.TempDir(), "desktop-fixture")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	display := &linuxDisplay{md: &screenrecording.ManagedDisplay{DisplayID: ":99", Width: 1280, Height: 720}}
	if display.DisplayID() != ":99" || display.Width() != 1280 || display.Height() != 720 || !display.IsRunning() {
		t.Fatalf("linux display = %#v", display)
	}
	process, err := backend.LaunchApp(context.Background(), display, script, LaunchOptions{EnvVars: map[string]string{"DESKTOP_TEST": "1"}, DarkMode: true, Locale: "C"})
	if err != nil || process.PID() <= 0 || !process.IsRunning() {
		t.Fatalf("LaunchApp = %#v, %v", process, err)
	}
	backend.KillApp(process)
	if process.IsRunning() {
		t.Fatal("KillApp did not stop the launched process")
	}
	display.Stop()
	if display.IsRunning() {
		t.Fatal("display Stop was not reflected")
	}
}

func TestLinuxBackendPassesElectronArgumentsWithoutShellExpansion(t *testing.T) {
	backend := NewLinuxBackend(slog.Default())
	marker := filepath.Join(t.TempDir(), "argv")
	script := filepath.Join(t.TempDir(), "desktop-argv-fixture")
	content := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ARG_MARKER\"\nsleep 30\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	display := &linuxDisplay{md: &screenrecording.ManagedDisplay{DisplayID: ":99", Width: 1280, Height: 720}}
	process, err := backend.LaunchApp(context.Background(), display, script, LaunchOptions{
		EnvVars:   map[string]string{"ARG_MARKER": marker},
		ExtraArgs: []string{"--remote-debugging-port=9223", "--user-data-dir=/tmp/profile with spaces"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.KillApp(process)

	var got []byte
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		got, _ = os.ReadFile(marker)
		if len(got) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if string(got) != "--remote-debugging-port=9223\n--user-data-dir=/tmp/profile with spaces\n" {
		t.Fatalf("argv = %q", got)
	}
}
