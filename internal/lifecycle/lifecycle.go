package lifecycle

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/ports"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

type Runner struct {
	Root   string
	Home   string
	Out    io.Writer
	Err    io.Writer
	Ports  *ports.Manager
	Logger *slog.Logger
}

type StartOptions struct {
	CustomPath         string
	CleanStale         bool
	BestEffort         bool
	ForceSetup         bool
	ForceSetupScenario string
}

type StopOptions struct {
	CustomPath string
}

type PhaseOptions struct {
	CustomPath              string
	Args                    []string
	AllowSkipMissingRuntime bool
	ManageRuntime           bool
	ProjectMode             bool
}

type Result struct {
	Scenario           scenario.Scenario
	AllocatedPorts     map[string]int
	Health             string
	FailedDependencies []string
	AlreadyRunning     bool
}

func NewRunner(root, home string, stdout, stderr io.Writer, logger ...*slog.Logger) (*Runner, error) {
	manager, err := ports.NewManager(root, home)
	if err != nil {
		return nil, err
	}
	baseLogger := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		baseLogger = logger[0]
	}
	return &Runner{
		Root:   filepath.Clean(root),
		Home:   filepath.Clean(home),
		Out:    stdout,
		Err:    stderr,
		Ports:  manager,
		Logger: logx.WithSubsystem(baseLogger, "lifecycle"),
	}, nil
}

func (r *Runner) Start(name string, opts StartOptions) (Result, error) {
	r.logInfo("Scenario start requested",
		logx.AttrScenario, name,
		"best_effort", opts.BestEffort,
		"clean_stale", opts.CleanStale,
		"force_setup", opts.ForceSetup,
	)
	ready := make(map[string]struct{})
	result, err := r.startWithState(name, opts, ready, nil)
	if err != nil {
		r.logError("Scenario start failed", err, logx.AttrScenario, name)
		return Result{}, err
	}
	r.logInfo("Scenario start completed",
		logx.AttrScenario, name,
		logx.AttrStatus, result.Health,
		"already_running", result.AlreadyRunning,
		"failed_dependencies", len(result.FailedDependencies),
	)
	return result, nil
}

func (r *Runner) startWithState(name string, opts StartOptions, ready map[string]struct{}, stack []string) (Result, error) {
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		return Result{}, err
	}
	return r.startScenario(item, opts, ready, stack)
}

