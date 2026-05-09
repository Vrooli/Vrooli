package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

type Detail struct {
	Scenario scenario.Scenario       `json:"scenario"`
	Runtime  process.ScenarioRuntime `json:"runtime"`
	Details  scenario.RuntimeDetails `json:"details"`
}

type StartResult struct {
	View               ScenarioView            `json:"view"`
	Scenario           scenario.Scenario       `json:"scenario"`
	AllocatedPorts     map[string]int          `json:"allocated_ports"`
	FailedDependencies []string                `json:"failed_dependencies,omitempty"`
	FailedResources    []string                `json:"failed_resources,omitempty"`
	AlreadyRunning     bool                    `json:"already_running,omitempty"`
	Details            scenario.RuntimeDetails `json:"details"`
}

type ResolvedPort struct {
	Detail Detail `json:"detail"`
	Name   string `json:"name"`
	Step   string `json:"step,omitempty"`
	Port   int    `json:"port"`
	URL    string `json:"url,omitempty"`
}

func (s *Service) Inventory() ([]Detail, error) {
	report, err := s.InventoryReport()
	if err != nil {
		return nil, err
	}
	return report.Items, nil
}

func (s *Service) InventoryReport() (InventoryReport, error) {
	discoveryReport, err := scenario.DiscoverReport(s.Root, scenario.SandboxEnvFromEnv())
	if err != nil {
		return InventoryReport{}, err
	}

	valid := make(map[string]struct{}, len(discoveryReport.Items))
	for _, item := range discoveryReport.Items {
		valid[item.Slug] = struct{}{}
	}

	ctx := context.Background()
	mode, err := s.registryReadMode()
	if err != nil {
		return InventoryReport{}, err
	}
	registryDetails := map[string]Detail{}
	if scenarioruntime.ReadEnabled(mode) {
		registryDetails, err = s.registryDetailsByScenario(ctx, discoveryReport.Items)
		if err != nil {
			return InventoryReport{}, err
		}
	}

	details := make([]Detail, 0, len(discoveryReport.Items))
	if !scenarioruntime.StrictReads(mode) || scenarioruntime.HasAllowlistByEnv() {
		running, err := process.DiscoverRunningScenarios(s.Home, func(name string) bool {
			_, ok := valid[name]
			return ok
		})
		if err != nil {
			return InventoryReport{}, err
		}

		runtimes := make(map[string]process.ScenarioRuntime, len(running))
		for _, runtime := range running {
			runtimes[runtime.Name] = runtime
		}

		for _, item := range discoveryReport.Items {
			if detail, ok := registryDetails[item.Slug]; ok {
				details = append(details, detail)
				continue
			}
			if scenarioruntime.StrictReadsForScenario(mode, item.Slug) {
				runtime := process.ScenarioRuntime{Name: item.Slug, Runtime: "N/A"}
				details = append(details, Detail{
					Scenario: item,
					Runtime:  runtime,
					Details:  scenario.DescribeRuntime(item.Manifest, runtime),
				})
				continue
			}
			runtime := runtimes[item.Slug]
			details = append(details, Detail{
				Scenario: item,
				Runtime:  runtime,
				Details:  scenario.DescribeRuntime(item.Manifest, runtime),
			})
		}
	} else {
		for _, item := range discoveryReport.Items {
			if detail, ok := registryDetails[item.Slug]; ok {
				details = append(details, detail)
				continue
			}
			runtime := process.ScenarioRuntime{Name: item.Slug, Runtime: "N/A"}
			details = append(details, Detail{
				Scenario: item,
				Runtime:  runtime,
				Details:  scenario.DescribeRuntime(item.Manifest, runtime),
			})
		}
	}
	sort.Slice(details, func(i, j int) bool { return details[i].Scenario.Slug < details[j].Scenario.Slug })
	return InventoryReport{
		Items:    details,
		Failures: append([]discovery.Failure(nil), discoveryReport.Failures...),
	}, nil
}

func (s *Service) Lookup(name string) (Detail, bool, error) {
	item, err := scenario.Load(s.Root, name, scenario.SandboxEnvFromEnv())
	if err != nil {
		if err == scenario.ErrNotFound {
			return Detail{}, false, nil
		}
		return Detail{}, false, err
	}

	ctx := context.Background()
	registryEnabled, strictRegistry, err := s.registryReadsEnabled(item.Slug)
	if err != nil {
		return Detail{}, true, err
	}
	if registryEnabled {
		detail, ok, err := s.registryDetail(ctx, item)
		if err != nil {
			return Detail{}, true, err
		}
		if ok {
			return detail, true, nil
		}
	}
	if strictRegistry {
		runtime := process.ScenarioRuntime{Name: name, Runtime: "N/A"}
		return Detail{
			Scenario: item,
			Runtime:  runtime,
			Details:  scenario.DescribeRuntime(item.Manifest, runtime),
		}, true, nil
	}

	records, err := process.ReadScenarioRecords(s.Home, name)
	if err != nil {
		return Detail{}, true, err
	}
	runtime := process.SummarizeScenario(name, records)
	return Detail{
		Scenario: item,
		Runtime:  runtime,
		Details:  scenario.DescribeRuntime(item.Manifest, runtime),
	}, true, nil
}

