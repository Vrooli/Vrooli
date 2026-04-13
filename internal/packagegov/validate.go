package packagegov

import (
	"fmt"
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

	return ValidationReport{Packages: items, Issues: normalizeIssues(issues)}, nil
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
