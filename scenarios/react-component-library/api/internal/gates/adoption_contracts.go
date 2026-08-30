package gates

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/database"
)

func validateEvidenceFreshness(scope Scope) (Result, error) {
	root := scope.Root
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(paths)}
	db, err := openEvidenceDatabase(root)
	if err != nil {
		return result, nil
	}
	if db == nil {
		return result, nil
	}
	defer db.Close()
	for _, storyPath := range paths {
		versionDir := filepath.Dir(storyPath)
		version := filepath.Base(versionDir)
		componentDir := filepath.Dir(filepath.Dir(versionDir))
		manifestPath := filepath.Join(componentDir, "component.json")
		manifest, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			continue
		}
		var metadata struct {
			LibraryID string `json:"libraryId"`
		}
		if json.Unmarshal(manifest, &metadata) != nil || metadata.LibraryID == "" {
			continue
		}
		var created string
		queryErr := db.QueryRowContext(context.Background(), `SELECT created_at FROM component_test_reports WHERE root_library_id = ? AND root_version = ? ORDER BY created_at DESC LIMIT 1`, metadata.LibraryID, version).Scan(&created)
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
		createdAt, parseErr := time.Parse(time.RFC3339Nano, created)
		contractInfo, statErr := os.Stat(storyPath)
		if parseErr != nil || statErr != nil || !createdAt.After(contractInfo.ModTime()) {
			result.Findings = append(result.Findings, freshnessFinding(root, storyPath, "newest component test evidence is older than story.json"))
		}
	}
	return nonEmpty(result, "evidence-freshness"), nil
}

func openEvidenceDatabase(root string) (*database.RoutedDB, error) {
	paths := []string{
		filepath.Join(root, "scenarios", "react-component-library", "data", "react-component-library.db"),
		filepath.Join(os.Getenv("HOME"), ".vrooli", "data", "vrooli", "react-component-library", "react-component-library.db"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		db, err := openGateDB(context.Background(), path)
		if err != nil {
			continue
		}
		var present int
		if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'component_test_reports'`).Scan(&present); err == nil && present > 0 {
			return db, nil
		}
		db.Close()
	}
	return nil, nil
}

func freshnessFinding(root, path, reason string) Finding {
	return Finding{
		Code: "catalog.evidence_freshness", AssetID: implementationName(path), File: repoRel(root, path),
		Message: reason, Remediation: "Run the component test sweep for this exact library version after its story contract changes; a stale or missing report cannot certify the current contract.", DocsRef: "docs/internal/TESTING.md",
	}
}
