package setup

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/resources"
)

// Canonical project defaults are copied unchanged. No policy or enabled intent
// is written before the first normal setup. Other resources stop at the existing
// install boundary; this fixture never installs software or invokes a vendor.
func TestOpenRGBCleanCanonicalDefaultSetupAndExplicitOptIn(t *testing.T) {
	storagetest.IsolateStorage(t)
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	source, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	testkitgo.WriteRepoContract(t, root, "scenarios")
	for _, name := range []string{".vrooli/service.json", ".vrooli/schemas/operator-state.schema.json", ".vrooli/schemas/common.schema.json", "resources/openrgb/resource.json"} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		testkitgo.WriteFile(t, filepath.Join(root, name), string(data))
	}
	controller := resources.NewController(root, home)
	names, err := controller.EnabledResourceNames()
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(names, "openrgb") {
		t.Fatal("canonical untouched defaults enable OpenRGB")
	}
	if len(names) == 0 {
		t.Fatal("fixture accidentally removed unrelated enabled defaults")
	}
	var dispatched, operations []string
	effectsBlocked := errors.New("explicit opt-in reached install boundary; effects intentionally refused")
	svc := stubSetupDeps(t)
	svc.deps.resourceController = func(gotRoot, gotHome string) resourceRunner {
		if gotRoot != root || gotHome != home {
			t.Fatal("setup escaped private owner")
		}
		return resourceRunnerFunc(func(name string, args []string, _, _ io.Writer) error {
			dispatched = append(dispatched, name)
			if name == "openrgb" {
				return effectsBlocked
			}
			return nil
		})
	}
	setup := func() []string {
		dispatched = nil
		operations = nil
		degraded, err := svc.maybeInstallResources(root, home, Options{}, io.Discard, io.Discard, func(operation string) { operations = append(operations, operation) })
		if err != nil {
			t.Fatal(err)
		}
		return degraded
	}
	if degraded := setup(); slices.Contains(dispatched, "openrgb") || slices.Contains(degraded, "openrgb") || slices.Contains(operations, "Installing resource openrgb") {
		t.Fatalf("untouched default entered lighting install/acquisition/process/probe path: dispatch=%v operations=%v degraded=%v", dispatched, operations, degraded)
	}
	// Positive control is separate from the untouched-default observation. The
	// sentinel is expected degradation, not fake successful backend qualification.
	if err := controller.SetEnabled("openrgb", true); err != nil {
		t.Fatal(err)
	}
	if degraded := setup(); !slices.Contains(dispatched, "openrgb") || !slices.Contains(degraded, "openrgb") || !slices.Contains(operations, "Installing resource openrgb") {
		t.Fatalf("explicit opt-in did not reach guarded install boundary: dispatch=%v degraded=%v", dispatched, degraded)
	}
	if err := controller.SetEnabled("openrgb", false); err != nil {
		t.Fatal(err)
	}
	if degraded := setup(); slices.Contains(dispatched, "openrgb") || slices.Contains(degraded, "openrgb") {
		t.Fatalf("explicit disable re-entered lighting install boundary: %v %v", dispatched, degraded)
	}
}
