package scenarioapp

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/resources"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type EnvironmentValidator interface {
	ValidateScenarioEnvironment(name string) (resources.ScenarioEnvValidationReport, error)
}

type ScenarioOperations interface {
	StartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error)
	RestartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error)
	Inventory() ([]orchestrator.Detail, error)
	InventoryReport() (orchestrator.InventoryReport, error)
	Detail(name string) (orchestrator.Detail, error)
	StartAll() (control.StartReport, error)
	StopAll() (control.StopReport, error)
	ResolvePort(name, portName string) (orchestrator.ResolvedPort, error)
}

type PhaseRunner interface {
	Stop(name string, opts lifecycle.StopOptions) error
	RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error)
	RunPhase(name, phase string, opts lifecycle.PhaseOptions) error
}

type Service struct {
	Scenarios ScenarioOperations
	Runner    PhaseRunner
	Validator EnvironmentValidator
	OpenURL   func(string) error
}

func (s Service) Start(req StartRequest) ([]LifecycleItemOutput, error) {
	items := make([]LifecycleItemOutput, 0, len(req.Names))
	for _, name := range req.Names {
		result, err := s.Scenarios.StartDetailed(name, req.Options)
		if err != nil {
			return nil, err
		}

		status := "started"
		if result.AlreadyRunning {
			status = "already_running"
		}
		items = append(items, LifecycleItemOutput{
			Name:               result.Scenario.Slug,
			Status:             status,
			Health:             result.Details.Health,
			Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
			Endpoints:          endpointOutputs(result.Scenario.Manifest, result.Details.Ports),
			FailedDependencies: append([]string(nil), result.FailedDependencies...),
			FailedResources:    append([]string(nil), result.FailedResources...),
		})

		if req.OpenAfter {
			resolved, err := s.Scenarios.ResolvePort(name, "UI_PORT")
			if err != nil {
				return nil, err
			}
			if err := s.OpenURL(resolved.URL); err != nil {
				return nil, err
			}
		}
	}
	return items, nil
}

func (s Service) Restart(req RestartRequest) ([]LifecycleItemOutput, error) {
	result, err := s.Scenarios.RestartDetailed(req.Name, req.Options)
	if err != nil {
		return nil, err
	}

	item := LifecycleItemOutput{
		Name:               result.Scenario.Slug,
		Status:             "restarted",
		Health:             result.Details.Health,
		Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
		Endpoints:          endpointOutputs(result.Scenario.Manifest, result.Details.Ports),
		FailedDependencies: append([]string(nil), result.FailedDependencies...),
		FailedResources:    append([]string(nil), result.FailedResources...),
	}

	if req.OpenAfter {
		resolved, err := s.Scenarios.ResolvePort(req.Name, "UI_PORT")
		if err != nil {
			return nil, err
		}
		if err := s.OpenURL(resolved.URL); err != nil {
			return nil, err
		}
	}

	return []LifecycleItemOutput{item}, nil
}

func (s Service) Stop(req StopRequest) ([]LifecycleItemOutput, error) {
	if err := s.Runner.Stop(req.Name, lifecycle.StopOptions{}); err != nil {
		return nil, err
	}
	return []LifecycleItemOutput{{Name: req.Name, Status: "stopped"}}, nil
}

func (s Service) List(req ListRequest) (ListResponse, error) {
	inventory, err := s.Scenarios.InventoryReport()
	if err != nil {
		return ListResponse{}, err
	}

	resp := ListResponse{
		Items:    make([]ListItemOutput, 0, len(inventory.Items)),
		Failures: append([]discovery.Failure(nil), inventory.Failures...),
	}
	for _, item := range inventory.Items {
		status := "available"
		if item.Details.Status == "running" {
			status = item.Details.Status
			resp.RunningCount++
		}

		listPorts := []ListPortOutput{}
		if req.IncludePorts && item.Details.Status == "running" {
			listPorts = RuntimePortOutputs(item.Details.PortBindings)
		}

		resp.Items = append(resp.Items, ListItemOutput{
			Name:        item.Scenario.Slug,
			Description: item.Scenario.Manifest.Service.Description,
			Version:     item.Scenario.Manifest.Service.Version,
			Status:      status,
			Tags:        CopyStrings(item.Scenario.Manifest.Service.Tags),
			Path:        item.Scenario.Path + string(os.PathSeparator),
			Ports:       listPorts,
		})
	}
	return resp, nil
}

func (s Service) Info(req InfoRequest) (InfoOutput, error) {
	detail, err := s.Scenarios.Detail(req.Name)
	if err != nil {
		return InfoOutput{}, err
	}
	return InfoOutput{
		Success:  true,
		Scenario: BuildInfoData(detail.Scenario),
		Runtime:  BuildRuntimeDataFromDetail(detail),
	}, nil
}

func (s Service) Status(req StatusRequest) (StatusResponse, error) {
	if req.Name == "" {
		inventory, err := s.Scenarios.InventoryReport()
		if err != nil {
			return StatusResponse{}, err
		}
		items := make([]StatusItemOutput, 0, len(inventory.Items))
		for _, item := range inventory.Items {
			items = append(items, BuildStatusDetail(item))
		}
		return StatusResponse{
			List:     items,
			Failures: append([]discovery.Failure(nil), inventory.Failures...),
		}, nil
	}

	detail, err := s.Scenarios.Detail(req.Name)
	if err != nil {
		return StatusResponse{}, err
	}
	output := StatusSingleOutput{
		Success:  true,
		Scenario: BuildStatusDetail(detail),
		Info:     BuildInfoData(detail.Scenario),
		Runtime:  BuildRuntimeDataFromDetail(detail),
	}
	return StatusResponse{Single: &output}, nil
}

