package agentinstall

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tuning"
)

// UpdateCommand exposes the latest-release updater to a resource CLI.
func UpdateCommand(spec Spec) cliapp.Command {
	return cliapp.Command{
		Name:        "update",
		Description: "Update the upstream CLI to the latest published release",
		Run: func(_ []string) error {
			return Update(context.Background(), spec)
		},
	}
}

// updateDeps carries the seams that let Update run without npm or the network.
type updateDeps struct {
	resolveLatest func(ctx context.Context, kind upstreamcheck.SourceKind, id string) (string, error)
	installed     func(ctx context.Context, path string) (string, error)
	install       func(ctx context.Context, prefix, binDir, binary, packageRef string) error
	lookPath      func(string) (string, error)
	stat          func(string) (os.FileInfo, error)
	stdout        io.Writer
}

func defaultUpdateDeps() updateDeps {
	return updateDeps{
		resolveLatest: func(ctx context.Context, kind upstreamcheck.SourceKind, id string) (string, error) {
			handlers := upstreamcheck.Default(upstreamcheck.Config{SourceKind: kind, SourceID: id})
			return handlers.LatestFetcher(ctx, kind, id)
		},
		installed: InstalledVersion,
		install:   installNPM,
		lookPath:  exec.LookPath,
		stat:      os.Stat,
		stdout:    os.Stdout,
	}
}

// Update installs the latest published release of an npm-acquired CLI into the
// user-owned prefix.
//
// The manifest acquisition pins a version for reproducible provisioning;
// `update` deliberately bypasses that pin so an operator can move a host tool
// forward without editing the committed manifest. npm still authenticates the
// downloaded bytes against the registry integrity for the resolved version.
func Update(ctx context.Context, spec Spec) error {
	return update(ctx, spec, defaultUpdateDeps())
}

func update(ctx context.Context, spec Spec, deps updateDeps) error {
	if strings.TrimSpace(spec.Binary) == "" || strings.TrimSpace(spec.BinDir) == "" {
		return fmt.Errorf("binary and bin directory are required")
	}
	acquisition := spec.Acquisition
	if acquisition == nil {
		loaded, err := acquisitionFromSourceManifest(spec.ResourceName)
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
	if runtime.GOOS == string(hostreqspec.PlatformWindows) {
		target += ".exe"
	}
	installedVersion := ""
	if raw, probeErr := deps.installed(ctx, target); probeErr == nil {
		installedVersion = upstreamcheck.ExtractSemver(raw)
	}
	if installedVersion != "" && upstreamcheck.Compare(installedVersion, latestVersion) == upstreamcheck.StatusUpToDate {
		fmt.Fprintf(deps.stdout, "%s is already up to date (%s)\n", spec.Binary, installedVersion)
		return nil
	}

	if err := os.MkdirAll(spec.BinDir, tuning.PermDir); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}
	if spec.DataDir != "" {
		if err := os.MkdirAll(spec.DataDir, tuning.PermPrivateDir); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}
	}
	if blocker, err := BlockingSystemInstall(spec.Binary, spec.BinDir, deps.lookPath, deps.stat); err != nil {
		return err
	} else if blocker != "" {
		return fmt.Errorf("refusing to overwrite root-owned %s at %s", spec.Binary, blocker)
	}

	if err := deps.install(ctx, filepath.Dir(spec.BinDir), spec.BinDir, spec.Binary, packageName+"@"+latestVersion); err != nil {
		return err
	}
	fmt.Fprintf(deps.stdout, "%s updated %s -> %s\n", spec.Binary, orUnknownVersion(installedVersion), latestVersion)
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
