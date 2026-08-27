package setup

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/envkit-go"
	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/onboardinghandoff"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/shell"
)

func RunBuild(root, home string, stdout, stderr io.Writer) error {
	if err := lifecycle.ProvisionGeneratedPackages(root, home, stdout, stdout); err != nil {
		return fmt.Errorf("provision generated packages: %w", err)
	}
	buildDir := filepath.Join(config.RepoConfigDir(root), "build")
	if err := os.MkdirAll(buildDir, tuning.PermDir); err != nil {
		return err
	}

	gitCommit := "unknown"
	if output, err := shell.Output(shell.Spec{Dir: root, Name: "git", Args: []string{"rev-parse", "HEAD"}}); err == nil {
		if value := strings.TrimSpace(string(output)); value != "" {
			gitCommit = value
		}
	}
	buildTime := time.Now().UTC().Format(time.RFC3339)

	if err := buildProjectBinary(root, filepath.Join(buildDir, "vrooli-api"), "./cmd/vrooli-api", []string{"cmd/vrooli-api", "internal"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	if err := buildProjectBinary(root, filepath.Join(buildDir, "vrooli"), "./cmd/vrooli", []string{"cmd/vrooli", "internal"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	if err := buildProjectBinary(root, filepath.Join(buildDir, "vrooli-agent-launcher"), "./cmd/vrooli-agent-launcher", []string{"cmd/vrooli-agent-launcher", "packages/cli-core"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	if err := buildNestedModuleBinary(filepath.Join(root, "cmd", "vrooli-policy-runner"), filepath.Join(buildDir, "vrooli-policy-runner"), stdout, stderr); err != nil {
		return err
	}
	return nil
}

func RunDevelopWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	return newSetupService(defaultSetupDeps(root)).RunDevelopWithOptions(root, home, opts, stdout, stderr)
}

func (s *setupService) RunDevelopWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	if err := s.deps.currentHost().ValidateDevelop(); err != nil {
		return err
	}

	projectScenario, err := s.deps.loadProject(root)
	if err != nil {
		return err
	}

	if setupNeeded(home, root, projectScenario.Slug) {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Running setup before develop")
		if err := s.RunSetupWithOptions(root, home, opts, stdout, stderr); err != nil {
			return err
		}
	}

	manager, err := s.deps.newPortsManager(root, home)
	if err != nil {
		return err
	}
	projectEnv, err := manager.BuildProjectEnvironment(projectScenario)
	if err != nil {
		return err
	}
	if err := s.applyDotEnv(root); err != nil {
		return err
	}
	overlay := make(envkit.Env, 0, len(projectEnv.EnvVars))
	for key, value := range projectEnv.EnvVars {
		overlay = append(overlay, key+"="+value)
	}
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.ForeignScenario, overlay)
	apiPort := resolveAPIPort(projectEnv.EnvVars)
	if apiPort <= 0 {
		apiPort = defaultAPIPort
	}

	healthy, err := apiAlreadyHealthy(apiPort)
	if err != nil {
		return err
	}
	healthTimeout := tuning.StandardOperationTimeout
	if !healthy {
		spec, err := buildAPILaunchSpec(root, home, env, apiPort)
		if err != nil {
			return err
		}
		healthTimeout = developHealthTimeout(spec)
		if err := s.deps.startProjectAPI(root, spec, stdout, stderr); err != nil {
			return err
		}
	}

	if err := s.deps.healthCheck(apiPort, healthTimeout); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "🚀 Vrooli API healthy on port %d with native scenario management\n", apiPort)
	if !configurationAlreadyComplete(home, root) {
		developOpts := opts
		if developOpts.Onboarding == "" {
			developOpts.Onboarding = onboardinghandoff.ModeURL
		}
		if _, handoffErr := s.runOnboardingHandoff(root, home, developOpts, stdout, stderr); handoffErr != nil {
			_, _ = fmt.Fprintf(stderr, "[WARN]    Onboarding handoff unavailable: %v\n", handoffErr)
		}
	}
	if strings.EqualFold(strings.TrimSpace(opts.Scenarios), "none") {
		_, _ = fmt.Fprintln(stdout, "Vrooli orchestrator skipped (--scenarios none)")
		return nil
	}
	return s.deps.startOrchestrator(root, home, stdout, stderr)
}

func developHealthTimeout(spec apiLaunchSpec) time.Duration {
	// A release install intentionally carries the bootstrap CLI, not a second
	// project API binary. The first develop therefore falls back to `go run`,
	// which may need to download the pinned toolchain and module graph on a
	// genuinely fresh host. Keep the normal fast-start budget for prebuilt
	// launchers while giving that one cold path enough deterministic runway.
	if filepath.Base(spec.Command) == "go" && len(spec.Args) > 0 && spec.Args[0] == "run" {
		return tuning.ExtendedOperationTimeout
	}
	return tuning.StandardOperationTimeout
}

func buildProjectBinary(root, outputPath, target string, fingerprintPaths []string, gitCommit, buildTime string, stdout, stderr io.Writer) error {
	fingerprint, err := buildinfo.ComputeSourceFingerprintForPaths(root, fingerprintPaths...)
	if err != nil {
		return err
	}

	ldflags := fmt.Sprintf(
		"-s -w -X %s.GitCommit=%s -X %s.BuildTime=%s -X %s.Fingerprint=%s",
		"github.com/vrooli/vrooli/internal/buildinfo",
		gitCommit,
		"github.com/vrooli/vrooli/internal/buildinfo",
		buildTime,
		"github.com/vrooli/vrooli/internal/buildinfo",
		fingerprint,
	)

	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"CGO_ENABLED=0"})
	return shell.Run(shell.Spec{
		Name:   "go",
		Args:   []string{"build", "-trimpath", "-ldflags", ldflags, "-o", outputPath, target},
		Dir:    root,
		Env:    env,
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  os.Stdin,
	})
}

