package smoketest

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultPlatformResolver implements PlatformResolver.
type DefaultPlatformResolver struct {
	executor         ProcessExecutor
	config           Config
	envReader        EnvironmentReader
	fs               FileSystem
	platformOverride string // For testing cross-platform behavior
}

// NewPlatformResolver creates a new platform resolver.
func NewPlatformResolver(executor ProcessExecutor, config Config, envReader EnvironmentReader, fs FileSystem) *DefaultPlatformResolver {
	return &DefaultPlatformResolver{
		executor:  executor,
		config:    config,
		envReader: envReader,
		fs:        fs,
	}
}

// SetPlatformOverride allows testing cross-platform behavior by overriding
// the detected platform. Pass an empty string to clear the override.
func (r *DefaultPlatformResolver) SetPlatformOverride(platform string) {
	r.platformOverride = platform
}

// CurrentPlatform returns the current platform identifier.
func (r *DefaultPlatformResolver) CurrentPlatform() string {
	if r.platformOverride != "" {
		return r.platformOverride
	}
	switch runtime.GOOS {
	case "windows":
		return "win"
	case "darwin":
		return "mac"
	default:
		return "linux"
	}
}

// effectivePlatform returns the platform for headless wrapper checks.
// This respects the override for testing.
func (r *DefaultPlatformResolver) effectivePlatform() string {
	if r.platformOverride != "" {
		return r.platformOverride
	}
	switch runtime.GOOS {
	case "windows":
		return "win"
	case "darwin":
		return "mac"
	default:
		return "linux"
	}
}

// ResolveCommand determines the command, args, and display string for running a smoke test.
func (r *DefaultPlatformResolver) ResolveCommand(platform, artifactPath string) (string, []string, string, error) {
	switch platform {
	case "linux":
		return r.resolveLinuxCommand(artifactPath)
	case "win":
		return r.resolveWindowsCommand(artifactPath)
	case "mac":
		return r.resolveMacCommand(artifactPath)
	default:
		return "", nil, "", fmt.Errorf("unsupported platform for smoke test: %s", platform)
	}
}

// RequiresHeadlessWrapper checks if a headless wrapper (xvfb-run) is needed and available.
func (r *DefaultPlatformResolver) RequiresHeadlessWrapper() (bool, string, []string, error) {
	// Only needed on Linux without DISPLAY
	// Use effectivePlatform to support testing with platform override
	platform := r.effectivePlatform()
	if platform != "linux" {
		return false, "", nil, nil
	}

	display := r.envReader.GetEnv("DISPLAY")
	if display != "" {
		return false, "", nil, nil
	}

	// Check if xvfb-run is available
	_, err := r.executor.LookPath(r.config.XvfbCommand)
	if err != nil {
		return true, "", nil, fmt.Errorf("DISPLAY is not set and %s is unavailable; cannot run Electron smoke test headlessly", r.config.XvfbCommand)
	}

	return true, r.config.XvfbCommand, []string{"-a"}, nil
}

func (r *DefaultPlatformResolver) resolveLinuxCommand(artifactPath string) (string, []string, string, error) {
	if !strings.HasSuffix(artifactPath, ".AppImage") {
		return "", nil, "", fmt.Errorf("unsupported linux artifact for smoke test: %s", filepath.Base(artifactPath))
	}

	if err := r.ensureExecutable(artifactPath); err != nil {
		return "", nil, "", fmt.Errorf("failed to set AppImage executable bit: %w", err)
	}

	args := []string{"--smoke-test"}
	display := fmt.Sprintf("%s --smoke-test", artifactPath)
	return artifactPath, args, display, nil
}

func (r *DefaultPlatformResolver) resolveWindowsCommand(artifactPath string) (string, []string, string, error) {
	if !strings.HasSuffix(strings.ToLower(artifactPath), ".exe") {
		return "", nil, "", fmt.Errorf("unsupported windows artifact for smoke test: %s", filepath.Base(artifactPath))
	}

	args := []string{"--smoke-test"}
	display := fmt.Sprintf("%s --smoke-test", artifactPath)
	return artifactPath, args, display, nil
}

func (r *DefaultPlatformResolver) resolveMacCommand(artifactPath string) (string, []string, string, error) {
	if !strings.HasSuffix(artifactPath, ".app") {
		return "", nil, "", fmt.Errorf("unsupported macOS artifact for smoke test: %s", filepath.Base(artifactPath))
	}

	executable, err := r.resolveMacAppExecutable(artifactPath)
	if err != nil {
		return "", nil, "", err
	}

	args := []string{"--smoke-test"}
	display := fmt.Sprintf("%s --smoke-test", executable)
	return executable, args, display, nil
}

func (r *DefaultPlatformResolver) resolveMacAppExecutable(appPath string) (string, error) {
	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	entries, err := r.fs.ReadDir(macosDir)
	if err != nil {
		return "", fmt.Errorf("failed to read app bundle: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		executable := filepath.Join(macosDir, entry.Name())
		if err := r.ensureExecutable(executable); err != nil {
			return "", fmt.Errorf("failed to make app executable: %w", err)
		}
		return executable, nil
	}

	return "", fmt.Errorf("no executable found in %s", macosDir)
}

func (r *DefaultPlatformResolver) ensureExecutable(path string) error {
	info, err := r.fs.Stat(path)
	if err != nil {
		return err
	}

	mode := info.Mode()
	if mode&0o111 != 0 {
		return nil // Already executable
	}

	return r.fs.Chmod(path, mode|0o111)
}
