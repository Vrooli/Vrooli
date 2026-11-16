package installgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// A published Go module is sized from the module proxy: the new module's
// requirement closure, minus whatever the target go.mod already requires at
// the same or a newer version and the module cache already holds. Bytes are
// counted from the proxy's responses and never written during planning.

const goExternalPrefixProbeLimit = 6

type externalGoWalk struct {
	ctx      context.Context
	proxy    string
	client   *http.Client
	required map[string]string // module path -> version the target go.mod requires
	modCache string
	// listed means the modules visited came from `go list -deps`, which already
	// names everything the build needs, so a cached module's requirement graph
	// (most of it pruned out of the build) is not walked.
	listed bool
	seen   map[string]bool
	result goModuleProbeResult
	total  int64
}

func probeExternalGoModuleGrowth(ctx context.Context, r Resolution, manifest *modfile.File, proxy string, client *http.Client) (goModuleProbeResult, error) {
	required := make(map[string]string, len(manifest.Require))
	for _, req := range manifest.Require {
		required[req.Mod.Path] = req.Mod.Version
	}
	modCache, err := goModuleCacheDir()
	if err != nil {
		return goModuleProbeResult{}, err
	}
	walk := &externalGoWalk{ctx: ctx, proxy: proxy, client: client, required: required, modCache: modCache, seen: map[string]bool{}}

	version := requestedGoVersion(r)
	modulePath, err := walk.resolveModule(r.PackageName, version)
	if err != nil {
		return goModuleProbeResult{}, err
	}
	if version == "" || version == "latest" {
		latest, latestErr := walk.latest(modulePath)
		if latestErr != nil {
			return goModuleProbeResult{}, latestErr
		}
		return goModuleProbeResult{}, fmt.Errorf("Go installs must name an exact version so governance reviews what is installed; the proxy's latest release of %s is %s, so repeat with %s@%s", modulePath, latest, r.PackageName, latest)
	}
	if !semver.IsValid(version) {
		return goModuleProbeResult{}, fmt.Errorf("Go version %q is not an exact semantic version such as v1.2.3; branch, commit and range queries cannot be size-admitted", version)
	}
	if err := walk.visit(modulePath, version, true); err != nil {
		return goModuleProbeResult{}, err
	}
	return walk.result, nil
}

func requestedGoVersion(r Resolution) string {
	if len(r.Argv) == 0 {
		return ""
	}
	_, version, ok := strings.Cut(r.Argv[len(r.Argv)-1], "@")
	if !ok {
		return ""
	}
	return version
}

// resolveModule finds the module that provides a package. A requirement of the
// target go.mod answers without the network; otherwise the proxy is asked about
// each leading path, longest first, the way the go command does.
func (w *externalGoWalk) resolveModule(pkg, version string) (string, error) {
	best := ""
	for path := range w.required {
		if (pkg == path || strings.HasPrefix(pkg, path+"/")) && len(path) > len(best) {
			best = path
		}
	}
	if best != "" {
		return best, nil
	}
	if version == "" || version == "latest" || !semver.IsValid(version) {
		version = "latest"
	}
	candidate := pkg
	for probes := 0; probes < goExternalPrefixProbeLimit && candidate != ""; probes++ {
		if module.CheckPath(candidate) == nil {
			found, err := w.moduleExists(candidate, version)
			if err != nil {
				return "", err
			}
			if found {
				return candidate, nil
			}
		}
		cut := strings.LastIndexByte(candidate, '/')
		if cut < 0 {
			break
		}
		candidate = candidate[:cut]
	}
	return "", fmt.Errorf("no Go module on the public proxy provides package %q at %s; check the import path and version", pkg, version)
}

func (w *externalGoWalk) endpoint(path, suffix string) (string, error) {
	escaped, err := module.EscapePath(path)
	if err != nil {
		return "", fmt.Errorf("escape Go module path %q: %w", path, err)
	}
	return strings.TrimRight(w.proxy, "/") + "/" + escaped + "/" + suffix, nil
}

func (w *externalGoWalk) moduleExists(path, version string) (bool, error) {
	suffix := "@latest"
	if version != "latest" {
		escaped, err := module.EscapeVersion(version)
		if err != nil {
			return false, err
		}
		suffix = "@v/" + escaped + ".info"
	}
	endpoint, err := w.endpoint(path, suffix)
	if err != nil {
		return false, err
	}
	if _, err := boundedHTTPGet(w.ctx, w.client, endpoint, 1<<20); err != nil {
		if strings.Contains(err.Error(), "HTTP 404") || strings.Contains(err.Error(), "HTTP 410") {
			return false, nil
		}
		return false, fmt.Errorf("probe Go module %s: %w", path, err)
	}
	return true, nil
}

