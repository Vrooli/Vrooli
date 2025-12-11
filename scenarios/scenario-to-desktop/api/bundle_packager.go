package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// Bundle size thresholds for warnings
const (
	BundleSizeWarningThreshold  = 500 * 1024 * 1024  // 500MB - warn
	BundleSizeCriticalThreshold = 1024 * 1024 * 1024 // 1GB - critical warning
)

// BundleSizeWarning represents a warning about bundle size.
type BundleSizeWarning struct {
	Level       string `json:"level"`   // "warning" or "critical"
	Message     string `json:"message"`
	TotalBytes  int64  `json:"total_bytes"`
	TotalHuman  string `json:"total_human"`
	LargeFiles  []LargeFileInfo `json:"large_files,omitempty"`
}

// LargeFileInfo describes a file contributing significantly to bundle size.
type LargeFileInfo struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
}

type bundlePackageResult struct {
	BundleDir       string
	ManifestPath    string
	RuntimeBinaries map[string]string
	CopiedArtifacts []string
	TotalSizeBytes  int64
	TotalSizeHuman  string
	SizeWarning     *BundleSizeWarning
}

func packageBundle(appPath, manifestPath string, requestedPlatforms []string) (*bundlePackageResult, error) {
	return newBundlePackager().packageBundle(appPath, manifestPath, requestedPlatforms)
}

type runtimeResolver func() (string, error)
type runtimeBuilder func(srcDir, outPath, goos, goarch, target string) error
type serviceBinaryCompiler func(svc bundlemanifest.Service, platform, manifestRoot string) (string, error)

type bundlePackager struct {
	runtimeResolver        runtimeResolver
	runtimeBuilder         runtimeBuilder
	serviceBinaryCompiler  serviceBinaryCompiler
}

func newBundlePackager() *bundlePackager {
	bp := &bundlePackager{
		runtimeResolver: findRuntimeSourceDir,
		runtimeBuilder:  buildRuntimeBinary,
	}
	bp.serviceBinaryCompiler = bp.compileServiceBinary
	return bp
}

