package rendererprofile

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	ffmpegvideo "github.com/vrooli/vrooli/internal/tools/ffmpeg-video"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	ManagedBlenderSelector = "blender"
	ManagedFFmpegSelector  = "ffmpeg-video"
)

// ManagedTree is a finite, owner-verified source for an optional development
// release package. The source path is derived from the invoking user's install
// ledger and is valid only while WithManagedTree's callback runs.
type ManagedTree struct {
	Selector, ToolID, ProfileID, OS, Architecture, TreeSHA256 string
	ArtifactName, Role, Provenance, SourceRoot                string
}

// WithManagedTree keeps the managed-tool install lock while the caller copies
// the verified tree, preventing replacement between verification and staging.
func WithManagedTree(ctx context.Context, selector string, use func(ManagedTree) error) error {
	if ctx == nil || use == nil {
		return errors.New("managed renderer source requires context and callback")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return errors.New("managed renderer release sources are supported only on Linux amd64")
	}
	var toolID, profileID, tree, artifactName, role string
	switch selector {
	case ManagedBlenderSelector:
		a := blenderrenderer.PinnedArtifact()
		toolID, profileID, tree, artifactName, role = blenderrenderer.ToolName, a.ProfileID, a.TreeSHA256, "tool_blender-renderer_linux_amd64", "renderer-profile:blender"
	case ManagedFFmpegSelector:
		a := ffmpegvideo.PinnedArtifact()
		toolID, profileID, tree, artifactName, role = ManagedFFmpegSelector, a.ProfileID, ffmpegvideo.TreeSHA256, "tool_ffmpeg-video_linux_amd64", "renderer-profile:ffmpeg-video"
	default:
		return fmt.Errorf("unsupported managed renderer selector %q", selector)
	}
	home, err := config.HomeDir()
	if err != nil {
		return fmt.Errorf("resolve managed renderer owner home: %w", err)
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return fmt.Errorf("resolve managed renderer root: %w", err)
	}
	release, err := artifactlock.AcquireContext(ctx, toolID)
	if err != nil {
		return fmt.Errorf("lock managed renderer %q: %w", selector, err)
	}
	defer release()
	source := filepath.Join(root, "opt", toolID)
	record, err := cliinstall.LoadInstallRecord(home)
	if err != nil {
		return fmt.Errorf("read managed renderer ownership: %w", err)
	}
	owned := false
	for _, entry := range record.Entries {
		if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == filepath.Clean(source) {
			owned = true
			break
		}
	}
	if !owned {
		return fmt.Errorf("managed renderer %q is not recorded as runtime-owned", selector)
	}
	if selector == ManagedBlenderSelector {
		observation, err := blenderrenderer.Resolve(ctx, source, true, blenderrenderer.PinnedArtifact(), runtime.GOOS, runtime.GOARCH, nil)
		if err != nil {
			return err
		}
		if observation.State != blenderrenderer.StatePresent {
			return fmt.Errorf("managed Blender source is %s: %s", observation.State, observation.Detail)
		}
	} else {
		profile, err := ffmpegvideo.ResolveBundleProfile(ctx, source, true)
		if err != nil {
			return err
		}
		if profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT {
			return fmt.Errorf("managed FFmpeg source is not verified: %s", profile.GetDetail())
		}
	}
	return use(ManagedTree{Selector: selector, ToolID: toolID, ProfileID: profileID, OS: "linux", Architecture: "amd64", TreeSHA256: tree, ArtifactName: artifactName, Role: role, Provenance: "verified-managed-tree; development-local; non-promotable", SourceRoot: source})
}
