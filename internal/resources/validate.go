package resources

import (
	"fmt"
	"sort"
	"strings"

	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	"github.com/vrooli/vrooli/internal/scenario"
)

type ValidationIssue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type ResourceValidationItem struct {
	Name         string            `json:"name"`
	ManifestPath string            `json:"manifest_path,omitempty"`
	Driver       string            `json:"driver,omitempty"`
	Issues       []ValidationIssue `json:"issues,omitempty"`
}

type ResourceValidationReport struct {
	Count  int                      `json:"count"`
	Passed bool                     `json:"passed"`
	Items  []ResourceValidationItem `json:"items,omitempty"`
	Issues []ValidationIssue        `json:"issues,omitempty"`
}

func (c *Controller) ValidateResources(name string) (ResourceValidationReport, error) {
	report := ResourceValidationReport{
		Passed: true,
		Items:  []ResourceValidationItem{},
		Issues: []ValidationIssue{},
	}

	items, err := c.Discover()
	if err != nil {
		return ResourceValidationReport{}, err
	}
	if strings.TrimSpace(name) != "" {
		filtered := items[:0]
		for _, item := range items {
			if item.Name == name {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	report.Count = len(items)

	portOwners := map[int]string{}
	for _, item := range items {
		if strings.TrimSpace(item.ManifestPath) == "" {
			continue
		}
		manifest, err := c.loadResourceManifest(item.ManifestPath)
		if err != nil {
			return ResourceValidationReport{}, err
		}
		entry := ResourceValidationItem{
			Name:         item.Name,
			ManifestPath: item.ManifestPath,
			Driver:       manifest.Driver,
			Issues:       []ValidationIssue{},
		}
		for _, issue := range resourceenv.ValidateResourceManifest(c.Root, manifest) {
			entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: issue})
		}
		for _, port := range manifest.Ports {
			hostPort := port.Host
			if hostPort <= 0 {
				hostPort = port.Container
			}
			if hostPort <= 0 {
				continue
			}
			if owner, exists := portOwners[hostPort]; exists && owner != item.Name {
				message := fmt.Sprintf("host port %d overlaps with resource %s", hostPort, owner)
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: message})
				report.Issues = append(report.Issues, ValidationIssue{Severity: "error", Message: fmt.Sprintf("%s: %s", item.Name, message)})
			} else {
				portOwners[hostPort] = item.Name
			}
		}
		if len(entry.Issues) > 0 {
			report.Passed = false
		}
		report.Items = append(report.Items, entry)
	}
	sort.Slice(report.Items, func(i, j int) bool {
		return report.Items[i].Name < report.Items[j].Name
	})
	if strings.TrimSpace(name) != "" && len(report.Items) == 0 {
		return ResourceValidationReport{}, fmt.Errorf("resource %s not found", name)
	}
	return report, nil
}

type ScenarioEnvValidationReport struct {
	Scenario        string                       `json:"scenario"`
	Values          map[string]string            `json:"values"`
	Issues          []ValidationIssue            `json:"issues,omitempty"`
	ResourceReports []resourceenv.ResourceReport `json:"resource_reports,omitempty"`
	Passed          bool                         `json:"passed"`
}

func (c *Controller) ValidateScenarioEnvironment(name string) (ScenarioEnvValidationReport, error) {
	item, err := scenario.Load(c.Root, name, scenario.SandboxEnv{})
	if err != nil {
		return ScenarioEnvValidationReport{}, err
	}
	resolution, issues, err := resourceenv.ValidateScenario(c.Root, c.Home, item.Slug, item.Manifest)
	if err != nil {
		return ScenarioEnvValidationReport{}, err
	}
	report := ScenarioEnvValidationReport{
		Scenario:        item.Slug,
		Values:          resolution.Values,
		ResourceReports: resolution.Resources,
		Passed:          len(issues) == 0,
	}
	for _, warning := range resolution.Warnings {
		report.Issues = append(report.Issues, ValidationIssue{Severity: "warning", Message: warning})
	}
	for _, issue := range issues {
		report.Issues = append(report.Issues, ValidationIssue{Severity: "error", Message: issue})
	}
	return report, nil
}
