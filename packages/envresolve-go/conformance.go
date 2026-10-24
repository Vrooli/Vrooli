package envresolve

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Scenario string `json:"scenario"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Variable string `json:"variable"`
	Producer string `json:"producer,omitempty"`
	Package  string `json:"package,omitempty"`
	Message  string `json:"message"`
}

// ConformanceScan derives producer findings from repository manifests and Go
// reads. It contains no variable-to-producer table; all names in messages are
// taken from the scanned read or the manifest-derived index.
func ConformanceScan(root string) ([]Finding, error) {
	index, err := Load(root)
	if err != nil {
		return nil, err
	}
	owners, err := resourceOwners(root)
	if err != nil {
		return nil, err
	}
	allow := OSStandardVariables()
	var findings []Finding
	for _, scenarioDir := range scenarioDirs(root) {
		manifestPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
		manifest, err := loadManifestFile(manifestPath)
		if err != nil {
			continue
		}
		scenario := filepath.Base(scenarioDir)
		manifest.Name = scenario
		err = filepath.Walk(scenarioDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info == nil || info.IsDir() || filepath.Ext(path) != ".go" || isTestSupportFile(path) || strings.Contains(path, string(filepath.Separator)+"vendor"+string(filepath.Separator)) {
				return nil
			}
			payload, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			reads, parseErr := FindEnvReads(payload)
			if parseErr != nil {
				return nil
			}
			// Resource fixture managers read host exports only to construct an
			// isolated resource environment. They are producers, not consumers
			// bypassing the shared application seam.
			if isResourceProvisioner(path, payload) {
				return nil
			}
			for _, read := range reads {
				variable := read.Variable
				if _, ok := allow[variable]; ok {
					continue
				}
				if relativeVariable(variable) {
					continue
				}
				line := read.Line
				producers := index.Producers(variable)
				satisfiable, _ := index.Satisfiable(manifest, variable)
				reported := false
				for _, producer := range producers {
					// A scenario's own SCENARIO_* namespace and tunables are
					// relative to that scenario; only a foreign scenario prefix
					// is an address-in-environment finding.
					if producer.Scenario == scenario {
						continue
					}
					switch producer.Kind {
					case ResourceProducer:
						owner := owners[producer.Resource]
						// A declared resource still must be consumed through its
						// owning package. Declaration makes the producer available;
						// it does not authorize raw environment reads.
						if owner == "" && satisfiable {
							continue
						}
						code := "env.undeclared_resource_producer"
						if owner != "" {
							code = "env.package_bypassed"
						}
						findings = append(findings, Finding{
							Code: code, Severity: "ERROR", Scenario: scenario, File: relative(root, path), Line: line, Variable: variable, Producer: producer.Resource, Package: owner,
							Message: resourceMessage(variable, relative(root, path), line, producer.Resource, owner),
						})
						reported = true
					case ScenarioPortProducer, ScenarioAbsoluteSource:
						if !IsScenarioAddressVariable(variable) {
							continue
						}
						if satisfiable {
							continue
						}
						findings = append(findings, Finding{
							Code: "env.address_in_environment", Severity: "ERROR", Scenario: scenario, File: relative(root, path), Line: line, Variable: variable, Producer: producer.Scenario,
							Message: fmt.Sprintf("%s is read in %s:%d but names scenario %s; resolve the peer at call time through discovery and declare the scenario dependency", variable, relative(root, path), line, producer.Scenario),
						})
						reported = true
					}
				}
				if len(producers) == 0 || (!satisfiable && !reported) {
					if dead := index.DeadResource(variable); dead != "" {
						findings = append(findings, Finding{
							Code: "env.dead_producer", Severity: "ERROR", Scenario: scenario, File: relative(root, path), Line: line, Variable: variable, Producer: dead,
							Message: fmt.Sprintf("%s is read in %s:%d but its resource producer %q does not exist in resources/ or the scenario dependency graph", variable, relative(root, path), line, dead),
						})
						continue
					}
					findings = append(findings, Finding{
						Code: "env.producer_absent", Severity: "WARNING", Scenario: scenario, File: relative(root, path), Line: line, Variable: variable,
						Message: fmt.Sprintf("%s is read in %s:%d but no manifest-derived producer exists; move it to scenario configuration or credentials, or delete the read", variable, relative(root, path), line),
					})
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		return strings.Join([]string{findings[i].Scenario, findings[i].File, fmt.Sprint(findings[i].Line), findings[i].Variable}, "\x00") < strings.Join([]string{findings[j].Scenario, findings[j].File, fmt.Sprint(findings[j].Line), findings[j].Variable}, "\x00")
	})
	return findings, nil
}

// isTestSupportFile keeps the conformance phase focused on production Go.
// Several scenarios intentionally keep integration fixtures in ordinary .go
// files (for example test_helpers.go) because those helpers are shared by
// external test packages. They are not runtime consumers of resource
// environment and must not create production seam findings.
func isTestSupportFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, "test_helpers") ||
		strings.Contains(base, "testing_utils") ||
		strings.Contains(base, "test_utils") ||
		strings.Contains(path, string(filepath.Separator)+"testutil"+string(filepath.Separator))
}

func isResourceProvisioner(path string, content []byte) bool {
	return strings.Contains(filepath.ToSlash(path), "/phases/isolation/") &&
		strings.Contains(string(content), "testcontainers")
}

func resourceOwners(root string) (map[string]string, error) {
	owners := map[string]string{}
	paths, err := filepath.Glob(filepath.Join(root, "packages", "*", ".vrooli", "package.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		var doc struct {
			Package struct {
				Name     string `json:"name"`
				Adoption struct {
					Owns []string `json:"owns_resource_environment"`
				} `json:"adoption"`
			} `json:"package"`
		}
		if err := readJSON(path, &doc); err != nil {
			return nil, err
		}
		for _, resource := range doc.Package.Adoption.Owns {
			if previous := owners[resource]; previous == "" || doc.Package.Name < previous {
				owners[resource] = doc.Package.Name
			}
		}
	}
	return owners, nil
}

func loadManifestFile(path string) (Manifest, error) {
	var manifest Manifest
	if err := readJSON(path, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func resourceMessage(variable, file string, line int, resource, owner string) string {
	if owner != "" {
		return fmt.Sprintf("%s is read in %s:%d, is produced by resource %s; package %s owns the resource environment seam", variable, file, line, resource, owner)
	}
	return fmt.Sprintf("%s is read in %s:%d, is produced by resource %s, and that resource is undeclared", variable, file, line, resource)
}

func relative(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(value)
}

func scenarioDirs(root string) []string {
	entries, _ := os.ReadDir(filepath.Join(root, "scenarios"))
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, filepath.Join(root, "scenarios", entry.Name()))
		}
	}
	return result
}

func relativeVariable(variable string) bool {
	return variable == "API_PORT" || variable == "UI_PORT" || strings.HasPrefix(variable, "SCENARIO_") || strings.HasPrefix(variable, "VROOLI_SCENARIO")
}
