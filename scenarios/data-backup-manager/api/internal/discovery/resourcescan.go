package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"data-backup-manager/internal/sources"
)

// ResourceDataScanner derives target candidates from the durable host state
// each external-cli resource declares in its manifest (durable_data). It is the
// second TargetSourceScanner alongside WellKnownScanner (~/.vrooli): together,
// behind a CompositeScanner, they cover both Vrooli's own runtime home and the
// coding agents' irreplaceable conversation history.
//
// Strictly read-only: it stats declared paths and, for directories, walks
// metadata for a bounded size estimate. It never reads file contents.
type ResourceDataScanner struct {
	resources      ResourceEnumerator
	home           string
	maxScanEntries int
}

// NewResourceDataScanner resolves the operator home via os.UserHomeDir (the API
// runs as the operator, consistent with WellKnownScanner).
func NewResourceDataScanner(enum ResourceEnumerator) *ResourceDataScanner {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return &ResourceDataScanner{resources: enum, home: home, maxScanEntries: defaultMaxScanEntries}
}

// NewResourceDataScannerWithHome constructs a scanner rooted at an explicit home
// — used by tests to point at a temp tree.
func NewResourceDataScannerWithHome(enum ResourceEnumerator, home string) *ResourceDataScanner {
	return &ResourceDataScanner{resources: enum, home: home, maxScanEntries: defaultMaxScanEntries}
}

// Compile-time guarantee.
var _ TargetSourceScanner = (*ResourceDataScanner)(nil)

// Scan returns a candidate for each durable (non-regenerable) declared entry
// that exists and is non-empty under the resolved home. Missing, empty, or
// unreadable paths are silently skipped, as is any resource whose manifest
// declares no durable_data.
func (s *ResourceDataScanner) Scan(ctx context.Context) ([]TargetCandidate, error) {
	if strings.TrimSpace(s.home) == "" || s.resources == nil {
		return nil, nil
	}
	refs, err := s.resources.Enumerate(ctx)
	if err != nil {
		return nil, fmt.Errorf("enumerate resources: %w", err)
	}
	var out []TargetCandidate
	for _, ref := range refs {
		dd := loadDurableData(ref.ManifestPath)
		if dd == nil {
			continue
		}
		base, ok := resolveBase(dd.Base, s.home)
		if !ok {
			continue // unrecognized base token → skip the whole resource defensively.
		}
		for _, key := range sortedEntryKeys(dd.Entries) {
			entry := dd.Entries[key]
			if entry.Regenerable {
				continue // declared reconstructable → not worth protecting.
			}
			candidate, ok := s.candidateFor(ctx, ref.Name, key, base, entry)
			if ok {
				out = append(out, candidate)
			}
		}
	}
	return out, nil
}

// candidateFor resolves and stats one declared entry, returning a candidate when
// it exists and is non-empty. The resolved path must stay under home (defense in
// depth against a manifest that slips a traversal past validation).
func (s *ResourceDataScanner) candidateFor(ctx context.Context, owner, name, base string, entry DurableDataEntry) (TargetCandidate, bool) {
	if hasParentTraversal(entry.Path) || strings.Contains(entry.Path, "\\") {
		return TargetCandidate{}, false
	}
	abs := filepath.Join(base, filepath.FromSlash(entry.Path))
	if !within(abs, s.home) {
		return TargetCandidate{}, false
	}
	info, err := os.Stat(abs)
	if err != nil {
		return TargetCandidate{}, false
	}
	var approx int64
	if entry.Kind == "file" {
		if info.IsDir() || info.Size() == 0 {
			return TargetCandidate{}, false
		}
		approx = info.Size()
	} else {
		if !info.IsDir() {
			return TargetCandidate{}, false
		}
		dirEntries, derr := os.ReadDir(abs)
		if derr != nil || len(dirEntries) == 0 {
			return TargetCandidate{}, false
		}
		approx = boundedDirSize(ctx, abs, s.maxScanEntries)
	}
	return TargetCandidate{
		Owner:       owner,
		Name:        name,
		SourceKind:  durableSourceKind(entry),
		Locator:     abs,
		Rationale:   durableRationale(owner, name, entry),
		ApproxBytes: approx,
		Sensitive:   entry.Sensitive,
	}, true
}

// durableSourceKind maps a declared format to a capture strategy: sqlite-format
// files get a consistent SQLite copy; everything else is a filesystem capture.
func durableSourceKind(e DurableDataEntry) sources.SourceKind {
	if e.Format == "sqlite" {
		return sources.KindSQLite
	}
	return sources.KindFilesystem
}

// durableRationale uses the manifest's authored rationale when present, else a
// generic fallback naming the resource and entry.
func durableRationale(owner, name string, e DurableDataEntry) string {
	if r := strings.TrimSpace(e.Rationale); r != "" {
		return r
	}
	return fmt.Sprintf("Durable %s data (%s) declared by the %s resource.", owner, name, owner)
}

// sortedEntryKeys returns the entry keys in deterministic order so suggestions
// (and tests) are stable across scans.
func sortedEntryKeys(entries map[string]DurableDataEntry) []string {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
