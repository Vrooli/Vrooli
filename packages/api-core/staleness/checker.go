package staleness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Environment variable names
const (
	// SkipEnvVar can be set to "true" to skip staleness checking entirely.
	SkipEnvVar = "VROOLI_API_SKIP_STALE_CHECK"

	// rebuildLoopEnvVar is set internally after a rebuild to detect infinite loops.
	rebuildLoopEnvVar = "API_CORE_REBUILD_TIMESTAMP"
)

// CheckerConfig configures the staleness checker behavior.
type CheckerConfig struct {
	// APIDir is the directory containing the API source code.
	// Default: directory containing the binary (os.Executable())
	APIDir string

	// BinaryPath is the path to the API binary being checked.
	// Default: os.Executable()
	BinaryPath string

	// Logger for output messages. Default: fmt.Fprintf(os.Stderr, ...)
	Logger func(format string, args ...interface{})

	// Disabled completely skips all staleness checking.
	Disabled bool

	// SkipRebuild only logs staleness without attempting rebuild.
	// Useful for debugging or when rebuild should be handled externally.
	SkipRebuild bool

	// CommandRunner overrides exec.Cmd.Run() for testing.
	CommandRunner func(cmd *exec.Cmd) error

	// Reexec overrides syscall.Exec for testing.
	Reexec func(binary string, args []string, env []string) error

	// LookPath overrides exec.LookPath for testing.
	LookPath func(file string) (string, error)
}

// Checker detects stale API binaries and optionally rebuilds them.
type Checker struct {
	cfg CheckerConfig
}

// NewChecker creates a new staleness checker with the given configuration.
func NewChecker(cfg CheckerConfig) *Checker {
	return &Checker{cfg: cfg}
}

// CheckAndMaybeRebuild checks if the API binary is stale and optionally rebuilds.
// Returns true if the process was restarted after rebuild.
//
// This should be called at the very start of main(), before any initialization:
//
//	func main() {
//	    checker := staleness.NewChecker(staleness.CheckerConfig{})
//	    if checker.CheckAndMaybeRebuild() {
//	        return // Process was re-exec'd
//	    }
//	    // ... rest of initialization
//	}
func (c *Checker) CheckAndMaybeRebuild() bool {
	// Check if disabled via config or environment
	if c.cfg.Disabled || os.Getenv(SkipEnvVar) == "true" {
		return false
	}

	// Resolve paths
	binaryPath, apiDir, err := c.resolvePaths()
	if err != nil {
		c.log("api-core: unable to resolve paths for staleness check: %v\n", err)
		return false
	}

	// Get binary modification time
	binaryTime := GetModTime(binaryPath)
	if binaryTime.IsZero() {
		c.log("api-core: unable to get binary modification time\n")
		return false
	}

	// Check staleness
	isStale, reason := c.checkStaleness(binaryPath, apiDir, binaryTime)
	if !isStale {
		return false
	}

	// Detect rebuild loops
	if c.detectLoop() {
		c.log("api-core: rebuild loop detected, skipping auto-rebuild\n")
		c.log("  Staleness reason: %s\n", reason)
		return false
	}

	// Log staleness
	c.log("api-core: binary is stale (%s)\n", reason)

	if c.cfg.SkipRebuild {
		return false
	}

	// Attempt auto-rebuild
	return c.autoRebuild(binaryPath, apiDir)
}

// resolvePaths determines the binary path and API source directory.
func (c *Checker) resolvePaths() (binaryPath, apiDir string, err error) {
	// Get binary path
	binaryPath = c.cfg.BinaryPath
	if binaryPath == "" {
		binaryPath, err = os.Executable()
		if err != nil {
			return "", "", fmt.Errorf("get executable path: %w", err)
		}
	}

	// Resolve symlinks to get the real path
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve symlinks: %w", err)
	}

	// Get API directory
	apiDir = c.cfg.APIDir
	if apiDir == "" {
		apiDir = filepath.Dir(binaryPath)
	}

	// Verify go.mod exists (confirms this is a Go module)
	goModPath := filepath.Join(apiDir, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		return "", "", fmt.Errorf("no go.mod in API directory: %s", apiDir)
	}

	return binaryPath, apiDir, nil
}

