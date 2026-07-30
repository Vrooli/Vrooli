// Package smoketest provides smoke testing services for desktop applications.
package smoketest

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// PrerequisiteKind categorizes different types of prerequisites.
type PrerequisiteKind int

const (
	// PrereqDisplay checks for display availability (X11/Wayland on Linux).
	PrereqDisplay PrerequisiteKind = iota
	// PrereqArtifactExecutable checks if the artifact is executable.
	PrereqArtifactExecutable
	// PrereqDiskSpace checks for minimum free disk space.
	PrereqDiskSpace
	// PrereqPort checks if a required port is available.
	PrereqPort
)

// String returns the string representation of a PrerequisiteKind.
func (k PrerequisiteKind) String() string {
	switch k {
	case PrereqDisplay:
		return "display"
	case PrereqArtifactExecutable:
		return "artifact_executable"
	case PrereqDiskSpace:
		return "disk_space"
	case PrereqPort:
		return "port"
	default:
		return "unknown"
	}
}

// PrerequisiteResult represents the result of a prerequisite check.
type PrerequisiteResult struct {
	// Kind identifies the type of prerequisite checked.
	Kind PrerequisiteKind

	// Passed indicates if the prerequisite was satisfied.
	Passed bool

	// Message provides details about the check result.
	Message string

	// Fatal indicates if failure should stop the smoke test.
	Fatal bool

	// Suggestion provides guidance on how to resolve a failure.
	Suggestion string
}

// PrerequisiteChecker validates system prerequisites before smoke test execution.
type PrerequisiteChecker struct {
	envReader  EnvironmentReader
	fileSystem FileSystem
	executor   ProcessExecutor
}

// NewPrerequisiteChecker creates a new prerequisite checker.
func NewPrerequisiteChecker(envReader EnvironmentReader, fileSystem FileSystem, executor ProcessExecutor) *PrerequisiteChecker {
	return &PrerequisiteChecker{
		envReader:  envReader,
		fileSystem: fileSystem,
		executor:   executor,
	}
}

// CheckAll runs all prerequisite checks for a smoke test.
func (c *PrerequisiteChecker) CheckAll(artifactPath, platform string, telemetryPort int) []PrerequisiteResult {
	results := make([]PrerequisiteResult, 0, 4)

	// Check display availability (Linux only)
	if platform == "linux" || runtime.GOOS == "linux" {
		results = append(results, c.CheckDisplayAvailable())
	}

	// Check artifact is executable
	results = append(results, c.CheckArtifactExecutable(artifactPath))

	// Check disk space in artifact directory
	results = append(results, c.CheckDiskSpace(artifactPath, 100*1024*1024)) // 100MB minimum

	// Check telemetry port is available
	if telemetryPort > 0 {
		results = append(results, c.CheckPortAvailable(telemetryPort))
	}

	return results
}

// HasFatalFailure returns true if any prerequisite check failed fatally.
func (c *PrerequisiteChecker) HasFatalFailure(results []PrerequisiteResult) bool {
	for _, r := range results {
		if !r.Passed && r.Fatal {
			return true
		}
	}
	return false
}

// CheckDisplayAvailable checks if a display is available for GUI apps on Linux.
func (c *PrerequisiteChecker) CheckDisplayAvailable() PrerequisiteResult {
	result := PrerequisiteResult{
		Kind:  PrereqDisplay,
		Fatal: false, // Not fatal since xvfb-run can provide a virtual display
	}

	// Check DISPLAY environment variable
	display := c.envReader.GetEnv("DISPLAY")
	if display != "" {
		result.Passed = true
		result.Message = fmt.Sprintf("DISPLAY is set to %s", display)
		return result
	}

	// Check WAYLAND_DISPLAY for Wayland sessions
	waylandDisplay := c.envReader.GetEnv("WAYLAND_DISPLAY")
	if waylandDisplay != "" {
		result.Passed = true
		result.Message = fmt.Sprintf("WAYLAND_DISPLAY is set to %s", waylandDisplay)
		return result
	}

	// Check if xvfb-run is available as fallback
	if _, err := c.executor.LookPath("xvfb-run"); err == nil {
		result.Passed = true
		result.Message = "No display set, but xvfb-run is available"
		return result
	}

	result.Passed = false
	result.Message = "No display available and xvfb-run not found"
	result.Suggestion = "Set DISPLAY environment variable or install xvfb: sudo apt-get install xvfb"
	return result
}

