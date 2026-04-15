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
	items, issues, err := LoadAll(root)
	if err != nil {
		return ValidationReport{}, err
	}

	if filter != "" {
		if item, ok := FindByName(items, filter); ok {
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

	issues = append(issues, validateLeafGoPackageDependencies(items)...)

	return ValidationReport{Packages: items, Issues: normalizeIssues(issues)}, nil
}

var leafSharedGoPackageAllowedDeps = map[string]map[string]struct{}{
	"cli-core": {
		"repo-contract-go": {},
	},
	"repo-contract-go": {},
}

func validateLeafGoPackageDependencies(items []Package) []ValidationIssue {
	if len(items) == 0 {
		return nil
	}

	moduleToPackage := make(map[string]Package)
	for _, item := range items {
		for _, id := range packageModuleIdentifiers(item) {
			moduleToPackage[id] = item
		}
	}

	var issues []ValidationIssue
	for _, item := range items {
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
