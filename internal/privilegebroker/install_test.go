package privilegebroker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSetupBrokerInstallationHonorsElevationConsent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux broker")
	}
	original := hostreqkit.ElevationFactsFn
	t.Cleanup(func() { hostreqkit.ElevationFactsFn = original })
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", CanElevate: true}
	}
	t.Setenv("VROOLI_PRIVILEGE_BROKER_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))
	calls := 0
	installer := Installer{Executable: "/reviewed/vrooli", RepoRoot: "/project", effectiveUID: func() int { return 1000 }, SetupRun: func(_ context.Context, command string, args []string, out, _ io.Writer) error {
		calls++
		if command != "sudo" || strings.Join(args, " ") != "/reviewed/vrooli __privilege-broker install --repo-root /project" {
			t.Fatalf("unexpected elevation: %s %v", command, args)
		}
		return json.NewEncoder(out).Encode(SetupStatus{Available: true, Supported: true, Capabilities: capabilities()})
	}}
	status, err := installer.InstallForSetup(context.Background(), "skip", io.Discard, io.Discard)
	if err != nil || status.Available || calls != 0 || !strings.Contains(status.Recovery, "vrooli setup --sudo-mode=ask") {
		t.Fatalf("ordinary setup=%+v err=%v calls=%d", status, err, calls)
	}
	status, err = installer.InstallForSetup(context.Background(), "ask", io.Discard, io.Discard)
	if err != nil || !status.Available || calls != 1 {
		t.Fatalf("consented setup=%+v err=%v calls=%d", status, err, calls)
	}
	_, err = installer.InstallForSetup(context.Background(), "error", io.Discard, io.Discard)
	if !errors.Is(err, hostreqkit.ErrSudoUnavailable) || calls != 1 {
		t.Fatalf("error policy: err=%v calls=%d", err, calls)
	}
}

func TestSetupBrokerDoesNotClaimAvailabilityAfterInstallationFailure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux broker")
	}
	original := hostreqkit.ElevationFactsFn
	t.Cleanup(func() { hostreqkit.ElevationFactsFn = original })
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", CanElevate: true}
	}
	installer := Installer{Executable: "/reviewed/vrooli", effectiveUID: func() int { return 1000 }, SetupRun: func(_ context.Context, _ string, _ []string, out, _ io.Writer) error {
		_, _ = io.WriteString(out, `{ "available": false }`)
		return nil
	}}
	if _, err := installer.InstallForSetup(context.Background(), "ask", io.Discard, io.Discard); err == nil {
		t.Fatal("failed installation was reported available")
	}
}

func TestSystemdUnitRunsOnlyInternalBrokerWithExplicitCallerIdentity(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	unit := systemdUnit("/usr/local/lib/vrooli/vrooli-privilege-broker", 1000, 1000)
	for _, want := range []string{
		"User=root",
		"__privilege-broker serve",
		"--socket /run/vrooli/privilege-broker.sock",
		"--allowed-uid 1000",
		"--socket-gid 1000",
		"RuntimeDirectoryMode=755",
		"NoNewPrivileges=true",
		"ProtectSystem=strict",
		"PrivateTmp=false",
		"ReadOnlyPaths=/tmp /var/tmp",
		"LogsDirectory=vrooli",
		"ReadWritePaths=/run/vrooli -/run/ufw.lock /etc/ufw /var/log/vrooli",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit missing %q:\n%s", want, unit)
		}
	}
	for _, forbidden := range []string{"sudo ", "sh -c", "--command", "--argv", "tcp://"} {
		if strings.Contains(unit, forbidden) {
			t.Fatalf("unit must not contain %q:\n%s", forbidden, unit)
		}
	}
}

func TestCopyAtomicallyReplacesExistingBrokerBinary(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "vrooli-privilege-broker")
	if err := os.WriteFile(source, []byte("new broker"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("old broker"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyAtomically(source, destination, func(string, int, int) error { return nil }); err != nil {
		t.Fatalf("replace running broker path: %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new broker" {
		t.Fatalf("destination=%q want new broker", got)
	}
	if mode, err := os.Stat(destination); err != nil || mode.Mode().Perm() != 0o755 {
		t.Fatalf("destination mode=%v err=%v", mode.Mode(), err)
	}
}

func TestBrokerServiceCommandsRestartAfterInstall(t *testing.T) {
	got := brokerServiceCommands()
	want := [][]string{
		{"daemon-reload"},
		{"enable", serviceName},
		{"restart", serviceName},
		{"is-active", "--quiet", serviceName},
	}
	if len(got) != len(want) {
		t.Fatalf("commands=%#v", got)
	}
	for index := range want {
		if strings.Join(got[index], " ") != strings.Join(want[index], " ") {
			t.Fatalf("command[%d]=%q want %q", index, got[index], want[index])
		}
	}
}

func TestSocketPathUsesRuntimeDirectoryWhenConfigured(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(t.TempDir(), "runtime"))
	if runtime.GOOS == "linux" {
		if got := SocketPath(); got != DefaultSocketPath {
			t.Fatalf("SocketPath() = %q, want system broker path %q", got, DefaultSocketPath)
		}
		return
	}
	want := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "vrooli", "privilege-broker.sock")
	if got := SocketPath(); got != want {
		t.Fatalf("SocketPath() = %q, want %q", got, want)
	}
}
