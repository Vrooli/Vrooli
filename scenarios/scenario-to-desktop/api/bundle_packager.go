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
	"sort"
	"strings"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

type bundlePackageResult struct {
	BundleDir       string
	ManifestPath    string
	RuntimeBinaries map[string]string
	CopiedArtifacts []string
}

func packageBundle(appPath, manifestPath string, requestedPlatforms []string) (*bundlePackageResult, error) {
	return newBundlePackager().packageBundle(appPath, manifestPath, requestedPlatforms)
}

type runtimeResolver func() (string, error)
type runtimeBuilder func(srcDir, outPath, goos, goarch, target string) error

type bundlePackager struct {
	runtimeResolver runtimeResolver
	runtimeBuilder  runtimeBuilder
}

func newBundlePackager() *bundlePackager {
	return &bundlePackager{
		runtimeResolver: findRuntimeSourceDir,
		runtimeBuilder:  buildRuntimeBinary,
	}
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
			if !ok {
				return nil, fmt.Errorf("service %s missing binary for %s", svc.ID, platform)
			}
			src, err := resolveManifestPath(manifestRoot, bin.Path)
			if err != nil {
				return nil, fmt.Errorf("resolve binary for %s: %w", svc.ID, err)
			}
			dst, err := resolveBundlePath(bundleDir, bin.Path)
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
			dst, err := resolveBundlePath(bundleDir, asset.Path)
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

	return &bundlePackageResult{
		BundleDir:       bundleDir,
		ManifestPath:    destManifest,
		RuntimeBinaries: runtimeBinaries,
		CopiedArtifacts: copied,
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
			if _, ok := resolveBinaryForPlatform(svc, platform); !ok {
				return fmt.Errorf("service %s missing binary for %s", svc.ID, platform)
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
	for _, key := range keys {
		if bin, ok := svc.Binaries[key]; ok {
			return bin, true
		}
	}
	return bundlemanifest.Binary{}, false
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

func resolveManifestPath(root, rel string) (string, error) {
	clean := bundlemanifest.ResolvePath(root, rel)
	if !withinBase(root, clean) {
		return "", fmt.Errorf("path escapes manifest root: %s", rel)
	}
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

func copyFile(src, dst string) error {
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
