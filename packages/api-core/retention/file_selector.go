package retention

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileEntry is the engine-neutral fact set needed to select one file. A
// caller adapts its privileged or test filesystem to FileWalker; retention
// owns the filtering and ordering policy.
type FileEntry struct {
	Path    string
	Size    int64
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
}

// FileWalker is deliberately read-only. Implementations may route the walk
// through a privileged host seam, but selection can never mutate the target.
type FileWalker interface {
	Walk(context.Context, string, func(FileEntry) error) error
}

// FileSelectionConfig controls selection beneath one root.
type FileSelectionConfig struct {
	Root     string
	Now      time.Time
	MinAge   time.Duration
	MaxBytes int64
	// AllowSingleOvershoot permits the oldest eligible file to exceed MaxBytes
	// when no candidate has been selected yet. Cap enforcement uses this to
	// make progress when one file is larger than the remaining excess.
	AllowSingleOvershoot bool
	ProtectedRoots       []string
	ProtectedGlobs       []string
	IsActive             func(string) bool
}

// SelectFiles walks one root and returns eligible files oldest-first. It does
// not remove anything; callers must pass the candidates to a deletion engine
// that re-stats and re-checks containment at the mutation boundary.
func SelectFiles(ctx context.Context, walker FileWalker, cfg FileSelectionConfig) (Candidates, error) {
	if walker == nil {
		return nil, fmt.Errorf("file selector: walker is required")
	}
	if strings.TrimSpace(cfg.Root) == "" || !filepath.IsAbs(cfg.Root) {
		return nil, fmt.Errorf("file selector: Root must be an absolute path")
	}
	root := filepath.Clean(cfg.Root)
	protected, err := NormalizeProtectedRoots(cfg.ProtectedRoots)
	if err != nil {
		return nil, fmt.Errorf("file selector: %w", err)
	}
	if cfg.Now.IsZero() {
		cfg.Now = time.Now()
	}
	var selected Candidates
	err = walker.Walk(ctx, root, func(entry FileEntry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Clean(entry.Path)
		if path == root || entry.IsDir || !PathContains(root, path) {
			return nil
		}
		if ProtectedPathOverlap(path, protected) || protectedGlobMatch(path, cfg.ProtectedGlobs) {
			return nil
		}
		if cfg.IsActive != nil && cfg.IsActive(path) {
			return nil
		}
		if cfg.MinAge > 0 && cfg.Now.Sub(entry.ModTime) < cfg.MinAge {
			return nil
		}
		selected = append(selected, Candidate{Path: path, Bytes: entry.Size, ModTime: entry.ModTime})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].ModTime.Equal(selected[j].ModTime) {
			return selected[i].Path < selected[j].Path
		}
		return selected[i].ModTime.Before(selected[j].ModTime)
	})
	if cfg.MaxBytes > 0 {
		limited := make(Candidates, 0, len(selected))
		for _, candidate := range selected {
			if selectedBytes(limited)+candidate.Bytes > cfg.MaxBytes && !(cfg.AllowSingleOvershoot && len(limited) == 0) {
				continue
			}
			limited = append(limited, candidate)
		}
		selected = limited
	}
	return selected, nil
}

func selectedBytes(candidates Candidates) int64 {
	var total int64
	for _, candidate := range candidates {
		total += candidate.Bytes
	}
	return total
}

func protectedGlobMatch(path string, patterns []string) bool {
	path = filepath.Clean(path)
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}
