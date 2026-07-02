package vps

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"
)

type fakeSCPRunner struct {
	localPath  string
	remotePath string
	calls      int
}

func (f *fakeSCPRunner) Copy(_ context.Context, _ ssh.Config, localPath, remotePath string, _ ssh.SCPOptions) error {
	f.calls++
	f.localPath = localPath
	f.remotePath = remotePath
	return nil
}

func TestDetectRemotePlatformNormalizesLinuxArchitectures(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		wantArch string
		wantErr  bool
	}{
		{name: "amd64", stdout: "Linux\nx86_64\n", wantArch: "amd64"},
		{name: "arm64", stdout: "Linux\naarch64\n", wantArch: "arm64"},
		{name: "unsupported kernel", stdout: "Darwin\nx86_64\n", wantErr: true},
		{name: "unsupported arch", stdout: "Linux\nppc64le\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{
				results: []fakeResult{{res: ssh.Result{Stdout: tt.stdout, ExitCode: 0}}},
			}
			got, err := detectRemotePlatform(context.Background(), ssh.Config{Host: "test"}, runner)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("detectRemotePlatform() error = %v", err)
			}
			if got.GOOS != "linux" || got.GOARCH != tt.wantArch {
				t.Fatalf("detectRemotePlatform() = %+v, want linux/%s", got, tt.wantArch)
			}
		})
	}
}

func TestBuildInstallVrooliPlanCommandTargetsDeploymentBinaryPath(t *testing.T) {
	cfg := ssh.NewConfig("203.0.113.10", 22, "root", "")
	workdir := "/root/Vrooli"
	command := buildInstallVrooliPlanCommand(cfg, workdir)

	if !strings.Contains(command, ".vrooli/bin/vrooli") {
		t.Fatalf("expected install command to reference deployment-local binary path: %s", command)
	}
	if !strings.Contains(command, "scp") {
		t.Fatalf("expected install command to include scp upload: %s", command)
	}
	if !strings.Contains(command, "chmod 0755") {
		t.Fatalf("expected install command to include chmod: %s", command)
	}
}

func TestBuildInstallVrooliPlanCommandMatchesApplyOrder(t *testing.T) {
	cfg := ssh.NewConfig("203.0.113.10", 22, "root", "")
	workdir := "/root/Vrooli"
	command := buildInstallVrooliPlanCommand(cfg, workdir)

	mkdirIdx := strings.Index(command, "mkdir -p")
	scpIdx := strings.Index(command, "scp")
	chmodIdx := strings.Index(command, "chmod 0755")
	if mkdirIdx == -1 || scpIdx == -1 || chmodIdx == -1 {
		t.Fatalf("expected mkdir, scp, and chmod in plan command, got: %s", command)
	}
	if !(mkdirIdx < scpIdx && scpIdx < chmodIdx) {
		t.Fatalf("expected plan command to mirror apply order (mkdir -> scp -> chmod), got: %s", command)
	}
}

func TestInstallRemoteVrooliCLIBuildsUploadsAndMarksExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	localBinary := filepath.Join(tmpDir, "vrooli")
	if err := os.WriteFile(localBinary, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write local binary: %v", err)
	}

	originalDetect := detectRemotePlatformFn
	originalBuild := buildLocalVrooliBinaryFn
	defer func() {
		detectRemotePlatformFn = originalDetect
		buildLocalVrooliBinaryFn = originalBuild
	}()

	detectRemotePlatformFn = func(context.Context, ssh.Config, ssh.Runner) (remotePlatform, error) {
		return remotePlatform{GOOS: "linux", GOARCH: "amd64", Kernel: "linux", Machine: "x86_64"}, nil
	}
	buildLocalVrooliBinaryFn = func(remotePlatform) (string, func(), error) {
		return localBinary, func() {}, nil
	}

	runner := &fakeRunner{
		results: []fakeResult{
			{res: ssh.Result{ExitCode: 0}},
			{res: ssh.Result{ExitCode: 0}},
		},
	}
	scpRunner := &fakeSCPRunner{}

	workdir := "/root/Vrooli"
	if err := installRemoteVrooliCLI(context.Background(), ssh.NewConfig("203.0.113.10", 22, "root", ""), workdir, runner, scpRunner); err != nil {
		t.Fatalf("installRemoteVrooliCLI() error = %v", err)
	}

	if scpRunner.calls != 1 {
		t.Fatalf("scp calls = %d, want 1", scpRunner.calls)
	}
	if scpRunner.localPath != localBinary {
		t.Fatalf("uploaded local path = %q, want %q", scpRunner.localPath, localBinary)
	}
	if scpRunner.remotePath != shellutil.RemoteVrooliPath(workdir) {
		t.Fatalf("uploaded remote path = %q, want %q", scpRunner.remotePath, shellutil.RemoteVrooliPath(workdir))
	}
	if len(runner.commands) != 2 {
		t.Fatalf("runner commands = %d, want 2", len(runner.commands))
	}
	if !strings.Contains(runner.commands[0], "mkdir -p") {
		t.Fatalf("expected first command to create remote binary dir, got %q", runner.commands[0])
	}
	if !strings.Contains(runner.commands[1], "chmod 0755") || !strings.Contains(runner.commands[1], ".vrooli/bin/vrooli") {
		t.Fatalf("expected second command to chmod remote binary, got %q", runner.commands[1])
	}
}