func (r *Runner) startScenario(item scenario.Scenario, opts StartOptions, ready map[string]struct{}, stack []string) (Result, error) {
	if opts.CleanStale && len(stack) == 0 {
		r.logDebug("Cleaning stale port locks before scenario start", logx.AttrScenario, item.Slug)
		if err := r.Ports.CleanStaleLocks(); err != nil {
			return Result{}, err
		}
	}

	failedDeps, err := r.ensureDependencies(item, opts, ready, append(stack, item.Slug))
	if err != nil {
		return Result{}, err
	}

	_ = os.Remove(process.ScenarioDegradedPath(r.Home, item.Slug))

	records, err := readScenarioRecords(r.Home, item.Slug)
	if err != nil {
		return Result{}, err
	}
	runtime := process.SummarizeScenario(item.Slug, records)
	forceSetup := opts.ForceSetup && (opts.ForceSetupScenario == "" || opts.ForceSetupScenario == item.Slug)
	if runtime.ProcessCount > 0 {
		strictHealthy := r.isScenarioHealthyStrict(item, runtime.Records)
		setupNeeded, _, setupErr := r.SetupNeeded(item, forceSetup)
		if setupErr != nil {
			return Result{}, setupErr
		}
		if strictHealthy && !setupNeeded {
			currentPorts := r.runtimePorts(item.Manifest, runtime.Records)
			health := scenario.EvaluateHealth(item.Manifest.HealthConfig(), currentPorts)
			r.logInfo("Scenario already running and healthy",
				logx.AttrScenario, item.Slug,
				logx.AttrStatus, health,
				logx.AttrProcesses, runtime.ProcessCount,
			)
			return Result{
				Scenario:           item,
				AllocatedPorts:     currentPorts,
				Health:             health,
				FailedDependencies: failedDeps,
				AlreadyRunning:     true,
			}, nil
		}
		if err := r.Stop(item.Slug, StopOptions{}); err != nil {
			return Result{}, err
		}
		time.Sleep(1 * time.Second)
	}

	env, err := r.Ports.BuildEnvironment(item, nil)
	if err != nil {
		return Result{}, err
	}

	if err := r.runWithLifecycleLog(item.Slug, func(logWriter io.Writer) error {
		return r.ensureScenarioDatabase(item, env.EnvVars, logWriter)
	}); err != nil {
		return Result{}, err
	}

	setupNeeded, _, err := r.SetupNeeded(item, forceSetup)
	if err != nil {
		return Result{}, err
	}

	if setupNeeded {
		r.logInfo("Executing setup phase for scenario", logx.AttrScenario, item.Slug, logx.AttrPhase, "setup")
		if err := r.runWithLifecycleLog(item.Slug, func(logWriter io.Writer) error {
			return r.ExecutePhase(item, "setup", env.EnvVars, nil, logWriter)
		}); err != nil {
			return Result{}, err
		}
	}

	r.logInfo("Executing develop phase for scenario", logx.AttrScenario, item.Slug, logx.AttrPhase, "develop")
	if err := r.runWithLifecycleLog(item.Slug, func(logWriter io.Writer) error {
		return r.ExecutePhase(item, "develop", env.EnvVars, nil, logWriter)
	}); err != nil {
		return Result{}, err
	}

	healthStatus, err := r.WaitForHealth(item, env.EnvVars)
	if err != nil {
		return Result{}, err
	}

	if len(failedDeps) > 0 {
		r.logWarn("Scenario started in degraded mode",
			logx.AttrScenario, item.Slug,
			logx.AttrStatus, healthStatus,
			"failed_dependencies", failedDeps,
		)
		// Preserve the degraded marker contract so remaining Bash-backed status
		// and log flows still recognize a partial best-effort startup.
		if err := r.writeDegradedState(item.Slug, failedDeps); err != nil {
			return Result{}, err
		}
	}

	return Result{
		Scenario:           item,
		AllocatedPorts:     env.AllocatedPorts,
		Health:             healthStatus,
		FailedDependencies: failedDeps,
	}, nil
}

