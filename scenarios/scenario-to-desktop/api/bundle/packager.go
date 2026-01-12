package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// DefaultPackager is the default implementation of Packager.
type DefaultPackager struct {
	runtimeResolver RuntimeResolver
	runtimeBuilder  RuntimeBuilder
	serviceCompiler ServiceCompiler
	cliStager       CLIStager
	sizeCalculator  SizeCalculator
	platform        PlatformResolver
	fileOps         FileOperations
}

// PackagerOption configures a DefaultPackager.
type PackagerOption func(*DefaultPackager)

// WithRuntimeResolver sets a custom runtime resolver.
func WithRuntimeResolver(resolver RuntimeResolver) PackagerOption {
	return func(p *DefaultPackager) {
		p.runtimeResolver = resolver
	}
}

// WithRuntimeBuilder sets a custom runtime builder.
func WithRuntimeBuilder(builder RuntimeBuilder) PackagerOption {
	return func(p *DefaultPackager) {
		p.runtimeBuilder = builder
	}
}

// WithServiceCompiler sets a custom service compiler.
func WithServiceCompiler(compiler ServiceCompiler) PackagerOption {
	return func(p *DefaultPackager) {
		p.serviceCompiler = compiler
	}
}

// WithCLIStager sets a custom CLI stager.
func WithCLIStager(stager CLIStager) PackagerOption {
	return func(p *DefaultPackager) {
		p.cliStager = stager
	}
}

// WithSizeCalculator sets a custom size calculator.
func WithSizeCalculator(calc SizeCalculator) PackagerOption {
	return func(p *DefaultPackager) {
		p.sizeCalculator = calc
	}
}

// WithPlatformResolver sets a custom platform resolver.
func WithPlatformResolver(resolver PlatformResolver) PackagerOption {
	return func(p *DefaultPackager) {
		p.platform = resolver
	}
}

// WithFileOperations sets custom file operations.
func WithFileOperations(ops FileOperations) PackagerOption {
	return func(p *DefaultPackager) {
		p.fileOps = ops
	}
}

