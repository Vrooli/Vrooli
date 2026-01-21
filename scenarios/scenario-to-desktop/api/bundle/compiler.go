package bundle

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// defaultRuntimeBuilder is the default implementation of RuntimeBuilder.
type defaultRuntimeBuilder struct{}

// Build compiles a runtime binary for the specified platform.
func (b *defaultRuntimeBuilder) Build(srcDir, outPath, goos, goarch, target string) error {
	args := []string{"build", "-o", outPath}
	switch target {
	case "runtime":
		args = append(args, "./cmd/runtime")
	case "runtimectl":
		args = append(args, "./cmd/runtimectl")
	default:
		return fmt.Errorf("unknown runtime target %q", target)
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+goos,
		"GOARCH="+goarch,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// defaultServiceCompiler is the default implementation of ServiceCompiler.
type defaultServiceCompiler struct {
	platform PlatformResolver
	fileOps  FileOperations
}

// Compile compiles a service binary for the specified platform.
// scenarioRoot is the path to the scenario directory (where source folders like api/, ui/, etc. are located).
func (c *defaultServiceCompiler) Compile(svc bundlemanifest.Service, platform, scenarioRoot string) (string, error) {
	if svc.Build == nil {
		return "", errors.New("no build configuration")
	}

	build := svc.Build
	goos, goarch, err := c.platform.ParseKey(platform)
	if err != nil {
		return "", err
	}

	// Resolve source directory relative to scenario root
	srcDir := filepath.Join(scenarioRoot, build.SourceDir)
	if _, err := os.Stat(srcDir); err != nil {
		return "", fmt.Errorf("source directory not found: %s", srcDir)
	}

	// Determine output path
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}

	outputPath := build.OutputPattern
	if outputPath == "" {
		// Default output pattern based on service ID
		outputPath = fmt.Sprintf("bin/%s/%s%s", platform, svc.ID, ext)
	} else {
		// Replace placeholders in output pattern
		outputPath = strings.ReplaceAll(outputPath, "{{platform}}", platform)
		outputPath = strings.ReplaceAll(outputPath, "{{ext}}", ext)
	}

	absOutput := filepath.Join(srcDir, outputPath)
	if err := os.MkdirAll(filepath.Dir(absOutput), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	// Build based on type
	switch strings.ToLower(build.Type) {
	case "go":
		return absOutput, compileGoBinary(srcDir, absOutput, goos, goarch, build)
	case "rust":
		return absOutput, compileRustBinary(srcDir, absOutput, goos, goarch, build, c.fileOps)
	case "npm", "node":
		return absOutput, compileNpmBinary(srcDir, absOutput, goos, goarch, build)
	case "custom":
		return absOutput, compileCustomBinary(srcDir, absOutput, goos, goarch, build)
	default:
		return "", fmt.Errorf("unsupported build type: %s", build.Type)
	}
}

// compileGoBinary compiles a Go binary for the specified platform.
func compileGoBinary(srcDir, outPath, goos, goarch string, build *bundlemanifest.BuildConfig) error {
	args := []string{"build", "-o", outPath}

	// Add any custom build args
	if len(build.Args) > 0 {
		args = append(args, build.Args...)
	}

	// Add entry point (default to current directory)
	entryPoint := build.EntryPoint
	if entryPoint == "" {
		entryPoint = "."
	}
	args = append(args, entryPoint)

	cmd := exec.Command("go", args...)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+goos,
		"GOARCH="+goarch,
	)

	// Add custom environment variables
	for k, v := range build.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// compileRustBinary compiles a Rust binary for the specified platform.
func compileRustBinary(srcDir, outPath, goos, goarch string, build *bundlemanifest.BuildConfig, fileOps FileOperations) error {
	// Map Go OS/arch to Rust target triple
	target, err := rustTarget(goos, goarch)
	if err != nil {
		return err
	}

	args := []string{"build", "--release", "--target", target}

	// Add any custom build args
	if len(build.Args) > 0 {
		args = append(args, build.Args...)
	}

	cmd := exec.Command("cargo", args...)
	cmd.Dir = srcDir

	// Add custom environment variables
	for k, v := range build.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cargo build failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// Cargo outputs to target/<triple>/release/<binary>
	binaryName := filepath.Base(srcDir)
	if build.EntryPoint != "" {
		binaryName = filepath.Base(build.EntryPoint)
	}
	if goos == "windows" {
		binaryName += ".exe"
	}

	cargoOutput := filepath.Join(srcDir, "target", target, "release", binaryName)
	if err := fileOps.CopyFile(cargoOutput, outPath); err != nil {
		return fmt.Errorf("copy rust binary: %w", err)
	}
	return nil
}

// rustTarget returns the Rust target triple for the given OS/arch.
func rustTarget(goos, goarch string) (string, error) {
	targets := map[string]map[string]string{
		"linux": {
			"amd64": "x86_64-unknown-linux-gnu",
			"arm64": "aarch64-unknown-linux-gnu",
		},
		"darwin": {
			"amd64": "x86_64-apple-darwin",
			"arm64": "aarch64-apple-darwin",
		},
		"windows": {
			"amd64": "x86_64-pc-windows-msvc",
			"arm64": "aarch64-pc-windows-msvc",
		},
	}

	osTargets, ok := targets[goos]
	if !ok {
		return "", fmt.Errorf("unsupported OS for Rust: %s", goos)
	}
	target, ok := osTargets[goarch]
	if !ok {
		return "", fmt.Errorf("unsupported arch for Rust on %s: %s", goos, goarch)
	}
	return target, nil
}

// compileNpmBinary builds a Node.js application and prepares it for bundling.
// For Electron apps, Node.js services run via the built-in Node.js runtime, not as native binaries.
// This function builds the project and copies the result to outPath as a directory.
func compileNpmBinary(srcDir, outPath, goos, goarch string, build *bundlemanifest.BuildConfig) error {
	// First, install dependencies (including devDependencies for build)
	installCmd := exec.Command("npm", "install")
	installCmd.Dir = srcDir
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// Build using the provided command or default to npm run build
	buildArgs := []string{"run", "build"}
	if len(build.Args) > 0 {
		buildArgs = build.Args
	}

	buildCmd := exec.Command("npm", buildArgs...)
	buildCmd.Dir = srcDir

	// Start with parent environment (required for PATH, npm to find node_modules/.bin, etc.)
	buildCmd.Env = os.Environ()

	// Add custom environment variables
	for k, v := range build.Env {
		buildCmd.Env = append(buildCmd.Env, k+"="+v)
	}
	// Set platform hints
	buildCmd.Env = append(buildCmd.Env,
		"TARGET_OS="+goos,
		"TARGET_ARCH="+goarch,
	)

	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm build failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// For Node.js projects, the build outputs JavaScript, not a native binary.
	// Verify dist/ was created
	distDir := filepath.Join(srcDir, "dist")
	if _, err := os.Stat(distDir); err != nil {
		return fmt.Errorf("npm build did not produce dist/ directory at %s", distDir)
	}

	// Look for common entry points to verify the build succeeded
	entryPoints := []string{"server.js", "index.js", "main.js"}
	var entryPoint string
	for _, ep := range entryPoints {
		epPath := filepath.Join(distDir, ep)
		if _, err := os.Stat(epPath); err == nil {
			entryPoint = ep
			break
		}
	}
	if entryPoint == "" {
		return fmt.Errorf("npm build did not produce a recognizable entry point (server.js, index.js, or main.js) in %s", distDir)
	}

	// Install production dependencies only for the bundle
	prodInstallCmd := exec.Command("npm", "install", "--omit=dev")
	prodInstallCmd.Dir = srcDir
	if output, err := prodInstallCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install --omit=dev failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// Create output directory structure
	// outPath is like: bin/playwright-driver/linux/browser-automation-studio-playwright-driver
	// We'll create a directory there containing dist/, node_modules/, and package.json
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Copy dist/ to output
	destDist := filepath.Join(outPath, "dist")
	if err := copyDir(distDir, destDist); err != nil {
		return fmt.Errorf("copy dist directory: %w", err)
	}

	// Copy node_modules/ to output (production deps only)
	srcNodeModules := filepath.Join(srcDir, "node_modules")
	if _, err := os.Stat(srcNodeModules); err == nil {
		destNodeModules := filepath.Join(outPath, "node_modules")
		if err := copyDir(srcNodeModules, destNodeModules); err != nil {
			return fmt.Errorf("copy node_modules directory: %w", err)
		}
	}

	// Copy package.json for runtime metadata
	srcPkg := filepath.Join(srcDir, "package.json")
	if _, err := os.Stat(srcPkg); err == nil {
		destPkg := filepath.Join(outPath, "package.json")
		if err := copyFile(srcPkg, destPkg); err != nil {
			return fmt.Errorf("copy package.json: %w", err)
		}
	}

	// Create a run.sh script that the runtime can execute
	runScript := filepath.Join(outPath, "run.sh")
	scriptContent := fmt.Sprintf(`#!/bin/sh
cd "$(dirname "$0")"
exec node dist/%s "$@"
`, entryPoint)
	if err := os.WriteFile(runScript, []byte(scriptContent), 0o755); err != nil {
		return fmt.Errorf("write run script: %w", err)
	}

	// For Windows, also create run.cmd
	if goos == "windows" {
		runCmd := filepath.Join(outPath, "run.cmd")
		cmdContent := fmt.Sprintf(`@echo off
cd /d "%%~dp0"
node dist\%s %%*
`, entryPoint)
		if err := os.WriteFile(runCmd, []byte(cmdContent), 0o755); err != nil {
			return fmt.Errorf("write run.cmd: %w", err)
		}
	}

	return nil
}

// copyDir recursively copies a directory, preserving symlinks
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		// Check if it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			// Read the symlink target
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return err
			}
			// Remove existing file/symlink if it exists
			_ = os.Remove(dstPath)
			// Create symlink at destination
			return os.Symlink(linkTarget, dstPath)
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}

	// Preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// compileCustomBinary runs a custom build command.
