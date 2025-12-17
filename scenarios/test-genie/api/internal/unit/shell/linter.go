package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"test-genie/internal/unit/types"
)

// Common file magic signatures for non-shell executable formats.
// These are checked to avoid running bash -n on binaries.
var nonShellMagicBytes = []struct {
	name   string
	magic  []byte
	offset int
}{
	{"ELF", []byte{0x7f, 'E', 'L', 'F'}, 0},                   // Linux/Unix executables
	{"MachO-32", []byte{0xfe, 0xed, 0xfa, 0xce}, 0},           // macOS 32-bit
	{"MachO-64", []byte{0xfe, 0xed, 0xfa, 0xcf}, 0},           // macOS 64-bit
	{"MachO-Fat", []byte{0xca, 0xfe, 0xba, 0xbe}, 0},          // macOS fat binary
	{"PE", []byte{'M', 'Z'}, 0},                               // Windows PE
	{"Java", []byte{0xca, 0xfe, 0xba, 0xbe}, 0},               // Java class (same as MachO fat, but context matters)
	{"WebAssembly", []byte{0x00, 'a', 's', 'm'}, 0},           // WASM
	{"Gzip", []byte{0x1f, 0x8b}, 0},                           // Gzip compressed
	{"Bzip2", []byte{'B', 'Z', 'h'}, 0},                       // Bzip2 compressed
	{"XZ", []byte{0xfd, '7', 'z', 'X', 'Z', 0x00}, 0},         // XZ compressed
	{"Zip", []byte{'P', 'K', 0x03, 0x04}, 0},                  // ZIP archive
	{"PDF", []byte{'%', 'P', 'D', 'F'}, 0},                    // PDF document
	{"SQLite", []byte{'S', 'Q', 'L', 'i', 't', 'e'}, 0},       // SQLite database
}

// Linter validates shell script syntax.
type Linter struct {
	scenarioDir  string
	scenarioName string
	exclude      []string
	executor     types.CommandExecutor
	logWriter    io.Writer
}

// Config holds configuration for the shell linter.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario.
	ScenarioName string

	// Exclude is a list of relative paths (from scenario root) to exclude from linting.
	// Paths can be files or directories. Example: ["cli/my-binary", "api/server"]
	Exclude []string

	// Executor is the command executor to use.
	Executor types.CommandExecutor

	// LogWriter is where to write log output.
	LogWriter io.Writer
}

// New creates a new shell linter.
func New(cfg Config) *Linter {
	executor := cfg.Executor
	if executor == nil {
		executor = types.NewDefaultExecutor()
	}
	logWriter := cfg.LogWriter
	if logWriter == nil {
		logWriter = io.Discard
	}
	return &Linter{
		scenarioDir:  cfg.ScenarioDir,
		scenarioName: cfg.ScenarioName,
		exclude:      cfg.Exclude,
		executor:     executor,
		logWriter:    logWriter,
	}
}

// Name returns the linter's language name.
func (l *Linter) Name() string {
	return "shell"
}

// Detect returns true if shell scripts exist to lint.
func (l *Linter) Detect() bool {
	_, err := l.discoverCLIBinary()
	return err == nil
}

// Run lints shell scripts and returns the result.
func (l *Linter) Run(ctx context.Context) types.Result {
	if err := ctx.Err(); err != nil {
		return types.FailSystem(err, "Context cancelled")
	}

	cliPath, err := l.discoverCLIBinary()
	if err != nil {
		logWarn(l.logWriter, "CLI binary not linted: %v", err)
		return types.Skip("No shell entrypoints detected")
	}

	// Check bash is available
	if err := types.EnsureCommand(l.executor, "bash"); err != nil {
		return types.FailMissingDependency(err, "Install bash so shell entrypoints can be linted.")
	}

	// Verify file exists
	if err := ensureFile(cliPath); err != nil {
		return types.FailMisconfiguration(err, "Restore the CLI binary so syntax checks can run.")
	}

	// Run bash -n
	logStep(l.logWriter, "running bash -n %s", cliPath)
	if err := l.executor.Run(ctx, "", l.logWriter, "bash", "-n", cliPath); err != nil {
		return types.FailTestFailure(
			fmt.Errorf("bash -n %s failed: %w", cliPath, err),
			fmt.Sprintf("Fix syntax errors in %s and re-run the suite.", cliPath),
		)
	}

	return types.OK().WithObservations(types.NewSuccessObservation(fmt.Sprintf("bash -n verified: %s", cliPath)))
}