func (p *bundlePackager) packageBundle(appPath, manifestPath string, requestedPlatforms []string) (*bundlePackageResult, error) {
	if appPath == "" || manifestPath == "" {
		return nil, errors.New("app_path and bundle_manifest_path are required")
	}

	appAbs, err := filepath.Abs(appPath)
	if err != nil {
		return nil, fmt.Errorf("resolve app path: %w", err)
	}
	if info, err := os.Stat(appAbs); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("app_path must be an existing directory: %w", err)
	}

	manifestAbs, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest path: %w", err)
	}
	if _, err := os.Stat(manifestAbs); err != nil {
		return nil, fmt.Errorf("manifest path invalid: %w", err)
	}

	m, err := bundlemanifest.LoadManifest(manifestAbs)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	platforms := requestedPlatforms
	if len(platforms) == 0 {
		platforms = collectPlatforms(*m)
	}
	if len(platforms) == 0 {
		return nil, errors.New("manifest has no platform binaries to package")
	}

	if err := validateManifestForPlatforms(m, platforms); err != nil {
		return nil, err
	}

	bundleDir := filepath.Join(appAbs, "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, fmt.Errorf("create bundle dir: %w", err)
	}

	destManifest := filepath.Join(bundleDir, "bundle.json")
	if err := copyFile(manifestAbs, destManifest); err != nil {
		return nil, fmt.Errorf("copy manifest: %w", err)
	}

	manifestRoot := filepath.Dir(manifestAbs)
	var copied []string
	copied = append(copied, destManifest)

	for _, svc := range m.Services {
		for _, platform := range platforms {
			bin, ok := resolveBinaryForPlatform(svc, platform)
			var src string

			if ok {
				// Try to resolve existing binary
				resolved, err := resolveManifestPath(manifestRoot, bin.Path)
				if err != nil {
					return nil, fmt.Errorf("resolve binary for %s: %w", svc.ID, err)
				}
				// Check if binary exists
				if _, statErr := os.Stat(resolved); statErr == nil {
					src = resolved
				}
			}

			// If binary doesn't exist or wasn't in manifest, try to compile
			if src == "" {
				if svc.Build == nil {
					if !ok {
						return nil, fmt.Errorf("service %s missing binary for %s and no build config", svc.ID, platform)
					}
					return nil, fmt.Errorf("service %s binary not found at %s and no build config", svc.ID, bin.Path)
				}

				// Compile the service binary
				compiledPath, err := p.serviceBinaryCompiler(svc, platform, manifestRoot)
				if err != nil {
					return nil, fmt.Errorf("compile binary for %s (%s): %w", svc.ID, platform, err)
				}
				src = compiledPath

				// Update manifest binary path for this platform if not set
				if !ok {
					if svc.Binaries == nil {
						svc.Binaries = make(map[string]bundlemanifest.Binary)
					}
					relPath, _ := filepath.Rel(manifestRoot, compiledPath)
					svc.Binaries[platform] = bundlemanifest.Binary{Path: relPath}
					bin = svc.Binaries[platform]
				}
			}

			// Normalize the destination path by stripping any parent directory traversal
			// This ensures binaries from outside the manifest root are still staged inside the bundle
			dstPath := normalizeBundlePath(bin.Path)
			dst, err := resolveBundlePath(bundleDir, dstPath)
			if err != nil {
				return nil, fmt.Errorf("stage binary for %s: %w", svc.ID, err)
			}
			if err := copyPath(src, dst); err != nil {
				return nil, fmt.Errorf("copy binary for %s: %w", svc.ID, err)
			}
			copied = append(copied, dst)
		}

		for _, asset := range svc.Assets {
			src, err := resolveManifestPath(manifestRoot, asset.Path)
			if err != nil {
				return nil, fmt.Errorf("resolve asset %s: %w", asset.Path, err)
			}
			// Normalize asset destination path like we do for binaries
			assetDstPath := normalizeBundlePath(asset.Path)
			dst, err := resolveBundlePath(bundleDir, assetDstPath)
			if err != nil {
				return nil, fmt.Errorf("stage asset %s: %w", asset.Path, err)
			}
			if err := copyPath(src, dst); err != nil {
				return nil, fmt.Errorf("copy asset %s: %w", asset.Path, err)
			}
			copied = append(copied, dst)
		}
	}

	runtimeDir, err := p.runtimeResolver()
	if err != nil {
		return nil, err
	}

	runtimeBinaries := map[string]string{}
	for _, platform := range platforms {
		goos, goarch, err := parsePlatformKey(platform)
		if err != nil {
			return nil, err
		}
		outDir := filepath.Join(bundleDir, "runtime", platform)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return nil, fmt.Errorf("create runtime dir: %w", err)
		}

		runtimePath := filepath.Join(outDir, runtimeBinaryName(goos))
		if err := p.runtimeBuilder(runtimeDir, runtimePath, goos, goarch, "runtime"); err != nil {
			return nil, fmt.Errorf("build runtime (%s): %w", platform, err)
		}
		runtimeBinaries[platform] = runtimePath

		runtimectlPath := filepath.Join(outDir, runtimeCtlBinaryName(goos))
		if err := p.runtimeBuilder(runtimeDir, runtimectlPath, goos, goarch, "runtimectl"); err == nil {
			copied = append(copied, runtimectlPath)
		}
		copied = append(copied, runtimePath)
	}

	if err := ensureBundleExtraResources(appAbs); err != nil {
		return nil, fmt.Errorf("update package.json: %w", err)
	}

	sort.Strings(copied)

	// Calculate bundle size and check for warnings
	totalSize, largeFiles := calculateBundleSize(bundleDir)
	sizeWarning := checkBundleSizeWarning(totalSize, largeFiles)

	return &bundlePackageResult{
		BundleDir:       bundleDir,
		ManifestPath:    destManifest,
		RuntimeBinaries: runtimeBinaries,
		CopiedArtifacts: copied,
		TotalSizeBytes:  totalSize,
		TotalSizeHuman:  humanReadableSize(totalSize),
		SizeWarning:     sizeWarning,
	}, nil
}

func validateManifestForPlatforms(m *bundlemanifest.Manifest, platforms []string) error {
	if m.SchemaVersion == "" {
		return errors.New("schema_version is required")
	}
	if m.Target != "desktop" {
		return fmt.Errorf("unsupported target %q (expected desktop)", m.Target)
	}
	if len(m.Services) == 0 {
		return errors.New("manifest requires at least one service")
	}
	for _, svc := range m.Services {
		for _, platform := range platforms {
			_, hasBinary := resolveBinaryForPlatform(svc, platform)
			hasBuild := svc.Build != nil && svc.Build.Type != ""
			if !hasBinary && !hasBuild {
				return fmt.Errorf("service %s missing binary for %s and no build config", svc.ID, platform)
			}
		}
	}
	return nil
}

