package agentinstall

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// UpdateCommand exposes the latest-release updater to a resource CLI.
func UpdateCommand(spec Spec) cliapp.Command {
	return cliapp.Command{
		Name:        "update",
		Description: "Update the upstream CLI to the latest published release",
		Run: func(args []string) error {
			if err := requireNoInstallArgs("update", args); err != nil {
				return err
			}
			return Update(context.Background(), spec)
		},
	}
}

// updateDeps carries the seams that let Update run without npm or the network.
type updateDeps struct {
	platform      platformDeps
	resolveLatest func(ctx context.Context, kind upstreamcheck.SourceKind, id string) (string, error)
	installed     func(ctx context.Context, path string) (string, error)
	install       func(ctx context.Context, spec Spec, packageRef string, platform platformDeps) error
	lookPath      func(string) (string, error)
	stat          func(string) (os.FileInfo, error)
	stdout        io.Writer
}

func defaultUpdateDeps() updateDeps {
	return updateDeps{
		platform: defaultPlatformDeps(),
		resolveLatest: func(ctx context.Context, kind upstreamcheck.SourceKind, id string) (string, error) {
			handlers := upstreamcheck.Default(upstreamcheck.Config{SourceKind: kind, SourceID: id})
			return handlers.LatestFetcher(ctx, kind, id)
		},
		install:  installNPMWithDeps,
		lookPath: cliutil.ResolveAgentBinary,
		stat:     os.Stat,
		stdout:   os.Stdout,
	}
}

// Update installs the latest published release of an npm-acquired CLI into the
// user-owned prefix.
//
// The manifest acquisition pins a version for reproducible provisioning;
// `update` deliberately bypasses that pin so an operator can move a host tool
// forward without editing the committed manifest. Dependency Analyzer admits
// the exact resource/package/version before any package-manager mutation.
func Update(ctx context.Context, spec Spec) error {
	return update(ctx, spec, defaultUpdateDeps())
}

func update(ctx context.Context, spec Spec, deps updateDeps) error {
	deps.platform = deps.platform.withDefaults()
	if deps.installed == nil {
		deps.installed = func(ctx context.Context, path string) (string, error) {
			return installedVersion(ctx, path, deps.platform)
		}
	}
	if strings.TrimSpace(spec.Binary) == "" || strings.TrimSpace(spec.BinDir) == "" {
		return fmt.Errorf("binary and bin directory are required")
	}
	if base := harnessManifestBase(spec.ResourceName); base != "" {
		return updateFromManifest(ctx, spec, base, deps)
	}
	acquisition := spec.Acquisition
	if acquisition == nil {
		loaded, err := acquisitionFromSourceManifestWithDeps(spec.ResourceName, deps.platform)
		if err != nil {
			return err
		}
		acquisition = loaded
	}
	kind := upstreamcheck.SourceKind(strings.TrimSpace(spec.UpstreamSourceKind))
	if kind == "" {
		kind = upstreamcheck.SourceNPM
	}
	id := strings.TrimSpace(spec.UpstreamSourceID)
	// The acquisition is authoritative when it is reachable, but a bare resource
	// CLI run has no source manifest to load, so an npm source id doubles as the
	// package to install.
	packageName := npmPackageName(acquisition)
	if packageName == "" && kind == upstreamcheck.SourceNPM {
		packageName = id
	}
	if packageName == "" {
		return fmt.Errorf("update %s: only npm-acquired resources can be updated to latest", spec.Binary)
	}
	if id == "" {
		id = packageName
	}

	latest, err := deps.resolveLatest(ctx, kind, id)
	if err != nil {
		return fmt.Errorf("resolve latest %s release: %w", packageName, err)
	}
	latestVersion := upstreamcheck.ExtractSemver(latest)
	if latestVersion == "" {
		return fmt.Errorf("resolve latest %s release: could not parse a version from %q", packageName, latest)
	}

	target := filepath.Join(spec.BinDir, spec.Binary)
	if deps.platform.windows {
		target += ".exe"
	}
	installedVersion := ""
	if raw, probeErr := deps.installed(ctx, target); probeErr == nil {
		installedVersion = upstreamcheck.ExtractSemver(raw)
	}
	if installedVersion != "" && (upstreamcheck.Compare(installedVersion, latestVersion) == upstreamcheck.StatusUpToDate || upstreamcheck.Compare(installedVersion, latestVersion) == upstreamcheck.StatusAhead) {
		if err := verifySelectedUpdate(ctx, spec.Binary, target, installedVersion, deps); err != nil {
			return err
		}
		fmt.Fprintf(deps.stdout, "%s verified at %s (latest release %s; no downgrade)\n", spec.Binary, installedVersion, latestVersion)
		return nil
	}

	if err := deps.platform.mkdirAll(spec.BinDir, deps.platform.permDir); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}
	if spec.DataDir != "" {
		if err := deps.platform.mkdirAll(spec.DataDir, deps.platform.permPrivate); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}
	}
	if blocker, err := BlockingSystemInstall(spec.Binary, spec.BinDir, deps.lookPath, deps.stat); err != nil {
		return err
	} else if blocker != "" {
		return fmt.Errorf("refusing to overwrite root-owned %s at %s", spec.Binary, blocker)
	}

	if err := deps.install(ctx, spec, packageName+"@"+latestVersion, deps.platform); err != nil {
		return err
	}
	change := &harnessChange{}
	if home, err := deps.platform.homeDir(); err == nil {
		delegate := Harness{Resource: spec.ResourceName, Binary: spec.Binary, BinDir: spec.BinDir, DelegatePath: harnessDelegate(spec.ResourceName, home)}
		if err := delegate.refreshDelegate(ctx, latestVersion, harnessDeps{version: deps.installed, rootOwn: ownedByRoot, now: time.Now}, change); err != nil {
			return err
		}
	}
	if err := verifySelectedUpdate(ctx, spec.Binary, target, latestVersion, deps); err != nil {
		return errors.Join(err, change.revert(ctx))
	}
	change.commit()
	fmt.Fprintf(deps.stdout, "%s updated %s -> %s\n", spec.Binary, orUnknownVersion(installedVersion), latestVersion)
	return nil
}

