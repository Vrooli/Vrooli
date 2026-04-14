package setup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	defaultEnvironment = "development"
	defaultTarget      = "docker"
	defaultLocation    = "Local"
	defaultAPIPort     = 8092
)

type Options struct {
	DryRun      bool
	SudoMode    string
	Environment string
	Resources   string
	Scenarios   string
	Yes         string
}

type apiLaunchSpec struct {
	Command string
	Args    []string
	LogFile string
	Env     []string
	Port    int
}

var (
	currentHostFn             = vrooliruntime.Current
	loadProjectFn             = project.LoadProject
	markCompleteFn            = markComplete
	resolveHostRequirementsFn = hostreq.Resolve
	inspectRequirementsFn     = vrooliruntime.InspectRequirements
	ensureRequirementsFn      = vrooliruntime.EnsureRequirements
	newPortsManagerFn         = func(root, home string) (*ports.Manager, error) {
		return ports.NewManager(root, home)
	}
	startProjectAPIFn   = startProjectAPI
	startOrchestratorFn = startOrchestrator
	healthCheckFn       = waitForHTTPHealth
	loadDotEnvFn        = loadDotEnv
	nowFn               = time.Now
)

func RunSetupWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	if err := currentHostFn().ValidateSetup(); err != nil {
		return err
	}

	projectScenario, err := loadProjectFn(root)
	if err != nil {
		return err
	}

	restoreEnv, err := applyEnvironment(root, projectScenario.ServicePath, opts)
	if err != nil {
		return err
	}
	defer restoreEnv()

	if !opts.DryRun {
		if err := ensureProjectFilesystem(root, home); err != nil {
			return err
		}
	}
	requirements, err := resolveHostRequirementsFn(root, home, hostreq.ResolveOptions{
		Environment: opts.Environment,
		When:        "setup",
		Resources:   opts.Resources,
		Scenarios:   opts.Scenarios,
		Platform:    hostreq.CurrentPlatform(),
	})
	if err != nil {
		return err
	}
	planReport, err := inspectRequirementsFn(opts.Environment, requirements)
	if err != nil {
		return err
	}
	renderSetupRequirementPlan(stdout, opts, planReport)

	report, ensureErr := ensureRequirementsFn(vrooliruntime.EnsureOptions{
		Environment: opts.Environment,
		SudoMode:    opts.SudoMode,
		DryRun:      opts.DryRun,
		AutoInstall: true,
		Stdout:      stdout,
		Stderr:      stderr,
	}, requirements)
	renderSetupRequirementResult(stdout, opts, report)
	if ensureErr != nil && !opts.DryRun {
		return ensureErr
	}
	if opts.DryRun {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Dry-run mode skips git configuration, resource installation, and setup completion markers")
		return nil
	}
	if err := configureGit(root); err != nil {
		return err
	}
	if err := maybeInstallResources(root, home, opts, stdout, stderr); err != nil {
		return err
	}
	return markCompleteFn(root)
}

