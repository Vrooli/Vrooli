package autohealwatchdog

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestInspectDedicatedRequiresLinuxLingering(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	loop := loopPath(root, "linux")
	if err := os.MkdirAll(filepath.Dir(loop), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loop, []byte("loop"), 0o755); err != nil {
		t.Fatal(err)
	}
	definition, path, _, err := nativeDefinition("linux", root, home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := testSeams(root, home, func(name string, args ...string) ([]byte, error) {
		if name == "loginctl" {
			return []byte("Linger=no\n"), nil
		}
		if strings.HasSuffix(strings.Join(args, " "), "is-enabled vrooli-autoheal.service") {
			return []byte("enabled\n"), nil
		}
		return []byte("active\n"), nil
	})
	defer restore()

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(hostreqkit.Host{OS: "linux", SupportsSystemd: true}, hostreqspec.ResolvedRequirement{Name: "autoheal_watchdog", Kind: hostreqspec.KindSafeguard, Required: true, Config: map[string]any{"boot_policy": "dedicated"}})
	if status.Applied || !strings.Contains(strings.Join(status.Notes, " "), "boot protection is incomplete") {
		t.Fatalf("status=%+v, want incomplete dedicated boot protection", status)
	}
}

func TestAsUserArgsTargetsInvokingUserBus(t *testing.T) {
	uid, _, ok := hostreqkit.InvokingUserIDs()
	if !ok {
		t.Skip("invoking user uid is unavailable in this test environment")
	}
	name, args := asUserArgs("systemctl", "--user", "is-active", "vrooli-autoheal.service")
	joined := name + " " + strings.Join(args, " ")
	if !strings.Contains(joined, "XDG_RUNTIME_DIR=/run/user/"+strconv.Itoa(uid)) {
		t.Fatalf("scheduler command %q does not target invoking user runtime", joined)
	}
	if !strings.Contains(joined, "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+strconv.Itoa(uid)+"/bus") {
		t.Fatalf("scheduler command %q does not target invoking user bus", joined)
	}
}

func TestInspectSharedDoesNotRequireLingering(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	loop := loopPath(root, "linux")
	if err := os.MkdirAll(filepath.Dir(loop), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loop, []byte("loop"), 0o755); err != nil {
		t.Fatal(err)
	}
	definition, path, _, err := nativeDefinition("linux", root, home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := testSeams(root, home, func(name string, args ...string) ([]byte, error) {
		if strings.HasSuffix(strings.Join(args, " "), "is-enabled vrooli-autoheal.service") {
			return []byte("enabled\n"), nil
		}
		return []byte("active\n"), nil
	})
	defer restore()

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(hostreqkit.Host{OS: "linux", SupportsSystemd: true}, hostreqspec.ResolvedRequirement{Name: "autoheal_watchdog", Kind: hostreqspec.KindSafeguard, Required: true, Config: map[string]any{"boot_policy": "shared"}})
	if !status.Applied {
		t.Fatalf("status=%+v, want shared policy satisfied", status)
	}
}

func testSeams(root, home string, command func(string, ...string) ([]byte, error)) func() {
	originalRoot, originalHome, originalCommand, originalLinger := resolveRootFn, userHomeFn, commandOutputFn, lingeringEnabledFn
	resolveRootFn = func() (string, error) { return root, nil }
	userHomeFn = func() (string, error) { return home, nil }
	commandOutputFn = command
	lingeringEnabledFn = func(string) bool { return false }
	return func() {
		resolveRootFn, userHomeFn, commandOutputFn, lingeringEnabledFn = originalRoot, originalHome, originalCommand, originalLinger
	}
}
