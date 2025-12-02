package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/orchestrator/workspace"
)

// runUnitPhase executes language-specific unit tests and shell syntax checks without relying on bash orchestration.
func runUnitPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{Err: err, FailureClassification: FailureClassSystem}
	}

	var observations []string
	goObs, goFailure := executeGoUnitTests(ctx, env, logWriter)
	if goFailure != nil {
		goFailure.Observations = append(goFailure.Observations, observations...)
		return *goFailure
	}
	observations = append(observations, goObs...)

	nodeObs, nodeFailure := executeNodeUnitTests(ctx, env, logWriter)
	if nodeFailure != nil {
		nodeFailure.Observations = append(nodeFailure.Observations, observations...)
		return *nodeFailure
	}
	observations = append(observations, nodeObs...)

	pythonObs, pythonFailure := executePythonUnitTests(ctx, env, logWriter)
	if pythonFailure != nil {
		pythonFailure.Observations = append(pythonFailure.Observations, observations...)
		return *pythonFailure
	}
	observations = append(observations, pythonObs...)

	shellObs, shellFailure := lintScenarioShellTargets(ctx, env, logWriter)
	if shellFailure != nil {
		shellFailure.Observations = append(shellFailure.Observations, observations...)
		return *shellFailure
	}
	observations = append(observations, shellObs...)

	logPhaseStep(logWriter, "unit validation complete")
	return RunReport{Observations: observations}
}

func executeGoUnitTests(ctx context.Context, env workspace.Environment, logWriter io.Writer) ([]string, *RunReport) {
	apiDir := filepath.Join(env.ScenarioDir, "api")
	if err := ensureDir(apiDir); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Ensure the api/ directory exists so Go unit tests can run.",
		}
	}
	if err := EnsureCommandAvailable("go"); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install the Go toolchain to execute API unit tests.",
		}
	}
	logPhaseStep(logWriter, "executing go test ./... inside %s", apiDir)
	if err := phaseCommandExecutor(ctx, apiDir, logWriter, "go", "test", "./..."); err != nil {
		return nil, &RunReport{
			Err:                   fmt.Errorf("go test ./... failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Fix failing Go tests under api/ before re-running the suite.",
		}
	}
	return []string{"go test ./... passed"}, nil
}

func lintScenarioShellTargets(ctx context.Context, env workspace.Environment, logWriter io.Writer) ([]string, *RunReport) {
	shellTargets := []string{
		filepath.Join(env.ScenarioDir, "cli", "test-genie"),
		filepath.Join(env.ScenarioDir, "test", "lib", "runtime.sh"),
		filepath.Join(env.ScenarioDir, "test", "lib", "orchestrator.sh"),
	}
	if err := EnsureCommandAvailable("bash"); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install bash so shell entrypoints can be linted.",
		}
	}
	var observations []string
	for _, target := range shellTargets {
		if err := ensureFile(target); err != nil {
			return nil, &RunReport{
				Err:                   err,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Restore the CLI binary and test/lib scripts so syntax checks can run.",
			}
		}
		logPhaseStep(logWriter, "running bash -n %s", target)
		if err := phaseCommandExecutor(ctx, "", logWriter, "bash", "-n", target); err != nil {
			return nil, &RunReport{
				Err:                   fmt.Errorf("bash -n %s failed: %w", target, err),
				FailureClassification: FailureClassSystem,
				Remediation:           fmt.Sprintf("Fix syntax errors in %s and re-run the suite.", target),
			}
		}
		observations = append(observations, fmt.Sprintf("bash -n verified: %s", target))
	}
	return observations, nil
}

