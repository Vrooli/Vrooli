package docker

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestInspectHealthyDaemonSatisfiesRequirement(t *testing.T) {
	restore := stubHostreqkit(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("ok"), nil
	}

	status := NewHandler(testManifest()).Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "docker", Kind: hostreqspec.KindTool, Required: true})
	if !status.Installed {
		t.Fatalf("Installed = false, notes = %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectDeadDaemonIsUnsatisfiedAndNeedsSudo(t *testing.T) {
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

	status := NewHandler(testManifest()).Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "docker", Kind: hostreqspec.KindTool, Required: true})
	if status.Installed {
		t.Fatalf("Installed = true, want false")
	}
	if status.BlockingReason != hostreqkit.BlockingNeedsSudo {
		t.Fatalf("BlockingReason = %q", status.BlockingReason)
	}
	if !noteContains(status.Notes, "Docker CLI is installed but daemon verification failed") {
		t.Fatalf("notes = %v", status.Notes)
	}
}

func TestInspectPermissionDeniedRequiresManualAccessFix(t *testing.T) {
	restore := stubHostreqkit(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "docker" {
			return []byte("permission denied while trying to connect to the Docker daemon socket"), errors.New("exit status 1")
		}
		return []byte("active"), nil
	}

	status := NewHandler(testManifest()).Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "docker", Kind: hostreqspec.KindTool, Required: true})
	if status.BlockingReason != hostreqkit.BlockingManual {
		t.Fatalf("BlockingReason = %q", status.BlockingReason)
	}
	if !noteContains(status.Notes, "docker group") {
		t.Fatalf("notes = %v", status.Notes)
	}
}

func TestApplyDeadDaemonRepairsAndVerifies(t *testing.T) {
	restore := stubHostreqkit(t)
	defer restore()

	dockerInfoOK := false
	commands := []string{}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return []byte(`{"default-cgroup-parent":"workload.slice"}`), nil
	}
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		return "/tmp/vrooli-managed-test", nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "docker" {
			if dockerInfoOK {
				return []byte("ok"), nil
			}
			return []byte("Cannot connect to the Docker daemon"), errors.New("exit status 1")
		}
		if name == "dockerd" {
			return []byte("configuration OK"), nil
		}
		if name == "systemctl" && len(args) > 0 && args[0] == "is-enabled" {
			return []byte("disabled"), errors.New("exit status 1")
		}
		return []byte("inactive"), errors.New("exit status 3")
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if strings.Join(args, " ") == "start docker" {
			dockerInfoOK = true
		}
		return nil
	}

	initial := NewHandler(testManifest()).Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "docker", Kind: hostreqspec.KindTool, Required: true})
	status, err := NewHandler(testManifest()).Apply(linuxHost(), initial, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !status.Installed {
		t.Fatalf("Installed = false, notes = %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	if status.BlockingReason != hostreqkit.BlockingNone {
		t.Fatalf("BlockingReason = %q", status.BlockingReason)
	}
	if noteContains(status.Notes, "Re-run as `sudo vrooli setup`") || noteContains(status.Notes, "daemon verification failed") {
		t.Fatalf("stale failure notes were retained: %v", status.Notes)
	}
	for _, want := range []string{"systemctl daemon-reload", "systemctl reset-failed docker", "systemctl start docker"} {
		if !containsCommand(commands, want) {
			t.Fatalf("commands = %v, missing %q", commands, want)
		}
	}
}

func testManifest() hostreqkit.ToolManifest {
	return hostreqkit.ToolManifest{
		Name:           "docker",
		Commands:       []string{"docker"},
		VersionArgs:    []string{"--version"},
		DefaultPackage: "docker.io",
		InstallHint:    "Install Docker Engine or Docker Desktop",
	}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt", SupportsSystemd: true, SupportsSetup: true, SupportsDevelop: true}
}

func stubHostreqkit(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origWriteTemp := hostreqkit.WriteTempFileFn
	origRoot := hostreqkit.RunningAsRootFn
	hostreqkit.RunningAsRootFn = func() bool { return true }
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.WriteTempFileFn = origWriteTemp
		hostreqkit.RunningAsRootFn = origRoot
	}
}

func noteContains(notes []string, want string) bool {
	for _, note := range notes {
		if strings.Contains(note, want) {
			return true
		}
	}
	return false
}

func containsCommand(commands []string, want string) bool {
	for _, command := range commands {
		if command == want {
			return true
		}
	}
	return false
}
