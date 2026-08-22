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
