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
	// ProtectedRoots are absolute paths that this pruner must never remove or
	// remove through. A configured root, or any child/ancestor overlap with a
	// protected root, is refused at the deletion boundary.
	ProtectedRoots []string
	// Now supplies the current time. Defaults to time.Now.
	Now func() time.Time
	// Logger receives cycle detail. Defaults to slog.Default.
	Logger *slog.Logger

	// MaxDeleteFraction bounds how much of a directory one cycle may remove,
	// as a fraction of its measured top-level entries between 0 and 1. Zero
	// disables the cap.
	//
	// It exists because a budget is a declaration, and declarations are
	// sometimes wrong by orders of magnitude -- a units slip, a ceiling copied
	// from a different entry, or a path that names a shared directory the
	// declarer only contributes a few files to. Pruning is oldest-first, so a
	// ceiling far below the steady-state size does not trim a tail: it walks
	// the whole directory from its oldest entry and stops only when almost
	// nothing is left. A healthy retention cycle removes the tail; one that
	// would remove nearly everything is evidence about the declaration, not
	// about the data. Refusing there keeps the budget working as an alarm and
	// leaves a human the chance to notice.
	MaxDeleteFraction float64

	// RemoveHook wraps the removal of one selected top-level entry. It receives
	// the entry's absolute path and the removal itself, and is responsible for
	// invoking it. The default invokes it directly.
	//
	// A wrapper rather than a replacement, because the two halves belong to
	// different layers. This package owns *how* an entry is deleted; a caller
	// owns what must be true around that deletion -- a durable receipt, a dry
	// run, a quarantine step. Handing callers a replacement would push the
	// deletion itself out into every caller, which is how a codebase ends up
	// with the same os.RemoveAll written in five places under five different
	// sets of guarantees.
	//
	// It also keeps this dependency pointing the right way. Which receipts a
	// deletion deserves is control-plane policy, and api-core must not acquire
	// a dependency on the control plane's state layout to express it.
	//
	// An error fails the cycle, leaving every later entry in place.
	RemoveHook func(path string, remove func() error) error
}

// DirectoryPruner enforces a budget over the top-level entries of one directory.
//
// It deletes whole top-level entries rather than walking into them to remove
// individual files. A half-deleted subtree is harder to reason about than a
// missing one: the caller can tell that a snapshot directory is gone, but cannot
// tell that one is intact.
type DirectoryPruner struct {
	cfg            DirectoryConfig
	protectedRoots []string
}

// NewDirectoryPruner validates cfg and returns the pruner.
func NewDirectoryPruner(cfg DirectoryConfig) (*DirectoryPruner, error) {
	if strings.TrimSpace(cfg.Path) == "" {
		return nil, fmt.Errorf("directory pruner: Path is required")
	}
	if !filepath.IsAbs(cfg.Path) {
		return nil, fmt.Errorf("directory pruner: Path %q must be absolute; resolve it through api-core/storage first", cfg.Path)
	}
	protectedRoots, err := NormalizeProtectedRoots(cfg.ProtectedRoots)
	if err != nil {
		return nil, fmt.Errorf("directory pruner: %w", err)
	}
	if cfg.MaxDeleteFraction < 0 || cfg.MaxDeleteFraction > 1 {
		return nil, fmt.Errorf("directory pruner: MaxDeleteFraction %v must be within [0,1]", cfg.MaxDeleteFraction)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.RemoveHook == nil {
		cfg.RemoveHook = func(_ string, remove func() error) error { return remove() }
	}
	cfg.Path = filepath.Clean(cfg.Path)
	return &DirectoryPruner{cfg: cfg, protectedRoots: protectedRoots}, nil
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

	// Selection is a separate pass from deletion so the cycle's full blast
	// radius is known before anything is destroyed. Deciding entry by entry
	// cannot express "this cycle would empty the directory, so do none of it".
	victims, boundBy := p.selectVictims(entries, b)
	if len(victims) == 0 {
		result.After = before
		return result, nil
	}

	if reason, refused := p.exceedsBlastRadius(len(victims), len(entries)); refused {
		result.After = before
		result.Refused = true
		result.RefusedReason = reason
		p.cfg.Logger.Warn("retention refused: blast radius exceeded",
			"budget", b.Name, "path", p.cfg.Path, "reason", reason)
		return result, nil
	}

	remainingBytes := before.Bytes
	for _, e := range victims {
		if err := ctx.Err(); err != nil {
			result.Incomplete = true
			result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
			result.FreedBytes = before.Bytes - remainingBytes
			return result, err
		}

		candidate := filepath.Join(p.cfg.Path, e.name)
		if ProtectedPathOverlap(candidate, p.protectedRoots) {
			result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
			result.FreedBytes = before.Bytes - remainingBytes
			return result, fmt.Errorf("refusing to remove protected path %s", candidate)
		}
		if err := p.cfg.RemoveHook(candidate, func() error { return os.RemoveAll(candidate) }); err != nil {
			result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
			result.FreedBytes = before.Bytes - remainingBytes
			return result, fmt.Errorf("remove %s: %w", candidate, err)
		}
		remainingBytes -= e.bytes
		result.Deleted++
	}

	result.BoundBy = boundBy
	result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
	result.FreedBytes = before.Bytes - remainingBytes
	return result, nil
}

// selectVictims returns the oldest-first prefix of entries that b does not
// permit to be retained, and which bound determined the retained set.
func (p *DirectoryPruner) selectVictims(entries []entry, b Budget) ([]entry, Bound) {
	remainingBytes := int64(0)
	for _, e := range entries {
		remainingBytes += e.bytes
	}
	cutoff := p.cfg.Now().Add(-b.MaxAge)
	boundBy := BoundNone
	for i, e := range entries {
		overAge := b.HasAgeBound() && e.modTime.Before(cutoff)
		overBytes := b.HasByteBound() && remainingBytes > b.MaxBytes
		if !overAge && !overBytes {
			// Entries are oldest-first, so once one is inside both bounds every
			// later one is too.
			return entries[:i], boundBy
		}
		// A byte overage that survives the age horizon is the signal: the
		// producer is outrunning the horizon it declared.
		if overBytes {
			boundBy = BoundBytes
		} else if boundBy == BoundNone {
			boundBy = BoundAge
		}
		remainingBytes -= e.bytes
	}
	return entries, boundBy
}

// exceedsBlastRadius reports whether removing victims of total entries is more
// destruction than one cycle is allowed to do unattended.
func (p *DirectoryPruner) exceedsBlastRadius(victims, total int) (string, bool) {
	if p.cfg.MaxDeleteFraction <= 0 || total == 0 {
		return "", false
	}
	fraction := float64(victims) / float64(total)
	if fraction <= p.cfg.MaxDeleteFraction {
		return "", false
	}
	return fmt.Sprintf(
		"cycle would remove %d of %d top-level entries (%.0f%%), above the %.0f%% ceiling; "+
			"a budget that prunes nearly all of its directory is evidence the declaration is wrong, so this cycle alarms instead of deleting",
		victims, total, fraction*100, p.cfg.MaxDeleteFraction*100), true
}
