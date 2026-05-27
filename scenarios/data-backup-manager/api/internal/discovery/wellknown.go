package discovery

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"

	"data-backup-manager/internal/sources"
)

// dbmRationale is the scenario's opinion layer over the runtime_home authority:
// for each durable (non-regenerable) contract entry, the operator-facing name
// and the "why back this up" copy. The set of *what* to protect comes entirely
// from the contract (entries with regenerable=false); this table only adds
// presentation. An entry missing here falls back to its contract key + a
// generic rationale, so a new durable contract entry is never silently dropped.
var dbmRationale = map[string]struct {
	name      string
	rationale string
}{
	repocontract.HomeKeyPlans:      {"plans", "Your authored Vrooli plans and backlog — the record of what the system is building."},
	repocontract.HomeKeyState:      {"state", "Vrooli runtime state — durable working data scenarios accumulate."},
	repocontract.HomeKeyConfig:     {"config", "Vrooli configuration — how this installation is wired up."},
	repocontract.HomeKeyData:       {"data", "Durable scenario data — application state scenarios persist between runs."},
	repocontract.HomeKeyRuntimeDB:  {"runtime-db", "Vrooli runtime database — captured as a consistent SQLite copy."},
	repocontract.HomeKeySecrets:    {"secrets", "Vrooli secrets store — irreplaceable credentials (backed up encrypted; contents never read here)."},
	repocontract.HomeKeySecretsEnc: {"secrets-enc", "Encrypted Vrooli secrets store — irreplaceable credentials (contents never read here)."},
}

// defaultMaxScanEntries bounds the shallow size estimate so a target with a
// huge tree never stalls the RPC on a deep walk.
const defaultMaxScanEntries = 4096

// WellKnownScanner probes the resolved runtime root for the durable entries the
// repo contract declares (regenerable=false). It is strictly read-only: it stats
// paths and (for directories) reads entry metadata for a bounded size estimate.
// It never reads file contents.
type WellKnownScanner struct {
	runtimeRoot    string
	maxScanEntries int
}

// NewWellKnownScanner resolves the runtime root via the runtime_home authority
// (the operator's $HOME joined with the contract dir name). There are no
// APP_DATA_DIR/VROOLI_DATA overrides (Contract Decision CD-2): the scenario uses
// the same single resolution path as the rest of the platform.
func NewWellKnownScanner() *WellKnownScanner {
	return &WellKnownScanner{runtimeRoot: RuntimeRoot(), maxScanEntries: defaultMaxScanEntries}
}

// NewWellKnownScannerWithRoot constructs a scanner rooted at an explicit
// runtime-home directory — used by tests to point at a temp tree.
func NewWellKnownScannerWithRoot(root string) *WellKnownScanner {
	return &WellKnownScanner{runtimeRoot: root, maxScanEntries: defaultMaxScanEntries}
}

// Compile-time guarantee.
var _ TargetSourceScanner = (*WellKnownScanner)(nil)

// Scan returns a candidate for each durable runtime_home entry that exists and
// is non-empty under the runtime root. Regenerable entries (bin/cache/logs/…)
// are excluded by the contract flag, not by omission. Missing or empty paths are
// silently skipped.
func (w *WellKnownScanner) Scan(ctx context.Context) ([]TargetCandidate, error) {
	if strings.TrimSpace(w.runtimeRoot) == "" {
		return nil, nil
	}
	entries, err := durableEntries()
	if err != nil {
		return nil, err
	}
	out := make([]TargetCandidate, 0, len(entries))
	for _, e := range entries {
		abs := filepath.Join(w.runtimeRoot, filepath.FromSlash(e.RelPath))
		info, err := os.Stat(abs)
		if err != nil {
			continue // missing or unreadable → not a candidate.
		}
		isFile := e.Kind == "file"
		var approx int64
		if isFile {
			if info.IsDir() || info.Size() == 0 {
				continue
			}
			approx = info.Size()
		} else {
			if !info.IsDir() {
				continue
			}
			dirEntries, derr := os.ReadDir(abs)
			if derr != nil || len(dirEntries) == 0 {
				continue // unreadable or empty dir → not worth protecting.
			}
			approx = boundedDirSize(ctx, abs, w.maxScanEntries)
		}
		meta := dbmRationale[e.Key]
		name := meta.name
		if name == "" {
			name = e.Key
		}
		rationale := meta.rationale
		if rationale == "" {
			rationale = "Durable Vrooli runtime data."
		}
		out = append(out, TargetCandidate{
			Owner:       platformOwner,
			Name:        name,
			SourceKind:  sourceKindFor(e),
			Locator:     abs,
			Rationale:   rationale,
			ApproxBytes: approx,
		})
	}
	return out, nil
}

// sourceKindFor maps a contract entry's declared format to a capture strategy:
// sqlite-formatted files get a consistent SQLite copy; everything else is a
// filesystem capture.
func sourceKindFor(e repocontract.HomeEntry) sources.SourceKind {
	if e.Format == "sqlite" {
		return sources.KindSQLite
	}
	return sources.KindFilesystem
}

// durableEntries returns the contract's runtime_home entries with
// regenerable=false, sorted by key for deterministic output. The home value is
// irrelevant here — only the relative structure/flags are used — so a fixed
// placeholder is passed and only RelPath/Kind/Format are consumed.
func durableEntries() ([]repocontract.HomeEntry, error) {
	contract, _, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return nil, err
	}
	all, err := contract.RuntimeHomeEntries(placeholderHome)
	if err != nil {
		return nil, err
	}
	durable := make([]repocontract.HomeEntry, 0, len(all))
	for _, e := range all {
		if e.Regenerable {
			continue
		}
		durable = append(durable, e)
	}
	sort.Slice(durable, func(i, j int) bool { return durable[i].Key < durable[j].Key })
	return durable, nil
}

// placeholderHome is a fixed home used only to extract relative structure from
// the contract; the resulting AbsPath is discarded (RelPath is what's used).
const placeholderHome = "/__dbm_runtime_home__"

// RuntimeRoot resolves the Vrooli runtime root (the operator's ~/.vrooli) via the
// runtime_home authority, so the composition root can include it in the
// protected-path set (Contract Decision D4) without duplicating resolution.
// The scenario API runs as the operator (not under sudo), so os.UserHomeDir is
// the correct home here; there are no env overrides (CD-2).
func RuntimeRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return ""
	}
	return root
}