func (r *Runner) Stop(name string, opts StopOptions) error {
	r.logInfo("Scenario stop requested", logx.AttrScenario, name)
	records, err := readScenarioRecords(r.Home, name)
	if err != nil {
		r.logError("Failed to read scenario records before stop", err, logx.AttrScenario, name)
		return err
	}

	processDir := process.ScenarioProcessDir(r.Home, name)
	stepFiles, globErr := filepath.Glob(filepath.Join(processDir, "*.json"))
	if globErr != nil {
		return globErr
	}

	groups := make(map[int]struct{})
	for _, record := range process.LiveRecords(records) {
		pgid := record.PGID
		if pgid <= 0 {
			pgid = record.PID
		}
		if pgid > 0 {
			groups[pgid] = struct{}{}
		}
	}

	for pgid := range groups {
		_ = signalProcessGroup(pgid, false)
	}
	if len(groups) > 0 {
		time.Sleep(2 * time.Second)
		for pgid := range groups {
			if process.IsPIDRunning(pgid) {
				_ = signalProcessGroup(pgid, true)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	for _, stepFile := range stepFiles {
		step := strings.TrimSuffix(filepath.Base(stepFile), filepath.Ext(stepFile))
		_ = process.RemoveScenarioRecord(r.Home, name, step)
	}
	_ = os.Remove(process.ScenarioDegradedPath(r.Home, name))

	portsToCheck := make(map[int]struct{})
	locks, err := r.Ports.LocksForScenario(name)
	if err != nil {
		r.logError("Failed to read scenario locks before stop", err, logx.AttrScenario, name)
		return err
	}
	for _, lock := range locks {
		portsToCheck[lock.Port] = struct{}{}
	}

	if item, loadErr := r.loadScenario(name, opts.CustomPath); loadErr == nil {
		for _, portSummary := range item.Manifest.SortedPorts() {
			if portSummary.FixedPort != nil {
				portsToCheck[*portSummary.FixedPort] = struct{}{}
			}
		}
	}

	if err := r.killOrphansOnPorts(portsToCheck); err != nil {
		r.logError("Failed to clean orphaned listeners", err, logx.AttrScenario, name)
		return err
	}

	if err := r.Ports.RemoveScenarioLocks(name); err != nil {
		r.logError("Failed to remove scenario locks", err, logx.AttrScenario, name)
		return err
	}
	r.logInfo("Scenario stop completed", logx.AttrScenario, name)
	return nil
}

func (r *Runner) Restart(name string, opts StartOptions) (Result, error) {
	r.logInfo("Scenario restart requested", logx.AttrScenario, name)
	if err := r.Stop(name, StopOptions{}); err != nil {
		r.logError("Scenario restart failed during stop", err, logx.AttrScenario, name)
		return Result{}, err
	}
	time.Sleep(1 * time.Second)
	opts.ForceSetup = true
	opts.ForceSetupScenario = name
	result, err := r.Start(name, opts)
	if err != nil {
		r.logError("Scenario restart failed during start", err, logx.AttrScenario, name)
		return Result{}, err
	}
	r.logInfo("Scenario restart completed", logx.AttrScenario, name, logx.AttrStatus, result.Health)
	return result, nil
}

func (r *Runner) logger() *slog.Logger {
	if r == nil || r.Logger == nil {
		return logx.WithSubsystem(slog.Default(), "lifecycle")
	}
	return r.Logger
}

func (r *Runner) logDebug(msg string, args ...any) {
	r.logger().Debug(msg, args...)
}

func (r *Runner) logInfo(msg string, args ...any) {
	r.logger().Info(msg, args...)
}

func (r *Runner) logWarn(msg string, args ...any) {
	r.logger().Warn(msg, args...)
}

func (r *Runner) logError(msg string, err error, args ...any) {
	logx.Error(r.logger(), msg, err, args...)
}

func (r *Runner) writeDegradedState(name string, failedDeps []string) error {
	processDir := process.ScenarioProcessDir(r.Home, name)
	if err := os.MkdirAll(processDir, 0o755); err != nil {
		return err
	}

	payload := map[string]any{
		"status":              "degraded",
		"reason":              "best-effort startup with failed dependencies",
		"failed_dependencies": failedDeps,
		"timestamp":           time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(process.ScenarioDegradedPath(r.Home, name), data, 0o644)
}

func (r *Runner) killOrphansOnPorts(portsToCheck map[int]struct{}) error {
	for port := range portsToCheck {
		pids, err := listeningPIDs(port)
		if err != nil {
			return err
		}
		for _, pid := range pids {
			_ = signalPID(pid, false)
		}
	}

	time.Sleep(500 * time.Millisecond)

	for port := range portsToCheck {
		pids, err := listeningPIDs(port)
		if err != nil {
			return err
		}
		for _, pid := range pids {
			_ = signalPID(pid, true)
		}
	}
	return nil
}

func listeningPIDs(port int) ([]int, error) {
	path, err := exec.LookPath("lsof")
	if err != nil {
		return nil, nil
	}
	cmd := exec.Command(path, "-tiTCP:"+strconv.Itoa(port), "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, err
	}

	pids := []int{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids, scanner.Err()
}

func (r *Runner) loadScenario(name, customPath string) (scenario.Scenario, error) {
	if strings.TrimSpace(customPath) == "" {
		item, err := scenario.Load(r.Root, name, scenario.SandboxEnvFromEnv())
		if err != nil {
			if errors.Is(err, scenario.ErrNotFound) {
				return scenario.Scenario{}, fmt.Errorf("scenario %q not found", name)
			}
			return scenario.Scenario{}, err
		}
		return item, nil
	}

	resolved := customPath
	if !filepath.IsAbs(resolved) {
		abs, err := filepath.Abs(resolved)
		if err != nil {
			return scenario.Scenario{}, err
		}
		resolved = abs
	}
	servicePath := filepath.Join(resolved, ".vrooli", "service.json")
	manifest, err := scenario.ReadService(servicePath)
	if err != nil {
		return scenario.Scenario{}, err
	}
	slug := name
	if slug == "" {
		slug = manifest.Service.Name
	}
	if slug == "" {
		slug = filepath.Base(resolved)
	}
	return scenario.Scenario{
		Slug:        slug,
		Path:        resolved,
		ServicePath: servicePath,
		Manifest:    manifest,
	}, nil
}

func stepConditionsMet(item scenario.Scenario, condition *scenario.Condition, env map[string]string) (bool, string, error) {
	if condition == nil {
		return true, "", nil
	}

	checkPath := func(target string) string {
		if strings.HasPrefix(target, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				target = filepath.Join(home, strings.TrimPrefix(target, "~"))
			}
		}
		if filepath.IsAbs(target) {
			return filepath.Clean(target)
		}
		return filepath.Join(item.Path, filepath.FromSlash(target))
	}

	if condition.FileExists != "" {
		if _, err := os.Stat(checkPath(condition.FileExists)); err != nil {
			return false, fmt.Sprintf("required file %q is missing", condition.FileExists), nil
		}
	}
	if fileNotExists := condition.FileNotExists; fileNotExists != "" {
		if _, err := os.Stat(checkPath(fileNotExists)); err == nil {
			return false, fmt.Sprintf("file %q must not exist", fileNotExists), nil
		}
	}
	if condition.DirectoryExists != "" {
		info, err := os.Stat(checkPath(condition.DirectoryExists))
		if err != nil || !info.IsDir() {
			return false, fmt.Sprintf("required directory %q is missing", condition.DirectoryExists), nil
		}
	}
	if condition.ResourceEnabled != "" {
		dep, ok := item.Manifest.Dependencies.Resources[condition.ResourceEnabled]
		if !ok || !dep.Enabled {
			return false, fmt.Sprintf("resource %q is disabled", condition.ResourceEnabled), nil
		}
	}
	if jsonSpec := condition.JSONPathExists; jsonSpec != "" {
		ok, err := jsonPathExists(checkPath(strings.SplitN(jsonSpec, ":", 2)[0]), jsonSpec)
		if err != nil {
			return false, "", err
		}
		if !ok {
			return false, fmt.Sprintf("JSON path %q was not found", jsonSpec), nil
		}
	}
	if command := condition.CommandExists; command != "" {
		if _, err := exec.LookPath(command); err != nil {
			return false, fmt.Sprintf("command %q is unavailable", command), nil
		}
	}
	if binary := condition.BinaryExists; binary != "" {
		if _, err := exec.LookPath(binary); err != nil {
			return false, fmt.Sprintf("command %q is unavailable", binary), nil
		}
	}
	if key := condition.EnvVarSet; key != "" {
		if strings.TrimSpace(env[key]) == "" && strings.TrimSpace(os.Getenv(key)) == "" {
			return false, fmt.Sprintf("environment variable %q is not set", key), nil
		}
	}
	if always := condition.Always; always != "" {
		lower := strings.ToLower(strings.TrimSpace(always))
		if lower == "false" || lower == "0" {
			return false, "step disabled by always=false", nil
		}
	}

	return true, "", nil
}

func jsonPathExists(filePath, spec string) (bool, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return false, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false, err
	}

	current := value
	for _, segment := range strings.Split(parts[1], ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[segment]
			if !ok {
				return false, nil
			}
			current = next
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return false, nil
			}
			current = typed[index]
		default:
			return false, nil
		}
	}
	return current != nil, nil
}

func binariesNeedSetup(appRoot string, check scenario.ConditionCheck) (bool, string, error) {
	for _, target := range check.Targets {
		path := resolveCheckPath(appRoot, target)
		info, err := os.Stat(path)
		if err != nil || info.Mode()&0o111 == 0 {
			return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
		}

		binaryDir := filepath.Dir(path)
		if anyFileNewer(binaryDir, path, func(path string, d fs.DirEntry) bool {
			return strings.HasSuffix(path, ".go")
		}) {
			return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
		}
		for _, depFile := range []string{"go.mod", "go.sum"} {
			depPath := filepath.Join(binaryDir, depFile)
			if info, err := os.Stat(depPath); err == nil && info.ModTime().After(getModTime(path)) {
				return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
			}
		}

		replacePaths, err := localReplacePaths(filepath.Join(binaryDir, "go.mod"))
		if err != nil {
			return false, "", err
		}
		for _, replacePath := range replacePaths {
			resolved := filepath.Join(binaryDir, replacePath)
			if anyFileNewer(resolved, path, func(path string, d fs.DirEntry) bool {
				return strings.HasSuffix(path, ".go") || filepath.Base(path) == "go.mod"
			}) {
				return true, "Missing binaries: " + strings.Join(check.Targets, ","), nil
			}
		}
	}
	return false, "", nil
}

func cliNeedsSetup(appRoot string, check scenario.ConditionCheck) (bool, string, error) {
	if strings.TrimSpace(check.Command) == "" {
		return false, "", nil
	}
	cliPath, err := exec.LookPath(check.Command)
	if err != nil {
		return true, "CLI not installed: " + check.Command, nil
	}

	sourceDir := filepath.Join(appRoot, "cli")
	if _, err := os.Stat(sourceDir); err != nil {
		return false, "", nil
	}

	if anyFileNewer(sourceDir, cliPath, func(path string, d fs.DirEntry) bool {
		return strings.HasSuffix(path, ".go")
	}) {
		return true, "CLI not installed: " + check.Command, nil
	}
	for _, depFile := range []string{"go.mod", "go.sum"} {
		depPath := filepath.Join(sourceDir, depFile)
		if info, err := os.Stat(depPath); err == nil && info.ModTime().After(getModTime(cliPath)) {
			return true, "CLI not installed: " + check.Command, nil
		}
	}

	replacePaths, err := localReplacePaths(filepath.Join(sourceDir, "go.mod"))
	if err != nil {
		return false, "", err
	}
	for _, replacePath := range replacePaths {
		resolved := filepath.Join(sourceDir, replacePath)
		if anyFileNewer(resolved, cliPath, func(path string, d fs.DirEntry) bool {
			return strings.HasSuffix(path, ".go") || filepath.Base(path) == "go.mod"
		}) {
			return true, "CLI not installed: " + check.Command, nil
		}
	}
	return false, "", nil
}

func uiBundleNeedsSetup(appRoot string, check scenario.ConditionCheck) (bool, string, error) {
	bundlePath := resolveCheckPath(appRoot, defaultIfEmpty(check.BundlePath, "ui/dist/index.html"))
	sourceDir := resolveCheckPath(appRoot, defaultIfEmpty(check.SourceDir, "ui/src"))
	if _, err := os.Stat(bundlePath); err != nil {
		return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
	}

	if anyFileNewer(sourceDir, bundlePath, func(path string, d fs.DirEntry) bool { return !d.IsDir() }) {
		return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
	}

	uiDir := filepath.Dir(filepath.Dir(bundlePath))
	for _, file := range []string{"package.json", "vite.config.ts", "vite.config.js", "tsconfig.json", "index.html"} {
		configPath := filepath.Join(uiDir, file)
		if info, err := os.Stat(configPath); err == nil && info.ModTime().After(getModTime(bundlePath)) {
			return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
		}
	}

	watchDeps := true
	if check.WatchFileDependencies != nil {
		watchDeps = *check.WatchFileDependencies
	}
	if watchDeps {
		packageJSON := filepath.Join(uiDir, "package.json")
		specs, err := fileDependencySpecs(packageJSON)
		if err != nil {
			return false, "", err
		}
		excluded := make(map[string]struct{}, len(check.DependencyExcludes))
		for _, path := range check.DependencyExcludes {
			excluded[resolveCheckPath(uiDir, path)] = struct{}{}
		}
		for _, spec := range specs {
			resolved := resolveCheckPath(uiDir, strings.TrimPrefix(spec, "file:"))
			if _, skip := excluded[resolved]; skip {
				continue
			}
			if _, err := os.Stat(resolved); err != nil {
				return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
			}
			if anyFileNewer(resolved, bundlePath, func(path string, d fs.DirEntry) bool {
				return !d.IsDir() && !strings.Contains(path, string(filepath.Separator)+"node_modules"+string(filepath.Separator)) && !strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator))
			}) {
				return true, "UI bundle outdated: " + defaultIfEmpty(check.BundlePath, "ui/dist/index.html"), nil
			}
		}
	}

	return false, "", nil
}

func resourcesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	if check.Populated {
		_, err := os.Stat(filepath.Join(appRoot, "data", ".resources-populated"))
		return err != nil
	}
	if len(check.Resources) == 0 {
		_, err := os.Stat(filepath.Join(appRoot, "data", ".resources-populated"))
		return err != nil
	}
	for _, resourceName := range check.Resources {
		if _, err := os.Stat(filepath.Join(appRoot, "data", "."+resourceName+"-populated")); err != nil {
			return true
		}
	}
	return false
}

func dependenciesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	for _, path := range check.Paths {
		resolved := resolveCheckPath(appRoot, path)
		switch {
		case strings.HasSuffix(resolved, "package.json"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "node_modules")); err != nil {
				return true
			}
		case strings.HasSuffix(resolved, "go.mod"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "go.sum")); err != nil {
				if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "vendor")); err != nil {
					return true
				}
			}
		case strings.HasSuffix(resolved, "requirements.txt"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "venv")); err != nil {
				if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), ".venv")); err != nil {
					return true
				}
			}
		case strings.HasSuffix(resolved, "Cargo.toml"):
			if _, err := os.Stat(filepath.Join(filepath.Dir(resolved), "target")); err != nil {
				return true
			}
		default:
			if _, err := os.Stat(resolved); err != nil {
				return true
			}
		}
	}
	return false
}

