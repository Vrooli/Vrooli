// Package toolprofiles resolves the finite renderer packages declared by a
// desktop bundle. It never searches PATH or accepts caller-supplied paths.
package toolprofiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	ffmpegvideo "github.com/vrooli/vrooli/internal/tools/ffmpeg-video"
	rendererprofile "github.com/vrooli/vrooli/internal/tools/renderer-profile"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"google.golang.org/protobuf/proto"
)

const (
	BlenderToolID = "blender"
	FFmpegToolID  = "ffmpeg-video"
)

// PinnedDeclaration exposes the finite package identity to release builders.
// It is derived from the same owners used by Resolve; callers cannot provide
// their own profile hashes or install paths.
func PinnedDeclaration(selector string) (manifest.ToolProfilePackage, string, bool) {
	switch selector {
	case BlenderToolID:
		artifact := blenderrenderer.PinnedArtifact()
		return manifest.ToolProfilePackage{ToolID: BlenderToolID, ProfileID: artifact.ProfileID, OS: "linux", Architecture: "amd64", TreeSHA256: artifact.TreeSHA256}, blenderrenderer.ToolName, true
	case FFmpegToolID:
		artifact := ffmpegvideo.PinnedArtifact()
		return manifest.ToolProfilePackage{ToolID: FFmpegToolID, ProfileID: artifact.ProfileID, OS: "linux", Architecture: "amd64", TreeSHA256: ffmpegvideo.TreeSHA256}, FFmpegToolID, true
	default:
		return manifest.ToolProfilePackage{}, "", false
	}
}

// Resolve returns the typed owner observation for one supported selector.
func Resolve(ctx context.Context, bundleRoot, selector string, declarations []manifest.ToolProfilePackage) (proto.Message, error) {
	if ctx == nil {
		return nil, errors.New("tool profile context is nil")
	}
	if !filepath.IsAbs(bundleRoot) {
		return nil, errors.New("bundle root must be absolute")
	}
	bundleRoot, err := canonicalRoot(bundleRoot)
	if err != nil {
		return nil, err
	}
	declaration, declared, err := findDeclaration(selector, declarations)
	if err != nil {
		return nil, err
	}
	switch selector {
	case BlenderToolID:
		artifact := blenderrenderer.PinnedArtifact()
		if declared {
			if err := checkDeclaration(declaration, artifact.ProfileID, artifact.TreeSHA256); err != nil {
				return nil, err
			}
		}
		root, err := selectedToolRoot(bundleRoot, BlenderToolID)
		if err != nil {
			return nil, err
		}
		observation, err := blenderrenderer.Resolve(ctx, root, declared, artifact, runtime.GOOS, runtime.GOARCH, nil)
		if err != nil {
			return nil, err
		}
		return rendererprofile.Blender(observation), nil
	case FFmpegToolID:
		artifact := ffmpegvideo.PinnedArtifact()
		if declared {
			if err := checkDeclaration(declaration, artifact.ProfileID, ffmpegvideo.TreeSHA256); err != nil {
				return nil, err
			}
		}
		root, err := selectedToolRoot(bundleRoot, FFmpegToolID)
		if err != nil {
			return nil, err
		}
		return ffmpegvideo.ResolveBundleProfile(ctx, root, declared)
	default:
		return nil, fmt.Errorf("unsupported renderer profile %q; supported profiles are blender and ffmpeg-video", selector)
	}
}

// canonicalRoot resolves existing symlink ancestors while preserving a
// missing bundle tail, which is a valid way to report an absent optional tool.
func canonicalRoot(root string) (string, error) {
	root = filepath.Clean(root)
	var missing []string
	candidate := root
	for {
		_, err := os.Lstat(candidate)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect bundle root: %w", err)
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", errors.New("bundle root has no existing ancestor")
		}
		missing = append(missing, filepath.Base(candidate))
		candidate = parent
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve bundle root: %w", err)
	}
	for i := len(missing) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, missing[i])
	}
	return filepath.Clean(resolved), nil
}

// selectedToolRoot rejects indirection in the bundle-controlled selector
// path while leaving payload-internal symlinks to the pinned tree owner.
func selectedToolRoot(bundleRoot, selector string) (string, error) {
	rootInfo, err := os.Lstat(bundleRoot)
	if err == nil && (!rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0) {
		return "", errors.New("bundle root is not a directory")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect bundle root: %w", err)
	}
	tools := filepath.Join(bundleRoot, "tools")
	toolsInfo, err := os.Lstat(tools)
	if err == nil && (!toolsInfo.IsDir() || toolsInfo.Mode()&os.ModeSymlink != 0) {
		return "", errors.New("bundle tools path must be a real directory")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect bundle tools path: %w", err)
	}
	selected := filepath.Join(tools, selector)
	selectedInfo, err := os.Lstat(selected)
	if err == nil && (!selectedInfo.IsDir() || selectedInfo.Mode()&os.ModeSymlink != 0) {
		return "", errors.New("selected renderer package must be a real directory")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect selected renderer package: %w", err)
	}
	return selected, nil
}

func findDeclaration(selector string, declarations []manifest.ToolProfilePackage) (manifest.ToolProfilePackage, bool, error) {
	var found manifest.ToolProfilePackage
	count := 0
	for _, item := range declarations {
		if item.ToolID == selector {
			found, count = item, count+1
		}
	}
	if count > 1 {
		return manifest.ToolProfilePackage{}, false, fmt.Errorf("bundle repeats renderer profile %q", selector)
	}
	if !counted(selector) {
		return manifest.ToolProfilePackage{}, false, fmt.Errorf("unsupported renderer profile %q; supported profiles are blender and ffmpeg-video", selector)
	}
	if count == 0 {
		return manifest.ToolProfilePackage{}, false, nil
	}
	return found, true, nil
}

func counted(selector string) bool { return selector == BlenderToolID || selector == FFmpegToolID }

func checkDeclaration(declaration manifest.ToolProfilePackage, profileID, treeSHA string) error {
	if declaration.ProfileID != profileID {
		return fmt.Errorf("bundle renderer profile %q declares unexpected profile ID", declaration.ToolID)
	}
	if declaration.TreeSHA256 != treeSHA {
		return fmt.Errorf("bundle renderer profile %q declares an unrecognized tree digest", declaration.ToolID)
	}
	if declaration.OS != "linux" || declaration.Architecture != "amd64" {
		return fmt.Errorf("bundle renderer profile %q declares an unrecognized platform", declaration.ToolID)
	}
	return nil
}
