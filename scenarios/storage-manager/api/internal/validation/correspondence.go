package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

func ownerManifestLocation(ac AnalyzerContext) string {
	if ac.Owner == nil || ac.Owner.ManifestPath == "" {
		return ""
	}
	if ac.RepoRoot != "" {
		if rel, err := filepath.Rel(ac.RepoRoot, ac.Owner.ManifestPath); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(ac.Owner.ManifestPath)
}

// storageCorrespondence is the merge gate between a declaration and the
// canonical owner resolver. Legacy path declarations are intentionally
// compared with the class-only form, which makes the historical scenario
// repo-relative defect fail instead of merely being reinterpreted.
type storageCorrespondence struct{}

func init() { register(&storageCorrespondence{}) }

func (storageCorrespondence) Name() string { return "storage.correspondence" }

func (storageCorrespondence) Applies(ac AnalyzerContext) bool {
	return ac.Owner != nil && len(ac.Owner.StorageEntries) > 0
}

func (storageCorrespondence) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	findings := make([]Finding, 0)
	for _, entry := range ac.Owner.StorageEntries {
		if entry.Class == "" {
			// Explicit pinned/byOS locations are the documented exception: there
			// is no class root that can express an upstream-owned system path.
			continue
		}
		if entry.Path.ByOS != nil || (ac.Owner.Kind != corestorage.OwnerScenario && entry.Path.Value != "") {
			// A platform map or a non-scenario explicit path records an
			// upstream/system layout that the class resolver cannot own.
			continue
		}
		// A superseded legacy location is intentionally explicit: it may sit
		// outside the canonical class root so retention can drain it to zero.
		// Its rationale is the review record that prevents an ordinary repo-
		// relative path from using this escape hatch.
		if entry.Path.Value != "" && (strings.HasPrefix(entry.Name, "legacy_") || strings.Contains(strings.ToLower(entry.Rationale), "superseded")) {
			continue
		}
		canonical := entry
		canonical.Path = corestorage.PortablePath{}
		declared, declaredErr := corestorage.ResolveOwnerStoragePath(ac.RepoRoot, *ac.Owner, entry, ac.Platform, corestorage.PlatformSeams{})
		resolved, resolvedErr := corestorage.ResolveOwnerStoragePath(ac.RepoRoot, *ac.Owner, canonical, ac.Platform, corestorage.PlatformSeams{})
		if declaredErr != nil || resolvedErr != nil {
			if _, ok := declaredErr.(*corestorage.NotApplicable); ok {
				continue
			}
			if _, ok := resolvedErr.(*corestorage.NotApplicable); ok {
				continue
			}
			return nil, fmt.Errorf("resolve storage correspondence for %s: declared=%v canonical=%v", entry.Name, declaredErr, resolvedErr)
		}
		if filepath.Clean(declared) == filepath.Clean(resolved) {
			continue
		}
		findings = append(findings, Finding{
			Code:        "STORAGE_PATH_DIVERGENT",
			Severity:    SeverityError,
			Title:       "Storage declaration disagrees with resolver",
			Message:     fmt.Sprintf("storage entry %q resolves to %q but the canonical class resolver returns %q", entry.Name, declared, resolved),
			Location:    ownerManifestLocation(ac),
			Remediation: "Remove the host path and declare the resolver-selected class with an optional relative subpath.",
			Analyzer:    "storage.correspondence",
		})
	}
	return findings, nil
}