func TestBuildLocalVrooliBinaryCommandUsesNativeGreenfieldContract(t *testing.T) {
	repoRoot := "/repo/root"
	outputPath := "/tmp/vrooli"
	platform := remotePlatform{GOOS: "linux", GOARCH: "arm64"}

	cmd := buildLocalVrooliBinaryCommand(repoRoot, outputPath, platform)

	if got, want := cmd.Dir, repoRoot; got != want {
		t.Fatalf("cmd.Dir = %q, want %q", got, want)
	}
	if len(cmd.Args) < 5 {
		t.Fatalf("cmd.Args = %v, want go build invocation", cmd.Args)
	}
	if got, want := cmd.Args[0], "go"; got != want {
		t.Fatalf("cmd.Args[0] = %q, want %q", got, want)
	}
	if got, want := strings.Join(cmd.Args[1:], " "), "build -trimpath -o "+outputPath+" ./cmd/vrooli"; got != want {
		t.Fatalf("cmd.Args[1:] = %q, want %q", got, want)
	}

	env := envMap(cmd.Env)
	if got := env["CGO_ENABLED"]; got != "0" {
		t.Fatalf("CGO_ENABLED = %q, want %q", got, "0")
	}
	if got := env["GOOS"]; got != platform.GOOS {
		t.Fatalf("GOOS = %q, want %q", got, platform.GOOS)
	}
	if got := env["GOARCH"]; got != platform.GOARCH {
		t.Fatalf("GOARCH = %q, want %q", got, platform.GOARCH)
	}
	if got := env["GOWORK"]; got != "off" {
		t.Fatalf("GOWORK = %q, want %q", got, "off")
	}
}

func TestBuildLocalVrooliBinaryCleansUpTempDirOnBuildFailure(t *testing.T) {
	originalFindRepoRoot := findRepoRootForCLIInstall
	originalExecCommand := execCommandForCLIInstall
	defer func() {
		findRepoRootForCLIInstall = originalFindRepoRoot
		execCommandForCLIInstall = originalExecCommand
	}()

	tempDirPattern := filepath.Join(os.TempDir(), "scenario-to-cloud-vrooli-*")
	beforeMatches, err := filepath.Glob(tempDirPattern)
	if err != nil {
		t.Fatalf("glob temp dirs before run: %v", err)
	}
	beforeSet := make(map[string]struct{}, len(beforeMatches))
	for _, match := range beforeMatches {
		beforeSet[match] = struct{}{}
	}

	findRepoRootForCLIInstall = func() (string, error) {
		return t.TempDir(), nil
	}
	execCommandForCLIInstall = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'simulated build failure' >&2; exit 1")
	}

	_, cleanup, err := buildLocalVrooliBinary(remotePlatform{GOOS: "linux", GOARCH: "amd64"})
	if err == nil {
		t.Fatal("expected buildLocalVrooliBinary to fail")
	}
	if cleanup != nil {
		t.Fatal("expected cleanup to be nil when build setup fails")
	}
	if !strings.Contains(err.Error(), "build native vrooli binary for linux/amd64") {
		t.Fatalf("unexpected error: %v", err)
	}

	afterMatches, globErr := filepath.Glob(tempDirPattern)
	if globErr != nil {
		t.Fatalf("glob temp dirs after run: %v", globErr)
	}
	for _, match := range afterMatches {
		if _, existed := beforeSet[match]; !existed {
			t.Fatalf("expected temp directory to be cleaned up, found leftover %s", match)
		}
	}
}

func envMap(env []string) map[string]string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	return values
}
