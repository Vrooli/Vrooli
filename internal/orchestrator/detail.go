package orchestrator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
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
	items, err := scenario.Discover(s.Root, scenario.SandboxEnvFromEnv())
	if err != nil {
		return nil, err
	}

	valid := make(map[string]struct{}, len(items))
	for _, item := range items {
		valid[item.Slug] = struct{}{}
	}

	running, err := process.DiscoverRunningScenarios(s.Home, func(name string) bool {
		_, ok := valid[name]
		return ok
	})
	if err != nil {
		return nil, err
	}

	runtimes := make(map[string]process.ScenarioRuntime, len(running))
	for _, runtime := range running {
		runtimes[runtime.Name] = runtime
	}

	details := make([]Detail, 0, len(items))
	for _, item := range items {
		runtime := runtimes[item.Slug]
		details = append(details, Detail{
			Scenario: item,
			Runtime:  runtime,
			Details:  scenario.DescribeRuntime(item.Manifest, runtime),
		})
	}
	sort.Slice(details, func(i, j int) bool { return details[i].Scenario.Slug < details[j].Scenario.Slug })
	return details, nil
}

func (s *Service) Lookup(name string) (Detail, bool, error) {
	item, err := scenario.Load(s.Root, name, scenario.SandboxEnvFromEnv())
	if err != nil {
		if err == scenario.ErrNotFound {
			return Detail{}, false, nil
		}
		return Detail{}, false, err
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
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr, s.logger())
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
		AlreadyRunning:     result.AlreadyRunning,
		Details:            detail.Details,
	}, nil
}

func (s *Service) RestartDetailed(name string, opts lifecycle.StartOptions) (StartResult, error) {
	runner, err := lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr, s.logger())
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
		AlreadyRunning:     result.AlreadyRunning,
		Details:            detail.Details,
	}, nil
}

func (s *Service) detailFor(item scenario.Scenario, name string) (Detail, error) {
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
	if detail.Runtime.ProcessCount == 0 {
		return ResolvedPort{}, &vroolierr.Error{
			Code:       "scenario_not_running",
			Category:   "Runtime",
			Message:    fmt.Sprintf("scenario %q is not running", name),
			HTTPStatus: 409,
		}
	}

	bindings := detail.Details.PortBindings
	ports := detail.Details.Ports
	key, portValue, stepName, ok := resolveRequestedPort(detail.Scenario.Manifest, bindings, ports, requested)
	if !ok && requested == "UI_PORT" {
		key, portValue, stepName, ok = resolveRequestedPort(detail.Scenario.Manifest, bindings, ports, "API_PORT")
	}
	if !ok && len(ports) == 1 {
		for onlyKey, onlyPort := range ports {
			key, portValue, ok = onlyKey, onlyPort, true
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
		Name:   key,
		Step:   stepName,
		Port:   portValue,
		URL:    "http://localhost:" + strconv.Itoa(portValue),
	}, nil
}

func resolveRequestedPort(manifest scenario.ServiceManifest, bindings []scenario.RuntimePortBinding, ports map[string]int, requested string) (string, int, string, bool) {
	candidates := []string{requested}
	if envVar := manifest.PortEnvVar(strings.ToLower(strings.TrimSuffix(requested, "_PORT"))); envVar != "" {
		candidates = append(candidates, envVar)
	}
	normalized := strings.ToUpper(strings.TrimSpace(requested))
	if normalized != "" && normalized != requested {
		candidates = append(candidates, normalized)
		if !strings.HasSuffix(normalized, "_PORT") {
			candidates = append(candidates, normalized+"_PORT")
		}
	}

	seen := map[string]struct{}{}
	for _, key := range candidates {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if portValue, ok := ports[key]; ok {
			stepName := ""
			for _, binding := range bindings {
				if binding.Key == key && binding.Port == portValue {
					stepName = binding.Step
					break
				}
			}
			return key, portValue, stepName, true
		}
	}
	return "", 0, "", false
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
