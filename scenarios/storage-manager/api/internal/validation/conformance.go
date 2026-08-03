package validation

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/retention"
	corestorage "github.com/vrooli/api-core/storage"
)

// storageEntryConformance proves the parts of storage.entries that Vrooli
// owns. It never checks whether a path happens to exist: lazy writers and
// freshly reclaimed caches are valid. It checks whether the source tree has a
// reachable writer for owned entries and reports declaration contradictions.
type storageEntryConformance struct{}

func init() { register(&storageEntryConformance{}) }

func (storageEntryConformance) Name() string { return "storage.entry-conformance" }

func (storageEntryConformance) Applies(ac AnalyzerContext) bool {
	return ac.Owner != nil && len(ac.Owner.StorageEntries) > 0
}

func (storageEntryConformance) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	owner := ac.Owner
	data, err := os.ReadFile(owner.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read owner manifest: %w", err)
	}
	findings := make([]Finding, 0)
	entries := append([]corestorage.StorageEntry(nil), owner.StorageEntries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	for _, entry := range entries {
		sidecar := isSQLiteSidecar(entry, entries)
		if entry.Regenerable && entry.Class == corestorage.ClassData {
			findings = append(findings, storageFinding("STORAGE_ENTRY_CLASS_CONFLICT", SeverityWarning, ac, entry, "regenerable data is contradictory", "Use class cache/state, or mark the entry non-regenerable."))
		}
		if entry.Regenerable && !sidecar && entry.Reclaim == nil && entry.Budget == nil {
			findings = append(findings, storageFinding("STORAGE_ENTRY_UNRECLAIMABLE", SeverityWarning, ac, entry, "regenerable storage has no reclaim command or builtin budget", "Declare reclaim.command, reclaim.pruner=builtin, or a framework-owned budget."))
		}
		if entry.Format == "sqlite" {
			findings = append(findings, sqliteSidecarFindings(ac, entry, entries)...)
		}
		if sidecar {
			continue
		}
		if entry.Rung != corestorage.RungOwned || entry.Path.ByOS != nil {
			continue
		}
		if hasWriterSuppression(ac, entry.Name) {
			continue
		}
		if !hasFrameworkWriter(entry) && !hasReachableWriter(ac, entry) {
			findings = append(findings, storageFinding("STORAGE_ENTRY_NO_WRITER", SeverityWarning, ac, entry, "no Vrooli-owned code path can create or route this entry", "Add a writer, or add a suppression comment explaining a reflective, generated, or shell writer."))
		}
		if entry.Budget != nil && entry.Budget.MaxBytes != "" {
			if budget, parseErr := retention.ParseBytes(entry.Budget.MaxBytes); parseErr == nil {
				if observed := observedBytes(ac, entry); observed > budget {
					findings = append(findings, storageFinding("STORAGE_BUDGET_BELOW_OBSERVED", SeverityError, ac, entry, fmt.Sprintf("observed size %d bytes exceeds max_bytes %s", observed, entry.Budget.MaxBytes), "Raise the ceiling or reclaim data before enforcement."))
				}
			}
		}
	}
	findings = append(findings, retentionConflicts(ac, data)...)
	return findings, nil
}

// hasFrameworkWriter recognizes the canonical framework-owned seam. A
// relative owned path is resolved beneath the owner's class root by
// api-core/storage; the owner does not need to repeat that path literal or
// mkdir call in application code. Absolute and tokenized paths remain on the
// source-evidence path because they may be host-managed or externally owned.
func hasFrameworkWriter(entry corestorage.StorageEntry) bool {
	path := strings.TrimSpace(entry.Path.Value)
	return entry.Rung == corestorage.RungOwned && path != "" && !filepath.IsAbs(path) && !strings.ContainsAny(path, "$%")
}

func isSQLiteSidecar(entry corestorage.StorageEntry, entries []corestorage.StorageEntry) bool {
	path := filepath.ToSlash(entry.Path.Value)
	for _, suffix := range []string{"-wal", "-shm"} {
		if !strings.HasSuffix(path, suffix) {
			continue
		}
		base := strings.TrimSuffix(path, suffix)
		for _, parent := range entries {
			if parent.Format == "sqlite" && filepath.ToSlash(parent.Path.Value) == base {
				return true
			}
		}
	}
	return false
}

func storageFinding(code string, severity Severity, ac AnalyzerContext, entry corestorage.StorageEntry, message, remediation string) Finding {
	location := filepath.ToSlash(filepath.Join(".vrooli", "service.json"))
	if ac.Owner != nil && ac.RepoRoot != "" {
		if rel, err := filepath.Rel(ac.RepoRoot, ac.Owner.ManifestPath); err == nil {
			location = filepath.ToSlash(rel)
		}
	}
	return Finding{Code: code, Severity: severity, Title: entry.Name + " storage declaration", Message: fmt.Sprintf("storage entry %q at %q: %s", entry.Name, entry.Path.Value, message), Location: location, Remediation: remediation, Analyzer: "storage.entry-conformance"}
}

