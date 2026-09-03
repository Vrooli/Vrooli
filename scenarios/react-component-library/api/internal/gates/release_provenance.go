package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/librarywalk"
)

func ValidateReleaseProvenance(scope Scope) (Result, error) {
	root := scope.Root
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	raw, err := os.ReadFile(filepath.Join(libraryRoot, "release-provenance.json"))
	if err != nil {
		return Result{RunnerError: []Finding{{
			Code: "catalog.release_provenance_unavailable", AssetID: "__corpus__.release-provenance",
			Message:     "release provenance ledger is unavailable: " + err.Error(),
			Remediation: "Restore library/release-provenance.json; the bypass-prevention gate cannot pass without its durable ledger.",
			DocsRef:     "docs/guides/asset-update-flow.md",
		}}}, nil
	}
	var ledger struct {
		Entries []struct {
			LibraryID    string `json:"libraryId"`
			Version      string `json:"version"`
			DraftVersion string `json:"draftVersion"`
			PublishedAt  string `json:"publishedAt"`
			Backfilled   bool   `json:"backfilled"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		return Result{}, fmt.Errorf("decode release provenance ledger: %w", err)
	}
	recorded := map[string]bool{}
	for _, entry := range ledger.Entries {
		if strings.TrimSpace(entry.PublishedAt) == "" || (!entry.Backfilled && (!strings.Contains(entry.DraftVersion, "-") || strings.TrimSpace(entry.DraftVersion) == "")) {
			continue
		}
		recorded[entry.LibraryID+"@"+entry.Version] = true
	}
	result := Result{}
	manifests, err := librarywalk.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	for _, manifestPath := range manifests {
		manifestRaw, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		var manifest struct {
			LibraryID string `json:"libraryId"`
		}
		if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
			return Result{}, err
		}
		entries, readErr := os.ReadDir(filepath.Join(filepath.Dir(manifestPath), "versions"))
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(entry.Name()) {
				continue
			}
			result.Inspected++
			if recorded[manifest.LibraryID+"@"+entry.Name()] {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.release_provenance_missing", AssetID: manifest.LibraryID,
				File:        repoRel(root, filepath.Join(filepath.Dir(manifestPath), "versions", entry.Name())),
				Message:     fmt.Sprintf("released directory %s@%s has no valid publish or backfill record", manifest.LibraryID, entry.Name()),
				Remediation: "Remove the bypass release and publish it through `react-component-library components draft publish`; never backfill a newly-created release.",
				DocsRef:     "docs/guides/asset-update-flow.md",
			})
		}
	}
	return nonEmpty(result, "release-provenance"), nil
}

// ValidateDependencyRank enforces the composition direction over generated
// per-version locks, which are the durable projection of real source imports.
