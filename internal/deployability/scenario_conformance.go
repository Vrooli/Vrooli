package deployability

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenario"
)

// ManifestRule is one mechanical rule over an authored scenario manifest.
// Keeping the registry here makes the fleet gate and fixture tests exercise
// exactly the same rule set.
type ManifestRule struct {
	Name  string
	Check func(string, scenario.ServiceManifest) []ConformanceFinding
}

// DevelopmentServerPatterns is the single vocabulary of launch patterns that
// cannot be used for a production scenario component. It is exported so
// future builders and conformance consumers extend one table rather than
// scattering literals through rule implementations.
var DevelopmentServerPatterns = []string{
	"vite dev",
	"vite",
	"--watch",
	"nodemon",
	"webpack serve",
	"next dev",
	"run dev",
}

var scenarioManifestRules = []ManifestRule{
	{Name: "known-builder-kind", Check: checkKnownBuilderKinds},
	{Name: "production-ui-artifact", Check: checkProductionUIArtifact},
	{Name: "no-development-server", Check: checkNoDevelopmentServer},
	{Name: "NO_SHELL_ENTRYPOINT", Check: checkNoShellEntrypoint},
}

// ScenarioManifestConformanceRules returns the registered manifest rules in
// stable order. The returned slice is defensive so callers cannot disable a
// rule for the process.
func ScenarioManifestConformanceRules() []ManifestRule {
	return append([]ManifestRule(nil), scenarioManifestRules...)
}

// CheckScenarioManifest applies every registered rule to one manifest path.
func CheckScenarioManifest(path string) ([]ConformanceFinding, error) {
	var manifest scenario.ServiceManifest
	if err := decodeJSON(path, &manifest); err != nil {
		return nil, err
	}
	var findings []ConformanceFinding
	for _, rule := range scenarioManifestRules {
		for _, finding := range rule.Check(path, manifest) {
			finding.Rule = rule.Name
			findings = append(findings, finding)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		return findings[i].Message < findings[j].Message
	})
	return findings, nil
}

// CheckScenarioFleet applies the registered rules to every live scenario
// service manifest under root. The report includes the number of manifests so
// a skipped or unexpectedly added scenario cannot look like a clean fleet.
func CheckScenarioFleet(root string) (ScenarioManifestReport, error) {
	scenariosRoot := filepath.Join(root, repocontractmeta.ScenarioDir)
	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		return ScenarioManifestReport{}, err
	}
	report := ScenarioManifestReport{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(scenariosRoot, entry.Name(), repocontractmeta.ProjectConfigDir, "service.json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return ScenarioManifestReport{}, err
		}
		report.ManifestCount++
		findings, err := CheckScenarioManifest(path)
		if err != nil {
			return ScenarioManifestReport{}, err
		}
		report.Findings = append(report.Findings, findings...)
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].ManifestPath != report.Findings[j].ManifestPath {
			return report.Findings[i].ManifestPath < report.Findings[j].ManifestPath
		}
		return report.Findings[i].Rule < report.Findings[j].Rule
	})
	return report, nil
}

type ScenarioManifestReport struct {
	ManifestCount int                  `json:"manifest_count"`
	Findings      []ConformanceFinding `json:"findings"`
}

func checkProductionUIArtifact(path string, manifest scenario.ServiceManifest) []ConformanceFinding {
	registry := lifecycle.BuilderRegistry()
	var findings []ConformanceFinding
	for name, component := range manifest.Components {
		spec, ok := registry[component.Build.Kind]
		if !ok || strings.TrimSpace(spec.DefaultOutput) == "" {
			continue
		}
		if len(component.Run.Argv) != 0 {
			continue
		}
		findings = append(findings, ConformanceFinding{
			ManifestPath: path,
			Message:      fmt.Sprintf("component %q declares builder %q with production output %q but no run.argv", name, component.Build.Kind, spec.DefaultOutput),
		})
	}
	return findings
}

func checkKnownBuilderKinds(path string, manifest scenario.ServiceManifest) []ConformanceFinding {
	registry := lifecycle.BuilderRegistry()
	var findings []ConformanceFinding
	for name, component := range manifest.Components {
		kind := strings.TrimSpace(component.Build.Kind)
		if kind == "" {
			continue
		}
		spec, ok := registry[kind]
		if ok && !spec.Reserved {
			continue
		}
		message := fmt.Sprintf("component %q declares unknown builder kind %q", name, kind)
		if ok && spec.Reserved {
			message = fmt.Sprintf("component %q declares reserved builder kind %q; no executable builder is registered", name, kind)
		}
		findings = append(findings, ConformanceFinding{
			ManifestPath: path,
			Message:      message,
		})
	}
	return findings
}

func checkNoShellEntrypoint(path string, manifest scenario.ServiceManifest) []ConformanceFinding {
	var findings []ConformanceFinding
	check := func(owner string, argv []string) {
		if len(argv) == 0 {
			return
		}
		first := strings.ToLower(filepath.Base(strings.TrimSpace(argv[0])))
		switch first {
		case "make", "bash", "sh", "zsh", "cmd", "powershell":
			findings = append(findings, ConformanceFinding{ManifestPath: path, Message: fmt.Sprintf("%s uses forbidden shell entrypoint %q", owner, argv[0])})
		}
	}
	for name, component := range manifest.Components {
		check(fmt.Sprintf("component %q run", name), component.Run.Argv)
	}
	for phaseName, phase := range map[string]scenario.Phase{
		"setup": manifest.Lifecycle.Setup,
		"develop": manifest.Lifecycle.Develop,
		"clean": manifest.Lifecycle.Clean,
	} {
		for _, step := range phase.Steps {
			check(fmt.Sprintf("%s step %q", phaseName, step.Name), step.Exec)
		}
	}
	return findings
}

func checkNoDevelopmentServer(path string, manifest scenario.ServiceManifest) []ConformanceFinding {
	var findings []ConformanceFinding
	for name, component := range manifest.Components {
		if matchesDevelopmentServer(component.Run.Argv) {
			findings = append(findings, ConformanceFinding{
				ManifestPath: path,
				Message:      fmt.Sprintf("component %q declares forbidden development server argv %q", name, strings.Join(component.Run.Argv, " ")),
			})
		}
	}
	for _, step := range manifest.Lifecycle.Develop.Steps {
		if matchesDevelopmentServer(step.Exec) {
			findings = append(findings, ConformanceFinding{
				ManifestPath: path,
				Message:      fmt.Sprintf("develop step %q declares forbidden development server argv %q", step.Name, strings.Join(step.Exec, " ")),
			})
		}
	}
	return findings
}

func matchesDevelopmentServer(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	normalized := make([]string, len(argv))
	for i, arg := range argv {
		normalized[i] = strings.ToLower(strings.TrimSpace(arg))
	}
	joined := strings.Join(normalized, " ")
	for _, pattern := range DevelopmentServerPatterns {
		pattern = strings.ToLower(pattern)
		if pattern == "vite" {
			if filepath.Base(normalized[0]) == "vite" && (len(normalized) == 1 || normalized[1] == "dev") {
				return true
			}
			continue
		}
		if strings.Contains(joined, pattern) {
			return true
		}
	}
	return false
}
