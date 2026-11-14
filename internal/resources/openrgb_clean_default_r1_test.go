package resources

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestOpenRGBCleanCanonicalDefaultStartupHasNoLightingEffects(t *testing.T) {
	storagetest.IsolateStorage(t)
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(home, "storage"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", filepath.Join(home, "artifacts"))
	testkitgo.WriteRepoContract(t, root, "scenarios")
	for _, name := range []string{".vrooli/service.json", ".vrooli/schemas/operator-state.schema.json", ".vrooli/schemas/common.schema.json", "resources/openrgb/resource.json"} {
		data, err := os.ReadFile(filepath.Join(projectRootForResourcesTest(t), name))
		if err != nil {
			t.Fatal(err)
		}
		testkitgo.WriteFile(t, filepath.Join(root, name), string(data))
	}
	c := NewController(root, home)
	items, err := c.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "openrgb" || items[0].Enabled {
		t.Fatalf("clean canonical catalog=%+v", items)
	}
	for _, path := range []string{filepath.Join(home, "storage"), filepath.Join(home, "artifacts")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("fixture began with lighting state: %s: %v", path, err)
		}
	}
	// This is the same control service Controller.StartAll delegates to, with
	// only its existing last-effect boundaries observed. No successful status,
	// install, native dispatch, detector/SDK probe or device access is substituted.
	control := c.resourceControl()
	statusCalls, driverCalls, cliCalls := 0, 0, 0
	blocked := errors.New("enabled start reached driver boundary; native effects refused")
	control.DriverStatusFn = func(context.Context, Resource, ResourceManifest, bool) (Status, error) {
		statusCalls++
		return Status{StatusCode: StatusCodeUnavailable}, blocked
	}
	control.DriverRunFn = func(context.Context, Resource, ResourceManifest, string, []string, io.Writer, io.Writer) error {
		driverCalls++
		return blocked
	}
	control.RunResourceCommandFn = func(context.Context, string, string, []string, io.Writer, io.Writer) error {
		cliCalls++
		return blocked
	}
	assertAbsent := func() {
		t.Helper()
		policyPath, err := openRGBDevicePolicyPath(home)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(policyPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("fixture unexpectedly has machine device policy: %s: %v", policyPath, err)
		}
		for _, path := range []string{filepath.Join(home, "storage"), filepath.Join(home, "artifacts")} {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("default startup created lighting state: %s: %v", path, err)
			}
		}
	}
	report, err := control.StartAll(io.Discard, io.Discard)
	if err != nil || statusCalls != 0 || driverCalls != 0 || cliCalls != 0 || len(report.Started) != 0 || len(report.Failed) != 0 {
		t.Fatalf("default reached effects: report=%+v status=%d driver=%d cli=%d err=%v", report, statusCalls, driverCalls, cliCalls, err)
	}
	assertAbsent()
	// Also exercise the public delegator with its ordinary production composition.
	// Its canonical disabled selection excludes the resource before any status.
	report, err = c.StartAll(io.Discard, io.Discard)
	if err != nil || len(report.Started) != 0 || len(report.Failed) != 0 {
		t.Fatalf("public default startup=%+v err=%v", report, err)
	}
	assertAbsent()
	// Separate selection transition control: opt-in reaches the observed effect
	// boundary and is deliberately refused; disable returns to zero dispatch.
	if err := c.SetEnabled("openrgb", true); err != nil {
		t.Fatal(err)
	}
	report, err = control.StartAll(io.Discard, io.Discard)
	if err != nil || statusCalls != 1 || driverCalls != 1 || cliCalls != 0 || len(report.Failed) != 1 {
		t.Fatalf("opt-in control not observable: report=%+v status=%d driver=%d cli=%d err=%v", report, statusCalls, driverCalls, cliCalls, err)
	}
	if err := c.SetEnabled("openrgb", false); err != nil {
		t.Fatal(err)
	}
	report, err = control.StartAll(io.Discard, io.Discard)
	if err != nil || statusCalls != 1 || driverCalls != 1 || cliCalls != 0 || len(report.Started) != 0 || len(report.Failed) != 0 {
		t.Fatalf("disabled startup re-entered effect boundary: report=%+v status=%d driver=%d cli=%d err=%v", report, statusCalls, driverCalls, cliCalls, err)
	}
	assertAbsent()
}
