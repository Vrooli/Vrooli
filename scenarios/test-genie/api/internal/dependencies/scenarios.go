package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ScenarioDependencyChecker validates declared scenario dependencies.
type ScenarioDependencyChecker interface {
	Check(ctx context.Context) ScenarioDependencyResult
}

type ScenarioDependencyResult struct {
	Success      bool
	Error        error
	FailureClass FailureClass
	Remediation  string
	Observations []Observation
	Checked      int
}

type ScenarioStatusFetcher interface {
	ScenarioStatus(ctx context.Context, name string) (*cliv1.ScenarioStatusSingle, error)
}

type scenarioDependencyChecker struct {
	scenarioDir string
	settings    ScenarioDependencySettings
	fetcher     ScenarioStatusFetcher
}

type scenarioDependencyManifest struct {
	Dependencies struct {
		Scenarios map[string]scenarioDependency `json:"scenarios"`
	} `json:"dependencies"`
}

type scenarioDependency struct {
	Enabled       *bool  `json:"enabled"`
	Required      bool   `json:"required"`
	StartupPolicy string `json:"startup_policy"`
}

func NewScenarioDependencyChecker(scenarioDir string, settings ScenarioDependencySettings, fetcher ScenarioStatusFetcher) ScenarioDependencyChecker {
	return &scenarioDependencyChecker{scenarioDir: scenarioDir, settings: settings, fetcher: fetcher}
}

func (c *scenarioDependencyChecker) Check(ctx context.Context) ScenarioDependencyResult {
	if !c.settings.Enabled {
		return ScenarioDependencyResult{
			Success:      true,
			Observations: []Observation{NewSkipObservation("scenario dependency checks disabled via .vrooli/testing.json")},
		}
	}
	dependencies, err := loadScenarioDependencies(c.scenarioDir)
	if err != nil {
		return ScenarioDependencyResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMisconfiguration,
			Remediation:  "Fix .vrooli/service.json so dependencies.scenarios can be read.",
		}
	}
	if len(dependencies) == 0 {
		return ScenarioDependencyResult{
			Success:      true,
			Observations: []Observation{NewInfoObservation("manifest declares no required scenario dependencies")},
		}
	}
	if c.settings.HealthPolicy == "skip" {
		return ScenarioDependencyResult{
			Success:      true,
			Observations: []Observation{NewSkipObservation("scenario dependency health checks skipped via .vrooli/testing.json")},
			Checked:      len(dependencies),
		}
	}
	if c.fetcher == nil {
		observation := NewWarningObservation("scenario dependency status unavailable because no Vrooli CLI client is configured")
		if c.settings.HealthPolicy == "warn" {
			return ScenarioDependencyResult{Success: true, Observations: []Observation{observation}, Checked: len(dependencies)}
		}
		return ScenarioDependencyResult{
			Success:      false,
			Error:        fmt.Errorf("scenario dependency status unavailable"),
			FailureClass: FailureClassMissingDependency,
			Remediation:  "Ensure the vrooli CLI is available so required scenario dependencies can be checked.",
			Observations: []Observation{observation},
			Checked:      len(dependencies),
		}
	}
	var observations []Observation
	var failures []string
	for _, name := range dependencies {
		status, err := c.fetcher.ScenarioStatus(ctx, name)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s (status unavailable: %v)", name, err))
			continue
		}
		item := status.GetScenario()
		if item.GetStatus() != "running" {
			failures = append(failures, fmt.Sprintf("%s (status=%s)", name, emptyAs(item.GetStatus(), "unknown")))
			continue
		}
		healthy, known := scenarioHealthy(item.GetHealthStatus())
		if known && !healthy {
			failures = append(failures, fmt.Sprintf("%s (health=false)", name))
			continue
		}
		if !known {
			observations = append(observations, NewSuccessObservation("scenario dependency running: "+name))
			continue
		}
		observations = append(observations, NewSuccessObservation("scenario dependency healthy: "+name))
	}
	if len(failures) == 0 {
		return ScenarioDependencyResult{Success: true, Observations: observations, Checked: len(dependencies)}
	}
	sort.Strings(failures)
	observation := NewErrorObservation("required_scenario_unhealthy: " + strings.Join(failures, "; "))
	if c.settings.HealthPolicy == "warn" {
		observation = NewWarningObservation("required_scenario_unhealthy: " + strings.Join(failures, "; "))
		return ScenarioDependencyResult{Success: true, Observations: append(observations, observation), Checked: len(dependencies)}
	}
	return ScenarioDependencyResult{
		Success:      false,
		Error:        fmt.Errorf("required scenario dependencies are not ready: %s", strings.Join(failures, "; ")),
		FailureClass: FailureClassMissingDependency,
		Remediation:  "Start or restart the reported dependencies, for example `vrooli scenario start <name>` or `vrooli scenario restart <name>`.",
		Observations: append(observations, observation),
		Checked:      len(dependencies),
	}
}

func loadScenarioDependencies(scenarioDir string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	var manifest scenarioDependencyManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	var names []string
	for name, dep := range manifest.Dependencies.Scenarios {
		if dep.Enabled != nil && !*dep.Enabled {
			continue
		}
		if dep.Required || strings.TrimSpace(dep.StartupPolicy) == "must_start" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func scenarioHealthy(value *structpb.Value) (bool, bool) {
	if value == nil {
		return false, false
	}
	if b, ok := value.GetKind().(*structpb.Value_BoolValue); ok {
		return b.BoolValue, true
	}
	return false, false
}

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

var _ ScenarioDependencyChecker = (*scenarioDependencyChecker)(nil)