func executeNodeUnitTests(ctx context.Context, env workspace.Environment, logWriter io.Writer) ([]string, *RunReport) {
	nodeDir := detectNodeWorkspaceDir(env.ScenarioDir)
	if nodeDir == "" {
		return nil, nil
	}

	if err := EnsureCommandAvailable("node"); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install Node.js so UI/unit suites can execute.",
		}
	}

	manifest, err := loadPackageManifest(filepath.Join(nodeDir, "package.json"))
	if err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Fix package.json so the Node workspace can be parsed.",
		}
	}
	testScript := ""
	if manifest != nil {
		testScript = manifest.Scripts["test"]
	}
	if manifest == nil || testScript == "" {
		return []string{"node workspace detected but package.json lacks a test script"}, nil
	}

	packageManager := detectPackageManager(manifest, nodeDir)
	if err := EnsureCommandAvailable(packageManager); err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           fmt.Sprintf("Install %s to run Node test suites.", packageManager),
		}
	}

	if _, err := os.Stat(filepath.Join(nodeDir, "node_modules")); os.IsNotExist(err) {
		logPhaseStep(logWriter, "installing Node dependencies via %s", packageManager)
		if installErr := installNodeDependencies(ctx, nodeDir, packageManager, logWriter); installErr != nil {
			return nil, &RunReport{
				Err:                   fmt.Errorf("%s install failed: %w", packageManager, installErr),
				FailureClassification: FailureClassSystem,
				Remediation:           "Resolve dependency installation issues before re-running unit tests.",
			}
		}
	}

	logPhaseStep(logWriter, "running Node unit tests with %s", packageManager)
	output, err := phaseCommandCapture(ctx, nodeDir, logWriter, packageManager, "test")
	if err != nil {
		return nil, &RunReport{
			Err:                   fmt.Errorf("Node unit tests failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Inspect the UI/unit test output above, fix failures, and rerun the suite.",
		}
	}

	observations := []string{fmt.Sprintf("node unit tests passed via %s", packageManager)}
	if pct := detectNodeCoverage(nodeDir, output); pct != "" {
		observations = append(observations, fmt.Sprintf("node coverage: %s%% statements", pct))
	}
	return observations, nil
}

func executePythonUnitTests(ctx context.Context, env workspace.Environment, logWriter io.Writer) ([]string, *RunReport) {
	pythonDir := detectPythonWorkspaceDir(env.ScenarioDir)
	if pythonDir == "" {
		return nil, nil
	}

	pythonCmd, err := resolvePythonCommand()
	if err != nil {
		return nil, &RunReport{
			Err:                   err,
			FailureClassification: FailureClassMissingDependency,
			Remediation:           "Install python3 so scenario-specific Python suites can run.",
		}
	}

	if !hasPythonTests(pythonDir) {
		return []string{"python workspace detected but no test_*.py files found"}, nil
	}

	usePytest := pythonSupportsModule(ctx, pythonCmd, "pytest")
	if usePytest {
		logPhaseStep(logWriter, "running pytest under %s", pythonDir)
		if err := phaseCommandExecutor(ctx, pythonDir, logWriter, pythonCmd, "-m", "pytest", "-q"); err != nil {
			return nil, &RunReport{
				Err:                   fmt.Errorf("pytest failed: %w", err),
				FailureClassification: FailureClassSystem,
				Remediation:           "Inspect pytest output above, fix failing tests, and rerun the suite.",
			}
		}
		return []string{"python unit tests passed via pytest"}, nil
	}

	logPhaseStep(logWriter, "running unittest discover under %s", pythonDir)
	if err := phaseCommandExecutor(ctx, pythonDir, logWriter, pythonCmd, "-m", "unittest", "discover"); err != nil {
		return nil, &RunReport{
			Err:                   fmt.Errorf("python -m unittest discover failed: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure the default unittest suites pass or install pytest for richer reporting.",
		}
	}
	return []string{"python unit tests passed via unittest"}, nil
}

func detectNodeWorkspaceDir(scenarioDir string) string {
	candidates := []string{
		filepath.Join(scenarioDir, "ui"),
		scenarioDir,
	}
	for _, candidate := range candidates {
		if fileExists(filepath.Join(candidate, "package.json")) {
			return candidate
		}
	}
	return ""
}

type packageManifest struct {
	Scripts        map[string]string `json:"scripts"`
	PackageManager string            `json:"packageManager"`
}

func loadPackageManifest(path string) (*packageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc packageManifest
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Scripts == nil {
		doc.Scripts = make(map[string]string)
	} else {
		testScript := strings.TrimSpace(doc.Scripts["test"])
		if testScript == "" || testScript == `echo "Error: no test specified" && exit 1` {
			doc.Scripts["test"] = ""
		} else {
			doc.Scripts["test"] = testScript
		}
	}
	return &doc, nil
}

func detectPackageManager(manifest *packageManifest, dir string) string {
	if manifest != nil {
		if mgr := parsePackageManager(manifest.PackageManager); mgr != "" {
			return mgr
		}
	}
	switch {
	case fileExists(filepath.Join(dir, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(dir, "yarn.lock")):
		return "yarn"
	default:
		return "npm"
	}
}

func parsePackageManager(raw string) string {
	if raw == "" {
		return ""
	}
	lowered := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lowered, "pnpm"):
		return "pnpm"
	case strings.HasPrefix(lowered, "yarn"):
		return "yarn"
	case strings.HasPrefix(lowered, "npm"):
		return "npm"
	default:
		return ""
	}
}

