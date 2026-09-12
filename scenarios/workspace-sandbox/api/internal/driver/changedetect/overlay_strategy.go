package changedetect

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// OverlayStrategy implements change detection for the kernel-overlay
// and fuse-overlayfs drivers. The walker walks only the upper layer;
// deletions are encoded inline as overlayfs whiteouts (character
// devices with rdev=0, or files prefixed `.wh.`).
type OverlayStrategy struct {
	// FileIDFn produces the stable per-(sandbox, path) UUID. Injected
	// to keep this package free of cyclic imports against driver/.
	FileIDFn func(sandboxID uuid.UUID, filePath string) uuid.UUID
}

// ShouldSkip filters only overlayfs-specific internals. Shared gitignore and
// .git policy belongs to the walker, so both drivers apply it identically.
func (s *OverlayStrategy) ShouldSkip(rel string) bool {
	if strings.HasPrefix(rel, ".overlay") {
		return true
	}
	if filepath.Base(rel) == ".wh..opq" {
		return true
	}
	return false
}

// SkipDir returns true for the .overlay subtree so the walker
// doesn't descend into them.
func (s *OverlayStrategy) SkipDir(rel string) bool {
	return strings.HasPrefix(rel, ".overlay")
}

// ClassifyUpper turns an upper-side entry into either a FileChange or
// nil. Directories don't produce changes (their contents do). Whiteout
// markers translate into Deleted entries against the lower layer.
func (s *OverlayStrategy) ClassifyUpper(opts WalkOpts, rel string, abs string, info fs.FileInfo, now time.Time) (*types.FileChange, error) {
	if info.IsDir() {
		return nil, nil
	}

	baseName := filepath.Base(rel)
	if strings.HasPrefix(baseName, ".wh.") {
		return s.classifyFilenameWhiteout(opts, rel, baseName, now), nil
	}

	if isCharDevWhiteout(abs) {
		return s.deletionChange(opts, rel, now), nil
	}

	changeType := s.detectChangeType(opts, rel, abs, info)
	return &types.FileChange{
		ID:             s.FileIDFn(opts.SandboxID, rel),
		SandboxID:      opts.SandboxID,
		FilePath:       rel,
		ChangeType:     changeType,
		FileSize:       info.Size(),
		FileMode:       int(info.Mode()),
		DetectedAt:     now,
		ApprovalStatus: types.ApprovalPending,
	}, nil
}

// DetectDeletions is a no-op: overlayfs encodes deletions inline as
// whiteouts inside the upper walk.
func (s *OverlayStrategy) DetectDeletions(opts WalkOpts, seen map[string]bool, now time.Time) ([]*types.FileChange, error) {
	return nil, nil
}

func (s *OverlayStrategy) classifyFilenameWhiteout(opts WalkOpts, rel, baseName string, now time.Time) *types.FileChange {
	targetName := strings.TrimPrefix(baseName, ".wh.")
	if targetName == "" || strings.HasPrefix(targetName, ".wh.") || targetName == ".wh..opq" {
		return nil
	}
	targetRel := targetName
	if dir := filepath.Dir(rel); dir != "." {
		targetRel = filepath.Join(dir, targetName)
	}
	return s.deletionChange(opts, targetRel, now)
}

// deletionChange synthesises a Deleted FileChange for an entry whose
// presence on the lower side determines the recorded size + mode. The
// upper-side whiteout marker contributes only the act of deletion.
func (s *OverlayStrategy) deletionChange(opts WalkOpts, rel string, now time.Time) *types.FileChange {
	var fileSize int64
	var fileMode int
	if lowerInfo, statErr := os.Stat(filepath.Join(opts.Lower, rel)); statErr == nil {
		fileSize = lowerInfo.Size()
		fileMode = int(lowerInfo.Mode())
	}
	return &types.FileChange{
		ID:             s.FileIDFn(opts.SandboxID, rel),
		SandboxID:      opts.SandboxID,
		FilePath:       rel,
		ChangeType:     types.ChangeTypeDeleted,
		FileSize:       fileSize,
		FileMode:       fileMode,
		DetectedAt:     now,
		ApprovalStatus: types.ApprovalPending,
	}
}

// detectChangeType decides Added vs. Modified for a non-whiteout upper
// entry by consulting the lower layer.
func (s *OverlayStrategy) detectChangeType(opts WalkOpts, rel, abs string, upperInfo fs.FileInfo) types.ChangeType {
	if opts.Lower == "" {
		return types.ChangeTypeAdded
	}
	lowerPath := filepath.Join(opts.Lower, rel)
	_, err := os.Stat(lowerPath)
	if lowerMissing(err) {
		return types.ChangeTypeAdded
	}
	if err != nil {
		return types.ChangeTypeModified
	}
	if isCharDevWhiteout(abs) {
		return types.ChangeTypeDeleted
	}
	return types.ChangeTypeModified
}
