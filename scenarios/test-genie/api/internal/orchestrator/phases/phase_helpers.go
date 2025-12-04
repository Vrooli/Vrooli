package phases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"test-genie/internal/orchestrator/workspace"
)

var commandLookup = exec.LookPath
var phaseCommandExecutor = runCommand
var phaseCommandCapture = runCommandCapture

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required directory missing: %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", path)
	}
	return nil
}

func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required file missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}

func EnsureCommandAvailable(name string) error {
	if _, err := commandLookup(name); err != nil {
		return fmt.Errorf("required command '%s' is not available: %w", name, err)
	}
	return nil
}

func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required executable missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected executable but found directory: %s", path)
	}
	if runtime.GOOS == "windows" {
		// Windows does not expose POSIX execute bits, so existence is enough.
		return nil
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("file is not executable: %s", path)
	}
	return nil
}

// logPhaseStep writes a structured step message to the log.
// Uses consistent formatting that can be parsed for observation streaming.
func logPhaseStep(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

// logPhaseSuccess writes a success message with a consistent marker.
func logPhaseSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}

// logPhaseInfo writes an info/progress message with a consistent marker.
func logPhaseInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

// logPhaseWarn writes a warning message with a structured marker.
// The format includes the phase name and warning type for easy parsing.
func logPhaseWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[WARNING] ⚠️ %s\n", msg)
}

// logPhaseError writes an error message with a structured marker.
func logPhaseError(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[ERROR] ❌ %s\n", msg)
}

func runCommand(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if logWriter == nil {
		logWriter = io.Discard
	}
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	return cmd.Run()
}

func runCommandCapture(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var output bytes.Buffer
	if logWriter != nil {
		cmd.Stdout = io.MultiWriter(logWriter, &output)
		cmd.Stderr = logWriter
	} else {
		cmd.Stdout = &output
		cmd.Stderr = io.Discard
	}
	err := cmd.Run()
	return output.String(), err
}

// OverrideCommandLookup temporarily replaces the binary lookup used by phases.
func OverrideCommandLookup(fn func(string) (string, error)) func() {
	prev := commandLookup
	commandLookup = fn
	return func() { commandLookup = prev }
}

// OverrideCommandExecutor temporarily replaces the command executor used by phases.
func OverrideCommandExecutor(fn func(context.Context, string, io.Writer, string, ...string) error) func() {
	prev := phaseCommandExecutor
	phaseCommandExecutor = fn
	return func() { phaseCommandExecutor = prev }
}

// OverrideCommandCapture temporarily replaces the capture executor used by phases.
func OverrideCommandCapture(fn func(context.Context, string, io.Writer, string, ...string) (string, error)) func() {
	prev := phaseCommandCapture
	phaseCommandCapture = fn
	return func() { phaseCommandCapture = prev }
}

func discoverScenarioCLIBinary(env workspace.Environment) (string, error) {
	cliDir := filepath.Join(env.ScenarioDir, "cli")
	info, err := os.Stat(cliDir)
	if err != nil {
		return "", fmt.Errorf("cli directory missing: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cli path is not a directory: %s", cliDir)
	}

	var candidates []string
	name := strings.TrimSpace(env.ScenarioName)
	if name != "" {
		candidates = append(candidates,
			filepath.Join(cliDir, name),
			filepath.Join(cliDir, name+".sh"),
			filepath.Join(cliDir, name+".exe"),
		)
	}
	candidates = append(candidates,
		filepath.Join(cliDir, "test-genie"),
		filepath.Join(cliDir, "test-genie.exe"),
	)
	for _, candidate := range candidates {
		if err := ensureExecutable(candidate); err == nil {
			return candidate, nil
		}
	}

	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return "", fmt.Errorf("failed to list cli directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(cliDir, entry.Name())
		if err := ensureExecutable(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no executable CLI binary found under %s", cliDir)
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// hasNodeWorkspace checks if a Node.js workspace exists in the scenario.
func hasNodeWorkspace(env workspace.Environment) bool {
	candidates := []string{
		filepath.Join(env.ScenarioDir, "package.json"),
		filepath.Join(env.ScenarioDir, "ui", "package.json"),
	}
	for _, path := range candidates {
		if fileExists(path) {
			return true
		}
	}
	return false
}

// Node.js helpers for performance phase

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

func findPrimaryBatsSuite(cliDir, scenarioName string) (string, error) {
	preferred := []string{}
	name := strings.TrimSpace(scenarioName)
	if name != "" {
		preferred = append(preferred,
			filepath.Join(cliDir, name+".bats"),
			filepath.Join(cliDir, name+"-cli.bats"),
		)
	}
	preferred = append(preferred,
		filepath.Join(cliDir, "test-genie.bats"),
	)

	for _, candidate := range preferred {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return "", fmt.Errorf("failed to scan cli directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bats") {
			continue
		}
		return filepath.Join(cliDir, entry.Name()), nil
	}
	return "", fmt.Errorf("no .bats suites found under %s", cliDir)
}