func (s Service) ValidateEnv(req ValidateEnvRequest) (ValidateEnvResponse, error) {
	report, err := s.Validator.ValidateScenarioEnvironment(req.Name)
	if err != nil {
		return ValidateEnvResponse{}, err
	}
	return ValidateEnvResponse{Report: report}, nil
}

func (s Service) Setup(req SetupRequest) (lifecycle.PhaseResult, error) {
	return s.Runner.RunPhaseDetailed(req.Name, "setup", req.Opts)
}

func (s Service) Test(req TestRequest) error {
	return s.Runner.RunPhase(req.Name, "test", req.Opts)
}

func (s Service) StartAll() (BatchResponse, error) {
	report, err := s.Scenarios.StartAll()
	if err != nil {
		return BatchResponse{}, err
	}
	return BatchResponseFromStartReport(report), nil
}

func (s Service) StopAll() (BatchResponse, error) {
	report, err := s.Scenarios.StopAll()
	if err != nil {
		return BatchResponse{}, err
	}
	return BatchResponseFromStopReport(report), nil
}

func (s Service) Port(req PortRequest) (PortResponse, error) {
	detail, err := s.Scenarios.Detail(req.ScenarioName)
	if err != nil {
		return PortResponse{}, err
	}
	listPorts := RuntimePortOutputs(detail.Details.PortBindings)
	portsMap := CopyIntMap(detail.Details.Ports)

	if req.PortName == "" {
		if detail.Details.Status != "running" || len(portsMap) == 0 {
			if req.JSON {
				return PortResponse{List: &PortListOutput{
					Success:  false,
					Scenario: req.ScenarioName,
					Ports:    []ListPortOutput{},
					Error:    "No running runtime ports found for scenario",
				}}, nil
			}
			return PortResponse{}, fmt.Errorf("no running runtime ports found for scenario %q", req.ScenarioName)
		}
		list := &PortListOutput{
			Success:  true,
			Scenario: req.ScenarioName,
			Ports:    listPorts,
		}
		if req.JSON {
			list.Metadata = map[string]int{"count": len(listPorts)}
		}
		return PortResponse{List: list}, nil
	}

	if detail.Details.Status != "running" {
		if req.JSON {
			return PortResponse{Single: &PortSingleOutput{
				Success:  false,
				Scenario: req.ScenarioName,
				PortName: req.PortName,
				Error:    "No running runtime ports found for scenario",
			}}, nil
		}
		return PortResponse{}, fmt.Errorf("no running runtime ports found for scenario %q", req.ScenarioName)
	}

	resolved, ok := scenariomodel.ResolveRuntimePort(detail.Scenario.Manifest, detail.Details.PortBindings, portsMap, req.PortName)
	if !ok {
		if req.JSON {
			return PortResponse{Single: &PortSingleOutput{
				Success:  false,
				Scenario: req.ScenarioName,
				PortName: req.PortName,
				Error:    fmt.Sprintf("No running port named %s for scenario", req.PortName),
			}}, nil
		}
		return PortResponse{}, fmt.Errorf("no running port named %s for scenario %q", req.PortName, req.ScenarioName)
	}

	return PortResponse{Single: &PortSingleOutput{
		Success:  true,
		Scenario: req.ScenarioName,
		PortName: resolved.Key,
		Step:     resolved.Step,
		Port:     resolved.Port,
	}}, nil
}

func (s Service) Open(req OpenRequest) (OpenOutput, error) {
	resolved, err := s.Scenarios.ResolvePort(req.ScenarioName, req.PortName)
	if err != nil {
		return OpenOutput{}, err
	}
	if !req.PrintURL && !req.JSON {
		if err := s.OpenURL(resolved.URL); err != nil {
			return OpenOutput{}, err
		}
		return OpenOutput{}, nil
	}
	return OpenOutput{
		Success:  true,
		Scenario: req.ScenarioName,
		PortName: resolved.Name,
		Port:     resolved.Port,
		URL:      resolved.URL,
	}, nil
}

func envPortMap(manifest scenariomodel.ServiceManifest, ports map[string]int) map[string]int {
	out := make(map[string]int, len(ports))
	for portName, port := range ports {
		envVar := manifest.PortEnvVar(portName)
		if envVar == "" {
			envVar = strings.ToUpper(strings.ReplaceAll(portName, "-", "_")) + "_PORT"
		}
		out[envVar] = port
	}
	return out
}

func endpointOutputs(manifest scenariomodel.ServiceManifest, ports map[string]int) []EndpointOutput {
	endpoints := scenariomodel.RuntimeEndpoints(manifest, ports)
	if len(endpoints) == 0 {
		return nil
	}
	out := make([]EndpointOutput, 0, len(endpoints))
	for _, endpoint := range endpoints {
		out = append(out, EndpointOutput{
			Name:        endpoint.Name,
			Key:         endpoint.Key,
			Description: endpoint.Description,
			Port:        endpoint.Port,
			URL:         endpoint.URL,
		})
	}
	return out
}
