package hostapp

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	rendererprofile "github.com/vrooli/vrooli/internal/tools/renderer-profile"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// BlenderProfile observes only the exact managed Blender installation. It
// does not install tools, consult PATH, or accept an executable from callers.
func (Service) BlenderProfile(ctx context.Context) (*cliv1.BlenderToolProfile, error) {
	if ctx == nil {
		return nil, errors.New("Blender profile context is nil")
	}
	home, err := config.HomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve Blender profile owner home: %w", err)
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return nil, fmt.Errorf("resolve Blender profile runtime home: %w", err)
	}
	optDir := filepath.Join(root, "opt", blenderrenderer.ToolName)
	record, err := cliinstall.LoadInstallRecord(home)
	if err != nil {
		return nil, fmt.Errorf("read managed Blender ownership: %w", err)
	}
	owned := false
	for _, entry := range record.Entries {
		if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == filepath.Clean(optDir) {
			owned = true
			break
		}
	}
	artifact := blenderrenderer.PinnedArtifact()
	goos, arch := blenderrenderer.RuntimePlatform()
	observation, err := blenderrenderer.Resolve(ctx, optDir, owned, artifact, goos, arch, nil)
	if err != nil {
		return nil, err
	}
	return rendererprofile.Blender(observation), nil
}

// RepairBlenderPythonCache quarantines only verified __pycache__ directories
// from the exact ledger-owned Blender archive. It shares the install lock with
// managed tool acquisition so the archive cannot be replaced concurrently.
func (Service) RepairBlenderPythonCache(ctx context.Context, dryRun bool) (blenderrenderer.PythonCacheRepairReceipt, error) {
	if ctx == nil {
		return blenderrenderer.PythonCacheRepairReceipt{}, errors.New("Blender cache repair context is nil")
	}
	goos, arch := blenderrenderer.RuntimePlatform()
	if goos != "linux" || arch != "amd64" {
		return blenderrenderer.PythonCacheRepairReceipt{}, errors.New("Blender Python cache repair is supported only for the pinned Linux amd64 profile")
	}
	home, err := config.HomeDir()
	if err != nil {
		return blenderrenderer.PythonCacheRepairReceipt{}, fmt.Errorf("resolve Blender cache repair home: %w", err)
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return blenderrenderer.PythonCacheRepairReceipt{}, fmt.Errorf("resolve Blender cache repair runtime root: %w", err)
	}
	optDir := filepath.Join(root, "opt", blenderrenderer.ToolName)
	stateRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return blenderrenderer.PythonCacheRepairReceipt{}, fmt.Errorf("resolve Blender cache quarantine root: %w", err)
	}
	quarantineRoot := filepath.Join(stateRoot, "tool-repair", blenderrenderer.ToolName)
	owned, err := exactBlenderDirectoryOwnership(home, optDir)
	if err != nil {
		return blenderrenderer.PythonCacheRepairReceipt{}, err
	}
	if !owned {
		return blenderrenderer.PythonCacheRepairReceipt{}, errors.New("refusing Blender cache repair: exact managed directory is not recorded as a runtime-owned archive")
	}
	release, err := artifactlock.AcquireContext(ctx, blenderrenderer.ToolName)
	if err != nil {
		return blenderrenderer.PythonCacheRepairReceipt{}, fmt.Errorf("lock managed Blender installation: %w", err)
	}
	defer release()
	owned, err = exactBlenderDirectoryOwnership(home, optDir)
	if err != nil {
		return blenderrenderer.PythonCacheRepairReceipt{}, err
	}
	if !owned {
		return blenderrenderer.PythonCacheRepairReceipt{}, errors.New("managed Blender directory ownership changed while waiting for install lock")
	}
	artifact := blenderrenderer.PinnedArtifact()
	receipt, err := blenderrenderer.RepairPythonCache(ctx, optDir, quarantineRoot, artifact, dryRun)
	if err != nil || dryRun || !receipt.Repaired {
		return receipt, err
	}
	profile, err := (Service{}).BlenderProfile(ctx)
	if err != nil {
		return receipt, fmt.Errorf("verify Blender profile after cache repair: %w", err)
	}
	if profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT {
		return receipt, fmt.Errorf("cache repair restored the tree digest but Blender profile verification remains %s", profile.GetState())
	}
	return receipt, nil
}

func exactBlenderDirectoryOwnership(home, optDir string) (bool, error) {
	record, err := cliinstall.LoadInstallRecord(home)
	if err != nil {
		return false, fmt.Errorf("read managed Blender ownership: %w", err)
	}
	matches := 0
	for _, entry := range record.Entries {
		if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == filepath.Clean(optDir) && filepath.Clean(entry.Prefix) == filepath.Clean(optDir) {
			matches++
		}
	}
	return matches == 1, nil
}
