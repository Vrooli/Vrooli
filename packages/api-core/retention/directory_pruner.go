package retention

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DirectoryConfig configures the builtin pruner for a directory target.
type DirectoryConfig struct {
	// Path is the absolute directory the budget bounds. Required.
	Path string
	// Now supplies the current time. Defaults to time.Now.
	Now func() time.Time
	// Logger receives cycle detail. Defaults to slog.Default.
	Logger *slog.Logger
}

// DirectoryPruner enforces a budget over the top-level entries of one directory.
//
// It deletes whole top-level entries rather than walking into them to remove
// individual files. A half-deleted subtree is harder to reason about than a
// missing one: the caller can tell that a snapshot directory is gone, but cannot
// tell that one is intact.
type DirectoryPruner struct {
	cfg DirectoryConfig
}

// NewDirectoryPruner validates cfg and returns the pruner.
func NewDirectoryPruner(cfg DirectoryConfig) (*DirectoryPruner, error) {
	if strings.TrimSpace(cfg.Path) == "" {
		return nil, fmt.Errorf("directory pruner: Path is required")
	}
	if !filepath.IsAbs(cfg.Path) {
		return nil, fmt.Errorf("directory pruner: Path %q must be absolute; resolve it through api-core/storage first", cfg.Path)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &DirectoryPruner{cfg: cfg}, nil
}

// entry is one top-level directory member with the two facts pruning needs.
type entry struct {
	name    string
	modTime time.Time
	bytes   int64
}

// Measure reports the total size and count of the directory's top-level entries.
func (p *DirectoryPruner) Measure(ctx context.Context) (Usage, error) {
	entries, err := p.scan(ctx)
	if err != nil {
		return Usage{}, err
	}
	var usage Usage
	for _, e := range entries {
		usage.Bytes += e.bytes
		usage.Items++
	}
	return usage, nil
}

// scan lists top-level entries, oldest first, with their recursive sizes.
//
// A directory that does not exist yet measures as empty rather than failing: a
// component that has not written anything is trivially within its budget, and
// erroring would make an unused budget look like a broken one.
func (p *DirectoryPruner) scan(ctx context.Context) ([]entry, error) {
	dirEntries, err := os.ReadDir(p.cfg.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read directory %s: %w", p.cfg.Path, err)
	}

	out := make([]entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		info, err := de.Info()
		if err != nil {
			// An entry that vanished between listing and stat is already gone,
			// which is the outcome pruning wants anyway.
			if os.IsNotExist(err) {
				continue
			}
			return out, fmt.Errorf("stat %s: %w", filepath.Join(p.cfg.Path, de.Name()), err)
		}
		size, err := p.entryBytes(ctx, filepath.Join(p.cfg.Path, de.Name()), info)
		if err != nil {
			return out, err
		}
		out = append(out, entry{name: de.Name(), modTime: info.ModTime(), bytes: size})
	}

	sort.Slice(out, func(i, j int) bool {
		if !out[i].modTime.Equal(out[j].modTime) {
			return out[i].modTime.Before(out[j].modTime)
		}
		return out[i].name < out[j].name
	})
	return out, nil
}

func (p *DirectoryPruner) entryBytes(ctx context.Context, path string, info fs.FileInfo) (int64, error) {
	if !info.IsDir() {
		return info.Size(), nil
	}
	var total int64
	err := filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		total += fi.Size()
		return nil
	})
	if err != nil {
		return total, fmt.Errorf("size %s: %w", path, err)
	}
	return total, nil
}

// Prune removes whole top-level entries, oldest first, until the directory is
// within b.
//
// Deletion frees space immediately, so unlike the SQLite pruner there is no
// separate compaction step and nothing for a free-space guard to protect.
func (p *DirectoryPruner) Prune(ctx context.Context, b Budget) (Result, error) {
	entries, err := p.scan(ctx)
	if err != nil {
		return Result{Budget: b.Name, Incomplete: isCancellation(err)}, err
	}

	before := Usage{}
	for _, e := range entries {
		before.Bytes += e.bytes
		before.Items++
	}
	result := Result{Budget: b.Name, Before: before, BoundBy: BoundNone}

	remainingBytes := before.Bytes
	cutoff := p.cfg.Now().Add(-b.MaxAge)

	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			result.Incomplete = true
			result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
			return result, err
		}

		overAge := b.HasAgeBound() && e.modTime.Before(cutoff)
		overBytes := b.HasByteBound() && remainingBytes > b.MaxBytes
		if !overAge && !overBytes {
			// Entries are oldest-first, so once one is inside both bounds every
			// later one is too.
			break
		}

		if err := os.RemoveAll(filepath.Join(p.cfg.Path, e.name)); err != nil {
			result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
			return result, fmt.Errorf("remove %s: %w", filepath.Join(p.cfg.Path, e.name), err)
		}
		remainingBytes -= e.bytes
		result.Deleted++

		// A byte overage that survives the age horizon is the signal: the
		// producer is outrunning the horizon it declared.
		if overBytes {
			result.BoundBy = BoundBytes
		} else if result.BoundBy == BoundNone {
			result.BoundBy = BoundAge
		}
	}

	result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
	result.FreedBytes = before.Bytes - remainingBytes
	return result, nil
}
