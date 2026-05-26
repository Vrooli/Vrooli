package discovery

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"data-backup-manager/internal/sources"
)

// rootKind names which base directory an entry's relPath is resolved against.
// v1 ships only "runtime" (→ ~/.vrooli). A future "repo" kind (resolved via
// packages/repo-contract-go) will add scenarios/*/store entries WITHOUT
// reworking this scanner — that is the deferred Track-B target scope (D9).
type rootKind string

const (
	rootRuntime rootKind = "runtime"
	// rootRepo rootKind = "repo" // DEFERRED (D9): scenario stores.
)

// wellKnownEntry is one manifest row. The manifest is data, not branching
// logic, so extending coverage is appending rows.
type wellKnownEntry struct {
	relPath    string
	root       rootKind
	owner      string
	name       string
	sourceKind sources.SourceKind
	isFile     bool
	rationale  string
}

// wellKnownManifest is the v1 (~/.vrooli-only) set of runtime state worth
// protecting. Ephemeral dirs (logs, cache, metrics, bin, processes) are
// deliberately absent — not listing them is how they're excluded.
var wellKnownManifest = []wellKnownEntry{
	{
		relPath: "plans", root: rootRuntime, owner: platformOwner, name: "plans",
		sourceKind: sources.KindFilesystem,
		rationale:  "Your authored Vrooli plans and backlog — the record of what the system is building.",
	},
	{
		relPath: "state", root: rootRuntime, owner: platformOwner, name: "state",
		sourceKind: sources.KindFilesystem,
		rationale:  "Vrooli runtime state — durable working data scenarios accumulate.",
	},
	{
		relPath: "config", root: rootRuntime, owner: platformOwner, name: "config",
		sourceKind: sources.KindFilesystem,
		rationale:  "Vrooli configuration — how this installation is wired up.",
	},
	{
		relPath: "secrets.json", root: rootRuntime, owner: platformOwner, name: "secrets",
		sourceKind: sources.KindFilesystem, isFile: true,
		rationale: "Vrooli secrets store — irreplaceable credentials (backed up encrypted; contents never read here).",
	},
	{
		relPath: "runtime.db", root: rootRuntime, owner: platformOwner, name: "runtime-db",
		sourceKind: sources.KindSQLite, isFile: true,
		rationale: "Vrooli runtime database — captured as a consistent SQLite copy.",
	},
}

// defaultMaxScanEntries bounds the shallow size estimate so a target with a
// huge tree never stalls the RPC on a deep walk.
const defaultMaxScanEntries = 4096

// WellKnownScanner probes the resolved runtime root for the manifest entries.
// It is strictly read-only: it stats paths and (for directories) reads entry
// metadata for a bounded size estimate. It never reads file contents.
type WellKnownScanner struct {
	runtimeRoot    string
	maxScanEntries int
}

// NewWellKnownScanner resolves the runtime root portably (APP_DATA_DIR ||
// VROOLI_DATA || ~/.vrooli) and constructs the production scanner.
func NewWellKnownScanner() *WellKnownScanner {
	return &WellKnownScanner{runtimeRoot: resolveRuntimeRoot(), maxScanEntries: defaultMaxScanEntries}
}

// NewWellKnownScannerWithRoot constructs a scanner rooted at an explicit
// directory — used by tests to point at a temp tree.
func NewWellKnownScannerWithRoot(root string) *WellKnownScanner {
	return &WellKnownScanner{runtimeRoot: root, maxScanEntries: defaultMaxScanEntries}
}

// Compile-time guarantee.
var _ TargetSourceScanner = (*WellKnownScanner)(nil)

// Scan returns a candidate for each manifest entry that exists and is non-empty
// under the runtime root. Missing or empty paths are silently skipped.
func (w *WellKnownScanner) Scan(ctx context.Context) ([]TargetCandidate, error) {
	if strings.TrimSpace(w.runtimeRoot) == "" {
		return nil, nil
	}
	out := make([]TargetCandidate, 0, len(wellKnownManifest))
	for _, e := range wellKnownManifest {
		if e.root != rootRuntime {
			continue // v1: only runtime-rooted entries are wired.
		}
		abs := filepath.Join(w.runtimeRoot, e.relPath)
		info, err := os.Stat(abs)
		if err != nil {
			continue // missing or unreadable → not a candidate.
		}
		var approx int64
		if e.isFile {
			if info.IsDir() || info.Size() == 0 {
				continue
			}
			approx = info.Size()
		} else {
			if !info.IsDir() {
				continue
			}
			entries, derr := os.ReadDir(abs)
			if derr != nil || len(entries) == 0 {
				continue // unreadable or empty dir → not worth protecting.
			}
			approx = w.boundedDirSize(ctx, abs)
		}
		out = append(out, TargetCandidate{
			Owner:       e.owner,
			Name:        e.name,
			SourceKind:  e.sourceKind,
			Locator:     abs,
			Rationale:   e.rationale,
			ApproxBytes: approx,
		})
	}
	return out, nil
}

// boundedDirSize sums regular-file sizes under root, bailing out after
// maxScanEntries files or on context cancellation. It reads only metadata, never
// file contents, and never fails the scan (unreadable entries are skipped).
func (w *WellKnownScanner) boundedDirSize(ctx context.Context, root string) int64 {
	var total int64
	count := 0
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil || count >= w.maxScanEntries {
			return filepath.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		count++
		if info, ierr := d.Info(); ierr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// RuntimeRoot exposes the portably-resolved Vrooli runtime root so the
// composition root can include it in the protected-path set (Contract Decision
// D4) without duplicating the resolution logic.
func RuntimeRoot() string { return resolveRuntimeRoot() }

// resolveRuntimeRoot resolves the Vrooli runtime root portably. Order:
// APP_DATA_DIR, then VROOLI_DATA, then the user's ~/.vrooli. Returns "" when no
// home can be resolved (the scanner then yields no candidates).
func resolveRuntimeRoot() string {
	if v := strings.TrimSpace(os.Getenv("APP_DATA_DIR")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("VROOLI_DATA")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".vrooli")
}
