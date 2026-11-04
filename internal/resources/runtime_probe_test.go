package resources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestRuntimeProbeReusesDeclaredWheelsEnvironment(t *testing.T) {
	root := t.TempDir()
	c := NewController(root, filepath.Join(root, "home"))
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	artifact := resourcedeployment.ServiceArtifact{Layout: "dir"}
	path := filepath.Join(root, "artifact")
	manifest.ManagedService.Environment = map[string]string{
		"PYTHONPATH":              "${RESOURCE_ARTIFACT_DIR}/source:${RESOURCE_ARTIFACT_DIR}/wheels",
		"PRIVATE_TEST_CREDENTIAL": "must-not-be-exported",
	}
	env, err := managedServiceRuntimeEnvironment(context.Background(), c, manifest, path, artifact)
	if err != nil {
		t.Fatal(err)
	}
	resolved := managedServiceEnvValues(env)
	want := path + "/source:" + path + "/wheels"
	if resolved["PYTHONPATH"] != want {
		t.Fatalf("managed wheels omitted: got %q want %q", resolved["PYTHONPATH"], want)
	}
	spec, err := runtimeProbeSpec("/usr/bin/bwrap", path+"/runtime/bin/python", env, []string{"numpy", "package.submodule"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(spec.Args, "\x00")
	for _, required := range []string{want, "--unshare-net", "--ro-bind", "--clearenv", "PYTHONDONTWRITEBYTECODE"} {
		if !strings.Contains(joined, required) {
			t.Errorf("probe missing %q", required)
		}
	}
	if strings.Contains(joined, "must-not-be-exported") || strings.Contains(joined, "PRIVATE_TEST_CREDENTIAL") {
		t.Fatal("probe exported application credentials")
	}
	if spec.Limits.WallTime <= 0 || spec.Limits.MemoryMaxBytes <= 0 || spec.Limits.TasksMax <= 0 || spec.Env == nil {
		t.Fatal("probe lost mandatory native ceilings or explicit environment")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("environment resolution wrote artifact/configuration: %v", err)
	}
}

func TestRuntimeProbeRejectsExecutableInputAndPreservesReceipts(t *testing.T) {
	for _, imports := range [][]string{nil, {"numpy; import os"}, {"../numpy"}, {"numpy\n"}, make([]string, 17)} {
		if _, err := runtimeProbeSpec("bwrap", "python", nil, imports); err == nil {
			t.Fatalf("unsafe or unbounded import accepted: %q", imports)
		}
	}
	path := filepath.Join(t.TempDir(), "receipt.json")
	if err := writeRuntimeProbeJSON(path, map[string]string{"status": "original"}); err != nil {
		t.Fatal(err)
	}
	if err := writeRuntimeProbeJSON(path, map[string]string{"status": "replacement"}); err == nil {
		t.Fatal("existing receipt overwritten")
	}
	bytes, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(bytes), "original") {
		t.Fatalf("original receipt changed: %s %v", bytes, err)
	}
}