// buildNestedModuleBinary builds a binary whose source lives in its own Go
// module beneath the repo root.
//
// vrooli-policy-runner is deliberately a separate module: it is the process
// boundary for native coding-agent hooks, runs on every Bash tool call, and
// must not drag in the main module's dependency graph. That isolation means it
// cannot be built with `go build ./cmd/...` from the repo root, so it builds
// from its own directory and does not receive the main module's buildinfo
// ldflags (those -X symbols do not exist in this module).
func buildNestedModuleBinary(moduleDir, outputPath string, stdout, stderr io.Writer) error {
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"CGO_ENABLED=0"})
	return shell.Run(shell.Spec{
		Name:   "go",
		Args:   []string{"build", "-trimpath", "-o", outputPath, "."},
		Dir:    moduleDir,
		Env:    env,
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  os.Stdin,
	})
}

func (s *setupService) applyDotEnv(root string) error {
	values, err := s.deps.loadDotEnv(filepath.Join(root, ".env"))
	if err != nil {
		return err
	}
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func loadDotEnv(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values, scanner.Err()
}

func resolveAPIPort(values map[string]string) int {
	if raw := strings.TrimSpace(os.Getenv("VROOLI_API_PORT")); raw != "" {
		if port, err := strconv.Atoi(raw); err == nil && port > 0 {
			return port
		}
	}
	if raw := strings.TrimSpace(values["VROOLI_API_PORT"]); raw != "" {
		if port, err := strconv.Atoi(raw); err == nil && port > 0 {
			return port
		}
	}
	return defaultAPIPort
}

func apiAlreadyHealthy(port int) (bool, error) {
	client := &http.Client{Timeout: tuning.ShortOperationDeadline}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 300, nil
}

func buildAPILaunchSpec(root, home string, env []string, port int) (apiLaunchSpec, error) {
	logsDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyLogs)
	if err != nil {
		return apiLaunchSpec{}, err
	}
	binDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return apiLaunchSpec{}, err
	}
	logFile := filepath.Join(logsDir, "vrooli-api.log")
	for _, candidate := range []struct {
		command string
		args    []string
	}{
		{command: filepath.Join(config.RepoConfigDir(root), "build", "vrooli-api")},
		{command: filepath.Join(binDir, "vrooli-api")},
		{command: "go", args: []string{"run", "./cmd/vrooli-api"}},
	} {
		if candidate.command == "go" {
			if _, err := exec.LookPath("go"); err != nil {
				continue
			}
		} else if _, err := os.Stat(candidate.command); err != nil {
			continue
		}
		return apiLaunchSpec{
			Command: candidate.command,
			Args:    candidate.args,
			LogFile: logFile,
			Env:     env,
			Port:    port,
		}, nil
	}
	return apiLaunchSpec{}, fmt.Errorf("no project-level vrooli-api launcher found")
}

func startProjectAPI(root string, spec apiLaunchSpec, stdout, stderr io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(spec.LogFile), tuning.PermDir); err != nil {
		return err
	}
	logFile, err := os.OpenFile(spec.LogFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, tuning.PermFile)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Dir = root
	cmd.Env = spec.Env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(tuning.SetupFilesystemSettleDelay)
	if !platform.IsPIDRunning(cmd.Process.Pid) {
		return fmt.Errorf("vrooli-api exited immediately: %w", err)
	}
	return nil
}

func waitForHTTPHealth(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: tuning.ShortOperationDeadline}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	for {
		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("vrooli-api failed health check on port %d", port)
		}
		time.Sleep(tuning.ShortOperationTimeout)
	}
}

func startOrchestrator(root, home string, stdout, stderr io.Writer) error {
	service := orchestrator.New(root, home, stdout, stderr)
	status, exists, err := service.Status("vrooli-orchestrator")
	if err == nil && exists && status.Processes > 0 {
		return nil
	}
	_, err = service.Start("vrooli-orchestrator", lifecycle.StartOptions{})
	return err
}
