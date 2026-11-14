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
	// EntryDepth selects whole entries at this depth (default 1). Grouped
	// caches use 2 for <application>/<build>, never the application directory.
	EntryDepth int
	// ExpandDirs names top-level containers whose children are separate entries
	// alongside the other top-level entries. Only valid with EntryDepth 1.
	ExpandDirs []string
	// MaxItems bounds deletions per cycle; zero means unlimited.
	MaxItems int
	// KeepLatest protects the newest entries across the complete inventory.
	KeepLatest int
	// Eligible supplies owner-specific live-work protection. Errors fail closed.
	// It is checked during selection and again immediately before removal.
	Eligible func(context.Context, string) (bool, error)
	// ProtectedRoots are absolute paths that this pruner must never remove or
	// remove through. A configured root, or any child/ancestor overlap with a
	// protected root, is refused at the deletion boundary.
	ProtectedRoots []string
	// ProtectedGlobs are path patterns, matched against the cleaned absolute
	// candidate path and its base name, that must never be removed.
	ProtectedGlobs []string
	// DirectoryMtimeReliable controls whether a directory's own mtime is a
	// sufficient age signal. Set false on filesystems that do not propagate
	// descendant changes to the directory entry; selection then uses the newest
	// descendant mtime.
	DirectoryMtimeReliable *bool
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
	if cfg.EntryDepth == 0 {
		cfg.EntryDepth = 1
	}
	if cfg.EntryDepth < 1 || cfg.EntryDepth > 8 || cfg.MaxItems < 0 || cfg.KeepLatest < 0 {
		return nil, fmt.Errorf("directory pruner: invalid depth, batch limit, or keep count")
	}
	for _, name := range cfg.ExpandDirs {
		if cfg.EntryDepth != 1 || name == "" || name == "." || name == ".." || filepath.Base(name) != name {
			return nil, fmt.Errorf("directory pruner: invalid expanded container %q", name)
		}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.DirectoryMtimeReliable == nil {
		reliable := true
		cfg.DirectoryMtimeReliable = &reliable
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
	name      string
	modTime   time.Time
	bytes     int64
	protected bool
}

// Select scans and orders candidates without deleting anything. It is the
// reusable selection half of the directory retention engine.
func (p *DirectoryPruner) Select(ctx context.Context, b Budget) (Candidates, error) {
	entries, err := p.scan(ctx)
	if err != nil {
		return nil, err
	}
	victims, _ := p.selectVictims(entries, b)
	selected := make(Candidates, 0, len(victims))
	for _, victim := range victims {
		selected = append(selected, Candidate{Path: filepath.Join(p.cfg.Path, victim.name), Bytes: victim.bytes, ModTime: victim.modTime})
	}
	return selected, nil
}

// Delete removes selected candidates with a re-stat and containment check at
// the deletion boundary. A caller may use Lock to share the host recovery
// lock with other deleters.
func (p *DirectoryPruner) Delete(ctx context.Context, candidates Candidates, batch Batch) (Receipt, error) {
	started := time.Now()
	receipt := Receipt{}
	if before, _, err := MeasureDirectory(ctx, p.cfg.Path); err == nil {
		receipt.BytesBefore = before
	}
	var deletedBytes int64
	if batch.Lock != nil {
		unlock, err := batch.Lock(ctx)
		if err != nil {
			return receipt, err
		}
		defer unlock()
	}
	for index, candidate := range candidates {
		if index > 0 && batch.MaxItems > 0 && receipt.Files >= int64(batch.MaxItems) {
			receipt.Partial = true
			break
		}
		if err := ctx.Err(); err != nil {
			receipt.Partial = true
			receipt.Duration = time.Since(started)
			return receipt, err
		}
		if !PathContains(p.cfg.Path, filepath.Clean(candidate.Path)) || ProtectedPathOverlap(candidate.Path, p.protectedRoots) {
			return receipt, fmt.Errorf("refusing to remove path outside directory target: %s", candidate.Path)
		}
		if protectedGlob(candidate.Path, p.cfg.ProtectedGlobs) {
			return receipt, fmt.Errorf("refusing to remove protected path pattern: %s", candidate.Path)
		}
		resolved, resolveErr := filepath.EvalSymlinks(candidate.Path)
		if resolveErr == nil {
			if !PathContains(p.cfg.Path, resolved) {
				return receipt, fmt.Errorf("refusing to remove path that resolves outside directory target: %s", candidate.Path)
			}
			if ProtectedPathOverlap(resolved, p.protectedRoots) {
				return receipt, fmt.Errorf("refusing to remove path that resolves to a protected root: %s", candidate.Path)
			}
		}
		if resolveErr != nil && !os.IsNotExist(resolveErr) {
			return receipt, fmt.Errorf("resolve deletion candidate %s: %w", candidate.Path, resolveErr)
		}
		info, err := os.Stat(candidate.Path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return receipt, err
		}
		bytes := candidate.Bytes
		if info.IsDir() {
			if measured, _, measureErr := MeasureDirectory(ctx, candidate.Path); measureErr == nil {
				bytes = measured
			}
		} else {
			bytes = info.Size()
		}
		if batch.MaxBytes > 0 && deletedBytes+bytes > batch.MaxBytes {
			receipt.Partial = true
			break
		}
		eligible, err := p.eligible(ctx, candidate.Path)
		if err != nil {
			return receipt, err
		}
		if !eligible {
			receipt.Partial = true
			continue
		}
		if err := p.cfg.RemoveHook(candidate.Path, func() error { return DeleteContained(ctx, p.cfg.Path, candidate.Path, p.protectedRoots) }); err != nil {
			return receipt, err
		}
		deletedBytes += bytes
		receipt.Files++
	}
	if usage, _, err := MeasureDirectory(ctx, p.cfg.Path); err == nil {
		receipt.BytesAfter = usage
	}
	receipt.Duration = time.Since(started)
	return receipt, nil
}

func protectedGlob(path string, patterns []string) bool {
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
	var names []string
	var enumerate func(string, int) error
	enumerate = func(dir string, depth int) error {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, de := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			path := filepath.Join(dir, de.Name())
			expanded := false
			if dir == p.cfg.Path && de.IsDir() && de.Type()&os.ModeSymlink == 0 {
				for _, name := range p.cfg.ExpandDirs {
					if name == de.Name() {
						expanded = true
						break
					}
				}
			}
			if expanded {
				if err := enumerate(path, depth); err != nil {
					return err
				}
				continue
			}
			if depth == p.cfg.EntryDepth {
				names = append(names, path)
				continue
			}
			if de.IsDir() && de.Type()&os.ModeSymlink == 0 {
				if err := enumerate(path, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := enumerate(p.cfg.Path, 1)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read directory %s: %w", p.cfg.Path, err)
	}

	out := make([]entry, 0, len(names))
	for _, path := range names {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		info, err := os.Lstat(path)
		if err != nil {
			// An entry that vanished between listing and stat is already gone,
			// which is the outcome pruning wants anyway.
			if os.IsNotExist(err) {
				continue
			}
			return out, fmt.Errorf("stat %s: %w", path, err)
		}
		size, newest, err := p.entryBytes(ctx, path, info)
		if err != nil {
			return out, err
		}
		modTime := info.ModTime()
		if !*p.cfg.DirectoryMtimeReliable && newest.After(modTime) {
			modTime = newest
		}
		eligible, err := p.eligible(ctx, path)
		if err != nil {
			return out, err
		}
		name, err := filepath.Rel(p.cfg.Path, path)
		if err != nil {
			return out, err
		}
		out = append(out, entry{name: name, modTime: modTime, bytes: size, protected: !eligible})
	}

	sort.Slice(out, func(i, j int) bool {
		if !out[i].modTime.Equal(out[j].modTime) {
			return out[i].modTime.Before(out[j].modTime)
		}
		return out[i].name < out[j].name
	})
	for i := len(out) - 1; i >= 0 && i >= len(out)-p.cfg.KeepLatest; i-- {
		out[i].protected = true
	}
	return out, nil
}

func (p *DirectoryPruner) eligible(ctx context.Context, path string) (bool, error) {
	if protectedGlob(path, p.cfg.ProtectedGlobs) {
		return false, nil
	}
	if p.cfg.Eligible != nil {
		return p.cfg.Eligible(ctx, path)
	}
	return true, nil
}

func (p *DirectoryPruner) entryBytes(ctx context.Context, path string, info fs.FileInfo) (int64, time.Time, error) {
	if !info.IsDir() {
		return info.Size(), info.ModTime(), nil
	}
	var total int64
	newest := info.ModTime()
	err := WalkDirectory(ctx, path, func(_ string, d fs.DirEntry, walkErr error) error {
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
			if info, infoErr := d.Info(); infoErr == nil && info.ModTime().After(newest) {
				newest = info.ModTime()
			}
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
		if fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
		return nil
	})
	if err != nil {
		return total, newest, fmt.Errorf("size %s: %w", path, err)
	}
	return total, newest, nil
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
		result.Incomplete = b.HasByteBound() && before.Bytes > b.MaxBytes
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
		if p.cfg.MaxItems > 0 && result.Deleted >= int64(p.cfg.MaxItems) {
			result.Incomplete = true
			break
		}
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
		eligible, err := p.eligible(ctx, candidate)
		if err != nil {
			result.After = Usage{Bytes: remainingBytes, Items: before.Items - result.Deleted}
			result.FreedBytes = before.Bytes - remainingBytes
			return result, err
		}
		if !eligible {
			result.Incomplete = true
			continue
		}
		if err := p.cfg.RemoveHook(candidate, func() error { return DeleteContained(ctx, p.cfg.Path, candidate, p.protectedRoots) }); err != nil {
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
	result.Incomplete = result.Incomplete || (b.HasByteBound() && remainingBytes > b.MaxBytes)
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
	victims := make([]entry, 0)
	for _, e := range entries {
		if e.protected {
			continue
		}
		overAge := b.HasAgeBound() && e.modTime.Before(cutoff)
		overBytes := b.HasByteBound() && remainingBytes > b.MaxBytes
		if !overAge && !overBytes {
			// Entries are oldest-first, so once one is inside both bounds every
			// later one is too.
			return victims, boundBy
		}
		// A byte overage that survives the age horizon is the signal: the
		// producer is outrunning the horizon it declared.
		if overBytes {
			boundBy = BoundBytes
		} else if boundBy == BoundNone {
			boundBy = BoundAge
		}
		remainingBytes -= e.bytes
		victims = append(victims, e)
	}
	return victims, boundBy
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
