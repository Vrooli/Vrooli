package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// buildRuntimeBinary builds the runtime binary for a given platform.
func buildRuntimeBinary(srcDir, outPath, goos, goarch, target string) error {
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
func compileRustBinary(srcDir, outPath, goos, goarch string, build *bundlemanifest.BuildConfig) error {
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
	// Find the binary and copy to outPath
	binaryName := filepath.Base(srcDir)
	if build.EntryPoint != "" {
		binaryName = filepath.Base(build.EntryPoint)
	}
	if goos == "windows" {
		binaryName += ".exe"
	}

	cargoOutput := filepath.Join(srcDir, "target", target, "release", binaryName)
	if err := copyFile(cargoOutput, outPath); err != nil {
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

// compileNpmBinary builds a Node.js application using npm/pkg or similar bundler.
func compileNpmBinary(srcDir, outPath, goos, goarch string, build *bundlemanifest.BuildConfig) error {
	// First, install dependencies
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

	// npm build typically outputs to dist/ - we need to check if output exists
	// For Node.js single-binary bundlers like pkg, the output path should be configured via build.Args
	if _, err := os.Stat(outPath); err != nil {
		return fmt.Errorf("npm build did not produce expected output at %s - ensure build.args configures output path correctly", outPath)
	}

	return nil
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