func resolveBinaryForPlatform(svc bundlemanifest.Service, platform string) (bundlemanifest.Binary, bool) {
	keys := []string{platform}
	if alias := aliasPlatformKey(platform); alias != "" {
		keys = append(keys, alias)
	}
	// Try exact and aliased matches first
	for _, key := range keys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	// For shorthand platforms (win, mac, linux), try architecture-specific keys
	archKeys := expandShorthandPlatform(platform)
	for _, key := range archKeys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	return bundlemanifest.Binary{}, false
}

// expandShorthandPlatform expands shorthand platform names to architecture-specific keys.
// For example, "win" -> ["win-x64", "win-arm64", "windows-x64", "windows-arm64"]
func expandShorthandPlatform(platform string) []string {
	archs := []string{"x64", "arm64", "amd64", "aarch64"}
	var keys []string

	switch platform {
	case "win", "windows":
		for _, arch := range archs {
			keys = append(keys, "win-"+arch, "windows-"+arch)
		}
	case "mac", "darwin":
		for _, arch := range archs {
			keys = append(keys, "darwin-"+arch, "mac-"+arch)
		}
	case "linux":
		for _, arch := range archs {
			keys = append(keys, "linux-"+arch)
		}
	}
	return keys
}

func aliasPlatformKey(key string) string {
	if strings.HasPrefix(key, "windows-") {
		return "win-" + strings.TrimPrefix(key, "windows-")
	}
	if strings.HasPrefix(key, "win-") {
		return "windows-" + strings.TrimPrefix(key, "win-")
	}
	if strings.HasPrefix(key, "darwin-") {
		return "mac-" + strings.TrimPrefix(key, "darwin-")
	}
	if strings.HasPrefix(key, "mac-") {
		return "darwin-" + strings.TrimPrefix(key, "mac-")
	}
	return ""
}

func parsePlatformKey(key string) (string, string, error) {
	parts := strings.Split(key, "-")

	// Handle shorthand platforms (win, mac, linux) without architecture
	if len(parts) == 1 {
		goos, goarch := expandShorthandToHostArch(key)
		if goos != "" {
			return goos, goarch, nil
		}
		return "", "", fmt.Errorf("invalid platform key %q", key)
	}

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid platform key %q", key)
	}
	goos := parts[0]
	switch goos {
	case "win":
		goos = "windows"
	case "mac":
		goos = "darwin"
	case "darwin", "linux", "windows":
	default:
		return "", "", fmt.Errorf("unsupported platform os %q", goos)
	}

	goarch := parts[1]
	switch goarch {
	case "x64", "amd64":
		goarch = "amd64"
	case "arm64", "aarch64":
		goarch = "arm64"
	default:
		return "", "", fmt.Errorf("unsupported arch %q", goarch)
	}

	return goos, goarch, nil
}

// expandShorthandToHostArch expands shorthand platform names to goos/goarch using host architecture.
func expandShorthandToHostArch(platform string) (string, string) {
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "amd64"
	case "arm64":
		goarch = "arm64"
	default:
		goarch = "amd64" // default fallback
	}

	switch platform {
	case "win", "windows":
		return "windows", goarch
	case "mac", "darwin":
		return "darwin", goarch
	case "linux":
		return "linux", goarch
	}
	return "", ""
}

func resolveManifestPath(root, rel string) (string, error) {
	// Allow paths outside the manifest root for source files (binaries, assets)
	// since they may be built elsewhere. The security boundary is enforced
	// on destination paths via resolveBundlePath.
	clean := bundlemanifest.ResolvePath(root, rel)
	return clean, nil
}

func resolveBundlePath(root, rel string) (string, error) {
	clean := bundlemanifest.ResolvePath(root, rel)
	if !withinBase(root, clean) {
		return "", fmt.Errorf("path escapes bundle root: %s", rel)
	}
	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return "", err
	}
	return clean, nil
}

func withinBase(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// normalizeBundlePath strips leading parent directory traversals from a path.
// For example: "../../../bin/linux-x64/api" becomes "bin/linux-x64/api"
// This ensures files are always staged inside the bundle directory.
func normalizeBundlePath(rel string) string {
	// Convert to forward slashes for consistent handling
	clean := filepath.ToSlash(rel)
	// Remove leading "../" segments
	for strings.HasPrefix(clean, "../") {
		clean = strings.TrimPrefix(clean, "../")
	}
	// Also handle edge case of just ".."
	if clean == ".." {
		clean = ""
	}
	return clean
}

func copyFile(src, dst string) error {
	// Resolve absolute paths to detect if src and dst are the same file
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if absSrc == absDst {
		// Source and destination are the same file, nothing to copy
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst, info.Mode())
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(dst, mode.Perm()); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target)
	})
}

