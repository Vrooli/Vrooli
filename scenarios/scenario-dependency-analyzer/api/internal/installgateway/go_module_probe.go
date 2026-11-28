package installgateway

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	goModuleProbeResponseLimit = 256 << 20
	goModuleProbeTotalLimit    = 2 << 30
	goModuleProbeCountLimit    = 256
	goBuildCacheFloorBytes     = 256 << 20
	// goInputsOnlyBuildAllowance is the build-cache reservation of a Go install
	// that downloads nothing.
	goInputsOnlyBuildAllowance = 16 << 20
)

type goModuleProbeResult struct {
	ArchiveBytes  int64
	ExpandedBytes int64
	MetadataBytes int64
	ModuleCount   int64
}

var goModuleHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(request *http.Request, _ []*http.Request) error {
		if request.URL.Scheme != "https" || request.URL.User != nil || (request.URL.Host != "proxy.golang.org" && request.URL.Host != "storage.googleapis.com") {
			return fmt.Errorf("Go module proxy redirected outside the approved HTTPS hosts")
		}
		return nil
	},
}

// probeGoModuleGrowth computes the bounded size of a governed `go get`. A
// package of a repository module costs the external modules its import closure
// needs (go_repo_package.go); a published module costs its requirement closure
// from the official proxy (go_external_module.go). Bytes are streamed from the
// proxy with strict limits and counted, never written during planning. Graph
// features it cannot bound fail closed with the reason.
func probeGoModuleGrowth(ctx context.Context, r Resolution) (int64, error) {
	probe, err := probeGoModuleGrowthDetails(ctx, r)
	if err != nil {
		return 0, err
	}
	return probe.ArchiveBytes + probe.ExpandedBytes + probe.MetadataBytes, nil
}

func probeGoModuleGrowthDetails(ctx context.Context, r Resolution) (goModuleProbeResult, error) {
	if r.RepositoryRoot == "" || r.PackageName == "" || r.HostTool || r.HostSource || r.HostToolRefresh || r.TemplateVersion != "" {
		return goModuleProbeResult{}, fmt.Errorf("Go disk admission requires a repository root, a package name, and a repository Go module manifest")
	}
	proxy, err := goModuleProxyURL()
	if err != nil {
		return goModuleProbeResult{}, err
	}
	topo, err := loadGoTopology(r.RepositoryRoot)
	if err != nil {
		return goModuleProbeResult{}, fmt.Errorf("load repository Go module topology: %w", err)
	}
	manifest, err := readBoundedTargetGoModule(r.RepositoryRoot, r.ManifestPath)
	if err != nil {
		return goModuleProbeResult{}, err
	}
	if manifest.Module == nil {
		return goModuleProbeResult{}, fmt.Errorf("target Go manifest has no module identity")
	}
	targetModule := manifest.Module.Mod.Path
	manifestAbs, err := filepath.Abs(r.ManifestPath)
	if err != nil {
		return goModuleProbeResult{}, err
	}
	if targetDir, ok := topo[targetModule]; !ok || filepath.Clean(targetDir) != filepath.Clean(filepath.Dir(manifestAbs)) {
		return goModuleProbeResult{}, fmt.Errorf("target Go manifest does not match the repository module topology")
	}
	modulePath := ""
	for path := range topo {
		if r.PackageName == path || strings.HasPrefix(r.PackageName, path+"/") {
			if len(path) > len(modulePath) {
				modulePath = path
			}
		}
	}
	if modulePath == "" {
		// Not a repository module: a published module fetched through the proxy.
		return probeExternalGoModuleGrowth(ctx, r, manifest, proxy, goModuleHTTPClient)
	}
	modCache, err := goModuleCacheDir()
	if err != nil {
		return goModuleProbeResult{}, err
	}
	ownerDir := topo[modulePath]
	if err := verifyWithinRoot(r.RepositoryRoot, ownerDir); err != nil {
		return goModuleProbeResult{}, err
	}
	owner, err := readGoModuleFile(filepath.Join(ownerDir, "go.mod"))
	if err != nil {
		return goModuleProbeResult{}, fmt.Errorf("read repository module %s: %w", modulePath, err)
	}
	return probeRepositoryPackageGrowth(ctx, r, ownerDir, owner, manifest, proxy, modCache, goModuleHTTPClient)
}