// discoverCLIBinary finds a shell script CLI entry point for the scenario.
// It checks exclusion lists and file magic bytes to avoid linting compiled binaries.
func (l *Linter) discoverCLIBinary() (string, error) {
	cliDir := filepath.Join(l.scenarioDir, "cli")
	info, err := os.Stat(cliDir)
	if err != nil {
		return "", fmt.Errorf("cli directory missing: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cli path is not a directory: %s", cliDir)
	}

	// Helper to check if a candidate is valid (executable, not excluded, is shell script)
	isValidCandidate := func(path string) bool {
		if err := ensureExecutable(path); err != nil {
			return false
		}
		// Check if excluded
		relPath, err := filepath.Rel(l.scenarioDir, path)
		if err != nil {
			relPath = path
		}
		if l.isExcluded(relPath) {
			logWarn(l.logWriter, "skipping excluded file: %s", relPath)
			return false
		}
		// Check if it's a shell script (not a binary)
		isShell, reason := isShellScript(path)
		if !isShell {
			logInfo(l.logWriter, "skipping non-shell file: %s (%s)", relPath, reason)
			return false
		}
		return true
	}

	// Try scenario-specific names first
	var candidates []string
	name := strings.TrimSpace(l.scenarioName)
	if name != "" {
		candidates = append(candidates,
			filepath.Join(cliDir, name+".sh"), // Prefer .sh extension
			filepath.Join(cliDir, name),
			filepath.Join(cliDir, name+".exe"),
		)
	}
	// Add fallback patterns
	candidates = append(candidates,
		filepath.Join(cliDir, "test-genie.sh"),
		filepath.Join(cliDir, "test-genie"),
		filepath.Join(cliDir, "test-genie.exe"),
	)

	for _, candidate := range candidates {
		if isValidCandidate(candidate) {
			return candidate, nil
		}
	}

	// Scan directory for any executable shell script
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return "", fmt.Errorf("failed to list cli directory: %w", err)
	}

	// First pass: prefer .sh files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}
		path := filepath.Join(cliDir, entry.Name())
		if isValidCandidate(path) {
			return path, nil
		}
	}

	// Second pass: try any executable that's a shell script
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sh") {
			continue // Already checked
		}
		path := filepath.Join(cliDir, entry.Name())
		if isValidCandidate(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("no shell script CLI entry point found under %s (compiled binaries and excluded files were skipped)", cliDir)
}

// ensureFile checks that a file exists and is not a directory.
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

// ensureExecutable checks that a file exists and is executable.
func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required executable missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected executable but found directory: %s", path)
	}
	if runtime.GOOS == "windows" {
		// Windows does not expose POSIX execute bits, so existence is enough
		return nil
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("file is not executable: %s", path)
	}
	return nil
}

// logStep writes a step message to the log.
func logStep(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

// logInfo writes an info message to the log.
func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[INFO] ℹ️ %s\n", msg)
}

// logWarn writes a warning message to the log.
func logWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[WARNING] ⚠️ %s\n", msg)
}

// isExcluded checks if a path should be excluded from linting.
// The path should be relative to the scenario directory.
func (l *Linter) isExcluded(relPath string) bool {
	// Normalize the path for comparison
	relPath = filepath.Clean(relPath)
	for _, exclude := range l.exclude {
		exclude = filepath.Clean(exclude)
		if relPath == exclude || strings.HasPrefix(relPath, exclude+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// isShellScript checks if a file appears to be a shell script by examining its contents.
// Returns true if the file looks like a shell script, false if it's a known binary format.
func isShellScript(path string) (bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Sprintf("cannot open: %v", err)
	}
	defer f.Close()

	// Read enough bytes to check magic signatures and shebang
	header := make([]byte, 64)
	n, err := f.Read(header)
	if err != nil && n == 0 {
		return false, fmt.Sprintf("cannot read: %v", err)
	}
	header = header[:n]

	// Check against known binary magic signatures
	for _, sig := range nonShellMagicBytes {
		if len(header) > sig.offset+len(sig.magic) {
			match := true
			for i, b := range sig.magic {
				if header[sig.offset+i] != b {
					match = false
					break
				}
			}
			if match {
				return false, sig.name + " binary"
			}
		}
	}

	// Check for shell shebang (#!/bin/bash, #!/bin/sh, etc.)
	if len(header) >= 2 && header[0] == '#' && header[1] == '!' {
		return true, "shebang detected"
	}

	// Check for common shell script patterns at the start
	// (files without shebang but clearly shell scripts)
	headerStr := string(header)
	shellPatterns := []string{
		"#!/",      // shebang
		"# shellcheck",
		"set -e",
		"set -o",
		"export ",
		"function ",
		"if [",
		"for ",
		"while ",
		"case ",
	}
	for _, pattern := range shellPatterns {
		if strings.HasPrefix(headerStr, pattern) || strings.Contains(headerStr, "\n"+pattern) {
			return true, "shell pattern detected"
		}
	}

	// Check if file is plain text (no null bytes in header)
	hasNullByte := false
	for _, b := range header {
		if b == 0 {
			hasNullByte = true
			break
		}
	}
	if hasNullByte {
		return false, "binary content (null bytes detected)"
	}

	// Default: assume it might be a shell script if it's text without extension
	// This is conservative - we'd rather lint and fail than skip a real script
	return true, "assumed text file"
}