// NewPackager creates a new bundle packager with default implementations.
func NewPackager(opts ...PackagerOption) *DefaultPackager {
	p := &DefaultPackager{
		runtimeResolver: &defaultRuntimeResolver{},
		runtimeBuilder:  &defaultRuntimeBuilder{},
		sizeCalculator:  &defaultSizeCalculator{},
		platform:        &defaultPlatformResolver{},
		fileOps:         &defaultFileOperations{},
	}

	// Service compiler needs access to the platform resolver
	p.serviceCompiler = &defaultServiceCompiler{platform: p.platform, fileOps: p.fileOps}
	p.cliStager = &defaultCLIStager{fileOps: p.fileOps}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Package packages a bundle from the given app path and manifest.
func (p *DefaultPackager) Package(appPath, manifestPath string, requestedPlatforms []string) (*PackageResult, error) {
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

	if err := p.validateManifestForPlatforms(m, platforms); err != nil {
		return nil, err
	}

	bundleDir := filepath.Join(appAbs, "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, fmt.Errorf("create bundle dir: %w", err)
	}

	destManifest := filepath.Join(bundleDir, "bundle.json")
	if err := p.fileOps.CopyFile(manifestAbs, destManifest); err != nil {
		return nil, fmt.Errorf("copy manifest: %w", err)
	}

	manifestRoot := filepath.Dir(manifestAbs)
	var copied []string
	copied = append(copied, destManifest)

	for _, svc := range m.Services {
		for _, platform := range platforms {
			bin, ok := p.platform.ResolveBinaryForPlatform(svc, platform)
			var src string

			if ok {
				// Try to resolve existing binary
				resolved, err := resolveManifestPath(p.fileOps, manifestRoot, bin.Path)
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
				compiledPath, err := p.serviceCompiler.Compile(svc, platform, manifestRoot)
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
			dstPath := p.fileOps.NormalizeBundlePath(bin.Path)
			dst, err := resolveBundlePath(p.fileOps, bundleDir, dstPath)
			if err != nil {
				return nil, fmt.Errorf("stage binary for %s: %w", svc.ID, err)
			}
			if err := p.fileOps.CopyPath(src, dst); err != nil {
				return nil, fmt.Errorf("copy binary for %s: %w", svc.ID, err)
			}
			copied = append(copied, dst)
		}

		for _, asset := range svc.Assets {
			src, err := resolveManifestPath(p.fileOps, manifestRoot, asset.Path)
			if err != nil {
				return nil, fmt.Errorf("resolve asset %s: %w", asset.Path, err)
			}
			assetDstPath := p.fileOps.NormalizeBundlePath(asset.Path)
			dst, err := resolveBundlePath(p.fileOps, bundleDir, assetDstPath)
			if err != nil {
				return nil, fmt.Errorf("stage asset %s: %w", asset.Path, err)
			}
			if err := p.fileOps.CopyPath(src, dst); err != nil {
				return nil, fmt.Errorf("copy asset %s: %w", asset.Path, err)
			}
			copied = append(copied, dst)
		}
	}

	// Stage CLI helpers for each platform requested
	for _, platform := range platforms {
		runtimePlatform := p.platform.NormalizeRuntime(platform)
		if err := p.cliStager.Stage(bundleDir, runtimePlatform); err != nil {
			return nil, fmt.Errorf("stage CLI helpers: %w", err)
		}
	}

	runtimeDir, err := p.runtimeResolver.Resolve()
	if err != nil {
		return nil, err
	}

	runtimeBinaries := map[string]string{}
	for _, platform := range platforms {
		runtimePlatform := p.platform.NormalizeRuntime(platform)
		goos, goarch, err := p.platform.ParseKey(runtimePlatform)
		if err != nil {
			return nil, err
		}
		outDir := filepath.Join(bundleDir, "runtime", runtimePlatform)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return nil, fmt.Errorf("create runtime dir: %w", err)
		}

		runtimePath := filepath.Join(outDir, p.platform.RuntimeBinaryName(goos))
		if err := p.runtimeBuilder.Build(runtimeDir, runtimePath, goos, goarch, "runtime"); err != nil {
			return nil, fmt.Errorf("build runtime (%s): %w", platform, err)
		}
		runtimeBinaries[platform] = runtimePath

		runtimectlPath := filepath.Join(outDir, p.platform.RuntimeCtlBinaryName(goos))
		if err := p.runtimeBuilder.Build(runtimeDir, runtimectlPath, goos, goarch, "runtimectl"); err == nil {
			copied = append(copied, runtimectlPath)
		}
		copied = append(copied, runtimePath)
	}

	if err := ensureBundleExtraResources(appAbs); err != nil {
		return nil, fmt.Errorf("update package.json: %w", err)
	}

	sort.Strings(copied)

	totalSize, largeFiles := p.sizeCalculator.Calculate(bundleDir)
	sizeWarning := p.sizeCalculator.CheckWarning(totalSize, largeFiles)

	return &PackageResult{
		BundleDir:       bundleDir,
		ManifestPath:    destManifest,
		RuntimeBinaries: runtimeBinaries,
		CopiedArtifacts: copied,
		TotalSizeBytes:  totalSize,
		TotalSizeHuman:  HumanReadableSize(totalSize),
		SizeWarning:     sizeWarning,
	}, nil
}

// validateManifestForPlatforms validates a manifest has required binaries or build config for all platforms.
func (p *DefaultPackager) validateManifestForPlatforms(m *bundlemanifest.Manifest, platforms []string) error {
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
			_, hasBinary := p.platform.ResolveBinaryForPlatform(svc, platform)
			hasBuild := svc.Build != nil && svc.Build.Type != ""
			if !hasBinary && !hasBuild {
				return fmt.Errorf("service %s missing binary for %s and no build config", svc.ID, platform)
			}
		}
	}
	return nil
}

// Helper functions

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

func resolveManifestPath(fileOps FileOperations, root, rel string) (string, error) {
	clean := bundlemanifest.ResolvePath(root, rel)
	if !fileOps.WithinBase(root, clean) {
		return "", fmt.Errorf("path escapes manifest root: %s", rel)
	}
	return clean, nil
}

func resolveBundlePath(fileOps FileOperations, root, rel string) (string, error) {
	clean := bundlemanifest.ResolvePath(root, rel)
	if !fileOps.WithinBase(root, clean) {
		return "", fmt.Errorf("path escapes bundle root: %s", rel)
	}
	if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
		return "", err
	}
	return clean, nil
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

// HumanReadableSize converts bytes to human-readable format.
func HumanReadableSize(bytes int64) string {
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
