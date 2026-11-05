package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/libspec"
)

func ValidateSpecifierShape(scope Scope) (Result, error) {
	root := filepath.Join(scope.Root, "scenarios", "react-component-library")
	provenance := map[string]bool{}
	var ledger struct {
		Entries []struct {
			LibraryID, Version string
			Backfilled         bool `json:"backfilled"`
		} `json:"entries"`
	}
	if raw, err := os.ReadFile(filepath.Join(root, "library", "release-provenance.json")); err == nil {
		_ = json.Unmarshal(raw, &ledger)
		for _, entry := range ledger.Entries {
			if entry.Backfilled {
				provenance[entry.LibraryID+"@"+entry.Version] = true
			}
		}
	}
	result := Result{}
	versionRE := regexp.MustCompile(`/versions/([^/]+)/`)
	for _, path := range librarySourceFiles(filepath.Join(root, "library"), scope) {
		match := versionRE.FindStringSubmatch(filepath.ToSlash(path))
		if len(match) != 2 {
			continue
		}
		version := match[1]
		if strings.Contains(version, "-") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		assetID := implementationName(path)
		for _, specifier := range libspec.ParseAll(string(raw)) {
			name := specifier.Name
			requested := specifier.Selector
			if provenance["react-component-library:"+name+"@"+version] {
				continue
			}
			code, replacement := "catalog.specifier_bare", "use the dependency major line"
			if strings.Contains(requested, ".") {
				code, replacement = "catalog.specifier_exact_pin", "use @vrooli/react-component-library/"+name+"/<major>"
			}
			if requested != "" && !strings.Contains(requested, ".") {
				continue
			}
			result.Findings = append(result.Findings, Finding{Code: code, AssetID: assetID, File: repoRel(scope.Root, path), Message: fmt.Sprintf("intra-library import must use a major line, found %s", libspec.Prefix+name+selectorSuffix(requested)), Remediation: replacement, DocsRef: "docs/guides/asset-update-flow.md"})
		}
	}
	return nonEmpty(result, "specifier-shape"), nil
}

func selectorSuffix(selector string) string {
	if selector == "" {
		return ""
	}
	return "/" + selector
}
