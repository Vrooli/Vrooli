// Package artifactretention bounds the regenerable build outputs that pile up
// inside isolated source-tree copies under the plan-artifacts runtime-home
// root (~/.vrooli/plan-artifacts).
//
// Plan Manager owns that root by the repo contract (runtime_home.plan_artifacts,
// protected, cleanup never). Most of it is durable plan source and evidence and
// is never touched here. The exception is the "isolated-tree" an agent creates
// inside an effort or plan workspace so a scenario variant can build and serve
// from an independent source tree (`vrooli scenario start --path`). Those trees
// carry gigabytes of gitignored build outputs (node_modules, dist,
// dist-electron, generated desktop platforms, compiled API binaries) that the
// scenario lifecycle regenerates on the next build.
//
// The owner never deletes source: an isolated tree may hold the only copy of
// edits that were never landed, and no authoritative lifecycle signal says
// whether they were. Whole-tree removal is therefore an operator decision that
// the receipt lists; automatic retention removes only regenerable outputs, and
// only from a tree that no live process uses and that has been idle for the
// grace window. Unfinished work loses at most a rebuild.
package artifactretention

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

// TreeDirName is the conventional name of an isolated source-tree copy.
const TreeDirName = "isolated-tree"

// DefaultGrace is how long a tree must be idle before its outputs are pruned.
const DefaultGrace = 7 * 24 * time.Hour

// maxListedRemovals caps how many removed paths one tree report lists.
const maxListedRemovals = 40

// Mode selects whether a cycle deletes or only reports.
type Mode string

const (
	ModeEnforce Mode = "enforce"
	ModeReport  Mode = "report"
)

// Tree statuses.
const (
	StatusPruned    = "pruned"
	StatusPrunable  = "prunable"
	StatusProtected = "protected"
	StatusClean     = "clean"
	StatusError     = "error"
)

// Config drives one retention cycle.
type Config struct {
	// Root is the plan-artifacts runtime-home root.
	Root  string
	Grace time.Duration
	Mode  Mode
	Now   func() time.Time
	// LivePaths returns the working directories and executables of live
	// processes. An error protects every tree (fail closed).
	LivePaths func() ([]string, error)
	// Remove deletes one regenerable output. Defaults to os.RemoveAll.
	Remove func(path string) error
}

// TreeReport is the per-tree outcome of a cycle.
type TreeReport struct {
	Path             string    `json:"path"`
	Status           string    `json:"status"`
	Reason           string    `json:"reason,omitempty"`
	LastActivity     time.Time `json:"last_activity"`
	Bytes            int64     `json:"bytes"`
	RegenerableBytes int64     `json:"regenerable_bytes"`
	FreedBytes       int64     `json:"freed_bytes"`
	RemovedCount     int       `json:"removed_count"`
	Removed          []string  `json:"removed,omitempty"`
	Error            string    `json:"error,omitempty"`
}

// Receipt is the durable record of the latest cycle.
type Receipt struct {
	Owner            string       `json:"owner"`
	CycleTime        time.Time    `json:"cycle_time"`
	Mode             Mode         `json:"mode"`
	Root             string       `json:"root"`
	Grace            string       `json:"grace"`
	FreedBytes       int64        `json:"freed_bytes"`
	ReclaimableBytes int64        `json:"reclaimable_bytes"`
	Trees            []TreeReport `json:"trees"`
	// OperatorReview lists idle, unused trees whose remaining source copy only
	// an operator may remove: the owner cannot prove its edits were landed.
	OperatorReview []string `json:"operator_review,omitempty"`
	Error          string   `json:"error,omitempty"`
}