func runtimeBinaryName(goos string) string {
	if goos == "windows" {
		return "runtime.exe"
	}
	return "runtime"
}

func runtimeCtlBinaryName(goos string) string {
	if goos == "windows" {
		return "runtimectl.exe"
	}
	return "runtimectl"
}

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

func findRuntimeSourceDir() (string, error) {
	candidates := []string{}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "..", "runtime"))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "..", "runtime"),
			filepath.Join(exeDir, "..", "..", "runtime"),
		)
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Abs(candidate)
		}
	}

	return "", errors.New("runtime source directory not found")
}

func ensureBundleExtraResources(appDir string) error {
	pkgPath := filepath.Join(appDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}

	build, _ := pkg["build"].(map[string]interface{})
	if build == nil {
		build = map[string]interface{}{}
	}

	extra, _ := build["extraResources"].([]interface{})
	if !bundleExtraExists(extra) {
		entry := map[string]interface{}{
			"from":   "bundle",
			"to":     "bundle",
			"filter": []interface{}{"**/*"},
		}
		extra = append(extra, entry)
	}
	build["extraResources"] = extra
	pkg["build"] = build

	updated, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}
	updated = append(updated, '\n')
	return os.WriteFile(pkgPath, updated, 0o644)
}

func bundleExtraExists(entries []interface{}) bool {
	for _, entry := range entries {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		from, _ := m["from"].(string)
		to, _ := m["to"].(string)
		if from == "bundle" || to == "bundle" {
			return true
		}
	}
	return false
}

func collectPlatforms(m bundlemanifest.Manifest) []string {
	seen := map[string]bool{}
	for _, svc := range m.Services {
		for key := range svc.Binaries {
			seen[key] = true
		}
	}
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// compileServiceBinary compiles a service binary for the specified platform.
// It supports Go, Rust, npm (Node.js), and custom build commands.
func (p *bundlePackager) compileServiceBinary(svc bundlemanifest.Service, platform, manifestRoot string) (string, error) {
	if svc.Build == nil {
		return "", errors.New("no build configuration")
	}

	build := svc.Build
	goos, goarch, err := parsePlatformKey(platform)
	if err != nil {
		return "", err
	}

	// Resolve source directory
	srcDir := filepath.Join(manifestRoot, build.SourceDir)
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
		return absOutput, compileRustBinary(srcDir, absOutput, goos, goarch, build)
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

// calculateBundleSize walks the bundle directory and returns total size and large files.
// Large files are those over 10MB.
func calculateBundleSize(bundleDir string) (int64, []LargeFileInfo) {
	const largeFileThreshold = 10 * 1024 * 1024 // 10MB

	var totalSize int64
	var largeFiles []LargeFileInfo

	_ = filepath.WalkDir(bundleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		size := info.Size()
		totalSize += size

		if size >= largeFileThreshold {
			relPath, _ := filepath.Rel(bundleDir, path)
			largeFiles = append(largeFiles, LargeFileInfo{
				Path:      relPath,
				SizeBytes: size,
				SizeHuman: humanReadableSize(size),
			})
		}

		return nil
	})

	// Sort large files by size descending
	sort.Slice(largeFiles, func(i, j int) bool {
		return largeFiles[i].SizeBytes > largeFiles[j].SizeBytes
	})

	// Keep only top 10 largest files
	if len(largeFiles) > 10 {
		largeFiles = largeFiles[:10]
	}

	return totalSize, largeFiles
}

// checkBundleSizeWarning returns a warning if the bundle exceeds size thresholds.
func checkBundleSizeWarning(totalSize int64, largeFiles []LargeFileInfo) *BundleSizeWarning {
	if totalSize < BundleSizeWarningThreshold {
		return nil
	}

	var level, message string
	if totalSize >= BundleSizeCriticalThreshold {
		level = "critical"
		message = fmt.Sprintf("Bundle size (%s) exceeds 1GB. This will result in very large installers and slow downloads. Consider removing large assets or using delta updates.",
			humanReadableSize(totalSize))
	} else {
		level = "warning"
		message = fmt.Sprintf("Bundle size (%s) exceeds 500MB. Consider optimizing assets to reduce download size.",
			humanReadableSize(totalSize))
	}

	return &BundleSizeWarning{
		Level:      level,
		Message:    message,
		TotalBytes: totalSize,
		TotalHuman: humanReadableSize(totalSize),
		LargeFiles: largeFiles,
	}
}

// humanReadableSize converts bytes to human-readable format.
func humanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
