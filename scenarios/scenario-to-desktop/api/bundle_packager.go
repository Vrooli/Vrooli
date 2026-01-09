package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

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
type cliStager func(bundleRoot, runtimePlatform string) error

type bundlePackager struct {
	runtimeResolver       runtimeResolver
	runtimeBuilder        runtimeBuilder
	serviceBinaryCompiler serviceBinaryCompiler
	cliStager             cliStager
}

func newBundlePackager() *bundlePackager {
	bp := &bundlePackager{
		runtimeResolver: findRuntimeSourceDir,
		runtimeBuilder:  buildRuntimeBinary,
	}
	bp.serviceBinaryCompiler = bp.compileServiceBinary
	bp.cliStager = stageCLIs
	return bp
}

func (p *bundlePackager) packageBundle(appPath, manifestPath string, requestedPlatforms []string) (*bundlePackageResult, error) {
	if appPath == "" || manifestPath == "" {
		return nil, errors.New("app_path and bundle_manifest_path are required")
	}

	cliStage := p.cliStager
	if cliStage == nil {
		cliStage = func(string, string) error { return nil }
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

	// Stage CLI helpers (bin directory and vrooli shim) for each platform requested.
	for _, platform := range platforms {
		runtimePlatform := normalizeRuntimePlatform(platform)
		if err := cliStage(bundleDir, runtimePlatform); err != nil {
			return nil, fmt.Errorf("stage CLI helpers: %w", err)
		}
	}

	runtimeDir, err := p.runtimeResolver()
	if err != nil {
		return nil, err
	}

	runtimeBinaries := map[string]string{}
	for _, platform := range platforms {
		runtimePlatform := normalizeRuntimePlatform(platform)
		goos, goarch, err := parsePlatformKey(runtimePlatform)
		if err != nil {
			return nil, err
		}
		outDir := filepath.Join(bundleDir, "runtime", runtimePlatform)
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

// validateManifestForPlatforms validates a manifest has required binaries or build config for all platforms.
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

// resolveManifestPath resolves a relative path against the manifest root.
func resolveManifestPath(root, rel string) (string, error) {
	clean := bundlemanifest.ResolvePath(root, rel)
	if !withinBase(root, clean) {
		return "", fmt.Errorf("path escapes manifest root: %s", rel)
	}
	return clean, nil
}

// resolveBundlePath resolves a relative path against the bundle root.
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

// findRuntimeSourceDir locates the runtime source directory.
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

// ensureBundleExtraResources updates package.json to include bundle in extraResources.
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

// bundleExtraExists checks if bundle entry exists in extraResources.
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

// collectPlatforms extracts unique platforms from manifest services.
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