// CheckArtifactExecutable checks if the artifact file exists and is executable.
func (c *PrerequisiteChecker) CheckArtifactExecutable(artifactPath string) PrerequisiteResult {
	result := PrerequisiteResult{
		Kind:  PrereqArtifactExecutable,
		Fatal: true, // Fatal - can't run without executable artifact
	}

	info, err := c.fileSystem.Stat(artifactPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Passed = false
			result.Message = fmt.Sprintf("Artifact does not exist: %s", artifactPath)
			result.Suggestion = "Verify the build stage completed successfully"
		} else {
			result.Passed = false
			result.Message = fmt.Sprintf("Cannot access artifact: %v", err)
			result.Suggestion = "Check file permissions"
		}
		return result
	}

	// Check if it's a directory (invalid)
	if info.IsDir() {
		result.Passed = false
		result.Message = fmt.Sprintf("Artifact path is a directory: %s", artifactPath)
		result.Suggestion = "Provide the path to the executable file, not the directory"
		return result
	}

	// On Unix-like systems, check executable permission
	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode&0o111 == 0 {
			result.Passed = false
			result.Message = fmt.Sprintf("Artifact is not executable: %s (mode: %o)", artifactPath, mode)
			result.Suggestion = fmt.Sprintf("Run: chmod +x %s", artifactPath)
			return result
		}
	}

	result.Passed = true
	result.Message = fmt.Sprintf("Artifact is executable: %s", artifactPath)
	return result
}

// CheckDiskSpace checks if there's sufficient free disk space.
func (c *PrerequisiteChecker) CheckDiskSpace(path string, minBytes uint64) PrerequisiteResult {
	result := PrerequisiteResult{
		Kind:  PrereqDiskSpace,
		Fatal: false, // Not fatal - smoke test might still work
	}

	// Get the directory of the path
	dir := path
	if info, err := c.fileSystem.Stat(path); err == nil && !info.IsDir() {
		// It's a file, get the parent directory
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' || path[i] == '\\' {
				dir = path[:i]
				break
			}
		}
	}

	availableBytes, err := availableDiskSpace(dir)
	if err != nil {
		result.Passed = true // Assume OK if we can't check
		result.Message = fmt.Sprintf("Could not check disk space: %v", err)
		return result
	}

	// Calculate available space
	minMB := minBytes / (1024 * 1024)
	availableMB := availableBytes / (1024 * 1024)

	if availableBytes < minBytes {
		result.Passed = false
		result.Message = fmt.Sprintf("Low disk space: %dMB available, %dMB required", availableMB, minMB)
		result.Suggestion = "Free up disk space before running smoke test"
		return result
	}

	result.Passed = true
	result.Message = fmt.Sprintf("Sufficient disk space: %dMB available", availableMB)
	return result
}

// CheckPortAvailable checks if a port is available for binding.
func (c *PrerequisiteChecker) CheckPortAvailable(port int) PrerequisiteResult {
	result := PrerequisiteResult{
		Kind:  PrereqPort,
		Fatal: false, // Not fatal - telemetry upload might still work
	}

	// Use ss or netstat to check port availability
	var output string
	var err error

	ctx := context.Background()
	if runtime.GOOS == "windows" {
		output, err = c.executor.Execute(ctx, "", "netstat", []string{"-an"}, nil, 0)
	} else {
		// Try ss first (modern Linux), fall back to netstat
		output, err = c.executor.Execute(ctx, "", "ss", []string{"-tlnp"}, nil, 0)
		if err != nil {
			output, err = c.executor.Execute(ctx, "", "netstat", []string{"-tlnp"}, nil, 0)
		}
	}

	if err != nil {
		result.Passed = true // Assume available if we can't check
		result.Message = fmt.Sprintf("Could not check port %d availability", port)
		return result
	}

	// Check if the port is in use
	portStr := fmt.Sprintf(":%d", port)
	if strings.Contains(output, portStr) {
		result.Passed = false
		result.Message = fmt.Sprintf("Port %d appears to be in use", port)
		result.Suggestion = fmt.Sprintf("Check what's using port %d: lsof -i :%d", port, port)
		return result
	}

	result.Passed = true
	result.Message = fmt.Sprintf("Port %d is available", port)
	return result
}

// FormatResults returns a human-readable summary of prerequisite results.
func FormatResults(results []PrerequisiteResult) string {
	var sb strings.Builder
	passed := 0
	failed := 0
	warnings := 0

	for _, r := range results {
		switch {
		case r.Passed:
			passed++
			sb.WriteString(fmt.Sprintf("✓ [%s] %s\n", r.Kind, r.Message))
		case r.Fatal:
			failed++
			sb.WriteString(fmt.Sprintf("✗ [%s] %s\n", r.Kind, r.Message))
			if r.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("  → %s\n", r.Suggestion))
			}
		default:
			warnings++
			sb.WriteString(fmt.Sprintf("⚠ [%s] %s\n", r.Kind, r.Message))
			if r.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("  → %s\n", r.Suggestion))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\nSummary: %d passed, %d failed, %d warnings\n", passed, failed, warnings))
	return sb.String()
}
