package validation

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

// accountabilityAnalyzer is the provenance label for the rung marker. It is not
// a registered Analyzer: the rung is a function of the other analyzers' output,
// so it runs as a post-pass once every analyzer has reported.
const accountabilityAnalyzer = "storage.accountability"

// reconciliationCodes are the defects that mean a declaration does not yet
// agree with observable reality. Any one of them holds the ladder at
// L1 Declared.
//
// STORAGE_TOKEN_SUPERSEDABLE is deliberately absent: a repeated byOS map is
// verbose, not wrong, and the declaration it produces still matches reality.
var reconciliationCodes = map[string]struct{}{
	"STORAGE_ENTRY_NO_WRITER":           {},
	"STORAGE_ENTRY_CLASS_CONFLICT":      {},
	"STORAGE_SQLITE_SIDECAR_UNDECLARED": {},
	"STORAGE_RETENTION_CONFLICT":        {},
	"STORAGE_BUDGET_BELOW_OBSERVED":     {},
	"STORAGE_PATH_NOT_PORTABLE":         {},
	"STORAGE_PATH_PLATFORM_MISMATCH":    {},
	"STORAGE_PATH_BRANCH_MISSING":       {},
	"STORAGE_PATH_UNACCOUNTED":          {},
	"STORAGE_PATH_ORPHANED":             {},
}

// accountabilityFindings derives the declaration_accountability rung and emits
// at most one marker finding naming the rung that is blocked.
//
// The maturity engine starts every capability at its top level and lowers it
// only on a finding whose mapping blocks (see maturity-go/assessment:
// blocksLocalMaturity). Without a marker, an owner that declares nothing at all
// produces no findings and therefore scores L3 "governed end to end" — the
// exact inversion this function exists to prevent. The markers carry
// clean_requirement "required" with a non-ERROR severity, so they lower the
// rung without failing the storage phase; adoption stays advisory by design.
func accountabilityFindings(ac AnalyzerContext, prior []Finding) []Finding {
	if ac.Owner == nil {
		return nil
	}
	entries := ac.Owner.StorageEntries
	if len(entries) == 0 {
		return []Finding{accountabilityFinding(
			ac,
			"STORAGE_ACCOUNTABILITY_UNDECLARED",
			SeverityInfo,
			"the owner declares no storage.entries, so its durable surface cannot be evaluated",
			"Run 'storage-manager declare suggest' for a measured starting block, or declare an empty surface deliberately.",
		)}
	}
	if blocked := matchingCodes(prior, reconciliationCodes); len(blocked) > 0 {
		return []Finding{accountabilityFinding(
			ac,
			"STORAGE_ACCOUNTABILITY_UNRECONCILED",
			SeverityWarning,
			fmt.Sprintf("the declaration does not yet agree with observed storage (%s)", strings.Join(blocked, ", ")),
			"Resolve the reconciliation findings above; each names the entry and the exact disagreement.",
		)}
	}
	if unbounded := unboundedEntries(entries); len(unbounded) > 0 {
		return []Finding{accountabilityFinding(
			ac,
			"STORAGE_ACCOUNTABILITY_UNGOVERNED",
			SeverityWarning,
			fmt.Sprintf("storage is reconciled but has no visible enforcement: %s %s no budget and no reclaim command", strings.Join(unbounded, ", "), plural(len(unbounded), "has", "have")),
			"Declare a budget (max_bytes and/or max_age) or a reclaim command for each entry, with the measured number and date in its rationale.",
		)}
	}
	return nil
}

// unboundedEntries returns the names of entries with neither a budget nor a
// reclaim command. SQLite sidecars are exempt: a -wal/-shm file is bounded by
// the lifecycle of the database it belongs to, never by its own policy.
func unboundedEntries(entries []corestorage.StorageEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if isSQLiteSidecar(entry, entries) {
			continue
		}
		if entry.Budget != nil || entry.Reclaim != nil {
			continue
		}
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	return names
}

// matchingCodes returns the sorted, deduplicated subset of codes present in
// findings that the caller considers blocking.
func matchingCodes(findings []Finding, codes map[string]struct{}) []string {
	seen := map[string]struct{}{}
	for _, finding := range findings {
		if _, ok := codes[finding.Code]; ok {
			seen[finding.Code] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for code := range seen {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func accountabilityFinding(ac AnalyzerContext, code string, severity Severity, message, remediation string) Finding {
	location := filepath.ToSlash(filepath.Join(".vrooli", "service.json"))
	if ac.Owner != nil && ac.Owner.ManifestPath != "" {
		location = filepath.ToSlash(ac.Owner.ManifestPath)
		if ac.RepoRoot != "" {
			if relative, err := filepath.Rel(ac.RepoRoot, ac.Owner.ManifestPath); err == nil {
				location = filepath.ToSlash(relative)
			}
		}
	}
	owner := ""
	if ac.Owner != nil {
		owner = fmt.Sprintf("%s %q", ac.Owner.Kind, ac.Owner.ID)
	}
	return Finding{
		Code:        code,
		Severity:    severity,
		Title:       "Storage declaration accountability",
		Message:     strings.TrimSpace(fmt.Sprintf("%s: %s", owner, message)),
		Location:    location,
		Remediation: remediation,
		Analyzer:    accountabilityAnalyzer,
	}
}
