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
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

const (
	goModuleProbeResponseLimit = 256 << 20
	goModuleProbeTotalLimit    = 2 << 30
	goModuleProbeCountLimit    = 256
	goBuildCacheFloorBytes     = 256 << 20
)

type goModuleProbeResult struct {
	SourceBytes   int64
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

// probeGoModuleGrowth supports only exact repository-local Go modules and
// computes a bounded upper estimate from their source trees and the complete
// pinned go.mod closure. External modules are streamed from the official Go
// module proxy with strict per-response and total limits; bytes are counted,
// never written during planning. Unsupported graph features fail closed.
func probeGoModuleGrowth(ctx context.Context, r Resolution) (int64, error) {
	probe, err := probeGoModuleGrowthDetails(ctx, r)
	if err != nil {
		return 0, err
	}
	return probe.SourceBytes + probe.ArchiveBytes + probe.ExpandedBytes + probe.MetadataBytes, nil
}

func probeGoModuleGrowthDetails(ctx context.Context, r Resolution) (goModuleProbeResult, error) {
	if r.RepositoryRoot == "" || r.PackageName == "" || r.HostTool || r.HostSource || r.HostToolRefresh || r.TemplateVersion != "" {
		return goModuleProbeResult{}, fmt.Errorf("Go disk admission requires an exact repository-local module target")
	}
	proxy, err := goModuleProxyURL()
	if err != nil {
		return goModuleProbeResult{}, err
	}
	topo, err := gomodreconcile.LoadTopology(r.RepositoryRoot)
	if err != nil {
		return goModuleProbeResult{}, fmt.Errorf("load repository Go module topology: %w", err)
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
		return goModuleProbeResult{}, fmt.Errorf("Go package %q is not backed by an exact repository-local module", r.PackageName)
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
	probe, err := probeGoModuleClosure(ctx, r.RepositoryRoot, modulePath, targetModule, manifest, topo, proxy, goModuleHTTPClient)
	if err != nil {
		return goModuleProbeResult{}, err
	}
	return probe, nil
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

func probeGoModuleClosure(ctx context.Context, repoRoot, rootModule, targetModule string, targetManifest *modfile.File, topo gomodreconcile.Topology, proxy string, client *http.Client) (goModuleProbeResult, error) {
	result := goModuleProbeResult{}
	seenLocal := map[string]bool{}
	seenLocalWalk := map[string]bool{}
	seenRemote := map[string]bool{}
	var closureBytes int64
	var visitLocal func(string, bool) error
	var visitRemote func(string, string) error
	count := func(size int64) error {
		if size < 0 || closureBytes > goModuleProbeTotalLimit-size {
			return fmt.Errorf("Go module dependency closure exceeds the %d-byte bounded probe limit", goModuleProbeTotalLimit)
		}
		closureBytes += size
		return nil
	}
	localTopology := make(gomodreconcile.Topology, len(topo))
	explicitLocal := make(map[string]bool, len(targetManifest.Replace))
	for path, dir := range topo {
		localTopology[path] = dir
	}
	if len(targetManifest.Exclude) > 0 {
		return goModuleProbeResult{}, fmt.Errorf("target Go module uses exclude directives unsupported by bounded admission")
	}
	for _, replacement := range targetManifest.Replace {
		if replacement.Old.Version != "" || replacement.New.Version != "" || replacement.New.Path == "" {
			return goModuleProbeResult{}, fmt.Errorf("target Go module has a versioned or empty replace unsupported by bounded admission")
		}
		replacementDir := replacement.New.Path
		if !filepath.IsAbs(replacementDir) {
			replacementDir = filepath.Join(filepath.Dir(targetManifest.Syntax.Name), replacementDir)
		}
		var err error
		replacementDir, err = filepath.Abs(replacementDir)
		if err != nil {
			return goModuleProbeResult{}, fmt.Errorf("resolve target local Go replacement %s: %w", replacement.Old.Path, err)
		}
		if err := verifyWithinRoot(repoRoot, replacementDir); err != nil {
			return goModuleProbeResult{}, fmt.Errorf("target Go replacement %s: %w", replacement.Old.Path, err)
		}
		replacementManifest, err := readGoModuleFile(filepath.Join(replacementDir, "go.mod"))
		if err != nil || replacementManifest.Module == nil || replacementManifest.Module.Mod.Path != replacement.Old.Path {
			return goModuleProbeResult{}, fmt.Errorf("target Go replacement %s does not resolve to an in-repository module with the same identity", replacement.Old.Path)
		}
		localTopology[replacement.Old.Path] = replacementDir
		explicitLocal[replacement.Old.Path] = true
	}
	visitLocal = func(path string, resolveRepositoryModules bool) error {
		walkKey := fmt.Sprintf("%s|%t", path, resolveRepositoryModules)
		if seenLocalWalk[walkKey] {
			return nil
		}
		seenLocalWalk[walkKey] = true
		dir, ok := localTopology[path]
		if !ok {
			return fmt.Errorf("repository Go module %q is missing from topology", path)
		}
		if err := verifyWithinRoot(repoRoot, dir); err != nil {
			return err
		}
		if !seenLocal[path] {
			if result.ModuleCount >= goModuleProbeCountLimit {
				return fmt.Errorf("Go module dependency closure exceeds %d modules", goModuleProbeCountLimit)
			}
			bytesOnDisk, err := boundedTreeSize(dir, goModuleProbeTotalLimit-closureBytes)
			if err != nil {
				return fmt.Errorf("measure repository module %s: %w", path, err)
			}
			if err := count(bytesOnDisk); err != nil {
				return err
			}
			result.SourceBytes += bytesOnDisk
			result.ModuleCount++
			if err := count(64 << 10); err != nil {
				return err
			}
			result.MetadataBytes += 64 << 10 // manifest, checksum, and module-cache metadata allowance
			seenLocal[path] = true
		}
		view, err := readGoModuleFile(filepath.Join(dir, "go.mod"))
		if err != nil {
			return fmt.Errorf("read repository module %s: %w", path, err)
		}
		if view.Module.Mod.Path != path {
			return fmt.Errorf("repository module identity mismatch: topology %q, go.mod %q", path, view.Module.Mod.Path)
		}
		requires := append([]*modfile.Require(nil), view.Require...)
		sort.Slice(requires, func(i, j int) bool { return requires[i].Mod.Path < requires[j].Mod.Path })
		for _, req := range requires {
			if explicitLocal[req.Mod.Path] {
				if err := visitLocal(req.Mod.Path, false); err != nil {
					return err
				}
				continue
			}
			if resolveRepositoryModules {
				if _, local := topo[req.Mod.Path]; local {
					if err := visitLocal(req.Mod.Path, true); err != nil {
						return err
					}
					continue
				}
			}
			if err := visitRemote(req.Mod.Path, req.Mod.Version); err != nil {
				return err
			}
		}
		return nil
	}
	visitRemote = func(path, version string) error {
		if err := module.CheckPath(path); err != nil {
			return fmt.Errorf("Go dependency module path %q is invalid", path)
		}
		if !semver.IsValid(version) {
			return fmt.Errorf("Go dependency %s has an unsupported or non-exact version %q", path, version)
		}
		key := path + "@" + version
		if seenRemote[key] {
			return nil
		}
		if result.ModuleCount >= goModuleProbeCountLimit {
			return fmt.Errorf("Go module dependency closure exceeds %d modules", goModuleProbeCountLimit)
		}
		seenRemote[key] = true
		result.ModuleCount++
		if err := count(64 << 10); err != nil {
			return err
		}
		result.MetadataBytes += 64 << 10 // .info, list, ziphash, and module-cache metadata allowance
		modPath, err := module.EscapePath(path)
		if err != nil {
			return fmt.Errorf("escape Go module path %q: %w", path, err)
		}
		modVersion, err := module.EscapeVersion(version)
		if err != nil {
			return fmt.Errorf("escape Go module version %q: %w", version, err)
		}
		base := strings.TrimRight(proxy, "/") + "/" + modPath + "/@v/" + modVersion
		modBytes, err := boundedHTTPGet(ctx, client, base+".mod", 1<<20)
		if err != nil {
			return fmt.Errorf("probe Go module metadata %s: %w", key, err)
		}
		if err := count(int64(len(modBytes))); err != nil {
			return err
		}
		result.MetadataBytes += int64(len(modBytes))
		parsed, err := modfile.Parse(path+".mod", modBytes, nil)
		if err != nil {
			return fmt.Errorf("parse Go module metadata %s: %w", key, err)
		}
		if parsed.Module == nil || parsed.Module.Mod.Path != path || len(parsed.Replace) > 0 || len(parsed.Exclude) > 0 {
			return fmt.Errorf("Go dependency %s has unsupported identity, replace, or exclude metadata", key)
		}
		zipBytes, err := boundedHTTPGet(ctx, client, base+".zip", goModuleProbeResponseLimit)
		if err != nil {
			return fmt.Errorf("probe Go module source %s: %w", key, err)
		}
		compressed := int64(len(zipBytes))
		expanded, err := zipExpandedBytes(path, version, zipBytes)
		if err != nil {
			return fmt.Errorf("inspect Go module source %s: %w", key, err)
		}
		if err := count(compressed + expanded); err != nil {
			return err
		}
		result.ArchiveBytes += compressed
		result.ExpandedBytes += expanded
		requires := append([]*modfile.Require(nil), parsed.Require...)
		sort.Slice(requires, func(i, j int) bool { return requires[i].Mod.Path < requires[j].Mod.Path })
		for _, req := range requires {
			if explicitLocal[req.Mod.Path] {
				if err := visitLocal(req.Mod.Path, false); err != nil {
					return err
				}
				continue
			}
			if err := visitRemote(req.Mod.Path, req.Mod.Version); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visitLocal(rootModule, true); err != nil {
		return goModuleProbeResult{}, err
	}
	if targetModule != rootModule {
		if err := visitLocal(targetModule, false); err != nil {
			return goModuleProbeResult{}, err
		}
	}
	return result, nil
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

func boundedTreeSize(root string, remaining int64) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".vrooli", "node_modules", "vendor", "testdata", "data", "dist", "build", ".cache":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("module source contains a non-regular or linked file")
		}
		if info.Size() > remaining-total {
			return fmt.Errorf("module source exceeds bounded probe limit")
		}
		total += info.Size()
		return nil
	})
	return total, err
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

func zipExpandedBytes(modulePath, version string, raw []byte) (int64, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return 0, err
	}
	want, err := module.EscapePath(modulePath)
	if err != nil {
		return 0, err
	}
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return 0, err
	}
	prefix := want + "@" + escapedVersion + "/"
	var total int64
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, prefix) || file.UncompressedSize64 > uint64(goModuleProbeTotalLimit-total) {
			return 0, fmt.Errorf("module zip has an invalid path or exceeds the expanded-size limit")
		}
		total += int64(file.UncompressedSize64)
	}
	if total == 0 {
		return 0, fmt.Errorf("module zip has no source bytes")
	}
	return total, nil
}