func hasReachableWriter(ac AnalyzerContext, entry corestorage.StorageEntry) bool {
	path := filepath.ToSlash(strings.TrimSpace(entry.Path.Value))
	base := filepath.Base(path)
	if base == "." || base == "/" || base == "" {
		return false
	}
	for _, file := range CollectGoFiles(ac) {
		source := ReadFile(file.AbsPath)
		if source == "" {
			continue
		}
		lines := strings.Split(source, "\n")
		for lineIndex, line := range lines {
			if !strings.Contains(line, base) {
				continue
			}
			// A basename appearing as an identifier, import, comment, or
			// framework vocabulary is not evidence that this entry is written.
			// Require a literal path segment (or the exact declared path) before
			// considering a nearby writer/resolver call.
			if !containsPathLiteral(line, path, base) {
				continue
			}
			// A literal path beside a writer call is direct proof. A resolver
			// call may be a few lines away because options are assembled before
			// the path literal; accept only explicit resolver calls, never a
			// generic `.Path(` or an unrelated write anywhere in the window.
			if containsWriterCall(line) {
				return true
			}
			start, end := lineIndex-4, lineIndex+4
			if start < 0 {
				start = 0
			}
			if end >= len(lines) {
				end = len(lines) - 1
			}
			for _, nearby := range lines[start : end+1] {
				if strings.Contains(nearby, "resolver.Path(") || strings.Contains(nearby, "storage.Path(") || strings.Contains(nearby, ".Resolve(") {
					return true
				}
			}
		}
	}
	return false
}

func containsWriterCall(line string) bool {
	for _, token := range []string{"MkdirAll(", "OpenFile(", "os.Create(", "os.WriteFile(", "os.Rename(", "os.Truncate("} {
		if strings.Contains(line, token) {
			return true
		}
	}
	return false
}

func containsPathLiteral(line, path, base string) bool {
	for _, candidate := range []string{path, base} {
		if candidate == "" {
			continue
		}
		for _, quote := range []string{"\"", "`"} {
			if strings.Contains(line, quote+candidate+quote) {
				return true
			}
		}
	}
	return false
}

func hasWriterSuppression(ac AnalyzerContext, entry string) bool {
	sentinel := "storage-manager:allow-no-writer " + entry
	for _, file := range CollectGoFiles(ac) {
		if strings.Contains(ReadFile(file.AbsPath), sentinel) {
			return true
		}
	}
	return false
}

func sqliteSidecarFindings(ac AnalyzerContext, entry corestorage.StorageEntry, entries []corestorage.StorageEntry) []Finding {
	base := strings.TrimSuffix(filepath.ToSlash(entry.Path.Value), "-wal")
	base = strings.TrimSuffix(base, "-shm")
	seen := map[string]bool{}
	for _, candidate := range entries {
		seen[filepath.ToSlash(candidate.Path.Value)] = true
	}
	var findings []Finding
	for _, suffix := range []string{"-wal", "-shm"} {
		wanted := base + suffix
		if !seen[wanted] {
			findings = append(findings, storageFinding("STORAGE_SQLITE_SIDECAR_UNDECLARED", SeverityWarning, ac, entry, "SQLite sidecar "+wanted+" is not declared", "Add storage entries for both the -wal and -shm sidecars."))
		}
	}
	return findings
}

func observedBytes(ac AnalyzerContext, entry corestorage.StorageEntry) int64 {
	if ac.Owner == nil || ac.RepoRoot == "" {
		return 0
	}
	platform := ac.Platform
	if platform == "" {
		platform = corestorage.HostPlatform()
	}
	path, err := corestorage.ResolveOwnerStoragePath(ac.RepoRoot, *ac.Owner, entry, platform, corestorage.PlatformSeams{})
	if err != nil {
		return 0
	}
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		info, statErr := d.Info()
		if statErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func retentionConflicts(ac AnalyzerContext, manifest []byte) []Finding {
	// ParseManifest validates the retention/storage contract and catches
	// duplicate target paths. The cross-block validator catches durable_data
	// shape disagreements. Keep both mappings warning-level except for the
	// explicit budget-below-observed finding, so adoption does not hard-fail.
	var findings []Finding
	if _, err := retention.ParseManifest(manifest); err != nil {
		findings = append(findings, Finding{Code: "STORAGE_RETENTION_CONFLICT", Severity: SeverityWarning, Title: "Retention declarations conflict", Message: err.Error(), Location: ".vrooli/service.json", Remediation: "Make each retention target and storage declaration describe one physical surface.", Analyzer: "storage.entry-conformance"})
	}
	conflicts, err := retention.ValidateRetentionAgainstDurableData(manifest)
	if err != nil {
		findings = append(findings, Finding{Code: "STORAGE_RETENTION_CONFLICT", Severity: SeverityWarning, Title: "Retention declarations cannot be checked", Message: err.Error(), Location: ".vrooli/service.json", Remediation: "Make the manifest valid JSON before checking retention declarations.", Analyzer: "storage.entry-conformance"})
		return findings
	}
	for _, conflict := range conflicts {
		findings = append(findings, Finding{Code: "STORAGE_RETENTION_CONFLICT", Severity: SeverityWarning, Title: "Retention declarations conflict", Message: conflict.String(), Location: ".vrooli/service.json", Remediation: "Make the durable_data and retention declarations agree about kind and format.", Analyzer: "storage.entry-conformance"})
	}
	return findings
}
