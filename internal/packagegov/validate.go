package packagegov

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

type ValidationReport struct {
	Packages []Package         `json:"packages"`
	Issues   []ValidationIssue `json:"issues"`
}

func Validate(root string, filter string) (ValidationReport, error) {
	allItems, issues, err := LoadAll(root)
	if err != nil {
		return ValidationReport{}, err
	}

	items := allItems
	if filter != "" {
		if item, ok := FindByName(allItems, filter); ok {
			items = []Package{item}
		} else {
			return ValidationReport{}, fmt.Errorf("package %q not found", filter)
		}
	}

	for _, item := range items {
		report, err := DiscoverDependents(root, item)
		if err != nil {
			return ValidationReport{}, err
		}
		issues = append(issues, report.Issues...)
		for _, dep := range report.Dependents {
			if item.Manifest.Package.Adoption.ScenarioAdoptable {
				if !slices.Contains(item.Manifest.Package.Adoption.AllowedConsumers, dep.ConsumerClass) {
					issues = append(issues, ValidationIssue{
						Severity:    "error",
						Code:        "package-adoption-supported",
						Message:     fmt.Sprintf("%s cannot be consumed as %s", item.Name, dep.ConsumerClass),
						Path:        dep.DependencyFile,
						PackageName: item.Name,
					})
				}
				if !slices.Contains(item.Manifest.Package.Adoption.AdoptionModes, dep.AdoptionMode) {
					issues = append(issues, ValidationIssue{
						Severity:    "error",
						Code:        "package-adoption-mode-valid",
						Message:     fmt.Sprintf("%s does not allow %s adoption", item.Name, dep.AdoptionMode),
						Path:        dep.DependencyFile,
						PackageName: item.Name,
					})
				}
			} else {
				issues = append(issues, ValidationIssue{
					Severity:    "error",
					Code:        "package-adoption-supported",
					Message:     fmt.Sprintf("%s is not scenario-adoptable", item.Name),
					Path:        dep.DependencyFile,
					PackageName: item.Name,
				})
			}
		}
	}

	// Resolve the governed-module map from the FULL discovered set, not the
	// filtered output. A single-package filter (e.g. `validate cli-core`) must
	// still be able to recognize that a required module like proto is a governed
	// local package; otherwise a leaf's forbidden dependency goes unflagged when
	// validation is scoped to that leaf alone.
	issues = append(issues, validateLeafGoPackageDependencies(items, allItems)...)

	return ValidationReport{Packages: items, Issues: normalizeIssues(issues)}, nil
}

var leafSharedGoPackageAllowedDeps = map[string]map[string]struct{}{
	"cli-core": {
		"repo-contract-go": {},
	},
	"repo-contract-go": {},
}

// validateLeafGoPackageDependencies flags governed leaf Go packages (the keys of
// leafSharedGoPackageAllowedDeps) that require a non-allowlisted governed local
// module. leaves is the (possibly filtered) set of packages to check; allItems
// is the full discovered set used to resolve which required modules are governed
// local packages — it must be the unfiltered set so a single-package filter does
// not blind the resolver to the dependency being required.
func validateLeafGoPackageDependencies(leaves []Package, allItems []Package) []ValidationIssue {
	if len(leaves) == 0 {
		return nil
	}

	moduleToPackage := make(map[string]Package)
	for _, item := range allItems {
		for _, id := range packageModuleIdentifiers(item) {
			moduleToPackage[id] = item
		}
	}

	var issues []ValidationIssue
	for _, item := range leaves {
		allowedDeps, ok := leafSharedGoPackageAllowedDeps[item.Name]
		if !ok {
			continue
		}
		goModPath := filepath.Join(item.RootPath, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "package-go-module-read-failed",
				Message:     fmt.Sprintf("failed to read %s go.mod: %v", item.Name, err),
				Path:        goModPath,
				PackageName: item.Name,
			})
			continue
		}

		mod := parseGoMod(string(data))
		for module := range mod.requires {
			dep, ok := moduleToPackage[module]
			if !ok || dep.Name == item.Name {
				continue
			}
			if _, allowed := allowedDeps[dep.Name]; allowed {
				continue
			}
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "package-go-leaf-local-dependency",
				Message:     fmt.Sprintf("%s is governed as a leaf Go package and must not require local package %s", item.Name, dep.Name),
				Path:        goModPath,
				PackageName: item.Name,
			})
		}
	}

	return issues
}

func normalizeIssues(issues []ValidationIssue) []ValidationIssue {
	out := make([]ValidationIssue, 0, len(issues))
	seen := make(map[string]struct{}, len(issues))
	for _, issue := range issues {
		issue.Path = filepath.Clean(issue.Path)
		key := issue.Severity + "\x00" + issue.Code + "\x00" + issue.Path + "\x00" + issue.Message + "\x00" + issue.PackageName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}
