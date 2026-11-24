package installgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const maxFrozenNPMHostToolPackages = 4096

type lockedRegistrySizer interface {
	LockedRegistry(ctx context.Context, name, version, resolved, integrity string) (artifactSize, error)
}

type frozenNPMPackage struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Resolved  string `json:"resolved"`
	Integrity string `json:"integrity"`
	Optional  bool   `json:"optional"`
}

type frozenNPMManifest struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

type frozenNPMLock struct {
	Name            string                        `json:"name"`
	Version         string                        `json:"version"`
	LockfileVersion int                           `json:"lockfileVersion"`
	Packages        map[string]frozenNPMLockEntry `json:"packages"`
}

type frozenNPMLockEntry struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Resolved             string            `json:"resolved"`
	Integrity            string            `json:"integrity"`
	Link                 bool              `json:"link"`
	Optional             bool              `json:"optional"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	OS                   []string          `json:"os"`
	CPU                  []string          `json:"cpu"`
}

// LockedRegistry sizes the exact tarball named by a lockfile. It verifies that
// registry metadata for the exact name/version returns the same URL and integrity
// before using the bounded archive-size probe.
func (r *registryHTTP) LockedRegistry(ctx context.Context, name, version, resolved, integrity string) (artifactSize, error) {
	if !validRegistryPackageName(name) || !exactNpmVersion.MatchString(version) || strings.TrimSpace(integrity) == "" {
		return artifactSize{}, fmt.Errorf("frozen npm entry %s@%s has an incomplete identity", name, version)
	}
	parsed, err := url.Parse(resolved)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "registry.npmjs.org" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return artifactSize{}, fmt.Errorf("frozen npm entry %s@%s does not use the approved registry URL", name, version)
	}
	doc, err := r.versionDocument(ctx, name, version)
	if err != nil {
		return artifactSize{}, err
	}
	if doc.Dist.Tarball == "" || resolved != doc.Dist.Tarball || doc.Dist.Integrity == "" || integrity != doc.Dist.Integrity {
		return artifactSize{}, fmt.Errorf("registry metadata for %s@%s differs from the frozen tarball URL or integrity", name, version)
	}
	archive, err := probePnpmArchiveSize(ctx, r.client, doc.Dist.Tarball, "registry.npmjs.org", r.beginRequest, r.recordResponse)
	if err != nil {
		return artifactSize{}, err
	}
	unpacked := doc.Dist.UnpackedSize
	if unpacked <= 0 {
		unpacked = archive * pnpmUnpackedFactor
	}
	if unpacked <= 0 || unpacked > DiskJobPlanningCeilingBytes/3 {
		return artifactSize{}, fmt.Errorf("unpacked size for %s@%s is unavailable or exceeds the bounded package limit", name, version)
	}
	return artifactSize{Unpacked: unpacked, Archive: archive}, nil
}

// planNPMFrozenHostToolDisk builds a conservative write-path map for a host
// tool whose executor runs npm ci against a committed package-lock.json. It
// rejects links, non-registry sources, floating versions, manifest/lock drift,
// and oversized closures; it never runs npm or writes into the install target.
func planNPMFrozenHostToolDisk(ctx context.Context, resolution Resolution) (DiskAdmissionPlan, error) {
	plan := DiskAdmissionPlan{Manager: "npm"}
	if resolution.Profile.LifecycleMode != "scripts-disabled-by-default" || !resolution.HostTool || !resolution.ExactVersion || resolution.SurfaceRoot == "" || resolution.PackageName == "" {
		plan.Unsupported = "frozen npm host-tool sizing requires the exact scripts-disabled host-tool profile"
		return plan, nil
	}
	if err := validateNpmRegistryFiles(resolution.SurfaceRoot); err != nil {
		plan.Unsupported = fmt.Sprintf("frozen npm host-tool registry configuration is not covered: %v", err)
		return plan, nil
	}
	manifestPath := filepath.Join(resolution.SurfaceRoot, "package.json")
	lockPath := filepath.Join(resolution.SurfaceRoot, "package-lock.json")
	manifestBytes, err := readRegularBounded(manifestPath, maxPnpmManifestBytes)
	if err != nil {
		plan.Unsupported = fmt.Sprintf("read frozen host-tool manifest: %v", err)
		return plan, nil
	}
	lockBytes, err := readRegularBounded(lockPath, maxNpmLockBytes)
	if err != nil {
		plan.Unsupported = fmt.Sprintf("read frozen host-tool lockfile: %v", err)
		return plan, nil
	}
	var manifest frozenNPMManifest
	var lock frozenNPMLock
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		plan.Unsupported = fmt.Sprintf("decode frozen host-tool manifest: %v", err)
		return plan, nil
	}
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		plan.Unsupported = fmt.Sprintf("decode frozen host-tool lockfile: %v", err)
		return plan, nil
	}
	root, ok := lock.Packages[""]
	if !ok || lock.LockfileVersion != 3 || manifest.Name == "" || manifest.Version == "" || manifest.Name != lock.Name || manifest.Version != lock.Version || manifest.Name != root.Name || manifest.Version != root.Version || !sameStringMap(manifest.Dependencies, root.Dependencies) || !sameStringMap(manifest.DevDependencies, root.DevDependencies) || !sameStringMap(manifest.OptionalDependencies, root.OptionalDependencies) {
		plan.Unsupported = "frozen host-tool package.json and lockfile root disagree or use an unsupported lockfile version"
		return plan, nil
	}
	targetVersion, targetFound := manifest.Dependencies[resolution.PackageName]
	targetLock, targetLocked := lock.Packages["node_modules/"+resolution.PackageName]
	if !targetFound || !exactNpmVersion.MatchString(targetVersion) || !targetLocked || targetLock.Version != targetVersion {
		plan.Unsupported = "frozen host-tool manifest does not directly pin its exact package in the lockfile"
		return plan, nil
	}
	keys := make([]string, 0, len(lock.Packages))
	for key := range lock.Packages {
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 || len(keys) > maxFrozenNPMHostToolPackages {
		plan.Unsupported = fmt.Sprintf("frozen host-tool lock has %d package entries outside the supported range [1,%d]", len(keys), maxFrozenNPMHostToolPackages)
		return plan, nil
	}
	packages := make([]frozenNPMPackage, 0, len(keys))
	for _, key := range keys {
		entry := lock.Packages[key]
		if !strings.HasPrefix(key, "node_modules/") || entry.Link {
			plan.Unsupported = fmt.Sprintf("frozen host-tool lock entry %q is not a regular node_modules package", key)
			return plan, nil
		}
		name := entry.Name
		if name == "" {
			name = key[strings.LastIndex(key, "node_modules/")+len("node_modules/"):]
		}
		if name == "" || !exactNpmVersion.MatchString(entry.Version) || entry.Resolved == "" || entry.Integrity == "" {
			plan.Unsupported = fmt.Sprintf("frozen host-tool lock entry %q lacks exact package, version, URL, or integrity", key)
			return plan, nil
		}
		packages = append(packages, frozenNPMPackage{Key: key, Name: name, Version: entry.Version, Resolved: entry.Resolved, Integrity: entry.Integrity, Optional: entry.Optional})
	}
	canonicalEntries, err := json.Marshal(packages)
	if err != nil {
		return DiskAdmissionPlan{}, err
	}
	closureDigest := sha256.Sum256(canonicalEntries)
	manifestDigest := sha256.Sum256(manifestBytes)
	lockDigest := sha256.Sum256(lockBytes)
	sizer, err := newRegistrySizer()
	if err != nil {
		plan.Unsupported = fmt.Sprintf("create frozen host-tool registry sizer: %v", err)
		return plan, nil
	}
	lockedSizer, ok := sizer.(lockedRegistrySizer)
	if !ok {
		plan.Unsupported = "registry sizer cannot verify a frozen npm tarball URL and integrity"
		return plan, nil
	}
	var archiveBytes, unpackedBytes int64
	for _, pkg := range packages {
		if err := ctx.Err(); err != nil {
			return DiskAdmissionPlan{}, err
		}
		measured, err := lockedSizer.LockedRegistry(ctx, pkg.Name, pkg.Version, pkg.Resolved, pkg.Integrity)
		if err != nil {
			plan.Unsupported = fmt.Sprintf("size frozen host-tool package %s@%s: %v", pkg.Name, pkg.Version, err)
			return plan, nil
		}
		if measured.Archive <= 0 || measured.Unpacked <= 0 || archiveBytes > DiskJobPlanningCeilingBytes-measured.Archive || unpackedBytes > DiskJobPlanningCeilingBytes-measured.Unpacked {
			plan.Unsupported = "frozen host-tool package closure exceeds the bounded size probe ceiling"
			return plan, nil
		}
		archiveBytes += measured.Archive
		unpackedBytes += measured.Unpacked
	}
	probeBytes := archiveBytes + unpackedBytes
	if probeBytes <= 0 || probeBytes > DiskJobPlanningCeilingBytes/4 {
		plan.Unsupported = fmt.Sprintf("frozen host-tool archive plus unpacked size %d exceeds the conservative per-job bound", probeBytes)
		return plan, nil
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return DiskAdmissionPlan{}, fmt.Errorf("resolve host-tool installation owner home: %w", err)
	}
	cacheHome, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyCache)
	if err != nil {
		return DiskAdmissionPlan{}, fmt.Errorf("resolve host-tool runtime cache path: %w", err)
	}
	installPath := filepath.Join(cacheHome, "host-tools", resolution.PackageName)
	cachePath, err := managerCachePath("npm", home)
	if err != nil {
		return DiskAdmissionPlan{}, err
	}
	envPeak := probeBytes * 2
	cachePeak := probeBytes
	tempPeak := probeBytes
	if envPeak > DiskJobPlanningCeilingBytes-cachePeak-tempPeak {
		plan.Unsupported = fmt.Sprintf("frozen host-tool write-path peak exceeds the %d-byte per-job planning ceiling", DiskJobPlanningCeilingBytes)
		return plan, nil
	}
	plan.CachePath = cachePath
	plan.ProbeBytes = probeBytes
	plan.Paths = []DiskPathRequirement{
		{Path: installPath, Role: "managed-host-tool-candidate-and-published-tree", PeakBytes: envPeak},
		{Path: cachePath, Role: "npm-frozen-host-tool-cache", PeakBytes: cachePeak},
		{Path: os.TempDir(), Role: "npm-frozen-host-tool-download-and-unpack-temp", PeakBytes: tempPeak},
	}
	plan.Notes = append(plan.Notes, frozenNPMHostToolReceipt{
		Version: 1, Package: resolution.PackageName, PackageVersion: targetVersion,
		LockfileVersion: lock.LockfileVersion, ManifestPath: manifestPath,
		ManifestSHA256: hex.EncodeToString(manifestDigest[:]), LockfilePath: lockPath,
		LockfileSHA256: hex.EncodeToString(lockDigest[:]), PackageSetSHA256: hex.EncodeToString(closureDigest[:]),
		PackageCount: len(packages), ArchiveBytes: archiveBytes, UnpackedBytes: unpackedBytes,
		ProbeBytes: probeBytes, InstallPath: installPath, InstallPathPeakBytes: envPeak,
		CachePath: cachePath, CachePathPeakBytes: cachePeak, TempPath: os.TempDir(), TempPathPeakBytes: tempPeak,
		Registry: "https://registry.npmjs.org", OptionalEntriesIncluded: true,
	}.note())
	return plan, nil
}

type frozenNPMHostToolReceipt struct {
	Version                 int    `json:"version"`
	Package                 string `json:"package"`
	PackageVersion          string `json:"package_version"`
	LockfileVersion         int    `json:"lockfile_version"`
	ManifestPath            string `json:"manifest_path"`
	ManifestSHA256          string `json:"manifest_sha256"`
	LockfilePath            string `json:"lockfile_path"`
	LockfileSHA256          string `json:"lockfile_sha256"`
	PackageSetSHA256        string `json:"package_set_sha256"`
	PackageCount            int    `json:"package_count"`
	ArchiveBytes            int64  `json:"archive_bytes"`
	UnpackedBytes           int64  `json:"unpacked_bytes"`
	ProbeBytes              int64  `json:"probe_bytes"`
	InstallPath             string `json:"install_path"`
	InstallPathPeakBytes    int64  `json:"install_path_peak_bytes"`
	CachePath               string `json:"cache_path"`
	CachePathPeakBytes      int64  `json:"cache_path_peak_bytes"`
	TempPath                string `json:"temp_path"`
	TempPathPeakBytes       int64  `json:"temp_path_peak_bytes"`
	Registry                string `json:"registry"`
	OptionalEntriesIncluded bool   `json:"optional_entries_included"`
}

func (r frozenNPMHostToolReceipt) note() string {
	data, _ := json.Marshal(r)
	return "frozen-npm-host-tool-size-receipt=" + string(data)
}

func sameStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}
