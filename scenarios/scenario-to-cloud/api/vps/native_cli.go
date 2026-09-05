package vps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/envkit-go"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"
)

type remotePlatform struct {
	GOOS    string
	GOARCH  string
	Kernel  string
	Machine string
}

var (
	findRepoRootForCLIInstall = bundle.FindRepoRootFromCWD
	buildLocalVrooliBinaryFn  = buildLocalVrooliBinary
	detectRemotePlatformFn    = detectRemotePlatform
	execCommandForCLIInstall  = exec.Command
)

func buildInstallVrooliPlanCommand(cfg ssh.Config, workdir string) string {
	remotePath := shellutil.RemoteVrooliPath(workdir)
	remoteBinDir := shellutil.QuoteSingle(filepath.ToSlash(filepath.Dir(remotePath)))
	return ssh.LocalSSHCommand(cfg, fmt.Sprintf("mkdir -p %s", remoteBinDir)) +
		" && " +
		ssh.LocalSCPCommand(cfg, "<local-vrooli-linux-<target-arch>>", remotePath) +
		" && " +
		ssh.LocalSSHCommand(cfg, fmt.Sprintf("chmod 0755 %s", shellutil.QuotedRemoteVrooliPath(workdir)))
}

func installRemoteVrooliCLI(ctx context.Context, cfg ssh.Config, workdir string, sshRunner ssh.Runner, scpRunner ssh.SCPRunner) error {
	platform, err := detectRemotePlatformFn(ctx, cfg, sshRunner)
	if err != nil {
		return err
	}

	localPath, cleanup, err := buildLocalVrooliBinaryFn(platform)
	if err != nil {
		return err
	}
	defer cleanup()

	remoteBinDir := filepath.ToSlash(filepath.Dir(shellutil.RemoteVrooliPath(workdir)))
	if err := RunStepWithRetry(ctx, sshRunner, cfg, "install_vrooli", fmt.Sprintf("mkdir -p %s", shellutil.QuoteSingle(remoteBinDir))); err != nil {
		return err
	}
	if err := scpRunner.Copy(ctx, cfg, localPath, shellutil.RemoteVrooliPath(workdir), ssh.DefaultSCPOptions()); err != nil {
		return err
	}
	return RunStepWithRetry(ctx, sshRunner, cfg, "install_vrooli", fmt.Sprintf("chmod 0755 %s", shellutil.QuotedRemoteVrooliPath(workdir)))
}

func detectRemotePlatform(ctx context.Context, cfg ssh.Config, sshRunner ssh.Runner) (remotePlatform, error) {
	result, err := sshRunner.Run(ctx, cfg, "uname -s && uname -m", ssh.DefaultRunOptions())
	if err != nil {
		return remotePlatform{}, err
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) < 2 {
		return remotePlatform{}, fmt.Errorf("detect remote platform: expected uname output, got %q", result.Stdout)
	}

	kernel := strings.ToLower(strings.TrimSpace(lines[0]))
	machine := strings.ToLower(strings.TrimSpace(lines[1]))
	if kernel != "linux" {
		return remotePlatform{}, fmt.Errorf("unsupported remote kernel %q: scenario-to-cloud mini installs currently support linux only", kernel)
	}

	goarch, err := normalizeGOARCH(machine)
	if err != nil {
		return remotePlatform{}, err
	}
	return remotePlatform{
		GOOS:    "linux",
		GOARCH:  goarch,
		Kernel:  kernel,
		Machine: machine,
	}, nil
}

func normalizeGOARCH(machine string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(machine)) {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported remote architecture %q", machine)
	}
}

func buildLocalVrooliBinary(platform remotePlatform) (string, func(), error) {
	repoRoot, err := findRepoRootForCLIInstall()
	if err != nil {
		return "", nil, fmt.Errorf("locate vrooli repo root: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "scenario-to-cloud-vrooli-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }

	outputPath := filepath.Join(tempDir, "vrooli")
	cmd := buildLocalVrooliBinaryCommand(repoRoot, outputPath, platform)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("build native vrooli binary for %s/%s: %w\n%s", platform.GOOS, platform.GOARCH, err, strings.TrimSpace(string(out)))
	}
	return outputPath, cleanup, nil
}

// buildLocalVrooliBinaryCommand centralizes the native CLI build invocation so
// tests can verify the exact greenfield contract without shelling out.
func buildLocalVrooliBinaryCommand(repoRoot, outputPath string, platform remotePlatform) *exec.Cmd {
	cmd := execCommandForCLIInstall("go", "build", "-trimpath", "-o", outputPath, "./cmd/vrooli")
	cmd.Dir = repoRoot
	cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{
		"CGO_ENABLED=0",
		"GOOS=" + platform.GOOS,
		"GOARCH=" + platform.GOARCH,
		"GOWORK=off",
	}), envkit.ToolchainOptions{})
	return cmd
}
