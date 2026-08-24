package packagegov

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

type dependencyInventory struct {
	root    string
	reports map[string]DiscoveryReport
}

func buildDependencyInventory(root string, packages []Package) (dependencyInventory, error) {
	inv := dependencyInventory{
		root:    root,
		reports: make(map[string]DiscoveryReport, len(packages)),
	}
	if len(packages) == 0 {
		return inv, nil
	}
	for _, pkg := range packages {
		inv.reports[pkg.Name] = DiscoveryReport{}
	}

	identifiers := packageIdentifierIndex(packages)
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return dependencyInventory{}, err
	}
	scenarioRoot, err := contract.TopLevelDir(root, "scenarios")
	if err != nil {
		return dependencyInventory{}, err
	}
	templateRoot := repocontract.ScenarioTemplateRoot(root)
	resourceRoot, err := contract.TopLevelDir(root, "resources")
	if err != nil {
		return dependencyInventory{}, err
	}

	for _, target := range []struct {
		root  string
		scope consumerScope
	}{
		{root: scenarioRoot, scope: scopeScenario},
		{root: templateRoot, scope: scopeTemplate},
		{root: resourceRoot, scope: scopeResource},
	} {
		if err := walkPackageJSONs(target.root, func(path string) error {
			return inv.scanPackageJSON(path, target.scope, identifiers)
		}); err != nil {
			return dependencyInventory{}, err
		}
		if err := walkGoMods(target.root, func(path string) error {
			return inv.scanGoMod(path, target.scope, identifiers)
		}); err != nil {
			return dependencyInventory{}, err
		}
	}

	for name, report := range inv.reports {
		sort.Slice(report.Dependents, func(i, j int) bool {
			if report.Dependents[i].ConsumerName == report.Dependents[j].ConsumerName {
				return report.Dependents[i].DependencyFile < report.Dependents[j].DependencyFile
			}
			return report.Dependents[i].ConsumerName < report.Dependents[j].ConsumerName
		})
		report.Issues = normalizeIssues(report.Issues)
		inv.reports[name] = report
	}
	return inv, nil
}

func packageIdentifierIndex(packages []Package) map[string][]Package {
	index := make(map[string][]Package)
	for _, pkg := range packages {
		for _, id := range packageModuleIdentifiers(pkg) {
			index[id] = append(index[id], pkg)
		}
	}
	return index
}

func (inv dependencyInventory) reportFor(pkg Package) DiscoveryReport {
	report := inv.reports[pkg.Name]
	report.Dependents = append([]Dependent(nil), report.Dependents...)
	report.Issues = append([]ValidationIssue(nil), report.Issues...)
	return report
}

func (inv dependencyInventory) scanPackageJSON(path string, scope consumerScope, identifiers map[string][]Package) error {
	manifest, err := readPackageJSON(path)
	if err != nil {
		return err
	}
	name, class := classifyConsumer(inv.root, path, scope)
	consumerPath := consumerRootFromFile(inv.root, path, scope)
	seenPackages := make(map[string]Package)
	for _, bucket := range []map[string]string{
		manifest.Dependencies,
		manifest.DevDependencies,
		manifest.PeerDependencies,
		manifest.OptionalDependencies,
	} {
		for id, version := range bucket {
			for _, pkg := range identifiers[strings.TrimSpace(id)] {
				seenPackages[pkg.Name] = pkg
				report := inv.reports[pkg.Name]
				report.Dependents = append(report.Dependents, Dependent{
					PackageName:      pkg.Name,
					ConsumerName:     name,
					ConsumerPath:     consumerPath,
					ConsumerClass:    class,
					AdoptionMode:     classifyPackageJSONAdoption(pkg, id, version),
					DependencyFile:   filepath.Clean(path),
					DependencyTarget: id,
					Version:          version,
				})
				if scope == scopeScenario && strings.TrimSpace(version) == "workspace:*" {
					report.Issues = append(report.Issues, ValidationIssue{
						Severity:    "error",
						Code:        "package-no-workspace-deps",
						Message:     fmt.Sprintf("real scenario %q uses workspace:* for shared package adoption", name),
						Path:        path,
						PackageName: pkg.Name,
					})
				}
				inv.reports[pkg.Name] = report
			}
		}
	}
	if postinstallTouchesSharedPackages(manifest.Scripts["postinstall"]) {
		if len(seenPackages) == 0 {
			for _, packages := range identifiers {
				for _, pkg := range packages {
					seenPackages[pkg.Name] = pkg
				}
			}
		}
		for _, pkg := range seenPackages {
			report := inv.reports[pkg.Name]
			report.Issues = append(report.Issues, ValidationIssue{
				Severity:    "error",
				Code:        "package-no-unauthorized-postinstall",
				Message:     fmt.Sprintf("consumer %q still uses postinstall shared-package propagation", name),
				Path:        path,
				PackageName: pkg.Name,
			})
			inv.reports[pkg.Name] = report
		}
	}
	return nil
}

func (inv dependencyInventory) scanGoMod(path string, scope consumerScope, identifiers map[string][]Package) error {
	mod, err := readGoMod(path)
	if err != nil {
		return err
	}
	name, class := classifyGoConsumer(inv.root, path, scope)
	consumerPath := consumerRootFromFile(inv.root, path, scope)
	candidateIDs := make(map[string]struct{})
	for id := range mod.requires {
		candidateIDs[id] = struct{}{}
	}
	for id := range mod.replaces {
		candidateIDs[id] = struct{}{}
	}
	for id := range candidateIDs {
		for _, pkg := range identifiers[id] {
			dep := mod.dependencyFor(path, inv.root, id)
			if !dep.Present {
				continue
			}
			report := inv.reports[pkg.Name]
			if !dep.HasGovernedReplace {
				report.Issues = append(report.Issues, ValidationIssue{
					Severity:    "error",
					Code:        "package-go-module-replace-required",
					Message:     fmt.Sprintf("%s requires a governed local replace for %s", name, id),
					Path:        filepath.Clean(path),
					PackageName: pkg.Name,
				})
			}
			report.Dependents = append(report.Dependents, Dependent{
				PackageName:      pkg.Name,
				ConsumerName:     name,
				ConsumerPath:     consumerPath,
				ConsumerClass:    class,
				AdoptionMode:     ModeGoModuleReplace,
				DependencyFile:   filepath.Clean(path),
				DependencyTarget: dep.Target,
				Version:          dep.Version,
			})
			inv.reports[pkg.Name] = report
		}
	}
	return nil
}