func RunBuild(root, home string, stdout, stderr io.Writer) error {
	if err := os.MkdirAll(filepath.Join(root, ".vrooli", "build"), 0o755); err != nil {
		return err
	}

	gitCommit := "unknown"
	if output, err := shell.Output(shell.Spec{Dir: root, Name: "git", Args: []string{"rev-parse", "HEAD"}}); err == nil {
		if value := strings.TrimSpace(string(output)); value != "" {
			gitCommit = value
		}
	}
	buildTime := time.Now().UTC().Format(time.RFC3339)

	if err := buildProjectBinary(root, filepath.Join(root, ".vrooli", "build", "vrooli-api"), "./cmd/vrooli-api", []string{"cmd/vrooli-api", "internal"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	if err := buildProjectBinary(root, filepath.Join(root, ".vrooli", "build", "vrooli"), "./cmd/vrooli", []string{"cmd/vrooli", "internal"}, gitCommit, buildTime, stdout, stderr); err != nil {
		return err
	}
	return nil
}

func RunDevelopWithOptions(root, home string, opts Options, stdout, stderr io.Writer) error {
	if err := currentHostFn().ValidateDevelop(); err != nil {
		return err
	}

	projectScenario, err := loadProjectFn(root)
	if err != nil {
		return err
	}

	restoreEnv, err := applyEnvironment(root, projectScenario.ServicePath, opts)
	if err != nil {
		return err
	}
	defer restoreEnv()

	if setupNeeded(root, projectScenario.Slug) {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Running setup before develop")
		if err := RunSetupWithOptions(root, home, opts, stdout, stderr); err != nil {
			return err
		}
	}

	manager, err := newPortsManagerFn(root, home)
	if err != nil {
		return err
	}
	projectEnv, err := manager.BuildProjectEnvironment(projectScenario)
	if err != nil {
		return err
	}
	if err := applyDotEnv(root); err != nil {
		return err
	}
	env := mergeEnvironment(os.Environ(), projectEnv.EnvVars)
	apiPort := resolveAPIPort(projectEnv.EnvVars)
	if apiPort <= 0 {
		apiPort = defaultAPIPort
	}

	healthy, err := apiAlreadyHealthy(apiPort)
	if err != nil {
		return err
	}
	if !healthy {
		spec, err := buildAPILaunchSpec(root, home, env, apiPort)
		if err != nil {
			return err
		}
		if err := startProjectAPIFn(root, spec, stdout, stderr); err != nil {
			return err
		}
	}

	if err := healthCheckFn(apiPort, 30*time.Second); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "🚀 Vrooli API healthy on port %d with native scenario management\n", apiPort)
	return startOrchestratorFn(root, home, stdout, stderr)
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

	env := append([]string(nil), os.Environ()...)
	env = append(env, "CGO_ENABLED=0")
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

type envSnapshot struct {
	value   string
	existed bool
}

func applyEnvironment(root, servicePath string, opts Options) (func(), error) {
	changes := map[string]envSnapshot{}
	set := func(key, value string, onlyIfUnset bool) error {
		current, existed := os.LookupEnv(key)
		if onlyIfUnset && existed && strings.TrimSpace(current) != "" {
			return nil
		}
		if _, tracked := changes[key]; !tracked {
			changes[key] = envSnapshot{value: current, existed: existed}
		}
		return os.Setenv(key, value)
	}

	if err := set("APP_ROOT", root, false); err != nil {
		return nil, err
	}
	if err := set("SERVICE_JSON_PATH", servicePath, false); err != nil {
		return nil, err
	}
	if err := set("TARGET", defaultTarget, true); err != nil {
		return nil, err
	}
	if err := set("LOCATION", defaultLocation, true); err != nil {
		return nil, err
	}
	if opts.Environment != "" {
		if err := set("ENVIRONMENT", opts.Environment, false); err != nil {
			return nil, err
		}
	} else if err := set("ENVIRONMENT", defaultEnvironment, true); err != nil {
		return nil, err
	}
	if opts.Resources != "" {
		if err := set("RESOURCES", opts.Resources, false); err != nil {
			return nil, err
		}
	}
	if opts.Scenarios != "" {
		if err := set("SCENARIOS", opts.Scenarios, false); err != nil {
			return nil, err
		}
	}
	if opts.Yes != "" {
		if err := set("YES", opts.Yes, false); err != nil {
			return nil, err
		}
	}
	if opts.SudoMode != "" {
		if err := set("SUDO_MODE", opts.SudoMode, false); err != nil {
			return nil, err
		}
		if err := set("SUDO_MODE_EXPLICIT", opts.SudoMode, false); err != nil {
			return nil, err
		}
	}
	if opts.DryRun {
		if err := set("DRY_RUN", "true", false); err != nil {
			return nil, err
		}
	}

	return func() {
		for key, snapshot := range changes {
			if snapshot.existed {
				_ = os.Setenv(key, snapshot.value)
				continue
			}
			_ = os.Unsetenv(key)
		}
	}, nil
}

func ensureProjectFilesystem(root, home string) error {
	paths := []string{
		filepath.Join(root, "data"),
		filepath.Join(root, ".vrooli", "build"),
		filepath.Join(root, ".vrooli", "logs"),
		filepath.Join(home, ".vrooli", "bin"),
		filepath.Join(home, ".vrooli", "logs"),
		filepath.Join(home, ".vrooli", "processes"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return makeShellScriptsExecutable(root)
}

func makeShellScriptsExecutable(root string) error {
	dirs := []string{filepath.Join(root, "scripts"), filepath.Join(root, "cli"), filepath.Join(root, "api")}
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".sh" {
				return walkErr
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			mode := info.Mode()
			if mode&0o111 != 0o111 {
				if err := os.Chmod(path, mode|0o755); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return nil
}

func configureGit(root string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return nil
	}
	cmd := exec.Command("git", "config", "core.filemode", "false")
	cmd.Dir = root
	return cmd.Run()
}

func maybeInstallResources(root, home string, opts Options, stdout, stderr io.Writer) error {
	selection := strings.TrimSpace(opts.Resources)
	if selection == "" || selection == "none" {
		return nil
	}

	controller := resourcesController(root, home)
	if selection == "enabled" {
		names, err := enabledResourceNames(root)
		if err != nil {
			return err
		}
		for _, name := range names {
			if err := controller.Run(name, []string{"install"}, stdout, stderr); err != nil {
				return err
			}
		}
		return nil
	}

	for _, raw := range strings.Split(selection, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if err := controller.Run(name, []string{"install"}, stdout, stderr); err != nil {
			return err
		}
	}
	return nil
}

type resourceRunner interface {
	Run(name string, args []string, stdout, stderr io.Writer) error
}

var resourcesController = func(root, home string) resourceRunner {
	return resources.NewController(root, home)
}

func enabledResourceNames(root string) ([]string, error) {
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	manifest, err := scenario.ReadService(servicePath)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(manifest.Dependencies.Resources))
	for name, dependency := range manifest.Dependencies.Resources {
		if dependency.Enabled {
			names = append(names, name)
		}
	}
	return names, nil
}

func setupNeeded(root, slug string) bool {
	if forceSetupApplies(slug) {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "data", ".setup-complete"))
	return err != nil
}

func applyDotEnv(root string) error {
	values, err := loadDotEnvFn(filepath.Join(root, ".env"))
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

func mergeEnvironment(base []string, overlay map[string]string) []string {
	env := append([]string(nil), base...)
	for key, value := range overlay {
		prefix := key + "="
		replaced := false
		for index, entry := range env {
			if strings.HasPrefix(entry, prefix) {
				env[index] = prefix + value
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, prefix+value)
		}
	}
	return env
}

func resolveAPIPort(values map[string]string) int {
	if raw := strings.TrimSpace(values["VROOLI_API_PORT"]); raw != "" {
		if port, err := strconv.Atoi(raw); err == nil && port > 0 {
			return port
		}
	}
	if raw := strings.TrimSpace(os.Getenv("VROOLI_API_PORT")); raw != "" {
		if port, err := strconv.Atoi(raw); err == nil && port > 0 {
			return port
		}
	}
	return defaultAPIPort
}

func apiAlreadyHealthy(port int) (bool, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 300, nil
}

func buildAPILaunchSpec(root, home string, env []string, port int) (apiLaunchSpec, error) {
	logFile := filepath.Join(home, ".vrooli", "logs", "vrooli-api.log")
	for _, candidate := range []struct {
		command string
		args    []string
	}{
		{command: filepath.Join(root, ".vrooli", "build", "vrooli-api")},
		{command: filepath.Join(home, ".vrooli", "bin", "vrooli-api")},
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
	if err := os.MkdirAll(filepath.Dir(spec.LogFile), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(spec.LogFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Dir = root
	cmd.Env = spec.Env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = detachedProcessAttr()
	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("vrooli-api exited immediately: %w", err)
	}
	return nil
}

func waitForHTTPHealth(port int, timeout time.Duration) error {
	deadline := nowFn().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	for {
		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return nil
			}
		}
		if nowFn().After(deadline) {
			return fmt.Errorf("vrooli-api failed health check on port %d", port)
		}
		time.Sleep(1 * time.Second)
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

func forceSetupApplies(slug string) bool {
	if strings.ToLower(strings.TrimSpace(os.Getenv("FORCE_SETUP"))) != "true" {
		return false
	}
	target := strings.TrimSpace(os.Getenv("FORCE_SETUP_SCENARIO"))
	return target == "" || target == slug
}

func markComplete(root string) error {
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	payload := map[string]any{
		"setup_version": "2.0.0",
		"completed_at":  time.Now().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dataDir, ".setup-complete"), data, 0o644); err != nil {
		return err
	}
	return nil
}
