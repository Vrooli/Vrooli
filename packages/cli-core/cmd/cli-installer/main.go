package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	moduleRoot := flag.String("module", ".", "path to the Go module directory")
	manifestSource := flag.String("manifest", "", "path to the source manifest that should be installed beside the binary")
	output := flag.String("output", "", "explicit output path for the built binary")
	installDir := flag.String("install-dir", "", "directory where the binary should be installed")
	name := flag.String("name", "", "binary name (defaults to module directory name)")
	contextRoot := flag.String("context-root", "", "path to the freshness context root (defaults to module root)")
	force := flag.Bool("force", true, "overwrite existing binary when present")
	var freshnessInputs stringListFlag
	flag.Var(&freshnessInputs, "freshness-input", "declared freshness input relative to the context root (repeatable)")
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
	manifestPath, err := resolveManifestPath(*manifestSource)
	if err != nil {
		return err
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

	binaryName := *name
	if binaryName == "" {
		binaryName = filepath.Base(modulePath)
	}
	spec, err := buildFreshnessSpec(modulePath, *contextRoot, []string(freshnessInputs), binaryName)
	if err != nil {
		return fmt.Errorf("build freshness spec: %w", err)
	}
	fingerprint, err := cliutil.ComputeFreshnessFingerprint(spec)
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
	if manifestPath != "" {
		if err := installManifest(manifestPath, cliutil.InstalledManifestPath(dst)); err != nil {
			return fmt.Errorf("install manifest: %w", err)
		}
	}
	if err := writeInstallMetadata(dst, installMetadata{
		BinaryName:  binaryName,
		ModulePath:  sourceRoot,
		Fingerprint: fingerprint,
		InstalledAt: timestamp,
	}); err != nil {
		return fmt.Errorf("write install metadata: %w", err)
	}

	fmt.Printf("✅ installed CLI to %s\n", dst)
	ensurePathHint(dst)
	ensureShellRefreshHint()
	return nil
}

type installMetadata struct {
	BinaryName  string `json:"binary_name"`
	ModulePath  string `json:"module_path"`
	Fingerprint string `json:"fingerprint"`
	InstalledAt string `json:"installed_at,omitempty"`
}

func buildFreshnessSpec(modulePath, contextRoot string, inputs []string, binaryName string) (cliutil.FreshnessSpec, error) {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return cliutil.FreshnessSpec{}, errors.New("module path must not be empty")
	}
	absModulePath, err := filepath.Abs(filepath.Clean(modulePath))
	if err != nil {
		return cliutil.FreshnessSpec{}, fmt.Errorf("resolve module path: %w", err)
	}
	modulePath = absModulePath

	if strings.TrimSpace(contextRoot) == "" {
		contextRoot = modulePath
	} else {
		absContextRoot, err := filepath.Abs(filepath.Clean(strings.TrimSpace(contextRoot)))
		if err != nil {
			return cliutil.FreshnessSpec{}, fmt.Errorf("resolve context root: %w", err)
		}
		contextRoot = absContextRoot
	}

	return cliutil.FreshnessSpec{
		SourceRoot:  modulePath,
		ContextRoot: contextRoot,
		Inputs:      append([]string(nil), inputs...),
		SkipFiles:   []string{binaryName},
	}, nil
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

func resolveManifestPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve manifest path: %w", err)
	}
	if info, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("manifest source must exist: %w", err)
	} else if info.IsDir() {
		return "", fmt.Errorf("manifest source must be a file: %s", absPath)
	}
	return absPath, nil
}

func installManifest(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("prepare manifest directory: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(dst), "cli-manifest-*")
	if err != nil {
		return fmt.Errorf("create temporary manifest: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := copyFileContents(src, tmpFile); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary manifest: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0o644); err != nil {
			return fmt.Errorf("chmod temporary manifest: %w", err)
		}
	}
	if err := replaceBinary(tmpPath, dst); err != nil {
		return fmt.Errorf("replace manifest: %w", err)
	}
	return nil
}

func copyFileContents(src string, dst *os.File) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open manifest source: %w", err)
	}
	defer in.Close()
	if _, err := io.Copy(dst, in); err != nil {
		return fmt.Errorf("copy manifest source: %w", err)
	}
	return nil
}

func writeInstallMetadata(binaryPath string, meta installMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(cliutil.InstalledBuildMetadataPath(binaryPath), data, 0o644)
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

func ensureShellRefreshHint() {
	if runtime.GOOS == "windows" {
		fmt.Println("ℹ️  If you are replacing an existing CLI command, open a new shell so command lookup refreshes.")
		return
	}
	fmt.Println("ℹ️  If you are replacing an existing CLI command in this shell, run: hash -r")
}

func escapeLdflagValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, " ", `\ `)
	return value
}
