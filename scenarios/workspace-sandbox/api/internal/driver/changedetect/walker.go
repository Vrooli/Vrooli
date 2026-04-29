// Package changedetect owns the single shared walker that produces a
// list of FileChange records for a sandbox. Driver-specific differences
// (overlayfs whiteout semantics vs. double-walk content compare) live
// behind the Strategy interface; everything else — directory traversal,
// path normalisation, context cancellation, error wrapping, stable
// ordering — is one implementation.
//
// I-CHANGE-1: GetChangedFiles is deterministic for a given filesystem
// state. The walker sorts results by FilePath using sort.SliceStable so
// both strategies produce a stable ordering — pinned by the contract
// test in walker_contract_test.go.
package changedetect

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// WalkOpts configures a walk. Lower is the read-only / original tree;
// Upper is the writable / workspace tree; SandboxID is used for stable
// file IDs in the resulting FileChange records.
type WalkOpts struct {
	Lower     string
	Upper     string
	SandboxID uuid.UUID
}

// Strategy plugs driver-specific change-detection semantics into the
// shared walker. Each strategy method is called at a well-defined point
// in the walk so the strategy can stay declarative.
type Strategy interface {
	// ShouldSkip reports whether a path encountered during the upper
	// walk should be ignored entirely. Called with the upper-relative
	// path (e.g. ".overlay/work" or ".git/HEAD"). Strategies that want
	// to forward filtering decisions to the walker can return true here
	// to drop subtrees too — see SkipDir below.
	ShouldSkip(rel string) bool

	// SkipDir reports whether, having decided to skip a directory at
	// rel, the entire subtree should also be skipped. Returning true is
	// equivalent to filepath.SkipDir.
	SkipDir(rel string) bool

	// ClassifyUpper inspects a single upper-side entry the walker has
	// not skipped and produces a FileChange (or nil to drop). The
	// strategy is responsible for deciding Added vs. Modified vs.
	// Deleted (the latter being inline whiteouts for overlayfs).
	ClassifyUpper(opts WalkOpts, rel string, abs string, info fs.FileInfo, now time.Time) (*types.FileChange, error)

	// DetectDeletions runs after the upper walk completes. seen
	// contains every upper-relative path that ClassifyUpper returned a
	// non-nil FileChange for. Strategies that derive deletions from
	// absence in upper (copy driver double-walk) walk the lower tree
	// here. Strategies that detect deletions inline via whiteouts
	// (overlayfs) return nil.
	DetectDeletions(opts WalkOpts, seen map[string]bool, now time.Time) ([]*types.FileChange, error)
}

// Walk executes the change-detection contract: walk Upper, classify
// each entry via the strategy, then run the strategy's deletion pass
// against Lower. Results are sorted by FilePath for stable ordering.
// ctx cancellation aborts mid-walk.
func Walk(ctx context.Context, opts WalkOpts, strategy Strategy, now time.Time) ([]*types.FileChange, error) {
	if opts.Upper == "" {
		return nil, fmt.Errorf("changedetect.Walk: Upper is empty")
	}
	if strategy == nil {
		return nil, fmt.Errorf("changedetect.Walk: Strategy is nil")
	}

	var changes []*types.FileChange
	seen := make(map[string]bool)

	walkErr := filepath.Walk(opts.Upper, func(path string, info fs.FileInfo, err error) error {
		if cancelErr := ctx.Err(); cancelErr != nil {
			return cancelErr
		}
		if err != nil {
			return err
		}
		if path == opts.Upper {
			return nil
		}
		rel, relErr := filepath.Rel(opts.Upper, path)
		if relErr != nil {
			return relErr
		}
		if strategy.ShouldSkip(rel) {
			if info != nil && info.IsDir() && strategy.SkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		// Record every visited upper entry (file or directory) so the
		// deletion pass can distinguish "removed from upper" from
		// "present but unchanged". Without this, an identical file on
		// both sides would be missing from `seen` and surface as
		// Deleted.
		if !info.IsDir() {
			seen[rel] = true
		}
		change, classifyErr := strategy.ClassifyUpper(opts, rel, path, info, now)
		if classifyErr != nil {
			return classifyErr
		}
		if change != nil {
			changes = append(changes, change)
			seen[change.FilePath] = true
		}
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
			return nil, walkErr
		}
		return nil, fmt.Errorf("walk upper directory %q: %w", opts.Upper, walkErr)
	}

	deletions, delErr := strategy.DetectDeletions(opts, seen, now)
	if delErr != nil {
		return nil, fmt.Errorf("detect deletions: %w", delErr)
	}
	changes = append(changes, deletions...)

	sort.SliceStable(changes, func(i, j int) bool {
		return changes[i].FilePath < changes[j].FilePath
	})

	return changes, nil
}
