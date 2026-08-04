package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
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
	p.cliStager = &defaultCLIStager{fileOps: p.fileOps, runtimeResolver: p.runtimeResolver}

	for _, opt := range opts {
		opt(p)
	}
	if stager, ok := p.cliStager.(*defaultCLIStager); ok {
		stager.runtimeResolver = p.runtimeResolver
	}

	return p
}

// packagePaths holds resolved absolute paths for packaging.
type packagePaths struct {
	appAbs        string
	outputRootAbs string
	manifestAbs   string
}

// resolvePackagePaths validates and resolves all input paths for packaging.
func resolvePackagePaths(appPath, manifestPath string, outputRoot []string) (*packagePaths, error) {
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

	outputRootAbs := appAbs
	if len(outputRoot) > 0 && outputRoot[0] != "" {
		rootAbs, err := filepath.Abs(outputRoot[0])
		if err != nil {
			return nil, fmt.Errorf("resolve output root: %w", err)
		}
		outputRootAbs = rootAbs
	}

	manifestAbs, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest path: %w", err)
	}
	if _, err := os.Stat(manifestAbs); err != nil {
		return nil, fmt.Errorf("manifest path invalid: %w", err)
	}

	return &packagePaths{
		appAbs:        appAbs,
		outputRootAbs: outputRootAbs,
		manifestAbs:   manifestAbs,
	}, nil
}

// resolvePlatforms determines which platforms to package from the manifest.
func resolvePlatforms(m *bundlemanifest.Manifest, requestedPlatforms []string) ([]string, error) {
	platforms := requestedPlatforms
	if len(platforms) == 0 {
		platforms = collectPlatforms(*m)
	}
	if len(platforms) == 0 {
		return nil, errors.New("manifest has no platform binaries to package")
	}
	return platforms, nil
}

// stageAllServices stages all service binaries and assets into the bundle directory.
func (p *DefaultPackager) stageAllServices(m *bundlemanifest.Manifest, platforms []string, appAbs, bundleDir, manifestRoot string) ([]string, error) {
	var copied []string
	for _, svc := range m.Services {
		if isUIService(svc.Type) {
			uiCopied, err := p.stageUIService(svc, appAbs, bundleDir, manifestRoot)
			if err != nil {
				return nil, err
			}
			copied = append(copied, uiCopied...)
			continue
		}

		binCopied, err := p.stageServiceBinaries(svc, platforms, bundleDir, manifestRoot, appAbs)
		if err != nil {
			return nil, err
		}
		copied = append(copied, binCopied...)

		assetCopied, err := p.stageServiceAssets(svc, bundleDir, manifestRoot)
		if err != nil {
			return nil, err
		}
		copied = append(copied, assetCopied...)
	}
	return copied, nil
}

// Package packages a bundle from the given app path and manifest.
// framework specifies the target framework (e.g., "electron") which determines the bundle output path.
// outputRoot optionally overrides where the bundle is written (defaults to appPath).
func (p *DefaultPackager) Package(appPath, manifestPath, framework string, requestedPlatforms []string, outputRoot ...string) (*PackageResult, error) {
	if framework == "" {
		framework = "electron"
	}

	paths, err := resolvePackagePaths(appPath, manifestPath, outputRoot)
	if err != nil {
		return nil, err
	}

	m, err := bundlemanifest.LoadManifest(paths.manifestAbs)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	platforms, err := resolvePlatforms(m, requestedPlatforms)
	if err != nil {
		return nil, err
	}

	if err := p.validateManifestForPlatforms(m, platforms); err != nil {
		return nil, err
	}

	bundleDir := filepath.Join(paths.outputRootAbs, "platforms", framework, "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, fmt.Errorf("create bundle dir: %w", err)
	}

	destManifest := filepath.Join(bundleDir, "bundle.json")
	if err := p.fileOps.CopyFile(paths.manifestAbs, destManifest); err != nil {
		return nil, fmt.Errorf("copy manifest: %w", err)
	}

	manifestRoot := filepath.Dir(paths.manifestAbs)
	copied := []string{destManifest}

	svcCopied, err := p.stageAllServices(m, platforms, paths.appAbs, bundleDir, manifestRoot)
	if err != nil {
		return nil, err
	}
	copied = append(copied, svcCopied...)

	// Desktop applications do not receive VROOLI_ROOT. Stage a deliberately
	// narrow, read-only manifest catalog so manifest-driven applications (such
	// as Vrooli Onboarding) remain functional without copying a working tree or
	// any operator configuration into the artifact.
	catalogCopied, err := p.stageManifestCatalog(paths.appAbs, bundleDir)
	if err != nil {
		return nil, err
	}
	copied = append(copied, catalogCopied...)

	if err := p.stageCLIHelpers(platforms, bundleDir); err != nil {
		return nil, err
	}

	runtimeBinaries, runtimeCopied, err := p.buildRuntimes(platforms, bundleDir)
	if err != nil {
		return nil, err
	}
	copied = append(copied, runtimeCopied...)

	if err := ensureBundleExtraResources(paths.outputRootAbs, framework); err != nil {
		return nil, fmt.Errorf("update package.json: %w", err)
	}

	sort.Strings(copied)

	totalSize, largeFiles := p.sizeCalculator.Calculate(bundleDir)
	sizeWarning := p.sizeCalculator.CheckWarning(totalSize, largeFiles)

	var manifestContent map[string]interface{}
	if manifestData, readErr := os.ReadFile(destManifest); readErr == nil {
		_ = json.Unmarshal(manifestData, &manifestContent)
	}

	return &PackageResult{
		BundleDir:       bundleDir,
		ManifestPath:    destManifest,
		ManifestContent: manifestContent,
		RuntimeBinaries: runtimeBinaries,
		CopiedArtifacts: copied,
		TotalSizeBytes:  totalSize,
		TotalSizeHuman:  HumanReadableSize(totalSize),
		SizeWarning:     sizeWarning,
	}, nil
}

