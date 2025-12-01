package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/cli-core/buildinfo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	moduleRoot := flag.String("module", ".", "path to the Go module directory")
	output := flag.String("output", "", "explicit output path for the built binary")
	installDir := flag.String("install-dir", "", "directory where the binary should be installed")
	name := flag.String("name", "", "binary name (defaults to module directory name)")
	force := flag.Bool("force", true, "overwrite existing binary when present")
	flag.Parse()

	if *output == "" && *installDir == "" {
		*installDir = defaultInstallDir()
	}

	modulePath, err := filepath.Abs(*moduleRoot)
	if err != nil {
		return fmt.Errorf("resolve module root: %w", err)
	}

	if _, err := os.Stat(filepath.Join(modulePath, "go.mod")); err != nil {
		return fmt.Errorf("module root must contain go.mod: %w", err)
	}

	dst, err := determineDestination(modulePath, *output, *installDir, *name)
	if err != nil {
		return err
	}

	if !*force {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("target already exists: %s (use --force to overwrite)", dst)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("prepare install directory: %w", err)
	}

	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go toolchain is required: %w", err)
	}

	fingerprint, err := buildinfo.ComputeFingerprint(modulePath)
	if err != nil {
		return fmt.Errorf("compute fingerprint: %w", err)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	sourceRoot := filepath.ToSlash(modulePath)
	flags := fmt.Sprintf(
		"-X main.buildFingerprint=%s -X main.buildTimestamp=%s -X main.buildSourceRoot=%s",
		fingerprint,
		timestamp,
		escapeLdflagValue(sourceRoot),
	)

	tmpFile, err := os.CreateTemp(filepath.Dir(dst), "cli-build-*")
	if err != nil {
		return fmt.Errorf("create temporary binary: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("go", "build", "-ldflags", flags, "-o", tmpPath, ".")
	cmd.Dir = modulePath
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	if err := replaceBinary(tmpPath, dst); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}

	fmt.Printf("✅ installed CLI to %s\n", dst)
	ensurePathHint(dst)
	return nil
}

func determineDestination(modulePath, explicitOutput, installDir, name string) (string, error) {
	if explicitOutput != "" {
		return filepath.Abs(explicitOutput)
	}

	if installDir == "" {
		return "", errors.New("install directory is required when --output is not set")
	}

	dir := installDir
	if !filepath.IsAbs(dir) {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("resolve install directory: %w", err)
		}
		dir = absDir
	}

	binaryName := name
	if binaryName == "" {
		binaryName = filepath.Base(modulePath)
	}
	if binaryName == "" {
		return "", errors.New("failed to infer binary name")
	}

	return filepath.Join(dir, binaryName), nil
}

func defaultInstallDir() string {
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("USERPROFILE"); dir != "" {
			return filepath.Join(dir, "bin")
		}
		if dir, err := os.UserHomeDir(); err == nil && dir != "" {
			return filepath.Join(dir, "bin")
		}
		return "."
	}
	if dir := os.Getenv("HOME"); dir != "" {
		return filepath.Join(dir, ".local", "bin")
	}
	if dir, err := os.UserHomeDir(); err == nil && dir != "" {
		return filepath.Join(dir, ".local", "bin")
	}
	return "."
}

func replaceBinary(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	renameErr := os.Rename(src, dst)
	if runtime.GOOS == "windows" {
		_ = os.Remove(dst)
		renameErr = os.Rename(src, dst)
	}
	if renameErr == nil {
		return nil
	}
	return fmt.Errorf("replace binary: %w", renameErr)
}

func ensurePathHint(binaryPath string) {
	installDir := filepath.Dir(binaryPath)
	pathEnv := os.Getenv("PATH")
	if strings.Contains(pathEnv, installDir) {
		return
	}

	if runtime.GOOS == "windows" {
		fmt.Printf("⚠️  Add to PATH (PowerShell): $Env:Path = \"%s;$Env:Path\"\n", installDir)
		return
	}

	fmt.Printf("⚠️  Add to PATH: export PATH=\"%s:$PATH\"\n", installDir)
}

func escapeLdflagValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, " ", `\ `)
	return value
}