func goModuleProxyURL() (string, error) {
	for _, key := range []string{"GOPROXY", "GOSUMDB", "GOWORK"} {
		if value, ok := os.LookupEnv(key); ok {
			switch key {
			case "GOPROXY":
				if strings.TrimSpace(value) != "https://proxy.golang.org,direct" && strings.TrimSpace(value) != "https://proxy.golang.org" {
					return "", fmt.Errorf("custom GOPROXY cannot be covered by the official bounded Go module probe")
				}
			case "GOSUMDB":
				if strings.TrimSpace(value) != "sum.golang.org" {
					return "", fmt.Errorf("custom GOSUMDB cannot be covered by the bounded Go module probe")
				}
			case "GOWORK":
				if strings.TrimSpace(value) != "off" {
					return "", fmt.Errorf("Go workspace mode cannot be covered by module-only disk admission")
				}
			}
		}
	}
	return "https://proxy.golang.org", nil
}

// goModuleSourceCached reports whether the module cache holds the module's
// extracted source.
func goModuleSourceCached(modCache, path, version string) bool {
	if modCache == "" {
		return false
	}
	escapedPath, pathErr := module.EscapePath(path)
	escapedVersion, versionErr := module.EscapeVersion(version)
	if pathErr != nil || versionErr != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(modCache, filepath.FromSlash(escapedPath)+"@"+escapedVersion))
	return err == nil && info.IsDir()
}

// goModuleCacheDir is the module cache every governed Go install uses.
func goModuleCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve Go module cache for disk admission: %w", err)
	}
	return managerCachePath("go", home)
}

// cachedGoModule returns a module's go.mod from the module cache when the cache
// also holds its extracted source, which is what a build needs.
func cachedGoModule(modCache, path, version string) ([]byte, bool) {
	if modCache == "" {
		return nil, false
	}
	escapedPath, pathErr := module.EscapePath(path)
	escapedVersion, versionErr := module.EscapeVersion(version)
	if pathErr != nil || versionErr != nil {
		return nil, false
	}
	if !goModuleSourceCached(modCache, path, version) {
		return nil, false
	}
	mod, err := readRegularBounded(filepath.Join(modCache, "cache", "download", filepath.FromSlash(escapedPath), "@v", escapedVersion+".mod"), 1<<20)
	if err != nil {
		return nil, false
	}
	return mod, true
}

func readGoModuleFile(path string) (*modfile.File, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 1<<20 {
		return nil, fmt.Errorf("Go module file is unavailable, linked, non-regular, or exceeds 1 MiB")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("Go module file exceeds 1 MiB")
	}
	return modfile.Parse(path, data, nil)
}

func readBoundedTargetGoModule(repoRoot, manifestPath string) (*modfile.File, error) {
	if filepath.Base(manifestPath) != "go.mod" {
		return nil, fmt.Errorf("Go disk admission requires the exact target go.mod path")
	}
	if err := verifyWithinRoot(repoRoot, filepath.Dir(manifestPath)); err != nil {
		return nil, fmt.Errorf("target go.mod path: %w", err)
	}
	info, err := os.Lstat(manifestPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 1<<20 {
		return nil, fmt.Errorf("target go.mod is unavailable, linked, non-regular, or exceeds 1 MiB")
	}
	return readGoModuleFile(manifestPath)
}

func verifyWithinRoot(root, candidate string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("Go module source path escapes repository root")
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return err
	}
	candidateReal, err := filepath.EvalSymlinks(candidateAbs)
	if err != nil {
		return err
	}
	if filepath.Clean(rootReal) != filepath.Clean(rootAbs) || filepath.Clean(candidateReal) != filepath.Clean(candidateAbs) {
		return fmt.Errorf("Go module source path contains a symlink")
	}
	return nil
}

func boundedHTTPGet(ctx context.Context, client *http.Client, endpoint string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	if response.ContentLength > limit {
		return nil, fmt.Errorf("response Content-Length %d exceeds %d-byte limit", response.ContentLength, limit)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d-byte limit", limit)
	}
	return body, nil
}

// zipExpandedBytes totals a module zip's uncompressed bytes. Entries are named
// with the module path and version exactly as written (case included); only the
// proxy URL and the on-disk cache escape upper-case letters.
func zipExpandedBytes(modulePath, version string, raw []byte) (int64, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return 0, err
	}
	prefix := modulePath + "@" + version + "/"
	var total int64
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, prefix) {
			return 0, fmt.Errorf("module zip entry %q is not under %q", file.Name, prefix)
		}
		if file.UncompressedSize64 > uint64(goModuleProbeTotalLimit-total) {
			return 0, fmt.Errorf("module zip exceeds the %d-byte expanded-size limit", goModuleProbeTotalLimit)
		}
		total += int64(file.UncompressedSize64)
	}
	if total == 0 {
		return 0, fmt.Errorf("module zip has no source bytes")
	}
	return total, nil
}