func (s *Service) Detail(name string) (Detail, error) {
	detail, exists, err := s.Lookup(name)
	if err != nil {
		return Detail{}, err
	}
	if !exists {
		return Detail{}, &vroolierr.Error{
			Code:       "scenario_not_found",
			Category:   "Usage",
			Message:    fmt.Sprintf("scenario %q not found", name),
			HTTPStatus: 404,
		}
	}
	return detail, nil
}

func (s *Service) StartDetailed(name string, opts lifecycle.StartOptions) (StartResult, error) {
	runner, err := s.runner()
	if err != nil {
		return StartResult{}, err
	}
	result, err := runner.Start(name, opts)
	if err != nil {
		return StartResult{}, err
	}
	detail, err := s.detailFor(result.Scenario, name)
	if err != nil {
		return StartResult{}, err
	}
	return StartResult{
		View:               s.viewForDetail(detail),
		Scenario:           result.Scenario,
		AllocatedPorts:     result.AllocatedPorts,
		FailedDependencies: append([]string(nil), result.FailedDependencies...),
		FailedResources:    append([]string(nil), result.FailedResources...),
		AlreadyRunning:     result.AlreadyRunning,
		Details:            detail.Details,
	}, nil
}

func (s *Service) RestartDetailed(name string, opts lifecycle.StartOptions) (StartResult, error) {
	runner, err := s.runner()
	if err != nil {
		return StartResult{}, err
	}
	result, err := runner.Restart(name, opts)
	if err != nil {
		return StartResult{}, err
	}
	detail, err := s.detailFor(result.Scenario, name)
	if err != nil {
		return StartResult{}, err
	}
	return StartResult{
		View:               s.viewForDetail(detail),
		Scenario:           result.Scenario,
		AllocatedPorts:     result.AllocatedPorts,
		FailedDependencies: append([]string(nil), result.FailedDependencies...),
		FailedResources:    append([]string(nil), result.FailedResources...),
		AlreadyRunning:     result.AlreadyRunning,
		Details:            detail.Details,
	}, nil
}

func (s *Service) detailFor(item scenario.Scenario, name string) (Detail, error) {
	ctx := context.Background()
	registryEnabled, strictRegistry, err := s.registryReadsEnabled(item.Slug)
	if err != nil {
		return Detail{}, err
	}
	if registryEnabled {
		detail, ok, err := s.registryDetail(ctx, item)
		if err != nil {
			return Detail{}, err
		}
		if ok {
			return detail, nil
		}
	}
	if strictRegistry {
		runtime := process.ScenarioRuntime{Name: name, Runtime: "N/A"}
		return Detail{
			Scenario: item,
			Runtime:  runtime,
			Details:  scenario.DescribeRuntime(item.Manifest, runtime),
		}, nil
	}

	records, err := process.ReadScenarioRecords(s.Home, name)
	if err != nil {
		return Detail{}, err
	}
	runtime := process.SummarizeScenario(name, records)
	return Detail{
		Scenario: item,
		Runtime:  runtime,
		Details:  scenario.DescribeRuntime(item.Manifest, runtime),
	}, nil
}

func (s *Service) ResolvePort(name, requested string) (ResolvedPort, error) {
	detail, err := s.Detail(name)
	if err != nil {
		return ResolvedPort{}, err
	}
	if detail.Details.Status != "running" {
		return ResolvedPort{}, &vroolierr.Error{
			Code:       "scenario_not_running",
			Category:   "Runtime",
			Message:    fmt.Sprintf("scenario %q is not running", name),
			HTTPStatus: 409,
		}
	}

	bindings := detail.Details.PortBindings
	ports := detail.Details.Ports
	resolved, ok := scenario.ResolveRuntimePort(detail.Scenario.Manifest, bindings, ports, requested)
	if !ok && requested == "UI_PORT" {
		resolved, ok = scenario.ResolveRuntimePort(detail.Scenario.Manifest, bindings, ports, "API_PORT")
	}
	if !ok && len(ports) == 1 {
		for onlyKey, onlyPort := range ports {
			resolved = scenario.RuntimePortResolution{Key: onlyKey, Port: onlyPort}
			ok = true
		}
	}
	if !ok {
		return ResolvedPort{}, &vroolierr.Error{
			Code:       "scenario_port_not_found",
			Category:   "Usage",
			Message:    fmt.Sprintf("no port %q found for scenario %q", requested, name),
			HTTPStatus: 404,
		}
	}

	return ResolvedPort{
		Detail: detail,
		Name:   resolved.Key,
		Step:   resolved.Step,
		Port:   resolved.Port,
		URL:    "http://localhost:" + strconv.Itoa(resolved.Port),
	}, nil
}

func (s *Service) viewForDetail(detail Detail) ScenarioView {
	status := detail.Details.Status
	health := any(nil)
	if status == "running" {
		health = detail.Details.Health
	}
	return ScenarioView{
		Name:        detail.Scenario.Slug,
		DisplayName: detail.Scenario.Manifest.Service.DisplayName,
		Description: detail.Scenario.Manifest.Service.Description,
		Tags:        append([]string(nil), detail.Scenario.Manifest.Service.Tags...),
		Status:      status,
		Processes:   detail.Details.Processes,
		StartedAt:   detail.Details.StartedAt,
		Runtime:     detail.Details.Runtime,
		Ports:       copyPortMap(detail.Details.Ports),
		Health:      health,
	}
}

func copyPortMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return map[string]int{}
	}
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
