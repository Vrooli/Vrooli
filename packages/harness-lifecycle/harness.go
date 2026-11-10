package agentinstall

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	harnesscatalog "github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// AntigravityManifestBase is the auto-updater service that publishes
// Antigravity's per-platform release manifests.
const AntigravityManifestBase = "https://antigravity-cli-auto-updater-974169037036.us-central1.run.app"

// Harness is what the update owner needs to know about one coding-agent
// resource on this host. It is derived from the resource manifest; nothing here
// restates a version or package the manifest owns.
type Harness struct {
	SignInProbe    *harnesscatalog.SignInProbe
	NativeArtifact *harnesscatalog.NativeArtifact
	Resource       string
	Binary         string
	Home           string
	BinDir         string
	// ReviewedVersion is the version the resource's permission policy was
	// authored against. It never selects what is installed.
	ReviewedVersion string
	Acquisition     *binaryfetch.Acquisition
	// Package is the npm package, empty for a harness released by manifest.
	Package string
	// Executable is the entry point relative to a version prefix.
	Executable string
	// ArchiveEntry names the executable inside a manifest-released archive.
	ArchiveEntry string
	// ManifestBase is set when upstream publishes a release manifest instead of
	// a package, which also means the Dependency Analyzer does not admit it.
	ManifestBase string
	// DelegatePath is a file the package's own launcher runs in preference to
	// the package's binary. See harnessDelegate.
	DelegatePath string
	// InstallerPaths are where an upstream installer puts its own copy.
	InstallerPaths []string
}

// harnessDelegate names the file a harness launcher prefers over its package.
//
// Grok's npm entry point is a wrapper: it runs $GROK_HOME/bin/grok when that
// exists and decompresses the package's binary there only when it does not.
// Installs run with lifecycle scripts disabled, so an older file left there by
// xAI's installer outlives every package update, and the selected package then
// reports the older version. That looks like a version-scheme mismatch; it is a
// stale delegate.
func harnessDelegate(resource, home string) string {
	if resource != "grok" {
		return ""
	}
	if custom := strings.TrimSpace(os.Getenv("GROK_HOME")); custom != "" {
		return filepath.Join(custom, "bin", "grok")
	}
	return filepath.Join(home, ".grok", "bin", "grok")
}

func harnessInstallerPaths(resource, home string) []string {
	if resource == "opencode" {
		return []string{filepath.Join(home, ".opencode", "bin", "opencode")}
	}
	return nil
}

func harnessManifestBase(resource string) string {
	if resource == "antigravity" {
		return AntigravityManifestBase
	}
	return ""
}

// LoadHarness reads one coding-agent resource from its manifest under root.
func LoadHarness(root, resource string) (Harness, error) {
	return loadHarness(root, resource, defaultHarnessLoadDeps())
}

// harnessLoadDeps isolates manifest and host-home lookups used while deriving
// a harness. These are narrow domain operations, not package-level hooks.
type harnessLoadDeps struct {
	loadManifest func(string) (harnessManifest, error)
	homeDir      func() (string, error)
	goos         string
	goarch       string
}

type harnessManifest struct {
	Name        string
	Acquisition *binaryfetch.Acquisition
}

func defaultHarnessLoadDeps() harnessLoadDeps {
	return defaultHarnessLoadDepsFromPlatform()
}

func loadHarness(root, resource string, deps harnessLoadDeps) (Harness, error) {
	platform := defaultPlatformDeps()
	if deps.goos == "" {
		deps.goos = platform.goos
	}
	if deps.goarch == "" {
		deps.goarch = platform.goarch
	}
	binary, err := cliutil.CodingAgentBinary(resource)
	if err != nil {
		return Harness{}, fmt.Errorf("%q is not a coding-agent resource: %w", resource, err)
	}
	path := filepath.Join(root, "resources", resource, "resource.json")
	manifest, err := deps.loadManifest(path)
	if err != nil {
		return Harness{}, err
	}
	home, err := deps.homeDir()
	if err != nil {
		return Harness{}, err
	}
	harness := Harness{
		Resource: resource, Binary: binary, Home: home, BinDir: filepath.Join(home, ".local", "bin"),
		Acquisition: manifest.Acquisition, ManifestBase: harnessManifestBase(resource),
		DelegatePath: harnessDelegate(resource, home), InstallerPaths: harnessInstallerPaths(resource, home),
		ReviewedVersion: reviewedVersion(path),
	}
	if manifest.Acquisition != nil {
		target, err := manifest.Acquisition.Resolve(binaryfetch.Facts{"os": deps.goos, "arch": deps.goarch})
		if err == nil && target.Unsupported == "" {
			harness.Package = target.Package
			if harness.Package != "" {
				harness.Executable = target.BinPath
				if harness.ReviewedVersion == "" {
					harness.ReviewedVersion = target.Version
				}
			} else {
				harness.ArchiveEntry = target.BinPath
			}
		}
	}
	if set, err := harnesscatalog.Load(root); err == nil {
		if descriptor, ok := set.Lookup(resource); ok {
			harness.SignInProbe = descriptor.SignInProbe
			if artifact, ok := descriptor.NativeArtifacts[deps.goos+"_"+deps.goarch]; ok {
				harness.NativeArtifact = &artifact
			}
		}
	}
	if harness.Executable == "" {
		harness.Executable = "node_modules/.bin/" + binary
		if harness.ManifestBase != "" {
			harness.Executable = binary
		}
	}
	return harness, nil
}