func (w *externalGoWalk) latest(path string) (string, error) {
	endpoint, err := w.endpoint(path, "@latest")
	if err != nil {
		return "", err
	}
	body, err := boundedHTTPGet(w.ctx, w.client, endpoint, 1<<20)
	if err != nil {
		return "", fmt.Errorf("probe latest Go module version for %s: %w", path, err)
	}
	var info struct{ Version string }
	if err := json.Unmarshal(body, &info); err != nil || !semver.IsValid(info.Version) {
		return "", fmt.Errorf("proxy returned no usable latest version for %s", path)
	}
	return info.Version, nil
}

func (w *externalGoWalk) count(size int64) error {
	if size < 0 || w.total > goModuleProbeTotalLimit-size {
		return fmt.Errorf("Go module dependency closure exceeds the %d-byte bounded probe limit", int64(goModuleProbeTotalLimit))
	}
	w.total += size
	return nil
}

// cached reports whether the module cache already holds the module's source.
func (w *externalGoWalk) cached(path, version string) bool {
	return goModuleSourceCached(w.modCache, path, version)
}

// visit counts one module and the requirements it would add. target is the
// module the request names; it is walked even when already required, because
// go get still needs its source in the module cache.
func (w *externalGoWalk) visit(path, version string, target bool) error {
	key := path + "@" + version
	if w.seen[key] {
		return nil
	}
	w.seen[key] = true
	if err := module.CheckPath(path); err != nil {
		return fmt.Errorf("Go dependency module path %q is invalid", path)
	}
	if !semver.IsValid(version) {
		return fmt.Errorf("Go dependency %s has an unsupported or non-exact version %q", path, version)
	}
	satisfied := false
	if have, ok := w.required[path]; ok && semver.Compare(have, version) >= 0 {
		// Minimal version selection keeps the version the target already
		// requires, so that is the module whose source the build needs.
		satisfied, version = true, have
	}
	if w.result.ModuleCount >= goModuleProbeCountLimit {
		return fmt.Errorf("Go module dependency closure exceeds %d modules", goModuleProbeCountLimit)
	}
	w.result.ModuleCount++
	if err := w.count(64 << 10); err != nil {
		return err
	}
	w.result.MetadataBytes += 64 << 10 // go.mod, ziphash, info and module-cache metadata allowance
	if satisfied && !target && w.cached(path, version) {
		// The build list is unchanged and the module cache holds the source.
		return nil
	}
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return fmt.Errorf("escape Go module version %q: %w", version, err)
	}
	base, err := w.endpoint(path, "@v/"+escapedVersion)
	if err != nil {
		return err
	}
	modBytes, haveMod := cachedGoModule(w.modCache, path, version)
	cached := w.cached(path, version)
	if !haveMod {
		if modBytes, err = boundedHTTPGet(w.ctx, w.client, base+".mod", 1<<20); err != nil {
			return fmt.Errorf("probe Go module metadata %s: %w", key, err)
		}
		if err := w.count(int64(len(modBytes))); err != nil {
			return err
		}
		w.result.MetadataBytes += int64(len(modBytes))
	}
	parsed, err := modfile.Parse(path+".mod", modBytes, nil)
	if err != nil {
		return fmt.Errorf("parse Go module metadata %s: %w", key, err)
	}
	if parsed.Module == nil || parsed.Module.Mod.Path != path {
		return fmt.Errorf("Go dependency %s declares a different module identity", key)
	}
	if !cached {
		zipBytes, err := boundedHTTPGet(w.ctx, w.client, base+".zip", goModuleProbeResponseLimit)
		if err != nil {
			return fmt.Errorf("probe Go module source %s: %w", key, err)
		}
		expanded, err := zipExpandedBytes(path, version, zipBytes)
		if err != nil {
			return fmt.Errorf("inspect Go module source %s: %w", key, err)
		}
		if err := w.count(int64(len(zipBytes)) + expanded); err != nil {
			return err
		}
		w.result.ArchiveBytes += int64(len(zipBytes))
		w.result.ExpandedBytes += expanded
	}
	if satisfied {
		return nil // The existing requirement graph already covers its dependencies.
	}
	if cached && w.listed {
		return nil // go list already named every module its packages need.
	}
	for _, req := range parsed.Require {
		if err := w.visit(req.Mod.Path, req.Mod.Version, false); err != nil {
			return err
		}
	}
	return nil
}
