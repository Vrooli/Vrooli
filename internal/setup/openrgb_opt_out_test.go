package setup

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/resources"
	testenv "github.com/vrooli/vrooli/internal/testenv"
)

func TestDefaultSetupDoesNotInstallDisabledOpenRGB(t *testing.T) {
	root := t.TempDir()
	home := testenv.RuntimeHome(t)
	configDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Keep the explicit false field: the Go fixture helper omits false values
	// through `omitempty`, which would test the default-enabled migration path.
	if err := os.WriteFile(filepath.Join(configDir, "service.json"), []byte(`{"dependencies":{"resources":{"openrgb":{"enabled":false}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	selected, err := resources.NewController(root, home).EnabledResourceNames()
	if err != nil {
		t.Fatalf("read fixture resource selection: %v", err)
	}
	for _, name := range selected {
		if name == "openrgb" {
			t.Fatalf("disabled OpenRGB appeared in fixture selection %v", selected)
		}
	}

	var installed []string
	var operations []string
	svc := stubSetupDeps(t)
	svc.deps.resourceController = func(_, _ string) resourceRunner {
		return resourceRunnerFunc(func(name string, args []string, stdout, stderr io.Writer) error {
			installed = append(installed, name)
			return nil
		})
	}

	if _, err := svc.maybeInstallResources(root, home, Options{}, io.Discard, io.Discard, func(operation string) {
		operations = append(operations, operation)
	}); err != nil {
		t.Fatalf("default resource setup: %v", err)
	}
	if len(installed) != 0 {
		t.Fatalf("default setup installed resources %v; disabled OpenRGB must remain opt-in", installed)
	}
	if len(operations) != 0 {
		t.Fatalf("default setup reported resource operations %v; disabled OpenRGB must not enter the install path", operations)
	}

	selected, err = resources.NewController(root, home).EnabledResourceNames()
	if err != nil {
		t.Fatalf("read effective fixture resource selection: %v", err)
	}
	for _, name := range selected {
		if name == "openrgb" {
			t.Fatalf("disabled OpenRGB appeared in effective resource selection %v", selected)
		}
	}
}