// reviewedVersion reads upstream_cli.version_pinned, the policy baseline Manage
// shows as "Permissions reviewed for". It is not part of the typed manifest
// because nothing else in the control plane acts on it.
func reviewedVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var manifest struct {
		UpstreamCLI struct {
			VersionPinned string `json:"version_pinned"`
		} `json:"upstream_cli"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return ""
	}
	return strings.TrimSpace(manifest.UpstreamCLI.VersionPinned)
}

func (h Harness) linkPath() string { return filepath.Join(h.BinDir, h.Binary) }

func (h Harness) versionsRoot() (string, error) {
	prefix, err := NPMInstallPrefix(h.Home, h.Resource, "0")
	if err != nil {
		return "", err
	}
	return filepath.Dir(prefix), nil
}

func (h Harness) spec() Spec {
	return Spec{ResourceName: h.Resource, Binary: h.Binary, BinDir: h.BinDir, Acquisition: h.Acquisition}
}

// managedInstall is one complete version prefix.
type managedInstall struct {
	Version    string
	Prefix     string
	Executable string
}

// installationRecord is the receipt written beside a prefix. Dependency
// Analyzer writes the same field names for npm installs.
type installationRecord struct {
	Resource   string
	Package    string
	Version    string
	TreeSHA256 string
	FileSHA256 string `json:",omitempty"`
}

func readInstallationRecord(prefix string) (installationRecord, error) {
	var record installationRecord
	data, err := os.ReadFile(filepath.Join(prefix, "installation.json"))
	if err != nil {
		return record, err
	}
	return record, json.Unmarshal(data, &record)
}

// managedInstalls lists complete prefixes for this platform, newest first. A
// directory without a matching receipt or executable is a failed candidate, not
// a version anything can return to.
func (h Harness) managedInstalls() ([]managedInstall, error) {
	root, err := h.versionsRoot()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	suffix := "-" + runtime.GOOS + "-" + runtime.GOARCH
	var installs []managedInstall
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		version := strings.TrimSuffix(entry.Name(), suffix)
		if upstreamcheck.ExtractSemver(version) != version {
			continue
		}
		prefix := filepath.Join(root, entry.Name())
		record, err := readInstallationRecord(prefix)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("invalid %s installation receipt: %w", prefix, err)
		}
		if record.Version != version || record.Resource != h.Resource || record.Package != h.Package {
			return nil, fmt.Errorf("installation receipt identity mismatch for %s: resource=%q package=%q version=%q expected=%q/%q/%q", prefix, record.Resource, record.Package, record.Version, h.Resource, h.Package, version)
		}
		if (record.FileSHA256 == "") == (record.TreeSHA256 == "") {
			return nil, fmt.Errorf("installation receipt has invalid digest fields for %s", prefix)
		}
		executable := filepath.Join(prefix, filepath.FromSlash(h.Executable))
		if _, err := os.Lstat(executable); err != nil {
			continue
		}
		install := managedInstall{Version: version, Prefix: prefix, Executable: executable}
		if err := h.verifyInstall(context.Background(), install); err != nil {
			return nil, fmt.Errorf("installation receipt verification failed for %s: %w", prefix, err)
		}
		installs = append(installs, managedInstall{Version: version, Prefix: prefix, Executable: executable})
	}
	sort.Slice(installs, func(i, j int) bool {
		return upstreamcheck.Compare(installs[i].Version, installs[j].Version) == upstreamcheck.StatusAhead
	})
	return installs, nil
}

type linkKind int

const (
	linkAbsent linkKind = iota
	linkManaged
	linkForeign
	linkStandalone
)

// linkState describes the path the harness is meant to be selected from.
type linkState struct {
	Kind    linkKind
	Version string
	Prefix  string
	Target  string
}

func (h Harness) readLink() (linkState, error) {
	path := h.linkPath()
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return linkState{Kind: linkAbsent}, nil
	}
	if err != nil {
		return linkState{}, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return linkState{Kind: linkStandalone}, nil
	}
	target, err := os.Readlink(path)
	if err != nil {
		return linkState{}, err
	}
	state := linkState{Kind: linkForeign, Target: target}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	root, err := h.versionsRoot()
	if err != nil {
		return linkState{}, err
	}
	relative, err := filepath.Rel(root, filepath.Clean(target))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return state, nil
	}
	suffix := "-" + runtime.GOOS + "-" + runtime.GOARCH
	directory := strings.Split(relative, string(filepath.Separator))[0]
	if version := strings.TrimSuffix(directory, suffix); version != directory && upstreamcheck.ExtractSemver(version) == version {
		state.Kind, state.Version, state.Prefix = linkManaged, version, filepath.Join(root, directory)
	}
	return state, nil
}

// RootOwnedHarnessPaths lists what a stray elevated run left root-owned among the
// files harness updates must replace. The harness_file_ownership safeguard hands
// exactly these back. It runs elevated, so its scope comes from home alone: it
// reads no GROK_HOME, which `sudo -E` could carry to a shared root-owned install,
// and it drops any candidate whose parent resolves outside home.
func RootOwnedHarnessPaths(home string) ([]string, error) {
	candidates, err := harnessFileCandidates(home)
	if err != nil {
		return nil, err
	}
	owned := []string{}
	for _, candidate := range candidates {
		if !parentWithinHome(candidate, home) {
			continue
		}
		if info, err := os.Lstat(candidate); err == nil && ownedByRoot(info) {
			owned = append(owned, candidate)
		}
	}
	return owned, nil
}

// codexScratchPaths lists Codex's per-launch scratch directories and their
// entries. A privileged launch leaves them root-owned, and every later launch
// then warns that it cannot clean up stale arg0 temp dirs.
func codexScratchPaths(home string) []string {
	var paths []string
	for _, pattern := range []string{
		filepath.Join(home, ".codex", "tmp", "arg0-linux", "*"),
		filepath.Join(home, ".codex", "tmp", "arg0-linux", "*", "*"),
		filepath.Join(home, ".codex", "tmp", "arg0-linux", "*", ".*"),
	} {
		matches, _ := filepath.Glob(pattern)
		paths = append(paths, matches...)
	}
	return paths
}

func harnessFileCandidates(home string) ([]string, error) {
	binDir := filepath.Join(home, ".local", "bin")
	candidates := []string{binDir}
	for _, alias := range cliutil.CodingAgentAliases() {
		candidates = append(candidates, filepath.Join(binDir, alias))
		resource, ok := cliutil.CodingAgentForAlias(alias)
		if !ok {
			continue
		}
		candidates = append(candidates, harnessInstallerPaths(resource, home)...)
		if resource == "codex" {
			candidates = append(candidates, codexScratchPaths(home)...)
		}
		if resource == "grok" {
			delegate := filepath.Join(home, ".grok", "bin", "grok")
			candidates = append(candidates, delegate)
			versioned, err := filepath.Glob(delegate + "-*")
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, versioned...)
		}
		prefix, err := NPMInstallPrefix(home, resource, "0")
		if err != nil {
			return nil, err
		}
		walkErr := filepath.WalkDir(filepath.Dir(prefix), func(path string, _ os.DirEntry, err error) error {
			if os.IsNotExist(err) {
				return nil
			}
			// An unreadable directory is itself the finding; its owner change is
			// what makes its contents readable on the next run.
			if os.IsPermission(err) {
				candidates = append(candidates, path)
				return filepath.SkipDir
			}
			if err != nil {
				return err
			}
			candidates = append(candidates, path)
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return candidates, nil
}

// parentWithinHome resolves the parent, because a symlinked directory is how a
// path that reads as under home ends up somewhere else.
func parentWithinHome(path, home string) bool {
	root, err := filepath.EvalSymlinks(home)
	if err != nil {
		return false
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	return err == nil && withinDir(parent, root)
}

// safeguardCovers says whether harness_file_ownership can repair this path.
func safeguardCovers(path, home string) bool { return parentWithinHome(path, home) }
