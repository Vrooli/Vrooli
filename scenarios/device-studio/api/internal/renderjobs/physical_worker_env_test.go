package renderjobs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/repo-contract-go/cliinvoke"
)

func TestGeometryWorkerEnvironmentResolvesCanonicalScenarioPortWithNoAmbientEnvironment(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_GEOMETRY_DISCOVERY_CHILD") == "1" {
		url, err := discovery.ResolveScenarioURLDefault(context.Background(), "fixture-gateway")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Print(url)
		return
	}
	if runtime.GOOS != "linux" {
		t.Skip("the constrained child process fixture uses a POSIX test CLI")
	}

	root := t.TempDir()
	home := filepath.Join(root, "operator-home")
	workspace := filepath.Join(root, "attempt")
	cli := filepath.Join(root, "vrooli-fixture")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nprintf '{\"port\":19349}\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	env, err := geometryWorkerEnvironmentFor(home, workspace, cli)
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 3 {
		t.Fatalf("geometry worker environment has %d entries, want only HOME, VROOLI_BIN and TMPDIR: %v", len(env), env)
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestGeometryWorkerEnvironmentResolvesCanonicalScenarioPortWithNoAmbientEnvironment$")
	cmd.Dir = repositoryRootForTest(t)
	cmd.Env = append(env, "DEVICE_STUDIO_GEOMETRY_DISCOVERY_CHILD=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("constrained child discovery failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "http://localhost:19349") {
		t.Fatalf("canonical discovery output = %q, want the fixture gateway API URL", output)
	}
}

func repositoryRootForTest(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../.."))
}

func TestPhysicalWorkerEnvironmentPassesOnlyResolvedControlPlaneCLIAndPrivateScratch(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "operator-home")
	workspace := filepath.Join(root, "attempt")
	cli := filepath.Join(home, ".vrooli", "bin", "vrooli")
	if err := os.MkdirAll(filepath.Dir(cli), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("test cli"), 0o700); err != nil {
		t.Fatal(err)
	}
	env, err := physicalWorkerEnvironmentFor(home, workspace, "")
	if err != nil {
		t.Fatal(err)
	}
	values := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			t.Fatalf("malformed environment item %q", item)
		}
		values[key] = value
	}
	if values["HOME"] != home || values[cliinvoke.BinaryEnvVar] != cli {
		t.Fatalf("worker owner lookup = HOME %q, VROOLI_BIN %q; want %q, %q", values["HOME"], values[cliinvoke.BinaryEnvVar], home, cli)
	}
	for _, key := range []string{"TMPDIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "BLENDER_USER_CONFIG", "BLENDER_USER_SCRIPTS", "BLENDER_USER_DATAFILES", "PYTHONPYCACHEPREFIX"} {
		if !strings.HasPrefix(values[key], workspace+string(filepath.Separator)) {
			t.Errorf("%s=%q is outside private attempt workspace %q", key, values[key], workspace)
		}
	}
	if _, ok := values["VROOLI_ROOT"]; ok {
		t.Fatal("worker inherited VROOLI_ROOT; only the canonical CLI path should be forwarded")
	}
	if len(values) != 11 {
		t.Fatalf("worker received %d environment entries, want fixed allowlist of 11: %v", len(values), values)
	}
}

func TestPhysicalWorkerCLIUsesConfiguredOwnerPathAndRejectsMissingLookup(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	first := filepath.Join(root, "first-vrooli")
	second := filepath.Join(root, "second-vrooli")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("vrooli"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := resolvePhysicalWorkerCLI(home, first); err != nil || got != first {
		t.Fatalf("explicit canonical CLI path = %q, %v; want %q", got, err, first)
	}
	if _, err := resolvePhysicalWorkerCLI(home, filepath.Join(root, "missing-vrooli")); err == nil {
		t.Fatal("missing explicit/runtime CLI lookup unexpectedly succeeded")
	}
}