func installNodeDependencies(ctx context.Context, dir, manager string, logWriter io.Writer) error {
	switch manager {
	case "pnpm":
		return phaseCommandExecutor(ctx, dir, logWriter, "pnpm", "install", "--frozen-lockfile", "--ignore-scripts")
	case "yarn":
		return phaseCommandExecutor(ctx, dir, logWriter, "yarn", "install", "--frozen-lockfile")
	default:
		return phaseCommandExecutor(ctx, dir, logWriter, "npm", "install")
	}
}

func detectNodeCoverage(dir, output string) string {
	summaryPath := filepath.Join(dir, "coverage", "coverage-summary.json")
	if data, err := os.ReadFile(summaryPath); err == nil {
		type totals struct {
			Statements struct {
				Pct float64 `json:"pct"`
			} `json:"statements"`
		}
		var doc struct {
			Total totals `json:"total"`
		}
		if err := json.Unmarshal(data, &doc); err == nil && doc.Total.Statements.Pct > 0 {
			return fmt.Sprintf("%.2f", doc.Total.Statements.Pct)
		}
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "%") {
			if pct := extractPercentage(line); pct != "" {
				return pct
			}
		}
	}
	return ""
}

func extractPercentage(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	for _, token := range strings.Fields(line) {
		if strings.HasSuffix(token, "%") {
			return strings.TrimSuffix(token, "%")
		}
	}
	return ""
}

func detectPythonWorkspaceDir(scenarioDir string) string {
	candidates := []string{
		filepath.Join(scenarioDir, "python"),
		scenarioDir,
	}
	for _, candidate := range candidates {
		if dirExists(candidate) && pythonIndicatorsPresent(candidate) {
			return candidate
		}
	}
	return ""
}

func pythonIndicatorsPresent(dir string) bool {
	indicators := []string{
		"requirements.txt",
		"pyproject.toml",
		"setup.py",
		filepath.Join("tests", "__init__.py"),
	}
	for _, indicator := range indicators {
		if fileExists(filepath.Join(dir, indicator)) {
			return true
		}
	}
	if dirExists(filepath.Join(dir, "tests")) {
		return true
	}
	return false
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func resolvePythonCommand() (string, error) {
	candidates := []string{"python3", "python"}
	for _, cmd := range candidates {
		if err := EnsureCommandAvailable(cmd); err == nil {
			return cmd, nil
		}
	}
	return "", fmt.Errorf("python runtime not available")
}

func pythonSupportsModule(ctx context.Context, pythonCmd, module string) bool {
	script := fmt.Sprintf("import importlib.util, sys; sys.exit(0 if importlib.util.find_spec('%s') else 1)", module)
	if err := phaseCommandExecutor(ctx, "", io.Discard, pythonCmd, "-c", script); err != nil {
		return false
	}
	return true
}

func hasPythonTests(dir string) bool {
	found := false
	errStopWalk := errors.New("stop-walk")
	skipDirs := map[string]struct{}{
		".git":         {},
		"node_modules": {},
		"dist":         {},
		"build":        {},
		".next":        {},
		"coverage":     {},
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || found {
			return walkErr
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") && (strings.HasPrefix(d.Name(), "test_") || strings.HasSuffix(d.Name(), "_test.py")) {
			found = true
			return errStopWalk
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStopWalk) {
		return false
	}
	return found
}