func compileCustomBinary(srcDir, outPath, goos, goarch string, build *bundlemanifest.BuildConfig) error {
	if len(build.Args) == 0 {
		return errors.New("custom build type requires args with command and arguments")
	}

	// First arg is the command, rest are arguments
	cmdName := build.Args[0]
	cmdArgs := build.Args[1:]

	// Replace placeholders in arguments
	for i, arg := range cmdArgs {
		arg = strings.ReplaceAll(arg, "{{platform}}", goos+"-"+goarch)
		arg = strings.ReplaceAll(arg, "{{goos}}", goos)
		arg = strings.ReplaceAll(arg, "{{goarch}}", goarch)
		arg = strings.ReplaceAll(arg, "{{output}}", outPath)
		ext := ""
		if goos == "windows" {
			ext = ".exe"
		}
		arg = strings.ReplaceAll(arg, "{{ext}}", ext)
		cmdArgs[i] = arg
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(),
		"GOOS="+goos,
		"GOARCH="+goarch,
		"OUTPUT_PATH="+outPath,
	)

	// Add custom environment variables
	for k, v := range build.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("custom build failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// Verify output was created
	if _, err := os.Stat(outPath); err != nil {
		return fmt.Errorf("custom build did not produce expected output at %s", outPath)
	}

	return nil
}