// Run executes one cycle and returns its receipt. A per-tree failure is
// recorded on that tree and joined into the returned error; it never stops the
// remaining trees.
func Run(ctx context.Context, cfg Config) (Receipt, error) {
	now := time.Now
	if cfg.Now != nil {
		now = cfg.Now
	}
	grace := cfg.Grace
	if grace <= 0 {
		grace = DefaultGrace
	}
	mode := cfg.Mode
	if mode != ModeReport {
		mode = ModeEnforce
	}
	remove := cfg.Remove
	if remove == nil {
		remove = os.RemoveAll
	}
	receipt := Receipt{Owner: "plan-manager", CycleTime: now().UTC(), Mode: mode, Root: cfg.Root, Grace: grace.String(), Trees: []TreeReport{}}
	if strings.TrimSpace(cfg.Root) == "" || !filepath.IsAbs(cfg.Root) {
		err := fmt.Errorf("plan-artifacts root %q is not an absolute path", cfg.Root)
		receipt.Error = err.Error()
		return receipt, err
	}
	trees, err := DiscoverTrees(cfg.Root)
	if err != nil {
		receipt.Error = err.Error()
		return receipt, err
	}
	var live []string
	var liveErr error
	if cfg.LivePaths == nil {
		liveErr = errors.New("no live-process probe configured")
	} else {
		live, liveErr = cfg.LivePaths()
	}

	var errs []error
	for _, tree := range trees {
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
		report := processTree(ctx, tree, live, liveErr, now(), grace, mode, remove)
		if report.Error != "" {
			errs = append(errs, fmt.Errorf("%s: %s", tree, report.Error))
		}
		receipt.FreedBytes += report.FreedBytes
		if report.Status == StatusPrunable {
			receipt.ReclaimableBytes += report.RegenerableBytes
		}
		if (report.Status == StatusPruned || report.Status == StatusClean || report.Status == StatusPrunable) && report.Error == "" {
			receipt.OperatorReview = append(receipt.OperatorReview, tree)
		}
		receipt.Trees = append(receipt.Trees, report)
	}
	joined := errors.Join(errs...)
	if joined != nil {
		receipt.Error = joined.Error()
	}
	return receipt, joined
}

// DiscoverTrees lists real (non-symlink) isolated-tree directories directly
// inside an effort workspace (<root>/efforts/<id>/isolated-tree) or a plan
// workspace (<root>/<slug>/isolated-tree).
func DiscoverTrees(root string) ([]string, error) {
	var out []string
	parents := []string{root, filepath.Join(root, "efforts")}
	for _, parent := range parents {
		entries, err := os.ReadDir(parent)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
				continue
			}
			candidate := filepath.Join(parent, entry.Name(), TreeDirName)
			info, err := os.Lstat(candidate)
			if err != nil || !info.IsDir() {
				continue
			}
			out = append(out, candidate)
		}
	}
	sort.Strings(out)
	return out, nil
}

func processTree(ctx context.Context, tree string, live []string, liveErr error, now time.Time, grace time.Duration, mode Mode, remove func(string) error) TreeReport {
	report := TreeReport{Path: tree}
	scan, err := scanTree(ctx, tree)
	report.LastActivity = scan.lastActivity.UTC()
	report.Bytes = scan.bytes
	report.RegenerableBytes = scan.regenerableBytes
	if err != nil {
		report.Status = StatusError
		report.Error = err.Error()
		return report
	}
	switch {
	case liveErr != nil:
		report.Status, report.Reason = StatusProtected, "live process probe unavailable: "+liveErr.Error()
		return report
	case usedByLiveProcess(tree, live):
		report.Status, report.Reason = StatusProtected, "live process uses the tree"
		return report
	case now.Sub(scan.lastActivity) < grace:
		report.Status, report.Reason = StatusProtected, fmt.Sprintf("active within grace window (%s)", grace)
		return report
	case len(scan.outputs) == 0:
		report.Status, report.Reason = StatusClean, "no regenerable outputs remain"
		return report
	case mode == ModeReport:
		report.Status, report.Reason = StatusPrunable, "report mode"
		return report
	}
	report.Status = StatusPruned
	var errs []error
	for _, output := range scan.outputs {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		if err := removeOutput(tree, output.path, remove); err != nil {
			errs = append(errs, err)
			continue
		}
		report.FreedBytes += output.bytes
		report.RemovedCount++
		if len(report.Removed) < maxListedRemovals {
			rel, _ := filepath.Rel(tree, output.path)
			report.Removed = append(report.Removed, rel)
		}
	}
	if joined := errors.Join(errs...); joined != nil {
		report.Error = joined.Error()
	}
	return report
}

// removeOutput re-validates a path immediately before deleting it: it must
// still be inside the tree, must not be a symlink and must not resolve through
// one. os.RemoveAll never follows symlinks found inside the directory.
func removeOutput(tree, path string, remove func(string) error) error {
	if !within(path, tree) || path == tree {
		return fmt.Errorf("refusing to remove %s outside %s", path, tree)
	}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove symlink %s", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	resolvedTree, err := filepath.EvalSymlinks(tree)
	if err != nil {
		return err
	}
	if !within(resolved, resolvedTree) {
		return fmt.Errorf("refusing to remove %s: resolves outside the tree to %s", path, resolved)
	}
	return remove(path)
}

