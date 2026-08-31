package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ValidateProvenanceStamp(scope Scope) (Result, error) {
	root := scope.Root
	var paths []string
	for _, base := range []string{filepath.Join(root, "scenarios", "react-component-library", "library"), filepath.Join(root, "scenarios", "react-component-library", "ui")} {
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return Result{}, err
		}
	}
	sort.Strings(paths)
	result := Result{Inspected: len(paths)}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		match := componentSourceRE.FindStringSubmatch(string(data))
		if len(match) < 2 {
			continue
		}
		stamp := strings.TrimSpace(match[1])
		key := stamp[strings.LastIndexAny(stamp, ".:")+1:]
		if pathWithin(filepath.Join(root, "scenarios", "react-component-library", "library"), path) {
			owner := implementationName(path)
			libraryID, catalogID := libraryManifestIdentities(path)
			valid := strings.EqualFold(stamp, libraryID) || strings.EqualFold(stamp, catalogID)
			if !valid && owner != "" {
				valid = strings.EqualFold(strings.TrimPrefix(owner, "react-component-library:"), key) || strings.Contains(strings.ToLower(owner), strings.ToLower(key))
			}
			if !valid {
				result.Findings = append(result.Findings, Finding{Code: "catalog.provenance-stamp", AssetID: owner, File: repoRel(root, path), Line: lineOf(data, match[0]), Message: fmt.Sprintf("component source stamp %q does not identify its owning library asset", stamp), Remediation: "Use the owning component's stable catalog identity in @vrooliComponentSource.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-provenance"})
			}
			continue
		}
		if !strings.Contains(strings.ToLower(string(data)), strings.ToLower(key)) {
			result.Findings = append(result.Findings, Finding{Code: "catalog.provenance-stamp", AssetID: stamp, File: repoRel(root, path), Line: lineOf(data, match[0]), Message: fmt.Sprintf("component source stamp %q is not imported or rendered by this file", stamp), Remediation: "Change the stamp to the library asset actually imported or rendered by this file.", DocsRef: "docs/concepts/ARCHITECTURE.md#catalog-provenance"})
		}
	}
	return nonEmpty(result, "provenance-stamp"), nil
}
