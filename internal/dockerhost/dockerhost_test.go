package dockerhost

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestSanitizeDaemonConfigRemovesInvalidKeyAndPreservesSettings(t *testing.T) {
	restore := stubHostreqkit(t)
	defer restore()

	var installed string
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if installed != "" {
			return []byte(installed), nil
		}
		return []byte(`{
  "default-cgroup-parent": "workload.slice",
  "default-runtime": "nvidia",
  "dns": ["1.1.1.1"],
  "runtimes": {"nvidia": {"path": "nvidia-container-runtime", "runtimeArgs": []}}
}`), nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "dockerd" || name == "sudo" { //nolint:goconst // command fixture
			return "/usr/bin/dockerd", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("configuration OK"), nil
	}
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		installed = content
		return "/tmp/vrooli-managed-test", nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return nil
	}

	result, err := SanitizeDaemonConfig(DaemonConfigPath, ConfigOptions{}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("SanitizeDaemonConfig: %v", err)
	}
	if !result.Changed {
		t.Fatal("expected config change")
	}
	if !contains(result.RemovedInvalidKeys, "default-cgroup-parent") {
		t.Fatalf("removed invalid keys = %v", result.RemovedInvalidKeys)
	}
	if strings.Contains(installed, "default-cgroup-parent") {
		t.Fatalf("installed config still has invalid key: %s", installed)
	}
	for _, want := range []string{`"default-runtime": "nvidia"`, `"dns": [`} {
		if !strings.Contains(installed, want) {
			t.Fatalf("installed config missing %s: %s", want, installed)
		}
	}
}

func TestSanitizeDaemonConfigAppliesWorkloadPolicyWithoutDuplicateExecOpt(t *testing.T) {
	restore := stubHostreqkit(t)
	defer restore()

	var installed string
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if installed != "" {
			return []byte(installed), nil
		}
		return []byte(`{"exec-opts":["native.cgroupdriver=systemd","log-level=debug"]}`), nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "dockerd" || name == "sudo" {
			return "/usr/bin/dockerd", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("configuration OK"), nil
	}
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		installed = content
		return "/tmp/vrooli-managed-test", nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return nil
	}

	_, err := SanitizeDaemonConfig(DaemonConfigPath, ConfigOptions{ApplyWorkloadCgroupPolicy: true}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("SanitizeDaemonConfig: %v", err)
	}
	if strings.Count(installed, "native.cgroupdriver=systemd") != 1 {
		t.Fatalf("expected one cgroupdriver opt: %s", installed)
	}
	if !strings.Contains(installed, `"cgroup-parent": "workload.slice"`) {
		t.Fatalf("expected workload cgroup parent: %s", installed)
	}
}

func TestInspectHealthClassifiesDaemonUnavailable(t *testing.T) {
	restore := stubHostreqkit(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "docker" {
			return []byte("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"), errors.New("exit status 1")
		}
		return []byte("inactive"), errors.New("exit status 3")
	}

	health := InspectHealth()
	if !health.ClientInstalled || health.InfoOK {
		t.Fatalf("unexpected health: %+v", health)
	}
	if !health.DaemonUnavailable {
		t.Fatalf("expected daemon unavailable: %+v", health)
	}
}

func stubHostreqkit(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return origLookPath(name)
	}
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.WriteTempFileFn = origWriteTemp
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