// checkStaleness compares file timestamps to determine if the binary is stale.
func (c *Checker) checkStaleness(binaryPath, apiDir string, binaryTime time.Time) (isStale bool, reason string) {
	// Check source files (*.go)
	if found, file := CheckNewerFiles(apiDir, "*.go", binaryTime); found {
		relFile, _ := filepath.Rel(apiDir, file)
		if relFile == "" {
			relFile = file
		}
		return true, fmt.Sprintf("source file modified: %s", relFile)
	}

	// Check go.mod
	goModPath := filepath.Join(apiDir, "go.mod")
	if IsFileNewer(goModPath, binaryTime) {
		return true, "go.mod modified"
	}

	// Check go.sum
	goSumPath := filepath.Join(apiDir, "go.sum")
	if IsFileNewer(goSumPath, binaryTime) {
		return true, "go.sum modified"
	}

	// Check local replace directives
	replacePaths, err := ParseReplaceDirectives(goModPath)
	if err == nil {
		for _, relPath := range replacePaths {
			absPath := filepath.Join(apiDir, relPath)

			// Check if it's a directory
			info, err := os.Stat(absPath)
			if err != nil || !info.IsDir() {
				continue
			}

			// Check *.go files in replaced package
			if found, file := CheckNewerFiles(absPath, "*.go", binaryTime); found {
				return true, fmt.Sprintf("dependency modified: %s (%s)", relPath, filepath.Base(file))
			}

			// Check go.mod in replaced package
			depGoMod := filepath.Join(absPath, "go.mod")
			if IsFileNewer(depGoMod, binaryTime) {
				return true, fmt.Sprintf("dependency go.mod modified: %s", relPath)
			}

			// Check go.sum in replaced package (dependency version updates)
			depGoSum := filepath.Join(absPath, "go.sum")
			if IsFileNewer(depGoSum, binaryTime) {
				return true, fmt.Sprintf("dependency go.sum modified: %s", relPath)
			}
		}
	}

	return false, ""
}

// detectLoop checks if we're in an infinite rebuild loop.
func (c *Checker) detectLoop() bool {
	lastRebuildStr := os.Getenv(rebuildLoopEnvVar)
	if lastRebuildStr == "" {
		return false
	}

	// Parse the timestamp
	lastRebuild, err := strconv.ParseInt(lastRebuildStr, 10, 64)
	if err != nil {
		return false
	}

	// If we rebuilt within the last 60 seconds and we're still stale, it's a loop
	return time.Since(time.Unix(lastRebuild, 0)) < 60*time.Second
}

// autoRebuild attempts to rebuild the binary and re-exec.
func (c *Checker) autoRebuild(binaryPath, apiDir string) bool {
	// Check if go is available
	lookPath := c.cfg.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if _, err := lookPath("go"); err != nil {
		c.log("api-core: 'go' not found in PATH, cannot auto-rebuild\n")
		return false
	}

	// Build the binary
	c.log("api-core: rebuilding binary...\n")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = apiDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	runner := c.cfg.CommandRunner
	if runner == nil {
		runner = func(cmd *exec.Cmd) error { return cmd.Run() }
	}

	if err := runner(cmd); err != nil {
		c.log("api-core: rebuild failed: %v\n", err)
		return false
	}

	c.log("api-core: rebuild successful, restarting...\n")

	// Re-exec with loop detection env var
	reexec := c.cfg.Reexec
	if reexec == nil {
		reexec = defaultReexec
	}

	// Set loop detection env var with current timestamp
	env := append(os.Environ(), fmt.Sprintf("%s=%d", rebuildLoopEnvVar, time.Now().Unix()))

	if err := reexec(binaryPath, os.Args, env); err != nil {
		c.log("api-core: failed to restart: %v\n", err)
		return false
	}

	return true
}

// defaultReexec replaces the current process with a new execution of the binary.
func defaultReexec(binary string, args []string, env []string) error {
	return syscall.Exec(binary, args, env)
}

// log outputs a message using the configured logger or stderr.
func (c *Checker) log(format string, args ...interface{}) {
	if c.cfg.Logger != nil {
		c.cfg.Logger(format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}
