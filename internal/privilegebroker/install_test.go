package privilegebroker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	want := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "vrooli", "privilege-broker.sock")
	if got := SocketPath(); got != want {
		t.Fatalf("SocketPath() = %q, want %q", got, want)
	}
}
