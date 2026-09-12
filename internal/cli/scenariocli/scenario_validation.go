package scenariocli

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/deployability"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type ValidateRequest struct {
	JSON bool
}

type ManifestValidationIssue struct {
	Scenario string `json:"scenario"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

type SupervisionIntentCounts struct {
	MustStart int `json:"must_start"`
	TryStart  int `json:"try_start"`
	Ignore    int `json:"ignore"`
}

type ManifestValidationReport struct {
	Passed              bool                      `json:"passed"`
	ManifestCount       int                       `json:"manifest_count"`
	DependencyEdgeCount int                       `json:"dependency_edge_count"`
	IntentCounts        SupervisionIntentCounts   `json:"supervision_intent_counts"`
	Issues              []ManifestValidationIssue `json:"issues"`
}

type ValidateResponse struct {
	Success bool                     `json:"success"`
	Report  ManifestValidationReport `json:"report"`
}

func ParseValidateRequest(globalsJSON bool, args []string) (ValidateRequest, error) {
	spec := commandSpec(CommandValidate)
	parsed, err := commandtree.ParseArgs("scenario validate", commandHelpText(CommandValidate), spec.Args, args)
	if err != nil {
		return ValidateRequest{}, err
	}
	return ValidateRequest{JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

// ValidateScenarioManifests loads every manifest through the canonical parser,
// then proves every edge carries an explicit policy and resolves to one intent.
func ValidateScenarioManifests(root string) ManifestValidationReport {
	report := ManifestValidationReport{Passed: true, Issues: []ManifestValidationIssue{}}
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		report.Passed = false
		report.Issues = append(report.Issues, ManifestValidationIssue{Path: root, Message: err.Error()})
		return report
	}
	sort.Strings(paths)
	for _, path := range paths {
		report.ManifestCount++
		name := filepath.Base(filepath.Dir(filepath.Dir(path)))
		manifest, err := scenariomodel.ReadService(path)
		if err != nil {
			report.Passed = false
			report.Issues = append(report.Issues, ManifestValidationIssue{Scenario: name, Path: path, Message: err.Error()})
			continue
		}
		validateDependencyIntents(&report, name, path, "resources", manifest.Dependencies.Resources)
		validateDependencyIntents(&report, name, path, "scenarios", manifest.Dependencies.Scenarios)
	}
	// Keep lifecycle builder and entrypoint conformance on the canonical
	// `scenario validate` path. JSON-schema validation alone accepts reserved
	// builder vocabulary, which would otherwise fail later during setup.
	if conformance, err := deployability.CheckScenarioFleet(root); err != nil {
		report.Passed = false
		report.Issues = append(report.Issues, ManifestValidationIssue{Path: root, Message: "scenario conformance: " + err.Error()})
	} else {
		for _, finding := range conformance.Findings {
			report.Passed = false
			report.Issues = append(report.Issues, ManifestValidationIssue{Path: finding.ManifestPath, Message: finding.Rule + ": " + finding.Message})
		}
	}
	return report
}

func validateDependencyIntents(report *ManifestValidationReport, scenarioName, path, kind string, dependencies map[string]scenariomodel.Dependency) {
	names := make([]string, 0, len(dependencies))
	for name := range dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		dependency := dependencies[name]
		report.DependencyEdgeCount++
		if dependency.StartupPolicy == "" {
			report.Passed = false
			report.Issues = append(report.Issues, ManifestValidationIssue{Scenario: scenarioName, Path: path, Message: fmt.Sprintf("dependencies.%s.%s must declare startup_policy explicitly", kind, name)})
			continue
		}
		switch dependency.SupervisionIntent() {
		case scenariomodel.DependencyStartupPolicyMustStart:
			report.IntentCounts.MustStart++
		case scenariomodel.DependencyStartupPolicyTryStart:
			report.IntentCounts.TryStart++
		case scenariomodel.DependencyStartupPolicyIgnore:
			report.IntentCounts.Ignore++
		default:
			report.Passed = false
			report.Issues = append(report.Issues, ManifestValidationIssue{Scenario: scenarioName, Path: path, Message: fmt.Sprintf("dependencies.%s.%s resolves to no valid supervision intent", kind, name)})
		}
	}
}

func RenderValidateResponse(w io.Writer, format cliout.Format, resp ValidateResponse) error {
	resp.Success = resp.Report.Passed
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		status := "passed"
		if !resp.Report.Passed {
			status = "failed"
		}
		if _, err := fmt.Fprintf(w, "Scenario manifest validation %s: %d manifests, %d dependency edges\n", status, resp.Report.ManifestCount, resp.Report.DependencyEdgeCount); err != nil {
			return err
		}
		for _, issue := range resp.Report.Issues {
			if _, err := fmt.Fprintf(w, "- %s: %s\n", issue.Path, issue.Message); err != nil {
				return err
			}
		}
		return nil
	})
}
