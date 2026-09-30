package gates

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	"react-component-library/internal/librarywalk"
)

func validateEvidenceFreshness(scope Scope) (Result, error) {
	root := scope.Root
	paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	db := scope.DB
	if db == nil {
		return result, nil
	}
	for _, storyPath := range paths {
		if len(scope.Assets) > 0 && !scopeReportsAsset(scope, implementationName(storyPath)) {
			continue
		}
		retired, retiredErr := isRetiredVersion(storyPath)
		if retiredErr != nil {
			return Result{}, retiredErr
		}
		if retired {
			continue
		}
		versionDir := filepath.Dir(storyPath)
		version := filepath.Base(versionDir)
		componentDir := filepath.Dir(filepath.Dir(versionDir))
		manifestPath := filepath.Join(componentDir, "component.json")
		manifest, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			continue
		}
		var metadata struct {
			LibraryID    string `json:"libraryId"`
			Supplemental bool   `json:"supplemental"`
		}
		if json.Unmarshal(manifest, &metadata) != nil || metadata.LibraryID == "" {
			continue
		}
		// Supplemental implementations are durable inputs used by the catalog,
		// but they are not catalog assets and do not require component-test
		// evidence. Treating them as evidence-bearing assets creates a false
		// blocking freshness finding for intentionally non-catalog sources.
		if metadata.Supplemental {
			continue
		}
		result.Inspected++
		if scope.Revision == nil {
			result.Findings = append(result.Findings, freshnessFinding(root, storyPath, "asset revision authority is unavailable"))
			continue
		}
		currentRevision, revisionErr := scope.Revision(metadata.LibraryID, version)
		if revisionErr != nil {
			result.Findings = append(result.Findings, freshnessFinding(root, storyPath, "asset revision is unavailable"))
			continue
		}
		var sourceRevision string
		queryErr := db.QueryRowContext(scope.Context, `SELECT source_revision FROM component_test_reports WHERE root_library_id = ? AND root_version = ? ORDER BY created_at DESC LIMIT 1`, metadata.LibraryID, version).Scan(&sourceRevision)
		if queryErr == sql.ErrNoRows {
			result.Findings = append(result.Findings, freshnessFinding(root, storyPath, "no component test report exists for this version"))
			continue
		}
		if queryErr != nil {
			// Older installations may not have the report table yet. Treat the
			// absence as unmeasured evidence, never as a pass.
			result.Findings = append(result.Findings, freshnessFinding(root, storyPath, "component test report table is unavailable"))
			continue
		}
		if sourceRevision == "" || sourceRevision != currentRevision {
			result.Findings = append(result.Findings, freshnessFinding(root, storyPath, "component test evidence is older than the asset's revision"))
		}
	}
	return nonEmpty(result, "evidence-freshness"), nil
}

func freshnessFinding(root, path, reason string) Finding {
	return Finding{
		Code: "catalog.evidence_freshness", AssetID: implementationName(path), File: repoRel(root, path),
		Message: reason, Remediation: "Run the component test sweep for this exact library version after its story contract changes; a stale or missing report cannot certify the current contract.", DocsRef: "docs/internal/TESTING.md",
	}
}
