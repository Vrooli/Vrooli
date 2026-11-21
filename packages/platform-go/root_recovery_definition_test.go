package platform

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRootRecoveryIsSeparateSourceIndependentBoundedNativeJob(t *testing.T) {
	home := filepath.Join(t.TempDir(), "operator")
	d, err := RootUpdateRecoveryDefinition("linux", home, filepath.Join(home, "bin", "watchdog"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := RenderDefinition(d, "linux")
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := CoreUnitByID(CoreUnitRootUpdateRecovery)
	service, ok := a.File(unit.Systemd)
	if !ok {
		t.Fatal("root recovery service missing")
	}
	for _, required := range []string{"--reconcile-core-updates", "TimeoutStartSec=75", "TimeoutStopSec=5", "MemoryMax=512M", "TasksMax=128", "WorkingDirectory=" + home} {
		if !strings.Contains(service.Content, required) {
			t.Fatalf("missing recovery bound %s", required)
		}
	}
	if strings.Contains(service.Content, "--report-only") || strings.Contains(service.Content, "VROOLI_SOURCE_ROOT") {
		t.Fatal("recovery depends on pressure mode or a source checkout")
	}
	timer, _ := CoreUnitByID(CoreUnitRootUpdateRecoveryTimer)
	body, ok := a.File(timer.Systemd)
	if !ok || !strings.Contains(body.Content, "Persistent=true") || !strings.Contains(body.Content, "OnUnitActiveSec=60") {
		t.Fatal("independent reboot/cadence recovery missing")
	}
	for _, target := range []string{"darwin", "windows"} {
		if _, err := RootUpdateRecoveryDefinition(target, home, d.Executable); err == nil {
			t.Fatal("unsupported recovery target silently accepted")
		}
	}
}