// stageManifestCatalog copies only declarative service and resource manifests
// from a repository-shaped scenario path. It intentionally excludes every
// config directory, generated file, and secret-bearing operator state.
// Synthetic and standalone scenarios have no repository catalog and simply
// receive no catalog.
func (p *DefaultPackager) stageManifestCatalog(appPath, bundleDir string) ([]string, error) {
	scenariosRoot := filepath.Dir(appPath)
	if filepath.Base(scenariosRoot) != "scenarios" {
		return nil, nil
	}
	repoRoot := filepath.Dir(scenariosRoot)
	resourcesRoot := filepath.Join(repoRoot, "resources")
	if info, err := os.Stat(resourcesRoot); err != nil || !info.IsDir() {
		return nil, nil
	}

	var copied []string
	for _, root := range []struct {
		source string
		file   string
	}{
		{source: scenariosRoot, file: filepath.Join(".vrooli", "service.json")},
		{source: resourcesRoot, file: "resource.json"},
	} {
		entries, err := os.ReadDir(root.source)
		if err != nil {
			return nil, fmt.Errorf("read manifest catalog %s: %w", root.source, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			src := filepath.Join(root.source, entry.Name(), root.file)
			if _, err := os.Stat(src); os.IsNotExist(err) {
				continue
			} else if err != nil {
				return nil, fmt.Errorf("stat manifest catalog entry %s: %w", src, err)
			}
			dst := filepath.Join(bundleDir, "catalog", filepath.Base(root.source), entry.Name(), root.file)
			if err := p.fileOps.CopyFile(src, dst); err != nil {
				return nil, fmt.Errorf("copy manifest catalog entry %s: %w", src, err)
			}
			copied = append(copied, dst)
		}
	}
	return copied, nil
}

// stageServiceBinaries resolves or compiles binaries for a single service across all requested platforms,
// then copies them into the bundle directory. Returns the list of copied destination paths.
func (p *DefaultPackager) stageServiceBinaries(svc bundlemanifest.Service, platforms []string, bundleDir, manifestRoot, appAbs string) ([]string, error) {
	var copied []string
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

			// Compile the service binary (source dirs are relative to scenario root, not manifest)
			compiledPath, err := p.serviceCompiler.Compile(svc, platform, appAbs)
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
	return copied, nil
}

// stageServiceAssets copies all declared assets for a single service into the bundle directory.
func (p *DefaultPackager) stageServiceAssets(svc bundlemanifest.Service, bundleDir, manifestRoot string) ([]string, error) {
	var copied []string
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
	return copied, nil
}

// stageCLIHelpers stages CLI helper binaries for each requested platform.
func (p *DefaultPackager) stageCLIHelpers(platforms []string, bundleDir string) error {
	for _, platform := range platforms {
		runtimePlatform := p.platform.NormalizeRuntime(platform)
		if err := p.cliStager.Stage(bundleDir, runtimePlatform); err != nil {
			return fmt.Errorf("stage CLI helpers: %w", err)
		}
	}
	return nil
}

// buildRuntimes compiles the runtime and runtimectl binaries for each platform.
// Returns the runtime binary paths map and the list of all copied paths.
func (p *DefaultPackager) buildRuntimes(platforms []string, bundleDir string) (map[string]string, []string, error) {
	runtimeDir, err := p.runtimeResolver.Resolve()
	if err != nil {
		return nil, nil, err
	}

	runtimeBinaries := map[string]string{}
	var copied []string
	for _, platform := range platforms {
		runtimePlatform := p.platform.NormalizeRuntime(platform)
		goos, goarch, err := p.platform.ParseKey(runtimePlatform)
		if err != nil {
			return nil, nil, err
		}
		outDir := filepath.Join(bundleDir, "runtime", runtimePlatform)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create runtime dir: %w", err)
		}

		runtimePath := filepath.Join(outDir, p.platform.RuntimeBinaryName(goos))
		if err := p.runtimeBuilder.Build(runtimeDir, runtimePath, goos, goarch, "runtime"); err != nil {
			return nil, nil, fmt.Errorf("build runtime (%s): %w", platform, err)
		}
		runtimeBinaries[platform] = runtimePath

		runtimectlPath := filepath.Join(outDir, p.platform.RuntimeCtlBinaryName(goos))
		if err := p.runtimeBuilder.Build(runtimeDir, runtimectlPath, goos, goarch, "runtimectl"); err == nil {
			copied = append(copied, runtimectlPath)
		}
		copied = append(copied, runtimePath)
	}
	return runtimeBinaries, copied, nil
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
		// Skip UI services - Electron handles UI bundling separately
		if isUIService(svc.Type) {
			continue
		}
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

// stageUIService copies a UI service's entry point and assets to the bundle.
// UI services contain pre-built static assets (HTML/CSS/JS from a build tool like Vite),
// not compiled binaries. We include them in the bundle to:
//  1. Make the bundle fully self-contained and portable
//  2. Allow preflight to validate all assets before the Electron build
//  3. Ensure the desktop app has everything it needs without external dependencies
//
// The entry point (typically index.html) is specified in the service's Binaries map,
// and additional assets are listed in the Assets array.
func (p *DefaultPackager) stageUIService(svc bundlemanifest.Service, appPath, bundleDir, manifestRoot string) ([]string, error) {
	var copied []string

	// UI services have their entry point in the Binaries map (e.g., "ui/dist/index.html").
	// All platforms share the same entry point, so we only need to copy once.
	// We use the first platform's binary path as the canonical entry.
	var entryPath string
	for _, bin := range svc.Binaries {
		if bin.Path != "" {
			entryPath = bin.Path
			break
		}
	}

	if entryPath != "" {
		// The entry path in the manifest is relative to the bundle root.
		// We need to find the source by looking in the app directory.
		srcPath := filepath.Join(appPath, entryPath)
		if _, err := os.Stat(srcPath); err != nil {
			// Try relative to manifest root as fallback
			srcPath = filepath.Join(manifestRoot, entryPath)
			if _, err := os.Stat(srcPath); err != nil {
				return nil, fmt.Errorf("UI entry point not found for %s: tried %s and %s",
					svc.ID, filepath.Join(appPath, entryPath), srcPath)
			}
		}

		dstPath := p.fileOps.NormalizeBundlePath(entryPath)
		dst, err := resolveBundlePath(p.fileOps, bundleDir, dstPath)
		if err != nil {
			return nil, fmt.Errorf("stage UI entry for %s: %w", svc.ID, err)
		}
		if err := p.fileOps.CopyPath(srcPath, dst); err != nil {
			return nil, fmt.Errorf("copy UI entry for %s: %w", svc.ID, err)
		}
		copied = append(copied, dst)
	}

	// Copy all UI assets (JS chunks, CSS, images, fonts, etc.)
	for _, asset := range svc.Assets {
		// Try to find the asset in the app directory first
		srcPath := filepath.Join(appPath, asset.Path)
		if _, err := os.Stat(srcPath); err != nil {
			// Fall back to manifest root
			srcPath = filepath.Join(manifestRoot, asset.Path)
			if _, err := os.Stat(srcPath); err != nil {
				return nil, fmt.Errorf("UI asset not found for %s: %s", svc.ID, asset.Path)
			}
		}

		dstPath := p.fileOps.NormalizeBundlePath(asset.Path)
		dst, err := resolveBundlePath(p.fileOps, bundleDir, dstPath)
		if err != nil {
			return nil, fmt.Errorf("stage UI asset %s: %w", asset.Path, err)
		}
		if err := p.fileOps.CopyPath(srcPath, dst); err != nil {
			return nil, fmt.Errorf("copy UI asset %s: %w", asset.Path, err)
		}
		copied = append(copied, dst)
	}

	return copied, nil
}

// isUIService returns true if the service type indicates a UI/frontend service.
// UI services contain pre-built static assets rather than compiled binaries,
// and are handled differently during bundling (see stageUIService).
func isUIService(serviceType string) bool {
	switch serviceType {
	case "ui", "ui-bundle", "frontend", "web":
		return true
	default:
		return false
	}
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

func ensureBundleExtraResources(outputRoot, framework string) error {
	if framework == "" {
		framework = "electron"
	}
	pkgPath := filepath.Join(outputRoot, "platforms", framework, "package.json")

	// Skip if no package.json (Go-only scenarios, etc.)
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		return nil
	}

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
