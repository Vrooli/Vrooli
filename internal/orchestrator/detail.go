package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/hostlifecycle"
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
	// StartOperation is the latest start/restart operation record for the
	// instance (in-flight progress or last terminal outcome); nil when none
	// exists or the runtime registry is unavailable. Populated on the
	// single-scenario paths (Lookup/Detail), not fleet listings.
	StartOperation *lifecycle.StartOperationView `json:"start_operation,omitempty"`
}

type StartResult struct {
	View               ScenarioView            `json:"view"`
	Scenario           scenario.Scenario       `json:"scenario"`
	AllocatedPorts     map[string]int          `json:"allocated_ports"`
	FailedDependencies []string                `json:"failed_dependencies,omitempty"`
	FailedResources    []string                `json:"failed_resources,omitempty"`
	AlreadyRunning     bool                    `json:"already_running,omitempty"`
	Details            scenario.RuntimeDetails `json:"details"`
	// Verdict is the lifecycle health verdict of the start (healthy |
	// degraded | running); it backs the exit-code contract.
	Verdict string `json:"verdict,omitempty"`
	// StartOperation is the durable operation record of this start.
	StartOperation *lifecycle.StartOperationView `json:"start_operation,omitempty"`
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

	ctx := context.Background()
	registryDetails, err := s.registryDetailsByScenario(ctx, discoveryReport.Items)
	if err != nil {
		return InventoryReport{}, err
	}

	details := make([]Detail, 0, len(discoveryReport.Items))
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
	sort.Slice(details, func(i, j int) bool { return details[i].Scenario.Slug < details[j].Scenario.Slug })
	return InventoryReport{
		Items:    details,
		Failures: append([]discovery.Failure(nil), discoveryReport.Failures...),
	}, nil
}

func (s *Service) Lookup(name string) (Detail, bool, error) {
	// Resolve an optional "@variant" suffix through the single shared parser
	// (§1a) so every reader that bottoms out in Lookup — Detail, ResolvePort,
	// status/port/info — selects the addressed instance. A bare name (no "@")
	// normalizes to the live variant, so existing callers are unchanged.
	key, err := scenarioruntime.ParseInstanceKey(name, "")
	if err != nil {
		return Detail{}, false, err
	}
	item, err := scenario.Load(s.Root, key.Scenario, scenario.SandboxEnvFromEnv())
	if err != nil {
		if err == scenario.ErrNotFound {
			return Detail{}, false, nil
		}
		return Detail{}, false, err
	}
	item.Variant = key.Variant

	ctx := context.Background()
	detail, ok, err := s.registryDetail(ctx, item)
	if err != nil {
		return Detail{}, true, err
	}
	if !ok {
		runtime := process.ScenarioRuntime{Name: name, Runtime: "N/A"}
		detail = Detail{
			Scenario: item,
			Runtime:  runtime,
			Details:  scenario.DescribeRuntime(item.Manifest, runtime),
		}
	}
	// The operation record is read regardless of registry-instance presence:
	// an in-flight start spends its dependency phase with no instance row yet,
	// and that is exactly when introspection matters most.
	detail.StartOperation = s.startOperationView(ctx, item)
	return detail, true, nil
}

// startOperationView reads and evaluates the latest start-operation record
// for an instance. Nil when none exists or the registry is unavailable — the
// record is progress, never authority, so read failures degrade silently.
func (s *Service) startOperationView(ctx context.Context, item scenario.Scenario) *lifecycle.StartOperationView {
	store, err := s.openRuntimeRegistry(ctx)
	if err != nil {
		return nil
	}
	defer store.Close()
	op, err := store.GetLatestStartOperation(ctx, item.Slug, item.Variant)
	if err != nil {
		return nil
	}
	estimates, err := store.PhaseDurationEstimates(ctx, item.Slug, item.Variant)
	if err != nil {
		estimates = nil
	}
	view := lifecycle.EvaluateStartOperation(op, process.IsPIDRunning, time.Now().UTC(), estimates)
	return &view
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
	if hostlifecycle.InSandbox() {
		if _, err := hostlifecycle.RunScenario(context.Background(), hostlifecycle.StartOptionsRequest("start", name, opts)); err != nil {
			return StartResult{}, err
		}
		return s.startResultFromLiveDetail(name, false)
	}
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
		Verdict:            result.Health,
		StartOperation:     detail.StartOperation,
	}, nil
}

func (s *Service) RestartDetailed(name string, opts lifecycle.StartOptions) (StartResult, error) {
	if hostlifecycle.InSandbox() {
		if _, err := hostlifecycle.RunScenario(context.Background(), hostlifecycle.StartOptionsRequest("restart", name, opts)); err != nil {
			return StartResult{}, err
		}
		return s.startResultFromLiveDetail(name, false)
	}
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
		Verdict:            result.Health,
		StartOperation:     detail.StartOperation,
	}, nil
}

func (s *Service) startResultFromLiveDetail(name string, alreadyRunning bool) (StartResult, error) {
	item, err := scenario.Load(s.Root, name, scenario.SandboxEnvFromEnv())
	if err != nil {
		return StartResult{}, err
	}
	var detail Detail
	var lastErr error
	awaitErr := lifecycle.Await(s.awaitClock(), lifecycle.SandboxStartPolicy, func() (bool, error) {
		detail, lastErr = s.detailFor(item, name)
		if lastErr != nil {
			// Retried through the policy bound; only the final look is surfaced.
			return false, nil
		}
		return detail.Details.Status == "running", nil
	})
	if awaitErr != nil {
		if lastErr != nil {
			return StartResult{}, lastErr
		}
		return StartResult{}, fmt.Errorf("scenario %s did not report running after host lifecycle proxy", name)
	}
	return StartResult{
		View:           s.viewForDetail(detail),
		Scenario:       item,
		AllocatedPorts: detailPorts(detail),
		AlreadyRunning: alreadyRunning,
		Details:        detail.Details,
	}, nil
}

func detailPorts(detail Detail) map[string]int {
	out := make(map[string]int, len(detail.Details.PortBindings))
	for _, binding := range detail.Details.PortBindings {
		out[binding.Key] = binding.Port
	}
	return out
}

func (s *Service) detailFor(item scenario.Scenario, name string) (Detail, error) {
	ctx := context.Background()
	detail, ok, err := s.registryDetail(ctx, item)
	if err != nil {
		return Detail{}, err
	}
	if !ok {
		runtime := process.ScenarioRuntime{Name: name, Runtime: "N/A"}
		detail = Detail{
			Scenario: item,
			Runtime:  runtime,
			Details:  scenario.DescribeRuntime(item.Manifest, runtime),
		}
	}
	detail.StartOperation = s.startOperationView(ctx, item)
	return detail, nil
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
