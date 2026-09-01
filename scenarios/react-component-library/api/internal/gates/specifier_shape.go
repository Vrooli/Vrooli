package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	for _, path := range librarySourceFiles(filepath.Join(root, "library")) {
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
		for _, occurrence := range libraryPackageSpecifierGateRE.FindAllStringSubmatchIndex(string(raw), -1) {
			name := string(raw[occurrence[2]:occurrence[3]])
			requested := ""
			if occurrence[4] >= 0 {
				requested = string(raw[occurrence[4]:occurrence[5]])
			}
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
			result.Findings = append(result.Findings, Finding{Code: code, AssetID: assetID, File: repoRel(scope.Root, path), Line: lineAt(raw, occurrence[0]), Message: fmt.Sprintf("intra-library import must use a major line, found %s", string(raw[occurrence[0]:occurrence[1]])), Remediation: replacement, DocsRef: "docs/guides/asset-update-flow.md"})
		}
	}
	return nonEmpty(result, "specifier-shape"), nil
}

var libraryPackageSpecifierGateRE = regexp.MustCompile(`@vrooli/react-component-library/([A-Za-z][A-Za-z0-9-]*)(?:/(\d+(?:\.\d+\.\d+)?))?`)
