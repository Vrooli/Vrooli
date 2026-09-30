// Package build provides cross-compilation capabilities for desktop bundle services.
package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"deployment-manager/bundles"
)

// Platform represents a target platform for cross-compilation.
type Platform struct {
	Name   string // e.g., "linux-x64", "darwin-arm64", "win-x64"
	GOOS   string // Go OS identifier
	GOARCH string // Go architecture identifier
	Ext    string // Binary extension (empty or ".exe")
}

// SupportedPlatforms defines the platforms we can cross-compile for.
var SupportedPlatforms = []Platform{
	{Name: "linux-x64", GOOS: "linux", GOARCH: "amd64", Ext: ""},
	{Name: "linux-arm64", GOOS: "linux", GOARCH: "arm64", Ext: ""},
	{Name: "darwin-x64", GOOS: "darwin", GOARCH: "amd64", Ext: ""},
	{Name: "darwin-arm64", GOOS: "darwin", GOARCH: "arm64", Ext: ""},
	{Name: "win-x64", GOOS: "windows", GOARCH: "amd64", Ext: ".exe"},
}

// RustTargets maps platform names to Rust target triples.
var RustTargets = map[string]string{
	"linux-x64":    "x86_64-unknown-linux-gnu",
	"linux-arm64":  "aarch64-unknown-linux-gnu",
	"darwin-x64":   "x86_64-apple-darwin",
	"darwin-arm64": "aarch64-apple-darwin",
	"win-x64":      "x86_64-pc-windows-msvc",
}

// BuildConfig is an alias for bundles.BuildConfig for convenience.
type BuildConfig = bundles.BuildConfig

