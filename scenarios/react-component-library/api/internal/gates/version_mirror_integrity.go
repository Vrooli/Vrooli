package gates

import (
	"context"
	"fmt"
	"path/filepath"
)

func ValidateVersionMirrorIntegrity(scope Scope) (Result, error) {
	if scope.DB == nil {
		return Result{RunnerError: []Finding{{Code: "catalog.version_mirror_missing", AssetID: "", Message: "version ledger database is unavailable", Remediation: "Run versions doctor and restore the ledger connection."}}}, nil
	}
	rows, err := scope.DB.QueryContext(context.Background(), `SELECT v.source_path, c.library_id, v.version, v.id FROM component_versions v JOIN components c ON c.id=v.component_id WHERE lower(COALESCE(v.presence,''))='evicted' AND lower(v.status)<>'retired'`)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	// The empty evicted set is a valid clean observation. Count the successful
	// ledger query itself so a corpus with no evictions is not mistaken for a
	// broken runner with zero inputs.
	result := Result{Inspected: 1}
	for rows.Next() {
		var sourcePath, libraryID, version, id string
		if err := rows.Scan(&sourcePath, &libraryID, &version, &id); err != nil {
			return Result{}, err
		}
		result.Inspected++
		var count int
		if err := scope.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM component_version_files WHERE version_id=?`, id).Scan(&count); err != nil {
			return Result{}, err
		}
		if count == 0 {
			result.Findings = append(result.Findings, Finding{Code: "catalog.version_mirror_missing", AssetID: implementationName(filepath.Join(scope.Root, "scenarios/react-component-library/library", sourcePath)), File: filepath.ToSlash(filepath.Join("scenarios/react-component-library/library", sourcePath)), Message: fmt.Sprintf("%s@%s has no file mirror rows", libraryID, version), Remediation: "Run `react-component-library versions materialize` or `versions doctor` before retention.", DocsRef: "docs/concepts/ARCHITECTURE.md#version-lifecycle"})
		}
	}
	return nonEmpty(result, "version-mirror-integrity"), rows.Err()
}

// ValidateSpecifierShape enforces D1 for releases created by the governed
// publisher. Historical backfilled releases are exempt because released bytes
// are immutable; the exemption is provenance-derived, never asset-specific.