func dataNeedsSetup(appRoot string, check scenario.ConditionCheck) bool {
	target := resolveCheckPath(appRoot, defaultIfEmpty(check.Path, "data"))
	entries, err := os.ReadDir(target)
	return err != nil || len(entries) == 0
}

func filesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	for _, path := range check.Paths {
		resolved := resolveCheckPath(appRoot, path)
		if _, err := os.Stat(resolved); err != nil {
			return true
		}
	}
	return false
}

func directoriesNeedSetup(appRoot string, check scenario.ConditionCheck) bool {
	for _, path := range check.Targets {
		resolved := resolveCheckPath(appRoot, path)
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return true
		}
	}
	return false
}

func readScenarioRecords(home, name string) ([]process.Record, error) {
	return process.ReadScenarioRecords(home, name)
}

func localReplacePaths(goModPath string) ([]string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	paths := []string{}
	inBlock := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch line {
		case "", ")":
			if line == ")" {
				inBlock = false
			}
			continue
		case "replace (":
			inBlock = true
			continue
		}

		candidate := line
		if strings.HasPrefix(line, "replace ") {
			candidate = strings.TrimSpace(strings.TrimPrefix(line, "replace "))
		} else if !inBlock {
			continue
		}
		if !strings.Contains(candidate, "=>") {
			continue
		}

		fields := strings.Fields(candidate)
		if len(fields) == 0 {
			continue
		}
		path := fields[len(fields)-1]
		if strings.HasPrefix(path, "../") {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func anyFileNewer(root, target string, include func(path string, d fs.DirEntry) bool) bool {
	if _, err := os.Stat(root); err != nil {
		return false
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		return false
	}
	targetTime := targetInfo.ModTime()

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if include != nil && !include(path, d) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(targetTime) {
			return errStopWalk
		}
		return nil
	})

	return errors.Is(walkErr, errStopWalk)
}