type output struct {
	path  string
	bytes int64
}

type treeScan struct {
	lastActivity     time.Time
	bytes            int64
	regenerableBytes int64
	outputs          []output
}

// scanTree walks a tree without following symlinks, recording total size,
// newest modification time and the regenerable outputs it contains.
func scanTree(ctx context.Context, tree string) (treeScan, error) {
	var scan treeScan
	current := -1 // index into scan.outputs of the enclosing regenerable dir
	err := filepath.WalkDir(tree, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if current >= 0 && !within(path, scan.outputs[current].path) {
			current = -1
		}
		// Activity is read from source and from each output's own root entry,
		// never from inside an output: package managers hard-link store files
		// (pnpm file: dependencies), so a rebuild elsewhere refreshes their
		// mtime here without anyone using this tree.
		if current < 0 && info.ModTime().After(scan.lastActivity) {
			scan.lastActivity = info.ModTime()
		}
		rel, _ := filepath.Rel(tree, path)
		if d.IsDir() {
			if current < 0 && path != tree && IsRegenerableDir(rel) {
				scan.outputs = append(scan.outputs, output{path: path})
				current = len(scan.outputs) - 1
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		size := info.Size()
		scan.bytes += size
		switch {
		case current >= 0:
			scan.outputs[current].bytes += size
			scan.regenerableBytes += size
		case IsRegenerableFile(rel):
			scan.outputs = append(scan.outputs, output{path: path, bytes: size})
			scan.regenerableBytes += size
		}
		return nil
	})
	return scan, err
}

// regenerableDirNames mirrors the repository .gitignore build-output rules:
// every one is rebuilt by the scenario lifecycle or package installer.
var regenerableDirNames = map[string]bool{
	"node_modules":  true,
	"dist":          true,
	"dist-electron": true,
	"test-dist":     true,
	"bin":           true,
	".vite":         true,
	".vitest":       true,
	"__pycache__":   true,
	".nyc_output":   true,
}

// IsRegenerableDir reports whether a tree-relative directory is a build
// output. Generated desktop platform bundles (scenarios/<s>/platforms) are
// regenerable except where the repository keeps source there.
func IsRegenerableDir(rel string) bool {
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	name := parts[len(parts)-1]
	if regenerableDirNames[name] || strings.HasPrefix(name, ".vrooli-artifact-stage-") || strings.HasPrefix(name, "v8-compile-cache-") {
		return true
	}
	if len(parts) == 3 && parts[0] == "scenarios" && name == "platforms" && parts[1] != "secrets-manager" {
		return true
	}
	return false
}

// IsRegenerableFile reports whether a tree-relative file is a compiled
// binary or build sidecar the lifecycle recreates.
func IsRegenerableFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	name := parts[len(parts)-1]
	if strings.HasSuffix(name, ".freshness.json") || name == ".vrooli-artifact-stage.lock" || strings.HasPrefix(name, ".vrooli-closure-go_module") {
		return true
	}
	if len(parts) < 2 {
		return false
	}
	parent := parts[len(parts)-2]
	switch parent {
	case "api":
		return strings.HasSuffix(name, "-api") || name == "api" || name == "main" || name == "webconsole"
	case "ui":
		return strings.HasSuffix(name, "-ui")
	case "cli":
		return name == "cli" || name == "main"
	}
	return false
}

func usedByLiveProcess(tree string, live []string) bool {
	resolved, err := filepath.EvalSymlinks(tree)
	if err != nil {
		resolved = tree
	}
	for _, path := range live {
		path = strings.TrimSuffix(path, " (deleted)")
		if within(path, tree) || within(path, resolved) {
			return true
		}
	}
	return false
}

func within(path, root string) bool {
	path, root = filepath.Clean(path), filepath.Clean(root)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

// ProcLivePaths reads the cwd, exe and root links of every process in /proc.
// Processes that vanish or are not readable are skipped; a missing /proc is
// an error so callers fail closed.
func ProcLivePaths() ([]string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if name == "" || name[0] < '0' || name[0] > '9' {
			continue
		}
		for _, link := range []string{"cwd", "exe", "root"} {
			target, err := os.Readlink(filepath.Join("/proc", name, link))
			if err == nil && target != "" && target != "/" {
				out = append(out, target)
			}
		}
	}
	return out, nil
}

// WriteReceipt atomically replaces the receipt file through the reviewed
// storage seam.
func WriteReceipt(path string, receipt Receipt) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, append(data, '\n'), 0o644)
}
