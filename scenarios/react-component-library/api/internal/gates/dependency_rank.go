package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"react-component-library/internal/librarywalk"
)

func ValidateDependencyRank(scope Scope) (Result, error) {
	root := scope.Root
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	rankByKind := map[string]int{"foundations": 1, "hooks": 2, "services": 2, "adapters": 2, "primitives": 3, "components": 4, "patterns": 5, "navigation": 5, "page-templates": 6}
	type assetRank struct {
		rank int
		kind string
	}
	byLibraryID := map[string]assetRank{}
	manifests, _ := librarywalk.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	for _, manifestPath := range manifests {
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			return Result{}, err
		}
		var manifest struct {
			LibraryID string `json:"libraryId"`
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return Result{}, err
		}
		kind := filepath.Base(filepath.Dir(filepath.Dir(manifestPath)))
		byLibraryID[manifest.LibraryID] = assetRank{rank: rankByKind[kind], kind: kind}
	}
	locks, _ := librarywalk.Glob(filepath.Join(libraryRoot, "*", "*", "versions", "*", "dependencies.json"))
	result := Result{}
	for _, lockPath := range locks {
		if len(scope.Assets) > 0 && !scopeReportsAsset(scope, implementationName(lockPath)) {
			continue
		}
		result.Inspected++
		raw, err := os.ReadFile(lockPath)
		if err != nil {
			return Result{}, err
		}
		var lock struct {
			LibraryID    string `json:"libraryId"`
			Version      string `json:"version"`
			Dependencies []struct {
				LibraryID string `json:"libraryId"`
				Version   string `json:"version"`
				Observed  string `json:"observed"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(raw, &lock); err != nil {
			return Result{}, err
		}
		owner, ownerKnown := byLibraryID[lock.LibraryID]
		if !ownerKnown || owner.rank == 0 {
			continue
		}
		for _, dependency := range lock.Dependencies {
			if dependency.Observed != "" {
				dependency.Version = dependency.Observed
			}
			target, known := byLibraryID[dependency.LibraryID]
			if !known {
				continue
			}
			// Story specimens are preview fixtures, not the implementation
			// composition boundary. Their imports are still retained in the
			// frozen lock for reproducible previews, but must not make a
			// lower-rank library implementation depend on a higher-rank UI
			// component. Keep the synthetic-lock test strict when no source is
			// available; in the live corpus, inspect the authored implementation
			// files and enforce rank only for implementation imports.
			versionDir := filepath.Dir(lockPath)
			if hasImplementationSources(versionDir) && !dependencyImportedByImplementation(versionDir, dependency.LibraryID) {
				continue
			}
			if target.kind != "fixtures" && target.kind != "generators" && target.rank <= owner.rank {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.dependency_rank", AssetID: "__corpus__.dependency-rank", File: repoRel(root, lockPath),
				Message:     fmt.Sprintf("%s@%s (%s rank %d) imports %s@%s (%s rank %d)", lock.LibraryID, lock.Version, owner.kind, owner.rank, dependency.LibraryID, dependency.Version, target.kind, target.rank),
				Remediation: "Invert the dependency, extract a lower-rank seam, or remove the fixture/generator import from the composing asset; do not waive the edge.",
				DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-composition-ranks",
			})
		}
	}
	return nonEmpty(result, "dependency-rank"), nil
}
