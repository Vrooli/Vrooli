package changedetect

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// lowerMissing reports whether a stat error against the lower layer
// indicates the path simply isn't there. ENOENT and ENOTDIR both qualify
// (ENOTDIR appears when an ancestor directory was replaced by a file).
func lowerMissing(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	return errors.Is(err, syscall.ENOTDIR)
}

// CopyStrategy implements change detection for the copy driver. The
// upper tree is a full copy of lower; the strategy classifies upper
// entries by comparing their content/metadata against lower, and walks
// lower in DetectDeletions to find files that were removed from upper.
type CopyStrategy struct {
	// FileIDFn produces the stable per-(sandbox, path) UUID. Injected
	// to keep this package free of cyclic imports against driver/.
	FileIDFn func(sandboxID uuid.UUID, filePath string) uuid.UUID
}

// ShouldSkip drops hidden entries and overlayfs whiteout markers. The
// copy driver doesn't itself produce whiteouts, but the helper handles
// them defensively in case a legacy upper carried any.
func (s *CopyStrategy) ShouldSkip(rel string) bool {
	if strings.HasPrefix(rel, ".") {
		return true
	}
	if isOverlayMarker(rel) {
		return true
	}
	return false
}

// SkipDir prunes the entire .git/.wh subtree from the walk.
func (s *CopyStrategy) SkipDir(rel string) bool {
	return strings.HasPrefix(rel, ".")
}

// ClassifyUpper compares an upper entry to its counterpart in lower and
// returns Added or Modified. Directories never surface as their own
// FileChange row; their contained files do.
func (s *CopyStrategy) ClassifyUpper(opts WalkOpts, rel string, abs string, info fs.FileInfo, now time.Time) (*types.FileChange, error) {
	if opts.Lower == "" {
		return nil, fmt.Errorf("copy strategy requires Lower")
	}
	if info.IsDir() {
		return nil, nil
	}
	originalPath := filepath.Join(opts.Lower, rel)
	originalInfo, originalErr := os.Stat(originalPath)

	var changeType types.ChangeType
	switch {
	case lowerMissing(originalErr):
		changeType = types.ChangeTypeAdded
	case originalErr != nil:
		changeType = types.ChangeTypeModified
	case originalInfo.IsDir():
		// upper is a file but lower is a directory at this path.
		// Treat as a replacement — Modified.
		changeType = types.ChangeTypeModified
	case filesAreDifferent(originalPath, abs, originalInfo, info):
		changeType = types.ChangeTypeModified
	default:
		return nil, nil
	}

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

// DetectDeletions walks the lower tree to find files that no longer
// exist in upper.
func (s *CopyStrategy) DetectDeletions(opts WalkOpts, seen map[string]bool, now time.Time) ([]*types.FileChange, error) {
	if opts.Lower == "" {
		return nil, nil
	}
	if _, statErr := os.Stat(opts.Lower); os.IsNotExist(statErr) {
		return nil, nil
	}
	var deletions []*types.FileChange
	walkErr := filepath.Walk(opts.Lower, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == opts.Lower {
			return nil
		}
		rel, relErr := filepath.Rel(opts.Lower, path)
		if relErr != nil {
			return relErr
		}
		if strings.HasPrefix(rel, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if isOverlayMarker(rel) {
			return nil
		}
		if seen[rel] {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		deletions = append(deletions, &types.FileChange{
			ID:             s.FileIDFn(opts.SandboxID, rel),
			SandboxID:      opts.SandboxID,
			FilePath:       rel,
			ChangeType:     types.ChangeTypeDeleted,
			FileSize:       info.Size(),
			FileMode:       int(info.Mode()),
			DetectedAt:     now,
			ApprovalStatus: types.ApprovalPending,
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk lower directory %q: %w", opts.Lower, walkErr)
	}
	return deletions, nil
}

// filesAreDifferent decides whether two regular files have different
// content. Cheap checks first (size, mode), then byte compare for small
// files, then mtime as a heuristic for large ones.
func filesAreDifferent(path1, path2 string, info1, info2 fs.FileInfo) bool {
	if info1.Size() != info2.Size() {
		return true
	}
	if info1.Mode() != info2.Mode() {
		return true
	}
	if info1.Size() < 64*1024 {
		content1, err1 := os.ReadFile(path1)
		content2, err2 := os.ReadFile(path2)
		if err1 != nil || err2 != nil {
			return true
		}
		return !bytesEqual(content1, content2)
	}
	return info1.ModTime() != info2.ModTime()
}

// bytesEqual is a stand-in for bytes.Equal that avoids pulling the
// "bytes" import for one helper.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isOverlayMarker reports whether rel names an overlayfs whiteout
// (`.wh.X` or `.wh..opq`). Defensive — copy driver upper layers should
// not contain these but a misconfigured driver swap could leave one.
func isOverlayMarker(rel string) bool {
	baseName := filepath.Base(rel)
	if baseName == ".wh..opq" {
		return true
	}
	return strings.HasPrefix(baseName, ".wh.")
}
