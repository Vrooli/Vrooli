// Package nodejsruntime installs and selects the pinned Node.js runtime used
// by control-plane tooling, so the host PATH and system Node version never
// decide what a build or test actually runs.
package nodejsruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/repo-contract-go"
	_ "github.com/vrooli/vrooli/internal/artifactcodecs" // register tar.xz decoding for official Node archives
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/tools"
)

var exactVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

type versionCatalog struct {
	Versions map[string]binaryfetch.Acquisition `json:"versions"`
}

type toolManifest struct {
	Version     string                   `json:"version"`
	Acquisition *binaryfetch.Acquisition `json:"acquisition"`
}

// RuntimeManager resolves checksum-pinned Node releases into one user-owned
// cache shared by all scenarios. A scenario references a version; it never
// owns a duplicate Node installation.
type RuntimeManager struct {
	Home            string
	OS              string
	Arch            string
	FetchDir        func(context.Context, binaryfetch.Target, string, binaryfetch.ProgressFunc) (string, error)
	AcquireLock     func(context.Context, string) (func(), error)
	RecordArtifacts func(string, ...cliinstall.InstallEntry) error
}

// Ensure returns the executable directory for an exact, supported release. The
// default host release reuses the normal host-tool install when present; other
// releases live under ~/.vrooli/opt/node-versions/<version>.
func (m RuntimeManager) Ensure(ctx context.Context, version string) (string, error) {
	version = strings.TrimSpace(version)
	if !exactVersion.MatchString(version) {
		return "", fmt.Errorf("Node runtime version %q must be an exact x.y.z release", version)
	}
	if m.Home == "" {
		return "", fmt.Errorf("Node runtime requires the invoking user's home directory")
	}
	if m.OS == "" {
		m.OS = runtime.GOOS
	}
	if m.Arch == "" {
		m.Arch = runtime.GOARCH
	}
	if m.FetchDir == nil {
		m.FetchDir = binaryfetch.FetchDir
	}
	if m.AcquireLock == nil {
		m.AcquireLock = artifactlock.AcquireContext
	}
	if m.RecordArtifacts == nil {
		m.RecordArtifacts = cliinstall.RecordToolArtifacts
	}

	acquisition, isDefault, err := acquisitionFor(version)
	if err != nil {
		return "", err
	}
	target, ok := hostreqkit.TargetFor(&acquisition, m.OS, m.Arch)
	if !ok || target.Layout != "dir" || target.BinPath == "" {
		return "", fmt.Errorf("Node %s has no verified archive for %s/%s", version, m.OS, m.Arch)
	}
	if err := acquisition.Validate(); err != nil {
		return "", fmt.Errorf("Node %s release catalog is invalid: %w", version, err)
	}

	runtimeRoot, err := repocontract.VrooliUserRoot(m.Home)
	if err != nil {
		return "", fmt.Errorf("resolve user-owned Vrooli runtime root: %w", err)
	}
	installRoot := filepath.Join(runtimeRoot, "opt", "node-versions", version)
	if isDefault {
		installRoot = filepath.Join(runtimeRoot, "opt", "node")
	}
	entryPath := filepath.Join(installRoot, filepath.FromSlash(target.BinPath))
	if err := verifyNode(ctx, entryPath, version); err == nil {
		return filepath.Dir(entryPath), nil
	}

	// Share the generic host-tool lock as the default release is installed at
	// the same opt/node root by `vrooli host install node`.
	unlock, err := m.AcquireLock(ctx, "node")
	if err != nil {
		return "", fmt.Errorf("lock Node %s runtime installation: %w", version, err)
	}
	defer unlock()
	if err := verifyNode(ctx, entryPath, version); err == nil {
		return filepath.Dir(entryPath), nil
	}

	spec := binaryfetch.Target{
		Name:    "node",
		URL:     target.URL,
		SHA256:  target.SHA256,
		Archive: target.Archive,
		Layout:  target.Layout,
		BinPath: target.BinPath,
		Mode:    target.Mode,
	}
	entryPath, err = m.FetchDir(ctx, spec, installRoot, nil)
	if err != nil {
		return "", fmt.Errorf("install checksum-pinned Node %s: %w", version, err)
	}
	if err := verifyNode(ctx, entryPath, version); err != nil {
		return "", fmt.Errorf("verify installed Node %s: %w", version, err)
	}
	if err := m.recordInstallArtifact(installRoot); err != nil {
		_ = os.RemoveAll(installRoot)
		return "", fmt.Errorf("record installed Node %s runtime: %w", version, err)
	}
	return filepath.Dir(entryPath), nil
}

func (m RuntimeManager) recordInstallArtifact(path string) error {
	record, err := cliinstall.LoadInstallRecord(m.Home)
	if err != nil {
		return err
	}
	for _, entry := range record.Entries {
		if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == filepath.Clean(path) {
			return nil
		}
	}
	return m.RecordArtifacts(m.Home, cliinstall.InstallEntry{
		Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory,
		Path: path, Prefix: path,
	})
}

func acquisitionFor(version string) (binaryfetch.Acquisition, bool, error) {
	var manifest toolManifest
	data, err := tools.Manifests.ReadFile("node/tool.json")
	if err != nil {
		return binaryfetch.Acquisition{}, false, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return binaryfetch.Acquisition{}, false, fmt.Errorf("parse default Node tool manifest: %w", err)
	}
	if version == manifest.Version {
		if manifest.Acquisition == nil {
			return binaryfetch.Acquisition{}, false, fmt.Errorf("default Node %s has no acquisition contract", version)
		}
		return *manifest.Acquisition, true, nil
	}

	data, err = tools.Manifests.ReadFile("node/versions.json")
	if err != nil {
		return binaryfetch.Acquisition{}, false, err
	}
	var catalog versionCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return binaryfetch.Acquisition{}, false, fmt.Errorf("parse Node runtime version catalog: %w", err)
	}
	acquisition, ok := catalog.Versions[version]
	if !ok {
		return binaryfetch.Acquisition{}, false, fmt.Errorf("Node %s is not in Vrooli's checksum-pinned runtime catalog", version)
	}
	return acquisition, false, nil
}

func verifyNode(ctx context.Context, path, expectedVersion string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		if err == nil {
			err = fmt.Errorf("path is a directory")
		}
		return fmt.Errorf("Node executable %s is unavailable: %w", path, err)
	}
	output, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return fmt.Errorf("run Node executable %s: %w", path, err)
	}
	got := strings.TrimPrefix(strings.TrimSpace(string(output)), "v")
	if got != expectedVersion {
		return fmt.Errorf("Node executable %s reports %q, want %s", path, got, expectedVersion)
	}
	return nil
}