// BuildResult represents the result of building for a single platform.
type BuildResult struct {
	Platform   string `json:"platform"`
	OutputPath string `json:"output_path"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Duration   string `json:"duration,omitempty"`
}

// BuildAllResult represents the results of building for all platforms.
type BuildAllResult struct {
	ServiceID    string        `json:"service_id"`
	Results      []BuildResult `json:"results"`
	AllSucceeded bool          `json:"all_succeeded"`
}

// Builder handles cross-compilation of service binaries.
type Builder struct {
	workDir string
	log     func(string, map[string]interface{})
}

// NewBuilder creates a new Builder.
func NewBuilder(workDir string, log func(string, map[string]interface{})) *Builder {
	return &Builder{
		workDir: workDir,
		log:     log,
	}
}

// BuildAll compiles a service for all supported platforms.
func (b *Builder) BuildAll(ctx context.Context, serviceID string, config *BuildConfig, platforms []string) (*BuildAllResult, error) {
	if config == nil {
		return nil, fmt.Errorf("build config is required")
	}

	// Filter platforms if specified
	targetPlatforms := SupportedPlatforms
	if len(platforms) > 0 {
		targetPlatforms = filterPlatforms(platforms)
	}

	result := &BuildAllResult{
		ServiceID:    serviceID,
		Results:      make([]BuildResult, 0, len(targetPlatforms)),
		AllSucceeded: true,
	}

	for _, platform := range targetPlatforms {
		buildResult := b.buildForPlatform(ctx, serviceID, config, platform)
		result.Results = append(result.Results, buildResult)
		if !buildResult.Success {
			result.AllSucceeded = false
		}
	}

	return result, nil
}

// buildForPlatform compiles for a single platform.
func (b *Builder) buildForPlatform(ctx context.Context, serviceID string, config *BuildConfig, platform Platform) BuildResult {
	result := BuildResult{
		Platform: platform.Name,
		Success:  false,
	}

	// Calculate output path
	outputPath := b.resolveOutputPath(config, platform)
	result.OutputPath = outputPath

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		result.Error = fmt.Sprintf("failed to create output dir: %v", err)
		return result
	}

	// Build based on type
	var err error
	switch strings.ToLower(config.Type) {
	case "go":
		err = b.buildGo(ctx, config, platform, outputPath)
	case "rust":
		err = b.buildRust(ctx, config, platform, outputPath)
	case "npm":
		err = b.buildNpm(ctx, config, platform, outputPath)
	case "custom":
		err = b.buildCustom(ctx, config, platform, outputPath)
	default:
		err = fmt.Errorf("unsupported build type: %s", config.Type)
	}

	if err != nil {
		result.Error = err.Error()
		b.log("error", map[string]interface{}{
			"msg":      "build failed",
			"service":  serviceID,
			"platform": platform.Name,
			"error":    err.Error(),
		})
	} else {
		result.Success = true
		b.log("info", map[string]interface{}{
			"msg":      "build succeeded",
			"service":  serviceID,
			"platform": platform.Name,
			"output":   outputPath,
		})
	}

	return result
}

// buildGo compiles a Go project for the target platform.
func (b *Builder) buildGo(ctx context.Context, config *BuildConfig, platform Platform, outputPath string) error {
	sourceDir := filepath.Join(b.workDir, config.SourceDir)

	// Determine entry point
	entryPoint := config.EntryPoint
	if entryPoint == "" {
		entryPoint = "."
	}

	// Build command
	args := []string{"build", "-o", outputPath}

	// Add CGO_ENABLED=0 for static binaries by default
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOOS=%s", platform.GOOS))
	env = append(env, fmt.Sprintf("GOARCH=%s", platform.GOARCH))

	// Check if CGO is explicitly set in config
	cgoSet := false
	for k := range config.Env {
		if k == "CGO_ENABLED" {
			cgoSet = true
			break
		}
	}
	if !cgoSet {
		env = append(env, "CGO_ENABLED=0")
	}

	// Add custom env from config
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Add extra args (e.g., -ldflags)
	args = append(args, config.Args...)
	args = append(args, entryPoint)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = sourceDir
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %v\n%s", err, string(output))
	}

	return nil
}

// buildRust compiles a Rust project for the target platform.
func (b *Builder) buildRust(ctx context.Context, config *BuildConfig, platform Platform, outputPath string) error {
	sourceDir := filepath.Join(b.workDir, config.SourceDir)

	target, ok := RustTargets[platform.Name]
	if !ok {
		return fmt.Errorf("unsupported Rust target for platform: %s", platform.Name)
	}

	args := []string{"build", "--release", "--target", target}
	args = append(args, config.Args...)

	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Dir = sourceDir

	// Add custom env
	env := os.Environ()
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cargo build failed: %v\n%s", err, string(output))
	}

	// Copy from Rust's target directory to expected output path
	// Rust outputs to target/<target-triple>/release/<binary-name>
	entryPoint := config.EntryPoint
	if entryPoint == "" {
		entryPoint = filepath.Base(config.SourceDir)
	}
	rustOutput := filepath.Join(sourceDir, "target", target, "release", entryPoint)
	if platform.Ext != "" {
		rustOutput += platform.Ext
	}

	if err := copyFile(rustOutput, outputPath); err != nil {
		return fmt.Errorf("failed to copy Rust output: %v", err)
	}

	return nil
}

// buildNpm builds an npm project that produces platform binaries.
func (b *Builder) buildNpm(ctx context.Context, config *BuildConfig, platform Platform, outputPath string) error {
	sourceDir := filepath.Join(b.workDir, config.SourceDir)

	// Run npm install first
	installCmd := exec.CommandContext(ctx, "npm", "install")
	installCmd.Dir = sourceDir
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install failed: %v\n%s", err, string(output))
	}

	// Run npm build with platform env vars
	buildArgs := []string{"run", "build"}
	buildArgs = append(buildArgs, config.Args...)

	env := os.Environ()
	env = append(env, fmt.Sprintf("TARGET_PLATFORM=%s", platform.Name))
	env = append(env, fmt.Sprintf("TARGET_OS=%s", platform.GOOS))
	env = append(env, fmt.Sprintf("TARGET_ARCH=%s", platform.GOARCH))
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	buildCmd := exec.CommandContext(ctx, "npm", buildArgs...)
	buildCmd.Dir = sourceDir
	buildCmd.Env = env

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm build failed: %v\n%s", err, string(output))
	}

	return nil
}

// buildCustom runs a custom build command.
func (b *Builder) buildCustom(ctx context.Context, config *BuildConfig, platform Platform, outputPath string) error {
	if len(config.Args) == 0 {
		return fmt.Errorf("custom build requires at least one arg (the command)")
	}

	sourceDir := filepath.Join(b.workDir, config.SourceDir)

	// Replace placeholders in args
	args := make([]string, len(config.Args))
	for i, arg := range config.Args {
		args[i] = b.replacePlaceholders(arg, platform, outputPath)
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = sourceDir

	env := os.Environ()
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("custom build failed: %v\n%s", err, string(output))
	}

	return nil
}

// ResolveOutputPath calculates the output path for a platform.
func ResolveOutputPath(workDir string, config *BuildConfig, platform Platform) string {
	pattern := config.OutputPattern
	if pattern == "" {
		// Default pattern
		pattern = "bin/{{platform}}/service{{ext}}"
	}

	builder := &Builder{workDir: workDir}
	return filepath.Join(workDir, builder.replacePlaceholders(pattern, platform, ""))
}

// resolveOutputPath calculates the output path for a platform.
func (b *Builder) resolveOutputPath(config *BuildConfig, platform Platform) string {
	return ResolveOutputPath(b.workDir, config, platform)
}

// replacePlaceholders substitutes template variables in a string.
func (b *Builder) replacePlaceholders(s string, platform Platform, output string) string {
	s = strings.ReplaceAll(s, "{{platform}}", platform.Name)
	s = strings.ReplaceAll(s, "{{goos}}", platform.GOOS)
	s = strings.ReplaceAll(s, "{{goarch}}", platform.GOARCH)
	s = strings.ReplaceAll(s, "{{ext}}", platform.Ext)
	s = strings.ReplaceAll(s, "{{output}}", output)
	return s
}

// filterPlatforms filters the supported platforms by name.
func filterPlatforms(names []string) []Platform {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[strings.ToLower(n)] = true
	}

	var result []Platform
	for _, p := range SupportedPlatforms {
		if nameSet[p.Name] {
			result = append(result, p)
		}
	}
	return result
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o755)
}
