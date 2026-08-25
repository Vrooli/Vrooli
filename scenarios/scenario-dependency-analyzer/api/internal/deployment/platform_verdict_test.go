package deployment

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListPlatformVerdictsDerivesBlockingResourceForEveryHostOS(t *testing.T) {
	root := t.TempDir()
	scenariosDir := filepath.Join(root, "scenarios")
	resourceDir := filepath.Join(root, "resources", "linux-only")
	if err := os.MkdirAll(filepath.Join(scenariosDir, "sample", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	service := []byte(`{"service":{"name":"sample"},"dependencies":{"resources":{"linux-only":{"required":true,"enabled":true}}}}`)
	resource := []byte(`{"name":"linux-only","platforms":{"linux":"supported","macos":"unsupported","windows":"unsupported"},"requirements":{"class":"small","weight":1,"source":"estimated","confidence":"low"},"deployment":{"profiles":{}}}`)
	if err := os.WriteFile(filepath.Join(scenariosDir, "sample", ".vrooli", "service.json"), service, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), resource, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ListPlatformVerdicts(scenariosDir, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || len(result[0].Platforms) != 3 {
		t.Fatalf("got %#v, want one scenario with three platform verdicts", result)
	}
	for _, platform := range result[0].Platforms {
		t.Logf("platform=%#v", platform)
		if platform.HostOS == "linux" && platform.Status != "eligible" {
			t.Errorf("linux status = %q, want eligible", platform.Status)
		}
		if platform.HostOS != "linux" && (platform.Status != "blocked" || platform.BlockingDependency != "linux-only") {
			t.Errorf("%s verdict = %#v, want blocked by linux-only", platform.HostOS, platform)
		}
	}
}

func TestListPlatformVerdictsAuthoredCapabilityOverridesDependencyDerivation(t *testing.T) {
	root := t.TempDir()
	scenariosDir := filepath.Join(root, "scenarios")
	resourceDir := filepath.Join(root, "resources", "windows-only")
	if err := os.MkdirAll(filepath.Join(scenariosDir, "sample", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	service := []byte(`{"service":{"name":"sample","platform_capabilities":{"sample-api":{"linux":{"status":"supported","mechanism":"fixture","evidence":"test"}}}},"dependencies":{"resources":{"windows-only":{"required":true,"enabled":true}}}}`)
	resource := []byte(`{"name":"windows-only","platforms":{"linux":"unsupported","macos":"unsupported","windows":"supported"},"requirements":{"class":"small","weight":1,"source":"estimated","confidence":"low"},"deployment":{"profiles":{}}}`)
	if err := os.WriteFile(filepath.Join(scenariosDir, "sample", ".vrooli", "service.json"), service, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), resource, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ListPlatformVerdicts(scenariosDir, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || !result[0].Overridden {
		t.Fatalf("got %#v, want an overridden scenario verdict", result)
	}
	if got := result[0].Platforms[0]; got.Status != "supported" || !got.Overridden || got.Derived {
		t.Fatalf("linux override = %#v, want supported, overridden, non-derived", got)
	}
}

func TestAuthoredUnsupportedCapabilityDegradesUnlessEssential(t *testing.T) {
	root := t.TempDir()
	scenariosDir := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(filepath.Join(scenariosDir, "sample", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := []byte(`{"service":{"name":"sample","platform_capabilities":{"optional-a":{"macos":{"status":"unsupported","mechanism":"fixture"}},"essential-b":{"macos":{"status":"unsupported","mechanism":"fixture","essential":true}},"optional-c":{"macos":{"status":"unsupported","mechanism":"fixture"}}}}}`)
	if err := os.WriteFile(filepath.Join(scenariosDir, "sample", ".vrooli", "service.json"), service, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ListPlatformVerdicts(scenariosDir, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, platform := range result[0].Platforms {
		if platform.HostOS == "macos" {
			if platform.Status != "blocked" || platform.BlockingDependency != "capability:essential-b" {
				t.Fatalf("macOS verdict = %#v, want essential capability block", platform)
			}
			if platform.Reason != "essential authored capabilities unsupported on macos: essential-b, optional-a, optional-c" {
				t.Fatalf("macOS reason = %q, want every unsupported capability named", platform.Reason)
			}
		}
	}
}
