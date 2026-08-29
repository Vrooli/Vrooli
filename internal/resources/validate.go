package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
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
	if strings.TrimSpace(name) != "" {
		path := filepath.Join(c.Root, "resources", name, "resource.json")
		if data, readErr := os.ReadFile(path); readErr == nil {
			var fields map[string]json.RawMessage
			if json.Unmarshal(data, &fields) == nil {
				if raw, present := fields["driver"]; present {
					var driver string
					if json.Unmarshal(raw, &driver) == nil && !slices.Contains([]string{"managed-service", "external-cli", "cloud-api", "native-cli"}, strings.TrimSpace(driver)) {
						var runtimeFields map[string]json.RawMessage
						_ = json.Unmarshal(data, &runtimeFields)
						if runtime, hasRuntime := runtimeFields["runtime"]; !hasRuntime || string(runtime) == "{}" {
							return ResourceValidationReport{Count: 1, Passed: false, Items: []ResourceValidationItem{{Name: name, ManifestPath: path, Driver: driver, Issues: []ValidationIssue{{Severity: "error", Message: fmt.Sprintf("unknown_driver: driver %q is not in the managed resource archetype enum", driver)}}}}}, nil
						}
					}
				}
			}
		}
	}

	items, err := c.Discover()
	if err != nil {
		message := err.Error()
		if strings.Contains(message, "driver ") && strings.Contains(message, " is invalid") {
			return ResourceValidationReport{
				Count:  1,
				Passed: false,
				Items:  []ResourceValidationItem{{Name: name, ManifestPath: filepath.Join(c.Root, "resources", name, "resource.json"), Issues: []ValidationIssue{{Severity: "error", Message: "unknown_driver: " + message}}}},
			}, nil
		}
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
	rootContract, err := scenario.LoadServiceManifest(filepath.Join(c.Root, ".vrooli", "service.json"))
	if err != nil {
		return ResourceValidationReport{}, err
	}

	portOwners := map[string]string{}
	for _, item := range items {
		if strings.TrimSpace(item.ManifestPath) == "" {
			continue
		}
		manifest, err := manifestpkg.Load(item.ManifestPath)
		if err != nil {
			return ResourceValidationReport{}, err
		}
		entry := ResourceValidationItem{
			Name:         item.Name,
			ManifestPath: item.ManifestPath,
			Driver:       manifest.Driver,
			Issues:       []ValidationIssue{},
		}
		for _, requiredFile := range []string{"resource.json", "README.md"} {
			path := filepath.Join(c.Root, "resources", item.Name, requiredFile)
			if info, statErr := os.Stat(path); statErr != nil || info.IsDir() {
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "missing_required_file: " + requiredFile})
			}
		}
		resourceDir := filepath.Join(c.Root, "resources", item.Name)
		cliDir := filepath.Join(resourceDir, "cli")
		if info, statErr := os.Stat(cliDir); statErr != nil || !info.IsDir() {
			if manifest.CLI != nil && manifest.CLI.Enabled {
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "cli_declared_without_module: cli.enabled requires a cli/ directory"})
			} else {
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "missing_required_dir: cli"})
			}
		} else if entries, readErr := os.ReadDir(cliDir); readErr == nil && len(entries) == 0 {
			entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "empty_required_dir: cli"})
		}
		_, hasTestDir := statDir(resourceDir, "test")
		_, hasTestsDir := statDir(resourceDir, "tests")
		if hasTestDir && hasTestsDir {
			entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "conflicting_test_dir: use either test/ or tests/, not both"})
		}
		if data, readErr := os.ReadFile(item.ManifestPath); readErr == nil {
			var fields map[string]json.RawMessage
			if json.Unmarshal(data, &fields) == nil {
				if _, present := fields["template"]; present {
					entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: "unknown_manifest_field: template is not a resource archetype field"})
				}
			}
		}
		if _, registeredInRoot := rootContract.Dependencies.Resources[item.Name]; !registeredInRoot {
			entry.Issues = append(entry.Issues, ValidationIssue{
				Severity: "error",
				Message:  "resource_absent_from_contract: resource is not declared in .vrooli/service.json",
			})
		}
		for _, issue := range resourceenv.ValidateResourceManifest(c.Root, manifest) {
			entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: issue})
		}
		if usesSingleManifestContract(manifest) {
			for _, relPath := range []string{
				"config/schema.json",
				"config/runtime.json",
				"config/defaults.json",
				"config/exports.sh",
			} {
				path := filepath.Join(c.Root, "resources", item.Name, filepath.FromSlash(relPath))
				if _, err := os.Stat(path); err == nil {
					entry.Issues = append(entry.Issues, ValidationIssue{
						Severity: "error",
						Message:  fmt.Sprintf("deprecated file retained for single-manifest resource: %s", relPath),
					})
				}
			}
		}
		for _, port := range manifest.Ports {
			hostPort := port.Host
			if hostPort <= 0 {
				hostPort = port.Container
			}
			if hostPort <= 0 {
				continue
			}
			key := resourcePortOwnerKey(port, hostPort)
			if owner, exists := portOwners[key]; exists && owner != item.Name {
				message := fmt.Sprintf("host port %s overlaps with resource %s", key, owner)
				entry.Issues = append(entry.Issues, ValidationIssue{Severity: "error", Message: message})
				report.Issues = append(report.Issues, ValidationIssue{Severity: "error", Message: fmt.Sprintf("%s: %s", item.Name, message)})
			} else {
				portOwners[key] = item.Name
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

func resourcePortOwnerKey(port manifestpkg.ResourcePort, hostPort int) string {
	hostIP := strings.TrimSpace(port.HostIP)
	if hostIP == "" {
		hostIP = "*"
	}
	protocol := strings.ToLower(strings.TrimSpace(port.Protocol))
	if protocol == "" {
		protocol = "tcp"
	}
	return fmt.Sprintf("%s:%d/%s", hostIP, hostPort, protocol)
}

func statDir(parent, name string) (os.FileInfo, bool) {
	info, err := os.Stat(filepath.Join(parent, name))
	return info, err == nil && info.IsDir()
}

func usesSingleManifestContract(manifest ResourceManifest) bool {
	version := strings.TrimSpace(manifest.TemplateVersion)
	return version != "" && version != "1"
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