// updateFromManifest is the resource CLI's route for a harness released by
// manifest. It keeps update's contract: never a downgrade, always verified.
func updateFromManifest(ctx context.Context, spec Spec, base string, deps updateDeps) error {
	deps.platform = deps.platform.withDefaults()
	release, err := fetchReleaseManifestWithDeps(ctx, base, releasePlatformWith(deps.platform), deps.platform)
	if err != nil {
		return fmt.Errorf("resolve latest %s release: %w", spec.Binary, err)
	}
	target := filepath.Join(spec.BinDir, spec.Binary)
	installedVersion := ""
	if raw, probeErr := deps.installed(ctx, target); probeErr == nil {
		installedVersion = upstreamcheck.ExtractSemver(raw)
	}
	if status := upstreamcheck.Compare(installedVersion, release.Version); status == upstreamcheck.StatusUpToDate || status == upstreamcheck.StatusAhead {
		if err := verifySelectedUpdate(ctx, spec.Binary, target, installedVersion, deps); err != nil {
			return err
		}
		fmt.Fprintf(deps.stdout, "%s verified at %s (latest release %s; no downgrade)\n", spec.Binary, installedVersion, release.Version)
		return nil
	}
	home, err := deps.platform.homeDir()
	if err != nil {
		return err
	}
	harness, err := loadHarness(cliutil.ResolveRepoRoot(), spec.ResourceName, harnessLoadDeps{loadManifest: deps.platform.loadManifest, homeDir: deps.platform.homeDir, goos: deps.platform.goos, goarch: deps.platform.goarch})
	if err != nil {
		return err
	}
	harness.Home, harness.BinDir = home, spec.BinDir
	if err := harness.installFromManifest(ctx, harnessDeps{platform: deps.platform, version: deps.installed, lookPath: deps.lookPath}); err != nil {
		return err
	}
	fmt.Fprintf(deps.stdout, "%s updated %s -> %s\n", spec.Binary, orUnknownVersion(installedVersion), release.Version)
	return nil
}

// npmPackageName returns the npm package declared by an acquisition contract.
func npmPackageName(acquisition *binaryfetch.Acquisition) string {
	if acquisition == nil {
		return ""
	}
	for _, target := range acquisition.Targets {
		if name := strings.TrimSpace(target.Package); name != "" {
			return name
		}
	}
	return ""
}

func orUnknownVersion(version string) string {
	if strings.TrimSpace(version) == "" {
		return "(none)"
	}
	return version
}

// Success means the binary launch/discovery selects is the verified installation,
// not merely that npm wrote bytes to an otherwise shadowed directory.
func verifySelectedUpdate(ctx context.Context, binary, target, version string, deps updateDeps) error {
	selected, err := deps.lookPath(binary)
	if err != nil {
		return fmt.Errorf("verify selected %s: %w", binary, err)
	}
	if filepath.Clean(selected) != filepath.Clean(target) {
		selectedInfo, selectedErr := deps.stat(selected)
		targetInfo, targetErr := deps.stat(target)
		if selectedErr != nil || targetErr != nil || !os.SameFile(selectedInfo, targetInfo) {
			return fmt.Errorf("updated %s at %s but PATH selects %s; resolve PATH shadowing before launch", binary, target, selected)
		}
	}
	raw, err := deps.installed(ctx, selected)
	if err != nil {
		return fmt.Errorf("verify updated %s at %s: %w", binary, selected, err)
	}
	if got := upstreamcheck.ExtractSemver(raw); got != version {
		return fmt.Errorf("verify updated %s: selected %s reports %q, expected %s", binary, selected, raw, version)
	}
	return nil
}