var errStopWalk = errors.New("stop walk")

func getModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func fileDependencySpecs(packageJSON string) ([]string, error) {
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	specs := []string{}
	for _, deps := range []map[string]string{
		doc.Dependencies,
		doc.DevDependencies,
		doc.PeerDependencies,
		doc.OptionalDependencies,
	} {
		for _, value := range deps {
			if strings.HasPrefix(value, "file:") {
				specs = append(specs, value)
			}
		}
	}
	sort.Strings(specs)
	return specs, nil
}

func resolveCheckPath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, filepath.FromSlash(path))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func mergeEnv(base []string, overrides map[string]string) []string {
	merged := append([]string(nil), base...)
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		merged = setEnvValue(merged, key, overrides[key])
	}
	return merged
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(env, prefix+value)
}

func inferStepPort(manifest scenario.ServiceManifest, step string, env map[string]string) int {
	key := scenario.InferPortEnvVar(manifest, step)
	if key == "" {
		return 0
	}
	port, _ := strconv.Atoi(env[key])
	return port
}

func healthPortsFromEnv(manifest scenario.ServiceManifest, env map[string]string) map[string]int {
	ports := make(map[string]int)
	for _, key := range manifest.PortEnvVars() {
		if port, err := strconv.Atoi(strings.TrimSpace(env[key])); err == nil && port > 0 {
			ports[key] = port
		}
	}
	for key, value := range env {
		if _, exists := ports[key]; exists || !strings.HasSuffix(key, "_PORT") {
			continue
		}
		if port, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && port > 0 {
			ports[key] = port
		}
	}
	return ports
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
